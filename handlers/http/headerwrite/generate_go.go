package headerwrite

import (
	"go/types"
	"os"
	"path/filepath"

	. "github.com/dave/jennifer/jen"
	"github.com/gagliardetto/codebox/gogentools"
	"github.com/gagliardetto/codemill/x"
	"github.com/gagliardetto/feparser"
	. "github.com/gagliardetto/utilz"
)

var (
	GenerateBoilerplate bool
)

const (
	// NOTE: hardcoded inside TestQueryContent const.
	InlineExpectationsTestTagHeaderKeyNode = "$headerKeyNode" // Must start with a $ sign.
	InlineExpectationsTestTagHeaderValNode = "$headerValNode" // Must start with a $ sign.

	InlineExpectationsTestTagHeaderKey = "$headerKey" // Must start with a $ sign.
	InlineExpectationsTestTagHeaderVal = "$headerVal" // Must start with a $ sign.
)

// TagDynamicHeader creates a comment for headers that have dynamic
// key and value nodes.
func TagDynamicHeader(keyVarName, valVarName string) Code {
	tg := Sf(
		"%s=%s %s=%s",
		InlineExpectationsTestTagHeaderKeyNode,
		keyVarName,
		InlineExpectationsTestTagHeaderValNode,
		valVarName,
	)

	return Comment(tg)
}

// TagDynamicContentType creates a comment for content-type headers
// that have a dynamic value node.
func TagDynamicContentType(headerKey, valVarNodeName string) Code {
	tg := Sf(
		"%s=%s %s=%s",
		InlineExpectationsTestTagHeaderKey,
		headerKey,
		InlineExpectationsTestTagHeaderValNode,
		valVarNodeName,
	)

	return Comment(tg)
}

// TagStaticContentType creates a comment for content-type headers
// that have a static value.
func TagStaticContentType(headerKey, headerValue string) Code {
	tg := Sf(
		"%s=%s %s=%s",
		InlineExpectationsTestTagHeaderKey,
		headerKey,
		InlineExpectationsTestTagHeaderVal,
		headerValue,
	)

	return Comment(tg)
}

const (
	TestQueryContent = `
import go
import TestUtilities.InlineExpectationsTest

class HttpHeaderWriteTest extends InlineExpectationsTest {
  HttpHeaderWriteTest() { this = "HttpHeaderWriteTest" }

  override string getARelevantTag() {
    result = ["headerKeyNode", "headerValNode", "headerKey", "headerVal"]
  }

  override predicate hasActualResult(string file, int line, string element, string tag, string value) {
    // Dynamic key-value header:
    exists(HTTP::HeaderWrite hw |
      hw.hasLocationInfo(file, line, _, _, _) and
      (
        element = hw.getName().toString() and
        value = hw.getName().toString() and
        tag = "headerKeyNode"
        or
        element = hw.getValue().toString() and
        value = hw.getValue().toString() and
        tag = "headerValNode"
      )
    )
    or
    // Static key, dynamic value header:
    exists(HTTP::HeaderWrite hw |
      hw.hasLocationInfo(file, line, _, _, _) and
      (
        element = hw.getHeaderName().toString() and
        value = hw.getHeaderName() and
        tag = "headerKey"
        or
        element = hw.getValue().toString() and
        value = hw.getValue().toString() and
        tag = "headerValNode"
      )
    )
    or
    // Static key, static value header:
    exists(HTTP::HeaderWrite hw |
      hw.hasLocationInfo(file, line, _, _, _) and
      (
        element = hw.getHeaderName().toString() and
        value = hw.getHeaderName() and
        tag = "headerKey"
        or
        element = hw.getHeaderValue().toString() and
        value = hw.getHeaderValue() and
        tag = "headerVal"
      )
    )
  }
}

`
)

