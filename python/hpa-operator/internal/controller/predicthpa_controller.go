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

	// 获取到 predictHPQ
	predictHPA := &hpav1.PredictHPA{}
	if err := r.Get(ctx, req.NamespacedName, predictHPA); err != nil {
		logger.Error(err, "unable to fetch")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: predictHPA.Spec.DeploymentName, Namespace: predictHPA.Spec.DeployNamespace}, deployment); err != nil {
		logger.Error(err, "unable to fetch deployment")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 请求推理服务，获取副本数
	recommendedReplicas, err := getRecommendedReplicas(predictHPA.Spec.PredictHost)
	if err != nil {
		logger.Error(err, "fail to get recommend replicas")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// 和工作负载的副本数座对比
	if *&deployment.Spec.Replicas != &recommendedReplicas {
		deployment.Spec.Replicas = &recommendedReplicas
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.Update(ctx, deployment)
		})
		if err != nil {
			logger.Error(err, "failed to update deployment")
			return ctrl.Result{}, err
		}
	}

	logger.Info("Update deployment replicas", "replicas", recommendedReplicas)

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func getRecommendedReplicas(predictHost string) (int32, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/predict", predictHost))
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var data struct {
		Instances int32 `json:"instances"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return 0, err
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
