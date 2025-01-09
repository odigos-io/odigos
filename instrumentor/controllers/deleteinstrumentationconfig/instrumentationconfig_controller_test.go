package deleteinstrumentationconfig_test

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("deleteInstrumentationConfig InstrumentationConfig controller", func() {
	ctx := context.Background()
	var namespace *corev1.Namespace
	var deployment *appsv1.Deployment
	var instrumentationConfig *odigosv1.InstrumentationConfig

	Describe("Delete InstrumentationConfig", func() {

		When("Object created after deployment reconciled", func() {

			BeforeEach(func() {
				namespace = testutil.NewMockNamespace()
				Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())

				deployment = testutil.SetOdigosInstrumentationDisabled(testutil.NewMockTestDeployment(namespace))
				Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
			})

			It("InstrumentationConfig created for deployment which is not enabled", func() {

				instrumentationConfig = testutil.NewMockInstrumentationConfig(deployment)
				Expect(k8sClient.Create(ctx, instrumentationConfig)).Should(Succeed())

				testutil.AssertInstrumentationConfigDeleted(ctx, k8sClient, instrumentationConfig)
			})

		})

	})

})
