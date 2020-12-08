package untrustedflowsource

import (
	"fmt"
	"go/types"
	"sort"
	"strings"

	. "github.com/dave/jennifer/jen"
	"github.com/gagliardetto/codebox/gogentools"
	"github.com/gagliardetto/codebox/scanner"
	"github.com/gagliardetto/codemill/x"
	"github.com/gagliardetto/feparser"
	"github.com/gagliardetto/golang-go/cmd/go/not-internal/search"
	. "github.com/gagliardetto/utilz"
)

func NewTestFile(includeBoilerplace bool) *File {
	file := NewFile("main")
	// Set a prefix to avoid collision between variable names and packages:
	file.PackagePrefix = "cql"
	// Add comment to file:
	file.HeaderComment("Code generated by https://github.com/gagliardetto/codemill. DO NOT EDIT.")

	if includeBoilerplace {
		{
			// main function:
			//file.Func().Id("main").Params().Block()
		}
		{
			// sink function:
			code := Func().
				Id("sink").
				//Params(Id("id").Int(), Id("v").Interface()).
				Params(Id("v").Op("...").Interface()).
				Block()
			file.Add(code.Line())
		}
		{
			// link function (Used in tests to transmit taint from param 0 into param 1):
			code := Func().
				Id("link").
				Params(Id("from").Interface(), Id("into").Interface()).
				Block()
			file.Add(code.Line())
		}
		{
			// source function returns a new tainted thing:
			code := Func().
				Id("source").
				Params(Id("id").Int()).
				Interface().
				Block(Return(Nil()))
			file.Add(code.Line())
		}
	}
	return file
}

var (
	IncludeCommentsInGeneratedGo bool
)

