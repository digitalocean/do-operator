// Package extgodo extends godo to support API functionality not yet included in godo.
package extgodo

import (
	"github.com/digitalocean/godo"
)

// DatabaseValidateCreateRequest is a request to validate a database creation
// request.
// TODO(awg) Get rid of this once validation is supported in godo.
type DatabaseValidateCreateRequest struct {
	godo.DatabaseCreateRequest
	DryRun bool `json:"dry_run"`
}
