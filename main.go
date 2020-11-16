package main

import (
	"fmt"
	"go/types"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/gagliardetto/codebox/scanner"
	"github.com/gagliardetto/feparser"
	"github.com/gagliardetto/golang-go/cmd/go/not-internal/get"
	"github.com/gagliardetto/golang-go/cmd/go/not-internal/modfetch"
	"github.com/gagliardetto/golang-go/cmd/go/not-internal/search"
	"github.com/gagliardetto/golang-go/cmd/go/not-internal/web"
	"github.com/gagliardetto/request"
	. "github.com/gagliardetto/utilz"
	"github.com/gin-gonic/gin"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/tools/go/packages"
)

type M map[string]interface{}

const (
	// Use default Golang proxy (???)
	proxy = "https://proxy.golang.org/"
)

type ModelKind string

const (
	ModelKindUntrustedFlowSource ModelKind = "UntrustedFlowSource"
)

func IsValidModelKind(kind ModelKind) bool {
	return IsAnyOf(
		string(kind),
		// All the valid kinds:
		string(ModelKindUntrustedFlowSource),
	)
}

func NewScavengeMethods(kind ModelKind) []*XMethod {

	switch kind {
	case ModelKindUntrustedFlowSource:
		{
			return []*XMethod{
				{
					Name:      "_Self",
					IsSelf:    true,
					Selectors: []*Selector{},
				},
			}
		}
	default:
		panic(Sf("No default method scavenging for %q kind", kind))
	}
}

type XSpec struct {
	Name   string // Name of the module
	Models []*XModel
	*sync.RWMutex
}

//
func (spec *XSpec) HasModelName(name string) bool {
	spec.RLock()
	defer spec.RUnlock()

	for _, md := range spec.Models {
		if md.Name == name {
			return true
		}
	}
	return false
}

//
func (spec *XSpec) PushModel(model *XModel) error {
	{ // Validate model before adding:

		ok := spec.HasModelName(model.Name)
		if ok {
			return fmt.Errorf("Class with the provided name already exists: %q", model.Name)
		}

		valid := IsValidModelKind(model.Kind)
		if !valid {
			return fmt.Errorf("Model Kind not valid: %q", model.Kind)
		}
	}

	model.Methods = NewScavengeMethods(model.Kind)
	spec.Lock()
	defer spec.Unlock()
	spec.Models = append(spec.Models, model)
	return nil
}

type XModel struct {
	Name      string
	Kind      ModelKind
	IsPrivate bool
	Methods   []*XMethod
}

type XMethod struct {
	Name      string
	IsSelf    bool
	Selectors []*Selector
}

type SelectorKind string

const (
	SelectorKindStruct SelectorKind = "Struct" // Qualifier for structs.
	SelectorKindFunc   SelectorKind = "Func"   // Qualifier for funcs, methods, interfaces.
)

type Selector struct {
	Kind      SelectorKind
	Qualifier interface{}
}
type Qualifier struct {
	Path    string
	Version string
	ID      string
}
type StructQualifier struct {
	Qualifier
	TypeName string
	Fields   map[string]bool
}

//
func (sel *Selector) GetFieldQualifier() *StructQualifier {
	got, ok := sel.Qualifier.(*StructQualifier)
	if !ok {
		return nil
	}
	return got
}

type FuncQualifier struct {
	Qualifier
	Pos []bool
}

//
func (sel *Selector) GetFuncQualifier() *FuncQualifier {
	got, ok := sel.Qualifier.(*FuncQualifier)
	if !ok {
		return nil
	}
	return got
}

var (
	// TODO: try loading spec from file.
	globalSpec = &XSpec{
		Name: "HelloWorldModule",
		Models: []*XModel{
			{
				Name: "UntrustedSource",
				Kind: ModelKindUntrustedFlowSource,
				Methods: []*XMethod{
					{
						Name:   "_Self",
						IsSelf: true,
						Selectors: []*Selector{
							{
								Kind: SelectorKindStruct,
								Qualifier: &StructQualifier{
									Qualifier: Qualifier{
										Path:    "github.com/aws/aws-sdk-go/aws",
										Version: "v1.9.44",
										ID:      "Struct-Config",
									},
									TypeName: "Config",
									Fields: map[string]bool{
										"Endpoint": true,
									},
								},
							},
							{
								Kind: SelectorKindFunc,
								Qualifier: &FuncQualifier{
									Qualifier: Qualifier{
										Path:    "github.com/aws/aws-sdk-go/aws",
										Version: "v1.9.44",
										ID:      "Type-Method-Config-WithRegion",
									},
									Pos: []bool{
										false, false, true,
									},
								},
							},
						},
					},
				},
			},
		},
		RWMutex: &sync.RWMutex{},
	}
)