func NewTestFile(includeBoilerplace bool) *File {
	file := NewFile("main")
	// Set a prefix to avoid collision between variable names and packages:
	file.PackagePrefix = "cql"
	// Add comment to file:
	file.HeaderComment("Code generated by https://github.com/gagliardetto. DO NOT EDIT.")

	if includeBoilerplace {
		file.PackageComment("//go:generate depstubber --vendor --auto")
		{
			// main function:
			file.Func().Id("main").Params().Block()
		}
		{
			// The `source` function returns a new URL:
			code := Func().
				Id("source").
				Params().
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

func (han *Handler) GenerateGo(parentDir string, mdl *x.XModel) error {
	if err := mdl.Validate(); err != nil {
		return err
	}
	if err := han.Validate(mdl); err != nil {
		return err
	}
	// TODO:
	// - Validate Pos.

	// Check if there are multiple versions of a same package:
	mods := mdl.ListModules()
	if x.HasMultiversion(mods) {
		Ln(RedBG("Has multiversion"))
	}
	// If there are no multiple versions of the same module,
	// that means we can save all the code to one file.
	allInOneFile := !x.HasMultiversion(mods)

	// Create the directory for the tests for this model:
	outDir := filepath.Join(parentDir, feparser.NewCodeQlName(mdl.Name))
	MustCreateFolderIfNotExists(outDir, os.ModePerm)

	allPathVersions := mdl.ListAllPathVersions()

	file := NewTestFile(GenerateBoilerplate)

	for _, pathVersion := range allPathVersions {
		if !allInOneFile {
			// Reset file:
			file = NewTestFile(GenerateBoilerplate)
		}

		codez := make([]Code, 0)
		{
			{
				tmpCodez, err := addTests_Header_KeyVal_Method(file, mdl, pathVersion)
				if err != nil {
					panic(err)
				}
				if tmpCodez != nil {
					codez = append(codez, tmpCodez...)
				}
			}
			{
				tmpCodez, err := addTests_ContentType_Dynamic(file, mdl, pathVersion)
				if err != nil {
					panic(err)
				}
				if tmpCodez != nil {
					codez = append(codez, tmpCodez...)
				}
			}
			{
				tmpCodez, err := addTests_ContentType_Static(file, mdl, pathVersion)
				if err != nil {
					panic(err)
				}
				if tmpCodez != nil {
					codez = append(codez, tmpCodez...)
				}
			}
		}

		{
			file.Commentf("Package %s", pathVersion)
			file.Func().Id(mdl.Name + "_" + feparser.FormatCodeQlName(pathVersion)).Params().Block(codez...)
		}

		if !allInOneFile {
			pkgDstDirpath := filepath.Join(outDir, feparser.FormatID(mdl.Name, "For", feparser.FormatCodeQlName(pathVersion)))
			MustCreateFolderIfNotExists(pkgDstDirpath, os.ModePerm)

			assetFileName := feparser.FormatID(mdl.Name, "For", feparser.FormatCodeQlName(pathVersion)) + ".go"
			if err := x.SaveGoFile(pkgDstDirpath, assetFileName, file); err != nil {
				Fatalf("Error while saving go file: %s", err)
			}

			if err := x.WriteGoModFile(pkgDstDirpath, pathVersion); err != nil {
				Fatalf("Error while saving go.mod file: %s", err)
			}
			if err := x.WriteCodeQLTestQuery(pkgDstDirpath, mdl.Name, TestQueryContent); err != nil {
				Fatalf("Error while saving <name>.ql file: %s", err)
			}
			if err := x.WriteEmptyCodeQLDotExpectedFile(pkgDstDirpath, mdl.Name); err != nil {
				Fatalf("Error while saving <name>.expected file: %s", err)
			}
		}
	}

	if allInOneFile {
		pkgDstDirpath := outDir
		MustCreateFolderIfNotExists(pkgDstDirpath, os.ModePerm)

		assetFileName := feparser.FormatID(mdl.Name) + ".go"
		if err := x.SaveGoFile(pkgDstDirpath, assetFileName, file); err != nil {
			Fatalf("Error while saving go file: %s", err)
		}

		if err := x.WriteGoModFile(pkgDstDirpath, allPathVersions...); err != nil {
			Fatalf("Error while saving go.mod file: %s", err)
		}
		if err := x.WriteCodeQLTestQuery(pkgDstDirpath, mdl.Name, TestQueryContent); err != nil {
			Fatalf("Error while saving <name>.ql file: %s", err)
		}
		if err := x.WriteEmptyCodeQLDotExpectedFile(pkgDstDirpath, mdl.Name); err != nil {
			Fatalf("Error while saving <name>.expected file: %s", err)
		}
	}
	return nil
}

func addTests_Header_KeyVal_Method(file *File, mdl *x.XModel, pathVersion string) ([]Code, error) {
	codez := make([]Code, 0)
	// Assuming the validation has already been done:
	MethodWriteHeaderKey := mdl.Methods.ByName(MethodWriteHeaderKey)
	if len(MethodWriteHeaderKey.Selectors) == 0 {
		Infof("No selectors found for %q method.", MethodWriteHeaderKey.Name)
		return nil, nil
	}

	MethodWriteHeaderVal := mdl.Methods.ByName(MethodWriteHeaderVal)
	if len(MethodWriteHeaderVal.Selectors) == 0 {
		Infof("No selectors found for %q method.", MethodWriteHeaderVal.Name)
		return nil, nil
	}

	_, b2tmKey, b2itmKey, err := x.GroupFuncSelectors(MethodWriteHeaderKey)
	if err != nil {
		Fatalf("Error while GroupFuncSelectors: %s", err)
	}
	_, b2tmVal, b2itmVal, err := x.GroupFuncSelectors(MethodWriteHeaderVal)
	if err != nil {
		Fatalf("Error while GroupFuncSelectors: %s", err)
	}
	// TODO: consider also header writes done with a function?

	{
		codezTypeMethods := make([]Code, 0)
		b2tmKey.IterValid(pathVersion,
			func(receiverTypeID string, methodQualifiers x.FuncQualifierSlice) {

				qual := methodQualifiers[0]
				// Find receiver type:
				typ := x.FindType(qual.Path, qual.Version, receiverTypeID)
				if typ == nil {
					Fatalf("Type not found: %q", receiverTypeID)
				}

				gogentools.ImportPackage(file, typ.PkgPath, typ.PkgName)

				code := BlockFunc(
					func(groupCase *Group) {

						for _, keyMethodQual := range methodQualifiers {
							fn := x.GetFuncByQualifier(keyMethodQual)
							thing := fn.(*feparser.FETypeMethod)
							x.AddImportsFromFunc(file, fn)

							// TODO:
							// - Check if found.
							valMethodQual := b2tmVal[pathVersion][receiverTypeID].ByBasicQualifier(keyMethodQual.BasicQualifier)

							{
								if AllFalse(keyMethodQual.Pos...) {
									continue
								}
								groupCase.Comment(thing.Func.Signature)

								blocksOfCases := generateGoTestBlock_DynamicHeaderKeyVal(
									file,
									thing,
									keyMethodQual,
									valMethodQual,
								)
								if len(blocksOfCases) == 1 {
									groupCase.Add(blocksOfCases...)
								} else {
									groupCase.Block(blocksOfCases...)
								}
							}

						}
					})
				// TODO: what if no flows are enabled? Check that before adding the comment.
				codezTypeMethods = append(codezTypeMethods,
					Commentf("Header write via method calls on %s.", typ.QualifiedName).
						Line().
						Add(code),
				)
			})
		if len(codezTypeMethods) > 0 {
			codez = append(codez,
				Comment("Header write via method calls.").
					Line().
					Block(codezTypeMethods...),
			)
		}
	}

	{
		codezIfaceMethods := make([]Code, 0)
		b2itmKey.IterValid(pathVersion,
			func(receiverTypeID string, methodQualifiers x.FuncQualifierSlice) {

				qual := methodQualifiers[0]
				// Find receiver type:
				typ := x.FindType(qual.Path, qual.Version, receiverTypeID)
				if typ == nil {
					Fatalf("Type not found: %q", receiverTypeID)
				}

				gogentools.ImportPackage(file, typ.PkgPath, typ.PkgName)

				code := BlockFunc(
					func(groupCase *Group) {

						for _, keyMethodQual := range methodQualifiers {
							fn := x.GetFuncByQualifier(keyMethodQual)
							thing := fn.(*feparser.FEInterfaceMethod)
							x.AddImportsFromFunc(file, fn)

							// TODO:
							// - Check if found.
							valMethodQual := b2itmVal[pathVersion][receiverTypeID].ByBasicQualifier(keyMethodQual.BasicQualifier)

							{
								if AllFalse(keyMethodQual.Pos...) {
									continue
								}
								groupCase.Comment(thing.Func.Signature)

								converted := feparser.FEIToFET(thing)
								blocksOfCases := generateGoTestBlock_DynamicHeaderKeyVal(
									file,
									converted,
									keyMethodQual,
									valMethodQual,
								)
								if len(blocksOfCases) == 1 {
									groupCase.Add(blocksOfCases...)
								} else {
									groupCase.Block(blocksOfCases...)
								}
							}
						}
					})
				codezIfaceMethods = append(codezIfaceMethods,
					Commentf("Header write via method calls on %s interface.", typ.QualifiedName).
						Line().
						Add(code),
				)
			})

		if len(codezIfaceMethods) > 0 {
			codez = append(codez,
				Comment("Header write via interface method calls.").
					Line().
					Block(codezIfaceMethods...),
			)
		}
	}
	return codez, nil
}

func addTests_ContentType_Dynamic(file *File, mdl *x.XModel, pathVersion string) ([]Code, error) {
	codez := make([]Code, 0)

	MethodCt := mdl.Methods.ByName(MethodCt)
	if len(MethodCt.Selectors) == 0 {
		Infof("No selectors found for %q method.", MethodCt.Name)
		return nil, nil
	}
	{
		_, b2tm, b2itm, err := x.GroupFuncSelectors(MethodCt)
		if err != nil {
			Fatalf("Error while GroupFuncSelectors: %s", err)
		}
		{
			codezTypeMethods := make([]Code, 0)
			b2tm.IterValid(pathVersion,
				func(receiverTypeID string, methodQualifiers x.FuncQualifierSlice) {

					qual := methodQualifiers[0]
					// Find receiver type:
					typ := x.FindType(qual.Path, qual.Version, receiverTypeID)
					if typ == nil {
						Fatalf("Type not found: %q", receiverTypeID)
					}

					gogentools.ImportPackage(file, typ.PkgPath, typ.PkgName)

					code := BlockFunc(
						func(groupCase *Group) {

							for _, methodQual := range methodQualifiers {
								fn := x.GetFuncByQualifier(methodQual)
								thing := fn.(*feparser.FETypeMethod)
								x.AddImportsFromFunc(file, fn)

								{
									if AllFalse(methodQual.Pos...) {
										continue
									}
									groupCase.Comment(thing.Func.Signature)

									blocksOfCases := par_go_ContentType_DynamicValue(
										file,
										methodQual,
									)
									if len(blocksOfCases) == 1 {
										groupCase.Add(blocksOfCases...)
									} else {
										groupCase.Block(blocksOfCases...)
									}
								}

							}
						})
					// TODO: what if no flows are enabled? Check that before adding the comment.
					codezTypeMethods = append(codezTypeMethods,
						Commentf("Dynamic Content-Type header via method calls on %s.", typ.QualifiedName).
							Line().
							Add(code),
					)
				})
			if len(codezTypeMethods) > 0 {
				codez = append(codez,
					Comment("Dynamic Content-Type header via method calls.").
						Line().
						Block(codezTypeMethods...),
				)
			}
		}

		{
			codezIfaceMethods := make([]Code, 0)
			b2itm.IterValid(pathVersion,
				func(receiverTypeID string, methodQualifiers x.FuncQualifierSlice) {

					qual := methodQualifiers[0]
					// Find receiver type:
					typ := x.FindType(qual.Path, qual.Version, receiverTypeID)
					if typ == nil {
						Fatalf("Type not found: %q", receiverTypeID)
					}

					gogentools.ImportPackage(file, typ.PkgPath, typ.PkgName)

					code := BlockFunc(
						func(groupCase *Group) {

							for _, methodQual := range methodQualifiers {
								fn := x.GetFuncByQualifier(methodQual)
								thing := fn.(*feparser.FEInterfaceMethod)
								x.AddImportsFromFunc(file, fn)

								{
									if AllFalse(methodQual.Pos...) {
										continue
									}
									groupCase.Comment(thing.Func.Signature)

									blocksOfCases := par_go_ContentType_DynamicValue(
										file,
										methodQual,
									)
									if len(blocksOfCases) == 1 {
										groupCase.Add(blocksOfCases...)
									} else {
										groupCase.Block(blocksOfCases...)
									}
								}
							}
						})
					codezIfaceMethods = append(codezIfaceMethods,
						Commentf("Dynamic Content-Type header via method calls on %s interface.", typ.QualifiedName).
							Line().
							Add(code),
					)
				})

			if len(codezIfaceMethods) > 0 {
				codez = append(codez,
					Comment("Dynamic Content-Type header via interface method calls.").
						Line().
						Block(codezIfaceMethods...),
				)
			}
		}
	}

	return codez, nil
}

func addTests_ContentType_Static(file *File, mdl *x.XModel, pathVersion string) ([]Code, error) {
	codez := make([]Code, 0)

	MethodCtFromFuncName := mdl.Methods.ByName(MethodCtFromFuncName)
	if len(MethodCtFromFuncName.Selectors) == 0 {
		Infof("No selectors found for %q method.", MethodCtFromFuncName.Name)
		return nil, nil
	}
	{
		_, b2tm, b2itm, err := x.GroupFuncSelectors(MethodCtFromFuncName)
		if err != nil {
			Fatalf("Error while GroupFuncSelectors: %s", err)
		}
		{
			codezTypeMethods := make([]Code, 0)
			b2tm.IterValid(pathVersion,
				func(receiverTypeID string, methodQualifiers x.FuncQualifierSlice) {

					qual := methodQualifiers[0]
					// Find receiver type:
					typ := x.FindType(qual.Path, qual.Version, receiverTypeID)
					if typ == nil {
						Fatalf("Type not found: %q", receiverTypeID)
					}

					gogentools.ImportPackage(file, typ.PkgPath, typ.PkgName)

					code := BlockFunc(
						func(groupCase *Group) {

							for _, methodQual := range methodQualifiers {
								fn := x.GetFuncByQualifier(methodQual)
								thing := fn.(*feparser.FETypeMethod)
								x.AddImportsFromFunc(file, fn)

								{
									if AllFalse(methodQual.Pos...) {
										continue
									}
									groupCase.Comment(thing.Func.Signature)

									blocksOfCases := par_go_ContentType_StaticValue(
										file,
										methodQual,
									)
									if len(blocksOfCases) == 1 {
										groupCase.Add(blocksOfCases...)
									} else {
										groupCase.Block(blocksOfCases...)
									}
								}

							}
						})
					// TODO: what if no flows are enabled? Check that before adding the comment.
					codezTypeMethods = append(codezTypeMethods,
						Commentf("Static Content-Type header write via method calls on %s.", typ.QualifiedName).
							Line().
							Add(code),
					)
				})
			if len(codezTypeMethods) > 0 {
				codez = append(codez,
					Comment("Static Content-Type header write via method calls.").
						Line().
						Block(codezTypeMethods...),
				)
			}
		}

		{
			codezIfaceMethods := make([]Code, 0)
			b2itm.IterValid(pathVersion,
				func(receiverTypeID string, methodQualifiers x.FuncQualifierSlice) {

					qual := methodQualifiers[0]
					// Find receiver type:
					typ := x.FindType(qual.Path, qual.Version, receiverTypeID)
					if typ == nil {
						Fatalf("Type not found: %q", receiverTypeID)
					}

					gogentools.ImportPackage(file, typ.PkgPath, typ.PkgName)

					code := BlockFunc(
						func(groupCase *Group) {

							for _, methodQual := range methodQualifiers {
								fn := x.GetFuncByQualifier(methodQual)
								thing := fn.(*feparser.FEInterfaceMethod)
								x.AddImportsFromFunc(file, fn)

								{
									if AllFalse(methodQual.Pos...) {
										continue
									}
									groupCase.Comment(thing.Func.Signature)

									blocksOfCases := par_go_ContentType_StaticValue(
										file,
										methodQual,
									)
									if len(blocksOfCases) == 1 {
										groupCase.Add(blocksOfCases...)
									} else {
										groupCase.Block(blocksOfCases...)
									}
								}
							}
						})
					codezIfaceMethods = append(codezIfaceMethods,
						Commentf("Static Content-Type header write via method calls on %s interface.", typ.QualifiedName).
							Line().
							Add(code),
					)
				})

			if len(codezIfaceMethods) > 0 {
				codez = append(codez,
					Comment("Static Content-Type header write via interface method calls.").
						Line().
						Block(codezIfaceMethods...),
				)
			}
		}
	}

	return codez, nil
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

func newStatement() *Statement {
	return &Statement{}
}

func generateGoTestBlock_DynamicHeaderKeyVal(
	file *File,
	fe *feparser.FETypeMethod,
	qualHeaderKey *x.FuncQualifier,
	qualHeaderVal *x.FuncQualifier,
) []Code {
	childBlocks := make([]Code, 0)

	headerKeyIndexes := x.MustPosToRelativeParamIndexes(fe, qualHeaderKey.Pos)
	if len(headerKeyIndexes) != 1 {
		Fatalf("headerKeyIndexes len is not 1: %v", qualHeaderKey)
	}
	headerValIndexes := x.MustPosToRelativeParamIndexes(fe, qualHeaderVal.Pos)
	if len(headerValIndexes) != 1 {
		Fatalf("headerValIndexes len is not 1: %v", qualHeaderVal)
	}

	childBlock := generate_Method(
		file,
		fe,
		headerKeyIndexes[0],
		headerValIndexes[0],
	)
	{
		if childBlock != nil {
			childBlocks = append(childBlocks, childBlock)
		} else {
			Warnf(Sf("NOTHING GENERATED; qualHeaderKey %v, qualHeaderVal %v", qualHeaderKey.Pos, qualHeaderVal.Pos))
		}
	}

	return childBlocks
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func generate_Method(file *File, fe *feparser.FETypeMethod, indexKey int, indexVal int) *Statement {

	keyParam := fe.Func.Parameters[indexKey]
	keyParam.VarName = gogentools.NewNameWithPrefix(feparser.NewLowerTitleName("key", keyParam.TypeName))

	valParam := fe.Func.Parameters[indexVal]
	valParam.VarName = gogentools.NewNameWithPrefix(feparser.NewLowerTitleName("val", valParam.TypeName))

	code := BlockFunc(
		func(groupCase *Group) {

			ComposeTypeAssertion(file, groupCase, keyParam.VarName, keyParam.GetOriginal().GetType(), keyParam.GetOriginal().IsVariadic())
			ComposeTypeAssertion(file, groupCase, valParam.VarName, valParam.GetOriginal().GetType(), valParam.GetOriginal().IsVariadic())

			Comments(groupCase, "Declare medium object/interface:")
			groupCase.Var().Id("rece").Qual(fe.Receiver.PkgPath, fe.Receiver.TypeName)

			gogentools.ImportPackage(file, fe.Func.PkgPath, fe.Func.PkgName)

			groupCase.Id("rece").Dot(fe.Func.Name).CallFunc(
				func(call *Group) {

					tpFun := fe.Func.GetOriginal().GetType().(*types.Signature)

					zeroVals := gogentools.ScanTupleOfZeroValues(file, tpFun.Params(), fe.Func.GetOriginal().IsVariadic())

					for i, zero := range zeroVals {
						isConsidered := IntSliceContains([]int{indexKey, indexVal}, i)
						if isConsidered {
							call.Id(fe.Func.Parameters[i].VarName)
						} else {
							call.Add(zero)
						}
					}

				},
			).Add(TagDynamicHeader(keyParam.VarName, valParam.VarName))

		})
	return code
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// declare `name := source().(Type)`
func ComposeTypeAssertion(file *File, group *Group, varName string, typ types.Type, isVariadic bool) {
	assertContent := newStatement()
	if isVariadic {
		if slice, ok := typ.(*types.Slice); ok {
			gogentools.ComposeTypeDeclaration(file, assertContent, slice.Elem())
		} else {
			gogentools.ComposeTypeDeclaration(file, assertContent, typ)
		}
	} else {
		gogentools.ComposeTypeDeclaration(file, assertContent, typ)
	}
	group.Id(varName).Op(":=").Id("source").Call().Assert(assertContent)
}

func par_go_ContentType_DynamicValue(
	file *File,
	ctQual *x.FuncQualifier,
) []Code {

	childBlocks := make([]Code, 0)

	ctFn := x.GetFuncByQualifier(ctQual)
	// TODO: support here multiple, too?
	ctIndexes := x.MustPosToRelativeParamIndexes(ctFn, ctQual.Pos)
	if len(ctIndexes) != 1 {
		Fatalf("ctIndexes len is not 1: %v", ctQual)
	}

	childBlock := goChildBlock_ContentType_DynamicValue(
		file,
		ctFn,
		ctIndexes[0],
	)
	{
		if childBlock != nil {
			childBlocks = append(childBlocks, childBlock)
		} else {
			Warnf(Sf("NOTHING GENERATED; ctQual %v", ctQual))
		}
	}

	return childBlocks
}

func goChildBlock_ContentType_DynamicValue(
	file *File,
	ctFn x.FuncInterface,
	ctIndex int,
) *Statement {

	ctParam := ctFn.GetFunc().Parameters[ctIndex]
	ctParam.VarName = gogentools.NewNameWithPrefix(feparser.NewLowerTitleName("val", ctParam.TypeName))

	ctFnHasReceiver := ctFn.GetReceiver() != nil

	code := BlockFunc(
		func(groupCase *Group) {

			gogentools.ImportPackage(file, ctFn.GetFunc().PkgPath, ctFn.GetFunc().PkgName)
			ComposeTypeAssertion(file, groupCase, ctParam.VarName, ctParam.GetOriginal().GetType(), ctParam.GetOriginal().IsVariadic())

			if ctFnHasReceiver {
				groupCase.Var().Id("rece").Qual(ctFn.GetReceiver().PkgPath, ctFn.GetReceiver().TypeName)
			}
			{
				var afterCt *Statement
				if ctFnHasReceiver {
					afterCt = groupCase.Id("rece").Dot(ctFn.GetFunc().Name)
				} else {
					afterCt = groupCase.Qual(ctFn.GetFunc().PkgPath, ctFn.GetFunc().Name)
				}
				afterCt.CallFunc(
					func(call *Group) {

						tpFun := ctFn.GetFunc().GetOriginal().GetType().(*types.Signature)

						zeroVals := gogentools.ScanTupleOfZeroValues(file, tpFun.Params(), ctFn.GetFunc().GetOriginal().IsVariadic())

						for i, zero := range zeroVals {
							isConsidered := IntSliceContains([]int{ctIndex}, i)
							if isConsidered {
								if ctIndex == i {
									if ctParam.GetOriginal().TypeString() == "string" {
										call.Id(ctParam.VarName)
									} else {
										// try to convert string to the desired type; NOTE: might not be possible.
										typ := getTypeContent(file, groupCase, ctParam.VarName, ctParam.GetOriginal().GetType(), ctParam.GetOriginal().IsVariadic())
										call.Add(typ).Call(Id(ctParam.VarName))
									}
								} else {
									call.Id(ctFn.GetFunc().Parameters[i].VarName)
								}
							} else {
								call.Add(zero)
							}
						}
					},
				).Add(TagDynamicContentType("content-type", ctParam.VarName))
			}

		})
	return code
}

func par_go_ContentType_StaticValue(
	file *File,
	ctQual *x.FuncQualifier,
) []Code {

	childBlocks := make([]Code, 0)

	ctFn := x.GetFuncByQualifier(ctQual)
	// TODO: support here multiple, too?

	childBlock := goChildBlock_ContentType_StaticValue(
		file,
		ctFn,
	)
	{
		if childBlock != nil {
			childBlocks = append(childBlocks, childBlock)
		} else {
			Warnf(Sf("NOTHING GENERATED; ctQual %v", ctQual))
		}
	}

	return childBlocks
}

func goChildBlock_ContentType_StaticValue(
	file *File,
	ctFn x.FuncInterface,
) *Statement {

	ctFnHasReceiver := ctFn.GetReceiver() != nil

	code := BlockFunc(
		func(groupCase *Group) {

			gogentools.ImportPackage(file, ctFn.GetFunc().PkgPath, ctFn.GetFunc().PkgName)

			if ctFnHasReceiver {
				groupCase.Var().Id("rece").Qual(ctFn.GetReceiver().PkgPath, ctFn.GetReceiver().TypeName)
			}

			{
				var afterCt *Statement
				if ctFnHasReceiver {
					// NOTE: this assumes that both functions are methods on the same receiver.
					afterCt = groupCase.Id("rece").Dot(ctFn.GetFunc().Name)
				} else {
					afterCt = groupCase.Qual(ctFn.GetFunc().PkgPath, ctFn.GetFunc().Name)
				}
				afterCt.Call().
					Add(TagStaticContentType("content-type", x.GuessContentTypeFromName(ctFn.GetFunc().Name)))
			}

		})
	return code
}

// TODO: verify
func getTypeContent(file *File, group *Group, varName string, typ types.Type, isVariadic bool) *Statement {
	assertContent := newStatement()
	if isVariadic {
		if slice, ok := typ.(*types.Slice); ok {
			gogentools.ComposeTypeDeclaration(file, assertContent, slice.Elem())
		} else {
			gogentools.ComposeTypeDeclaration(file, assertContent, typ)
		}
	} else {
		gogentools.ComposeTypeDeclaration(file, assertContent, typ)
	}
	return assertContent
}
