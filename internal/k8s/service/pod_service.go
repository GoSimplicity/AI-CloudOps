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

package service

import (
	"bufio"
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils/query"
	utilsStream "github.com/GoSimplicity/AI-CloudOps/pkg/utils/stream"
	"github.com/gin-gonic/gin"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport/spdy"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkgutils "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodService interface {
	GetPodList(ctx context.Context, queryParams *query.Query, req *model.GetPodListReq) (*model.ListResp[*model.K8sPod], error)
	GetPodsByNodeName(ctx context.Context, req *model.PodsByNodeReq) ([]*model.K8sPod, error)
	GetPod(ctx context.Context, req *model.K8sGetPodReq) (*model.K8sPod, error)
	GetPodYaml(ctx context.Context, req *model.K8sGetPodReq) (string, error)
	GetContainersByPod(ctx context.Context, req *model.PodContainersReq) ([]*model.K8sPodContainer, error)
	GetPodLogs(ctx *gin.Context, req *model.PodLogReq) error
	DeletePodWithOptions(ctx context.Context, req *model.K8sDeletePodReq) error
	BatchDeletePods(ctx context.Context, req *model.K8sPodBatchDeleteReq) error
	ExecInPod(ctx *gin.Context, req *model.PodExecReq) error
	PortForward(ctx context.Context, req *model.PodPortForwardReq) error
	DownloadPodFile(ctx *gin.Context, req *model.PodFileReq) error
	UploadFileToPod(ctx *gin.Context, req *model.PodFileReq) error
}

type podService struct {
	dao        dao.ClusterDAO
	client     client.K8sClient   // 保持向后兼容
	podManager manager.PodManager // 新的依赖注入
	logger     *zap.Logger
}

func NewPodService(dao dao.ClusterDAO, client client.K8sClient, podManager manager.PodManager, logger *zap.Logger) PodService {
	return &podService{
		dao:        dao,
		client:     client,     // 保持向后兼容，某些方法可能仍需要
		podManager: podManager, // 使用新的 manager
		logger:     logger,
	}
}

// GetContainersByPod 获取指定 Pod 中的容器列表
func (p *podService) GetContainersByPod(ctx context.Context, req *model.PodContainersReq) ([]*model.K8sPodContainer, error) {
	// 使用新的 PodManager
	pod, err := p.podManager.GetPod(ctx, req.ClusterID, req.Namespace, req.PodName)
	if err != nil {
		p.logger.Error("获取 Pod 失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("pod_ame", req.PodName),
			zap.Error(err))
		return nil, err
	}

	return k8sutils.ConvertK8sContainers(pod.Spec.Containers), nil
}

// GetPodYaml 获取指定 Pod 的 YAML 配置
func (p *podService) GetPodYaml(ctx context.Context, req *model.K8sGetPodReq) (string, error) {

	pod, err := p.podManager.GetPod(ctx, req.ClusterID, req.Namespace, req.PodName)
	if err != nil {
		return "", err
	}

	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(pod)
	if err != nil {
		return "", pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取Pod yaml失败")
	}

	unstructuredObj := &unstructured.Unstructured{Object: m}

	d, err := utils.ConvertUnstructuredToYAML(unstructuredObj)
	if err != nil {
		p.logger.Error("序列化Ingress为yaml失败", zap.Error(err))
		return "", pkg.NewBusinessError(constants.ErrK8sResourceOperation, "序列化Pod失败")
	}

	return d, nil
}

// GetPodsByNodeName 获取指定节点的 Pod 列表
func (p *podService) GetPodsByNodeName(ctx context.Context, req *model.PodsByNodeReq) ([]*model.K8sPod, error) {
	// 使用新的 PodManager
	pods, err := p.podManager.GetPodsByNodeName(ctx, req.ClusterID, req.NodeName)
	if err != nil {
		p.logger.Error("获取节点 Pod 列表失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("nodeName", req.NodeName),
			zap.Error(err))
		return nil, err
	}

	return k8sutils.ConvertToK8sPods(pods.Items), nil
}

// GetPodList 获取Pod列表
func (p *podService) GetPodList(ctx context.Context, queryParam *query.Query, req *model.GetPodListReq) (*model.ListResp[*model.K8sPod], error) {

	_ = queryParam.AppendLabelSelector(req.Labels)

	if req.Namespace == "" {
		req.Namespace = corev1.NamespaceAll
	}

	pods, err := p.podManager.GetPodList(ctx, req.ClusterID, req.Namespace, queryParam)
	if err != nil {
		p.logger.Error("获取Ingress列表失败",
			zap.String("namespace", req.Namespace),
			zap.Error(err))

		return nil, pkg.NewBusinessError(constants.ErrK8sResourceList, "获取Ingress列表失败")
	}
	return pods, nil
}

// GetPod 获取单个Pod详情
func (p *podService) GetPod(ctx context.Context, req *model.K8sGetPodReq) (*model.K8sPod, error) {

	pod, err := p.podManager.GetPod(ctx, req.ClusterID, req.Namespace, req.PodName)
	if err != nil {
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取Pod详情失败")
	}
	return k8sutils.ConvertToK8sPod(pod), nil
}

// GetPodLogs 获取Pod日志
func (p *podService) GetPodLogs(ctx *gin.Context, req *model.PodLogReq) error {

	logOptions := &corev1.PodLogOptions{
		Container:    req.Container,
		Follow:       req.Follow,
		Previous:     req.Previous,
		SinceSeconds: req.SinceSeconds,
		Timestamps:   req.Timestamps,
		TailLines:    req.TailLines,
		LimitBytes:   req.LimitBytes,
	}

	if req.SinceTime != "" {
		sinceTime, err := time.Parse(time.RFC3339, req.SinceTime)
		if err != nil {
			p.logger.Error("解析时间参数失败", zap.String("sinceTime", req.SinceTime), zap.Error(err))
			return pkg.NewBusinessError(constants.ErrInvalidParam, "时间参数格式错误")
		}
		metaTime := metav1.NewTime(sinceTime)
		logOptions.SinceTime = &metaTime
	}
	out, err := p.podManager.GetPodLogs(ctx, req.ClusterID, req.Namespace, req.Container, logOptions)
	if err != nil {
		return err
	}

	utilsStream.SseStream(ctx, func(ctx context.Context, msgChan chan interface{}) {

		defer func() {
			if err = out.Close(); err != nil {
				p.logger.Error("关闭 Pod 日志流失败", zap.Error(err))
			}
		}()

		reader := bufio.NewReader(out)
		wait.UntilWithContext(ctx, func(ctx context.Context) {

			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					line = strings.TrimSpace(line)
					if len(line) > 0 {
						msgChan <- line
					}
					return
				}
				p.logger.Error("读取 Pod 日志失败", zap.Error(err))
				return
			}
			// 处理 \r 覆盖输出
			if strings.ContainsRune(line, '\r') {
				segments := strings.Split(line, "\r")
				for _, seg := range segments {
					seg = strings.TrimSpace(seg)
					if seg != "" {
						msgChan <- seg
					}
				}
			} else {
				msgChan <- strings.TrimSpace(line)
			}
		}, 0)
	}, p.logger)

	return err
}

