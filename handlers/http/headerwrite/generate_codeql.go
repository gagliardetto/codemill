package headerwrite

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
	MethodWriteHeaderKey := mdl.Methods.ByName(MethodWriteHeaderKey)
	if len(MethodWriteHeaderKey.Selectors) == 0 {
		Infof("No selectors found for %q method.", MethodWriteHeaderKey.Name)
		return nil
	}

	MethodWriteHeaderVal := mdl.Methods.ByName(MethodWriteHeaderVal)
	if len(MethodWriteHeaderVal.Selectors) == 0 {
		Infof("No selectors found for %q method.", MethodWriteHeaderVal.Name)
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

	_, b2tmKey, b2itmKey, err := x.GroupFuncSelectors(MethodWriteHeaderKey)
	if err != nil {
		Fatalf("Error while GroupFuncSelectors: %s", err)
	}
	_, b2tmVal, b2itmVal, err := x.GroupFuncSelectors(MethodWriteHeaderVal)
	if err != nil {
		Fatalf("Error while GroupFuncSelectors: %s", err)
	}

	{
		addedCount := 0
		funcModelsClassName := feparser.NewCodeQlName(className)
		tmp := DoGroup(func(tempFuncsModel *Group) {
			tempFuncsModel.Doc("Models HTTP header writes.")
			tempFuncsModel.Private().Class().Id(funcModelsClassName).Extends().List(
				Id("HTTP::HeaderWrite::Range"),
				Id("DataFlow::CallNode"),
			).BlockFunc(
				func(funcModelsClassGroup *Group) {
					funcModelsClassGroup.Id("DataFlow::Node").Id("nameNode").Semicolon().Line()
					funcModelsClassGroup.Id("DataFlow::Node").Id("valueNode").Semicolon().Line()

					funcModelsClassGroup.Id(funcModelsClassName).Call().BlockFunc(
						func(funcModelsSelfMethodGroup *Group) {
							{
								funcModelsSelfMethodGroup.DoGroup(
									func(groupCase *Group) {
										for _, pathVersion := range allPathVersions {
											pathCodez := make([]Code, 0)

											// Type methods:
											{
												cont, ok := b2tmKey[pathVersion]
												if ok {
													keys := func(v map[string]x.FuncQualifierSlice) []string {
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
																	for _, keyMethodQual := range methodQualifiers {
																		if AllFalse(keyMethodQual.Pos...) {
																			continue
																		}
																		if methodIndex > 0 {
																			parMethods.Or()
																		}
																		methodIndex++

																		fn := GetFunc(keyMethodQual)
																		thing := fn.(*feparser.FETypeMethod)

																		// TODO:
																		// - Check if found.
																		valMethodQual := b2tmVal[pathVersion][receiverTypeID].ByBasicQualifier(keyMethodQual.BasicQualifier)

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
																							path, _ := scanner.SplitPathVersion(pathVersion)

																							gr.Id("m").Dot("hasQualifiedName").Call(
																								x.CqlFormatPackagePath(path),
																								Lit(thing.Receiver.TypeName),
																								Lit(thing.Func.Name),
																							)
																						}),
																						nil,
																					).Dot("getACall").Call()

																				par.And()

																				{
																					_, code := GetFuncQualifierCodeElements(keyMethodQual)
																					par.Id("nameNode").Eq().Add(code).And()
																				}

																				{
																					_, code := GetFuncQualifierCodeElements(valMethodQual)
																					par.Id("valNode").Eq().Add(code)
																				}
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
												contb2itm, ok := b2itmKey[pathVersion]
												if ok {
													keys := func(v map[string]x.FuncQualifierSlice) []string {
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
																	for _, keyMethodQual := range methodQualifiers {
																		if AllFalse(keyMethodQual.Pos...) {
																			continue
																		}
																		if methodIndex > 0 {
																			parMethods.Or()
																		}
																		methodIndex++

																		fn := GetFunc(keyMethodQual)
																		thing := fn.(*feparser.FEInterfaceMethod)

																		// TODO:
																		// - Check if found.
																		valMethodQual := b2itmVal[pathVersion][receiverTypeID].ByBasicQualifier(keyMethodQual.BasicQualifier)

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

																				{
																					_, code := GetFuncQualifierCodeElements(keyMethodQual)
																					par.Id("urlNode").Eq().Add(code).And()
																				}

																				{
																					_, code := GetFuncQualifierCodeElements(valMethodQual)
																					par.Id("valNode").Eq().Add(code)
																				}
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
												groupCase.Commentf("HTTP header write model for package: %s", pathVersion)

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

					funcModelsClassGroup.Override().Id("DataFlow::Node").Id("getName").Call().BlockFunc(
						func(overrideBlockGroup *Group) {
							overrideBlockGroup.Id("result").Eq().Id("nameNode")
						})
					funcModelsClassGroup.Override().Id("DataFlow::Node").Id("getValue").Call().BlockFunc(
						func(overrideBlockGroup *Group) {
							overrideBlockGroup.Id("result").Eq().Id("valueNode")
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

func GetFuncQualifierCodeElements(qual *x.FuncQualifier) (x.FuncInterface, Code) {

	fn := GetFunc(qual)

	parameterIndexes := x.MustPosToRelativeParamIndexes(fn, qual.Pos)
	code := x.GenCqlParamQual("this", "getArgument", fn, parameterIndexes)

	return fn, code
}
