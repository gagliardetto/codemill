package redirect

import (
	"errors"
	"fmt"

	"github.com/gagliardetto/codemill/x"
)

const (
	Kind x.ModelKind = "HttpRedirect"
)

type Handler struct{}

const (
	MethodGetURL = "GetUrl"
)

//
func (han *Handler) ScavengeMethods() []*x.XMethod {
	return []*x.XMethod{
		{
			Name:      MethodGetURL,
			IsSelf:    false,
			Selectors: []*x.XSelector{},
		},
	}
}
func (han *Handler) Validate(mdl *x.XModel) error {
	if len(mdl.Methods) != 1 {
		return fmt.Errorf("wrong number of methods; expected 1, got %v", len(mdl.Methods))
	}
	{
		if mdl.Methods[0].IsSelf {
			return errors.New("#0 method must not be self")
		}
		if mdl.Methods[0].Name != MethodGetURL {
			return fmt.Errorf("#0 method is not called %s", MethodGetURL)
		}
	}
	return nil
}
