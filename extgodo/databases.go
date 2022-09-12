// Package extgodo extends godo to support API functionality not yet included in godo.
package extgodo

import (
	"context"
	"net/http"

	"github.com/digitalocean/godo"
)

// DatabaseValidateCreateRequest is a request to validate a database creation
// request.
// TODO(awg) Get rid of this once validation is supported in godo.
type DatabaseValidateCreateRequest struct {
	godo.DatabaseCreateRequest
	DryRun bool `json:"dry_run"`
}

// DatabaseLayout represents a valid nodes/size combination for a particular
// database engine.
type DatabaseLayout struct {
	NumNodes int      `json:"num_nodes"`
	Sizes    []string `json:"sizes"`
}

// DatabaseEngineOptions represents the valid options available for a particular
// database engine.
type DatabaseEngineOptions struct {
	Regions  []string          `json:"regions"`
	Versions []string          `json:"versions"`
	Layouts  []*DatabaseLayout `json:"layouts"`
}

// DatabaseOptions represents the response to the /v2/databases/options endpoint
// in the DigitalOcean public API.
// TODO(awg) Get rid of this once the options endpoint is supported in godo.
type DatabaseOptions struct {
	OptionsByEngine map[string]DatabaseEngineOptions `json:"options"`
}

// GetDatabaseOptions gets database options.
func GetDatabaseOptions(ctx context.Context, cl *godo.Client) (*DatabaseOptions, error) {
	req, err := cl.NewRequest(ctx, http.MethodGet, "/v2/databases/options", nil)
	if err != nil {
		return nil, err
	}

	var options DatabaseOptions
	_, err = cl.Do(ctx, req, &options)
	if err != nil {
		return nil, err
	}

	return &options, nil
}
