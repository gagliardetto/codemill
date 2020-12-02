package untrustedflowsource

import (
	"github.com/gagliardetto/codemill/x"
	. "github.com/gagliardetto/utilz"
)

func (han *Handler) GenerateCodeQL(dir string, mdl *x.XModel) error {
	// TODO
	Sfln(
		"Generating codeql code for model %q into %q dir",
		mdl.Name,
		dir,
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

	for _, selector := range self.Selectors {
		// TODO: do validation here?
		if err := selector.Validate(); err != nil {
			return err
		}
		rawQual := selector.Qualifier

		switch qual := rawQual.(type) {
		case *x.FuncQualifier:
			if err := qual.Validate(); err != nil {
				return err
			}
		case *x.StructQualifier:
			if err := qual.Validate(); err != nil {
				return err
			}
		case *x.TypeQualifier:
			if err := qual.Validate(); err != nil {
				return err
			}
		default:
			panic(Sf("Unknown type: %T", rawQual))
		}

	}

	return nil
}
