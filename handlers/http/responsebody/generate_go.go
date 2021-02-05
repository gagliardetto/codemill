package responsebody

import (
	"go/types"
	"os"
	"path/filepath"
	"sort"

	. "github.com/dave/jennifer/jen"
	"github.com/gagliardetto/codebox/gogentools"
	"github.com/gagliardetto/codemill/x"
	"github.com/gagliardetto/feparser"
	. "github.com/gagliardetto/utilz"
)

const (
	// NOTE: hardcoded inside TestQueryContent const.
	InlineExpectationsTestTagResponseBody = "$responseBody" // Must start with a $ sign.
	InlineExpectationsTestTagContentType  = "$contentType"  // Must start with a $ sign.
)

func TagResponseBody(vals ...string) string {
	tg := ""
	for i, v := range vals {
		if i > 0 {
			tg += " "
		}
		tg += InlineExpectationsTestTagResponseBody + "=" + v
	}
	return tg
}

func TagContentType(vals ...string) string {
	tg := ""
	for i, v := range vals {
		if i > 0 {
			tg += " "
		}
		tg += InlineExpectationsTestTagContentType + "=" + v
	}
	return tg
}
func Tag(contentTypes string, respBodies string) Code {
	if contentTypes == "" {
		return Comment(respBodies)
	}
	return Comment(contentTypes + " " + respBodies)
}

