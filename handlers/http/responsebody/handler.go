package responsebody

import (
	"errors"
	"fmt"

	"github.com/gagliardetto/codemill/x"
)

const (
	Kind x.ModelKind = "HttpResponseBody"
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
			IsSelf:    true,
			Selectors: []*x.XSelector{},
		},
	}
}
func (han *Handler) Validate(mdl *x.XModel) error {
	if len(mdl.Methods) != 1 {
		return fmt.Errorf("wrong number of methods; expected 1, got %v", len(mdl.Methods))
	}
	{
		if !mdl.Methods[0].IsSelf {
			return errors.New("#0 method must be self")
		}
		if mdl.Methods[0].Name != MethodSelf {
			return fmt.Errorf("#0 method is not called %s", MethodSelf)
		}
	}
	return nil
}
