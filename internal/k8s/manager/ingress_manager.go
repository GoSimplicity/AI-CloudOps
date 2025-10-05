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
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type IngressManager interface {
	CreateIngress(ctx context.Context, clusterID int, namespace string, ingress *networkingv1.Ingress) error
	GetIngress(ctx context.Context, clusterID int, namespace, name string) (*networkingv1.Ingress, error)
	GetIngressList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sIngress, error)
	UpdateIngress(ctx context.Context, clusterID int, namespace string, ingress *networkingv1.Ingress) error
	DeleteIngress(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error
}

type ingressManager struct {
	clientFactory client.K8sClient
	logger        *zap.Logger
}

func NewIngressManager(clientFactory client.K8sClient, logger *zap.Logger) IngressManager {
	return &ingressManager{
		clientFactory: clientFactory,
		logger:        logger,
	}
}

// getKubeClient 私有方法：获取Kubernetes客户端
func (m *ingressManager) getKubeClient(clusterID int) (*kubernetes.Clientset, error) {
	kubeClient, err := m.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}
	return kubeClient, nil
}

func (m *ingressManager) CreateIngress(ctx context.Context, clusterID int, namespace string, ingress *networkingv1.Ingress) error {
	if ingress == nil {
		return fmt.Errorf("ingress 不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 如果ingress对象中没有指定namespace，使用参数中的namespace
	targetNamespace := ingress.Namespace
	if targetNamespace == "" {
		targetNamespace = namespace
		ingress.Namespace = namespace
	}

	_, err = kubeClient.NetworkingV1().Ingresses(targetNamespace).Create(ctx, ingress, metav1.CreateOptions{})
	if err != nil {
		m.logger.Error("创建 Ingress 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", targetNamespace),
			zap.String("name", ingress.Name),
			zap.Error(err))
		return fmt.Errorf("创建 Ingress 失败: %w", err)
	}

	m.logger.Info("成功创建 Ingress",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", targetNamespace),
		zap.String("name", ingress.Name))
	return nil
}

func (m *ingressManager) GetIngress(ctx context.Context, clusterID int, namespace, name string) (*networkingv1.Ingress, error) {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	ingress, err := kubeClient.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取 Ingress 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, fmt.Errorf("获取 Ingress 失败: %w", err)
	}

	m.logger.Debug("成功获取 Ingress",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return ingress, nil
}

func (m *ingressManager) GetIngressList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sIngress, error) {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	ingressList, err := kubeClient.NetworkingV1().Ingresses(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取 Ingress 列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Error(err))
		return nil, fmt.Errorf("获取 Ingress 列表失败: %w", err)
	}

	var k8sIngresses []*model.K8sIngress
	for _, ingress := range ingressList.Items {
		k8sIngress := utils.ConvertToK8sIngress(&ingress, clusterID)
		k8sIngresses = append(k8sIngresses, k8sIngress)
	}

	m.logger.Debug("成功获取 Ingress 列表",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(k8sIngresses)))
	return k8sIngresses, nil
}

func (m *ingressManager) UpdateIngress(ctx context.Context, clusterID int, namespace string, ingress *networkingv1.Ingress) error {
	if ingress == nil {
		return fmt.Errorf("ingress 不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 如果ingress对象中没有指定namespace，使用参数中的namespace
	targetNamespace := ingress.Namespace
	if targetNamespace == "" {
		targetNamespace = namespace
		ingress.Namespace = namespace
	}

	_, err = kubeClient.NetworkingV1().Ingresses(targetNamespace).Update(ctx, ingress, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新 Ingress 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", targetNamespace),
			zap.String("name", ingress.Name),
			zap.Error(err))
		return fmt.Errorf("更新 Ingress 失败: %w", err)
	}

	m.logger.Info("成功更新 Ingress",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", targetNamespace),
		zap.String("name", ingress.Name))
	return nil
}

func (m *ingressManager) DeleteIngress(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = kubeClient.NetworkingV1().Ingresses(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		m.logger.Error("删除 Ingress 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("删除 Ingress 失败: %w", err)
	}

	m.logger.Info("成功删除 Ingress",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return nil
}
