package main

import (
	"net/http"
	"net/url"

	"github.com/gagliardetto/codemill/cmd/go/not-internal/get"
	"github.com/gagliardetto/codemill/cmd/go/not-internal/modfetch"
	"github.com/gagliardetto/codemill/cmd/go/not-internal/web"
	"github.com/gagliardetto/request"
	. "github.com/gagliardetto/utilz"
	"github.com/gin-gonic/gin"
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
		// Use default Golang proxy (???)
		proxy := "https://proxy.golang.org/"

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

		prefix := ""
		// Get list of versions:
		versions, err := repo.Versions(prefix)
		if err != nil {
			panic(err)
		}
		Ln(versions)
		// If no versions found, then get latest commit:
		if len(versions) == 0 {
			latest, err := repo.Latest()
			if err != nil {
				panic(err)
			}
			versions = []string{latest.Version}
		}
		c.IndentedJSON(200, versions)
	})

	r.Run() // listen and serve on 0.0.0.0:8080
}
