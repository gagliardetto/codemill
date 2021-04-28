package headerwrite

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
	methodWriteHeaderKey := mdl.Methods.ByName(MethodWriteHeaderKey)
	if len(methodWriteHeaderKey.Selectors) == 0 {
		Infof("No selectors found for %q method.", methodWriteHeaderKey.Name)
		return nil
	}

	methodWriteHeaderVal := mdl.Methods.ByName(MethodWriteHeaderVal)
	if len(methodWriteHeaderKey.Selectors) == 0 {
		Infof("No selectors found for %q method.", methodWriteHeaderKey.Name)
		return nil
	}

	{
		// Add imports:
		//impAdder.Import("DataFlow::PathGraph")
	}

	className := mdl.Name
	allPathVersions := mdl.ListAllPathVersions()

	_, b2tmKey, b2itmKey, err := x.GroupFuncSelectors(methodWriteHeaderKey)
	if err != nil {
		Fatalf("Error while GroupFuncSelectors: %s", err)
	}
	_, b2tmVal, b2itmVal, err := x.GroupFuncSelectors(methodWriteHeaderVal)
	if err != nil {
		Fatalf("Error while GroupFuncSelectors: %s", err)
	}

	{
		addedCount := 0
		funcModelsClassName := feparser.NewCodeQlName(className)
		for _, pathVersion := range allPathVersions {
			tmp := DoGroup(func(tempFuncsModel *Group) {
				tempFuncsModel.Doc(Sf("Models HTTP header writer models for package: %s", pathVersion))
				tempFuncsModel.Private().Class().Id(funcModelsClassName).Extends().List(
					Id("HTTP::HeaderWrite::Range"),
					Id("DataFlow::CallNode"),
				).BlockFunc(
					func(funcModelsClassGroup *Group) {

						funcModelsClassGroup.String().Id("receiverName").Semicolon()
						funcModelsClassGroup.String().Id("methodName").Semicolon()
						funcModelsClassGroup.Qual("DataFlow", "Node").Id("headerNameNode").Semicolon()
						funcModelsClassGroup.Qual("DataFlow", "Node").Id("headerValueNode").Semicolon()

						funcModelsClassGroup.Id(funcModelsClassName).Call().BlockFunc(
							func(funcModelsSelfMethodGroup *Group) {
								{
									funcModelsSelfMethodGroup.DoGroup(
										func(groupCase *Group) {
											pathCodez := make([]Code, 0)

											// Type methods:
											{
												typeMethodPathCodez := generateCasesForHeaderKeyValWriters(
													x.BasicToReceiverIDToMethods(b2tmKey),
													x.BasicToReceiverIDToMethods(b2tmVal),
													pathVersion,
													false,
												)

												if len(typeMethodPathCodez) > 0 {
													typeMethodCases := ParensFunc(
														func(par *Group) {
															par.Comment("Type methods:")
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
																			Id("receiverName"),
																			Id("methodName"),
																		)
																	}),
																	nil,
																).Dot("getACall").Call()

															par.And()

															par.Parens(
																Join(
																	Or(),
																	typeMethodPathCodez...,
																),
															)

														})

													pathCodez = append(pathCodez, typeMethodCases)
												}
											}

											// Interface methods:
											{
												interfaceMethodPathCodez := generateCasesForHeaderKeyValWriters(
													x.BasicToReceiverIDToMethods(b2itmKey),
													x.BasicToReceiverIDToMethods(b2itmVal),
													pathVersion,
													true,
												)

												if len(interfaceMethodPathCodez) > 0 {
													interfaceMethodCases := ParensFunc(
														func(par *Group) {
															par.Comment("Interface methods:")
															par.This().
																Eq().
																Any(
																	DoGroup(func(gr *Group) {
																		gr.Id("Method").Id("m")
																	}),
																	DoGroup(func(gr *Group) {
																		path, _ := scanner.SplitPathVersion(pathVersion)

																		gr.Id("m").Dot("implements").Call(
																			x.CqlFormatPackagePath(path),
																			Id("receiverName"),
																			Id("methodName"),
																		)
																	}),
																	nil,
																).Dot("getACall").Call()

															par.And()

															par.Parens(
																Join(
																	Or(),
																	interfaceMethodPathCodez...,
																),
															)

														})

													pathCodez = append(pathCodez, interfaceMethodCases)
												}
											}

											if len(pathCodez) > 0 {
												if addedCount > 0 {
													groupCase.Or()
												}

												groupCase.Join(
													Or(),
													pathCodez...,
												)

												addedCount++
											}
										})
								}
							})

						funcModelsClassGroup.Override().Id("DataFlow::Node").Id("getName").Call().BlockFunc(
							func(overrideBlockGroup *Group) {
								overrideBlockGroup.Id("result").Eq().Id("headerNameNode")
							})
						funcModelsClassGroup.Override().Id("DataFlow::Node").Id("getValue").Call().BlockFunc(
							func(overrideBlockGroup *Group) {
								overrideBlockGroup.Id("result").Eq().Id("headerValueNode")
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
	}

	{
		{
			// Static content-type:
			funcModelsClassName := feparser.NewCodeQlName("StaticContentTypeSetter")
			tmp := DoGroup(func(tempFuncsModel *Group) {
				tempFuncsModel.Doc("Models an HTTP static content-type setter.")
				tempFuncsModel.Private().Class().Id(funcModelsClassName).Extends().List(
					Id("HTTP::HeaderWrite::Range"),
					Id("DataFlow::CallNode"),
				).BlockFunc(
					func(blockContent *Group) {

						blockContent.Id("DataFlow::Node").Id("receiverNode").Semicolon().Line()
						blockContent.String().Id("contentTypeString").Semicolon().Line()

						blockContent.Id(funcModelsClassName).Call().Block(
							Id("setsStaticContentType").Call(
								DontCare(),
								DontCare(),
								This(),
								Id("contentTypeString"),
								Id("receiverNode"),
							),
						)

						blockContent.Override().String().Id("getHeaderName").Call().Block(
							Id("result").Eq().Lit("content-type"),
						)
						blockContent.Override().String().Id("getHeaderValue").Call().Block(
							Id("result").Eq().Id("contentTypeString"),
						)
						blockContent.Override().Id("DataFlow::Node").Id("getName").Call().Block(
							None(),
						)
						blockContent.Override().Id("DataFlow::Node").Id("getValue").Call().Block(
							None(),
						)
						blockContent.Override().Id("HTTP::ResponseWriter").Id("getResponseWriter").Call().BlockFunc(
							func(overrideBlockGroup *Group) {
								overrideBlockGroup.Id("result").Dot("getANode").Call().Eq().Id("receiverNode")
							})
					})
			})
			pred := predicate_setsStaticContentType(allPathVersions, mdl)
			if pred != nil {
				rootModuleGroup.Add(tmp)
				rootModuleGroup.Add(pred)
			}
		}
		{
			// Dynamic content-type:
			funcModelsClassName := feparser.NewCodeQlName("DynamicContentTypeSetter")
			tmp := DoGroup(func(tempFuncsModel *Group) {
				tempFuncsModel.Doc("Models an HTTP dynamic content-type setter.")
				tempFuncsModel.Private().Class().Id(funcModelsClassName).Extends().List(
					Id("HTTP::HeaderWrite::Range"),
					Id("DataFlow::CallNode"),
				).BlockFunc(
					func(blockContent *Group) {

						blockContent.Id("DataFlow::Node").Id("receiverNode").Semicolon().Line()
						blockContent.Id("DataFlow::Node").Id("contentTypeNode").Semicolon().Line()

						blockContent.Id(funcModelsClassName).Call().Block(
							Id("setsDynamicContentType").Call(
								DontCare(),
								DontCare(),
								This(),
								Id("contentTypeNode"),
								Id("receiverNode"),
							),
						)

						blockContent.Override().String().Id("getHeaderName").Call().Block(
							Id("result").Eq().Lit("content-type"),
						)
						blockContent.Override().String().Id("getHeaderValue").Call().Block(
							None(),
						)
						blockContent.Override().Id("DataFlow::Node").Id("getName").Call().Block(
							None(),
						)
						blockContent.Override().Id("DataFlow::Node").Id("getValue").Call().Block(
							Id("result").Eq().Id("contentTypeNode"),
						)
						blockContent.Override().Id("HTTP::ResponseWriter").Id("getResponseWriter").Call().BlockFunc(
							func(overrideBlockGroup *Group) {
								overrideBlockGroup.Id("result").Dot("getANode").Call().Eq().Id("receiverNode")
							})
					})
			})
			pred := predicate_setsDynamicContentType(allPathVersions, mdl)
			if pred != nil {
				rootModuleGroup.Add(tmp)
				rootModuleGroup.Add(pred)
			}
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

func generateCasesForHeaderKeyValWriters(
	b2Key x.BasicToReceiverIDToMethods,
	b2Val x.BasicToReceiverIDToMethods,
	pathVersion string,
	isInterface bool,
) []Code {

	methodPathCodez := make([]Code, 0)
	b2Key.IterValid(pathVersion,
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

				if isInterface {
					mtdGroup.Commentf("Receiver interface: %s", typ.TypeString)
				} else {
					mtdGroup.Commentf("Receiver type: %s", typ.TypeString)
				}

				mtdGroup.Id("receiverName").Eq().Lit(typ.TypeName)
				mtdGroup.And()

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
							thing := fn

							// TODO:
							// - Check if found.
							valMethodQual := b2Val[pathVersion][receiverTypeID].ByBasicQualifier(keyMethodQual.BasicQualifier)
							if valMethodQual == nil {
								Fatalf("Header val method not found: %v", keyMethodQual.BasicQualifier)
							}
							parMethods.ParensFunc(
								func(par *Group) {
									par.Commentf("signature: %s", thing.GetFunc().Signature)

									par.Id("methodName").Eq().Lit(thing.GetFunc().Name)
									par.And()

									{
										_, code := GetFuncQualifierCodeElements(keyMethodQual)
										par.Id("headerNameNode").Eq().Add(code)
									}
									par.And()
									{
										_, code := GetFuncQualifierCodeElements(valMethodQual)
										par.Id("headerValueNode").Eq().Add(code)
									}
								},
							)

						}
					},
				)

			})
			methodPathCodez = append(methodPathCodez, codez)
		})

	return methodPathCodez
}

