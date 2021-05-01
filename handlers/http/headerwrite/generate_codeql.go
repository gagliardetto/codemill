package headerwrite

import (
	"github.com/gagliardetto/codebox/scanner"
	"github.com/gagliardetto/codemill/x"
	. "github.com/gagliardetto/cqlgen/jen"
	"github.com/gagliardetto/feparser"
	. "github.com/gagliardetto/utilz"
)

// Predicate names:
const (
	setsHeaderDynamicKeyValue = "setsHeaderDynamicKeyValue"
)

func (han *Handler) GenerateCodeQL(impAdder x.ImportAdder, mdl *x.XModel, rootModuleGroup *Group) error {
	if err := mdl.Validate(); err != nil {
		return err
	}
	if err := han.Validate(mdl); err != nil {
		return err
	}

	{
		// Add imports:
		//impAdder.Import("DataFlow::PathGraph")
	}

	className := mdl.Name
	allPathVersions := mdl.ListAllPathVersions()

	{
		// Header key-value:
		// TODO: what's the name of the class? How to avoid duplicates?
		funcModelsClassName := feparser.NewCodeQlName(className)
		tmp := DoGroup(func(tempFuncsModel *Group) {
			tempFuncsModel.Doc(
				"Models HTTP header writers.",
				"The write is done with a call where you can set both the key and the value of the header.",
			)
			tempFuncsModel.Private().Class().Id(funcModelsClassName).Extends().List(
				Id("HTTP::HeaderWrite::Range"),
				Id("DataFlow::CallNode"),
			).BlockFunc(
				func(blockBody *Group) {

					blockBody.Id("DataFlow::Node").Id("receiverNode").Semicolon().Line()
					blockBody.Id("DataFlow::Node").Id("headerNameNode").Semicolon().Line()
					blockBody.Id("DataFlow::Node").Id("headerValueNode").Semicolon().Line()

					blockBody.Id(funcModelsClassName).Call().Block(
						Id(setsHeaderDynamicKeyValue).Call(
							DontCare(),
							DontCare(),
							This(),
							Id("headerNameNode"),
							Id("headerValueNode"),
							Id("receiverNode"),
						),
					)

					blockBody.Override().Id("DataFlow::Node").Id("getName").Call().Block(
						Id("result").Eq().Id("headerNameNode"),
					)
					blockBody.Override().Id("DataFlow::Node").Id("getValue").Call().Block(
						Id("result").Eq().Id("headerValueNode"),
					)
					blockBody.Override().Id("HTTP::ResponseWriter").Id("getResponseWriter").Call().BlockFunc(
						func(overrideBlockGroup *Group) {
							overrideBlockGroup.Id("result").Dot("getANode").Call().Eq().Id("receiverNode")
						})
				})
		})
		pred := predicate_setsHeaderDynamicKeyValue(allPathVersions, mdl)
		if pred != nil {
			rootModuleGroup.Add(tmp)
			rootModuleGroup.Add(pred)
		}
	}

	{ // Content-Type header writers:
		contentTypeHeaderKey := "content-type"

		{
			// Static content-type:
			mtdStaticValueFromFuncName := mdl.Methods.ByName(MethodCtFromFuncName)
			if len(mtdStaticValueFromFuncName.Selectors) == 0 {
				Infof("No selectors found for %q method.", mtdStaticValueFromFuncName.Name)
			} else {
				hardcodedKey_staticValue(
					contentTypeHeaderKey,
					allPathVersions,
					mtdStaticValueFromFuncName,
					rootModuleGroup,
					x.GuessContentTypeFromName,
				)
			}
		}
		{
			// Dynamic content-type:
			mtdDynamicValue := mdl.Methods.ByName(MethodCt)
			if len(mtdDynamicValue.Selectors) == 0 {
				Infof("No selectors found for %q method.", mtdDynamicValue.Name)
			} else {
				hardcodedKey_dynamicValue(
					contentTypeHeaderKey,
					allPathVersions,
					mtdDynamicValue,
					rootModuleGroup,
				)
			}
		}
	}

	return nil
}

