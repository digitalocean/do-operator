/*
Copyright 2022 DigitalOcean.

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

package v1alpha1

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/digitalocean/do-operator/extgodo"
	"github.com/digitalocean/do-operator/fakegodo"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	//+kubebuilder:scaffold:imports
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg                  *rest.Config
	k8sClient            client.Client
	testEnv              *envtest.Environment
	ctx                  context.Context
	cancel               context.CancelFunc
	fakeDatabasesService = &fakegodo.FakeDatabasesService{}
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Webhook Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: false,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "config", "webhook")},
		},
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	scheme := runtime.NewScheme()
	err = AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = admissionv1beta1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// start webhook server using Manager
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme,
		Host:               webhookInstallOptions.LocalServingHost,
		Port:               webhookInstallOptions.LocalServingPort,
		CertDir:            webhookInstallOptions.LocalServingCertDir,
		LeaderElection:     false,
		MetricsBindAddress: "0",
	})
	Expect(err).NotTo(HaveOccurred())

	godoServer := httptest.NewServer(&fakegodo.Handler{
		DatabaseOptions: &extgodo.DatabaseOptions{
			OptionsByEngine: map[string]extgodo.DatabaseEngineOptions{
				"mongodb": extgodo.DatabaseEngineOptions{
					Regions:  []string{"dev0"},
					Versions: []string{"6"},
					Layouts: []*extgodo.DatabaseLayout{{
						NumNodes: 1,
						Sizes: []string{
							"db-s-1vcpu-1gb",
						},
					}},
				},
				"mysql": extgodo.DatabaseEngineOptions{
					Regions:  []string{"dev0"},
					Versions: []string{"6"},
					Layouts: []*extgodo.DatabaseLayout{
						{
							NumNodes: 1,
							Sizes: []string{
								"db-s-1vcpu-1gb",
								"db-s-2vcpu-2gb",
							},
						},
						{
							NumNodes: 2,
							Sizes: []string{
								"db-s-1vcpu-1gb",
								"db-s-2vcpu-2gb",
							},
						},
					},
				},
				"redis": extgodo.DatabaseEngineOptions{
					Regions:  []string{"dev0"},
					Versions: []string{"6"},
					Layouts: []*extgodo.DatabaseLayout{{
						NumNodes: 1,
						Sizes: []string{
							"db-s-1vcpu-1gb",
						},
					}},
				},
			},
		},
	})

	godoClient, err := godo.New(http.DefaultClient, godo.SetBaseURL(godoServer.URL))
	Expect(err).NotTo(HaveOccurred())
	godoClient.Databases = fakeDatabasesService

	err = (&DatabaseClusterReference{}).SetupWebhookWithManager(mgr, godoClient)
	Expect(err).NotTo(HaveOccurred())

	err = (&DatabaseUser{}).SetupWebhookWithManager(mgr, godoClient)
	Expect(err).NotTo(HaveOccurred())

	err = (&DatabaseUserReference{}).SetupWebhookWithManager(mgr, godoClient)
	Expect(err).NotTo(HaveOccurred())

	err = (&DatabaseCluster{}).SetupWebhookWithManager(mgr, godoClient)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:webhook

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	// wait for the webhook server to get ready
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)
	Eventually(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	}).Should(Succeed())

	// Create fixtures for tests.
	createUserWebhookTestFixtures()
}, 60)

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