// Predicate names:
const (
	setsStaticContentType  = "setsStaticContentType"
	setsDynamicContentType = "setsDynamicContentType"
)

func predicate_setsStaticContentType(allPathVersions []string, mdl *x.XModel) Code {
	predicate := Comment("Holds for a call that sets the content-type (implicit).").
		Private().Predicate().Id(setsStaticContentType).Call(
		List(
			String().Id("package"),
			String().Id("receiverName"),
			Id("DataFlow::CallNode").Id("contentTypeSetterCall"),
			String().Id("contentTypeString"),
			Id("DataFlow::Node").Id("receiverNode"),
		),
	)

	addedCount := 0
	predicate.BlockFunc(func(predicateBlock *Group) {
		{
			pc := par_cql_MethodCtFromFuncName(mdl, allPathVersions)
			if len(pc) > 0 {
				addedCount++
			}
			predicateBlock.Add(pc...)
		}
	})
	if addedCount == 0 {
		return nil
	}
	return predicate
}

func predicate_setsDynamicContentType(allPathVersions []string, mdl *x.XModel) Code {
	predicate := Comment("Holds for a call that sets the content-type via a parameter.").
		Private().Predicate().Id(setsDynamicContentType).Call(
		List(
			String().Id("package"),
			String().Id("receiverName"),
			Id("DataFlow::CallNode").Id("contentTypeSetterCall"),
			Id("DataFlow::Node").Id("contentTypeNode"),
			Id("DataFlow::Node").Id("receiverNode"),
		),
	)

	addedCount := 0
	predicate.BlockFunc(func(predicateBlock *Group) {
		{
			pc := par_cql_MethodCt(mdl, allPathVersions)
			if len(pc) > 0 {
				addedCount++
			}
			predicateBlock.Add(pc...)
		}
	})
	if addedCount == 0 {
		return nil
	}
	return predicate
}

