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
func (han *Handler) GenerateCodeQL(dir string, modelSpec *x.XModel) error {
	// TODO
	Sfln(
		"Generating codeql code for model %q into %q dir",
		modelSpec.Name,
		dir,
	)
	return nil
}
func (han *Handler) GenerateGo(dir string, modelSpec *x.XModel) error {
	// TODO
	Sfln(
		"Generating go for model %q into %q dir",
		modelSpec.Name,
		dir,
	)
	return nil
}
