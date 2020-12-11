package tainttracking

import (
	"github.com/gagliardetto/codemill/x"
	. "github.com/gagliardetto/cqlgen/jen"
	. "github.com/gagliardetto/utilz"
)

func (han *Handler) GenerateCodeQL(impAdder x.ImportAdder, mdl *x.XModel, moduleGroup *Group) error {
	Sfln(
		"Generating grouped codeql code for model %q",
		mdl.Name,
	)
	if err := mdl.Validate(); err != nil {
		return err
	}
	if err := han.Validate(mdl); err != nil {
		return err
	}

	// Assuming the validation has already been done:
	self := mdl.Methods[0]

	if len(self.Selectors) == 0 {
		Infof("No selectors found for %q method.", self.Name)
		return nil
	}

	{
		// Add imports:
		impAdder.Import("DataFlow::PathGraph")
	}
	return nil
}