func par_cql_MethodCtFromFuncName(mdl *x.XModel, pathVersions []string) []Code {

	mtdCtFromFuncName := mdl.Methods.ByName(MethodCtFromFuncName)
	if len(mtdCtFromFuncName.Selectors) == 0 {
		Infof("No selectors found for %q method.", mtdCtFromFuncName.Name)
		return nil
	}

	b2fe, b2tm, b2itm, err := x.GroupFuncSelectors(mtdCtFromFuncName)
	if err != nil {
		Fatalf("Error while GroupFuncSelectors: %s", err)
	}

	pathCodez := make([]Code, 0)
	// Functions:
	{
		for _, pathVersion := range pathVersions {
			_, ok := b2fe[pathVersion]
			if ok {
				panic("Not implemented")
			}
		}
	}
	// Type methods:
	{
		addedCount := 0
		exists := Exists(
			List(
				String().Id("methodName"),
				Id("Method").Id("met"),
			),
			DoGroup(func(st *Group) {
				st.Id("met").Dot("hasQualifiedName").Call(
					Id("package"),
					Id("receiverName"),
					Id("methodName"),
				)
				st.And()
				st.Id("contentTypeSetterCall").Eq().Id("met").Dot("getACall").Call()
				st.And()
				st.Id("receiverNode").Eq().Id("contentTypeSetterCall").Dot("getReceiver").Call()
			}),
			DoGroup(func(exists3 *Group) {
				for _, pathVersion := range pathVersions {
					pathVersionAddedCount := 0

					tempForPathVersion := make([]Code, 0)
					b2tm.IterValid(pathVersion,
						func(receiverTypeID string, methodQualifiers x.FuncQualifierSlice) {

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

							receiverGroup := &Group{}

							receiverGroup.Commentf("Receiver type: %s", typ.TypeString)
							receiverGroup.Id("receiverName").Eq().Lit(typ.TypeName)
							receiverGroup.And()
							receiverGroup.ParensFunc(func(st *Group) {
								methodIndex := 0
								for _, methodQual := range methodQualifiers {
									if AllFalse(methodQual.Pos...) {
										continue
									}
									if methodIndex > 0 {
										st.Or()
									}
									methodIndex++

									fn := GetFunc(methodQual)
									pathVersionAddedCount++

									st.DoGroup(
										func(par *Group) {
											par.Commentf("signature: %s", fn.GetFunc().Signature)

											par.Id("methodName").Eq().Lit(fn.GetFunc().Name)

											par.And()

											{
												par.Id("contentTypeString").Eq().Lit(x.GuessContentTypeFromName(fn.GetFunc().Name))
											}
										},
									)
								}
							})

							tempForPathVersion = append(tempForPathVersion, receiverGroup)
						})

					if pathVersionAddedCount > 0 {
						if addedCount > 0 {
							exists3.Or()
						}
						addedCount++
						path, _ := scanner.SplitPathVersion(pathVersion)
						exists3.Id("package").Eq().Add(x.CqlFormatPackagePath(path))
						exists3.And()
						exists3.Parens(
							Join(Or(), tempForPathVersion...),
						)
					}
				}
			}),
		)
		if addedCount > 0 {
			pathCodez = append(pathCodez, exists)
		}
	}
	// Interface methods:
	{
		addedCount := 0
		exists := Exists(
			List(
				String().Id("methodName"),
				Id("Method").Id("met"),
			),
			DoGroup(func(st *Group) {
				st.Id("met").Dot("implements").Call(
					Id("package"),
					Id("interfaceName"),
					Id("methodName"),
				)
				st.And()
				st.Id("contentTypeSetterCall").Eq().Id("met").Dot("getACall").Call()
				st.And()
				st.Id("receiverNode").Eq().Id("contentTypeSetterCall").Dot("getReceiver").Call()
			}),
			DoGroup(func(exists3 *Group) {
				for _, pathVersion := range pathVersions {
					pathVersionAddedCount := 0

					tempForPathVersion := make([]Code, 0)
					b2itm.IterValid(pathVersion,
						func(receiverTypeID string, methodQualifiers x.FuncQualifierSlice) {

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

							receiverGroup := &Group{}

							receiverGroup.Commentf("Receiver interface: %s", typ.TypeString)
							receiverGroup.Id("interfaceName").Eq().Lit(typ.TypeName)
							receiverGroup.And()
							receiverGroup.ParensFunc(func(st *Group) {
								methodIndex := 0
								for _, methodQual := range methodQualifiers {
									if AllFalse(methodQual.Pos...) {
										continue
									}
									if methodIndex > 0 {
										st.Or()
									}
									methodIndex++

									fn := GetFunc(methodQual)
									pathVersionAddedCount++

									st.DoGroup(
										func(par *Group) {
											par.Commentf("signature: %s", fn.GetFunc().Signature)

											par.Id("methodName").Eq().Lit(fn.GetFunc().Name)

											par.And()

											{
												par.Id("contentTypeString").Eq().Lit(x.GuessContentTypeFromName(fn.GetFunc().Name))
											}
										},
									)
								}
							})

							tempForPathVersion = append(tempForPathVersion, receiverGroup)
						})

					if pathVersionAddedCount > 0 {
						if addedCount > 0 {
							exists3.Or()
						}
						addedCount++
						path, _ := scanner.SplitPathVersion(pathVersion)
						exists3.Id("package").Eq().Add(x.CqlFormatPackagePath(path))
						exists3.And()
						exists3.Parens(
							Join(Or(), tempForPathVersion...),
						)
					}
				}
			}),
		)
		if addedCount > 0 {
			pathCodez = append(pathCodez, exists)
		}
	}
	return pathCodez
}

