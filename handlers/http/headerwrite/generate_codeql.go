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
	setsStaticContentType     = "setsStaticContentType"
	setsDynamicContentType    = "setsDynamicContentType"
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
		// TODO: what's the name?
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
				func(blockContent *Group) {

					blockContent.Id("DataFlow::Node").Id("receiverNode").Semicolon().Line()
					blockContent.Id("DataFlow::Node").Id("headerNameNode").Semicolon().Line()
					blockContent.Id("DataFlow::Node").Id("headerValueNode").Semicolon().Line()

					blockContent.Id(funcModelsClassName).Call().Block(
						Id(setsHeaderDynamicKeyValue).Call(
							DontCare(),
							DontCare(),
							This(),
							Id("headerNameNode"),
							Id("headerValueNode"),
							Id("receiverNode"),
						),
					)

					blockContent.Override().Id("DataFlow::Node").Id("getName").Call().Block(
						Id("result").Eq().Id("headerNameNode"),
					)
					blockContent.Override().Id("DataFlow::Node").Id("getValue").Call().Block(
						Id("result").Eq().Id("headerValueNode"),
					)
					blockContent.Override().Id("HTTP::ResponseWriter").Id("getResponseWriter").Call().BlockFunc(
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
							Id(setsStaticContentType).Call(
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
							Id(setsDynamicContentType).Call(
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

									fn := x.GetFuncByQualifier(methodQual)
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

									fn := x.GetFuncByQualifier(methodQual)
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

									fn := x.GetFuncByQualifier(methodQual)
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

									fn := x.GetFuncByQualifier(methodQual)
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
	return x.CqlParamQualToCode("contentTypeSetterCall", "getArgument", qual)
}

func GetFuncQualifierCodeElements(qual *x.FuncQualifier) (x.FuncInterface, Code) {
	return x.CqlParamQualToCode("this", "getArgument", qual)
}
