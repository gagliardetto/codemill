package untrustedflowsource

import (
	"sort"

	"github.com/gagliardetto/codebox/scanner"
	"github.com/gagliardetto/codemill/x"
	. "github.com/gagliardetto/cqlgen/jen"
	"github.com/gagliardetto/feparser"
	. "github.com/gagliardetto/utilz"
)

func (han *Handler) GenerateCodeQL(impAdder x.ImportAdder, mdl *x.XModel, moduleGroup *Group) error {
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
		//impAdder.Import("DataFlow::PathGraph")
	}

	className := mdl.Name

	moduleGroup.Doc("Provides models of untrusted flow sources.")
	moduleGroup.Private().Class().Id(className).Extends().List(Qual("UntrustedFlowSource", "Range")).
		BlockFunc(func(classGr *Group) {
			classGr.Id(className).Call().BlockFunc(func(metGr *Group) {
				b2fe, b2tm, b2itm, err := x.GroupFuncSelectors(self)
				if err != nil {
					Fatalf("Error while GroupFuncSelectors: %s", err)
				}

				{
					index := 0
					keys := func(v x.BasicToFEFuncs) []string {
						res := make([]string, 0)
						for key := range v {
							res = append(res, key)
						}
						sort.Strings(res)
						return res
					}(b2fe)
					for _, pathVersion := range keys {
						cont, ok := b2fe[pathVersion]
						if !ok {
							continue
						}
						if index > 0 {
							metGr.Or()
						}
						index++

						metGr.Comment("Functions of package: " + pathVersion)
						metGr.Exists(
							List(
								Id("Function").Id("fn"),
								Id("FunctionOutput").Id("outp"),
							),
							DoGroup(func(st *Group) {
								st.This().Eq().Id("outp").Dot("getExitNode").Call(Id("fn").Dot("getACall").Call())
							}),
							DoGroup(func(st *Group) {
								for i, funcQual := range cont {
									if i > 0 {
										st.Or()
									}

									fn, codeElements := GetFuncQualifierCodeElements(funcQual)
									thing := fn.(*feparser.FEFunc)
									st.Comment("signature: " + thing.Signature)
									st.Id("fn").Dot("hasQualifiedName").Call(x.CqlFormatPackagePath(funcQual.Path), Lit(thing.Name)).
										And().
										Parens(
											Join(
												Or(),
												codeElements...,
											),
										)
								}
							}),
						)
					}
				}

				if len(b2fe) > 0 && len(b2tm) > 0 {
					metGr.Or()
				}

				{
					index := 0
					keys := func(v x.BasicToTypeIDToMethods) []string {
						res := make([]string, 0)
						for key := range v {
							res = append(res, key)
						}
						sort.Strings(res)
						return res
					}(b2tm)
					for _, pathVersion := range keys {
						cont, ok := b2tm[pathVersion]
						if !ok {
							continue
						}
						if index > 0 {
							metGr.Or()
						}
						index++

						path, _ := scanner.SplitPathVersion(pathVersion)

						metGr.Comment("Methods on types of package: " + pathVersion)
						metGr.Exists(
							List(
								String().Id("receiverName"),
								String().Id("methodName"),
								Id("Method").Id("mtd"),
								Id("FunctionOutput").Id("outp"),
							),
							DoGroup(func(st *Group) {
								st.This().Eq().Id("outp").Dot("getExitNode").Call(Id("mtd").Dot("getACall").Call())

								st.And()

								st.Id("mtd").Dot("hasQualifiedName").Call(
									x.CqlFormatPackagePath(path),
									Id("receiverName"),
									Id("methodName"),
								)
							}),
							DoGroup(func(st *Group) {
								typeIndex := 0
								keys := func(v map[string][]*x.FuncQualifier) []string {
									res := make([]string, 0)
									for key := range v {
										res = append(res, key)
									}
									sort.Strings(res)
									return res
								}(cont)
								for _, receiverTypeID := range keys {
									methodQualifiers, ok := cont[receiverTypeID]
									if !ok {
										continue
									}
									if typeIndex > 0 {
										st.Or()
									}
									typeIndex++

									qual := methodQualifiers[0]
									source := x.GetCachedSource(qual.Path, qual.Version)
									if source == nil {
										Fatalf("Source not found: %s@%s", qual.Path, qual.Version)
									}
									// Find receiver type:
									typ := x.FindTypeByID(source, receiverTypeID)
									if typ == nil {
										Fatalf("Type not found: %q", receiverTypeID)
									}

									st.Id("receiverName").Eq().Lit(typ.TypeString)
									st.And()

									st.ParensFunc(
										func(parMethods *Group) {
											for i, methodQual := range methodQualifiers {
												if i > 0 {
													parMethods.Or()
												}

												fn, codeElements := GetFuncQualifierCodeElements(methodQual)
												thing := fn.(*feparser.FETypeMethod)

												parMethods.ParensFunc(
													func(par *Group) {
														par.Commentf("signature: %s", thing.Func.Signature)
														par.Id("methodName").Eq().Lit(thing.Func.Name)
														par.And()
														par.Parens(
															Join(
																Or(),
																codeElements...,
															),
														)
													},
												)

											}
										},
									)
								}
							}),
						)
					}
				}

				if (len(b2fe) > 0 || len(b2tm) > 0) && len(b2itm) > 0 {
					metGr.Or()
				}

				{
					index := 0
					keys := func(v x.BasicToInterfaceIDToMethods) []string {
						res := make([]string, 0)
						for key := range v {
							res = append(res, key)
						}
						sort.Strings(res)
						return res
					}(b2itm)
					for _, pathVersion := range keys {
						cont, ok := b2itm[pathVersion]
						if !ok {
							continue
						}
						if index > 0 {
							metGr.Or()
						}
						index++

						path, _ := scanner.SplitPathVersion(pathVersion)

						metGr.Comment("Interfaces of package: " + pathVersion)
						metGr.Exists(
							List(
								String().Id("interfaceName"),
								String().Id("methodName"),
								Id("Method").Id("mtd"),
								Id("FunctionOutput").Id("outp"),
							),
							DoGroup(func(st *Group) {
								st.This().Eq().Id("outp").Dot("getExitNode").Call(Id("mtd").Dot("getACall").Call())

								st.And()

								st.Id("mtd").Dot("implements").Call(
									x.CqlFormatPackagePath(path),
									Id("interfaceName"),
									Id("methodName"),
								)
							}),
							DoGroup(func(st *Group) {
								typeIndex := 0
								keys := func(v map[string][]*x.FuncQualifier) []string {
									res := make([]string, 0)
									for key := range v {
										res = append(res, key)
									}
									sort.Strings(res)
									return res
								}(cont)
								for _, receiverTypeID := range keys {
									methodQualifiers, ok := cont[receiverTypeID]
									if !ok {
										continue
									}
									if typeIndex > 0 {
										st.Or()
									}
									typeIndex++

									qual := methodQualifiers[0]
									source := x.GetCachedSource(qual.Path, qual.Version)
									if source == nil {
										Fatalf("Source not found: %s@%s", qual.Path, qual.Version)
									}

									// Find interface type:
									typ := x.FindTypeByID(source, receiverTypeID)
									if typ == nil {
										Fatalf("Type not found: %q", receiverTypeID)
									}

									st.Id("interfaceName").Eq().Lit(typ.TypeString)

									st.And()

									st.ParensFunc(
										func(parMethods *Group) {
											for i, methodQual := range methodQualifiers {
												if i > 0 {
													parMethods.Or()
												}

												fn, codeElements := GetFuncQualifierCodeElements(methodQual)
												thing := fn.(*feparser.FEInterfaceMethod)

												parMethods.ParensFunc(
													func(par *Group) {
														par.Commentf("signature: %s", thing.Func.Signature)
														par.Id("methodName").Eq().Lit(thing.Func.Name)
														par.And()
														par.Parens(
															Join(
																Or(),
																codeElements...,
															),
														)
													},
												)

											}
										},
									)
								}
							}),
						)
					}
				}

				b2st, err := x.GroupStructSelectors(self)
				if err != nil {
					Fatalf("Error while GroupStructSelectors: %s", err)
				}
				if (len(b2fe) > 0 || len(b2tm) > 0 || len(b2itm) > 0) && len(b2st) > 0 {
					metGr.Or()
				}
				{
					index := 0
					keys := func(v x.BasicToStructIDToFields) []string {
						res := make([]string, 0)
						for key := range v {
							res = append(res, key)
						}
						sort.Strings(res)
						return res
					}(b2st)
					for _, pathVersion := range keys {
						structQualifiers, ok := b2st[pathVersion]
						if !ok {
							continue
						}
						if index > 0 {
							metGr.Or()
						}
						index++
						path, _ := scanner.SplitPathVersion(pathVersion)

						metGr.Comment("Structs of package: " + pathVersion)
						metGr.Exists(
							List(
								String().Id("structName"),
								String().Id("fields"),
								Qual("DataFlow", "Field").Id("fld"),
							),
							DoGroup(func(st *Group) {
								st.This().Eq().Id("fld").Dot("getARead").Call()

								st.And()

								st.Id("fld").Dot("hasQualifiedName").Call(
									x.CqlFormatPackagePath(path),
									Id("structName"),
									Id("fields"),
								)
							}),
							DoGroup(func(st *Group) {
								for qualIndex, qual := range structQualifiers {
									if qualIndex > 0 {
										st.Or()
									}
									source := x.GetCachedSource(qual.Path, qual.Version)
									if source == nil {
										Fatalf("Source not found: %s@%s", qual.Path, qual.Version)
									}
									// Make sure that the struct exist:
									str := x.FindStructByID(source, qual.ID)
									if str == nil {
										Fatalf("Struct not found: %q", qual.ID)
									}

									fieldNames := make([]string, 0)
									for fieldName := range qual.Fields {
										//fld := x.FindFieldByName(str, fieldName)
										//if fld == nil {
										//	Fatalf("Field not found: %q", fieldName)
										//}
										// TODO: add a comment on the type for each field?
										fieldNames = append(fieldNames, fieldName)
									}
									st.Id("structName").Eq().Lit(str.TypeName)
									st.And()
									st.Id("fields").Eq().Add(StringsToSetOrLit(fieldNames...))
								}
							}),
						)
					}
				}

				b2typ, err := x.GroupTypeSelectors(self)
				if err != nil {
					Fatalf("Error while GroupTypeSelectors: %s", err)
				}
				if (len(b2fe) > 0 || len(b2tm) > 0 || len(b2itm) > 0 || len(b2st) > 0) && len(b2typ) > 0 {
					metGr.Or()
				}
				{
					index := 0
					keys := func(v x.BasicToTypes) []string {
						res := make([]string, 0)
						for key := range v {
							res = append(res, key)
						}
						sort.Strings(res)
						return res
					}(b2typ)
					for _, pathVersion := range keys {
						typeQualifiers, ok := b2typ[pathVersion]
						if !ok {
							continue
						}
						if index > 0 {
							metGr.Or()
						}
						index++
						path, _ := scanner.SplitPathVersion(pathVersion)

						metGr.Comment("Types of package: " + pathVersion)
						metGr.Exists(
							List(
								Id("ValueEntity").Id("v"),
							),
							DoGroup(func(st *Group) {
								var typeNames []string
								for _, qual := range typeQualifiers {
									source := x.GetCachedSource(qual.Path, qual.Version)
									if source == nil {
										Fatalf("Source not found: %s@%s", qual.Path, qual.Version)
									}
									// Find the type:
									typ := x.FindTypeByID(source, qual.ID)
									if typ == nil {
										Fatalf("Type not found: %q", qual.ID)
									}
									typeNames = append(typeNames, typ.TypeName)
								}

								sort.Strings(typeNames)

								st.Id("v").Dot("getType").Call().Dot("hasQualifiedName").Call(
									x.CqlFormatPackagePath(path),
									StringsToSetOrLit(typeNames...),
								)
							}),
							DoGroup(func(st *Group) {
								st.This().Eq().Id("v").Dot("getARead").Call()
							}),
						)
					}
				}

			})
		})

	return nil
}
func GetFuncQualifierCodeElements(qual *x.FuncQualifier) (x.FuncInterface, []Code) {

	source := x.GetCachedSource(qual.Path, qual.Version)
	if source == nil {
		Fatalf("Source not found: %s@%s", qual.Path, qual.Version)
	}
	// Find the func/type-method/interface-method:
	fn := x.FindFuncByID(source, qual.ID)
	if fn == nil {
		Fatalf("Func not found: %q", qual.ID)
	}

	codeElements := make([]Code, 0)
	parameterIndexes := make([]int, 0)
	resultIndexes := make([]int, 0)
PosLoop:
	for pos, ok := range qual.Pos {
		if !ok {
			continue PosLoop
		}

		elTyp, _, relIndex, err := fn.GetRelativeElement(pos)
		if err != nil {
			Fatalf("Error while GetRelativeElement: %s", err)
		}

		switch elTyp {
		case feparser.ElementReceiver:
			{
				codeElements = append(codeElements,
					Id("outp").Dot("isReceiver").Call(),
				)
			}
		case feparser.ElementParameter:
			{
				parameterIndexes = append(parameterIndexes,
					relIndex,
				)
			}
		case feparser.ElementResult:
			{
				resultIndexes = append(resultIndexes,
					relIndex,
				)
			}
		default:
			panic(Sf("Unknown type: %q", elTyp))
		}
	}

	_, lenParams, lenResults := fn.Lengths()

	if len(parameterIndexes) > 0 {
		// If all parameters are selected,
		// and there is more than one possible parameters,
		// then use a `_`:
		if lenParams == len(parameterIndexes) && lenParams > 1 {
			codeElements = append(codeElements,
				Id("outp").Dot("isParameter").Call(DontCare()),
			)

		} else {
			// If multiple parameters are selected (but not all)
			// then use a set, or just the index.
			// If there is only one possible parameter and it is selected,
			// then `isParameter(0)` is used.
			codeElements = append(codeElements,
				Id("outp").Dot("isParameter").Call(
					DoGroup(func(callGroup *Group) {
						if fn.GetFunc().GetOriginal().Variadic {

							lits := make([]Code, 0)
							if len(parameterIndexes) == 1 && parameterIndexes[0] == 0 {
								lits = append(lits, DontCare())
							} else {
								for _, index := range parameterIndexes {
									isLast := index == lenParams-1
									if isLast {
										lits = append(lits, Any(
											Add(Int(), Id("i")),
											Add(Id("i").Gte().Lit(index)),
											nil,
										))
									} else {
										lits = append(lits, Lit(index))
									}
								}
							}

							if len(parameterIndexes) == 1 {
								callGroup.Add(lits...)
							} else {
								callGroup.Add(Set(lits...))
							}

						} else {
							callGroup.Add(IntsToSetOrLit(parameterIndexes...))
						}
					}),
				),
			)
		}
	}
	if len(resultIndexes) > 0 {
		if lenResults == len(resultIndexes) {
			if lenResults == 1 {
				// If there is only one result possible, then use a `isResult()`:
				codeElements = append(codeElements,
					Id("outp").Dot("isResult").Call(),
				)
			} else {
				// If there are more than one results,
				// and all results are selected, then use a `_`:
				codeElements = append(codeElements,
					Id("outp").Dot("isResult").Call(DontCare()),
				)
			}
		} else {
			codeElements = append(codeElements,
				Id("outp").Dot("isResult").Call(IntsToSetOrLit(resultIndexes...)),
			)
		}
	}

	return fn, codeElements
}
