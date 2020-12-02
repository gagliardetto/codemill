package untrustedflowsource

import (
	"github.com/gagliardetto/codemill/x"
	. "github.com/gagliardetto/cqlgen/jen"
	"github.com/gagliardetto/feparser"
	. "github.com/gagliardetto/utilz"
)

func (han *Handler) GenerateCodeQL(mdl *x.XModel, moduleGroup *Group) error {
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
