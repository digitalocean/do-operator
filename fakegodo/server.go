package fakegodo

import (
	"encoding/json"
	"net/http"

	"github.com/digitalocean/do-operator/extgodo"
)

// Handler is an HTTP handler that can handle requests we need to make that are
// not yet supported in godo proper. This is for use with the httptest package
// in unit/integration tests.
type Handler struct{}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/v2/databases":
		if r.Body == nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		var createReq extgodo.DatabaseValidateCreateRequest
		err := json.NewDecoder(r.Body).Decode(&createReq)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		// We should use Databases.Create when we don't want dry run.
		if !createReq.DryRun {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		switch createReq.EngineSlug {
		case "mysql", "pg", "mongodb", "redis":
			rw.WriteHeader(http.StatusOK)
			return
		}

		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	rw.WriteHeader(http.StatusNotFound)
}
