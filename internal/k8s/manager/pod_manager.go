/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package manager

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/retry"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/terminal"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/kubectl/pkg/scheme"
)

type PodManager interface {
	CreatePod(ctx context.Context, clusterID int, namespace string, pod *corev1.Pod) (*corev1.Pod, error)
	GetPod(ctx context.Context, clusterID int, namespace, name string) (*corev1.Pod, error)
	GetPodList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sPod, error)
	UpdatePod(ctx context.Context, clusterID int, namespace string, pod *corev1.Pod) (*corev1.Pod, error)
	DeletePod(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error
	GetPodsByNodeName(ctx context.Context, clusterID int, nodeName string) ([]*model.K8sPod, error)
	GetPodLogs(ctx context.Context, clusterID int, namespace, name string, logOptions *corev1.PodLogOptions) (io.ReadCloser, error)
	BatchDeletePods(ctx context.Context, clusterID int, namespace string, podNames []string, deleteOptions metav1.DeleteOptions) error
	PodTerminalSession(ctx context.Context, clusterID int, namespace, pod, container, shell string, conn *websocket.Conn) error
	UploadFileToPod(ctx *gin.Context, clusterID int, namespace, pod, container, filePath string) error
	PortForward(ctx context.Context, ports []string, dialer httpstream.Dialer) error
	PodPortForward(ctx context.Context, clusterID int, namespace, podName string, ports []model.PodPortForwardPort) error
	DownloadPodFile(ctx context.Context, clusterID int, namespace, pod, container, filePath string) (*k8sutils.PodFileStreamPipe, error)
}

type podManager struct {
	clientFactory client.K8sClient
	logger        *zap.Logger
}

func NewPodManager(clientFactory client.K8sClient, logger *zap.Logger) PodManager {
	return &podManager{
		clientFactory: clientFactory,
		logger:        logger,
	}
}

// getKubeClient 获取Kubernetes客户端
func (m *podManager) getKubeClient(clusterID int) (*kubernetes.Clientset, error) {
	kubeClient, err := m.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}
	return kubeClient, nil
}

// GetPod 获取单个 Pod
func (m *podManager) GetPod(ctx context.Context, clusterID int, namespace, name string) (*corev1.Pod, error) {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取 Pod 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, fmt.Errorf("获取 Pod 失败: %w", err)
	}

	m.logger.Debug("成功获取 Pod",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return pod, nil
}

// GetPodList 获取Pod列表
func (m *podManager) GetPodList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sPod, error) {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	podList, err := kubeClient.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取Pod列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Error(err))
		return nil, fmt.Errorf("获取Pod列表失败: %w", err)
	}

	// 转换为model结构
	var k8sPods []*model.K8sPod
	for _, pod := range podList.Items {
		k8sPod := utils.ConvertToK8sPod(&pod)
		if k8sPod != nil {
			k8sPod.ClusterID = int64(clusterID)
			k8sPods = append(k8sPods, k8sPod)
		}
	}

	m.logger.Debug("成功获取Pod列表",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(k8sPods)))
	return k8sPods, nil
}

// GetPodsByNodeName 获取指定节点上的Pod列表
func (m *podManager) GetPodsByNodeName(ctx context.Context, clusterID int, nodeName string) ([]*model.K8sPod, error) {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	listOptions := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	}

	pods, err := kubeClient.CoreV1().Pods("").List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取节点Pod列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("nodeName", nodeName),
			zap.Error(err))
		return nil, fmt.Errorf("获取节点Pod列表失败: %w", err)
	}

	// 转换为model结构
	var k8sPods []*model.K8sPod
	for _, pod := range pods.Items {
		k8sPod := utils.ConvertToK8sPod(&pod)
		if k8sPod != nil {
			k8sPod.ClusterID = int64(clusterID)
			k8sPods = append(k8sPods, k8sPod)
		}
	}

	m.logger.Debug("成功获取节点Pod列表",
		zap.Int("clusterID", clusterID),
		zap.String("nodeName", nodeName),
		zap.Int("count", len(k8sPods)))
	return k8sPods, nil
}

