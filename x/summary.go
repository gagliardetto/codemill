package x

import (
	"sort"

	"github.com/gagliardetto/ref"
	. "github.com/gagliardetto/utilz"
)

// CreateSummary creates a summary of the types for each model kind.
func CreateSummary(spec *XSpec) ([]string, error) {
	var summaryLines multiLines
	summaryLines.Append(
		Sf("## CodeQL Module Summary for `%s`:", spec.Name),
		"",
	)

	summaryLines.Append(
		"### Packages:",
		"",
	)
	mods := spec.ListModules()
	for _, qual := range mods {
		summaryLines.Append(Sf(
			" - [%s](https://pkg.go.dev/%s#%s)",
			qual.PathVersionClean(),
			qual.Path,
			qual.Version,
		))
	}
	full := HasMultiversion(mods) || HasMultiplePackages(mods)
	summaryLines.Append("")

	for index, mdl := range spec.Models {
		if index > 0 {
			summaryLines.Append(
				"",
				"---",
				"",
			)
		}
		summaryLines.Append(Sf("### Model kind `%s`:", mdl.Kind))

		funcs := make([]*textWithLink, 0)
		structs := make([]*textWithLink, 0)
		types := make([]*textWithLink, 0)

		for _, method := range mdl.Methods {

			for _, sel := range method.Selectors {
				{
					qual := sel.GetFuncQualifier()
					if qual != nil {
						fn := GetFuncByQualifier(qual)

						tl := &textWithLink{}

						if fn.GetReceiver() == nil {
							tl.link = Sf(
								"https://pkg.go.dev/%s#%s",
								sel.GetBasicQualifier().PathVersionClean(),
								fn.GetFunc().Name,
							)
						} else {
							tl.link = Sf(
								"https://pkg.go.dev/%s#%s.%s",
								sel.GetBasicQualifier().PathVersionClean(),
								fn.GetReceiver().TypeName,
								fn.GetFunc().Name,
							)
						}

						if full {
							tl.text = fn.GetFunc().GetOriginal().Signature
							// TODO: if multiversion, include the package version in the signature.
						} else {
							tl.text = fn.GetFunc().Signature
						}
						funcs = append(funcs, tl)
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
							{
								tl := &textWithLink{}
								tl.text = st.QualifiedName
								tl.link = Sf(
									"https://pkg.go.dev/%s#%s",
									sel.GetBasicQualifier().PathVersionClean(),
									st.TypeName,
								)
								structs = append(structs, tl)
							}
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
							{
								tl := &textWithLink{}
								tl.text = typ.QualifiedName
								tl.link = Sf(
									"https://pkg.go.dev/%s#%s",
									sel.GetBasicQualifier().PathVersionClean(),
									typ.TypeName,
								)
								types = append(types, tl)
							}
						}
						continue
					}
				}
			}
		}

		{
			{
				ref.DeduplicateSlice2(&funcs, func(i int) string {
					return funcs[i].text
				})
				sort.Slice(funcs, func(i, j int) bool {
					return funcs[i].text < funcs[j].text
				})
			}

			{
				ref.DeduplicateSlice2(&structs, func(i int) string {
					return structs[i].text
				})
				sort.Slice(structs, func(i, j int) bool {
					return structs[i].text < structs[j].text
				})
			}

			{
				ref.DeduplicateSlice2(&types, func(i int) string {
					return types[i].text
				})
				sort.Slice(types, func(i, j int) bool {
					return types[i].text < types[j].text
				})
			}
		}
		{
			if len(funcs) > 0 {
				summaryLines.Append("  - `FUNCS`:")
				for _, v := range funcs {
					summaryLines.Append(Sf("    - [%s](%s)", v.text, v.link))
				}
				// summaryLines.Append("")
			}
			if len(structs) > 0 {
				summaryLines.Append("  - `STRUCTS`:")
				for _, v := range structs {
					summaryLines.Append(Sf("    - [%s](%s)", v.text, v.link))
				}
				// summaryLines.Append("")
			}
			if len(types) > 0 {
				summaryLines.Append("  - `TYPES`:")
				for _, v := range types {
					summaryLines.Append(Sf("    - [%s](%s)", v.text, v.link))
				}
				// summaryLines.Append("")
			}
		}
	}

	return summaryLines, nil
}

type textWithLink struct {
	text string
	link string
}
type multiLines []string

//
func (ml *multiLines) Append(lines ...string) {
	*ml = append(*ml, lines...)
}
