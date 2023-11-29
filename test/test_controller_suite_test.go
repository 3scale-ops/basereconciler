package test

import (
	"context"

	"github.com/3scale-ops/basereconciler/test/api/v1alpha1"
	"github.com/3scale-ops/basereconciler/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Test controller", func() {
	var namespace string
	var instance *v1alpha1.Test

	BeforeEach(func() {
		// Create a namespace for each block
		namespace = "test-ns-" + nameGenerator.Generate()
		n := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
		err := k8sClient.Create(context.Background(), n)
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() error {
			return k8sClient.Get(context.Background(), types.NamespacedName{Name: namespace}, n)
		}, timeout, poll).ShouldNot(HaveOccurred())

	})

	AfterEach(func() {
		n := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
		err := k8sClient.Delete(context.Background(), n, client.PropagationPolicy(metav1.DeletePropagationForeground))
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Creates resources", func() {

		BeforeEach(func() {
			By("creating a Test simple resource")
			instance = &v1alpha1.Test{
				ObjectMeta: metav1.ObjectMeta{Name: "instance", Namespace: namespace},
				Spec: v1alpha1.TestSpec{
					PDB: pointer.Bool(true),
					HPA: pointer.Bool(true),
				},
			}
			err := k8sClient.Create(context.Background(), instance)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: "instance", Namespace: namespace}, instance)
			}, timeout, poll).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			k8sClient.Delete(context.Background(), instance, client.PropagationPolicy(metav1.DeletePropagationForeground))
		})

		It("creates the required resources", func() {

			dep := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: "deployment", Namespace: namespace},
					dep,
				)
			}, timeout, poll).ShouldNot(HaveOccurred())

			svc := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: "service", Namespace: namespace},
					svc,
				)
			}, timeout, poll).ShouldNot(HaveOccurred())

			pdb := &policyv1.PodDisruptionBudget{}
			Eventually(func() error {
				return k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: "pdb", Namespace: namespace},
					pdb,
				)
			}, timeout, poll).ShouldNot(HaveOccurred())

			hpa := &autoscalingv2.HorizontalPodAutoscaler{}
			Eventually(func() error {
				return k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: "hpa", Namespace: namespace},
					hpa,
				)
			}, timeout, poll).ShouldNot(HaveOccurred())
		})

		It("triggers a Deployment rollout on Secret contents change", func() {

			dep := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: "deployment", Namespace: namespace},
					dep,
				)
			}, timeout, poll).ShouldNot(HaveOccurred())

			// Annotations should be empty when Secret does not exists
			value, ok := dep.Spec.Template.ObjectMeta.Annotations["example.com/secret.secret-hash"]
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal(""))

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: namespace},
				Type:       corev1.SecretTypeOpaque,
				Data:       map[string][]byte{"KEY": []byte("value")},
			}
			err := k8sClient.Create(context.Background(), secret)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				err := k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: "deployment", Namespace: namespace},
					dep,
				)
				Expect(err).ToNot(HaveOccurred())
				value, ok := dep.Spec.Template.ObjectMeta.Annotations["example.com/secret.secret-hash"]
				Expect(ok).To(BeTrue())
				// Value of the annotation should be the hash of the Secret contents
				return value == util.Hash(secret.Data)
			}, timeout, poll).Should(BeTrue())

			patch := client.MergeFrom(secret.DeepCopy())
			secret.Data = map[string][]byte{"KEY": []byte("new-value")}
			err = k8sClient.Patch(context.Background(), secret, patch)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				err := k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: "deployment", Namespace: namespace},
					dep,
				)
				Expect(err).ToNot(HaveOccurred())
				value, ok := dep.Spec.Template.ObjectMeta.Annotations["example.com/secret.secret-hash"]
				Expect(ok).To(BeTrue())
				// Value of the annotation should be the hash of the Secret new contents
				return value == util.Hash(secret.Data)
			}, timeout, poll).Should(BeTrue())
		})

		It("deletes specific resources when disabled", func() {
			// Wait for resources to be created
			pdb := &policyv1.PodDisruptionBudget{}
			Eventually(func() error {
				return k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: "pdb", Namespace: namespace},
					pdb,
				)
			}, timeout, poll).ShouldNot(HaveOccurred())
			hpa := &autoscalingv2.HorizontalPodAutoscaler{}
			Eventually(func() error {
				return k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: "hpa", Namespace: namespace},
					hpa,
				)
			}, timeout, poll).ShouldNot(HaveOccurred())

			// disable pdb and hpa
			instance = &v1alpha1.Test{}
			Eventually(func() error {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "instance", Namespace: namespace}, instance)
				if err != nil {
					return err
				}
				instance.Spec.PDB = pointer.Bool(false)
				instance.Spec.HPA = pointer.Bool(false)
				err = k8sClient.Update(context.Background(), instance)
				return err

			}, timeout, poll).ShouldNot(HaveOccurred())

			Eventually(func() error {
				return k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: "pdb", Namespace: namespace},
					pdb,
				)
			}, timeout, poll).Should(HaveOccurred())

			Eventually(func() error {
				return k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: "hpa", Namespace: namespace},
					hpa,
				)
			}, timeout, poll).Should(HaveOccurred())

		})

		It("deletes all owned resources when custom resource is deleted", func() {
			// Wait for all resources to be created

			dep := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: "deployment", Namespace: namespace},
					dep,
				)
			}, timeout, poll).ShouldNot(HaveOccurred())

			svc := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: "service", Namespace: namespace},
					svc,
				)
			}, timeout, poll).ShouldNot(HaveOccurred())

			// Delete the custom resource
			err := k8sClient.Delete(context.Background(), instance)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "instance", Namespace: namespace}, instance)
				if err != nil && errors.IsNotFound(err) {
					return true
				}
				return false
			}, timeout, poll).Should(BeTrue())

		})

		It("updates service annotations", func() {
			svc := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: "service", Namespace: namespace},
					svc,
				)
			}, timeout, poll).ShouldNot(HaveOccurred())

			Eventually(func() error {
				return k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: "instance", Namespace: namespace},
					instance,
				)
			}, timeout, poll).ShouldNot(HaveOccurred())

			patch := client.MergeFrom(instance.DeepCopy())
			instance.Spec.ServiceAnnotations = map[string]string{"key": "value"}
			err := k8sClient.Patch(context.Background(), instance, patch)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				err := k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: "service", Namespace: namespace},
					svc,
				)
				Expect(err).ToNot(HaveOccurred())
				return svc.GetAnnotations()["key"] == "value"
			}, timeout, poll).Should(BeTrue())
		})

		It("updates the status of the custom resource", func() {
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: instance.GetName(), Namespace: instance.GetNamespace()}, instance)
				Expect(err).ToNot(HaveOccurred())
				return instance.Status.DeploymentStatus != nil
			}, timeout, poll).Should(BeTrue())
		})
	})

})