func hardcodedKey_staticValue(
	headerKey string,
	allPathVersions []string,
	mtdStaticValueFromFuncName *x.XMethod,
	rootModuleGroup *Group,
	guesser func(string) string,
) {
	// Static value:
	funcModelsClassName := feparser.NewCodeQlName("Static", headerKey, "HeaderSetter")
	tmp := DoGroup(func(tempFuncsModel *Group) {
		tempFuncsModel.Doc(Sf("Models an HTTP static `%s` header setter.", headerKey))
		tempFuncsModel.Private().Class().Id(funcModelsClassName).Extends().List(
			Id("HTTP::HeaderWrite::Range"),
			Id("DataFlow::CallNode"),
		).BlockFunc(
			func(blockBody *Group) {

				blockBody.Id("DataFlow::Node").Id("receiverNode").Semicolon().Line()
				blockBody.String().Id("valueString").Semicolon().Line()

				blockBody.Id(funcModelsClassName).Call().Block(
					Id("setsStaticHeader"+feparser.NewCodeQlName(headerKey)).Call(
						DontCare(),
						DontCare(),
						This(),
						Id("valueString"),
						Id("receiverNode"),
					),
				)

				blockBody.Override().String().Id("getHeaderName").Call().Block(
					Id("result").Eq().Lit(headerKey),
				)
				blockBody.Override().String().Id("getHeaderValue").Call().Block(
					Id("result").Eq().Id("valueString"),
				)
				blockBody.Override().Id("DataFlow::Node").Id("getName").Call().Block(
					None(),
				)
				blockBody.Override().Id("DataFlow::Node").Id("getValue").Call().Block(
					None(),
				)
				blockBody.Override().Id("HTTP::ResponseWriter").Id("getResponseWriter").Call().BlockFunc(
					func(overrideBlockGroup *Group) {
						overrideBlockGroup.Id("result").Dot("getANode").Call().Eq().Id("receiverNode")
					})
			})
	})
	pred := predicate_setsStaticHeaderValue(
		headerKey,
		allPathVersions,
		mtdStaticValueFromFuncName,
		guesser,
	)
	if pred != nil {
		rootModuleGroup.Add(tmp)
		rootModuleGroup.Add(pred)
	}
}

