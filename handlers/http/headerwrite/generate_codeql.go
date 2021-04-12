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
