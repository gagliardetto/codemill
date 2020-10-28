package main

import (
	"fmt"
	"go/types"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
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

var (
	spec = &XSpec{
		Name:    "HelloWorldModule",
		Classes: make(map[string]*XClass),
	}
	specMu = &sync.RWMutex{}
)

type ClassProto struct {
	Extends []string
}

var (
	// class kinds:
	classes = map[string]*ClassProto{
		"UntrustedFlowSource": {
			Extends: []string{"UntrustedFlowSource::Range"},
		},
	}
)

type XSpec struct {
	Name    string // Name of the module
	Classes map[string]*XClass
}

type XClass struct {
	Name      string
	IsPrivate bool
	Extends   []string
	Methods   map[string]*XMethod
}

type XMethod struct {
	Name      string
	Selectors []Selector
}

type Selector interface {
	IsSelf() bool
}

func main() {
	r := gin.Default()
	r.StaticFile("", "./index.html")
	r.Static("/static", "./static")
	httpClient := new(http.Client)

	r.GET("/api/spec", func(c *gin.Context) {
		specMu.RLock()
		defer specMu.RUnlock()
		c.IndentedJSON(200, spec)
	})
	r.POST("/api/spec/classes", func(c *gin.Context) {
		var addClassPayload struct {
			Name      string
			IsPrivate bool
			Kind      string
		}
		err := c.BindJSON(&addClassPayload)
		if err != nil {
			Q(err)
			Abort400(c, err.Error())
			return
		}
		specMu.Lock()
		defer specMu.Unlock()

		addClassPayload.Name = ToCamel(addClassPayload.Name)
		if len(addClassPayload.Name) == 0 {
			Abort400(c, "Class name not valid")
			return
		}

		_, ok := spec.Classes[addClassPayload.Name]
		if ok {
			Abort400(c, "Class with the provided name already exists")
			return
		}

		proto, ok := classes[addClassPayload.Kind]
		if !ok {
			Abort400(c, "Kind not found")
			return
		}

		created := &XClass{
			Name:      addClassPayload.Name,
			IsPrivate: addClassPayload.IsPrivate,
			Extends:   make([]string, len(proto.Extends)),
			Methods:   make(map[string]*XMethod),
		}

		copy(created.Extends, proto.Extends)

		spec.Classes[addClassPayload.Name] = created
		c.IndentedJSON(200, spec)
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
			c.IndentedJSON(200, M{"results": []string{runtime.Version()}})
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
		sort.Sort(sort.Reverse(sort.StringSlice(versions)))

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
			// TODO: if version not specified, use latest?
			// TODO: if package is std, specifying a version is not needed.
			Abort400(c, "`v` (version) parameter not specified")
			return
		}

		Infof("Loading package %q", path+"@"+version)

		isStd := search.IsStandardImportPath(path)
		if isStd {
			Infof("Package %q is part of standard library", path)
		}

		var rootPath string
		if isStd {
			rootPath = path
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

			pks, err := sc.ScanWithCustomScanner(scanner.ScannerFunc(scannerFunc))
			if err != nil {
				Q(err)
				Abort400(c, Sf("Errors occurred while loading %q: %s.", path, err))
				return
			}
			pk := pks[0]

			// compose the fePackage:
			Infof("Composing fePackage %q", pk.Path)
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

	r.Run("0.0.0.0:8070") // listen and serve on 0.0.0.0:8080
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

func x() {
	r := gin.Default()
	r.POST("/api/x/module", func(c *gin.Context) {
		// NOTE: this is EXPERIMENTAL and does not currently work.
		// TODO: see https://github.com/golang/go/issues/33655 for a possible approach.

		path := c.Query("path")
		version := c.Query("v")

		isStd := search.IsStandardImportPath(path)
		if isStd {
			Abort400(c, Sf("Package %q is from the standard library", path))
			return
		}

		// Find out the root of the package:
		root, err := get.RepoRootForImportPath(path, get.IgnoreMod, web.DefaultSecurity)
		if err != nil {
			panic(err)
		}
		Q(root)
		path = root.Root

		// Lookup the repo:
		repo, err := modfetch.Lookup(proxy, path)
		if err != nil {
			panic(err)
		}
		_ = repo
		modfileBytes, err := repo.GoMod(version)
		if err != nil {
			panic(err)
		}

		mf, err := modfile.Parse("go.mod", modfileBytes, nil)
		if err != nil {
			panic(err)
		}

		var args []string
		for _, v := range mf.Require {
			args = append(args, v.Mod.Path)
			Q(modfetch.Download(v.Mod))
		}

		Q(mf)

		mod := module.Version{
			Path:    path,
			Version: version,
		}

		modfetch.PkgMod = os.ExpandEnv("$GOPATH/pkg/mod")

		gotPath, err := modfetch.Download(mod)
		if err != nil {
			panic(err)
		}
		Q(gotPath, err)

		{
			config := &packages.Config{
				Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
					packages.NeedImports | packages.NeedDeps | packages.NeedExportsFile |
					packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedTypesSizes | packages.NeedModule,
			}
			pkgs, err := packages.Load(config, path)
			if err != nil {
				panic(err)
			}
			Ln("parsed")
			packages.PrintErrors(pkgs)

			for _, pkg := range pkgs {
				Q(pkg.Module)
			}
			for _, pkg := range pkgs {
				fmt.Println(pkg.ID, pkg.GoFiles)
			}
		}
	})
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
	sourceCache   = make(map[string]*feparser.FEPackage)
	sourceCacheMu = &sync.RWMutex{}
)

func GetCachedSource(path string, version string) *feparser.FEPackage {
	sourceCacheMu.RLock()
	defer sourceCacheMu.RUnlock()
	got, ok := sourceCache[path+"@"+version]
	if !ok {
		return nil
	}
	return got
}

func SetCachedSource(path string, version string, pkg *feparser.FEPackage) {
	sourceCacheMu.Lock()
	defer sourceCacheMu.Unlock()
	sourceCache[path+"@"+version] = pkg
}