func hardcodedKey_dynamicValue(
	headerKey string,
	allPathVersions []string,
	mtdDynamicValue *x.XMethod,
	rootModuleGroup *Group,
) {
	// Dynamic value:
	funcModelsClassName := feparser.NewCodeQlName("Dynamic", headerKey, "HeaderSetter")
	tmp := DoGroup(func(tempFuncsModel *Group) {
		tempFuncsModel.Doc(Sf("Models an HTTP dynamic `%s` header setter.", headerKey))
		tempFuncsModel.Private().Class().Id(funcModelsClassName).Extends().List(
			Id("HTTP::HeaderWrite::Range"),
			Id("DataFlow::CallNode"),
		).BlockFunc(
			func(blockBody *Group) {

				blockBody.Id("DataFlow::Node").Id("receiverNode").Semicolon().Line()
				blockBody.Id("DataFlow::Node").Id("valueNode").Semicolon().Line()

				blockBody.Id(funcModelsClassName).Call().Block(
					Id("setsDynamicHeader"+feparser.NewCodeQlName(headerKey)).Call(
						DontCare(),
						DontCare(),
						This(),
						Id("valueNode"),
						Id("receiverNode"),
					),
				)

				blockBody.Override().String().Id("getHeaderName").Call().Block(
					Id("result").Eq().Lit(headerKey),
				)
				blockBody.Override().Id("DataFlow::Node").Id("getName").Call().Block(
					None(),
				)
				blockBody.Override().Id("DataFlow::Node").Id("getValue").Call().Block(
					Id("result").Eq().Id("valueNode"),
				)
				blockBody.Override().Id("HTTP::ResponseWriter").Id("getResponseWriter").Call().BlockFunc(
					func(overrideBlockGroup *Group) {
						overrideBlockGroup.Id("result").Dot("getANode").Call().Eq().Id("receiverNode")
					})
			})
	})
	pred := predicate_setsDynamicHeaderValue(
		headerKey,
		allPathVersions,
		mtdDynamicValue,
	)
	if pred != nil {
		rootModuleGroup.Add(tmp)
		rootModuleGroup.Add(pred)
	}
}
func predicate_setsHeaderDynamicKeyValue(allPathVersions []string, mdl *x.XModel) Code {
	predicate := Comment("Holds for a call that sets a header with a key-value combination.").
		Private().Predicate().Id(setsHeaderDynamicKeyValue).Call(
		List(
			String().Id("package"),
			String().Id("receiverName"),
			Id("DataFlow::CallNode").Id("headerSetterCall"),
			Id("DataFlow::Node").Id("headerNameNode"),
			Id("DataFlow::Node").Id("headerValueNode"),
			Id("DataFlow::Node").Id("receiverNode"),
		),
	)

	addedCount := 0
	predicate.BlockFunc(func(predicateBlock *Group) {
		{
			pc := par_cql_DynamicHeaderKeyValue(mdl, allPathVersions)
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

func predicate_setsStaticHeaderValue(
	headerKey string,
	allPathVersions []string,
	mtdStaticValueFromFuncName *x.XMethod,
	guesser func(string) string,
) Code {
	predicate := Commentf("Holds for a call that sets the `%s` header (implicit).", headerKey).
		Private().Predicate().Id("setsStaticHeader" + feparser.NewCodeQlName(headerKey)).Call(
		List(
			String().Id("package"),
			String().Id("receiverName"),
			Id("DataFlow::CallNode").Id("setterCall"),
			String().Id("valueString"),
			Id("DataFlow::Node").Id("receiverNode"),
		),
	)

	addedCount := 0
	predicate.BlockFunc(func(predicateBlock *Group) {
		{
			pc := par_cql_MethodStaticValueFromFuncName(
				headerKey,
				allPathVersions,
				mtdStaticValueFromFuncName,
				guesser,
			)
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

func predicate_setsDynamicHeaderValue(
	headerKey string,
	allPathVersions []string,
	mtdDynamicValue *x.XMethod,
) Code {
	predicate := Commentf("Holds for a call that sets the `%s` header via a parameter.", headerKey).
		Private().Predicate().Id("setsDynamicHeader" + feparser.NewCodeQlName(headerKey)).Call(
		List(
			String().Id("package"),
			String().Id("receiverName"),
			Id("DataFlow::CallNode").Id("setterCall"),
			Id("DataFlow::Node").Id("valueNode"),
			Id("DataFlow::Node").Id("receiverNode"),
		),
	)

	addedCount := 0
	predicate.BlockFunc(func(predicateBlock *Group) {
		{
			pc := par_cql_MethodHeaderValueNode(
				headerKey,
				allPathVersions,
				mtdDynamicValue,
			)
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

func generateCasesForHeaderKeyValWriters(
	b2Key x.BasicToReceiverIDToMethods,
	b2Val x.BasicToReceiverIDToMethods,
	pathVersion string,
	isInterface bool,
) []Code {

	tempForPathVersion := make([]Code, 0)
	b2Key.IterValid(pathVersion,
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

			if isInterface {
				receiverGroup.Commentf("Receiver interface: %s", typ.TypeString)
			} else {
				receiverGroup.Commentf("Receiver type: %s", typ.TypeString)
			}
			receiverGroup.Id("receiverName").Eq().Lit(typ.TypeName)
			receiverGroup.And()

			receiverGroup.ParensFunc(func(st *Group) {
				methodIndex := 0
				for _, keyQual := range methodQualifiers {
					if AllFalse(keyQual.Pos...) {
						continue
					}
					if methodIndex > 0 {
						st.Or()
					}
					methodIndex++

					fn := x.GetFuncByQualifier(keyQual)

					// TODO:
					// - Check if found.
					valQual := b2Val[pathVersion][receiverTypeID].ByBasicQualifier(keyQual.BasicQualifier)
					if valQual == nil {
						Fatalf("Header val method not found: %v", keyQual.BasicQualifier)
					}
					st.DoGroup(
						func(par *Group) {
							par.Commentf("signature: %s", fn.GetFunc().Signature)

							par.Id("methodName").Eq().Lit(fn.GetFunc().Name)

							par.And()

							{
								_, code := x.CqlParamQualToCode("headerSetterCall", "getArgument", keyQual)
								par.Id("headerNameNode").Eq().Add(code)
							}
							par.And()
							{
								_, code := x.CqlParamQualToCode("headerSetterCall", "getArgument", valQual)
								par.Id("headerValueNode").Eq().Add(code)
							}
						},
					)
				}
			})

			tempForPathVersion = append(tempForPathVersion, receiverGroup)
		})

	return tempForPathVersion
}

func par_cql_DynamicHeaderKeyValue(mdl *x.XModel, pathVersions []string) []Code {

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

	_, b2tmKey, b2itmKey, err := x.GroupFuncSelectors(methodWriteHeaderKey)
	if err != nil {
		Fatalf("Error while GroupFuncSelectors: %s", err)
	}
	_, b2tmVal, b2itmVal, err := x.GroupFuncSelectors(methodWriteHeaderVal)
	if err != nil {
		Fatalf("Error while GroupFuncSelectors: %s", err)
	}

	pathCodez := make([]Code, 0)

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
				st.Id("headerSetterCall").Eq().Id("met").Dot("getACall").Call()
				st.And()
				st.Id("receiverNode").Eq().Id("headerSetterCall").Dot("getReceiver").Call()
			}),
			DoGroup(func(exists3 *Group) {
				for _, pathVersion := range pathVersions {

					tempForPathVersion := generateCasesForHeaderKeyValWriters(
						x.BasicToReceiverIDToMethods(b2tmKey),
						x.BasicToReceiverIDToMethods(b2tmVal),
						pathVersion,
						false,
					)

					if len(tempForPathVersion) > 0 {
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
				st.Id("headerSetterCall").Eq().Id("met").Dot("getACall").Call()
				st.And()
				st.Id("receiverNode").Eq().Id("headerSetterCall").Dot("getReceiver").Call()
			}),
			DoGroup(func(exists3 *Group) {
				for _, pathVersion := range pathVersions {

					tempForPathVersion := generateCasesForHeaderKeyValWriters(
						x.BasicToReceiverIDToMethods(b2itmKey),
						x.BasicToReceiverIDToMethods(b2itmVal),
						pathVersion,
						true,
					)

					if len(tempForPathVersion) > 0 {
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

func par_cql_MethodStaticValueFromFuncName(
	headerKey string,
	pathVersions []string,
	mtdStaticValueFromFuncName *x.XMethod,
	guesser func(string) string,
) []Code {

	b2fe, b2tm, b2itm, err := x.GroupFuncSelectors(mtdStaticValueFromFuncName)
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
				st.Id("setterCall").Eq().Id("met").Dot("getACall").Call()
				st.And()
				st.Id("receiverNode").Eq().Id("setterCall").Dot("getReceiver").Call()
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

									fn := x.GetFuncByQualifier(methodQual)
									pathVersionAddedCount++

									st.DoGroup(
										func(par *Group) {
											par.Commentf("signature: %s", fn.GetFunc().Signature)

											par.Id("methodName").Eq().Lit(fn.GetFunc().Name)

											par.And()

											{
												par.Id("valueString").Eq().Lit(guesser(fn.GetFunc().Name))
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
				st.Id("setterCall").Eq().Id("met").Dot("getACall").Call()
				st.And()
				st.Id("receiverNode").Eq().Id("setterCall").Dot("getReceiver").Call()
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

									fn := x.GetFuncByQualifier(methodQual)
									pathVersionAddedCount++

									st.DoGroup(
										func(par *Group) {
											par.Commentf("signature: %s", fn.GetFunc().Signature)

											par.Id("methodName").Eq().Lit(fn.GetFunc().Name)

											par.And()

											{
												par.Id("valueString").Eq().Lit(guesser(fn.GetFunc().Name))
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

func par_cql_MethodHeaderValueNode(
	headerKey string,
	pathVersions []string,
	mtdDynamicValue *x.XMethod,
) []Code {

	b2fe, b2tm, b2itm, err := x.GroupFuncSelectors(mtdDynamicValue)
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
				st.Id("setterCall").Eq().Id("met").Dot("getACall").Call()
				st.And()
				st.Id("receiverNode").Eq().Id("setterCall").Dot("getReceiver").Call()
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

									fn := x.GetFuncByQualifier(methodQual)
									pathVersionAddedCount++

									st.DoGroup(
										func(par *Group) {
											par.Commentf("signature: %s", fn.GetFunc().Signature)

											par.Id("methodName").Eq().Lit(fn.GetFunc().Name)

											par.And()

											{
												_, code := GetHeaderValueSetterFuncQualifierCodeElements(methodQual)
												par.Id("valueNode").Eq().Add(code)
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
				st.Id("setterCall").Eq().Id("met").Dot("getACall").Call()
				st.And()
				st.Id("receiverNode").Eq().Id("setterCall").Dot("getReceiver").Call()
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

									fn := x.GetFuncByQualifier(methodQual)
									pathVersionAddedCount++

									st.DoGroup(
										func(par *Group) {
											par.Commentf("signature: %s", fn.GetFunc().Signature)

											par.Id("methodName").Eq().Lit(fn.GetFunc().Name)

											par.And()

											{
												_, code := GetHeaderValueSetterFuncQualifierCodeElements(methodQual)
												par.Id("valueNode").Eq().Add(code)
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

func GetHeaderValueSetterFuncQualifierCodeElements(qual *x.FuncQualifier) (x.FuncInterface, Code) {
	return x.CqlParamQualToCode("setterCall", "getArgument", qual)
}

func GetFuncQualifierCodeElements(qual *x.FuncQualifier) (x.FuncInterface, Code) {
	return x.CqlParamQualToCode("this", "getArgument", qual)
}
