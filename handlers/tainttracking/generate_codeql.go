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
													if !funcQual.Flows.Enabled || x.AllBlocksEmpty(funcQual.Flows.Blocks...) {
														continue
													}

													fn, codeElements := GetFuncQualifierCodeElements(funcQual)
													thing := fn.(*feparser.FEFunc)
													pathCodez = append(pathCodez,
														ParensFunc(
															func(par *Group) {
																par.Commentf("Function: %s", thing.Signature)
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
													typeIndex := 0
													for _, receiverTypeID := range keys {
														methodQualifiers := cont[receiverTypeID]
														if len(methodQualifiers) == 0 {
															continue
														}
														codez := DoGroup(func(mtdGroup *Group) {
															if typeIndex > 0 {
																mtdGroup.Or()
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

															mtdGroup.Commentf("Receiver: %s", typ.TypeString)

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
																				par.Commentf("Method: %s", thing.Func.Signature)
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
												typeIndex := 0
												for _, receiverTypeID := range keys {
													methodQualifiers := contb2itm[receiverTypeID]
													if len(methodQualifiers) == 0 {
														continue
													}
													codez := DoGroup(func(mtdGroup *Group) {
														if typeIndex > 0 {
															mtdGroup.Or()
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
														mtdGroup.Commentf("Receiver: %s", typ.TypeString)

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
																			par.Commentf("Method: %s", thing.Func.Signature)
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
		inpParameterIndexes := make([]int, 0)
		inpResultIndexes := make([]int, 0)

	InpLoop:
		for inpPos, ok := range block.Inp {
			if !ok {
				continue InpLoop
			}

			inpElTyp, _, inpRelIndex, err := fn.GetRelativeElement(inpPos)
			if err != nil {
				Fatalf("Error while GetRelativeElement: %s", err)
			}

			switch inpElTyp {
			case feparser.ElementReceiver:
				{
					inpCodeElements = append(inpCodeElements,
						Id("inp").Dot("isReceiver").Call(),
					)
				}
			case feparser.ElementParameter:
				{
					inpParameterIndexes = append(inpParameterIndexes,
						inpRelIndex,
					)
				}
			case feparser.ElementResult:
				{
					inpResultIndexes = append(inpResultIndexes,
						inpRelIndex,
					)
				}
			default:
				panic(Sf("Unknown type: %q", inpElTyp))
			}
		}

		outCodeElements := make([]Code, 0)
		outParameterIndexes := make([]int, 0)
		outResultIndexes := make([]int, 0)
	OutLoop:
		for outPos, ok := range block.Out {
			if !ok {
				continue OutLoop
			}

			outElTyp, _, outRelIndex, err := fn.GetRelativeElement(outPos)
			if err != nil {
				Fatalf("Error while GetRelativeElement: %s", err)
			}

			switch outElTyp {
			case feparser.ElementReceiver:
				{
					outCodeElements = append(outCodeElements,
						Id("out").Dot("isReceiver").Call(),
					)
				}
			case feparser.ElementParameter:
				{
					outParameterIndexes = append(outParameterIndexes,
						outRelIndex,
					)
				}
			case feparser.ElementResult:
				{
					outResultIndexes = append(outResultIndexes,
						outRelIndex,
					)
				}
			default:
				panic(Sf("Unknown type: %q", outElTyp))
			}
		}

		inpCodeElements = append(inpCodeElements, genFunctionInputOutput("inp", fn, inpParameterIndexes, inpResultIndexes)...)
		outCodeElements = append(outCodeElements, genFunctionInputOutput("out", fn, outParameterIndexes, outResultIndexes)...)

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

func genFunctionInputOutput(idName string, fn x.FuncInterface, parameterIndexes []int, resultIndexes []int) []Code {
	codeElements := make([]Code, 0)
	_, lenParams, lenResults := fn.Lengths()

	if len(parameterIndexes) > 0 {
		// If all parameters are selected,
		// and there is more than one possible parameters,
		// then use a `_`:
		if lenParams == len(parameterIndexes) && lenParams > 1 {
			codeElements = append(codeElements,
				Id(idName).Dot("isParameter").Call(DontCare()),
			)

		} else {
			// If multiple parameters are selected (but not all)
			// then use a set, or just the index.
			// If there is only one possible parameter and it is selected,
			// then `isParameter(0)` is used.
			codeElements = append(codeElements,
				Id(idName).Dot("isParameter").Call(
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
					Id(idName).Dot("isResult").Call(),
				)
			} else {
				// If there are more than one results,
				// and all results are selected, then use a `_`:
				codeElements = append(codeElements,
					Id(idName).Dot("isResult").Call(DontCare()),
				)
			}
		} else {
			codeElements = append(codeElements,
				Id(idName).Dot("isResult").Call(IntsToSetOrLit(resultIndexes...)),
			)
		}
	}
	return codeElements
}
