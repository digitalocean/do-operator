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

package main

import (
	"context"
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"golang.org/x/oauth2"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/digitalocean/godo"

	databasesv1alpha1 "github.com/digitalocean/do-operator/api/v1alpha1"
	"github.com/digitalocean/do-operator/api/webhooks"
	"github.com/digitalocean/do-operator/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
	version  string
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(databasesv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var (
		metricsAddr          string
		enableLeaderElection bool
		probeAddr            string
		doAPIToken           string
		doAPIURL             string
	)
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&doAPIToken, "do-api-token", "", "DigitalOcean public API token for managing resources.")
	flag.StringVar(&doAPIURL, "do-api-url", "https://api.digitalocean.com", "Base URL of the DigitalOcean API.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	if version == "" {
		version = "dev"
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)).WithValues("version", version))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		WebhookServer: &webhook.DefaultServer{
			Options: webhook.Options{
				Port: 9443,
			},
		},
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "f103eaf3.digitalocean.com",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	godoClient, err := makeGodo(context.Background(), doAPIToken, doAPIURL)
	if err != nil {
		setupLog.Error(err, "unable to create godo client")
		os.Exit(1)
	}

	if err = (&controllers.DatabaseClusterReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		GodoClient: godoClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DatabaseCluster")
		os.Exit(1)
	}
	if err = (&controllers.DatabaseClusterReferenceReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		GodoClient: godoClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DatabaseClusterReference")
		os.Exit(1)
	}
	if err = webhooks.SetupDatabaseClusterReferenceWebhookWithManager(mgr, godoClient); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "DatabaseClusterReference")
		os.Exit(1)
	}
	if err = (&controllers.DatabaseUserReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		GodoClient: godoClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DatabaseUser")
		os.Exit(1)
	}
	if err = webhooks.SetupDatabaseUserWebhookWithManager(mgr, godoClient); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "DatabaseUser")
		os.Exit(1)
	}
	if err = (&controllers.DatabaseUserReferenceReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		GodoClient: godoClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DatabaseUserReference")
		os.Exit(1)
	}
	if err = webhooks.SetupDatabaseUserReferenceWebhookWithManager(mgr, godoClient); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "DatabaseUserReference")
		os.Exit(1)
	}
	if err = webhooks.SetupDatabaseClusterWebhookWithManager(mgr, godoClient); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "DatabaseCluster")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.WithValues("version", version).Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func makeGodo(ctx context.Context, token, addr string) (*godo.Client, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
	return godo.New(client, godo.SetBaseURL(addr), godo.SetUserAgent("do-operator/"+version))
}
