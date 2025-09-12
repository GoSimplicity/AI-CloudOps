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
	"k8s.io/kubectl/pkg/scheme"
)

// PodManager Pod资源管理器接口
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
	DownloadPodFile(ctx context.Context, clusterID int, namespace, pod, container, filePath string) (*k8sutils.PodFileStreamPipe, error)
}

// podManager Pod资源管理器实现
type podManager struct {
	clientFactory client.K8sClient
	logger        *zap.Logger
}

// NewPodManager 创建新的Pod管理器实例
func NewPodManager(clientFactory client.K8sClient, logger *zap.Logger) PodManager {
	return &podManager{
		clientFactory: clientFactory,
		logger:        logger,
	}
}

// getKubeClient 私有方法：获取Kubernetes客户端
func (p *podManager) getKubeClient(clusterID int) (*kubernetes.Clientset, error) {
	kubeClient, err := p.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}
	return kubeClient, nil
}

// GetPod 获取单个 Pod
func (p *podManager) GetPod(ctx context.Context, clusterID int, namespace, name string) (*corev1.Pod, error) {
	kubeClient, err := p.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		p.logger.Error("获取 Pod 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, fmt.Errorf("获取 Pod 失败: %w", err)
	}

	p.logger.Debug("成功获取 Pod",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return pod, nil
}

// GetPodList 获取Pod列表
func (p *podManager) GetPodList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sPod, error) {
	kubeClient, err := p.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	podList, err := kubeClient.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		p.logger.Error("获取Pod列表失败",
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

	p.logger.Debug("成功获取Pod列表",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(k8sPods)))
	return k8sPods, nil
}

