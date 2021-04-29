package redirect

import (
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
	allPathVersions := mdl.ListAllPathVersions()

	b2fe, b2tm, b2itm, err := x.GroupFuncSelectors(methodGetURL)
	if err != nil {
		Fatalf("Error while GroupFuncSelectors: %s", err)
	}
	{
		addedCount := 0
		funcModelsClassName := feparser.NewCodeQlName(className)
		tmp := DoGroup(func(tempFuncsModel *Group) {
			tempFuncsModel.Doc("Models HTTP redirects.")
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
														fn := x.GetFuncByQualifier(funcQual)
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
												b2tm.IterValid(pathVersion,
													func(receiverTypeID string, methodQualifiers x.FuncQualifierSlice) {
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

																		fn := x.GetFuncByQualifier(methodQual)
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
													})
											}
											// Interface methods:
											{
												b2itm.IterValid(pathVersion,
													func(receiverTypeID string, methodQualifiers x.FuncQualifierSlice) {
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

																		fn := x.GetFuncByQualifier(methodQual)
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
													})
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
							overrideBlockGroup.Id("result").Dot("getANode").Call().Eq().This().Dot("getReceiver").Call()
						})
				})
		})
		if addedCount > 0 {

			rootModuleGroup.Add(tmp)
		}
	}

	return nil
}

func GetFuncQualifierCodeElements(qual *x.FuncQualifier) (x.FuncInterface, Code) {
	return x.CqlParamQualToCode("this", "getArgument", qual)
}
