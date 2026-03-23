/*
Copyright 2026.

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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1 "github.com/ShaunakJoshi1407/ai-inference-orchestrator/api/v1"
)

func ptr[T any](v T) *T { return &v }

func reconcileResource(ctx context.Context, name string) {
	GinkgoHelper()
	controllerReconciler := &AIDeploymentReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
	}
	_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
		NamespacedName: types.NamespacedName{Name: name, Namespace: "default"},
	})
	Expect(err).NotTo(HaveOccurred())
}

var _ = Describe("AIDeployment Controller", func() {

	ctx := context.Background()

	AfterEach(func() {
		list := &infrav1.AIDeploymentList{}
		Expect(k8sClient.List(ctx, list)).To(Succeed())
		for i := range list.Items {
			Expect(k8sClient.Delete(ctx, &list.Items[i])).To(Succeed())
		}
	})

	Context("Deployment creation", func() {

		It("uses the default image when spec.image is not set", func() {
			cr := &infrav1.AIDeployment{
				ObjectMeta: metav1.ObjectMeta{Name: "default-image", Namespace: "default"},
				Spec:       infrav1.AIDeploymentSpec{Model: "tinyllama"},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			reconcileResource(ctx, cr.Name)

			deploy := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      cr.Name + "-deployment",
				Namespace: "default",
			}, deploy)).To(Succeed())

			Expect(deploy.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(deploy.Spec.Template.Spec.Containers[0].Image).To(Equal("ollama/ollama:latest"))
		})

		It("uses spec.image when provided", func() {
			cr := &infrav1.AIDeployment{
				ObjectMeta: metav1.ObjectMeta{Name: "custom-image", Namespace: "default"},
				Spec: infrav1.AIDeploymentSpec{
					Model: "mistral",
					Image: ptr("myregistry/custom-ollama:v2"),
				},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			reconcileResource(ctx, cr.Name)

			deploy := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      cr.Name + "-deployment",
				Namespace: "default",
			}, deploy)).To(Succeed())

			Expect(deploy.Spec.Template.Spec.Containers[0].Image).To(Equal("myregistry/custom-ollama:v2"))
		})

		It("sets the MODEL env var from spec.model", func() {
			cr := &infrav1.AIDeployment{
				ObjectMeta: metav1.ObjectMeta{Name: "model-env", Namespace: "default"},
				Spec:       infrav1.AIDeploymentSpec{Model: "llama3"},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			reconcileResource(ctx, cr.Name)

			deploy := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      cr.Name + "-deployment",
				Namespace: "default",
			}, deploy)).To(Succeed())

			envVars := deploy.Spec.Template.Spec.Containers[0].Env
			Expect(envVars).To(ContainElement(corev1.EnvVar{Name: "MODEL", Value: "llama3"}))
		})

	})

	Context("Service creation", func() {

		It("uses the default port (8080) when spec.port is not set", func() {
			cr := &infrav1.AIDeployment{
				ObjectMeta: metav1.ObjectMeta{Name: "default-port", Namespace: "default"},
				Spec:       infrav1.AIDeploymentSpec{Model: "tinyllama"},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			reconcileResource(ctx, cr.Name)

			svc := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      cr.Name + "-service",
				Namespace: "default",
			}, svc)).To(Succeed())

			Expect(svc.Spec.Ports).To(HaveLen(1))
			Expect(svc.Spec.Ports[0].Port).To(Equal(int32(8080)))
		})

		It("uses spec.port when provided", func() {
			cr := &infrav1.AIDeployment{
				ObjectMeta: metav1.ObjectMeta{Name: "custom-port", Namespace: "default"},
				Spec: infrav1.AIDeploymentSpec{
					Model: "tinyllama",
					Port:  ptr(int32(9090)),
				},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			reconcileResource(ctx, cr.Name)

			svc := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      cr.Name + "-service",
				Namespace: "default",
			}, svc)).To(Succeed())

			Expect(svc.Spec.Ports[0].Port).To(Equal(int32(9090)))
		})

		It("uses the default serviceType (ClusterIP) when spec.serviceType is not set", func() {
			cr := &infrav1.AIDeployment{
				ObjectMeta: metav1.ObjectMeta{Name: "default-svctype", Namespace: "default"},
				Spec:       infrav1.AIDeploymentSpec{Model: "tinyllama"},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			reconcileResource(ctx, cr.Name)

			svc := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      cr.Name + "-service",
				Namespace: "default",
			}, svc)).To(Succeed())

			Expect(svc.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
		})

		It("uses spec.serviceType when provided", func() {
			cr := &infrav1.AIDeployment{
				ObjectMeta: metav1.ObjectMeta{Name: "custom-svctype", Namespace: "default"},
				Spec: infrav1.AIDeploymentSpec{
					Model:       "tinyllama",
					ServiceType: ptr(corev1.ServiceTypeNodePort),
				},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			reconcileResource(ctx, cr.Name)

			svc := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      cr.Name + "-service",
				Namespace: "default",
			}, svc)).To(Succeed())

			Expect(svc.Spec.Type).To(Equal(corev1.ServiceTypeNodePort))
		})

		It("reconciles service port drift when spec.port changes", func() {
			cr := &infrav1.AIDeployment{
				ObjectMeta: metav1.ObjectMeta{Name: "port-drift", Namespace: "default"},
				Spec: infrav1.AIDeploymentSpec{
					Model: "tinyllama",
					Port:  ptr(int32(8080)),
				},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			reconcileResource(ctx, cr.Name)

			// update the port
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cr.Name, Namespace: "default"}, cr)).To(Succeed())
			cr.Spec.Port = ptr(int32(9999))
			Expect(k8sClient.Update(ctx, cr)).To(Succeed())
			reconcileResource(ctx, cr.Name)

			svc := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      cr.Name + "-service",
				Namespace: "default",
			}, svc)).To(Succeed())

			Expect(svc.Spec.Ports[0].Port).To(Equal(int32(9999)))
		})

	})

	Context("Basic reconciliation", func() {

		It("creates owned Deployment and Service for a minimal spec", func() {
			cr := &infrav1.AIDeployment{
				ObjectMeta: metav1.ObjectMeta{Name: "minimal", Namespace: "default"},
				Spec:       infrav1.AIDeploymentSpec{Model: "tinyllama"},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			reconcileResource(ctx, cr.Name)

			deploy := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      cr.Name + "-deployment",
				Namespace: "default",
			}, deploy)).To(Succeed())

			svc := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      cr.Name + "-service",
				Namespace: "default",
			}, svc)).To(Succeed())
		})

		It("does not return an error for a missing resource", func() {
			controllerReconciler := &AIDeploymentReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "does-not-exist", Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("is idempotent — reconciling twice produces no error", func() {
			cr := &infrav1.AIDeployment{
				ObjectMeta: metav1.ObjectMeta{Name: "idempotent", Namespace: "default"},
				Spec:       infrav1.AIDeploymentSpec{Model: "tinyllama"},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())

			err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.Name, Namespace: "default"}, cr)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			}

			reconcileResource(ctx, cr.Name)
			reconcileResource(ctx, cr.Name)
		})

	})
})
