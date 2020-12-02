package untrustedflowsource

import (
	"errors"
	"fmt"

	"github.com/gagliardetto/codemill/x"
	. "github.com/gagliardetto/utilz"
)

const (
	Kind x.ModelKind = "UntrustedFlowSource"
)

type Handler struct{}

const (
	MethodSelf = "Self"
)

//
func (han *Handler) ScavengeMethods() []*x.XMethod {
	return []*x.XMethod{
		{
			Name:      MethodSelf,
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
	if len(mdl.Methods) != 1 {
		return fmt.Errorf("wrong number of methods; expected 1, got %v", len(mdl.Methods))
	}
	if !mdl.Methods[0].IsSelf {
		return errors.New("First method is not self")
	}
	if mdl.Methods[0].Name != MethodSelf {
		return fmt.Errorf("First method is not called %s", MethodSelf)
	}
	return nil
}

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
func (han *Handler) GenerateGo(dir string, mdl *x.XModel) error {
	// TODO
	Sfln(
		"Generating go for model %q into %q dir",
		mdl.Name,
		dir,
	)
	return nil
}
