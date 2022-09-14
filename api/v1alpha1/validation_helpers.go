package v1alpha1

import (
	"sync"

	"github.com/digitalocean/godo"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// godoClient is the godo client used to validate objects in validating
	// webhooks. It's a package global because there's no tidy way to scope it to a
	// specific webhook, and it's concurrency-safe anyway.
	godoClient     *godo.Client
	godoClientOnce sync.Once

	// webhookClient is the k8s client used to validate objects in validating
	// webhooks. It's a package global because there's no tidy way to scope it
	// to a specific webhook, and it's concurrency-safe anyway.
	webhookClient     client.Client
	webhookClientOnce sync.Once
)

// initGlobalGodoClient initializes the package-global godo client. It should be
// called by each webhook setup function.
func initGlobalGodoClient(client *godo.Client) {
	godoClientOnce.Do(func() {
		godoClient = client
	})
}

// initGlobalK8sClient initializes the package-global k8s client. It should be
// called by the webhook setup functions for all webhooks that require a k8s
// client.
func initGlobalK8sClient(cl client.Client) {
	webhookClientOnce.Do(func() {
		webhookClient = cl
	})
}

func engineOptsFromOptions(opts *godo.DatabaseOptions, engine string) (godo.DatabaseEngineOptions, bool) {
	switch engine {
	case "mysql":
		return opts.MySQLOptions, true
	case "pg":
		return opts.PostgresSQLOptions, true
	case "redis":
		return opts.RedisOptions, true
	case "mongodb":
		return opts.MongoDBOptions, true
	}
	return godo.DatabaseEngineOptions{}, false
}
