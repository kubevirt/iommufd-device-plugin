package tests

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	iommufdResourceName = "devices.kubevirt.io/iommufd"
	daemonSetName       = "iommufd-device-plugin"
	daemonSetNamespace  = "kube-system"
	pollInterval        = 2 * time.Second
	pollTimeout         = 5 * time.Minute
)

var _ = Describe("iommufd-device-plugin", func() {
	Context("DaemonSet", func() {
		It("should be deployed and ready", func() {
			Eventually(func(g Gomega) {
				ds := &appsv1.DaemonSet{}
				g.Expect(k8sClient.Get(context.Background(), client.ObjectKey{
					Namespace: daemonSetNamespace,
					Name:      daemonSetName,
				}, ds)).To(Succeed())
				g.Expect(ds.Status.DesiredNumberScheduled).To(BeNumerically(">", 0),
					"expected at least one node to schedule the DaemonSet")
				g.Expect(ds.Status.NumberReady).To(Equal(ds.Status.DesiredNumberScheduled),
					"expected all DaemonSet pods to be ready")
			}, pollTimeout, pollInterval).Should(Succeed())
		})
	})

	Context("device resource", func() {
		It("should be advertised on nodes", func() {
			Eventually(func(g Gomega) {
				nodeList := &corev1.NodeList{}
				g.Expect(k8sClient.List(context.Background(), nodeList)).To(Succeed())
				g.Expect(nodeList.Items).NotTo(BeEmpty(), "expected at least one node")

				for _, node := range nodeList.Items {
					capacity := node.Status.Capacity
					g.Expect(capacity).NotTo(BeNil(), "expected node %s to have capacity", node.Name)
					qty, ok := capacity[corev1.ResourceName(iommufdResourceName)]
					g.Expect(ok).To(BeTrue(),
						"expected node %s to advertise %s resource", node.Name, iommufdResourceName)
					g.Expect(qty.Value()).To(BeNumerically(">", 0),
						"expected node %s to have positive %s capacity", node.Name, iommufdResourceName)
				}
			}, pollTimeout, pollInterval).Should(Succeed())
		})
	})

	Context("pod allocation", func() {
		It("should allocate iommufd resource to a pod", func() {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "iommufd-test-consumer",
					Namespace: testNamespace,
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:    "test",
							Image:   "registry.k8s.io/pause:3.9",
							Command: []string{"/pause"},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceName(iommufdResourceName): resource.MustParse("1"),
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), pod)).To(Succeed())
			DeferCleanup(func() {
				_ = k8sClient.Delete(context.Background(), pod)
			})

			By("waiting for the pod to be running")
			Eventually(func(g Gomega) {
				updatedPod := &corev1.Pod{}
				g.Expect(k8sClient.Get(context.Background(), client.ObjectKeyFromObject(pod), updatedPod)).To(Succeed())
				g.Expect(updatedPod.Status.Phase).To(Equal(corev1.PodRunning),
					"expected pod to reach Running phase")
			}, pollTimeout, pollInterval).Should(Succeed())
		})
	})
})
