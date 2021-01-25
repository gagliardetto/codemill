package x

import (
	. "github.com/gagliardetto/cqlgen/jen"
	"github.com/gagliardetto/feparser"
	. "github.com/gagliardetto/utilz"
)

func PosToRelativeIndexes(fn FuncInterface, positions []bool) (receiver bool, parameterIndexes []int, resultIndexes []int) {
InpLoop:
	for inpPos, ok := range positions {
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
				receiver = true
			}
		case feparser.ElementParameter:
			{
				parameterIndexes = append(parameterIndexes,
					inpRelIndex,
				)
			}
		case feparser.ElementResult:
			{
				resultIndexes = append(resultIndexes,
					inpRelIndex,
				)
			}
		default:
			panic(Sf("Unknown type: %q", inpElTyp))
		}
	}

	return
}

func GenFunctionInputOutput(idName string, fn FuncInterface, receiver bool, parameterIndexes []int, resultIndexes []int) []Code {
	codeElements := make([]Code, 0)

	if receiver {
		codeElements = append(codeElements,
			Id(idName).Dot("isReceiver").Call(),
		)
	}

	if len(parameterIndexes) > 0 {
		codeElements = append(codeElements,
			GenCqlParamQual(idName, "isParameter", fn, parameterIndexes),
		)
	}

	_, _, lenResults := fn.Lengths()
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
func GenCqlParamQual(idName string, dotName string, fn FuncInterface, parameterIndexes []int) (res Code) {

	_, lenParams, _ := fn.Lengths()

	if len(parameterIndexes) > 0 {
		// If all parameters are selected,
		// and there is more than one possible parameters,
		// then use a `_`:
		if lenParams == len(parameterIndexes) && lenParams > 1 {
			res = Id(idName).Dot(dotName).Call(DontCare())
		} else {
			// If multiple parameters are selected (but not all)
			// then use a set, or just the index.
			// If there is only one possible parameter and it is selected,
			// then `isParameter(0)` is used.
			res = Id(idName).Dot(dotName).Call(
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
			)
		}
	}

	return
}
