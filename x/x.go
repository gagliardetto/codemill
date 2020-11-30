package x

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/gagliardetto/feparser"
	. "github.com/gagliardetto/utilz"
)

type ModelKind string

func IsValidModelKind(kind ModelKind) bool {
	return isAnyOfModelKinds(
		kind,
		// All the valid kinds:
		Router().ListModelKinds()...,
	)
}
func isAnyOfModelKinds(s ModelKind, candidates ...ModelKind) bool {
	for _, v := range candidates {
		if s == v {
			return true
		}
	}
	return false
}

// NewScavengeMethods returns an array of XMethod that are specific
// to the provided kind.
func NewScavengeMethods(kind ModelKind) []*XMethod {
	handler := Router().GetHandler(kind)
	if handler == nil {
		panic(Sf("No default method scavenging for %q kind", kind))
	}
	return handler.ScavengeMethods()
}

type XSpec struct {
	Name   string // Name of the module
	Models []*XModel
	*sync.RWMutex
}

func (spec *XSpec) NormalizeName() error {
	spec.Name = ToCamel(spec.Name)
	if spec.Name == "" {
		return errors.New("Name is not valid")
	}
	return nil
}

// Cleanup cleans up a spec..
func (spec *XSpec) Cleanup() error {
	// Remove empty selectors:

	for _, mdl := range spec.Models {
		for _, mtd := range mdl.Methods {
			for _, sel := range mtd.Selectors {

				basicQual := sel.GetBasicQualifier()
				switch qual := sel.Qualifier.(type) {

				case *StructQualifier:
					{
						if len(qual.Fields) == 0 {
							// If all fields are disabled, then remove the selector:
							mtd.DeleteSelector(
								basicQual.Path,
								basicQual.Version,
								basicQual.ID,
							)
						}
					}
				case *FuncQualifier:
					{
						if AllFalse(qual.Pos...) {
							// If all false, then remove the selector:
							mtd.DeleteSelector(
								basicQual.Path,
								basicQual.Version,
								basicQual.ID,
							)
						}
					}
				case *TypeQualifier:
					{
						if !qual.Value {
							// If false, then remove the selector:
							mtd.DeleteSelector(
								basicQual.Path,
								basicQual.Version,
								basicQual.ID,
							)
						}
					}
				default:
					panic(Sf("Unknown type: %T", sel.Qualifier))
				}

			}
		}
	}

	return nil
}

// AddMeta populates a spec with meta.
func (spec *XSpec) AddMeta() error {
	for _, mdl := range spec.Models {
		for _, mtd := range mdl.Methods {
			for _, sel := range mtd.Selectors {

				basicQual := sel.GetBasicQualifier()
				switch qual := sel.Qualifier.(type) {

				case *StructQualifier:
					{
						{
							source := GetCachedSource(basicQual.Path, basicQual.Version)
							if source == nil {
								return fmt.Errorf("Source not found: %s@%s", basicQual.Path, basicQual.Version)
							}
							// Make sure that the struct exist:
							st := FindStructByID(source, basicQual.ID)
							if st == nil {
								return fmt.Errorf("Struct not found: %q", basicQual.ID)
							}

							for fieldName := range qual.Fields {
								fld := FindFieldByName(st, fieldName)
								if fld == nil {
									return fmt.Errorf("Field not found: %q", fieldName)
								}

								qual.Fields[fld.VarName] = &FieldMeta{
									Name:       fld.VarName,
									TypeString: fld.TypeString,
									KindString: fld.KindString,
								}
							}

							{ // Update counts:
								qual.Total = len(st.Fields)
								qual.Left = len(st.Fields) - len(qual.Fields)
							}

						}
					}
				case *FuncQualifier:
					{
						source := GetCachedSource(basicQual.Path, basicQual.Version)
						if source == nil {
							return fmt.Errorf("Source not found: %s@%s", basicQual.Path, basicQual.Version)
						}
						// Find the func/type-method/interface-method:
						fn := FindFuncByID(source, basicQual.ID)
						if fn == nil {
							return fmt.Errorf("Func not found: %q", basicQual.ID)
						}

						meta := CompileFuncQualifierElementsMeta(fn)
						qual.Elements = meta
					}
				case *TypeQualifier:
					{
						source := GetCachedSource(basicQual.Path, basicQual.Version)
						if source == nil {
							return fmt.Errorf("Source not found: %s@%s", basicQual.Path, basicQual.Version)
						}
						// Find the type:
						typ := FindTypeByID(source, basicQual.ID)
						if typ == nil {
							return fmt.Errorf("Type not found: %q", basicQual.ID)
						}

						qual.TypeName = typ.TypeName
						qual.KindString = typ.KindString
					}
				default:
					panic(Sf("Unknown type: %T", sel.Qualifier))
				}

			}
		}
	}

	return nil
}

