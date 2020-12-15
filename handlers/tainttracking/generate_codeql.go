package tainttracking

import (
	"sort"

	"github.com/gagliardetto/codemill/x"
	. "github.com/gagliardetto/cqlgen/jen"
	"github.com/gagliardetto/feparser"
	. "github.com/gagliardetto/utilz"
)

func (han *Handler) GenerateCodeQL(impAdder x.ImportAdder, mdl *x.XModel, rootModuleGroup *Group) error {
	Sfln(
		"%s: Generating grouped codeql code for model %q",
		Kind,
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

	{
		// Add imports:
		impAdder.Import("DataFlow::PathGraph")
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
	taintTrackingModule := Private().Module().Id(feparser.FormatCodeQlName(className))

	taintTrackingModule.BlockFunc(func(ttModuleGroup *Group) {

		for _, pathVersion := range allPathVersions {

			ttModuleGroup.Commentf("Provides classes modeling security-relevant aspects of the %s package.", pathVersion)
			ttModuleGroup.Private().Module().Id(feparser.FormatCodeQlName(pathVersion)).BlockFunc(func(thisModuleGroup *Group) {

				b2fe, b2tm, b2itm, err := x.GroupFuncSelectors(self)
				if err != nil {
					Fatalf("Error while GroupFuncSelectors: %s", err)
				}
				_, _ = b2tm, b2itm
				cont, ok := b2fe[pathVersion]
				if ok {
					thisModuleGroup.Private().Class().Id("FunctionModels").Extends().Qual("TaintTracking", "FunctionModel").BlockFunc(
						func(funcModelsClassGroup *Group) {
							funcModelsClassGroup.Id("FunctionInput").Id("inp").Semicolon().Line()
							funcModelsClassGroup.Id("FunctionOutput").Id("out").Semicolon().Line()

							funcModelsClassGroup.Id("FunctionModels").Call().BlockFunc(
								func(funcModelsSelfMethodGroup *Group) {
									{

										code := DoGroup(
											func(groupCase *Group) {

												for i, funcQual := range cont {
													if i > 0 {
														groupCase.Or()
													}
													{
														if !funcQual.Flows.Enabled {
															continue
														}

														fn, codeElements := GetFuncQualifierCodeElements(funcQual)
														thing := fn.(*feparser.FEFunc)
														groupCase.Comment("Function: " + thing.Signature)
														groupCase.Id("hasQualifiedName").Call(Lit(funcQual.Path), Lit(thing.Name)).
															And().
															Parens(
																Join(
																	Or(),
																	codeElements...,
																),
															)
													}

												}
											})
										funcModelsSelfMethodGroup.
											Comment("Taint-tracking through functions.").
											Line().
											Add(code)
									}
								})

						})
				}

			})

		}
	})

	rootModuleGroup.Add(taintTrackingModule)
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

			outCodeElements := make([]Code, 0)
			outElTyp, _, outRelIndex, err := fn.GetRelativeElement(outPos)
			if err != nil {
				Fatalf("Error while GetRelativeElement: %s", err)
			}

			switch outElTyp {
			case feparser.ElementReceiver:
				{
					outCodeElements = append(outCodeElements,
						Id("outp").Dot("isReceiver").Call(),
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

		inpCodeElements = append(inpCodeElements, genStuff("inp", fn, inpParameterIndexes, inpResultIndexes)...)
		outCodeElements = append(outCodeElements, genStuff("out", fn, outParameterIndexes, outResultIndexes)...)

		codeElements = append(codeElements,
			Parens(
				Join(
					Or(),
					inpCodeElements...,
				),
			).And().Add(
				Join(
					Or(),
					outCodeElements...,
				)),
		)
	}

	return fn, codeElements
}

func genStuff(idName string, fn x.FuncInterface, parameterIndexes []int, resultIndexes []int) []Code {
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