func main() {
	r := gin.Default()
	r.StaticFile("", "./index.html")
	r.Static("/static", "./static")
	httpClient := new(http.Client)

	r.GET("/api/spec", func(c *gin.Context) {
		globalSpec.RLock()
		defer globalSpec.RUnlock()
		c.IndentedJSON(200, globalSpec)
	})

	r.GET("/api/cached", func(c *gin.Context) {
		// List already cached sources:
		list := GetListCachedSources()

		sort.Slice(list, func(i, j int) bool {
			return FormatPathVersion(list[i].Path, list[i].Version) < FormatPathVersion(list[j].Path, list[j].Version)
		})
		c.IndentedJSON(200, M{"results": list})
	})

	r.GET("/api/models/kinds", func(c *gin.Context) {
		kinds := []ModelKind{ModelKindUntrustedFlowSource}
		c.IndentedJSON(200, M{"results": kinds})
	})

	r.POST("/api/spec/models", func(c *gin.Context) {
		var req struct {
			Name      string
			Kind      ModelKind
			IsPrivate bool
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

		created := &XModel{
			Name:      req.Name,
			Kind:      req.Kind,
			IsPrivate: req.IsPrivate,
		}

		err = globalSpec.PushModel(created)
		if err != nil {
			Abort400(c, Sf("Error adding model: %s", err))
			return
		}
		c.IndentedJSON(200, globalSpec)
	})

	r.GET("/api/search", func(c *gin.Context) {
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

		Infof("Loading package %q", path+"@"+version)

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
				Q(err)
				Abort400(c, err.Error())
				return
			}
			Q(root)
			rootPath = root.Root
		}

		if !isStd {
			// Lookup the repo:
			repo, err := modfetch.Lookup(proxy, rootPath)
			if err != nil {
				Q(err)
				Abort400(c, err.Error())
				return
			}

			if version == "" {
				// If version not specified, we'll use the latest.
				Infof("no version specified; using latest")
				latest, err := repo.Latest()
				if err != nil {
					Q(err)
					Abort400(c, err.Error())
					return
				}
				Q(latest)
				version = latest.Version
			}

			rev, err := repo.Stat(version)
			if err != nil {
				Q(err)
				if strings.Contains(err.Error(), "invalid version: unknown revision") {
					// TODO: cleanup
					e, ok := err.(*module.ModuleError)
					if ok {
						wE, ok := e.Err.(*web.HTTPError)
						if ok {
							Abort404(c, wE.Detail)
							return
						}
						iVE, ok := e.Err.(*module.InvalidVersionError)
						if ok {
							Abort404(c, iVE.Error())
							return
						}
					}
					Abort400(c, err.Error())
					return
				} else {
					Abort400(c, err.Error())
					return
				}
			}
			Q(rev)
		}

		cached := GetCachedSource(path, version)
		if cached != nil {
			c.IndentedJSON(200, cached)
			return
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
				Q(err)
				Abort400(c, err.Error())
				return
			}
			//defer os.RemoveAll(tmpDir)
			tmpDir = MustAbs(tmpDir)
			Q(tmpDir)

			// Create a `go.mod` file requiring the specified version of the package:
			mf := &modfile.File{}
			mf.AddModuleStmt("example.com/hello/world")

			if !isStd {
				mf.AddNewRequire(rootPath, version, true)
			}
			mf.Cleanup()

			mfBytes, err := mf.Format()
			if err != nil {
				Q(err)
				Abort400(c, err.Error())
				return
			}
			// Write `go.mod` file:
			err = ioutil.WriteFile(filepath.Join(tmpDir, "go.mod"), mfBytes, 0666)
			if err != nil {
				Q(err)
				Abort400(c, err.Error())
				return
			}
			Ln(string(mfBytes))

			// Set the package loader Dir to the `tmpDir`; that will force
			// the package loader to use the `go.mod` file and thus
			// load the wanted version of the package:
			config.Dir = tmpDir
			// NOTE: Why /api/source?path=github.com/revel/revel/testing&v=v0.9.1 gets the github.com/revel/revel@v1.0.0/testing ???
		}

		{
			// Initialize scanner:
			sc, err := scanner.New(path)
			if err != nil {
				Q(err)
				Abort400(c, err.Error())
				return
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
					fmt.Println(pkg.ID, pkg.GoFiles)
				}
				return pkgs[0], nil
			}

			Infof("Loading pkg=%q version=%q ...", path, version)
			pks, err := sc.ScanWithCustomScanner(scanner.ScannerFunc(scannerFunc))
			if err != nil {
				Q(err)
				Abort400(c, Sf("Errors occurred while loading %q: %s.", path, err))
				return
			}
			pk := pks[0]

			// compose the fePackage:
			Infof("Composing fePackage for pkg=%q version=%q ...", pk.Path, version)
			fePackage, err := feparser.Load(pk)
			if err != nil {
				Q(err)
				Abort400(c, err.Error())
				return
			}

			fePackage.IsStandard = isStd

			SetCachedSource(path, version, fePackage)
			c.IndentedJSON(200, fePackage)
		}

	})

	r.Run("0.0.0.0:8070")
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

var (
	sourceCache   = make(map[PathVersion]*feparser.FEPackage)
	sourceCacheMu = &sync.RWMutex{}
)

type PathVersion struct {
	Path    string
	Version string
}

func FormatPathVersion(path string, version string) string {
	return path + "@" + version
}

func GetListCachedSources() []PathVersion {
	list := make([]PathVersion, 0)

	sourceCacheMu.RLock()
	defer sourceCacheMu.RUnlock()

	for key := range sourceCache {
		list = append(list, key)
	}

	return list
}

func GetCachedSource(path string, version string) *feparser.FEPackage {
	sourceCacheMu.RLock()
	defer sourceCacheMu.RUnlock()

	key := PathVersion{
		Path:    path,
		Version: version,
	}
	got, ok := sourceCache[key]
	if !ok {
		return nil
	}
	return got
}

func SetCachedSource(path string, version string, pkg *feparser.FEPackage) {
	sourceCacheMu.Lock()
	defer sourceCacheMu.Unlock()

	key := PathVersion{
		Path:    path,
		Version: version,
	}
	sourceCache[key] = pkg
}
