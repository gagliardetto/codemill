package responsebody

import (
	"strings"

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

	{
		// Add imports:
		//impAdder.Import("DataFlow::PathGraph")
	}

	className := mdl.Name
	allPathVersions := mdl.ListAllPathVersions()

	{
		// Body + Static content type:
		funcModelsClassName := feparser.NewCodeQlName(className, "StaticContentType")
		tmp := DoGroup(func(tempFuncsModel *Group) {
			tempFuncsModel.Doc("Models HTTP ResponseBody where the content-type is static and non-modifiable.")
			tempFuncsModel.Private().Class().Id(funcModelsClassName).Extends().List(
				Id("HTTP::ResponseBody::Range"),
			).BlockFunc(
				func(blockContent *Group) {

					blockContent.String().Id("contentTypeString").Semicolon().Line()
					blockContent.Id("DataFlow::Node").Id("receiverNode").Semicolon().Line()

					blockContent.Id(funcModelsClassName).Call().BlockFunc(
						func(funcModelsSelfMethodGroup *Group) {
							funcModelsSelfMethodGroup.Exists(
								List(
									String().Id("package"),
									String().Id("receiverName"),
								),
								Id(setsBodyAndStaticContentType).Call(
									Id("package"),
									Id("receiverName"),
									This(),
									Id("contentTypeString"),
									Id("receiverNode"),
								),
								nil,
							)
						})

					blockContent.Override().Id("string").Id("getAContentType").Call().BlockFunc(
						func(overrideBlockGroup *Group) {
							overrideBlockGroup.Id("result").Eq().Id("contentTypeString")
						})

					blockContent.Override().Id("HTTP::ResponseWriter").Id("getResponseWriter").Call().BlockFunc(
						func(overrideBlockGroup *Group) {
							overrideBlockGroup.Id("result").Dot("getANode").Call().Eq().Id("receiverNode")
						})
				})
		})
		pred := predicate_setsBody_Static_ContentType(allPathVersions, mdl)
		if pred != nil {
			rootModuleGroup.Add(tmp)
			rootModuleGroup.Add(pred)
		}
	}
	{
		// Body + Dynamic content type:
		funcModelsClassName := feparser.NewCodeQlName(className, "DynamicContentType")
		tmp := DoGroup(func(tempFuncsModel *Group) {
			tempFuncsModel.Doc("Models HTTP ResponseBody where the content-type can be dynamically set by the caller.")
			tempFuncsModel.Private().Class().Id(funcModelsClassName).Extends().List(
				Id("HTTP::ResponseBody::Range"),
			).BlockFunc(
				func(blockContent *Group) {

					blockContent.Id("DataFlow::Node").Id("contentTypeNode").Semicolon().Line()
					blockContent.Id("DataFlow::Node").Id("receiverNode").Semicolon().Line()

					blockContent.Id(funcModelsClassName).Call().BlockFunc(
						func(funcModelsSelfMethodGroup *Group) {
							funcModelsSelfMethodGroup.Exists(
								List(
									String().Id("package"),
									String().Id("receiverName"),
								),
								Id(setsBodyAndDynamicContentType).Call(
									Id("package"),
									Id("receiverName"),
									This(),
									Id("contentTypeNode"),
									Id("receiverNode"),
								),
								nil,
							)
						})

					blockContent.Override().Id("DataFlow::Node").Id("getAContentTypeNode").Call().BlockFunc(
						func(overrideBlockGroup *Group) {
							overrideBlockGroup.Id("result").Eq().Id("contentTypeNode")
						})

					blockContent.Override().Id("HTTP::ResponseWriter").Id("getResponseWriter").Call().BlockFunc(
						func(overrideBlockGroup *Group) {
							overrideBlockGroup.Id("result").Dot("getANode").Call().Eq().Id("receiverNode")
						})
				})
		})
		pred := predicate_setsBody_Dynamic_ContentType(allPathVersions, mdl)
		if pred != nil {
			rootModuleGroup.Add(tmp)
			rootModuleGroup.Add(pred)
		}
	}
	{
		// Just Body with No content type:
		funcModelsClassName := feparser.NewCodeQlName(className, "NoContentType")
		tmp := DoGroup(func(tempFuncsModel *Group) {
			tempFuncsModel.Doc("Models HTTP ResponseBody where the content-type is set by another call.")
			tempFuncsModel.Private().Class().Id(funcModelsClassName).Extends().List(
				Id("HTTP::ResponseBody::Range"),
			).BlockFunc(
				func(blockContent *Group) {

					blockContent.Id("DataFlow::Node").Id("receiverNode").Semicolon().Line()

					blockContent.Id(funcModelsClassName).Call().BlockFunc(
						func(funcModelsSelfMethodGroup *Group) {
							funcModelsSelfMethodGroup.Exists(
								List(
									String().Id("package"),
									String().Id("receiverName"),
								),
								Id(setsBody).Call(
									Id("package"),
									Id("receiverName"),
									Id("receiverNode"),
									This(),
								),
								nil,
							)
						})

					blockContent.Override().Id("HTTP::ResponseWriter").Id("getResponseWriter").Call().BlockFunc(
						func(overrideBlockGroup *Group) {
							overrideBlockGroup.Id("result").Dot("getANode").Call().Eq().Id("receiverNode")
						})
				})
		})
		pred := predicate_setsBody(allPathVersions, mdl)
		if pred != nil {
			rootModuleGroup.Add(tmp)
			rootModuleGroup.Add(pred)
		}
	}
	{
		// TODO: move to http/headerwrite
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
		}
		{
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
	}

	return nil
}

