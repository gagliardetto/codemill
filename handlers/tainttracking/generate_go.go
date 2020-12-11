package tainttracking

import (
	//. "github.com/dave/jennifer/jen"
	"github.com/gagliardetto/codemill/x"
	. "github.com/gagliardetto/utilz"
)

func (han *Handler) GenerateGo(parentDir string, mdl *x.XModel) error {
	Sfln(
		"Generating go code for model %q into %q parentDir",
		mdl.Name,
		parentDir,
	)

	if err := mdl.Validate(); err != nil {
		return err
	}
	if err := han.Validate(mdl); err != nil {
		return err
	}
	return nil
}
