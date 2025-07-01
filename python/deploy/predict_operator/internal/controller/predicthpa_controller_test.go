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
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	hpav1 "github.com/lostar01/hpa-operator/api/v1"
)

var _ = Describe("PredictHPA Controller", func() {
	Context("当调谐资源时", func() {
		const resourceName = "test-predicthpa"
		const deploymentName = "test-deployment"
		const namespace = "default"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: namespace,
		}
		predicthpa := &hpav1.PredictHPA{}
		deployment := &appsv1.Deployment{}
		var mockServer *httptest.Server

		BeforeEach(func() {
			By("创建测试用的 Deployment 资源")
			initialReplicas := int32(1)
			deployment = &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      deploymentName,
					Namespace: namespace,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &initialReplicas,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "test"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test-container",
									Image: "nginx:latest",
								},
							},
						},
					},
				},
			}
			err := k8sClient.Create(ctx, deployment)
			Expect(err).NotTo(HaveOccurred())

			By("创建模拟服务器")
			mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"instances": 3}`))
			}))

			By("创建 PredictHPA 自定义资源")
			err = k8sClient.Get(ctx, typeNamespacedName, predicthpa)
			if err != nil && errors.IsNotFound(err) {
				resource := &hpav1.PredictHPA{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: namespace,
					},
					Spec: hpav1.PredictHPASpec{
						DeploymentName:  deploymentName,
						DeployNamespace: namespace,
						PredictHost:     mockServer.URL[7:], // 去掉 "http://" 前缀
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			By("关闭模拟服务器")
			if mockServer != nil {
				mockServer.Close()
			}

			By("清理 PredictHPA 资源")
			resource := &hpav1.PredictHPA{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}

			By("清理 Deployment 资源")
			deploymentNN := types.NamespacedName{Name: deploymentName, Namespace: namespace}
			err = k8sClient.Get(ctx, deploymentNN, deployment)
			if err == nil {
				Expect(k8sClient.Delete(ctx, deployment)).To(Succeed())
			}
		})

		It("应该成功调谐资源并更新 Deployment 副本数", func() {
			By("调谐创建的资源")
			controllerReconciler := &PredictHPAReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("验证 Deployment 副本数已被更新")
			updatedDeployment := &appsv1.Deployment{}
			deploymentNN := types.NamespacedName{Name: deploymentName, Namespace: namespace}
			Eventually(func() int32 {
				err := k8sClient.Get(ctx, deploymentNN, updatedDeployment)
				if err != nil {
					return 0
				}
				return *updatedDeployment.Spec.Replicas
			}, "10s", "1s").Should(Equal(int32(3)))
		})
	})
})