// DeletePodWithOptions 删除Pod
func (p *podService) DeletePodWithOptions(ctx context.Context, req *model.K8sDeletePodReq) error {
	var deleteOpts = metav1.DeleteOptions{
		GracePeriodSeconds: req.GracePeriodSeconds,
		PropagationPolicy: func() *metav1.DeletionPropagation {
			if req.Force {
				policy := metav1.DeletePropagationBackground
				return &policy
			}
			return nil
		}(),
	}

	if err := p.podManager.DeletePod(ctx, req.ClusterID, req.Namespace, req.PodName, deleteOpts); err != nil {
		return err
	}

	p.logger.Info("成功删除Pod",
		zap.String("Namespace", req.Namespace),
		zap.String("PodName", req.PodName))
	return nil
}

// BatchDeletePods 批量删除Pod
func (p *podService) BatchDeletePods(ctx context.Context, req *model.K8sPodBatchDeleteReq) error {

	var deleteOpts = metav1.DeleteOptions{
		GracePeriodSeconds: req.GracePeriodSeconds,
		PropagationPolicy: func() *metav1.DeletionPropagation {
			if req.Force {
				policy := metav1.DeletePropagationBackground
				return &policy
			}
			return nil
		}(),
	}

	if err := p.podManager.BatchDeletePods(ctx, req.ClusterID, req.Namespace, req.Names, deleteOpts); err != nil {
		return err
	}

	p.logger.Info("成功批量删除Pod",
		zap.String("Namespace", req.Namespace),
		zap.Int("Count", len(req.Names)))
	return nil
}