func (han *Handler) GenerateGo(dir string, mdl *x.XModel) error {
	// TODO
	Sfln(
		"Generating go code for model %q into %q dir",
		mdl.Name,
		dir,
	)

	{
		// Check if there are multiple versions of a same package:
		mods := mdl.ListModules()
		if x.HasMultiversion(mods) {
			Ln(RedBG("Has multiversion"))
		}
	}
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

	allPathVersions := func() []string {
		res := make([]string, 0)
		mods := mdl.ListModules()
		for _, mod := range mods {
			res = append(res, mod.PathVersionClean())
		}
		return res
	}()

	pathVersionToTypeNames := make(map[string][]string)
	pathVersionToFuncAndVarNames := make(map[string][]string)
	for _, pathVersion := range allPathVersions {
		file := NewTestFile(true)
		codez := make([]Code, 0)

		b2fe, b2tm, b2itm, err := x.GroupFuncSelectors(self)
		if err != nil {
			Fatalf("Error while GroupFuncSelectors: %s", err)
		}

		b2st, err := x.GroupStructSelectors(self)
		if err != nil {
			Fatalf("Error while GroupStructSelectors: %s", err)
		}

		b2typ, err := x.GroupTypeSelectors(self)
		if err != nil {
			Fatalf("Error while GroupTypeSelectors: %s", err)
		}

		{
			cont, ok := b2fe[pathVersion]
			if ok {
				code := BlockFunc(
					func(groupCase *Group) {

						for _, funcQual := range cont {
							fn := x.GetFuncQualifier(funcQual)
							thing := fn.(*feparser.FEFunc)

							gogentools.ImportPackage(file, thing.PkgPath, thing.PkgName)
							pathVersionToFuncAndVarNames[pathVersion] = append(pathVersionToFuncAndVarNames[pathVersion], thing.Name)

							groupCase.Comment(thing.Signature)
							_, codeElements := GoGetFuncQualifierCodeElements(file, funcQual)
							groupCase.Add(codeElements...)

						}
					})
				codez = append(codez,
					Comment("Untrusted flow sources from functions.").
						Line().
						Add(code),
				)
			}
		}
		{
			cont, ok := b2tm[pathVersion]
			if ok {
				codezTypeMethods := make([]Code, 0)
				keys := func(v map[string][]*x.FuncQualifier) []string {
					res := make([]string, 0)
					for key := range v {
						res = append(res, key)
					}
					sort.Strings(res)
					return res
				}(cont)
				for _, receiverTypeID := range keys {
					methodQualifiers := cont[receiverTypeID]
					if len(methodQualifiers) == 0 {
						continue
					}

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

					gogentools.ImportPackage(file, typ.PkgPath, typ.PkgName)
					pathVersionToTypeNames[pathVersion] = append(pathVersionToTypeNames[pathVersion], typ.TypeName)

					code := BlockFunc(
						func(groupCase *Group) {

							for _, methodQual := range methodQualifiers {
								fn := x.GetFuncQualifier(methodQual)
								thing := fn.(*feparser.FETypeMethod)

								groupCase.Comment(thing.Func.Signature)
								_, codeElements := GoGetFuncQualifierCodeElements(file, methodQual)
								groupCase.Add(codeElements...)

							}
						})
					codezTypeMethods = append(codezTypeMethods,
						Commentf("Untrusted flow sources from method calls on %s.", typ.QualifiedName).
							Line().
							Add(code),
					)
				}
				codez = append(codez,
					Comment("Untrusted flow sources from method calls.").
						Line().
						Block(codezTypeMethods...),
				)
			}
		}

		{
			cont, ok := b2itm[pathVersion]
			if ok {
				codezIfaceMethods := make([]Code, 0)
				keys := func(v map[string][]*x.FuncQualifier) []string {
					res := make([]string, 0)
					for key := range v {
						res = append(res, key)
					}
					sort.Strings(res)
					return res
				}(cont)
				for _, receiverTypeID := range keys {
					methodQualifiers := cont[receiverTypeID]
					if len(methodQualifiers) == 0 {
						continue
					}

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

					file := NewTestFile(true)
					gogentools.ImportPackage(file, typ.PkgPath, typ.PkgName)
					pathVersionToTypeNames[pathVersion] = append(pathVersionToTypeNames[pathVersion], typ.TypeName)

					code := BlockFunc(
						func(groupCase *Group) {

							for _, methodQual := range methodQualifiers {
								fn := x.GetFuncQualifier(methodQual)
								thing := fn.(*feparser.FEInterfaceMethod)

								groupCase.Comment(thing.Func.Signature)
								_, codeElements := GoGetFuncQualifierCodeElements(file, methodQual)
								groupCase.Add(codeElements...)

							}
						})
					codezIfaceMethods = append(codezIfaceMethods,
						Commentf("Untrusted flow sources from method calls on %s interface.", typ.QualifiedName).
							Line().
							Add(code),
					)
				}

				codez = append(codez,
					Comment("Untrusted flow sources from interface method calls.").
						Line().
						Block(codezIfaceMethods...),
				)
			}
		}

		{
			structQualifiers, ok := b2st[pathVersion]
			if ok {
				code := BlockFunc(
					func(groupCase *Group) {

						for _, qual := range structQualifiers {
							source := x.GetCachedSource(qual.Path, qual.Version)
							if source == nil {
								Fatalf("Source not found: %s@%s", qual.Path, qual.Version)
							}
							// Make sure that the struct exist:
							str := x.FindStructByID(source, qual.ID)
							if str == nil {
								Fatalf("Struct not found: %q", qual.ID)
							}
							gogentools.ImportPackage(file, str.PkgPath, str.PkgName)
							pathVersionToTypeNames[pathVersion] = append(pathVersionToTypeNames[pathVersion], str.TypeName)

							fieldNames := make([]string, 0)
							for fieldName := range qual.Fields {
								fieldNames = append(fieldNames, fieldName)
							}

							groupCase.Commentf("Untrusted flow sources from %s struct fields.", str.QualifiedName)
							groupCase.BlockFunc(
								func(subGroup *Group) {
									structVarName := gogentools.NewNameWithPrefix(feparser.NewLowerTitleName("struct", str.TypeName))
									subGroup.Id(structVarName).Op(":=").New(Qual(str.PkgPath, str.TypeName))

									if len(fieldNames) > 0 {
										if len(fieldNames) == 1 {
											fieldName := fieldNames[0]
											subGroup.Id("sink").Call(Id(structVarName).Dot(fieldName))
										} else {
											codeParamIDs := make([]Code, 0)
											for _, fieldName := range fieldNames {
												codeParamIDs = append(codeParamIDs, Id(structVarName).Dot(fieldName).Op(",").Line())
											}
											subGroup.Id("sink").Call(Line().Add(codeParamIDs...).Line())
										}
									}
								})

						}
					})

				codez = append(codez,
					Comment("Untrusted flow sources from struct fields.").
						Line().
						Add(code),
				)
			}
		}

		{
			typeQualifiers, ok := b2typ[pathVersion]
			if ok {
				code := BlockFunc(
					func(groupCase *Group) {
						for _, qual := range typeQualifiers {
							source := x.GetCachedSource(qual.Path, qual.Version)
							if source == nil {
								Fatalf("Source not found: %s@%s", qual.Path, qual.Version)
							}
							// Find the type:
							typ := x.FindTypeByID(source, qual.ID)
							if typ == nil {
								Fatalf("Type not found: %q", qual.ID)
							}
							gogentools.ImportPackage(file, typ.PkgPath, typ.PkgName)
							pathVersionToTypeNames[pathVersion] = append(pathVersionToTypeNames[pathVersion], typ.TypeName)

							typeVarName := gogentools.NewNameWithPrefix(feparser.NewLowerTitleName("type", typ.TypeName))

							groupCase.BlockFunc(
								func(subGroup *Group) {
									subGroup.Var().Id(typeVarName).Qual(typ.PkgPath, typ.TypeName)
									subGroup.Id("sink").Call(Id(typeVarName))
								})
						}
					})
				codez = append(codez,
					Comment("Untrusted flow sources from types.").
						Line().
						Add(code),
				)
			}
		}

		{
			path, _ := scanner.SplitPathVersion(pathVersion)
			isStd := search.IsStandardImportPath(path)
			if !isStd {
				// If path is NOT part of standard library, then add the depstubber generation comment.
				file.Comment(generateDepstubberComment(path, pathVersionToTypeNames[pathVersion], pathVersionToFuncAndVarNames[pathVersion]))
				file.Comment("//go:generate depstubber -write_module_txt").Line()
			}

			file.Comment("Untrusted flow sources from package: " + pathVersion)
		}
		file.Func().Id("main").Params().Block(codez...)
		fmt.Printf("%#v", file)
	}

	Q(pathVersionToFuncAndVarNames)
	Q(pathVersionToTypeNames)
	return nil
}

