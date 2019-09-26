package controller

import (
	"github.com/digitalocean/dodb-operator/pkg/controller/database"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, database.Add)
}