// DeletePod 删除 Pod
func (m *podManager) DeletePod(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = kubeClient.CoreV1().Pods(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		m.logger.Error("删除 Pod 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("删除 Pod 失败: %w", err)
	}

	m.logger.Info("成功删除 Pod",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return nil
}

// GetPodLogs 获取 Pod 日志
func (m *podManager) GetPodLogs(ctx context.Context, clusterID int, namespace, name string, logOptions *corev1.PodLogOptions) (io.ReadCloser, error) {
	kubeClient, err := m.getKubeClient(clusterID)

	if err != nil {
		return nil, err
	}

	podLogRequest := kubeClient.CoreV1().Pods(namespace).GetLogs(name, logOptions)

	stream, err := podLogRequest.Stream(ctx)
	if err != nil {
		m.logger.Error("获取 Pod 日志流失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return stream, fmt.Errorf("获取 Pod 日志流失败: %w", err)
	}
	return stream, err
}

// BatchDeletePods 批量删除 Pod
func (m *podManager) BatchDeletePods(ctx context.Context, clusterID int, namespace string, podNames []string, deleteOpts metav1.DeleteOptions) error {
	kubeClient, err := m.getKubeClient(clusterID)

	if err != nil {
		return err
	}

	tasks := make([]retry.WrapperTask, 0, len(podNames))
	for _, name := range podNames {

		tasks = append(tasks, retry.WrapperTask{
			Backoff: retry.DefaultBackoff,

			Task: func(ctx context.Context) error {
				if err := kubeClient.CoreV1().Pods(namespace).Delete(ctx, name, deleteOpts); err != nil {
					m.logger.Error("删除Pod失败", zap.Error(err),
						zap.Int("cluster_id", clusterID),
						zap.String("namespace", namespace),
						zap.String("name", name))
				}
				return nil
			},
			RetryCheck: func(err error) bool {
				return k8serrors.IsTimeout(err) ||
					k8serrors.IsTooManyRequests(err) ||
					k8serrors.IsServerTimeout(err) ||
					k8serrors.IsConflict(err)
			},
		})
	}
	err = retry.RunRetryWithConcurrency(ctx, 3, tasks)
	if err != nil {
		m.logger.Warn("批量删除Pod失败",
			zap.Error(err))

		return pkg.NewBusinessError(constants.ErrK8sResourceDelete, "批量删除Pod失败")
	}
	return nil
}

func (m *podManager) PodTerminalSession(
	ctx context.Context,
	clusterID int,
	namespace, pod, container, shell string,
	conn *websocket.Conn,
) error {

	kubeClient, err := m.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	restConfig, err := m.clientFactory.GetRestConfig(clusterID)
	if err != nil {
		m.logger.Error("获取集群配置失败", zap.Int("clusterID", clusterID), zap.Error(err))
		return fmt.Errorf("获取集群配置失败: %w", err)
	}

	if len(shell) == 0 {
		shell = "sh"
	}

	terminal.NewTerminalHandler(kubeClient, restConfig, m.logger).
		HandleSession(ctx, shell, namespace, pod, container, conn)
	return nil
}

func (m *podManager) UploadFileToPod(ctx *gin.Context, clusterID int, namespace, pod, container, filePath string) error {
	// 参数验证
	if namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}
	if pod == "" {
		return fmt.Errorf("Pod名称不能为空")
	}
	if container == "" {
		return fmt.Errorf("容器名称不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	restConfig, err := m.clientFactory.GetRestConfig(clusterID)
	if err != nil {
		m.logger.Error("获取集群配置失败", zap.Int("clusterID", clusterID), zap.Error(err))
		return fmt.Errorf("获取集群配置失败: %w", err)
	}

	// 验证Pod是否存在并且正在运行
	podObj, err := kubeClient.CoreV1().Pods(namespace).Get(ctx.Request.Context(), pod, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取Pod信息失败",
			zap.Error(err),
			zap.String("namespace", namespace),
			zap.String("pod", pod))
		return fmt.Errorf("获取Pod信息失败: %w", err)
	}

	if podObj.Status.Phase != corev1.PodRunning {
		return fmt.Errorf("Pod状态不是Running，当前状态: %s", podObj.Status.Phase)
	}

	// 验证容器是否存在
	var containerExists bool
	for _, c := range podObj.Spec.Containers {
		if c.Name == container {
			containerExists = true
			break
		}
	}
	if !containerExists {
		return fmt.Errorf("容器 %s 在Pod中不存在", container)
	}

	targetDir := filePath
	if targetDir == "" {
		targetDir = "/tmp"
	}

	// 解析上传的文件
	files, err := parseMultipartFiles(ctx)
	if err != nil {
		m.logger.Error("解析上传文件失败", zap.Error(err))
		return fmt.Errorf("解析上传文件失败: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("没有找到要上传的文件")
	}

	m.logger.Info("开始上传文件到Pod",
		zap.String("namespace", namespace),
		zap.String("pod", pod),
		zap.String("container", container),
		zap.String("targetDir", targetDir),
		zap.Int("fileCount", len(files)))

	reader, writer := io.Pipe()
	var tarErr error

	go func() {
		defer writer.Close()
		if tarErr = writeFilesToTar(files, writer); tarErr != nil {
			m.logger.Error("打包文件成 tar 失败", zap.Error(tarErr))
			_ = writer.CloseWithError(tarErr)
		}
	}()

	// 检查目标目录是否存在，如果不存在则创建
	createDirReq := kubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   []string{"mkdir", "-p", targetDir},
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(restConfig, http.MethodPost, createDirReq.URL())
	if err != nil {
		return fmt.Errorf("创建目录命令executor失败: %w", err)
	}

	var stdout, stderr strings.Builder
	err = exec.StreamWithContext(ctx.Request.Context(), remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		m.logger.Warn("创建目录失败，可能目录已存在",
			zap.Error(err),
			zap.String("stdout", stdout.String()),
			zap.String("stderr", stderr.String()))
	}

	// 执行文件上传
	execReq := kubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   []string{"tar", "-xmf", "-", "-C", targetDir},
			Stdin:     true,
			Stdout:    false,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err = remotecommand.NewSPDYExecutor(restConfig, http.MethodPost, execReq.URL())
	if err != nil {
		return fmt.Errorf("创建上传executor失败: %w", err)
	}

	var uploadStderr strings.Builder
	err = exec.StreamWithContext(ctx.Request.Context(), remotecommand.StreamOptions{
		Stdin:             reader,
		Stdout:            nil,
		Stderr:            &uploadStderr,
		TerminalSizeQueue: nil,
	})

	if err != nil {
		m.logger.Error("执行上传失败",
			zap.Error(err),
			zap.String("stderr", uploadStderr.String()))
		return fmt.Errorf("执行上传失败: %w, stderr: %s", err, uploadStderr.String())
	}

	if tarErr != nil {
		return fmt.Errorf("打包tar文件失败: %w", tarErr)
	}

	m.logger.Info("成功上传文件到Pod",
		zap.String("namespace", namespace),
		zap.String("pod", pod),
		zap.String("container", container),
		zap.String("targetDir", targetDir),
		zap.Int("fileCount", len(files)))

	return nil
}

func (m *podManager) PortForward(ctx context.Context, ports []string, dialer httpstream.Dialer) error {

	// 创建 PortForwarder
	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})

	forwarder, err := portforward.New(dialer, ports, stopChan, readyChan, io.Discard, io.Discard)
	if err != nil {
		return pkg.NewBusinessError(constants.ErrK8sPortForward, err.Error())
	}

	// 自动关闭转发
	go func() {
		<-ctx.Done()
		close(stopChan)
	}()

	// 异步开启转发
	go func() {
		if err := forwarder.ForwardPorts(); err != nil {
			m.logger.Error("创建端口转发失败",
				zap.Error(err),
				zap.Strings("ports", ports),
			)
			return
		}
	}()
	// 等待就绪
	<-readyChan
	return nil
}

// PodPortForward Pod端口转发
func (m *podManager) PodPortForward(ctx context.Context, clusterID int, namespace, podName string, ports []model.PodPortForwardPort) error {
	if len(ports) == 0 {
		return fmt.Errorf("端口转发配置不能为空")
	}

	// 获取Kubernetes客户端和配置
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Error(err),
			zap.Int("clusterID", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	restConfig, err := m.clientFactory.GetRestConfig(clusterID)
	if err != nil {
		m.logger.Error("获取集群配置失败",
			zap.Error(err),
			zap.Int("clusterID", clusterID))
		return fmt.Errorf("获取集群配置失败: %w", err)
	}

	// 验证Pod是否存在
	pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取Pod失败",
			zap.Error(err),
			zap.String("namespace", namespace),
			zap.String("podName", podName))
		return fmt.Errorf("获取Pod失败: %w", err)
	}

	if pod.Status.Phase != corev1.PodRunning {
		return fmt.Errorf("Pod状态不是Running，当前状态: %s", pod.Status.Phase)
	}

	// 构建端口转发URL
	req := kubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("portforward")

	// 创建SPDY升级器和拨号器
	transport, upgrader, err := spdy.RoundTripperFor(restConfig)
	if err != nil {
		m.logger.Error("创建SPDY升级器失败",
			zap.Error(err))
		return fmt.Errorf("创建SPDY升级器失败: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, req.URL())

	// 构建端口映射字符串
	portSpecs := make([]string, len(ports))
	for i, port := range ports {
		portSpecs[i] = fmt.Sprintf("%d:%d", port.LocalPort, port.RemotePort)
	}

	m.logger.Info("开始端口转发",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("podName", podName),
		zap.Strings("ports", portSpecs))

	// 调用底层端口转发方法
	err = m.PortForward(ctx, portSpecs, dialer)
	if err != nil {
		m.logger.Error("端口转发失败",
			zap.Error(err),
			zap.Strings("ports", portSpecs))
		return fmt.Errorf("端口转发失败: %w", err)
	}

	m.logger.Info("端口转发成功建立",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("podName", podName),
		zap.Strings("ports", portSpecs))

	return nil
}

func (m *podManager) DownloadPodFile(ctx context.Context, clusterID int, namespace, pod, container, filePath string) (*k8sutils.PodFileStreamPipe, error) {
	// 参数验证
	if namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}
	if pod == "" {
		return nil, fmt.Errorf("Pod名称不能为空")
	}
	if container == "" {
		return nil, fmt.Errorf("容器名称不能为空")
	}
	if filePath == "" {
		return nil, fmt.Errorf("文件路径不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	restConfig, err := m.clientFactory.GetRestConfig(clusterID)
	if err != nil {
		m.logger.Error("获取集群配置失败", zap.Int("clusterID", clusterID), zap.Error(err))
		return nil, fmt.Errorf("获取集群配置失败: %w", err)
	}

	// 验证Pod是否存在并且正在运行
	podObj, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, pod, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取Pod信息失败",
			zap.Error(err),
			zap.String("namespace", namespace),
			zap.String("pod", pod))
		return nil, fmt.Errorf("获取Pod信息失败: %w", err)
	}

	if podObj.Status.Phase != corev1.PodRunning {
		return nil, fmt.Errorf("Pod状态不是Running，当前状态: %s", podObj.Status.Phase)
	}

	// 验证容器是否存在
	var containerExists bool
	for _, c := range podObj.Spec.Containers {
		if c.Name == container {
			containerExists = true
			break
		}
	}
	if !containerExists {
		return nil, fmt.Errorf("容器 %s 在Pod中不存在", container)
	}

	m.logger.Info("开始下载Pod文件",
		zap.String("namespace", namespace),
		zap.String("pod", pod),
		zap.String("container", container),
		zap.String("filePath", filePath))

	reader, err := k8sutils.NewPodFileStreamPipe(
		ctx, restConfig, kubeClient, namespace, pod, container, filePath)

	if err != nil {
		m.logger.Error("创建Pod文件流失败",
			zap.Error(err),
			zap.String("namespace", namespace),
			zap.String("podName", pod),
			zap.String("containerName", container),
			zap.String("filePath", filePath))
		return nil, fmt.Errorf("创建Pod文件流失败: %w", err)
	}

	m.logger.Info("成功创建Pod文件流",
		zap.String("namespace", namespace),
		zap.String("pod", pod),
		zap.String("container", container),
		zap.String("filePath", filePath))

	return reader, nil
}

// CreatePod 创建Pod
func (m *podManager) CreatePod(ctx context.Context, clusterID int, namespace string, pod *corev1.Pod) (*corev1.Pod, error) {
	if pod == nil {
		return nil, fmt.Errorf("pod不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	// 如果pod对象中没有指定namespace，使用参数中的namespace
	targetNamespace := pod.Namespace
	if targetNamespace == "" {
		targetNamespace = namespace
		pod.Namespace = namespace
	}

	createdPod, err := kubeClient.CoreV1().Pods(targetNamespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		m.logger.Error("创建Pod失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", targetNamespace),
			zap.String("name", pod.Name),
			zap.Error(err))
		return nil, fmt.Errorf("创建Pod失败: %w", err)
	}

	m.logger.Info("成功创建Pod",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", targetNamespace),
		zap.String("name", pod.Name))
	return createdPod, nil
}

// UpdatePod 更新Pod
func (m *podManager) UpdatePod(ctx context.Context, clusterID int, namespace string, pod *corev1.Pod) (*corev1.Pod, error) {
	if pod == nil {
		return nil, fmt.Errorf("pod不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	updatedPod, err := kubeClient.CoreV1().Pods(namespace).Update(ctx, pod, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新Pod失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", pod.Name),
			zap.Error(err))
		return nil, fmt.Errorf("更新Pod失败: %w", err)
	}

	m.logger.Info("成功更新Pod",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", pod.Name))
	return updatedPod, nil
}

type fileWithHeader struct {
	file   multipart.File
	header *multipart.FileHeader
}

func parseMultipartFiles(ctx *gin.Context) ([]fileWithHeader, error) {
	if err := ctx.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		return nil, fmt.Errorf("解析多部分表单失败: %w", err)
	}

	if ctx.Request.MultipartForm == nil || len(ctx.Request.MultipartForm.File) == 0 {
		return nil, fmt.Errorf("没有找到上传的文件")
	}

	files := make([]fileWithHeader, 0)
	for name := range ctx.Request.MultipartForm.File {
		file, header, err := ctx.Request.FormFile(name)
		if err != nil {
			return nil, fmt.Errorf("解析文件 %s 失败: %w", name, err)
		}

		// 验证文件大小
		if header.Size > 100<<20 { // 100MB per file limit
			file.Close()
			return nil, fmt.Errorf("文件 %s 太大，最大允许100MB", header.Filename)
		}

		// 验证文件名
		if header.Filename == "" {
			file.Close()
			return nil, fmt.Errorf("文件名不能为空")
		}

		files = append(files, fileWithHeader{file: file, header: header})
	}
	return files, nil
}

func writeFilesToTar(files []fileWithHeader, w io.Writer) error {
	if len(files) == 0 {
		return fmt.Errorf("没有文件需要打包")
	}

	tarWriter := tar.NewWriter(w)
	defer func() {
		if err := tarWriter.Close(); err != nil {
			// 日志记录tarWriter关闭错误，但不返回错误
		}
	}()

	for i, f := range files {
		// 确保每个文件都会被正确关闭
		err := func(fileInfo fileWithHeader, index int) error {
			defer func() {
				if err := fileInfo.file.Close(); err != nil {
					// 记录文件关闭错误，但继续处理
				}
			}()

			// 验证文件名，避免路径遍历攻击
			filename := fileInfo.header.Filename
			if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
				return fmt.Errorf("文件名包含非法字符: %s", filename)
			}

			hdr := &tar.Header{
				Name: filename,
				Mode: 0644, // 使用更安全的权限
				Size: fileInfo.header.Size,
			}

			if err := tarWriter.WriteHeader(hdr); err != nil {
				return fmt.Errorf("写入tar头失败(文件 %s): %w", filename, err)
			}

			bytesWritten, err := io.Copy(tarWriter, fileInfo.file)
			if err != nil {
				return fmt.Errorf("写入文件内容失败(文件 %s): %w", filename, err)
			}

			if bytesWritten != fileInfo.header.Size {
				return fmt.Errorf("文件大小不匹配(文件 %s): 期望 %d 字节，实际写入 %d 字节",
					filename, fileInfo.header.Size, bytesWritten)
			}

			return nil
		}(f, i)

		if err != nil {
			return err
		}
	}

	return nil
}
