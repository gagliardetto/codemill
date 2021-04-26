package untrustedflowsource

import (
	"fmt"

	"github.com/gagliardetto/codemill/x"
)

const (
	Kind x.ModelKind = "UntrustedFlowSource"
)

type Handler struct{}

// TODO:
// There are imaginary groups;
// each group can be made of one or more components;
// each component is sourced from a "method", which is basically the key to a set of selectors.

const (
	MethodSelf = "{source:[](Param|Result|Fields|Type)} <- $source"
)

//
func (han *Handler) ScavengeMethods() []*x.XMethod {
	return []*x.XMethod{
		{
			Name:      MethodSelf,
			Selectors: []*x.XSelector{},
		},
	}
}
func (han *Handler) Validate(mdl *x.XModel) error {
	if len(mdl.Methods) != 1 {
		return fmt.Errorf("wrong number of methods; expected 1, got %v", len(mdl.Methods))
	}
	if mdl.Methods[0].Name != MethodSelf {
		return fmt.Errorf("First method is not called %s", MethodSelf)
	}
	return nil
}