const (
	TestQueryContent = `
import go
import TestUtilities.InlineExpectationsTest

class HttpResponseBodyTest extends InlineExpectationsTest {
  HttpResponseBodyTest() { this = "HttpResponseBodyTest" }

  override string getARelevantTag() { result = ["contentType", "responseBody"] }

  override predicate hasActualResult(string file, int line, string element, string tag, string value) {
    exists(HTTP::ResponseBody rd |
      rd.hasLocationInfo(file, line, _, _, _) and
      (
        element = rd.getAContentType().toString() and
        value = rd.getAContentType().toString() and
        tag = "contentType"
        or
        element = rd.toString() and
        value = rd.toString() and
        tag = "responseBody"
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
		{
			// main function:
			file.Func().Id("main").Params().Block()
		}
		{
			// The `source` function:
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

	// Assuming the validation has already been done:
	method := mdl.Methods[0]

	if len(method.Selectors) == 0 {
		Infof("No selectors found for %q method.", method.Name)
		return nil
	}

	allPathVersions := func() []string {
		res := make([]string, 0)
		mods := mdl.ListModules()
		for _, mod := range mods {
			res = append(res, mod.PathVersionClean())
		}
		sort.Strings(res)
		return res
	}()

	file := NewTestFile(true)

	for _, pathVersion := range allPathVersions {
		if !allInOneFile {
			// Reset file:
			file = NewTestFile(true)
		}
		codez := make([]Code, 0)

		b2fe, b2tm, b2itm, err := x.GroupFuncSelectors(method)
		if err != nil {
			Fatalf("Error while GroupFuncSelectors: %s", err)
		}

		{
			cont, ok := b2fe[pathVersion]
			if ok && x.HasValidPos(cont...) {
				code := BlockFunc(
					func(groupCase *Group) {

						for _, funcQual := range cont {
							fn := x.GetFuncQualifier(funcQual)
							thing := fn.(*feparser.FEFunc)

							x.AddImportsFromFunc(file, thing)

							{
								if AllFalse(funcQual.Pos...) {
									continue
								}
								groupCase.Comment(thing.Signature)

								blocksOfCases := generateGoTestBlock_Func(
									file,
									thing,
									funcQual,
								)
								if len(blocksOfCases) == 1 {
									groupCase.Add(blocksOfCases...)
								} else {
									groupCase.Block(blocksOfCases...)
								}
							}

						}
					})
				codez = append(codez,
					Comment("Set ResponseBody via function call.").
						Line().
						Add(code),
				)
			}
		}
		{
			cont, ok := b2tm[pathVersion]
			if ok {
				codezTypeMethods := make([]Code, 0)
				keys := func(v map[string]x.FuncQualifierSlice) []string {
					res := make([]string, 0)
					for key := range v {
						res = append(res, key)
					}
					sort.Strings(res)
					return res
				}(cont)
				for _, receiverTypeID := range keys {
					methodQualifiers := cont[receiverTypeID]
					if len(methodQualifiers) == 0 || !x.HasValidPos(methodQualifiers...) {
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
						Fatalf("Type not found: %q", receiverTypeID)
					}

					gogentools.ImportPackage(file, typ.PkgPath, typ.PkgName)

					code := BlockFunc(
						func(groupCase *Group) {

							for _, methodQual := range methodQualifiers {
								fn := x.GetFuncQualifier(methodQual)
								thing := fn.(*feparser.FETypeMethod)
								x.AddImportsFromFunc(file, fn)

								{
									if AllFalse(methodQual.Pos...) {
										continue
									}
									groupCase.Comment(thing.Func.Signature)

									blocksOfCases := generateGoTestBlock_Method(
										file,
										thing,
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
						Commentf("Set ResponseBody via method calls on %s.", typ.QualifiedName).
							Line().
							Add(code),
					)
				}
				codez = append(codez,
					Comment("Set ResponseBody via method calls.").
						Line().
						Block(codezTypeMethods...),
				)
			}
		}

		{
			cont, ok := b2itm[pathVersion]
			if ok {
				codezIfaceMethods := make([]Code, 0)
				keys := func(v map[string]x.FuncQualifierSlice) []string {
					res := make([]string, 0)
					for key := range v {
						res = append(res, key)
					}
					sort.Strings(res)
					return res
				}(cont)
				for _, receiverTypeID := range keys {
					methodQualifiers := cont[receiverTypeID]
					if len(methodQualifiers) == 0 || !x.HasValidPos(methodQualifiers...) {
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
						Fatalf("Type not found: %q", receiverTypeID)
					}

					gogentools.ImportPackage(file, typ.PkgPath, typ.PkgName)

					code := BlockFunc(
						func(groupCase *Group) {

							for _, methodQual := range methodQualifiers {
								fn := x.GetFuncQualifier(methodQual)
								thing := fn.(*feparser.FEInterfaceMethod)
								x.AddImportsFromFunc(file, fn)

								{
									if AllFalse(methodQual.Pos...) {
										continue
									}
									groupCase.Comment(thing.Func.Signature)

									converted := feparser.FEIToFET(thing)
									blocksOfCases := generateGoTestBlock_Method(
										file,
										converted,
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
						Commentf("Set ResponseBody via method calls on %s interface.", typ.QualifiedName).
							Line().
							Add(code),
					)
				}

				codez = append(codez,
					Comment("Set ResponseBody via interface method calls.").
						Line().
						Block(codezIfaceMethods...),
				)
			}
		}

		{
			file.Commentf("Package %s", pathVersion)
			file.Func().Id(feparser.FormatCodeQlName(pathVersion)).Params().Block(codez...)
		}

		if !allInOneFile {
			file.PackageComment("//go:generate depstubber --vendor --auto")

			pkgDstDirpath := filepath.Join(outDir, feparser.FormatID("Model", mdl.Name, "For", feparser.FormatCodeQlName(pathVersion)))
			MustCreateFolderIfNotExists(pkgDstDirpath, os.ModePerm)

			assetFileName := feparser.FormatID("Model", mdl.Name, "For", feparser.FormatCodeQlName(pathVersion)) + ".go"
			if err := x.SaveGoFile(pkgDstDirpath, assetFileName, file); err != nil {
				Fatalf("Error while saving go file: %s", err)
			}

			if err := x.WriteGoModFile(pkgDstDirpath, pathVersion); err != nil {
				Fatalf("Error while saving go.mod file: %s", err)
			}
			if err := x.WriteCodeQLTestQuery(pkgDstDirpath, x.DefaultCodeQLTestFileName, TestQueryContent); err != nil {
				Fatalf("Error while saving <name>.ql file: %s", err)
			}
			if err := x.WriteEmptyCodeQLDotExpectedFile(pkgDstDirpath, x.DefaultCodeQLTestFileName); err != nil {
				Fatalf("Error while saving <name>.expected file: %s", err)
			}
		}
	}

	if allInOneFile {
		file.PackageComment("//go:generate depstubber --vendor --auto")

		pkgDstDirpath := outDir
		MustCreateFolderIfNotExists(pkgDstDirpath, os.ModePerm)

		assetFileName := feparser.FormatID("Model", mdl.Name) + ".go"
		if err := x.SaveGoFile(pkgDstDirpath, assetFileName, file); err != nil {
			Fatalf("Error while saving go file: %s", err)
		}

		if err := x.WriteGoModFile(pkgDstDirpath, allPathVersions...); err != nil {
			Fatalf("Error while saving go.mod file: %s", err)
		}
		if err := x.WriteCodeQLTestQuery(pkgDstDirpath, x.DefaultCodeQLTestFileName, TestQueryContent); err != nil {
			Fatalf("Error while saving <name>.ql file: %s", err)
		}
		if err := x.WriteEmptyCodeQLDotExpectedFile(pkgDstDirpath, x.DefaultCodeQLTestFileName); err != nil {
			Fatalf("Error while saving <name>.expected file: %s", err)
		}
	}
	return nil
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

func generateGoTestBlock_Func(file *File, fe *feparser.FEFunc, qual *x.FuncQualifier) []Code {
	childBlocks := make([]Code, 0)

	indexes := x.MustPosToRelativeParamIndexes(fe, qual.Pos)

	childBlock := generate_Func(
		file,
		fe,
		indexes,
	)
	{
		if childBlock != nil {
			childBlocks = append(childBlocks, childBlock)
		} else {
			Warnf(Sf("NOTHING GENERATED; pos %v, param indexes %v", qual.Pos, indexes))
		}
	}

	return childBlocks
}
func generateGoTestBlock_Method(file *File, fe *feparser.FETypeMethod, qual *x.FuncQualifier) []Code {
	childBlocks := make([]Code, 0)

	indexes := x.MustPosToRelativeParamIndexes(fe, qual.Pos)

	childBlock := generate_Method(
		file,
		fe,
		indexes,
	)
	{
		if childBlock != nil {
			childBlocks = append(childBlocks, childBlock)
		} else {
			Warnf(Sf("NOTHING GENERATED; pos %v, param indexes %v", qual.Pos, indexes))
		}
	}

	return childBlocks
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
func generate_Func(file *File, fe *feparser.FEFunc, indexes []int) *Statement {

	for _, index := range indexes {
		in := fe.Parameters[index]

		in.VarName = gogentools.NewNameWithPrefix(feparser.NewLowerTitleName("body", in.TypeName))
	}

	varNames := make([]string, 0)
	for _, index := range indexes {
		in := fe.Parameters[index]

		varNames = append(varNames, in.VarName)
	}

	code := BlockFunc(
		func(groupCase *Group) {

			for _, index := range indexes {
				in := fe.Parameters[index]

				ComposeTypeAssertion(file, groupCase, in.VarName, in.GetOriginal().GetType(), in.GetOriginal().IsVariadic())
			}

			groupCase.Qual(fe.PkgPath, fe.Name).CallFunc(
				func(call *Group) {

					tpFun := fe.GetOriginal().GetType().(*types.Signature)

					zeroVals := gogentools.ScanTupleOfZeroValues(file, tpFun.Params(), fe.GetOriginal().IsVariadic())

					for i, zero := range zeroVals {
						isConsidered := IntSliceContains(indexes, i)
						if isConsidered {
							call.Id(fe.Parameters[i].VarName)
						} else {
							call.Add(zero)
						}
					}

				},
			).Add(Tag(TagContentType(guessContentTypeFromFuncName(fe.Name)), TagResponseBody(varNames...)))

		})
	return code
}
func generate_Method(file *File, fe *feparser.FETypeMethod, indexes []int) *Statement {

	for _, index := range indexes {
		in := fe.Func.Parameters[index]

		in.VarName = gogentools.NewNameWithPrefix(feparser.NewLowerTitleName("body", in.TypeName))
	}

	varNames := make([]string, 0)
	for _, index := range indexes {
		in := fe.Func.Parameters[index]

		varNames = append(varNames, in.VarName)
	}

	code := BlockFunc(
		func(groupCase *Group) {

			for _, index := range indexes {
				in := fe.Func.Parameters[index]

				ComposeTypeAssertion(file, groupCase, in.VarName, in.GetOriginal().GetType(), in.GetOriginal().IsVariadic())
			}

			groupCase.Var().Id("rece").Qual(fe.Receiver.PkgPath, fe.Receiver.TypeName)

			gogentools.ImportPackage(file, fe.Func.PkgPath, fe.Func.PkgName)

			groupCase.Id("rece").Dot(fe.Func.Name).CallFunc(
				func(call *Group) {

					tpFun := fe.Func.GetOriginal().GetType().(*types.Signature)

					zeroVals := gogentools.ScanTupleOfZeroValues(file, tpFun.Params(), fe.Func.GetOriginal().IsVariadic())

					for i, zero := range zeroVals {
						isConsidered := IntSliceContains(indexes, i)
						if isConsidered {
							call.Id(fe.Func.Parameters[i].VarName)
						} else {
							call.Add(zero)
						}
					}

				},
			).Add(Tag(TagContentType(guessContentTypeFromFuncName(fe.Func.Name)), TagResponseBody(varNames...)))

		})
	return code
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// declare `name := source(1).(Type)`
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