// ExecInPod Pod命令执行
func (p *podService) ExecInPod(ctx *gin.Context, req *model.PodExecReq) error {

	restConfig, err := p.getRestConfig(ctx, req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubeconfig配置失败",
			zap.Error(err))
		return err
	}

	conn, err := pkgutils.UpGrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		p.logger.Error("升级ws失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrorWsUpgradeFailed, "初始化ws失败")
	}

	return p.podManager.PodTerminalSession(ctx,
		req.ClusterID, req.Namespace, req.PodName, req.Container, req.Shell, conn, restConfig)
}

// UploadFileToPod 上传文件到pod
func (p *podService) UploadFileToPod(ctx *gin.Context, req *model.PodFileReq) error {

	restConfig, err := p.getRestConfig(ctx, req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubeconfig配置失败",
			zap.Error(err))
		return err
	}
	return p.podManager.UploadFileToPod(ctx, req.ClusterID, req.Namespace, req.PodName,
		req.ContainerName, req.FilePath, restConfig)
}

// PortForward Pod端口转发
func (p *podService) PortForward(ctx context.Context, req *model.PodPortForwardReq) error {

	// 获取 restconfig
	restConfig, err := p.getRestConfig(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	// 构造端口映射
	portsSpec := buildPortsSpec(req.Ports)

	// 构造 SPDY dialer
	dialer, err := p.buildDialer(restConfig, req.Namespace, req.ResourceName)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return p.podManager.PortForward(ctx, portsSpec, dialer)
}

// DownloadPodFile 实现Pod内文件下载
func (p *podService) DownloadPodFile(ctx *gin.Context, req *model.PodFileReq) error {

	restConfig, err := p.getRestConfig(ctx, req.ClusterID)
	if err != nil {
		return err
	}

	fileName := filepath.Base(req.FilePath)
	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename=%s.tar`, fileName))
	ctx.Header("Content-Type", "application/octet-stream")

	reader, err := p.podManager.DownloadPodFile(ctx.Request.Context(),
		req.ClusterID, req.Namespace, req.PodName, req.ContainerName, req.FilePath, restConfig)

	if err != nil {
		p.logger.Error("创建Pod文件流失败",
			zap.Error(err),
			zap.String("Namespace", req.Namespace),
			zap.String("PodName", req.PodName),
			zap.String("ContainerName", req.ContainerName),
			zap.String("FilePath", req.FilePath),
		)
		return pkg.NewBusinessError(constants.ErrPodFileStream, "无法创建 Pod 文件流")
	}
	defer reader.Close()
	// 把流复制到响应
	if _, err := io.Copy(ctx.Writer, reader); err != nil {
		p.logger.Error("文件下载过程中出错", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrPodFileStream, "下载文件过程中发生错误")
	}
	return nil
}

func (p *podService) getRestConfig(ctx context.Context, clusterID int) (*rest.Config, error) {
	cluster, err := p.dao.GetClusterByID(ctx, clusterID)
	if err != nil {
		p.logger.Error("获取集群信息失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sClusterInfo, "无法获取集群信息")
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfigContent))
	if err != nil {
		p.logger.Error("解析 kubeconfig 失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sParseKubeConfig, "无法解析 kubeconfig")
	}
	restConfig.QPS = 100
	restConfig.Burst = 200
	return restConfig, nil
}

func (p *podService) buildDialer(restConfig *rest.Config, namespace, podName string) (httpstream.Dialer, error) {
	roundTripper, upgrader, err := spdy.RoundTripperFor(restConfig)
	if err != nil {
		return nil, pkg.NewBusinessError(constants.ErrK8sPortForward, err.Error())
	}
	// 基础 API URL
	baseURL, err := url.Parse(restConfig.Host)
	if err != nil {
		return nil, fmt.Errorf("非法 k8s API host: %w", err)
	}

	podPath := path.Join("/api/v1/namespaces", namespace, "pods", podName, "portforward")
	finalURL := baseURL.ResolveReference(&url.URL{Path: podPath})
	return spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, finalURL), nil
}

func buildPortsSpec(ports []model.PortForwardPort) []string {
	specs := make([]string, len(ports))
	for i, port := range ports {
		specs[i] = fmt.Sprintf("%d:%d", port.LocalPort, port.RemotePort)
	}
	return specs
}
