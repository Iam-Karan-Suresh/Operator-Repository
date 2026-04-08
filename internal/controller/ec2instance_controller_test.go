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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	computev1 "github.com/Iam-Karan-Suresh/operator-repo/api/v1"
)

// The Ec2Instance Controller test uses Ginkgo (BDD) and Gomega (Matchers).
// These tests run against 'envtest', which starts a real Kubernetes API Server and Etcd
// locally, but DOES NOT start a real AWS client. AWS calls should usually be mocked
// or verified in higher-level E2E tests.
var _ = Describe("Ec2Instance Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		ec2instance := &computev1.Ec2Instance{}

		BeforeEach(func() {
			// This block runs before every 'It' test case.
			// It ensures the custom resource exists in the mock API server.
			By("creating the custom resource for the Kind Ec2Instance")
			err := k8sClient.Get(ctx, typeNamespacedName, ec2instance)
			if err != nil && errors.IsNotFound(err) {
				resource := &computev1.Ec2Instance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					// Initial Spec configuration for testing
					Spec: computev1.Ec2InstanceSpec{
						InstanceType: "t3.micro",
						AMIId:        "ami-12345",
						Region:       "us-east-1",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &computev1.Ec2Instance{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Ec2Instance")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			// We manually invoke the Reconcile method for unit-testing the logic.
			// In a real cluster, the Manager handles this trigger automatically via watches.
			controllerReconciler := &Ec2InstanceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// After reconciliation, we should ideally verify if status was updated
			// or if expectations on external mocks were met.
		})
	})
})