func par_cql_MethodCt(mdl *x.XModel, pathVersions []string) []Code {

	mtdCt := mdl.Methods.ByName(MethodCt)
	if len(mtdCt.Selectors) == 0 {
		Infof("No selectors found for %q method.", mtdCt.Name)
		return nil
	}

	b2fe, b2tm, b2itm, err := x.GroupFuncSelectors(mtdCt)
	if err != nil {
		Fatalf("Error while GroupFuncSelectors: %s", err)
	}

	pathCodez := make([]Code, 0)
	// Functions:
	{
		for _, pathVersion := range pathVersions {
			_, ok := b2fe[pathVersion]
			if ok {
				panic("Not implemented")
			}
		}
	}
	// Type methods:
	{
		addedCount := 0
		exists := Exists(
			List(
				String().Id("methodName"),
				Id("Method").Id("met"),
			),
			DoGroup(func(st *Group) {
				st.Id("met").Dot("hasQualifiedName").Call(
					Id("package"),
					Id("receiverName"),
					Id("methodName"),
				)
				st.And()
				st.Id("contentTypeSetterCall").Eq().Id("met").Dot("getACall").Call()
				st.And()
				st.Id("receiverNode").Eq().Id("contentTypeSetterCall").Dot("getReceiver").Call()
			}),
			DoGroup(func(exists3 *Group) {
				for _, pathVersion := range pathVersions {
					pathVersionAddedCount := 0

					tempForPathVersion := make([]Code, 0)
					b2tm.IterValid(pathVersion,
						func(receiverTypeID string, methodQualifiers x.FuncQualifierSlice) {

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

							receiverGroup := &Group{}

							receiverGroup.Commentf("Receiver type: %s", typ.TypeString)
							receiverGroup.Id("receiverName").Eq().Lit(typ.TypeName)
							receiverGroup.And()
							receiverGroup.ParensFunc(func(st *Group) {
								methodIndex := 0
								for _, methodQual := range methodQualifiers {
									if AllFalse(methodQual.Pos...) {
										continue
									}
									if methodIndex > 0 {
										st.Or()
									}
									methodIndex++

									fn := GetFunc(methodQual)
									pathVersionAddedCount++

									st.DoGroup(
										func(par *Group) {
											par.Commentf("signature: %s", fn.GetFunc().Signature)

											par.Id("methodName").Eq().Lit(fn.GetFunc().Name)

											par.And()

											{
												_, code := GetContentTypeSetterFuncQualifierCodeElements(methodQual)
												par.Id("contentTypeNode").Eq().Add(code)
											}
										},
									)
								}
							})

							tempForPathVersion = append(tempForPathVersion, receiverGroup)
						})

					if pathVersionAddedCount > 0 {
						if addedCount > 0 {
							exists3.Or()
						}
						addedCount++
						path, _ := scanner.SplitPathVersion(pathVersion)
						exists3.Id("package").Eq().Add(x.CqlFormatPackagePath(path))
						exists3.And()
						exists3.Parens(
							Join(Or(), tempForPathVersion...),
						)
					}
				}
			}),
		)
		if addedCount > 0 {
			pathCodez = append(pathCodez, exists)
		}
	}
	// Interface methods:
	{
		addedCount := 0
		exists := Exists(
			List(
				String().Id("methodName"),
				Id("Method").Id("met"),
			),
			DoGroup(func(st *Group) {
				st.Id("met").Dot("implements").Call(
					Id("package"),
					Id("interfaceName"),
					Id("methodName"),
				)
				st.And()
				st.Id("contentTypeSetterCall").Eq().Id("met").Dot("getACall").Call()
				st.And()
				st.Id("receiverNode").Eq().Id("contentTypeSetterCall").Dot("getReceiver").Call()
			}),
			DoGroup(func(exists3 *Group) {
				for _, pathVersion := range pathVersions {
					pathVersionAddedCount := 0

					tempForPathVersion := make([]Code, 0)
					b2itm.IterValid(pathVersion,
						func(receiverTypeID string, methodQualifiers x.FuncQualifierSlice) {

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

							receiverGroup := &Group{}

							receiverGroup.Commentf("Receiver interface: %s", typ.TypeString)
							receiverGroup.Id("interfaceName").Eq().Lit(typ.TypeName)
							receiverGroup.And()
							receiverGroup.ParensFunc(func(st *Group) {
								methodIndex := 0
								for _, methodQual := range methodQualifiers {
									if AllFalse(methodQual.Pos...) {
										continue
									}
									if methodIndex > 0 {
										st.Or()
									}
									methodIndex++

									fn := GetFunc(methodQual)
									pathVersionAddedCount++

									st.DoGroup(
										func(par *Group) {
											par.Commentf("signature: %s", fn.GetFunc().Signature)

											par.Id("methodName").Eq().Lit(fn.GetFunc().Name)

											par.And()

											{
												_, code := GetContentTypeSetterFuncQualifierCodeElements(methodQual)
												par.Id("contentTypeNode").Eq().Add(code)
											}
										},
									)
								}
							})

							tempForPathVersion = append(tempForPathVersion, receiverGroup)
						})

					if pathVersionAddedCount > 0 {
						if addedCount > 0 {
							exists3.Or()
						}
						addedCount++
						path, _ := scanner.SplitPathVersion(pathVersion)
						exists3.Id("package").Eq().Add(x.CqlFormatPackagePath(path))
						exists3.And()
						exists3.Parens(
							Join(Or(), tempForPathVersion...),
						)
					}
				}
			}),
		)
		if addedCount > 0 {
			pathCodez = append(pathCodez, exists)
		}
	}
	return pathCodez
}
func GetContentTypeSetterFuncQualifierCodeElements(qual *x.FuncQualifier) (x.FuncInterface, Code) {
	fn := GetFunc(qual)

	parameterIndexes := x.MustPosToRelativeParamIndexes(fn, qual.Pos)
	code := x.GenCqlParamQual("contentTypeSetterCall", "getArgument", fn, parameterIndexes)

	return fn, code
}
