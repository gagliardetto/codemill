package redirect

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
	InlineExpectationsTestTag = "$redirectUrl" // Must start with a $ sign.
)

func Tag(vals ...string) Code {
	tg := ""
	for i, v := range vals {
		if i > 0 {
			tg += " "
		}
		tg += InlineExpectationsTestTag + "=" + v
	}
	return Comment(tg)
}

const (
	TestQueryContent = `
import go
import TestUtilities.InlineExpectationsTest

class HttpRedirectTest extends InlineExpectationsTest {
  HttpRedirectTest() { this = "HttpRedirectTest" }

  override string getARelevantTag() { result = "redirectUrl" }

  override predicate hasActualResult(string file, int line, string element, string tag, string value) {
    tag = "redirectUrl" and
    exists(HTTP::Redirect rd |
      rd.hasLocationInfo(file, line, _, _, _) and
      element = rd.getUrl().toString() and
      value = rd.getUrl().toString()
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

	// Assuming the validation has already been done:
	methodGetURL := mdl.Methods[0]

	if len(methodGetURL.Selectors) == 0 {
		Infof("No selectors found for %q method.", methodGetURL.Name)
		return nil
	}

	allPathVersions := mdl.ListAllPathVersions()

	file := NewTestFile(GenerateBoilerplate)

	for _, pathVersion := range allPathVersions {
		if !allInOneFile {
			// Reset file:
			file = NewTestFile(GenerateBoilerplate)
		}
		codez := make([]Code, 0)

		b2fe, b2tm, b2itm, err := x.GroupFuncSelectors(methodGetURL)
		if err != nil {
			Fatalf("Error while GroupFuncSelectors: %s", err)
		}

		{
			cont, ok := b2fe[pathVersion]
			if ok && x.HasValidPos(cont...) {
				addedCount := 0
				code := BlockFunc(
					func(groupCase *Group) {

						for _, funcQual := range cont {
							fn := x.GetFuncByQualifier(funcQual)
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
								addedCount++
							}

						}
					})
				if addedCount > 0 {
					codez = append(codez,
						Comment("Redirect via function call.").
							Line().
							Add(code),
					)
				}
			}
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
						Commentf("Redirect via method calls on %s.", typ.QualifiedName).
							Line().
							Add(code),
					)
				})
			if len(codezTypeMethods) > 0 {
				codez = append(codez,
					Comment("Redirect via method calls.").
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
						Commentf("Redirect via method calls on %s interface.", typ.QualifiedName).
							Line().
							Add(code),
					)
				})

			if len(codezIfaceMethods) > 0 {
				codez = append(codez,
					Comment("Redirect via interface method calls.").
						Line().
						Block(codezIfaceMethods...),
				)
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

		in.VarName = gogentools.NewNameWithPrefix(feparser.NewLowerTitleName("url", in.TypeName))
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
			).Add(Tag(varNames...))

		})
	return code
}
func generate_Method(file *File, fe *feparser.FETypeMethod, indexes []int) *Statement {

	for _, index := range indexes {
		in := fe.Func.Parameters[index]

		in.VarName = gogentools.NewNameWithPrefix(feparser.NewLowerTitleName("url", in.TypeName))
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

			Comments(groupCase, "Declare medium object/interface:")
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
			).Add(Tag(varNames...))

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