func (spec *XSpec) RemoveMeta() error {
	for _, mdl := range spec.Models {
		for _, mtd := range mdl.Methods {
			for _, sel := range mtd.Selectors {

				switch qual := sel.Qualifier.(type) {

				case *StructQualifier:
					{
						for fieldName := range qual.Fields {
							qual.Fields[fieldName] = nil // TODO: will this still work in json?
						}

						{ // Update counts:
							qual.Total = 0
							qual.Left = 0
						}
					}
				case *FuncQualifier:
					{
						// TODO
						qual.Elements = nil
					}
				case *TypeQualifier:
					{
						// TODO
						//qual.TypeName = ""
						qual.KindString = ""
						// TODO: move TypeName, KindString to a Meta struct.
					}
				default:
					panic(Sf("Unknown type: %T", sel.Qualifier))
				}

			}
		}
	}

	return nil
}

// Sort sorts things inside the spec.
func (spec *XSpec) Sort() {
	{
		// For each method, sort selectors by PathVersion:
		for _, mdl := range spec.Models {
			for _, mtd := range mdl.Methods {
				sort.Slice(mtd.Selectors, func(i, j int) bool {
					basicI := mtd.Selectors[i].GetBasicQualifier()
					basicJ := mtd.Selectors[j].GetBasicQualifier()

					return basicI.PathVersion() < basicJ.PathVersion()
				})
			}
		}
	}
}

// ListModules lists all the modules (unique) used inside the spec.
func (spec *XSpec) ListModules() []*BasicQualifier {
	qualifiers := make([]*BasicQualifier, 0)

	for _, mdl := range spec.Models {
		for _, mtd := range mdl.Methods {
			for _, sel := range mtd.Selectors {

				qual := sel.GetBasicQualifier()
				if qual != nil {
					qualifiers = append(qualifiers, qual)
				}

			}
		}
	}

	// Deduplicate:
	qualifiers = DeduplicateSlice(qualifiers, func(i int) string {
		return FormatPathVersion(qualifiers[i].Path, qualifiers[i].Version)
	}).([]*BasicQualifier)

	return qualifiers
}

// Validate validates a spec.
func (spec *XSpec) Validate() error {
	if err := spec.NormalizeName(); err != nil {
		return err
	}

	// Normalize and validate names of models:
	for _, mdl := range spec.Models {
		err := mdl.NormalizeName()
		if err != nil {
			return fmt.Errorf("error for model %q: %s", mdl.Name, err)
		}
	}
	{
		// Check whether model names are unique:
		var names []string
		for _, mdl := range spec.Models {
			if SliceContains(names, mdl.Name) {
				return fmt.Errorf("Model name %q is not unique", mdl.Name)
			}
			names = append(names, mdl.Name)
		}
	}

	// Validate models:
	for _, mdl := range spec.Models {
		err := mdl.Validate()
		if err != nil {
			return fmt.Errorf("error for model %q: %s", mdl.Name, err)
		}
	}

	return nil
}

func (mdl *XModel) NormalizeName() error {
	mdl.Name = ToCamel(mdl.Name)
	if mdl.Name == "" {
		return errors.New("Name is not valid")
	}
	return nil
}

