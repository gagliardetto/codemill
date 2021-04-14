package x

import (
	"sort"

	. "github.com/gagliardetto/utilz"
)

// CreateSummary creates a summary of the types for each model kind.
func CreateSummary(spec *XSpec) ([]string, error) {
	var summaryLines multiLines
	summaryLines.Append(
		"Module summary:",
		"",
	)

	summaryLines.Append(
		"Packages:",
		"",
	)
	mods := spec.ListModules()
	for _, qual := range mods {
		summaryLines.Append(" - " + qual.PathVersionClean())
	}
	full := HasMultiversion(mods) || HasMultiplePackages(mods)
	summaryLines.Append("")

	for index, mdl := range spec.Models {
		if index > 0 {
			summaryLines.Append(
				"",
				"---",
			)
		}
		summaryLines.Append(string(mdl.Kind) + ":")

		funcs := make([]string, 0)
		structs := make([]string, 0)
		types := make([]string, 0)

		for _, method := range mdl.Methods {

			for _, sel := range method.Selectors {
				{
					qual := sel.GetFuncQualifier()
					if qual != nil {
						fn := GetFuncByQualifier(qual)
						if full {
							// TODO: if multiversion, include the package version in the signature.
							funcs = append(funcs, fn.GetFunc().GetOriginal().Signature)
						} else {
							funcs = append(funcs, fn.GetFunc().Signature)
						}
						continue
					}
				}
				{
					qual := sel.GetStructQualifier()
					if qual != nil {
						{
							source := GetCachedSource(qual.Path, qual.Version)
							if source == nil {
								// TODO
								continue
							}
							// Make sure that the struct exist:
							st := FindStructByID(source, qual.ID)
							if st == nil {
								// TODO
								continue
							}
							structs = append(structs, st.QualifiedName)
						}
						continue
					}
				}
				{
					qual := sel.GetTypeQualifier()
					if qual != nil {
						{
							source := GetCachedSource(qual.Path, qual.Version)
							if source == nil {
								// TODO
								continue
							}
							typ := FindTypeByID(source, qual.ID)
							if typ == nil {
								// TODO
								continue
							}
							types = append(types, typ.QualifiedName)
						}
						continue
					}
				}
			}
		}

		{
			funcs = Deduplicate(funcs)
			sort.Strings(funcs)

			structs = Deduplicate(structs)
			sort.Strings(structs)

			types = Deduplicate(types)
			sort.Strings(types)
		}
		{
			if len(funcs) > 0 {
				summaryLines.Append("  FUNCS:")
				for _, v := range funcs {
					summaryLines.Append(Sf("    %s", v))
				}
				// summaryLines.Append("")
			}
			if len(structs) > 0 {
				summaryLines.Append("  STRUCTS:")
				for _, v := range structs {
					summaryLines.Append(Sf("    %s", v))
				}
				// summaryLines.Append("")
			}
			if len(types) > 0 {
				summaryLines.Append("  TYPES:")
				for _, v := range types {
					summaryLines.Append(Sf("    %s", v))
				}
				// summaryLines.Append("")
			}
		}
	}

	return summaryLines, nil
}

type multiLines []string

//
func (ml *multiLines) Append(lines ...string) {
	*ml = append(*ml, lines...)
}
