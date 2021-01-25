package redirect

import (
	"sort"

	"github.com/gagliardetto/codebox/scanner"
	"github.com/gagliardetto/codemill/x"
	. "github.com/gagliardetto/cqlgen/jen"
	"github.com/gagliardetto/feparser"
	. "github.com/gagliardetto/utilz"
)

func (han *Handler) GenerateCodeQL(impAdder x.ImportAdder, mdl *x.XModel, rootModuleGroup *Group) error {
	if err := mdl.Validate(); err != nil {
		return err
	}
	if err := han.Validate(mdl); err != nil {
		return err
	}

	// Assuming the validation has already been done:
	methodGetURL := mdl.Methods[0]

	if len(methodGetURL.Selectors) == 0 {
		Infof("No selectors found for %q method.", methodGetURL.Name)
		return nil
	}

	{
		// Add imports:
		//impAdder.Import("DataFlow::PathGraph")
	}

	className := mdl.Name
	allPathVersions := func() []string {
		res := make([]string, 0)
		mods := mdl.ListModules()
		for _, mod := range mods {
			res = append(res, mod.PathVersionClean())
		}
		sort.Strings(res)
		return res
	}()

	b2fe, b2tm, b2itm, err := x.GroupFuncSelectors(methodGetURL)
	if err != nil {
		Fatalf("Error while GroupFuncSelectors: %s", err)
	}
	{
		addedCount := 0
		funcModelsClassName := feparser.NewCodeQlName(className)
		tmp := DoGroup(func(tempFuncsModel *Group) {
			tempFuncsModel.Comment("Models HTTP redirects.")
			tempFuncsModel.Private().Class().Id(funcModelsClassName).Extends().List(
				Id("HTTP::Redirect::Range"),
				Id("DataFlow::CallNode"),
			).BlockFunc(
				func(funcModelsClassGroup *Group) {
					funcModelsClassGroup.String().Id("package").Semicolon().Line()
					funcModelsClassGroup.Id("DataFlow::Node").Id("urlNode").Semicolon().Line()

					funcModelsClassGroup.Id(funcModelsClassName).Call().BlockFunc(
						func(funcModelsSelfMethodGroup *Group) {
							{
								funcModelsSelfMethodGroup.DoGroup(
									func(groupCase *Group) {
										for _, pathVersion := range allPathVersions {
											pathCodez := make([]Code, 0)
											// Functions:
											{
												cont, ok := b2fe[pathVersion]
												if ok {
													for _, funcQual := range cont {
														if AllFalse(funcQual.Pos...) {
															continue
														}
														fn := GetFunc(funcQual)
														thing := fn.(*feparser.FEFunc)
														pathCodez = append(pathCodez,
															ParensFunc(
																func(par *Group) {
																	par.Commentf("signature: %s", thing.Signature)
																	par.This().
																		Dot("getTarget").Call().
																		Dot("hasQualifiedName").Call(
																		Id("package"),
																		Lit(thing.Name),
																	)

																	par.And()

																	_, code := GetFuncQualifierCodeElements(funcQual)
																	par.Id("urlNode").Eq().Add(code)
																},
															),
														)
													}

												}
											}
											// Type methods:
											{
												cont, ok := b2tm[pathVersion]
												if ok {
													keys := func(v map[string][]*x.FuncQualifier) []string {
														res := make([]string, 0)
														for key := range v {
															res = append(res, key)
														}
														sort.Strings(res)
														return res
													}(cont)
													for _, receiverTypeID := range keys {
														methodQualifiers := cont[receiverTypeID]
														if len(methodQualifiers) == 0 || !x.HasValidPos(methodQualifiers...) {
															continue
														}
														codez := DoGroup(func(mtdGroup *Group) {
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

															mtdGroup.Commentf("Receiver type: %s", typ.TypeString)

															methodIndex := 0
															mtdGroup.ParensFunc(
																func(parMethods *Group) {
																	for _, methodQual := range methodQualifiers {
																		if AllFalse(methodQual.Pos...) {
																			continue
																		}
																		if methodIndex > 0 {
																			parMethods.Or()
																		}
																		methodIndex++

																		fn := GetFunc(methodQual)
																		thing := fn.(*feparser.FETypeMethod)

																		parMethods.ParensFunc(
																			func(par *Group) {
																				par.Commentf("signature: %s", thing.Func.Signature)

																				par.This().
																					Eq().
																					Any(
																						DoGroup(func(gr *Group) {
																							gr.Id("Method").Id("m")
																						}),
																						DoGroup(func(gr *Group) {
																							gr.Id("m").Dot("hasQualifiedName").Call(
																								Id("package"),
																								Lit(thing.Receiver.TypeName),
																								Lit(thing.Func.Name),
																							)
																						}),
																						nil,
																					).Dot("getACall").Call()

																				par.And()

																				_, code := GetFuncQualifierCodeElements(methodQual)
																				par.Id("urlNode").Eq().Add(code)
																			},
																		)

																	}
																},
															)

														})
														pathCodez = append(pathCodez, codez)
													}
												}
											}
											// Interface methods:
											{
												contb2itm, ok := b2itm[pathVersion]
												if ok {
													keys := func(v map[string][]*x.FuncQualifier) []string {
														res := make([]string, 0)
														for key := range v {
															res = append(res, key)
														}
														sort.Strings(res)
														return res
													}(contb2itm)
													for _, receiverTypeID := range keys {
														methodQualifiers := contb2itm[receiverTypeID]
														if len(methodQualifiers) == 0 || !x.HasValidPos(methodQualifiers...) {
															continue
														}
														codez := DoGroup(func(mtdGroup *Group) {
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
															mtdGroup.Commentf("Receiver interface: %s", typ.TypeString)

															methodIndex := 0
															mtdGroup.ParensFunc(
																func(parMethods *Group) {
																	for _, methodQual := range methodQualifiers {
																		if AllFalse(methodQual.Pos...) {
																			continue
																		}
																		if methodIndex > 0 {
																			parMethods.Or()
																		}
																		methodIndex++

																		fn := GetFunc(methodQual)
																		thing := fn.(*feparser.FEInterfaceMethod)

																		parMethods.ParensFunc(
																			func(par *Group) {
																				par.Commentf("signature: %s", thing.Func.Signature)

																				par.This().
																					Eq().
																					Any(
																						DoGroup(func(gr *Group) {
																							gr.Id("Method").Id("m")
																						}),
																						DoGroup(func(gr *Group) {
																							gr.Id("m").Dot("implements").Call(
																								Id("package"),
																								Lit(thing.Receiver.TypeName),
																								Lit(thing.Func.Name),
																							)
																						}),
																						nil,
																					).Dot("getACall").Call()

																				par.And()

																				_, code := GetFuncQualifierCodeElements(methodQual)
																				par.Id("urlNode").Eq().Add(code)
																			},
																		)

																	}
																},
															)

														})
														pathCodez = append(pathCodez, codez)
													}
												}
											}

											if len(pathCodez) > 0 {
												if addedCount > 0 {
													groupCase.Or()
												}
												path, _ := scanner.SplitPathVersion(pathVersion)
												groupCase.Commentf("HTTP redirect models for package: %s", pathVersion)
												groupCase.Id("package").Eq().Add(x.CqlFormatPackagePath(path)).And()

												groupCase.Parens(
													Join(
														Or(),
														pathCodez...,
													),
												)

												addedCount++
											}
										}
									})
							}
						})

					funcModelsClassGroup.Override().Id("DataFlow::Node").Id("getUrl").Call().BlockFunc(
						func(overrideBlockGroup *Group) {
							overrideBlockGroup.Id("result").Eq().Id("urlNode")
						})

					funcModelsClassGroup.Override().Id("HTTP::ResponseWriter").Id("getResponseWriter").Call().BlockFunc(
						func(overrideBlockGroup *Group) {
							overrideBlockGroup.None()
						})
				})
		})
		if addedCount > 0 {

			rootModuleGroup.Add(tmp)
		}
	}

	return nil
}

