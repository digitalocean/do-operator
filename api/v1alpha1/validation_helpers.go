package v1alpha1

import (
	"sync"

	"github.com/digitalocean/godo"
)

// godoClient is the godo client used to validate objects in validating
// webhooks. It's a package global because there's no tidy way to scope it to a
// specific webhook, and it's concurrency-safe anyway.
var (
	godoClient     *godo.Client
	godoClientOnce sync.Once
)

// initGodoClientOnce initializes the package-global godo client. It should be
// called by each webhook setup function.
func initGlobalGodoClient(client *godo.Client) {
	godoClientOnce.Do(func() {
		godoClient = client
	})
}
