package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"sort"

	"github.com/gagliardetto/codemill/cmd/go/not-internal/get"
	"github.com/gagliardetto/codemill/cmd/go/not-internal/modfetch"
	"github.com/gagliardetto/codemill/cmd/go/not-internal/search"
	"github.com/gagliardetto/codemill/cmd/go/not-internal/web"
	"github.com/gagliardetto/request"
	. "github.com/gagliardetto/utilz"
	"github.com/gin-gonic/gin"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/tools/go/packages"
)

const (
	// Use default Golang proxy (???)
	proxy = "https://proxy.golang.org/"
)

func main() {
	r := gin.Default()
	r.StaticFile("", "./index.html")
	r.Static("/static", "./static")
	httpClient := new(http.Client)

	r.GET("/api/search", func(c *gin.Context) {
		req := request.NewRequest(httpClient)

		query := c.Query("q")
		searchURL := "https://api.godoc.org/search?q=" + url.QueryEscape(query)
		Ln(searchURL)
		resp, err := req.Get(searchURL)
		if err != nil {
			panic(err)
		}
		j, err := resp.Json()
		if err != nil {
			panic(err)
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
			panic(err)
		}
		Q(root)
		path = root.Root

		// Lookup the repo:
		repo, err := modfetch.Lookup(proxy, path)
		if err != nil {
			panic(err)
		}

		Ln(repo.ModulePath())
		Q(repo.Stat(""))

		{ // Get only the latest version:
			// TODO: return all versions (i.e. remove this) when I'll find a way to `packages.Load` a specific package version.
			var versions []string
			latest, err := repo.Latest()
			if err != nil {
				c.AbortWithStatusJSON(400, M{"error": Sf("Error getting latest version for %q: %s", path, err)})
				return
			}
			Q(latest)
			versions = []string{latest.Version}
			c.IndentedJSON(200, M{"results": versions})
			return
		}

		prefix := ""
		// Get list of versions:
		versions, err := repo.Versions(prefix)
		if err != nil {
			panic(err)
		}

		// Reverse versions' order to show the (presumably) most recent at the top of the list:
		sort.Sort(sort.Reverse(sort.StringSlice(versions)))

		Ln(versions)

		// If no versions found, then get latest commit:
		if len(versions) == 0 {
			latest, err := repo.Latest()
			if err != nil {
				panic(err)
			}
			Q(latest)
			versions = []string{latest.Version}
		}
		c.IndentedJSON(200, M{"results": versions})
	})

	r.POST("/api/code", func(c *gin.Context) {
		// Retrieve and parse the specified package.

		path := c.Query("path")
		version := c.Query("v")
		_ = version
		Infof("Loading package %q", path)

		isStd := search.IsStandardImportPath(path)
		if isStd {
			Infof("Package %q is part of standard library", path)
		}

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
			Infof("Loaded package %q", path)
			if packages.PrintErrors(pkgs) > 0 {
				c.AbortWithStatusJSON(400, M{"error": Sf("Errors occurred while loading %q; see server logs.", path)})
				return
			}

			for _, pkg := range pkgs {
				Q(pkg.Module)
			}
			for _, pkg := range pkgs {
				fmt.Println(pkg.ID, pkg.GoFiles)
			}
		}

	})

	r.Run() // listen and serve on 0.0.0.0:8080
}

type M map[string]interface{}

func x() {
	r := gin.Default()
	r.POST("/api/x/module", func(c *gin.Context) {
		// NOTE: this is EXPERIMENTAL and does not currently work.
		// TODO: see https://github.com/golang/go/issues/33655 for a possible approach.

		path := c.Query("path")
		version := c.Query("v")

		isStd := search.IsStandardImportPath(path)
		if isStd {
			c.AbortWithStatusJSON(400, M{"error": Sf("Package %q is from the standard library", path)})
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