// Predicate names:
const (
	setsBodyAndStaticContentType  = "setsBodyAndStaticContentType"
	setsBodyAndDynamicContentType = "setsBodyAndDynamicContentType"
	setsBody                      = "setsBody"

	setsStaticContentType  = "setsStaticContentType"
	setsDynamicContentType = "setsDynamicContentType"
)

func predicate_setsBody_Static_ContentType(allPathVersions []string, mdl *x.XModel) Code {
	predicate :=
		Comment("Holds for a call that sets the body; the content-type is implicitly set.").
			Private().Predicate().Id(setsBodyAndStaticContentType).Call(
			List(
				String().Id("package"),
				String().Id("receiverName"),
				Id("DataFlow::Node").Id("bodyNode"),
				String().Id("contentTypeString"),
				Id("DataFlow::Node").Id("receiverNode"),
			),
		)

	addedCount := 0
	predicate.BlockFunc(func(predicateBlock *Group) {
		{
			pc := cql_MethodBodyWithCtFromFuncName(mdl, allPathVersions)
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

func predicate_setsBody_Dynamic_ContentType(allPathVersions []string, mdl *x.XModel) Code {
	predicate :=
		Comment("Holds for a call that sets the body; the content-type is a parameter.").
			Comment("Both body and content-type are parameters in the same func call.").
			Private().Predicate().Id(setsBodyAndDynamicContentType).Call(
			List(
				String().Id("package"),
				String().Id("receiverName"),
				Id("DataFlow::Node").Id("bodyNode"),
				Id("DataFlow::Node").Id("contentTypeNode"),
				Id("DataFlow::Node").Id("receiverNode"),
			),
		)

	addedCount := 0
	predicate.BlockFunc(func(predicateBlock *Group) {
		{
			pc := cql_MethodBodyWithCt(mdl, allPathVersions)
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

func predicate_setsBody(allPathVersions []string, mdl *x.XModel) Code {
	predicate := Comment("Holds for a call that sets the body. The content-type is not defined.").
		Private().Predicate().Id(setsBody).Call(
		List(
			String().Id("package"),
			String().Id("receiverName"),
			Id("DataFlow::Node").Id("receiverNode"),
			Id("DataFlow::Node").Id("bodyNode"),
		),
	)

	addedCount := 0
	predicate.BlockFunc(func(predicateBlock *Group) {
		{
			pc := cql_MethodBody(mdl, allPathVersions)
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

func GetBodySetterFuncQualifierCodeElements(qual *x.FuncQualifier) (x.FuncInterface, Code) {
	fn := GetFunc(qual)

	parameterIndexes := x.MustPosToRelativeParamIndexes(fn, qual.Pos)
	code := x.GenCqlParamQual("bodySetterCall", "getArgument", fn, parameterIndexes)

	return fn, code
}
func GetContentTypeSetterFuncQualifierCodeElements(qual *x.FuncQualifier) (x.FuncInterface, Code) {
	fn := GetFunc(qual)

	parameterIndexes := x.MustPosToRelativeParamIndexes(fn, qual.Pos)
	code := x.GenCqlParamQual("contentTypeSetterCall", "getArgument", fn, parameterIndexes)

	return fn, code
}

func guessContentTypeFromFuncName(name string) string {
	name = strings.ToLower(name)

	if strings.Contains(name, "jsonp") {
		return "application/javascript"
	}
	if strings.Contains(name, "json") {
		return "application/json"
	}
	if strings.Contains(name, "xml") {
		return "text/xml"
	}
	if strings.Contains(name, "yaml") || strings.Contains(name, "yml") {
		return "application/x-yaml"
	}
	if strings.Contains(name, "html") {
		return "text/html"
	}
	if strings.Contains(name, "string") || strings.Contains(name, "text") {
		return "text/plain"
	}
	if strings.Contains(name, "error") {
		// NOTE: this might be not correct.
		return "text/plain"
	}
	return "TODO"
}

// cql_MethodBodyWithCtFromFuncName generates model statements for MethodBodyWithCtFromFuncName
func cql_MethodBodyWithCtFromFuncName(mdl *x.XModel, pathVersions []string) []Code {

	// Assuming the validation has already been done:
	mtdBodyWithCtFromFuncName := mdl.Methods.ByName(MethodBodyWithCtFromFuncName)
	if len(mtdBodyWithCtFromFuncName.Selectors) == 0 {
		Infof("No selectors found for %q method.", mtdBodyWithCtFromFuncName.Name)
		return nil
	}

	b2fe, b2tm, b2itm, err := x.GroupFuncSelectors(mtdBodyWithCtFromFuncName)
	if err != nil {
		Fatalf("Error while GroupFuncSelectors: %s", err)
	}

	pathCodez := make([]Code, 0)
	// Functions:
	{
		// TODO:
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
				Id("DataFlow::CallNode").Id("bodySetterCall"),
			),
			DoGroup(func(st *Group) {
				st.Id("met").Dot("hasQualifiedName").Call(
					Id("package"),
					Id("receiverName"),
					Id("methodName"),
				)
				st.And()
				st.Id("bodySetterCall").Eq().Id("met").Dot("getACall").Call()
				st.And()
				st.Id("receiverNode").Eq().Id("bodySetterCall").Dot("getReceiver").Call()
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
												_, code := GetBodySetterFuncQualifierCodeElements(methodQual)
												par.Id("bodyNode").Eq().Add(code)
											}

											par.And()

											par.Id("contentTypeString").Eq().Lit(guessContentTypeFromFuncName(fn.GetFunc().Name))
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
				Id("DataFlow::CallNode").Id("bodySetterCall"),
			),
			DoGroup(func(st *Group) {
				st.Id("met").Dot("implements").Call(
					Id("package"),
					Id("interfaceName"),
					Id("methodName"),
				)
				st.And()
				st.Id("bodySetterCall").Eq().Id("met").Dot("getACall").Call()
				st.And()
				st.Id("receiverNode").Eq().Id("bodySetterCall").Dot("getReceiver").Call()
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
												_, code := GetBodySetterFuncQualifierCodeElements(methodQual)
												par.Id("bodyNode").Eq().Add(code)
											}

											par.And()

											par.Id("contentTypeString").Eq().Lit(guessContentTypeFromFuncName(fn.GetFunc().Name))
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

// cql_MethodBodyWithCt generates model statements combining MethodBodyWithCtIsBody and MethodBodyWithCtIsCt.
func cql_MethodBodyWithCt(mdl *x.XModel, pathVersions []string) []Code {

	mtdBodyWithCtIsBody := mdl.Methods.ByName(MethodBodyWithCtIsBody)
	if len(mtdBodyWithCtIsBody.Selectors) == 0 {
		Infof("No selectors found for %q method.", mtdBodyWithCtIsBody.Name)
		return nil
	}

	b2feBody, b2tmBody, b2itmBody, err := x.GroupFuncSelectors(mtdBodyWithCtIsBody)
	if err != nil {
		Fatalf("Error while GroupFuncSelectors: %s", err)
	}
	//
	mtdBodyWithCtIsCt := mdl.Methods.ByName(MethodBodyWithCtIsCt)
	if len(mtdBodyWithCtIsCt.Selectors) == 0 {
		Infof("No selectors found for %q method.", mtdBodyWithCtIsCt.Name)
		return nil
	}

	_, b2tmCt, b2itmCt, err := x.GroupFuncSelectors(mtdBodyWithCtIsCt)
	if err != nil {
		Fatalf("Error while GroupFuncSelectors: %s", err)
	}

	pathCodez := make([]Code, 0)
	// Functions:
	{
		for _, pathVersion := range pathVersions {
			_, ok := b2feBody[pathVersion]
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
				Id("DataFlow::CallNode").Id("bodySetterCall"),
			),
			DoGroup(func(st *Group) {
				st.Id("met").Dot("hasQualifiedName").Call(
					Id("package"),
					Id("receiverName"),
					Id("methodName"),
				)
				st.And()
				st.Id("bodySetterCall").Eq().Id("met").Dot("getACall").Call()
				st.And()
				st.Id("receiverNode").Eq().Id("bodySetterCall").Dot("getReceiver").Call()
			}),
			DoGroup(func(exists3 *Group) {
				for _, pathVersion := range pathVersions {
					pathVersionAddedCount := 0

					tempForPathVersion := make([]Code, 0)
					b2tmBody.IterValid(pathVersion,
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
												_, code := GetBodySetterFuncQualifierCodeElements(methodQual)
												par.Id("bodyNode").Eq().Add(code)
											}
											par.And()
											{
												ctQual := b2tmCt[pathVersion][receiverTypeID].ByBasicQualifier(methodQual.BasicQualifier)
												_, code := GetBodySetterFuncQualifierCodeElements(ctQual)
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
				Id("DataFlow::CallNode").Id("bodySetterCall"),
			),
			DoGroup(func(st *Group) {
				st.Id("met").Dot("implements").Call(
					Id("package"),
					Id("interfaceName"),
					Id("methodName"),
				)
				st.And()
				st.Id("bodySetterCall").Eq().Id("met").Dot("getACall").Call()
				st.And()
				st.Id("receiverNode").Eq().Id("bodySetterCall").Dot("getReceiver").Call()
			}),
			DoGroup(func(exists3 *Group) {
				for _, pathVersion := range pathVersions {
					pathVersionAddedCount := 0

					tempForPathVersion := make([]Code, 0)
					b2itmBody.IterValid(pathVersion,
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
												_, code := GetBodySetterFuncQualifierCodeElements(methodQual)
												par.Id("bodyNode").Eq().Add(code)
											}

											par.And()

											{
												ctQual := b2itmCt[pathVersion][receiverTypeID].ByBasicQualifier(methodQual.BasicQualifier)
												_, code := GetBodySetterFuncQualifierCodeElements(ctQual)
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

func cql_MethodBody(mdl *x.XModel, pathVersions []string) []Code {

	// Assuming the validation has already been done:
	mtdBody := mdl.Methods.ByName(MethodBody)
	if len(mtdBody.Selectors) == 0 {
		Infof("No selectors found for %q method.", mtdBody.Name)
		return nil
	}

	b2fe, b2tm, b2itm, err := x.GroupFuncSelectors(mtdBody)
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
				Id("DataFlow::CallNode").Id("bodySetterCall"),
			),
			DoGroup(func(st *Group) {
				st.Id("met").Dot("hasQualifiedName").Call(
					Id("package"),
					Id("receiverName"),
					Id("methodName"),
				)
				st.And()
				st.Id("bodySetterCall").Eq().Id("met").Dot("getACall").Call()
				st.And()
				st.Id("receiverNode").Eq().Id("bodySetterCall").Dot("getReceiver").Call()
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
												_, code := GetBodySetterFuncQualifierCodeElements(methodQual)
												par.Id("bodyNode").Eq().Add(code)
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
				Id("DataFlow::CallNode").Id("bodySetterCall"),
			),
			DoGroup(func(st *Group) {
				st.Id("met").Dot("implements").Call(
					Id("package"),
					Id("interfaceName"),
					Id("methodName"),
				)
				st.And()
				st.Id("bodySetterCall").Eq().Id("met").Dot("getACall").Call()
				st.And()
				st.Id("receiverNode").Eq().Id("bodySetterCall").Dot("getReceiver").Call()
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
												_, code := GetBodySetterFuncQualifierCodeElements(methodQual)
												par.Id("bodyNode").Eq().Add(code)
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
												par.Id("contentTypeString").Eq().Lit(guessContentTypeFromFuncName(fn.GetFunc().Name))
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
												par.Id("contentTypeString").Eq().Lit(guessContentTypeFromFuncName(fn.GetFunc().Name))
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
