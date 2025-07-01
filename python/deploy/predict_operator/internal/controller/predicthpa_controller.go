/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	hpav1 "github.com/lostar01/hpa-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
)

// PredictHPAReconciler reconciles a PredictHPA object
type PredictHPAReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=hpa.aiops.com,resources=predicthpas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=hpa.aiops.com,resources=predicthpas/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=hpa.aiops.com,resources=predicthpas/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PredictHPA object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *PredictHPAReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 获取 PredictHPA 对象
	predictHPA := &hpav1.PredictHPA{}
	if err := r.Get(ctx, req.NamespacedName, predictHPA); err != nil {
		logger.Error(err, "无法获取 PredictHPA 对象")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 获取关联的 Deployment 对象
	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: predictHPA.Spec.DeploymentName, Namespace: predictHPA.Spec.DeployNamespace}, deployment); err != nil {
		logger.Error(err, "无法获取关联的 Deployment 对象", "部署名称", predictHPA.Spec.DeploymentName, "命名空间", predictHPA.Spec.DeployNamespace)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 请求推理服务，获取推荐的副本数
	recommendedReplicas, err := getRecommendedReplicas(predictHPA.Spec.PredictHost)
	if err != nil {
		logger.Error(err, "获取推荐副本数失败", "预测服务地址", predictHPA.Spec.PredictHost)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// 对比当前副本数与推荐副本数
	currentReplicas := deployment.Spec.Replicas
	if currentReplicas == nil || *currentReplicas != recommendedReplicas {
		// 更新 Deployment 副本数
		deployment.Spec.Replicas = &recommendedReplicas
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.Update(ctx, deployment)
		})
		if err != nil {
			logger.Error(err, "更新 Deployment 副本数失败", "部署名称", deployment.Name, "命名空间", deployment.Namespace)
			return ctrl.Result{}, err
		}
		logger.Info("已更新 Deployment 副本数", "部署名称", deployment.Name, "命名空间", deployment.Namespace, "原副本数", currentReplicas, "新副本数", recommendedReplicas)
	} else {
		logger.Info("副本数无需调整", "部署名称", deployment.Name, "当前副本数", *currentReplicas)
	}

	// 30秒后重新调谐
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// 获取推荐的副本数
func getRecommendedReplicas(predictHost string) (int32, error) {
	// 构建请求URL
	predictURL := fmt.Sprintf("http://%s/predict", predictHost)

	// 发送GET请求
	resp, err := http.Get(predictURL)
	if err != nil {
		return 0, fmt.Errorf("请求预测服务失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("预测服务返回非200状态码: %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("读取响应体失败: %w", err)
	}

	// 解析JSON响应
	var data struct {
		Instances int32 `json:"instances"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, fmt.Errorf("解析JSON响应失败: %w", err)
	}

	// 确保副本数不小于1
	if data.Instances < 1 {
		return 1, nil
	}

	return data.Instances, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PredictHPAReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hpav1.PredictHPA{}).
		Named("predicthpa").
		Complete(r)
}