func generateDepstubberComment(path string, typeNames []string, funcAndVarNames []string) string {
	return Sf(
		"//go:generate depstubber -vendor %s %s %s",
		path,
		strings.Join(typeNames, ","),
		strings.Join(funcAndVarNames, ","),
	)
}

// Comments adds comments to a Group (if enabled), and returns the group.
func Comments(group *Group, comments ...string) *Group {
	if IncludeCommentsInGeneratedGo {
		for _, comment := range comments {
			group.Line().Comment(comment)
		}
	}
	return group
}
func GoGetFuncQualifierCodeElements(file *File, qual *x.FuncQualifier) (x.FuncInterface, []Code) {

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
	considerReceiver := false
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
				considerReceiver = true
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

	lenReceiver, _, _ := fn.Lengths()
	hasReceiver := lenReceiver == 1

	fe := fn.GetFunc()
	tpFun := fe.GetOriginal().GetType().(*types.Signature)
	receiver := fn.GetReceiver()

	// Compile array of the zero values of the function parameters:
	paramZeroVals := gogentools.ScanTupleOfZeroValues(file, tpFun.Params(), fe.GetOriginal().IsVariadic())

	// Compile array of the zero values of the function results:
	resultZeroVals := gogentools.ScanTupleOfZeroValues(file, tpFun.Results(), fe.GetOriginal().IsVariadic())

	code := BlockFunc(
		func(groupCase *Group) {

			codeCallFunc := Null()
			if hasReceiver {
				varName := gogentools.NewNameWithPrefix(feparser.NewLowerTitleName("receiver", receiver.TypeName))
				receiver.VarName = varName
				gogentools.ComposeVarDeclaration(file, groupCase, varName, receiver.GetOriginal(), fe.GetOriginal().Variadic)
				codeCallFunc = Id(varName).Dot(fe.Name)
			} else {
				codeCallFunc = Qual(fe.PkgPath, fe.Name)
			}

			// Decide parameter names, and declare variables that will be passed as those parameters:
			if len(parameterIndexes) > 0 {
				if len(parameterIndexes) == 1 {
					// If only one parameter is considered, the use a single var declaration:
					i := parameterIndexes[0]
					varName := gogentools.NewNameWithPrefix(feparser.NewLowerTitleName("param", fe.Parameters[i].VarName))
					fe.Parameters[i].VarName = varName
					gogentools.ComposeVarDeclaration(file, groupCase, varName, fe.Parameters[i].GetOriginal().GetType(), fe.GetOriginal().Variadic)
				} else {
					// If multiple parameters are considered, then use a group var declaration:
					varTypes := make([]*VarNameAndType, 0)
					for i := range paramZeroVals {
						isConsidered := IntSliceContains(parameterIndexes, i)
						if isConsidered {
							varName := gogentools.NewNameWithPrefix(feparser.NewLowerTitleName("param", fe.Parameters[i].VarName))
							fe.Parameters[i].VarName = varName

							varTypes = append(varTypes, &VarNameAndType{
								Name: varName,
								Type: fe.Parameters[i].GetOriginal().GetType(),
							})
						}
					}
					ComposeGroupVarDeclaration(file, groupCase, varTypes, fe.GetOriginal().Variadic)
				}
			}

			codeResultList := Null()
			if len(resultIndexes) > 0 {
				for i := range resultZeroVals {
					isConsidered := IntSliceContains(resultIndexes, i)
					if isConsidered {
						varName := gogentools.NewNameWithPrefix(feparser.NewLowerTitleName("result", fe.Results[i].VarName))
						fe.Results[i].VarName = varName
					}
				}

				codeResultList = ListFunc(func(resGroup *Group) {
					for i, v := range fe.Results {
						isConsidered := IntSliceContains(resultIndexes, i)
						if isConsidered {
							resGroup.Id(v.VarName)
						} else {
							resGroup.Id("_")
						}
					}
				}).Op(":=")
			}

			// Call the function, passing the considered parameters:
			groupCase.Add(codeResultList).Add(codeCallFunc).CallFunc(
				func(call *Group) {
					for i, zero := range paramZeroVals {
						isConsidered := IntSliceContains(parameterIndexes, i)
						if isConsidered {
							call.Id(fe.Parameters[i].VarName)
						} else {
							call.Add(zero)
						}
					}
				},
			)

			// Sink the parameters:
			if len(parameterIndexes) > 0 {
				//groupCase.Comment("Sink parameters:")
				if len(parameterIndexes) == 1 {
					i := parameterIndexes[0]
					groupCase.Id("sink").Call(Id(fe.Parameters[i].VarName))
				} else {
					codeParamIDs := make([]Code, 0)
					for i := range paramZeroVals {
						isConsidered := IntSliceContains(parameterIndexes, i)
						if isConsidered {
							codeParamIDs = append(codeParamIDs, Id(fe.Parameters[i].VarName).Op(",").Line())
						}
					}
					groupCase.Id("sink").Call(Line().Add(codeParamIDs...).Line())
				}
			}
			// Sink the results:
			if len(resultIndexes) > 0 {
				//groupCase.Comment("Sink results:")
				if len(resultIndexes) == 1 {
					i := resultIndexes[0]
					groupCase.Id("sink").Call(Id(fe.Results[i].VarName))
				} else {
					codeResultIDs := make([]Code, 0)
					for i := range resultZeroVals {
						isConsidered := IntSliceContains(resultIndexes, i)
						if isConsidered {
							codeResultIDs = append(codeResultIDs, Id(fe.Results[i].VarName).Op(",").Line())
						}
					}
					groupCase.Id("sink").Call(Line().Add(codeResultIDs...).Line())
				}
			}
			// Sink the receiver:
			if considerReceiver {
				//groupCase.Comment("Sink the receiver:")
				groupCase.Id("sink").Call(Id(receiver.VarName))
			}
		})

	codeElements = append(codeElements,
		code,
	)

	return fn, codeElements
}

type VarNameAndType struct {
	Name string
	Type types.Type
}

// declare:
// `var (
//		name1 Type1
//		name2 Type2
// 	)`
func ComposeGroupVarDeclaration(file *File, group *Group, decs []*VarNameAndType, isVariadic bool) {

	stat := newStatement()

	for _, dec := range decs {
		if isVariadic {
			gogentools.ComposeTypeDeclaration(file, stat.Id(dec.Name), dec.Type.(*types.Slice).Elem())
		} else {
			gogentools.ComposeTypeDeclaration(file, stat.Id(dec.Name), dec.Type)
		}
		stat.Line()
	}

	group.Var().Parens(stat)
}
func newStatement() *Statement {
	return &Statement{}
}
