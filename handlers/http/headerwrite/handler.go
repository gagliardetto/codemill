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
	MethodWriteHeaderKey = "{key:Param, val:Param} <- $key"
	MethodWriteHeaderVal = "{key:Param, val:Param} <- $val"

	MethodCt = "{ct:Param} <- $ct" // Content-type parameter; the function only allows to specify content-type.

	MethodCtFromFuncName = "{ct:Inferred} <- *" // Content-type inferred from the function name.
)

//
func (han *Handler) ScavengeMethods() []*x.XMethod {
	return x.ScavengeMethods(
		MethodWriteHeaderKey,
		MethodWriteHeaderVal,

		MethodCt, // "Select content-type param of any function that allows to set the content-type but does NOT set the body.",

		MethodCtFromFuncName, // "Select any function that sets the content-type independently of params; content-type will be inferred from the func name.",
	)
}
func (han *Handler) Validate(mdl *x.XModel) error {
	defaultMthNum := len(han.ScavengeMethods())
	if len(mdl.Methods) != defaultMthNum {
		return fmt.Errorf("wrong number of methods; expected %v, got %v", defaultMthNum, len(mdl.Methods))
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
