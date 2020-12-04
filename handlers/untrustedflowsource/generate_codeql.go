package untrustedflowsource

import (
	"fmt"

	"github.com/gagliardetto/codemill/x"
	. "github.com/gagliardetto/cqlgen/jen"
	"github.com/gagliardetto/feparser"
	. "github.com/gagliardetto/utilz"
)

func (han *Handler) GenerateCodeQL(mdl *x.XModel, moduleGroup *Group) error {
	return han.GenerateGroupedCodeQL(mdl, moduleGroup)
	// TODO
	Sfln(
		"Generating codeql code for model %q",
		mdl.Name,
	)
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

	className := mdl.Name

	moduleGroup.Doc("Doc about class")
	moduleGroup.Private().Class().Id(className).Extends().List(Qual("UntrustedFlowSource", "Range")).
		BlockFunc(func(classGr *Group) {
			classGr.Id(className).Call().BlockFunc(func(metGr *Group) {
				for selectorIndex, selector := range self.Selectors {
					rawQual := selector.Qualifier

					//isLast := selectorIndex == len(self.Selectors)-1
					if selectorIndex > 0 {
						metGr.Or()
					}

					// TODO:
					// - Group qualifiers by PathVersion, and then by qualifier type.
					switch qual := rawQual.(type) {
					case *x.FuncQualifier:
						{
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
							parameterIndexes := make([]int, 0)
							resultIndexes := make([]int, 0)
						PosLoop:
							for pos, ok := range qual.Pos {
								if !ok {
									continue PosLoop
								}

								elTyp, _, relIndex, err := fn.GetRelativeElement(pos)
								if err != nil {
									Fatalf("Error while GetRelativeElement: %s", err)
								}

								switch elTyp {
								case feparser.ElementReceiver:
									{
										codeElements = append(codeElements,
											This().Eq().Id("call").Dot("getReceiver").Call(),
										)
									}
								case feparser.ElementParameter:
									{
										parameterIndexes = append(parameterIndexes,
											relIndex,
										)
									}
								case feparser.ElementResult:
									{
										resultIndexes = append(resultIndexes,
											relIndex,
										)
									}
								default:
									panic(Sf("Unknown type: %q", elTyp))
								}
							}

							if len(parameterIndexes) > 0 {
								codeElements = append(codeElements,
									This().Eq().Qual("FunctionOutput", "parameter").Call(IntsToSetOrLit(parameterIndexes...)).Dot("getExitNode").Call(Id("call")),
								)
							}
							if len(resultIndexes) > 0 {
								codeElements = append(codeElements,
									This().Eq().Id("call").Dot("getResult").Call(IntsToSetOrLit(resultIndexes...)),
								)
							}

							switch thing := fn.(type) {
							case *feparser.FEFunc:
								{
									metGr.Comment("Package: " + qual.PathVersionClean())
									metGr.Comment("Function: " + thing.Signature)
									metGr.Exists(
										List(
											Id("Function").Id("fn"),
											Qual("DataFlow", "CallNode").Id("call"),
										),
										DoGroup(func(st *Group) {
											st.Id("fn").Dot("hasQualifiedName").Call(Lit(qual.Path), Lit(thing.Name))
										}),
										DoGroup(func(st *Group) {
											//st.Commentf("The source is the %s:",)

											st.Id("call").Eq().Id("fn").Dot("getACall").Call().
												And().
												Parens(
													Join(
														Or(),
														codeElements...,
													),
												)
										}),
									)
								}
							case *feparser.FETypeMethod:
								{
									// TODO: group methods per receiver.
									metGr.Comment("Package: " + qual.PathVersionClean())
									metGr.Commentf("Receiver: %s", thing.Receiver.TypeString)
									metGr.Exists(
										List(
											Qual("DataFlow", "MethodCallNode").Id("call"),
											String().Id("methodName"),
										),
										DoGroup(func(st *Group) {
											st.Id("call").Dot("getTarget").Call().Dot("hasQualifiedName").Call(
												Lit(qual.Path),
												Lit(thing.Receiver.TypeName),
												Id("methodName"),
											)
											st.And()
											st.ParensFunc(
												func(par *Group) {
													par.Commentf("Method: %s", thing.Func.Signature)
													par.Id("methodName").Eq().Lit(thing.Func.Name)
													par.And()
													par.Parens(
														Join(
															Or(),
															codeElements...,
														),
													)

													//par.Or()

												},
											)
										}),
										nil,
									)
								}
							case *feparser.FEInterfaceMethod:
								{
									// TODO: group methods per receiver.
									metGr.Comment("Package: " + qual.PathVersionClean())
									metGr.Commentf("Interface: %s", thing.Receiver.TypeString)
									metGr.Exists(
										List(
											Qual("DataFlow", "MethodCallNode").Id("call"),
											String().Id("methodName"),
										),
										DoGroup(func(st *Group) {
											// TODO: the only difference here is "implements" instead of hasQualifiedName.
											st.Id("call").Dot("getTarget").Call().Dot("implements").Call(
												Lit(qual.Path),
												Lit(thing.Receiver.TypeName),
												Id("methodName"),
											)
											st.And()
											st.ParensFunc(
												func(par *Group) {
													par.Commentf("Method: %s", thing.Func.Signature)
													par.Id("methodName").Eq().Lit(thing.Func.Name)
													par.And()
													par.Parens(
														Join(
															Or(),
															codeElements...,
														),
													)

													//par.Or()

												},
											)
										}),
										nil,
									)
								}
							default:
								panic(Sf("Unknown type: %T", fn))
							}

						}
					case *x.StructQualifier:
						{
							source := x.GetCachedSource(qual.Path, qual.Version)
							if source == nil {
								Fatalf("Source not found: %s@%s", qual.Path, qual.Version)
							}
							// Make sure that the struct exist:
							str := x.FindStructByID(source, qual.ID)
							if str == nil {
								Fatalf("Struct not found: %q", qual.ID)
							}

							fieldNames := make([]string, 0)
							for fieldName := range qual.Fields {
								//fld := x.FindFieldByName(str, fieldName)
								//if fld == nil {
								//	Fatalf("Field not found: %q", fieldName)
								//}
								// TODO: add a comment on the type for each field?
								fieldNames = append(fieldNames, fieldName)
							}

							metGr.Comment("Package: " + qual.PathVersionClean())
							metGr.Comment("Struct: " + str.TypeName)
							metGr.Exists(
								List(
									Qual("DataFlow", "Field").Id("fld"),
									String().Id("fieldName"),
								),
								DoGroup(func(st *Group) {
									st.Id("fld").Dot("hasQualifiedName").Call(Lit(qual.Path), Lit(str.TypeName), Id("fieldName"))

									st.And()

									st.Id("fieldName").In().Add(StringsToSet(fieldNames...))
									st.And()
									st.This().Eq().Id("fld").Dot("getARead").Call()
								}),
								nil,
							)
						}
					case *x.TypeQualifier:
						{
							source := x.GetCachedSource(qual.Path, qual.Version)
							if source == nil {
								Fatalf("Source not found: %s@%s", qual.Path, qual.Version)
							}
							// Find the type:
							typ := x.FindTypeByID(source, qual.ID)
							if typ == nil {
								Fatalf("Type not found: %q", qual.ID)
							}

							metGr.Comment("Package: " + qual.PathVersionClean())
							metGr.Commentf("Type: %s", typ.TypeName)
							metGr.Exists(
								List(
									Qual("DataFlow", "ReadNode").Id("read"),
									Id("ValueEntity").Id("v"),
								),
								DoGroup(func(st *Group) {
									st.Id("read").Dot("reads").Call(Id("v"))
									st.And()
									st.Id("v").Dot("getType").Call().Dot("hasQualifiedName").Call(Lit(qual.Path), Lit(typ.TypeName))
								}),
								DoGroup(func(st *Group) {
									st.This().Eq().Id("read")
								}),
							)
						}
					default:
						panic(Sf("Unknown type: %T", rawQual))
					}
				}
			})
		})

	return nil
}