// Validate validates a model.
func (mdl *XModel) Validate() error {
	if err := mdl.NormalizeName(); err != nil {
		return err
	}

	// Validate model kind:
	isValid := IsValidModelKind(mdl.Kind)
	if !isValid {
		return fmt.Errorf("model kind not valid: %q", mdl.Kind)
	}

	// Normalize and validate names of methods:
	for _, mtd := range mdl.Methods {
		err := mtd.NormalizeName()
		if err != nil {
			return fmt.Errorf("error for model %q: %s", mtd.Name, err)
		}
	}
	{
		// Check whether method names are unique:
		var names []string
		for _, mtd := range mdl.Methods {
			if SliceContains(names, mtd.Name) {
				return fmt.Errorf("Method name %q is not unique", mtd.Name)
			}
			names = append(names, mtd.Name)
		}
	}

	// Validate Methods:
	for _, mtd := range mdl.Methods {
		err := mtd.Validate()
		if err != nil {
			return fmt.Errorf("error for method %q: %s", mtd.Name, err)
		}
	}

	return nil
}

func (mtd *XMethod) NormalizeName() error {
	mtd.Name = ToCamel(mtd.Name)
	if mtd.Name == "" {
		return errors.New("Name is not valid")
	}
	return nil
}

// Validate validates a method.
func (mtd *XMethod) Validate() error {
	if err := mtd.NormalizeName(); err != nil {
		return err
	}
	return nil
}

