package responsebody

import (
	"fmt"

	"github.com/gagliardetto/codemill/x"
)

// NOTES:
// - Assumes all selectors are added because they work in the same context.

const (
	Kind x.ModelKind = "HTTP::ResponseBody"
)

type Handler struct{}

const (
	MethodBodyWithCtFromFuncName = "{ct:Inferred, body:Param} <- $body" // Specify the body parameter; the content-type will be inferred from the function name.

	MethodBodyWithCtIsBody = "{ct:Param, body:Param} <- $body" // Body parameter; content-type will be from another parameter of the same function.
	MethodBodyWithCtIsCt   = "{ct:Param, body:Param} <- $ct"   // Content-type parameter; body will be from another parameter of the same function.

	MethodBody = "{body:Param} <- $body" // Body parameter; the function only allows to specify a body parameter and does not set content-type in any way.
)

//
func (han *Handler) ScavengeMethods() []*x.XMethod {
	return x.ScavengeMethods(
		MethodBodyWithCtFromFuncName, // "Select the body parameter; the content-type will be inferred from the function name.",

		// For funcs that allow to specify two parameters: body and content-type.
		// Each function that you add to MethodBodyWithCtIsBody,
		// you must also add it to MethodBodyWithCtIsCt.
		MethodBodyWithCtIsBody, // "Coupled 1/2: Select body parameter of func; content-type will be selected from another parameter of the same function.",
		MethodBodyWithCtIsCt,   // "Coupled 2/2: Select body parameter of func; content-type will be selected from another parameter of the same function.",

		MethodBody, // "Select body param of any function that allows to set the body but does NOT determine the content-type.",
	)
}
func (han *Handler) Validate(mdl *x.XModel) error {
	defaultMthNum := len(han.ScavengeMethods())
	if len(mdl.Methods) != defaultMthNum {
		return fmt.Errorf("wrong number of methods; expected %v, got %v", defaultMthNum, len(mdl.Methods))
	}
	// TODO: validate
	{
		for i, must := range han.ScavengeMethods() {
			if mdl.Methods[i].Name != must.Name {
				return fmt.Errorf("#%v method is not called %s", i, must.Name)
			}
		}
	}
	return nil
}