// Func selectors:
type (
	// For each PathVersionClean, there is an array of FEFunc.
	BasicToFEFuncs map[string][]*x.FuncQualifier

	// For each PathVersionClean, there is a map of TypeIDs; for each TypeID, there is an array of methods.
	BasicToTypeIDToMethods map[string]map[string][]*x.FuncQualifier

	// For each PathVersionClean, there is a map of InterfaceIDs (TypeID); for each TypeID, there is an array of methods.
	BasicToInterfaceIDToMethods map[string]map[string][]*x.FuncQualifier
)

// Struct selectors:
type (
	// For each PathVersionClean, there is a map of StructIDs (TypeID); for each TypeID, there is an array of fields.
	BasicToStructIDToFields map[string][]*x.StructQualifier
)

// Type selectors:
type (
	// For each PathVersionClean, there is an array of types.
	BasicToTypes map[string][]*x.TypeQualifier
)

func GroupFuncSelectors(mtd *x.XMethod) (b2fe BasicToFEFuncs, b2tm BasicToTypeIDToMethods, b2itm BasicToInterfaceIDToMethods, err error) {

	b2fe = make(BasicToFEFuncs)
	b2tm = make(BasicToTypeIDToMethods)
	b2itm = make(BasicToInterfaceIDToMethods)

	for _, sel := range mtd.Selectors {
		qual := sel.GetFuncQualifier()
		if qual == nil {
			continue
		}

		source := x.GetCachedSource(qual.Path, qual.Version)
		if source == nil {
			return nil, nil, nil, fmt.Errorf("Source not found: %s@%s", qual.Path, qual.Version)
		}
		// Find the func/type-method/interface-method:
		fn := x.FindFuncByID(source, qual.ID)
		if fn == nil {
			return nil, nil, nil, fmt.Errorf("Func not found: %q", qual.ID)
		}
		basic := *(sel.GetBasicQualifier())
		pathVersion := basic.PathVersionClean()

		switch thing := fn.(type) {
		case *feparser.FEFunc:
			{
				if _, ok := b2fe[pathVersion]; !ok {
					b2fe[pathVersion] = make([]*x.FuncQualifier, 0)
				}
				b2fe[pathVersion] = append(b2fe[pathVersion], qual)
			}
		case *feparser.FETypeMethod:
			{
				if _, ok := b2tm[pathVersion]; !ok {
					b2tm[pathVersion] = make(map[string][]*x.FuncQualifier)
				}
				typeID := thing.Receiver.ID
				if _, ok := b2tm[pathVersion][typeID]; !ok {
					b2tm[pathVersion][typeID] = make([]*x.FuncQualifier, 0)
				}
				b2tm[pathVersion][typeID] = append(b2tm[pathVersion][typeID], qual)
			}
		case *feparser.FEInterfaceMethod:
			{
				if _, ok := b2itm[pathVersion]; !ok {
					b2itm[pathVersion] = make(map[string][]*x.FuncQualifier)
				}
				interfaceID := thing.Receiver.ID
				if _, ok := b2itm[pathVersion][interfaceID]; !ok {
					b2itm[pathVersion][interfaceID] = make([]*x.FuncQualifier, 0)
				}
				b2itm[pathVersion][interfaceID] = append(b2itm[pathVersion][interfaceID], qual)
			}
		default:
			panic(Sf("Unknown type: %T", fn))
		}

	}

	return
}
func GroupStructSelectors(mtd *x.XMethod) (b2st BasicToStructIDToFields, err error) {

	b2st = make(BasicToStructIDToFields)

	for _, sel := range mtd.Selectors {
		qual := sel.GetStructQualifier()
		if qual == nil {
			continue
		}

		{ // TODO: is this useful?
			source := x.GetCachedSource(qual.Path, qual.Version)
			if source == nil {
				return nil, fmt.Errorf("Source not found: %s@%s", qual.Path, qual.Version)
			}
			// Find the struct:
			st := x.FindStructByID(source, qual.ID)
			if st == nil {
				return nil, fmt.Errorf("Struct not found: %q", qual.ID)
			}
		}
		basic := *(sel.GetBasicQualifier())
		pathVersion := basic.PathVersionClean()

		if _, ok := b2st[pathVersion]; !ok {
			b2st[pathVersion] = make([]*x.StructQualifier, 0)
		}

		b2st[pathVersion] = append(b2st[pathVersion], qual)

	}

	return
}
func GroupTypeSelectors(mtd *x.XMethod) (b2typ BasicToTypes, err error) {

	b2typ = make(BasicToTypes)

	for _, sel := range mtd.Selectors {
		qual := sel.GetTypeQualifier()
		if qual == nil {
			continue
		}

		source := x.GetCachedSource(qual.Path, qual.Version)
		if source == nil {
			return nil, fmt.Errorf("Source not found: %s@%s", qual.Path, qual.Version)
		}
		// Find the type:
		typ := x.FindTypeByID(source, qual.ID)
		if typ == nil {
			return nil, fmt.Errorf("Type not found: %q", qual.ID)
		}
		basic := *(sel.GetBasicQualifier())
		pathVersion := basic.PathVersionClean()

		if _, ok := b2typ[pathVersion]; !ok {
			b2typ[pathVersion] = make([]*x.TypeQualifier, 0)
		}

		b2typ[pathVersion] = append(b2typ[pathVersion], qual)

	}

	return
}
func (han *Handler) GenerateGroupedCodeQL(mdl *x.XModel, moduleGroup *Group) error {
	// TODO
	Sfln(
		"Generating grouped codeql code for model %q",
		mdl.Name,
	)
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

	className := mdl.Name

	moduleGroup.Doc("Doc about class")
	moduleGroup.Private().Class().Id(className).Extends().List(Qual("UntrustedFlowSource", "Range")).
		BlockFunc(func(classGr *Group) {
			classGr.Id(className).Call().BlockFunc(func(metGr *Group) {
				b2fe, b2tm, b2itm, err := GroupFuncSelectors(self)
				if err != nil {
					Fatalf("Error while GroupFuncSelectors: %s", err)
				}

				{
					index := 0
					for pathVersion, cont := range b2fe {
						if index > 0 {
							metGr.Or()
						}
						index++

						metGr.Comment("Functions of package: " + pathVersion)
						metGr.Exists(
							List(
								Id("Function").Id("fn"),
								Qual("DataFlow", "CallNode").Id("call"),
							),
							DoGroup(func(st *Group) {
								for i, funcQual := range cont {
									if i > 0 {
										st.Or()
									}

									fn, codeElements := getFuncQualifierCodeElements(funcQual)
									thing := fn.(*feparser.FEFunc)
									st.Comment("Function: " + thing.Signature)
									st.Id("fn").Dot("hasQualifiedName").Call(Lit(funcQual.Path), Lit(thing.Name)).
										And().
										Parens(
											Join(
												Or(),
												codeElements...,
											),
										)
								}
							}),
							DoGroup(func(st *Group) {
								st.Id("call").Eq().Id("fn").Dot("getACall").Call()
							}),
						)
					}
				}

				if len(b2fe) > 0 && len(b2tm) > 0 {
					metGr.Or()
				}

				{
					index := 0
					for pathVersion, cont := range b2tm {
						if index > 0 {
							metGr.Or()
						}
						index++

						metGr.Comment("Methods on types of package: " + pathVersion)
						metGr.Exists(
							List(
								Qual("DataFlow", "MethodCallNode").Id("call"),
								String().Id("methodName"),
							),
							DoGroup(func(st *Group) {
								typeIndex := 0
								for receiverTypeID, methodQualifiers := range cont {
									if typeIndex > 0 {
										st.Or()
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
										Fatalf("Type not found: %q", qual.ID)
									}

									st.Commentf("Receiver: %s", typ.TypeString)
									st.Id("call").Dot("getTarget").Call().Dot("hasQualifiedName").Call(
										Lit(methodQualifiers[0].Path),
										Lit(typ.TypeName),
										Id("methodName"),
									)
									st.And()

									st.ParensFunc(
										func(parMethods *Group) {
											for i, methodQual := range methodQualifiers {
												if i > 0 {
													parMethods.Or()
												}

												fn, codeElements := getFuncQualifierCodeElements(methodQual)
												thing := fn.(*feparser.FETypeMethod)

												parMethods.ParensFunc(
													func(par *Group) {

														par.Commentf("Method: %s", thing.Func.Signature)
														par.Id("methodName").Eq().Lit(thing.Func.Name)
														par.And()
														par.Parens(
															Join(
																Or(),
																codeElements...,
															),
														)

														//par.Or()

													},
												)

											}
										},
									)
								}
							}),
							nil,
						)
					}
				}

				if (len(b2fe) > 0 || len(b2tm) > 0) && len(b2itm) > 0 {
					metGr.Or()
				}

				{
					index := 0
					for pathVersion, cont := range b2itm {
						if index > 0 {
							metGr.Or()
						}
						index++

						metGr.Comment("Interfaces of package: " + pathVersion)
						metGr.Exists(
							List(
								Qual("DataFlow", "MethodCallNode").Id("call"),
								String().Id("methodName"),
							),
							DoGroup(func(st *Group) {
								typeIndex := 0
								for receiverTypeID, methodQualifiers := range cont {
									if typeIndex > 0 {
										st.Or()
									}
									typeIndex++

									qual := methodQualifiers[0]
									source := x.GetCachedSource(qual.Path, qual.Version)
									if source == nil {
										Fatalf("Source not found: %s@%s", qual.Path, qual.Version)
									}
									// Find interface type:
									typ := x.FindTypeByID(source, receiverTypeID)
									if typ == nil {
										Fatalf("Type not found: %q", qual.ID)
									}

									st.Commentf("Interface: %s", typ.TypeString)
									st.Id("call").Dot("getTarget").Call().Dot("implements").Call(
										Lit(methodQualifiers[0].Path),
										Lit(typ.TypeName),
										Id("methodName"),
									)
									st.And()

									st.ParensFunc(
										func(parMethods *Group) {
											for i, methodQual := range methodQualifiers {
												if i > 0 {
													parMethods.Or()
												}

												fn, codeElements := getFuncQualifierCodeElements(methodQual)
												thing := fn.(*feparser.FEInterfaceMethod)

												parMethods.ParensFunc(
													func(par *Group) {

														par.Commentf("Method: %s", thing.Func.Signature)
														par.Id("methodName").Eq().Lit(thing.Func.Name)
														par.And()
														par.Parens(
															Join(
																Or(),
																codeElements...,
															),
														)

														//par.Or()

													},
												)

											}
										},
									)
								}
							}),
							nil,
						)
					}
				}

				b2st, err := GroupStructSelectors(self)
				if err != nil {
					Fatalf("Error while GroupFuncSelectors: %s", err)
				}
				if (len(b2fe) > 0 || len(b2tm) > 0 || len(b2itm) > 0) && len(b2st) > 0 {
					metGr.Or()
				}
				{
					index := 0
					for pathVersion, structQualifiers := range b2st {
						if index > 0 {
							metGr.Or()
						}
						index++

						metGr.Comment("Structs of package: " + pathVersion)
						metGr.Exists(
							List(
								Qual("DataFlow", "Field").Id("fld"),
							),
							DoGroup(func(st *Group) {
								for qualIndex, qual := range structQualifiers {
									if qualIndex > 0 {
										st.Or()
									}
									source := x.GetCachedSource(qual.Path, qual.Version)
									if source == nil {
										Fatalf("Source not found: %s@%s", qual.Path, qual.Version)
									}
									// Make sure that the struct exist:
									str := x.FindStructByID(source, qual.ID)
									if str == nil {
										Fatalf("Struct not found: %q", qual.ID)
									}

									fieldNames := make([]string, 0)
									for fieldName := range qual.Fields {
										//fld := x.FindFieldByName(str, fieldName)
										//if fld == nil {
										//	Fatalf("Field not found: %q", fieldName)
										//}
										// TODO: add a comment on the type for each field?
										fieldNames = append(fieldNames, fieldName)
									}
									st.Comment("Struct: " + str.TypeName)
									st.Id("fld").Dot("hasQualifiedName").Call(Lit(qual.Path), Lit(str.TypeName), StringsToSetOrLit(fieldNames...))

								}

							}),
							This().Eq().Id("fld").Dot("getARead").Call(),
						)
					}
				}

			})
		})

	return nil
}
func getFuncQualifierCodeElements(qual *x.FuncQualifier) (x.FuncInterface, []Code) {

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
	parameterIndexes := make([]int, 0)
	resultIndexes := make([]int, 0)
PosLoop:
	for pos, ok := range qual.Pos {
		if !ok {
			continue PosLoop
		}

		elTyp, _, relIndex, err := fn.GetRelativeElement(pos)
		if err != nil {
			Fatalf("Error while GetRelativeElement: %s", err)
		}

		switch elTyp {
		case feparser.ElementReceiver:
			{
				codeElements = append(codeElements,
					This().Eq().Id("call").Dot("getReceiver").Call(),
				)
			}
		case feparser.ElementParameter:
			{
				parameterIndexes = append(parameterIndexes,
					relIndex,
				)
			}
		case feparser.ElementResult:
			{
				resultIndexes = append(resultIndexes,
					relIndex,
				)
			}
		default:
			panic(Sf("Unknown type: %q", elTyp))
		}
	}

	if len(parameterIndexes) > 0 {
		codeElements = append(codeElements,
			This().Eq().Qual("FunctionOutput", "parameter").Call(IntsToSetOrLit(parameterIndexes...)).Dot("getExitNode").Call(Id("call")),
		)
	}
	if len(resultIndexes) > 0 {
		codeElements = append(codeElements,
			This().Eq().Id("call").Dot("getResult").Call(IntsToSetOrLit(resultIndexes...)),
		)
	}

	return fn, codeElements
}
