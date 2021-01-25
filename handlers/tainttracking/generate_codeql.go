package tainttracking

import (
	"sort"

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
	allPathVersions := func() []string {
		res := make([]string, 0)
		mods := mdl.ListModules()
		for _, mod := range mods {
			res = append(res, mod.PathVersionClean())
		}
		sort.Strings(res)
		return res
	}()

	b2fe, b2tm, b2itm, err := x.GroupFuncSelectors(self)
	if err != nil {
		Fatalf("Error while GroupFuncSelectors: %s", err)
	}
	{
		addedCount := 0
		funcModelsClassName := feparser.NewCodeQlName(className, "FunctionModels")
		tmp := DoGroup(func(tempFuncsModel *Group) {
			tempFuncsModel.Comment("Models taint-tracking through functions.")
			tempFuncsModel.Private().Class().Id(funcModelsClassName).Extends().Qual("TaintTracking", "FunctionModel").BlockFunc(
				func(funcModelsClassGroup *Group) {
					funcModelsClassGroup.Id("FunctionInput").Id("inp").Semicolon().Line()
					funcModelsClassGroup.Id("FunctionOutput").Id("out").Semicolon().Line()

					funcModelsClassGroup.Id(funcModelsClassName).Call().BlockFunc(
						func(funcModelsSelfMethodGroup *Group) {
							{
								funcModelsSelfMethodGroup.DoGroup(
									func(groupCase *Group) {
										for _, pathVersion := range allPathVersions {
											cont, ok := b2fe[pathVersion]
											if ok {
												pathCodez := make([]Code, 0)
												for _, funcQual := range cont {
													if !x.HasValidEnabledFlow(funcQual) {
														continue
													}

													fn, codeElements := GetFuncQualifierCodeElements(funcQual)
													thing := fn.(*feparser.FEFunc)
													pathCodez = append(pathCodez,
														ParensFunc(
															func(par *Group) {
																par.Commentf("signature: %s", thing.Signature)
																par.This().Dot("hasQualifiedName").Call(x.CqlFormatPackagePath(funcQual.Path), Lit(thing.Name))
																par.And()

																joined := Join(
																	Or(),
																	codeElements...,
																)
																if len(codeElements) > 1 {
																	par.Parens(
																		joined,
																	)
																} else {
																	par.Add(joined)
																}
															},
														),
													)
												}

												if len(pathCodez) > 0 {
													if addedCount > 0 {
														groupCase.Or()
													}
													groupCase.Commentf("Taint-tracking models for package: %s", pathVersion).Parens(
														Join(
															Or(),
															pathCodez...,
														),
													)
													addedCount++
												}
											}
										}
									})
							}
						})

					funcModelsClassGroup.Override().Predicate().Id("hasTaintFlow").Call(Id("FunctionInput").Id("input"), Id("FunctionOutput").Id("output")).BlockFunc(
						func(overrideBlockGroup *Group) {
							overrideBlockGroup.Id("input").Eq().Id("inp").And().Id("output").Eq().Id("out")
						})
				})
		})
		if addedCount > 0 {
			rootModuleGroup.Add(tmp)
		}
	}

	{
		addedCount := 0
		methodModelsClassName := feparser.NewCodeQlName(className, "MethodModels")
		tmp := DoGroup(func(tempMethodsModel *Group) {
			tempMethodsModel.Comment("Models taint-tracking through method calls.")
			tempMethodsModel.Private().Class().Id(methodModelsClassName).Extends().List(Qual("TaintTracking", "FunctionModel"), Id("Method")).BlockFunc(
				func(methodModelsClassGroup *Group) {
					methodModelsClassGroup.Id("FunctionInput").Id("inp").Semicolon().Line()
					methodModelsClassGroup.Id("FunctionOutput").Id("out").Semicolon().Line()

					methodModelsClassGroup.Id(methodModelsClassName).Call().BlockFunc(
						func(methodModelsSelfMethodGroup *Group) {
							{
								methodModelsSelfMethodGroup.DoGroup(
									func(groupCase *Group) {
										for _, pathVersion := range allPathVersions {
											pathCodez := make([]Code, 0)
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
														if len(methodQualifiers) == 0 || !x.HasValidEnabledFlow(methodQualifiers...) {
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
																		if !methodQual.Flows.Enabled || x.AllBlocksEmpty(methodQual.Flows.Blocks...) {
																			continue
																		}
																		if methodIndex > 0 {
																			parMethods.Or()
																		}
																		methodIndex++

																		fn, codeElements := GetFuncQualifierCodeElements(methodQual)
																		thing := fn.(*feparser.FETypeMethod)

																		parMethods.ParensFunc(
																			func(par *Group) {
																				par.Commentf("signature: %s", thing.Func.Signature)
																				par.This().Dot("hasQualifiedName").Call(x.CqlFormatPackagePath(methodQual.Path), Lit(thing.Receiver.TypeName), Lit(thing.Func.Name))
																				par.And()

																				joined := Join(
																					Or(),
																					codeElements...,
																				)
																				if len(codeElements) > 1 {
																					par.Parens(
																						joined,
																					)
																				} else {
																					par.Add(joined)
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
													if len(methodQualifiers) == 0 || !x.HasValidEnabledFlow(methodQualifiers...) {
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
																	if !methodQual.Flows.Enabled || x.AllBlocksEmpty(methodQual.Flows.Blocks...) {
																		continue
																	}
																	if methodIndex > 0 {
																		parMethods.Or()
																	}
																	methodIndex++

																	fn, codeElements := GetFuncQualifierCodeElements(methodQual)
																	thing := fn.(*feparser.FEInterfaceMethod)

																	parMethods.ParensFunc(
																		func(par *Group) {
																			par.Commentf("signature: %s", thing.Func.Signature)
																			par.This().Dot("implements").Call(x.CqlFormatPackagePath(methodQual.Path), Lit(thing.Receiver.TypeName), Lit(thing.Func.Name))
																			par.And()

																			joined := Join(
																				Or(),
																				codeElements...,
																			)
																			if len(codeElements) > 1 {
																				par.Parens(
																					joined,
																				)
																			} else {
																				par.Add(joined)
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

											if len(pathCodez) > 0 {
												if addedCount > 0 {
													groupCase.Or()
												}
												groupCase.Commentf("Taint-tracking models for package: %s", pathVersion).Parens(
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

					methodModelsClassGroup.Override().Predicate().Id("hasTaintFlow").Call(Id("FunctionInput").Id("input"), Id("FunctionOutput").Id("output")).BlockFunc(
						func(overrideBlockGroup *Group) {
							overrideBlockGroup.Id("input").Eq().Id("inp").And().Id("output").Eq().Id("out")
						})
				})
		})
		if addedCount > 0 {
			rootModuleGroup.Add(tmp)
		}
	}

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

	for _, block := range qual.Flows.Blocks {
		inpCodeElements := make([]Code, 0)
		{
			receiver, parameterIndexes, resultIndexes := x.PosToRelativeIndexes(fn, block.Inp)
			inpCodeElements = x.GenFunctionInputOutput("inp", fn, receiver, parameterIndexes, resultIndexes)
		}

		outCodeElements := make([]Code, 0)
		{
			receiver, parameterIndexes, resultIndexes := x.PosToRelativeIndexes(fn, block.Out)
			outCodeElements = x.GenFunctionInputOutput("out", fn, receiver, parameterIndexes, resultIndexes)
		}

		codeElements = append(codeElements,
			Parens(
				Join(
					Or(),
					inpCodeElements...,
				),
			).
				And().
				Parens(
					Join(
						Or(),
						outCodeElements...,
					),
				),
		)
	}

	return fn, codeElements
}
