package headerwrite

import (
	"fmt"

	"github.com/gagliardetto/codemill/x"
)

// NOTE:
// - The method (on type or interface) must write both key and value.
// - Functions without a receiver are not supported.

const (
	Kind x.ModelKind = "HttpHeaderWrite"
)

type Handler struct{}

const (
	MethodGetHeaderKey = "GetHeaderKey"
	MethodGetHeaderVal = "GetHeaderVal"
)

//
func (han *Handler) ScavengeMethods() []*x.XMethod {
	return []*x.XMethod{
		{
			Name:      MethodGetHeaderKey,
			Selectors: []*x.XSelector{},
		},
		{
			Name:      MethodGetHeaderVal,
			Selectors: []*x.XSelector{},
		},
	}
}
func (han *Handler) Validate(mdl *x.XModel) error {
	if len(mdl.Methods) != 2 {
		return fmt.Errorf("wrong number of methods; expected 2, got %v", len(mdl.Methods))
	}
	{
		if mdl.Methods[0].Name != MethodGetHeaderKey {
			return fmt.Errorf("#0 method is not called %s", MethodGetHeaderKey)
		}
		if mdl.Methods[1].Name != MethodGetHeaderVal {
			return fmt.Errorf("#1 method is not called %s", MethodGetHeaderVal)
		}
	}
	// TODO:
	// - Make sure that each selector has a key and a value method.
	return nil
}