// GetPodsByNodeName 获取指定节点上的Pod列表
func (p *podManager) GetPodsByNodeName(ctx context.Context, clusterID int, nodeName string) ([]*model.K8sPod, error) {
	kubeClient, err := p.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	listOptions := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	}

	pods, err := kubeClient.CoreV1().Pods("").List(ctx, listOptions)
	if err != nil {
		p.logger.Error("获取节点Pod列表失败",
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

	p.logger.Debug("成功获取节点Pod列表",
		zap.Int("clusterID", clusterID),
		zap.String("nodeName", nodeName),
		zap.Int("count", len(k8sPods)))
	return k8sPods, nil
}

// DeletePod 删除 Pod
func (p *podManager) DeletePod(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error {
	kubeClient, err := p.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = kubeClient.CoreV1().Pods(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		p.logger.Error("删除 Pod 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("删除 Pod 失败: %w", err)
	}

	p.logger.Info("成功删除 Pod",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return nil
}

// GetPodLogs 获取 Pod 日志
func (p *podManager) GetPodLogs(ctx context.Context, clusterID int, namespace, name string, logOptions *corev1.PodLogOptions) (io.ReadCloser, error) {
	kubeClient, err := p.getKubeClient(clusterID)

	if err != nil {
		return nil, err
	}

	podLogRequest := kubeClient.CoreV1().Pods(namespace).GetLogs(name, logOptions)

	stream, err := podLogRequest.Stream(ctx)
	if err != nil {
		p.logger.Error("获取 Pod 日志流失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return stream, fmt.Errorf("获取 Pod 日志流失败: %w", err)
	}
	return stream, err
}

// BatchDeletePods 批量删除 Pod
func (p *podManager) BatchDeletePods(ctx context.Context, clusterID int, namespace string, podNames []string, deleteOpts metav1.DeleteOptions) error {
	kubeClient, err := p.getKubeClient(clusterID)

	if err != nil {
		return err
	}

	tasks := make([]retry.WrapperTask, 0, len(podNames))
	for _, name := range podNames {

		tasks = append(tasks, retry.WrapperTask{
			Backoff: retry.DefaultBackoff,

			Task: func(ctx context.Context) error {
				if err := kubeClient.CoreV1().Pods(namespace).Delete(ctx, name, deleteOpts); err != nil {
					p.logger.Error("删除Pod失败", zap.Error(err),
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
		p.logger.Warn("批量删除Pod失败",
			zap.Error(err))

		return pkg.NewBusinessError(constants.ErrK8sResourceDelete, "批量删除Pod失败")
	}
	return nil
}

func (p *podManager) PodTerminalSession(
	ctx context.Context,
	clusterID int,
	namespace, pod, container, shell string,
	conn *websocket.Conn,
) error {

	kubeClient, err := p.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	restConfig, err := p.clientFactory.GetRestConfig(clusterID)
	if err != nil {
		p.logger.Error("获取集群配置失败", zap.Int("clusterID", clusterID), zap.Error(err))
		return fmt.Errorf("获取集群配置失败: %w", err)
	}

	if len(shell) == 0 {
		shell = "sh"
	}

	terminal.NewTerminalerHandler(kubeClient, restConfig).
		HandleSession(ctx, shell, namespace, pod, container, conn)
	return nil
}

func (p *podManager) UploadFileToPod(ctx *gin.Context, clusterID int, namespace, pod, container, filePath string) error {

	kubeClient, err := p.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	restConfig, err := p.clientFactory.GetRestConfig(clusterID)
	if err != nil {
		p.logger.Error("获取集群配置失败", zap.Int("clusterID", clusterID), zap.Error(err))
		return fmt.Errorf("获取集群配置失败: %w", err)
	}

	targetDir := filePath
	if targetDir == "" {
		targetDir = "/"
	}
	// 解析上传的文件
	files, err := parseMultipartFiles(ctx)
	if err != nil {
		return err
	}

	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()
		if tarErr := writeFilesToTar(files, writer); tarErr != nil {
			p.logger.Error("打包文件成 tar 失败", zap.Error(tarErr))
			_ = writer.CloseWithError(tarErr)
		}
	}()

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

	exec, err := remotecommand.NewSPDYExecutor(restConfig, http.MethodPost, execReq.URL())
	if err != nil {
		return pkg.NewBusinessError(constants.ErrK8sResourceDelete, "创建 Executor 失败")
	}

	if err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             reader,
		Stdout:            nil,
		Stderr:            ctx.Writer,
		TerminalSizeQueue: nil,
	}); err != nil {
		return pkg.NewBusinessError(constants.ErrK8sResourceDelete, "执行上传失败")
	}
	return nil
}

func (p *podManager) PortForward(ctx context.Context, ports []string, dialer httpstream.Dialer) error {

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
			p.logger.Error("创建端口转发失败",
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

func (p *podManager) DownloadPodFile(ctx context.Context, clusterID int, namespace, pod, container, filePath string) (*k8sutils.PodFileStreamPipe, error) {
	kubeClient, err := p.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	restConfig, err := p.clientFactory.GetRestConfig(clusterID)
	if err != nil {
		p.logger.Error("获取集群配置失败", zap.Int("clusterID", clusterID), zap.Error(err))
		return nil, fmt.Errorf("获取集群配置失败: %w", err)
	}

	reader, err := k8sutils.NewPodFileStreamPipe(
		ctx, restConfig, kubeClient, namespace, pod, container, filePath)

	if err != nil {
		p.logger.Error("创建Pod文件流失败",
			zap.Error(err),
			zap.String("Namespace", namespace),
			zap.String("PodName", pod),
			zap.String("ContainerName", container),
			zap.String("FilePath", filePath),
		)
	}
	return reader, err
}

// CreatePod 创建Pod
func (p *podManager) CreatePod(ctx context.Context, clusterID int, namespace string, pod *corev1.Pod) (*corev1.Pod, error) {
	if pod == nil {
		return nil, fmt.Errorf("pod不能为空")
	}

	kubeClient, err := p.getKubeClient(clusterID)
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
		p.logger.Error("创建Pod失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", targetNamespace),
			zap.String("name", pod.Name),
			zap.Error(err))
		return nil, fmt.Errorf("创建Pod失败: %w", err)
	}

	p.logger.Info("成功创建Pod",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", targetNamespace),
		zap.String("name", pod.Name))
	return createdPod, nil
}

// UpdatePod 更新Pod
func (p *podManager) UpdatePod(ctx context.Context, clusterID int, namespace string, pod *corev1.Pod) (*corev1.Pod, error) {
	if pod == nil {
		return nil, fmt.Errorf("pod不能为空")
	}

	kubeClient, err := p.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	updatedPod, err := kubeClient.CoreV1().Pods(namespace).Update(ctx, pod, metav1.UpdateOptions{})
	if err != nil {
		p.logger.Error("更新Pod失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", pod.Name),
			zap.Error(err))
		return nil, fmt.Errorf("更新Pod失败: %w", err)
	}

	p.logger.Info("成功更新Pod",
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
	if err := ctx.Request.ParseMultipartForm(10 << 20); err != nil {
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceDelete, "读取上传文件失败")
	}
	files := make([]fileWithHeader, 0)
	for name := range ctx.Request.MultipartForm.File {
		file, header, err := ctx.Request.FormFile(name)
		if err != nil {
			return nil, pkg.NewBusinessError(constants.ErrK8sResourceDelete, "解析上传文件失败")
		}
		files = append(files, fileWithHeader{file: file, header: header})
	}
	return files, nil
}

func writeFilesToTar(files []fileWithHeader, w io.Writer) error {
	tarWriter := tar.NewWriter(w)
	defer tarWriter.Close()
	for _, f := range files {
		func(f fileWithHeader) {
			defer f.file.Close()
			hdr := &tar.Header{
				Name: f.header.Filename,
				Mode: 0600,
				Size: f.header.Size,
			}
			if err := tarWriter.WriteHeader(hdr); err != nil {
				_ = w.(io.WriteCloser).Close()
				return
			}
			if _, err := io.Copy(tarWriter, f.file); err != nil {
				_ = w.(io.WriteCloser).Close()
				return
			}
		}(f)
	}
	return nil
}
