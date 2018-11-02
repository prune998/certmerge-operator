package controller

import (
	"github.com/prune998/certmerge-operator/pkg/controller/certmerge"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, certmerge.Add)
}