// Validate validates a selector.
func (sel *XSelector) Validate() error {
	isValid := IsValidSelectorKind(sel.Kind)
	if !isValid {
		return fmt.Errorf("selector kind not valid: %q", sel.Kind)
	}

	switch sel.Kind {
	case SelectorKindFunc:
		{
			if err := sel.GetFuncQualifier().Validate(); err != nil {
				return err
			}
		}
	case SelectorKindStruct:
		{
			if err := sel.GetStructQualifier().Validate(); err != nil {
				return err
			}
		}
	case SelectorKindType:
		{
			if err := sel.GetTypeQualifier().Validate(); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("Unknown selector kind: %s", sel.Kind)
	}

	return nil
}

func (sel *XSelector) UnmarshalJSON(data []byte) error {

	var temp struct {
		Kind      SelectorKind
		Qualifier interface{}
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	if !IsValidSelectorKind(temp.Kind) {
		return fmt.Errorf("selector kind not valid: %q", sel.Kind)
	}

	sel.Kind = temp.Kind

	switch sel.Kind {
	case SelectorKindFunc:
		{
			var v FuncQualifier
			if err := TranscodeJSON(temp.Qualifier, &v); err != nil {
				return err
			}
			sel.Qualifier = &v
		}
	case SelectorKindStruct:
		{
			var v StructQualifier
			if err := TranscodeJSON(temp.Qualifier, &v); err != nil {
				return err
			}
			sel.Qualifier = &v
		}
	case SelectorKindType:
		{
			var v TypeQualifier
			if err := TranscodeJSON(temp.Qualifier, &v); err != nil {
				return err
			}
			sel.Qualifier = &v
		}
	default:
		return fmt.Errorf("Unknown selector kind: %s", sel.Kind)
	}

	return nil
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
func (spec *XSpec) ModifyModelByName(
	name string,
	modifier func(*XModel) error,
) error {
	spec.Lock()
	defer spec.Unlock()

	for _, md := range spec.Models {
		if md.Name == name {
			return modifier(md)
		}
	}
	return fmt.Errorf("Model %q (on spec %q) not found", name, spec.Name)
}

//
func (mdl *XModel) ModifyMethodByName(
	name string,
	modifier func(*XMethod) error,
) error {
	// TODO: lock here too??
	// NOTE: it's already locked at model level in XSpec.
	for _, md := range mdl.Methods {
		if md.Name == name {
			return modifier(md)
		}
	}
	return fmt.Errorf("Method %q (on model %q) not found", name, mdl.Name)
}

//
func (mt *XMethod) GetStructSelector(
	path string,
	version string,
	structID string,
) *StructQualifier {
	for _, sel := range mt.Selectors {
		stQual := sel.GetStructQualifier()
		if stQual == nil {
			continue
		}
		if stQual.BasicQualifier.Is(path, version, structID) {
			return stQual
		}
	}
	return nil
}

//
func (mt *XMethod) GetFuncSelector(
	path string,
	version string,
	funcID string,
) *FuncQualifier {
	for _, sel := range mt.Selectors {
		stQual := sel.GetFuncQualifier()
		if stQual == nil {
			continue
		}
		if stQual.BasicQualifier.Is(path, version, funcID) {
			return stQual
		}
	}
	return nil
}

//
func (mt *XMethod) DeleteSelector(
	path string,
	version string,
	id string,
) bool {
	for i, sel := range mt.Selectors {
		qual := sel.GetBasicQualifier()
		if qual == nil {
			continue
		}
		if qual.Is(path, version, id) {
			return mt.deleteSelectorAtIndex(i)
		}
	}
	return false
}

//
func (mt *XMethod) GetTypeSelector(
	path string,
	version string,
	funcID string,
) *TypeQualifier {
	for _, sel := range mt.Selectors {
		stQual := sel.GetTypeQualifier()
		if stQual == nil {
			continue
		}
		if stQual.BasicQualifier.Is(path, version, funcID) {
			return stQual
		}
	}
	return nil
}

//
func (mt *XMethod) DeleteTypeSelector(
	path string,
	version string,
	funcID string,
) bool {
	for i, sel := range mt.Selectors {
		stQual := sel.GetTypeQualifier()
		if stQual == nil {
			continue
		}
		if stQual.BasicQualifier.Is(path, version, funcID) {
			return mt.deleteSelectorAtIndex(i)
		}
	}
	return false
}

//
func (mt *XMethod) deleteSelectorAtIndex(index int) bool {
	for i := range mt.Selectors {
		if i == index {
			// Remove the element at index i from a.
			mt.Selectors[i] = mt.Selectors[len(mt.Selectors)-1] // Copy last element to index i.
			mt.Selectors[len(mt.Selectors)-1] = nil             // Erase last element (write zero value).
			mt.Selectors = mt.Selectors[:len(mt.Selectors)-1]   // Truncate slice.
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

	{
		spec.Lock()
		defer spec.Unlock()
		spec.Models = append(spec.Models, model)
	}
	return nil
}

type XModel struct {
	Name    string
	Kind    ModelKind
	Methods []*XMethod
}

type XMethod struct {
	Name      string
	IsSelf    bool
	Selectors []*XSelector
}

type SelectorKind string

const (
	SelectorKindStruct SelectorKind = "Struct" // Qualifier for structs.
	SelectorKindFunc   SelectorKind = "Func"   // Qualifier for funcs, type methods, interface methods.
	SelectorKindType   SelectorKind = "Type"   // Qualifier for types.
)

func IsValidSelectorKind(kind SelectorKind) bool {
	return IsAnyOf(
		string(kind),
		// All the valid kinds:
		string(SelectorKindStruct),
		string(SelectorKindFunc),
		string(SelectorKindType),
	)
}

type XSelector struct {
	Kind      SelectorKind
	Qualifier interface{}
}
type BasicQualifier struct {
	Path    string
	Version string
	ID      string
}
type StructQualifier struct {
	BasicQualifier
	TypeName string
	Fields   map[string]*FieldMeta
	Total    int `json:",omitempty"`
	Left     int `json:",omitempty"`
}

// Validate validates a BasicQualifier.
func (qual *BasicQualifier) Validate() error {
	if qual.Path == "" {
		return errors.New("Path not set")
	}
	if qual.Version == "" {
		return errors.New("Version not set")
	}
	if qual.ID == "" {
		return errors.New("ID not set")
	}
	return nil
}

func (qual *BasicQualifier) Is(path string, version string, id string) bool {
	if qual.Path != path {
		return false
	}
	if qual.Version != version {
		return false
	}
	if qual.ID != id {
		return false
	}
	return true
}

// Validate validates a FuncQualifier.
func (qual *FuncQualifier) Validate() error {
	if err := qual.BasicQualifier.Validate(); err != nil {
		return fmt.Errorf("error while validating BasicQualifier: %s", err)
	}
	// TODO
	return nil
}

// Validate validates a StructQualifier.
func (qual *StructQualifier) Validate() error {
	if err := qual.BasicQualifier.Validate(); err != nil {
		return fmt.Errorf("error while validating BasicQualifier: %s", err)
	}
	// TODO
	return nil
}

type FieldMeta struct {
	Name       string `json:",omitempty"`
	TypeString string `json:",omitempty"`
	KindString string `json:",omitempty"`
}

//
func (sel *XSelector) GetStructQualifier() *StructQualifier {
	got, ok := sel.Qualifier.(*StructQualifier)
	if !ok {
		return nil
	}
	return got
}

type FuncQualifier struct {
	BasicQualifier
	Pos      []bool
	Name     string                     // Name of the func.
	Elements *FuncQualifierElementsMeta `json:",omitempty"`
}

type FuncQualifierElementsMeta struct {
	Receiver   *FuncElementMeta
	Parameters []*FuncElementMeta
	Results    []*FuncElementMeta
}

type FuncElementMeta struct {
	AI         int    // Absolute index
	RI         int    // Relative index
	Name       string // The VarName
	TypeString string
	KindString string
}

type TypeQualifier struct {
	BasicQualifier
	TypeName   string // Name of the type.
	KindString string `json:",omitempty"`
	Value      bool
}

func compileFuncElemMeta(ai int, ri int, typ *feparser.FEType) *FuncElementMeta {
	return &FuncElementMeta{
		AI:         ai,
		RI:         ri,
		Name:       typ.VarName,
		TypeString: typ.TypeString,
		KindString: typ.KindString,
	}
}

func CompileFuncQualifierElementsMeta(raw interface{}) *FuncQualifierElementsMeta {
	switch thing := raw.(type) {
	case *feparser.FEFunc:
		{
			out := &FuncQualifierElementsMeta{
				Receiver: nil,
			}

			for i, re := range thing.Parameters {
				out.Parameters = append(out.Parameters, compileFuncElemMeta(i, i, re))
			}
			for i, re := range thing.Results {
				out.Results = append(out.Results, compileFuncElemMeta(i+len(thing.Parameters), i, re))
			}
			return out
		}
	case *feparser.FETypeMethod:
		{
			out := &FuncQualifierElementsMeta{
				Receiver: compileFuncElemMeta(0, 0, &(thing.Receiver.FEType)),
			}

			for i, re := range thing.Func.Parameters {
				out.Parameters = append(out.Parameters, compileFuncElemMeta(i+1, i, re))
			}
			for i, re := range thing.Func.Results {
				out.Results = append(out.Results, compileFuncElemMeta(i+len(thing.Func.Parameters)+1, i, re))
			}
			return out
		}
	case *feparser.FEInterfaceMethod:
		{
			out := &FuncQualifierElementsMeta{
				Receiver: compileFuncElemMeta(0, 0, &(thing.Receiver.FEType)),
			}

			for i, re := range thing.Func.Parameters {
				out.Parameters = append(out.Parameters, compileFuncElemMeta(i+1, i, re))
			}
			for i, re := range thing.Func.Results {
				out.Results = append(out.Results, compileFuncElemMeta(i+len(thing.Func.Parameters)+1, i, re))
			}
			return out
		}
	default:
		panic(Sf("Unknown type: %T", raw))
	}

}

func GetFuncName(raw interface{}) string {
	switch thing := raw.(type) {
	case *feparser.FEFunc:
		{
			return thing.Name
		}
	case *feparser.FETypeMethod:
		{
			return thing.Func.Name
		}
	case *feparser.FEInterfaceMethod:
		{
			return thing.Func.Name
		}
	default:
		panic(Sf("Unknown type: %T", raw))
	}
}

//
func (sel *XSelector) GetBasicQualifier() *BasicQualifier {
	{
		got, ok := sel.Qualifier.(*FuncQualifier)
		if ok {
			return &got.BasicQualifier
		}
	}
	{
		got, ok := sel.Qualifier.(*StructQualifier)
		if ok {
			return &got.BasicQualifier
		}
	}
	{
		got, ok := sel.Qualifier.(*TypeQualifier)
		if ok {
			return &got.BasicQualifier
		}
	}
	return nil
}

//
func (bq *BasicQualifier) PathVersion() string {
	return FormatPathVersion(bq.Path, bq.Version)
}

//
func (sel *XSelector) GetFuncQualifier() *FuncQualifier {
	got, ok := sel.Qualifier.(*FuncQualifier)
	if !ok {
		return nil
	}
	return got
}

//
func (sel *XSelector) GetTypeQualifier() *TypeQualifier {
	got, ok := sel.Qualifier.(*TypeQualifier)
	if !ok {
		return nil
	}
	return got
}

func NewXSpecWithName(name string) *XSpec {
	name = ToCamel(name)
	if name == "" {
		panic("provided empty name")
	}

	spec := newXSpec()
	spec.Name = name
	return spec
}

func newXSpec() *XSpec {
	return &XSpec{
		Models:  []*XModel{},
		RWMutex: &sync.RWMutex{},
	}
}

var (
	globalModelKindRouter *ModelKindRouter
)

// InitRouter intializes and returns a new ModelKindRouter.
// InitRouter can be called only once; after the first call,
// you can subsequently access the router by calling Router().
func InitRouter(conf *ModelKindRouterConfig) (*ModelKindRouter, error) {
	if globalModelKindRouter != nil {
		return nil, errors.New("model kind router already initialized")
	}
	rt, err := NewModelKindRouter(conf)
	if err != nil {
		return nil, err
	}

	globalModelKindRouter = rt
	return globalModelKindRouter, nil
}

// Router returns the initialized global ModelKind router.
// Panics if the router hasn't been created yet.
func Router() *ModelKindRouter {
	if globalModelKindRouter == nil {
		panic("model kind router not initialized; you need to call InitRouter first.")
	}
	return globalModelKindRouter
}

// ModelKindRouter is a router that handles the generation of code
// for each registered ModelKind.
type ModelKindRouter struct {
	handlers map[ModelKind]ModelKindHandler
	mu       *sync.RWMutex
	conf     *ModelKindRouterConfig
}

type ModelKindRouterConfig struct {
	Dir string // Dir is the folder where the generated code will be saved to.
}

// Validate validates ModelKindRouterConfig
func (conf *ModelKindRouterConfig) Validate() error {
	if conf.Dir == "" {
		return errors.New("Dir is empty")
	}
	return nil
}

func NewModelKindRouter(conf *ModelKindRouterConfig) (*ModelKindRouter, error) {
	if err := conf.Validate(); err != nil {
		return nil, fmt.Errorf("error while validating configuration: %s", err)
	}

	rt := &ModelKindRouter{
		handlers: make(map[ModelKind]ModelKindHandler),
		mu:       &sync.RWMutex{},
		conf:     conf,
	}
	return rt, nil
}

//
func (rt *ModelKindRouter) RegisterHandler(kind ModelKind, handler ModelKindHandler) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	_, has := rt.handlers[kind]
	if has {
		return fmt.Errorf("error: model kind already has a registered handler: %s", kind)
	}

	if handler == nil {
		return errors.New("handler is nil")
	}

	rt.handlers[kind] = handler
	return nil
}

//
func (rt *ModelKindRouter) GetHandler(kind ModelKind) ModelKindHandler {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	handler, has := rt.handlers[kind]
	if has {
		return handler
	}

	return nil
}

//
func (rt *ModelKindRouter) HasHandler(kind ModelKind) bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	_, has := rt.handlers[kind]
	return has
}

//
func (rt *ModelKindRouter) ListModelKinds() []ModelKind {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	kinds := make([]ModelKind, 0)
	for key := range rt.handlers {
		kinds = append(kinds, key)
	}
	return kinds
}

//
func (rt *ModelKindRouter) Handle(kind ModelKind, mdl *XModel) error {
	handler := rt.GetHandler(kind)
	if handler == nil {
		return errors.New("handler not found")
	}

	{
		// Validate provided model:
		err := handler.Validate(mdl)
		if err != nil {
			return fmt.Errorf(
				"error while validating model %q (kind=%s): %s",
				mdl.Name,
				kind,
				err,
			)
		}
	}
	dir := rt.conf.Dir

	{
		// Generate codeql:
		err := handler.GenerateCodeQL(dir, mdl)
		if err != nil {
			return fmt.Errorf(
				"error while generating codeql code for model %q (kind=%s): %s",
				mdl.Name,
				kind,
				err,
			)
		}
	}
	{
		// Generate go:
		err := handler.GenerateGo(dir, mdl)
		if err != nil {
			return fmt.Errorf(
				"error while generating go code for model %q (kind=%s): %s",
				mdl.Name,
				kind,
				err,
			)
		}
	}

	return nil
}

type ModelKindHandler interface {
	// GenerateCodeQL generates codeql code based on the
	// provided model; the generated code is then saved in the
	// destination dir.
	GenerateCodeQL(dir string, mdl *XModel) error

	// GenerateGo generates go code based on the
	// provided model; the generated code is then saved in the
	// destination dir.
	GenerateGo(dir string, mdl *XModel) error

	// ScavengeMethods returns an array of initialized
	// methods unique to the ModelKind.
	ScavengeMethods() []*XMethod

	// Validate validates the provided XModel.
	Validate(mdl *XModel) error
}

type PackageLoader func(path string, version string) (*feparser.FEPackage, error)

func TryLoadSpecFromFile(path string, loader PackageLoader) (*XSpec, error) {
	spec := newXSpec()
	err := LoadJSON(spec, path)
	if err != nil {
		return nil, fmt.Errorf("error while loading spec file: %s", err)
	}
	// TODO:
	// - validate names
	// - validate classes
	// - validate methods
	// - validate selectors
	// - check for duplicate names
	// - remove empty selectors
	// - populate selector meta

	if err := spec.Validate(); err != nil {
		return nil, err
	}
	if err := spec.Cleanup(); err != nil {
		return nil, err
	}
	{
		// Load all used packages (modules):
		mods := spec.ListModules()
		for _, m := range mods {
			_, err := loader(m.Path, m.Version)
			if err != nil {
				return nil, fmt.Errorf("error while loading package %s@%s: %s", m.Path, m.Version, err)
			}
		}
	}
	if err := spec.AddMeta(); err != nil {
		return nil, err
	}
	return spec, nil
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

	cleanupFEPackage(pkg)
	sourceCache[key] = pkg
}

// cleanupFEPackage removes superfuous stuff.
func cleanupFEPackage(pkg *feparser.FEPackage) {
	for _, v := range pkg.Funcs {
		v.CodeQL = nil
		for _, param := range v.Parameters {
			param.Identity = nil
		}
		for _, res := range v.Results {
			res.Identity = nil
		}
	}
	for _, v := range pkg.TypeMethods {
		v.CodeQL = nil
		for _, param := range v.Func.Parameters {
			param.Identity = nil
		}
		for _, res := range v.Func.Results {
			res.Identity = nil
		}
	}
	for _, v := range pkg.InterfaceMethods {
		v.CodeQL = nil
		for _, param := range v.Func.Parameters {
			param.Identity = nil
		}
		for _, res := range v.Func.Results {
			res.Identity = nil
		}
	}
}

func FindStructByID(fe *feparser.FEPackage, id string) *feparser.FEStruct {
	for _, st := range fe.Structs {
		if st.ID == id {
			return st
		}
	}
	return nil
}
func FindFieldByID(st *feparser.FEStruct, id string) *feparser.FEField {
	for _, st := range st.Fields {
		if st.ID == id {
			return st
		}
	}
	return nil
}
func FindFieldByName(st *feparser.FEStruct, name string) *feparser.FEField {
	for _, st := range st.Fields {
		if st.VarName == name {
			return st
		}
	}
	return nil
}

type LenInterface interface {
	Len() int
}

func FindFuncByID(fe *feparser.FEPackage, id string) LenInterface {
	for _, st := range fe.Funcs {
		if st.ID == id {
			return st
		}
	}
	for _, st := range fe.TypeMethods {
		if st.ID == id {
			return st
		}
	}
	for _, st := range fe.InterfaceMethods {
		if st.ID == id {
			return st
		}
	}
	return nil
}
func FindTypeByID(fe *feparser.FEPackage, id string) *feparser.FEType {
	for _, st := range fe.Types {
		if st.ID == id {
			return st
		}
	}
	return nil
}
