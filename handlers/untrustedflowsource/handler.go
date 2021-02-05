package untrustedflowsource

import (
	"fmt"

	"github.com/gagliardetto/codemill/x"
)

const (
	Kind x.ModelKind = "UntrustedFlowSource"
)

type Handler struct{}

const (
	MethodSelf = "Self"
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
