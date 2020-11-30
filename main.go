package main

import (
	"errors"
	"flag"
	"fmt"
	"go/types"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/gagliardetto/codebox/scanner"
	"github.com/gagliardetto/codemill/handlers/untrustedflowsource"
	"github.com/gagliardetto/codemill/x"
	"github.com/gagliardetto/feparser"
	"github.com/gagliardetto/golang-go/cmd/go/not-internal/get"
	"github.com/gagliardetto/golang-go/cmd/go/not-internal/modfetch"
	"github.com/gagliardetto/golang-go/cmd/go/not-internal/search"
	"github.com/gagliardetto/golang-go/cmd/go/not-internal/web"
	"github.com/gagliardetto/request"
	. "github.com/gagliardetto/utilz"
	"github.com/gin-gonic/gin"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

type M map[string]interface{}

const (
	// Use default Golang proxy (???)
	proxy = "https://proxy.golang.org/"
)

var (
	globalSpec *x.XSpec
)

func main() {
	r := gin.Default()
	r.StaticFile("", "./index.html")
	r.Static("/static", "./static")
	httpClient := new(http.Client)

	var specFilepath string
	var outDir string
	flag.StringVar(&specFilepath, "spec", "", "Path to spec file; file will be created if not already existing.")
	flag.StringVar(&outDir, "dir", "", "Path to dir where to save generated files.")
	flag.Parse()

	if specFilepath == "" {
		// specFilepath is ALWAYS necessary,
		// either for knowing from where to load a spec,
		// or where to save a created one.
		panic("--spec flag not provided")
	}

	if outDir == "" {
		panic("--dir flag not provided")
	}

	{
		// Initialize ModelKind router:
		rt, err := x.InitRouter(&x.ModelKindRouterConfig{
			Dir: outDir,
		})
		if err != nil {
			panic(fmt.Errorf("error while intializing ModelKind router: %s", err))
		}
		{
			// Register ModelKind handlers:
			rt.RegisterHandler(untrustedflowsource.Kind, &untrustedflowsource.Handler{})
		}
	}

	if MustFileExists(specFilepath) {
		// If the file exists, try loading the spec:
		spec, err := x.TryLoadSpecFromFile(specFilepath, LoadPackage)
		if err != nil {
			panic(err)
		}
		globalSpec = spec
	} else {
		// If the file does NOT exist,
		// create a new spec named after the filename:
		name := ToCamel(TrimExt(filepath.Base(specFilepath)))
		if name == "" {
			name = "DefaultSpec"
		}
		globalSpec = x.NewXSpecWithName(name)
	}

	onExitCallback := func() {
		globalSpec.Lock()
		defer globalSpec.Unlock()

		globalSpec.RemoveMeta()

		Infof("Saving spec to %q", MustAbs(specFilepath))
		err := SaveAsJSON(globalSpec, specFilepath)
		if err != nil {
			panic(err)
		}
		// NOTE: after this point, any modification to globalSpec will be volatile,
		// i.e. discarded the instant this program hits os.Exit.

		// Sort stuff for visual convenience in the generated code:
		globalSpec.Sort()

		// Create output dir if it doesn't exist:
		MustCreateFolderIfNotExists(outDir, 0750)

		for _, modelSpec := range globalSpec.Models {

			// Handle generation:
			err := x.Router().Handle(modelSpec.Kind, modelSpec)
			if err != nil {
				Fatalf(
					"Error while handling model %q: %s",
					modelSpec.Name,
					err,
				)
			}

		}

		os.Exit(0)
	}

	var once sync.Once
	go Notify(
		func(os.Signal) bool {
			once.Do(onExitCallback)
			return false
		},
		os.Kill,
		os.Interrupt,
	)
	defer once.Do(onExitCallback)

	r.GET("/api/spec", func(c *gin.Context) {
		globalSpec.RLock()
		defer globalSpec.RUnlock()
		c.IndentedJSON(200, globalSpec)
	})

	r.GET("/api/cached", func(c *gin.Context) {
		// List already cached sources:
		list := x.GetListCachedSources()

		sort.Slice(list, func(i, j int) bool {
			return x.FormatPathVersion(list[i].Path, list[i].Version) < x.FormatPathVersion(list[j].Path, list[j].Version)
		})
		c.IndentedJSON(200, M{"results": list})
	})

	r.GET("/api/models/kinds", func(c *gin.Context) {
		// List available model kinds:
		kinds := x.Router().ListModelKinds()
		c.IndentedJSON(200, M{"results": kinds})
	})

	r.POST("/api/spec/models", func(c *gin.Context) {
		// Add a new model to the spec:
		var req struct {
			Name string
			Kind x.ModelKind
		}
		err := c.BindJSON(&req)
		if err != nil {
			Q(err)
			Abort400(c, err.Error())
			return
		}

		req.Name = ToCamel(req.Name)
		if len(req.Name) == 0 {
			Abort400(c, "Class name not valid")
			return
		}

		created := &x.XModel{
			Name: req.Name,
			Kind: req.Kind,
		}

		err = globalSpec.PushModel(created)
		if err != nil {
			Abort400(c, Sf("Error adding model: %s", err))
			return
		}
		c.IndentedJSON(200, globalSpec)
	})

	r.PATCH("/api/spec/structs", func(c *gin.Context) {
		// Patch a struct, i.e. add/remove a field:
		var req struct {
			Where struct {
				Path    string
				Version string
				Model   string
				Method  string
			}
			What struct {
				StructID string
				FieldID  string
				Value    bool
			}
		}
		err := c.BindJSON(&req)
		if err != nil {
			Q(err)
			Abort400(c, err.Error())
			return
		}

		source := x.GetCachedSource(req.Where.Path, req.Where.Version)
		if source == nil {
			Abort404(c, Sf("Source not found: %s@%s", req.Where.Path, req.Where.Version))
			return
		}
		// Make sure that the struct exist:
		st := x.FindStructByID(source, req.What.StructID)
		if st == nil {
			Abort404(c, Sf("Struct not found: %q", req.What.StructID))
			return
		}
		fld := x.FindFieldByID(st, req.What.FieldID)
		if fld == nil {
			Abort404(c, Sf("Field not found: %q", req.What.FieldID))
			return
		}

		err = globalSpec.ModifyModelByName(
			req.Where.Model,
			func(mdl *x.XModel) error {
				err := mdl.ModifyMethodByName(
					req.Where.Method,
					func(mt *x.XMethod) error {

						existingSel := mt.GetStructSelector(
							req.Where.Path,
							req.Where.Version,
							req.What.StructID,
						)
						if existingSel == nil {
							// Add a new selector only if the value is true:
							if req.What.Value == true {
								// If there is no existing selector for the struct,
								// then create a new one:
								newSel := &x.XSelector{
									Kind: x.SelectorKindStruct,
									Qualifier: &x.StructQualifier{
										BasicQualifier: x.BasicQualifier{
											Path:    req.Where.Path,
											Version: req.Where.Version,
											ID:      req.What.StructID,
										},
										TypeName: st.TypeName,
										Fields: map[string]*x.FieldMeta{
											fld.VarName: {
												Name:       fld.VarName,
												TypeString: fld.TypeString,
												KindString: fld.KindString,
											},
										},
										Total: len(st.Fields),
										Left:  len(st.Fields) - 1,
									},
								}

								mt.Selectors = append(mt.Selectors, newSel)
							}
						} else {
							if req.What.Value {
								// Enable field:
								existingSel.Fields[fld.VarName] = &x.FieldMeta{
									Name:       fld.VarName,
									TypeString: fld.TypeString,
									KindString: fld.KindString,
								}
							} else {
								// Remove field:
								delete(existingSel.Fields, fld.VarName)
							}

							{ // Update counts:
								existingSel.Total = len(st.Fields)
								existingSel.Left = len(st.Fields) - len(existingSel.Fields)
							}

							if len(existingSel.Fields) == 0 {
								// If all fields are disabled, then remove the selector:
								mt.DeleteStructSelector(
									req.Where.Path,
									req.Where.Version,
									req.What.StructID,
								)
							}
						}

						return nil
					},
				)
				if err != nil {
					return err
				}
				return nil
			},
		)
		if err != nil {
			Abort400(c, Sf("Error modifying model: %s", err))
			return
		}

		c.IndentedJSON(200, globalSpec)
	})

	r.PATCH("/api/spec/funcs", func(c *gin.Context) {
		// Patch a func (func/type-method/interface-method), i.e. select/unselect its components:
		var req struct {
			Where struct {
				Path    string
				Version string
				Model   string
				Method  string
			}
			What struct {
				FuncID string
				Index  int
				Value  bool
			}
		}
		err := c.BindJSON(&req)
		if err != nil {
			Q(err)
			Abort400(c, err.Error())
			return
		}

		source := x.GetCachedSource(req.Where.Path, req.Where.Version)
		if source == nil {
			Abort404(c, Sf("Source not found: %s@%s", req.Where.Path, req.Where.Version))
			return
		}
		// Find the func/type-method/interface-method:
		fn := x.FindFuncByID(source, req.What.FuncID)
		if fn == nil {
			Abort404(c, Sf("Func not found: %q", req.What.FuncID))
			return
		}

		if req.What.Index >= fn.Len() {
			Abort400(c, Sf("Index out of bounds: index=%v, but v.Len() = %v", req.What.Index, fn.Len()))
			return
		}

		err = globalSpec.ModifyModelByName(
			req.Where.Model,
			func(mdl *x.XModel) error {
				err := mdl.ModifyMethodByName(
					req.Where.Method,
					func(mt *x.XMethod) error {

						existingSel := mt.GetFuncSelector(
							req.Where.Path,
							req.Where.Version,
							req.What.FuncID,
						)
						meta := x.CompileFuncQualifierElementsMeta(fn)
						if existingSel == nil {
							// Add a new selector only if the value is true:
							if req.What.Value {
								pos := make([]bool, fn.Len())
								pos[req.What.Index] = req.What.Value

								// If there is no existing selector,
								// then create a new one:
								newSel := &x.XSelector{
									Kind: x.SelectorKindFunc,
									Qualifier: &x.FuncQualifier{
										BasicQualifier: x.BasicQualifier{
											Path:    req.Where.Path,
											Version: req.Where.Version,
											ID:      req.What.FuncID,
										},
										Pos:      pos,
										Name:     x.GetFuncName(fn),
										Elements: meta,
									},
								}

								mt.Selectors = append(mt.Selectors, newSel)
							}
						} else {
							existingSel.Pos[req.What.Index] = req.What.Value
							existingSel.Elements = meta

							if AllFalse(existingSel.Pos...) {
								// If all false, then remove the selector:
								mt.DeleteFuncSelector(
									req.Where.Path,
									req.Where.Version,
									req.What.FuncID,
								)
							}
						}

						return nil
					},
				)
				if err != nil {
					return err
				}
				return nil
			},
		)
		if err != nil {
			Abort400(c, Sf("Error modifying model: %s", err))
			return
		}

		c.IndentedJSON(200, globalSpec)
	})

	r.PATCH("/api/spec/types", func(c *gin.Context) {
		// Patch a type selector:
		var req struct {
			Where struct {
				Path    string
				Version string
				Model   string
				Method  string
			}
			What struct {
				TypeID string
				Value  bool
			}
		}
		err := c.BindJSON(&req)
		if err != nil {
			Q(err)
			Abort400(c, err.Error())
			return
		}

		source := x.GetCachedSource(req.Where.Path, req.Where.Version)
		if source == nil {
			Abort404(c, Sf("Source not found: %s@%s", req.Where.Path, req.Where.Version))
			return
		}
		// Find the type:
		typ := x.FindTypeByID(source, req.What.TypeID)
		if typ == nil {
			Abort404(c, Sf("Type not found: %q", req.What.TypeID))
			return
		}

		err = globalSpec.ModifyModelByName(
			req.Where.Model,
			func(mdl *x.XModel) error {
				err := mdl.ModifyMethodByName(
					req.Where.Method,
					func(mt *x.XMethod) error {

						existingSel := mt.GetTypeSelector(
							req.Where.Path,
							req.Where.Version,
							req.What.TypeID,
						)
						if existingSel == nil {
							// Add a new selector only if the value is true:
							if req.What.Value {
								// If there is no existing selector,
								// then create a new one:
								newSel := &x.XSelector{
									Kind: x.SelectorKindType,
									Qualifier: &x.TypeQualifier{
										BasicQualifier: x.BasicQualifier{
											Path:    req.Where.Path,
											Version: req.Where.Version,
											ID:      req.What.TypeID,
										},
										TypeName:   typ.TypeName,
										KindString: typ.KindString,
										Value:      true,
									},
								}

								mt.Selectors = append(mt.Selectors, newSel)
							}
						} else {
							if req.What.Value == false {
								// If false, then remove the selector:
								mt.DeleteTypeSelector(
									req.Where.Path,
									req.Where.Version,
									req.What.TypeID,
								)
							}
						}

						return nil
					},
				)
				if err != nil {
					return err
				}
				return nil
			},
		)
		if err != nil {
			Abort400(c, Sf("Error modifying model: %s", err))
			return
		}

		c.IndentedJSON(200, globalSpec)
	})

	r.GET("/api/search", func(c *gin.Context) {
		// Search packages on godoc:
		req := request.NewRequest(httpClient)

		query := c.Query("q")
		searchURL := "https://api.godoc.org/search?q=" + url.QueryEscape(query)
		Ln(searchURL)
		resp, err := req.Get(searchURL)
		if err != nil {
			Q(err)
			Abort400(c, err.Error())
			return
		}
		j, err := resp.Json()
		if err != nil {
			Q(err)
			Abort400(c, err.Error())
			return
		}
		defer resp.Body.Close() // Don't forget close the response body

		c.IndentedJSON(200, j.Interface())
	})

	r.GET("/api/versions", func(c *gin.Context) {
		// List versions of a package:

		path := c.Query("path")

		isStd := search.IsStandardImportPath(path)
		if isStd {
			// Return the current Go version:
			// TODO: get the version
			c.IndentedJSON(200, M{"results": []string{"local"}})
			return
		}

		// Find out the root of the package:
		root, err := get.RepoRootForImportPath(path, get.IgnoreMod, web.DefaultSecurity)
		if err != nil {
			Q(err)
			Abort400(c, err.Error())
			return
		}
		Q(root)
		path = root.Root

		// Lookup the repo:
		repo, err := modfetch.Lookup(proxy, path)
		if err != nil {
			Q(err)
			Abort400(c, err.Error())
			return
		}

		Ln(repo.ModulePath())
		Q(repo.Stat(""))

		prefix := ""
		// Get list of versions:
		versions, err := repo.Versions(prefix)
		if err != nil {
			Q(err)
			Abort400(c, err.Error())
			return
		}

		// Reverse versions' order to show the (presumably) most recent at the top of the list:
		ReverseStringSlice(versions)

		Ln(versions)

		// If no versions found, then get latest commit:
		if len(versions) == 0 {
			latest, err := repo.Latest()
			if err != nil {
				Q(err)
				Abort400(c, err.Error())
				return
			}
			Q(latest)
			versions = []string{latest.Version}
		}
		c.IndentedJSON(200, M{"results": versions})
	})

	r.GET("/api/source", func(c *gin.Context) {
		// Retrieve and parse the specified package.

		path := c.Query("path")
		version := c.Query("v")

		if path == "" {
			Abort400(c, "`path` parameter not specified")
			return
		}
		if version == "" {
			// If version not specified, we'll use the latest.
		}
		pkg, err := LoadPackage(path, version)
		if err != nil {
			Q(err)
			Abort400(c, err.Error())
			return
		}
		c.IndentedJSON(200, pkg)

	})

	r.Run("0.0.0.0:8070")
}

func LoadPackage(path string, version string) (*feparser.FEPackage, error) {

	if path == "" {
		return nil, errors.New("path not specified")
	}
	if version == "" {
		// If version not specified, we'll use the latest.
	}

	Infof(ShakespeareBG("Loading package %q"), path+"@"+version)

	isStd := search.IsStandardImportPath(path)
	if isStd {
		Infof("Package %q is part of standard library", path)
	}

	var rootPath string
	if isStd {
		rootPath = path
		version = "local"
	} else {
		// Find out the root of the package:
		root, err := get.RepoRootForImportPath(path, get.IgnoreMod, web.DefaultSecurity)
		if err != nil {
			return nil, err
		}
		//Q(root)
		Infof(
			"Package %q has root %q",
			path+"@"+version,
			root.Root,
		)
		rootPath = root.Root
	}

	if !isStd {
		// Lookup the repo:
		repo, err := modfetch.Lookup(proxy, rootPath)
		if err != nil {
			return nil, err
		}

		if version == "" {
			// If version not specified, we'll use the latest.
			Infof("no version specified; using latest")
			latest, err := repo.Latest()
			if err != nil {
				return nil, err
			}
			Q(latest)
			version = latest.Version
		}

		rev, err := repo.Stat(version)
		if err != nil {
			return nil, err
		}
		Q(rev)
	}

	cached := x.GetCachedSource(path, version)
	if cached != nil {
		return cached, nil
	}

	config := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedDeps | packages.NeedExportsFile |
			packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedTypesSizes | packages.NeedModule,
	}
	{
		// Create a temporary folder:
		tmpDir, err := ioutil.TempDir("", "codemill")
		if err != nil {
			return nil, err
		}
		//defer os.RemoveAll(tmpDir)
		tmpDir = MustAbs(tmpDir)
		//Q(tmpDir)

		// Create a `go.mod` file requiring the specified version of the package:
		mf := &modfile.File{}
		mf.AddModuleStmt("example.com/hello/world")

		if !isStd {
			mf.AddNewRequire(rootPath, version, true)
		}
		mf.Cleanup()

		mfBytes, err := mf.Format()
		if err != nil {
			return nil, err
		}
		// Write `go.mod` file:
		err = ioutil.WriteFile(filepath.Join(tmpDir, "go.mod"), mfBytes, 0666)
		if err != nil {
			return nil, err
		}
		//Ln(string(mfBytes))

		// Set the package loader Dir to the `tmpDir`; that will force
		// the package loader to use the `go.mod` file and thus
		// load the wanted version of the package:
		config.Dir = tmpDir
		// NOTE: Why /api/source?path=github.com/revel/revel/testing&v=v0.9.1 gets the github.com/revel/revel@v1.0.0/testing ???
	}

	// Initialize scanner:
	sc, err := scanner.New(path)
	if err != nil {
		return nil, err
	}

	scannerFunc := func(path string) (*packages.Package, error) {
		// - If you set `config.Dir` to a dir that contains a `go.mod` file,
		// and a version of `path` package is specified in that `go.mod` file,
		// then that specific version will be parsed.
		// - You can have a temporary folder with only a `go.mod` file
		// that contains a reuire for the package+version you want, and
		// go will add the missing deps, and load that version you specified.
		pkgs, err := packages.Load(config, path)
		if err != nil {
			return nil, fmt.Errorf("error while packages.Load: %s", err)
		}
		Infof("Loaded package %q", path)

		var errs []error
		packages.Visit(pkgs, nil, func(pkg *packages.Package) {
			for _, err := range pkg.Errors {
				errs = append(errs, err)
			}
		})
		err = CombineErrors(errs...)
		if len(errs) > 0 {
			return nil, fmt.Errorf("error while packages.Load: %s", err)
		}

		for _, pkg := range pkgs {
			Q(pkg.Module)
		}
		for _, pkg := range pkgs {
			Sfln(
				"%s has %v files",
				pkg.ID,
				len(pkg.GoFiles),
			)
		}
		return pkgs[0], nil
	}

	Infof("Loading pkg=%q version=%q ...", path, version)
	pks, err := sc.ScanWithCustomScanner(scanner.ScannerFunc(scannerFunc))
	if err != nil {
		return nil, fmt.Errorf("Errors occurred while loading %q: %s.", path, err)
	}
	pk := pks[0]

	// compose the fePackage:
	Infof("Composing fePackage for pkg=%q version=%q ...", pk.Path, version)
	fePackage, err := feparser.Load(pk)
	if err != nil {
		return nil, err
	}

	fePackage.IsStandard = isStd

	x.SetCachedSource(path, version, fePackage)
	return fePackage, nil
}

// Interfaces returns a map of interfaces which are declared in the package.
func Interfaces(pkg *types.Package) map[string]*types.Interface {
	ifs := map[string]*types.Interface{}

	for _, name := range pkg.Scope().Names() {
		o := pkg.Scope().Lookup(name)
		if o != nil {
			i, ok := o.Type().Underlying().(*types.Interface)
			if ok {
				ifs[name] = i
			}
		}
	}

	return ifs
}

// Structs returns a map of structs which are declared in the package.
func Structs(pkg *types.Package) map[string]*types.Struct {
	structs := map[string]*types.Struct{}

	for _, name := range pkg.Scope().Names() {
		o := pkg.Scope().Lookup(name)
		if o != nil {
			s, ok := o.Type().Underlying().(*types.Struct)
			if ok {
				structs[name] = s
			}
		}
	}

	return structs
}

func Abort400(c *gin.Context, errorString string) {
	abort(c, 400, errorString)
}
func Abort404(c *gin.Context, errorString string) {
	abort(c, 404, errorString)
}
func abort(c *gin.Context, statusCode int, errorString string) {
	c.AbortWithStatusJSON(statusCode, M{"error": errorString})
}
