package headerwrite

import (
	"fmt"

	"github.com/gagliardetto/codemill/x"
)

// NOTE:
// - The method (on type or interface) must write both key and value.
// - Functions without a receiver are not supported.
// - One method per model.

const (
	Kind x.ModelKind = "HTTP::HeaderWrite"
)

type Handler struct{}

const (
	MethodWriteHeaderKey = "WriteKey"
	MethodWriteHeaderVal = "WriteVal"
)

//
func (han *Handler) ScavengeMethods() []*x.XMethod {
	return []*x.XMethod{
		{
			Name:      MethodWriteHeaderKey,
			Selectors: []*x.XSelector{},
		},
		{
			Name:      MethodWriteHeaderVal,
			Selectors: []*x.XSelector{},
		},
	}
}
func (han *Handler) Validate(mdl *x.XModel) error {
	if len(mdl.Methods) != 2 {
		return fmt.Errorf("wrong number of methods; expected 2, got %v", len(mdl.Methods))
	}
	{
		if mdl.Methods[0].Name != MethodWriteHeaderKey {
			return fmt.Errorf("#0 method is not called %s", MethodWriteHeaderKey)
		}
		if mdl.Methods[1].Name != MethodWriteHeaderVal {
			return fmt.Errorf("#1 method is not called %s", MethodWriteHeaderVal)
		}
	}
	// TODO:
	// - Make sure that each selector has a key and a value method.
	return nil
}
