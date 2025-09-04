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
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils/query"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	//"gopkg.in/yaml.v3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IngressService interface {
	GetIngressList(ctx context.Context, queryParam *query.Query, req *model.GetIngressListReq) (*model.ListResp[*model.K8sIngress], error)
	GetIngressDetails(ctx context.Context, req *model.GetIngressDetailsReq) (*model.K8sIngress, error)
	GetIngressYaml(ctx context.Context, req *model.GetIngressYamlReq) (string, error)
	CreateIngress(ctx context.Context, req *model.K8sIngressCreateOrUpdateReq) error
	UpdateIngress(ctx context.Context, req *model.K8sIngressCreateOrUpdateReq) error
	DeleteIngress(ctx context.Context, req *model.K8sIngressDeleteReq) error
	BatchDeleteIngresses(ctx context.Context, req *model.K8sIngressBatchDeleteReq) error
	TestIngressTLS(ctx context.Context, req *model.K8sIngressTLSTestReq) (*model.K8sTLSTestResult, error)
	CheckIngressBackendHealth(ctx context.Context, req *model.K8sIngressBackendHealthReq) ([]*model.K8sBackendHealth, error)
}

type ingressService struct {
	dao            dao.ClusterDAO         // 保持对DAO的依赖
	client         client.K8sClient       // 保持向后兼容
	ingressManager manager.IngressManager // 新的依赖注入
	logger         *zap.Logger
}

// NewIngressService 创建新的 IngressService 实例
func NewIngressService(dao dao.ClusterDAO, client client.K8sClient, ingressManager manager.IngressManager, logger *zap.Logger) IngressService {
	return &ingressService{
		dao:            dao,
		client:         client,
		ingressManager: ingressManager,
		logger:         logger,
	}
}

// GetIngressList 获取Ingress列表
func (i *ingressService) GetIngressList(ctx context.Context, queryParam *query.Query, req *model.GetIngressListReq) (*model.ListResp[*model.K8sIngress], error) {

	_ = queryParam.AppendLabelSelector(req.Labels)

	if req.Namespace == "" {
		req.Namespace = corev1.NamespaceAll
	}

	// 使用 IngressManager 获取列表
	ingresses, err := i.ingressManager.GetIngressList(ctx, req.ClusterID, req.Namespace, queryParam)
	if err != nil {
		i.logger.Error("获取Ingress列表失败",
			zap.String("namespace", req.Namespace),
			zap.Error(err))

		return nil, pkg.NewBusinessError(constants.ErrK8sResourceList, "获取Ingress列表失败")
	}
	return ingresses, nil
}

// GetIngressDetails 获取单个Ingress详情
func (i *ingressService) GetIngressDetails(ctx context.Context, req *model.GetIngressDetailsReq) (*model.K8sIngress, error) {

	ingress, err := i.ingressManager.GetIngress(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}

	return utils.ConvertToK8sIngress(ingress, req.ClusterID), nil
}

// GetIngressYaml 获取Ingress的YAML
func (i *ingressService) GetIngressYaml(ctx context.Context, req *model.GetIngressYamlReq) (string, error) {

	ingress, err := i.ingressManager.GetIngress(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return "", err
	}

	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(ingress)
	if err != nil {
		return "", pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取Ingress yaml失败")
	}

	unstructuredObj := &unstructured.Unstructured{Object: m}

	d, err := utils.ConvertUnstructuredToYAML(unstructuredObj)
	if err != nil {
		i.logger.Error("序列化Ingress为yaml失败", zap.Error(err))
		return "", pkg.NewBusinessError(constants.ErrK8sResourceOperation, "序列化Ingress失败")
	}

	return d, nil
}

// CreateIngress TODO 创建Ingress
func (i *ingressService) CreateIngress(ctx context.Context, req *model.K8sIngressCreateOrUpdateReq) error {

	cfg, err := i.getRestConfig(ctx, req.ClusterID)
	if err != nil {
		return err
	}

	if err = i.ingressManager.CreateIngress(ctx, req.ClusterID, req.Namespace, cfg, req.IngressYaml); err != nil {
		return err
	}

	i.logger.Info("成功创建Ingress",
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

// UpdateIngress TODO 更新Ingress
func (i *ingressService) UpdateIngress(ctx context.Context, req *model.K8sIngressCreateOrUpdateReq) error {

	cfg, err := i.getRestConfig(ctx, req.ClusterID)
	if err != nil {
		return err
	}

	if err = i.ingressManager.UpdateIngress(ctx, req.ClusterID, req.Namespace, cfg, req.IngressYaml); err != nil {
		return err
	}

	i.logger.Info("成功更新Ingress",
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// DeleteIngress 删除Ingress
func (i *ingressService) DeleteIngress(ctx context.Context, req *model.K8sIngressDeleteReq) error {

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

	return i.ingressManager.DeleteIngress(ctx, req.ClusterID, req.Namespace, req.Name, deleteOpts)
}

// BatchDeleteIngresses 批量删除Ingress
func (i *ingressService) BatchDeleteIngresses(ctx context.Context, req *model.K8sIngressBatchDeleteReq) error {

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

	return i.ingressManager.BatchDeleteIngresses(ctx, req.ClusterID, req.Namespace, req.Names, deleteOpts)
}

// TestIngressTLS 测试Ingress TLS配置
func (i *ingressService) TestIngressTLS(ctx context.Context, req *model.K8sIngressTLSTestReq) (*model.K8sTLSTestResult, error) {
	host := req.Host
	port := req.Port
	if port == 0 {
		port = 443
	}

	checkResult, err := i.ingressManager.TestIngressTLS(ctx, req.ClusterID, host, port)
	if err != nil {
		i.logger.Error("failed to test ingress tls",
			zap.Error(err), zap.String("host", host), zap.Int("port", port))
	}

	return checkResult, err
}

// CheckIngressBackendHealth 检查Ingress后端健康状态
func (i *ingressService) CheckIngressBackendHealth(ctx context.Context, req *model.K8sIngressBackendHealthReq) ([]*model.K8sBackendHealth, error) {

	healths, err := i.ingressManager.CheckIngressBackendHealth(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		i.logger.Error("Ingress健康检查失败",
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))

		return nil, pkg.NewBusinessError(constants.ErrorK8sIngressHealthFailed, "Ingress健康检查失败")
	}
	return healths, nil
}

func (i *ingressService) getRestConfig(ctx context.Context, clusterID int) (*rest.Config, error) {
	cluster, err := i.dao.GetClusterByID(ctx, clusterID)
	if err != nil {
		i.logger.Error("获取集群信息失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sClusterInfo, "无法获取集群信息")
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfigContent))
	if err != nil {
		i.logger.Error("解析 kubeconfig 失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sParseKubeConfig, "无法解析 kubeconfig")
	}
	restConfig.QPS = 100
	restConfig.Burst = 200
	return restConfig, nil
}
