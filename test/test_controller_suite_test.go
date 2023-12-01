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
	var resources []client.Object

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

			By("checking that owned resources are created")
			resources = []client.Object{
				&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "deployment", Namespace: namespace}},
				&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "service", Namespace: namespace}},
				&policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Name: "pdb", Namespace: namespace}},
				&autoscalingv2.HorizontalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Name: "hpa", Namespace: namespace}},
			}

			for _, res := range resources {
				Eventually(func() error {
					return k8sClient.Get(context.Background(), util.ObjectKey(res), res)
				}, timeout, poll).ShouldNot(HaveOccurred())
			}
		})

		AfterEach(func() {
			k8sClient.Delete(context.Background(), instance, client.PropagationPolicy(metav1.DeletePropagationForeground))
		})

		It("triggers a Deployment rollout on Secret contents change", func() {

			dep := resources[0].(*appsv1.Deployment)
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
			pdb := resources[2].(*policyv1.PodDisruptionBudget)
			hpa := resources[3].(*autoscalingv2.HorizontalPodAutoscaler)

			// disable pdb and hpa
			patch := client.MergeFrom(instance.DeepCopy())
			instance.Spec.PDB = util.Pointer(false)
			instance.Spec.HPA = util.Pointer(false)
			err := k8sClient.Patch(context.Background(), instance, patch)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() error {
				return k8sClient.Get(context.Background(), util.ObjectKey(pdb), pdb)
			}, timeout, poll).Should(HaveOccurred())

			Eventually(func() error {
				return k8sClient.Get(context.Background(), util.ObjectKey(hpa), hpa)
			}, timeout, poll).Should(HaveOccurred())

		})

		It("updates service annotations", func() {
			svc := resources[1].(*corev1.Service)

			patch := client.MergeFrom(instance.DeepCopy())
			instance.Spec.ServiceAnnotations = map[string]string{"key": "value"}
			err := k8sClient.Patch(context.Background(), instance, patch)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				if err := k8sClient.Get(context.Background(), util.ObjectKey(svc), svc); err != nil {
					return false
				}
				return svc.GetAnnotations()["key"] == "value"
			}, timeout, poll).Should(BeTrue())
		})

		It("prunes the service", func() {

			patch := client.MergeFrom(instance.DeepCopy())
			instance.Spec.PruneService = util.Pointer(true)
			err := k8sClient.Patch(context.Background(), instance, patch)
			Expect(err).ToNot(HaveOccurred())

			svc := resources[1].(*corev1.Service)
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), util.ObjectKey(svc), svc)
				if err != nil && errors.IsNotFound(err) {
					return true
				}
				return false
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
