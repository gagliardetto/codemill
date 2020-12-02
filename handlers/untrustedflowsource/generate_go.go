package untrustedflowsource

import (
	"github.com/gagliardetto/codemill/x"
	. "github.com/gagliardetto/utilz"
)

func (han *Handler) GenerateGo(dir string, mdl *x.XModel) error {
	// TODO
	Sfln(
		"Generating go code for model %q into %q dir",
		mdl.Name,
		dir,
	)
	return nil
}
