package main

//go:generate statik -src=./public -include=*.html,*.css,*.js
import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gagliardetto/codebox/scanner"
	_ "github.com/gagliardetto/codemill/statik"
	"github.com/gagliardetto/codemill/x"
	cqljen "github.com/gagliardetto/cqlgen/jen"
	"github.com/gagliardetto/feparser"
	"github.com/gagliardetto/golang-go/cmd/go/not-internal/get"
	"github.com/gagliardetto/golang-go/cmd/go/not-internal/modfetch"
	"github.com/gagliardetto/golang-go/cmd/go/not-internal/search"
	"github.com/gagliardetto/golang-go/cmd/go/not-internal/web"
	"github.com/gagliardetto/request"
	. "github.com/gagliardetto/utilz"
	"github.com/gin-gonic/gin"
	"github.com/rakyll/statik/fs"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"

	"github.com/gagliardetto/codemill/handlers/http/headerwrite"
	"github.com/gagliardetto/codemill/handlers/http/redirect"
	"github.com/gagliardetto/codemill/handlers/http/responsebody"
	"github.com/gagliardetto/codemill/handlers/tainttracking"
	"github.com/gagliardetto/codemill/handlers/untrustedflowsource"
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

	statikFS, err := fs.New()
	if err != nil {
		Fataln(err)
	}

	{ // Add http handlers for static files:
		r.GET("/", func(c *gin.Context) {
			reader, err := statikFS.Open("/index.html")
			if err != nil {
				Q(err)
				Abort404(c, err.Error())
				return
			}
			defer reader.Close()
			contents, err := ioutil.ReadAll(reader)
			if err != nil {
				Q(err)
				Abort404(c, err.Error())
				return
			}
			c.Data(200, "text/html; charset=UTF-8", contents)
		})
		r.GET("/static/:filename", func(c *gin.Context) {
			name := c.Param("filename")
			if name == "" {
				c.AbortWithStatus(400)
				return
			}
			reader, err := statikFS.Open("/static/" + name)
			if err != nil {
				c.AbortWithError(400, err)
				Q(err)
				return
			}
			defer reader.Close()
			contents, err := ioutil.ReadAll(reader)
			if err != nil {
				c.AbortWithError(400, err)
				Q(err)
				return
			}
			m := mime.TypeByExtension(filepath.Ext(name))
			c.Data(200, Sf("%s; charset=UTF-8", m), contents)
		})
	}
	httpClient := new(http.Client)

	var specFilepath string
	var outDir string
	var runServer bool
	var doGen bool
	var doSummary bool
	flag.StringVar(&specFilepath, "spec", "", "Path to spec file; file will be created if not already existing.")
	flag.StringVar(&outDir, "dir", "", "Path to dir where to save generated files.")
	flag.BoolVar(&runServer, "http", true, "Run http server.")
	flag.BoolVar(&doGen, "gen", true, "Generate code.")
	flag.BoolVar(&doSummary, "summary", true, "Output a summary.")
	flag.Parse()

	if specFilepath == "" {
		// specFilepath is ALWAYS necessary,
		// either for knowing from where to load a spec,
		// or where to save a new created one.
		panic("--spec flag not provided")
	}

	if outDir == "" {
		panic("--dir flag not provided")
	}

	{
		rt := x.Router()
		// Register ModelKind handlers in the router:
		{
			// untrustedflowsource handler:
			err := rt.RegisterHandler(untrustedflowsource.Kind, &untrustedflowsource.Handler{})
			if err != nil {
				Fatalf("error while registering handler: %s", err)
			}

			// tainttracking handler:
			err = rt.RegisterHandler(tainttracking.Kind, &tainttracking.Handler{})
			if err != nil {
				Fatalf("error while registering handler: %s", err)
			}

			// http redirect handler:
			err = rt.RegisterHandler(redirect.Kind, &redirect.Handler{})
			if err != nil {
				Fatalf("error while registering handler: %s", err)
			}

			// http responsebody handler:
			err = rt.RegisterHandler(responsebody.Kind, &responsebody.Handler{})
			if err != nil {
				Fatalf("error while registering handler: %s", err)
			}

			// http headerwrite handler:
			err = rt.RegisterHandler(headerwrite.Kind, &headerwrite.Handler{})
			if err != nil {
				Fatalf("error while registering handler: %s", err)
			}
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
		// TODO: cleanup before saving.

		Infof("Saving spec to %q", MustAbs(specFilepath))
		err := SaveAsIndentedJSON(globalSpec, specFilepath)
		if err != nil {
			panic(err)
		}
		if doSummary {
			Ln("\n", strings.Repeat("-", 60), "\n")
			summary, err := x.CreateSummary(globalSpec)
			if err != nil {
				panic(err)
			}
			for _, v := range summary {
				Ln(v)
			}
			Ln("\n", strings.Repeat("-", 60), "\n")
		}
		if !doGen {
			Ln(LimeBG(">>> Completed without generation <<<"))
			os.Exit(0)
		}
		// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
		// NOTE: after this point, any modification to globalSpec will be volatile,
		// i.e. discarded the instant this program hits os.Exit.
		// <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

		// Sort stuff for visual convenience in the generated code:
		globalSpec.Sort()

		// Create output dir if it doesn't exist:
		MustCreateFolderIfNotExists(outDir, os.ModePerm)

		ts := time.Now()
		// Create subfolder for package for generated assets:
		packageAssetFolderName := feparser.FormatCodeQlName(globalSpec.Name)
		packageAssetFolderPath := path.Join(outDir, packageAssetFolderName)
		MustCreateFolderIfNotExists(packageAssetFolderPath, os.ModePerm)
		// Create folder for assets generated during this run:
		thisRunAssetFolderName := feparser.FormatCodeQlName(globalSpec.Name) + "_" + ts.Format(FilenameTimeFormat)
		thisRunAssetFolderPath := path.Join(packageAssetFolderPath, thisRunAssetFolderName)
		// Create a new assets folder inside the main assets folder:
		MustCreateFolderIfNotExists(thisRunAssetFolderPath, os.ModePerm)

		{
			// Validate all specs:
			for _, mdl := range globalSpec.Models {

				handler := x.Router().GetHandler(mdl.Kind)
				if handler == nil {
					Fatalf(
						"handler not found for kind %s",
						mdl.Kind,
					)
				}

				{
					// Validate provided model:
					err := handler.Validate(mdl)
					if err != nil {
						Fatalf(
							"error while validating model %q (kind=%s): %s",
							mdl.Name,
							mdl.Kind,
							err,
						)
					}
				}
			}
		}

		{ // Generate codeql:
			cqlFile := cqljen.NewFile()
			for _, hdr := range x.CqlFormatHeaderDoc(globalSpec.ListModules()) {
				cqlFile.HeaderDoc(hdr)
			}

			// `go` is always imported:
			cqlFile.Import("go")

			cqlFile.Doc(x.CqlFormatHeaderDoc(globalSpec.ListModules())...)
			cqlFile.Private().Module().Id(feparser.FormatCodeQlName(globalSpec.Name)).BlockFunc(func(moduleGroup *cqljen.Group) {
				for _, mdl := range globalSpec.Models {

					handler := x.Router().MustGetHandler(mdl.Kind)
					{
						// Generate codeql with the handler of the ModelKind;
						// the handler might generate predicates, classes, etc.
						// all within the module block.
						err := handler.GenerateCodeQL(cqlFile, mdl, moduleGroup)
						if err != nil {
							Fatalf(
								"error while generating codeql code for model %q (kind=%s): %s",
								mdl.Name,
								mdl.Kind,
								err,
							)
						}
					}

				}

			})
			{
				// Save codeql assets:
				assetFileName := feparser.FormatCodeQlName(globalSpec.Name) + ".qll"
				assetFilepath := path.Join(thisRunAssetFolderPath, assetFileName)

				// Create file codeql file:
				codeqlFile, err := os.Create(assetFilepath)
				if err != nil {
					panic(err)
				}
				defer codeqlFile.Close()

				// Write generated codeql to file:
				Infof("Saving codeql assets to %q", MustAbs(assetFilepath))
				err = cqlFile.Render(codeqlFile)
				if err != nil {
					panic(err)
				}
			}
		}
		{
			goTestsFolderPath := path.Join(thisRunAssetFolderPath, "tests")
			// Create a folder for Go code:
			MustCreateFolderIfNotExists(goTestsFolderPath, os.ModePerm)
			// Generate Go code:
			for _, mdl := range globalSpec.Models {

				handler := x.Router().MustGetHandler(mdl.Kind)
				{
					err := handler.GenerateGo(goTestsFolderPath, mdl)
					if err != nil {
						Fatalf(
							"error while generating Go code for model %q (kind=%s): %s",
							mdl.Name,
							mdl.Kind,
							err,
						)
					}
				}

			}
		}

		Ln(LimeBG(">>> Generation completed <<<"))
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
		sort.Slice(kinds, func(i, j int) bool {
			return kinds[i] < kinds[j]
		})
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
								mt.DeleteSelector(
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

		const FlowKeyInp = "Inp"
		const FlowKeyOut = "Out"
		validFlowKeys := []string{FlowKeyInp, FlowKeyOut}
		type FlowValueSet struct {
			BlockIndex int
			Key        string // Either Inp out Out.
			Index      int    // Index on the Func total length.
			Value      bool
		}
		type PosValueSet struct {
			Index int
			Value bool
		}
		var req struct {
			Where struct {
				Path    string
				Version string
				Model   string
				Method  string
			}
			What struct {
				FuncID string
			}

			Pos  *PosValueSet
			Flow *FlowValueSet
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

		if req.Pos != nil && req.Flow != nil {
			Abort400(c, "Non-valid request: req.Pos and req.Flow are both set.")
			return
		}

		if req.Pos != nil {
			if req.Pos.Index < 0 || req.Pos.Index >= fn.Len() {
				Abort400(c, Sf("req.Pos.Index out of bounds: index=%v, but v.Len() = %v", req.Pos.Index, fn.Len()))
				return
			}
		}

		if req.Flow != nil {
			// Validate flow Key:
			isValidFlowKey := SliceContains(validFlowKeys, req.Flow.Key)
			if !isValidFlowKey {
				Abort400(c, Sf("Provided req.Flow.Key is not valid: %q", req.Flow.Key))
				return
			}
			// Validate Index:
			if req.Flow.Index < 0 || req.Flow.Index >= fn.Len() {
				Abort400(c, Sf("req.Flow.Index out of bounds: index=%v, but v.Len() = %v", req.Flow.Index, fn.Len()))
				return
			}
		}

		err = globalSpec.ModifyModelByName(
			req.Where.Model,
			func(mdl *x.XModel) error {
				// Currently, only the tainttracking.Handler is the only handler
				// that supports flow handling.
				if req.Flow != nil && !ModelSupportsFuncFlow(mdl) {
					return errors.New("This model does not support func flow qualifiers.")
				}
				err := mdl.ModifyMethodByName(
					req.Where.Method,
					func(mt *x.XMethod) error {

						meta := x.CompileFuncQualifierElementsMeta(fn)
						existingSel := mt.GetFuncSelector(
							req.Where.Path,
							req.Where.Version,
							req.What.FuncID,
						)

						// Handle Pos:
						if req.Pos != nil {
							if existingSel == nil {
								// Add a new selector only if the value is true:
								if req.Pos.Value {
									pos := make([]bool, fn.Len())
									pos[req.Pos.Index] = req.Pos.Value

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
								existingSel.Pos[req.Pos.Index] = req.Pos.Value
								existingSel.Elements = meta

								if AllFalse(existingSel.Pos...) {
									// If all false, then remove the selector:
									mt.DeleteSelector(
										req.Where.Path,
										req.Where.Version,
										req.What.FuncID,
									)
								}
							}
							return nil
						}

						// Handle Flow:
						if req.Flow != nil {
							// TODO:
							if existingSel == nil {
								// Add a new selector only if the value is true:
								if req.Flow.Value {

									// If there is no existing selector,
									// then create a new one:

									// If the selector did not exist before,
									// then the BlockIndex must be = 0.
									if req.Flow.BlockIndex != 0 {
										return errors.New("req.Flow.BlockIndex must be zero when first creating.")
									}

									// Create a new FlowSpec:
									flSpec := &x.FlowSpec{
										Enabled: true,
										Blocks:  make([]*x.FlowBlock, 0),
									}
									// Create a new block:
									newBlock := &x.FlowBlock{
										Inp: make([]bool, fn.Len()),
										Out: make([]bool, fn.Len()),
									}

									// Set value:
									switch req.Flow.Key {
									case FlowKeyInp:
										newBlock.Inp[req.Flow.Index] = req.Flow.Value
									case FlowKeyOut:
										newBlock.Out[req.Flow.Index] = req.Flow.Value
									}
									flSpec.Blocks = append(flSpec.Blocks, newBlock)

									newSel := &x.XSelector{
										Kind: x.SelectorKindFunc,
										Qualifier: &x.FuncQualifier{
											BasicQualifier: x.BasicQualifier{
												Path:    req.Where.Path,
												Version: req.Where.Version,
												ID:      req.What.FuncID,
											},
											Flows:    flSpec,
											Name:     x.GetFuncName(fn),
											Elements: meta,
										},
									}

									// Save selector:
									mt.Selectors = append(mt.Selectors, newSel)
								}
							} else {
								if existingSel.Flows == nil {
									// TODO: what to do in this case?
									return errors.New("Found sel.Flows is nil")
								}

								if req.Flow.BlockIndex < 0 || req.Flow.BlockIndex > len(existingSel.Flows.Blocks) /* Block is beyond len+1 */ {
									return fmt.Errorf(
										"req.Flow.BlockIndex is out of bounds: BlockIndex=%v, but blocks.Len() = %v",
										req.Flow.BlockIndex,
										len(existingSel.Flows.Blocks),
									)
								}

								// TODO:
								// - we are len(blocks)+(n>1): error.
								// - we are within the existing blocks: modify block.
								// - we are len(blocks)+1: create a new block and modify it.

								if req.Flow.BlockIndex == len(existingSel.Flows.Blocks) {
									// If the BlockIndex is for a not-yet existing block,
									// the add one new block.

									// This can be done ONLY if it's just one block incremental difference,
									// i.e. we cannot edit the 4th block if we have 2 blocks,
									// but we can edit the 3rd block if we have 2 blocks (the 3rd block will be created here).
									newBlock := &x.FlowBlock{
										Inp: make([]bool, fn.Len()),
										Out: make([]bool, fn.Len()),
									}
									existingSel.Flows.Blocks = append(existingSel.Flows.Blocks, newBlock)
								}

								// Set value:
								switch req.Flow.Key {
								case FlowKeyInp:
									existingSel.Flows.Blocks[req.Flow.BlockIndex].Inp[req.Flow.Index] = req.Flow.Value
								case FlowKeyOut:
									existingSel.Flows.Blocks[req.Flow.BlockIndex].Out[req.Flow.Index] = req.Flow.Value
								}
								existingSel.Elements = meta

								if x.AllBlocksEmpty(existingSel.Flows.Blocks...) {
									existingSel.Flows.Enabled = false

									// If all blocks are empty, then remove the selector:
									mt.DeleteSelector(
										req.Where.Path,
										req.Where.Version,
										req.What.FuncID,
									)
								}
							}
							return nil
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

	r.PATCH("/api/spec/funcs/flow/enable", func(c *gin.Context) {
		// Enable/disable a flow selector:
		type FlowValueSet struct {
			Enable bool // Enable selector.
		}
		var req struct {
			Where struct {
				Path    string
				Version string
				Model   string
				Method  string
			}
			What struct {
				FuncID string
			}
			Flow *FlowValueSet
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

		err = globalSpec.ModifyModelByName(
			req.Where.Model,
			func(mdl *x.XModel) error {
				// Currently, only the tainttracking.Handler is the only handler
				// that supports flow handling.
				if !ModelSupportsFuncFlow(mdl) {
					return errors.New("This model does not support func flow qualifiers.")
				}
				err := mdl.ModifyMethodByName(
					req.Where.Method,
					func(mt *x.XMethod) error {

						meta := x.CompileFuncQualifierElementsMeta(fn)
						existingSel := mt.GetFuncSelector(
							req.Where.Path,
							req.Where.Version,
							req.What.FuncID,
						)

						// Handle Flow:
						if existingSel == nil {
							// The selctor does not exist.
							// TODO: Do nothing, or return a 404??
							return nil
						} else {
							if existingSel.Flows == nil {
								// TODO: what to do in this case?
								return errors.New("Found sel.Flows is nil")
							}

							// Set Enabled to true/false:
							existingSel.Flows.Enabled = req.Flow.Enable

							existingSel.Elements = meta
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

	r.DELETE("/api/spec/funcs/flow/blocks", func(c *gin.Context) {
		// Delete a block:
		type FlowValueSet struct {
			BlockIndex int
		}
		var req struct {
			Where struct {
				Path    string
				Version string
				Model   string
				Method  string
			}
			What struct {
				FuncID string
			}
			Flow *FlowValueSet
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

		err = globalSpec.ModifyModelByName(
			req.Where.Model,
			func(mdl *x.XModel) error {
				if !ModelSupportsFuncFlow(mdl) {
					return errors.New("This model does not support func flow qualifiers.")
				}
				err := mdl.ModifyMethodByName(
					req.Where.Method,
					func(mt *x.XMethod) error {

						meta := x.CompileFuncQualifierElementsMeta(fn)
						existingSel := mt.GetFuncSelector(
							req.Where.Path,
							req.Where.Version,
							req.What.FuncID,
						)

						// Handle Flow:
						if existingSel == nil {
							// The selctor does not exist.
							// TODO: Do nothing, or return a 404??
							return nil
						} else {
							if existingSel.Flows == nil {
								// TODO: what to do in this case?
								return errors.New("Found sel.Flows is nil")
							}

							// Delete block:
							existingSel.Flows.DeleteBlock(req.Flow.BlockIndex)

							existingSel.Elements = meta
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

	if runServer {
		r.Run("0.0.0.0:8070")
	}
}

func ModelSupportsFuncFlow(mdl *x.XModel) bool {
	// Currently, only the tainttracking.Handler is the only handler
	// that supports flow handling.
	if mdl.Kind == tainttracking.Kind {
		return true
	}
	return false
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
		Infof("Package %q belongs to the Go standard library.", path)
	}

	if version != "" {
		cached := x.GetCachedSource(path, version)
		if cached != nil {
			return cached, nil
		}
	}

	var rootPath string
	if isStd {
		rootPath = path
		version = "local"
	} else {
		// TODO: which get.ModuleMode is better?
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
		Mode: packages.LoadSyntax | packages.NeedModule,
	}
	{
		// Create a temporary folder:
		tmpDir, err := ioutil.TempDir("", "codemill")
		if err != nil {
			return nil, err
		}
		// TODO: remove tmpDir or not?
		defer os.RemoveAll(tmpDir)
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
		if len(errs) > 0 {
			return nil, fmt.Errorf("error while packages.Load: %s", CombineErrors(errs...))
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

func Abort400(c *gin.Context, errorString string) {
	abort(c, 400, errorString)
}
func Abort404(c *gin.Context, errorString string) {
	abort(c, 404, errorString)
}
func abort(c *gin.Context, statusCode int, errorString string) {
	c.AbortWithStatusJSON(statusCode, M{"error": errorString})
}
