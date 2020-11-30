package untrustedflowsource

import (
	"github.com/gagliardetto/codemill/x"
	. "github.com/gagliardetto/utilz"
)

const (
	Kind x.ModelKind = "UntrustedFlowSource"
)

type Handler struct {
}

//
func (han *Handler) ScavengeMethods() []*x.XMethod {
	return []*x.XMethod{
		{
			Name:      "Self",
			IsSelf:    true,
			Selectors: []*x.XSelector{},
		},
	}
}
func (han *Handler) Validate(mdl *x.XModel) error {
	// TODO
	Sfln(
		"Validating model %q",
		mdl.Name,
	)
	return nil
}

func (han *Handler) GenerateCodeQL(dir string, mdl *x.XModel) error {
	// TODO
	Sfln(
		"Generating codeql code for model %q into %q dir",
		mdl.Name,
		dir,
	)
	return nil
}
func (han *Handler) GenerateGo(dir string, mdl *x.XModel) error {
	// TODO
	Sfln(
		"Generating go for model %q into %q dir",
		mdl.Name,
		dir,
	)
	return nil
}