func GetFunc(qual *x.FuncQualifier) x.FuncInterface {

	source := x.GetCachedSource(qual.Path, qual.Version)
	if source == nil {
		Fatalf("Source not found: %s@%s", qual.Path, qual.Version)
	}
	// Find the func/type-method/interface-method:
	fn := x.FindFuncByID(source, qual.ID)
	if fn == nil {
		Fatalf("Func not found: %q", qual.ID)
	}

	return fn
}
func posToRelativeParamIndexes(fe x.FuncInterface, positions []bool) []int {
	indexes := make([]int, 0)
	for posIndex, pos := range positions {
		if !pos {
			continue
		}

		elTyp, _, relIndex, err := fe.GetRelativeElement(posIndex)
		if err != nil {
			Fatalf("Error while GetRelativeElement: %s", err)
		}
		if elTyp != feparser.ElementParameter {
			Fatalf("Is not a parameter")
		}

		indexes = append(indexes, relIndex)
	}
	return indexes
}
func GetFuncQualifierCodeElements(qual *x.FuncQualifier) (x.FuncInterface, Code) {

	fn := GetFunc(qual)

	parameterIndexes := posToRelativeParamIndexes(fn, qual.Pos)
	code := x.GenCqlParamQual("this", "getArgument", fn, parameterIndexes)

	return fn, code
}
