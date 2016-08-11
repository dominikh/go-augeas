// Package augeas provides Go bindings for Augeas, the configuration
// editing tool.
//
// For more information on Augeas itself, check out http://augeas.net/
package augeas // import "honnef.co/go/augeas"

// #cgo pkg-config: libxml-2.0 augeas
// #include <augeas.h>
// #include <stdlib.h>
import "C"
import (
	"unsafe"
)

// Flag describes flags that influence the behaviour of Augeas when
// passed to New.
type Flag uint

// Bits or'ed together to modify the behavior of Augeas.
const (
	None Flag = 1 << iota

	// Keep the original file with a .augsave extension
	SaveBackup

	// Save changes into a file with extension .augnew, and do not
	// overwrite the original file. Takes precedence over SaveBackup
	SaveNewFile

	// Typecheck lenses; since it can be very expensive it is not done
	// by default
	TypeCheck

	// Do not use the built-in load path for modules
	NoStdinc

	// Make save a no-op process, just record what would have changed
	SaveNoop

	// Do not load the tree automatically
	NoLoad

	NoModlAutoload

	// Track the span in the input of nodes
	EnableSpan

	// Do not close automatically when encountering error during
	// initialization
	NoErrClose
)

// Augeas encapsulates an Augeas handle.
type Augeas struct {
	handle *C.augeas
}

// A Span describes the position of a node in the file it was parsed
// from.
type Span struct {
	Filename   string
	LabelStart uint
	LabelEnd   uint
	ValueStart uint
	ValueEnd   uint
	SpanStart  uint
	SpanEnd    uint
}

// New creates a new Augeas handle, specifying the file system root, a
// list of module directories and flags.
//
// Call the Close method once done with the handle.
func New(root, loadPath string, flags Flag) (Augeas, error) {
	cRoot := C.CString(root)
	defer C.free(unsafe.Pointer(cRoot))
	cLoadPath := C.CString(loadPath)
	defer C.free(unsafe.Pointer(cLoadPath))

	handle := C.aug_init(cRoot, cLoadPath, C.uint(flags))
	if flags&NoErrClose > 0 {
		a := Augeas{handle}
		return a, a.error()
	}
	if handle == nil {
		return Augeas{}, Error{CouldNotInitialize, "Could not initialize Augeas tree", "", ""}
	}

	return Augeas{handle}, nil
}

// DefineVariable defines a variable whose value is the result of
// evaluating the expression. If a variable with the name already
// exists, it will be replaced. Context will not be applied to the
// expression.
//
// If the expression is empty, the variable will be removed.
//
// Path variables can be used in path expressions later on by prefixing
// them with '$'.
//
// Returns the number of nodes if the expression evaluates to a
// nodeset.
func (a Augeas) DefineVariable(name, expression string) (int, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	var cExpression *C.char
	if expression != "" {
		cExpression = C.CString(expression)
		defer C.free(unsafe.Pointer(cExpression))
	}

	ret := C.aug_defvar(a.handle, cName, cExpression)

	if ret == -1 {
		return 0, a.error()
	}

	return int(ret), nil
}

// RemoveVariable removes a variable previously defined by
// DefineVariable.
func (a Augeas) RemoveVariable(name string) error {
	_, err := a.DefineVariable(name, "")
	return err
}

// DefineNode defines a variable whose value is the result of
// evaluating the expression, which must not be empty and evaluate to
// a nodeset. If a variable with the name already exists, it will be
// replaced.
//
// If the expression evaluates to an empty nodeset, a node is created.
//
// Returns the number of nodes in the nodeset and whether a node has
// been created or of it already existed.
func (a Augeas) DefineNode(name, expression, value string) (num int, created bool, err error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	cExpression := C.CString(expression)
	defer C.free(unsafe.Pointer(cExpression))

	cValue := C.CString(value)
	defer C.free(unsafe.Pointer(cValue))

	var cCreated C.int
	cNum := C.aug_defnode(a.handle, cName, cExpression, cValue, &cCreated)
	num = int(cNum)
	created = cCreated == 1
	if cNum == -1 {
		err = a.error()
		return
	}

	return
}

// Get looks up the value associated with a path.
//
// Returns an error if there are no or too many matching nodes, or if
// the path is not a legal path expression.
func (a Augeas) Get(path string) (value string, err error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var cValue *C.char
	ret := C.aug_get(a.handle, cPath, &cValue)

	if ret == 1 {
		return C.GoString(cValue), nil
	} else if ret == 0 {
		return "", Error{NoMatch, "No matching node", "", ""}
	} else if ret < 0 {
		return "", a.error()
	}

	panic("Unexpected return value")
}

// GetAll gets all values associated with a path.
func (a Augeas) GetAll(path string) (values []string, err error) {
	paths, err := a.Match(path)
	if err != nil {
		return
	}

	for _, path := range paths {
		value, err := a.Get(path)
		if err != nil {
			return values, err
		}

		values = append(values, value)
	}

	return
}

// Set the value associated with a path. Intermediate entries are
// created if they don't exist.
func (a Augeas) Set(path, value string) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	cValue := C.CString(value)
	defer C.free(unsafe.Pointer(cValue))

	ret := C.aug_set(a.handle, cPath, cValue)

	if ret == -1 {
		return a.error()
	}

	return nil
}

// SetMultiple sets the value of multiple nodes in one operation. Find
// or create a node matching sub by interpreting sub as a path
// expression relative to each node matching base. sub may be empty,
// in which case all the nodes matching base will be modified.
//
// Returns the number of modified nodes.
func (a Augeas) SetMultiple(base, sub, value string) (int, error) {
	cBase := C.CString(base)
	defer C.free(unsafe.Pointer(cBase))

	var cSub *C.char
	if sub != "" {
		cSub := C.CString(sub)
		defer C.free(unsafe.Pointer(cSub))
	}

	cValue := C.CString(value)
	defer C.free(unsafe.Pointer(cValue))

	ret := C.aug_setm(a.handle, cBase, cSub, cValue)

	if ret == -1 {
		return 0, a.error()
	}

	return int(ret), nil
}

// Span gets the span according to input file of the node associated
// with a path. If the node is associated with a file, the filename,
// label and value start and end positions are set.
func (a Augeas) Span(path string) (Span, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var cFilename *C.char
	var labelStart, labelEnd, valueStart, valueEnd, spanStart, spanEnd C.uint
	var span Span

	ret := C.aug_span(a.handle, cPath, &cFilename,
		&labelStart, &labelEnd,
		&valueStart, &valueEnd,
		&spanStart, &spanEnd)

	if ret == -1 {
		return span, a.error()
	}

	span.LabelStart = uint(labelStart)
	span.LabelEnd = uint(labelEnd)
	span.ValueStart = uint(valueStart)
	span.ValueEnd = uint(valueEnd)
	span.SpanStart = uint(spanStart)
	span.SpanEnd = uint(spanEnd)
	span.Filename = C.GoString(cFilename)
	C.free(unsafe.Pointer(cFilename))

	return span, nil
}

// Insert creates a new sibling for a path by inserting into the
// tree just before the path if before is true or just after it if
// before is false.
//
// The path must match exactly one existing node in the tree, and the
// label must not contain a '/', '*' or end with a bracketed index
// '[N]'.
func (a Augeas) Insert(path, label string, before bool) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	cLabel := C.CString(label)
	defer C.free(unsafe.Pointer(cLabel))

	var cBefore C.int
	if before {
		cBefore = 1
	} else {
		cBefore = 0
	}

	ret := C.aug_insert(a.handle, cPath, cLabel, cBefore)
	if ret == -1 {
		return a.error()
	}

	return nil
}

// Remove removes a path and all its children. Returns the number of
// entries removed. All nodes that match the given path, and their
// descendants, are removed.
func (a Augeas) Remove(path string) (num int) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	return int(C.aug_rm(a.handle, cPath))
}

// Move moves the node src to dst. src must match exactly one node in
// the tree. dst must either match exactly one node in the tree, or
// may not exist yet. If dst exists already, it and all its
// descendants are deleted. If dst does not exist yet, it and all its
// missing ancestors are created.
//
// Note that the node src always becomes the node dst: when you move /a/b
// to /x, the node /a/b is now called /x, no matter whether /x existed
// initially or not.
func (a Augeas) Move(source, destination string) error {
	cSource := C.CString(source)
	defer C.free(unsafe.Pointer(cSource))

	cDestination := C.CString(destination)
	defer C.free(unsafe.Pointer(cDestination))

	ret := C.aug_mv(a.handle, cSource, cDestination)

	if ret == -1 {
		return a.error()
	}

	return nil
}

// Match returns all paths matching a given path. The returned paths
// are sufficiently qualified to make sure that they match exactly one
// node in the current tree.
//
// Path expressions use a very simple subset of XPath: the path
// consists of a number of segments, separated by '/'; each segment can
// either be a '*', matching any tree node, or a string, optionally
// followed by an index in brackets, matching tree nodes labelled with
// exactly that string. If no index is specified, the expression matches
// all nodes with that label; the index can be a positive number N, which
// matches exactly the Nth node with that label (counting from 1), or the
// special expression 'last()' which matches the last node with the given
// label. All matches are done in fixed positions in the tree, and nothing
// matches more than one path segment.
func (a Augeas) Match(path string) (matches []string, err error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var cMatches **C.char
	cNum := C.aug_match(a.handle, cPath, &cMatches)
	num := int(cNum)

	if num < 0 {
		return nil, a.error()
	}

	q := unsafe.Pointer(cMatches)
	for i := 0; i < num; i++ {
		p := (**C.char)(q)
		matches = append(matches, C.GoString(*p))
		C.free(unsafe.Pointer(*p))
		q = unsafe.Pointer(uintptr(q) + unsafe.Sizeof(uintptr(0)))
	}

	C.free(unsafe.Pointer(cMatches))

	return
}

// Label gets the label associated with a path.
//
// Returns an error if there are no or too many matching nodes, or if
// the path is not a legal path expression.
func (a Augeas) Label(path string) (value string, err error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var cValue *C.char
	ret := C.aug_label(a.handle, cPath, &cValue)

	if ret == 1 {
		return C.GoString(cValue), nil
	} else if ret == 0 {
		return "", Error{NoMatch, "No matching node", "", ""}
	} else if ret < 0 {
		return "", a.error()
	}

	panic("Unexpected return value")
}

// Save writes all pending changes to disk.
func (a Augeas) Save() error {
	ret := C.aug_save(a.handle)
	if ret == -1 {
		return a.error()
	}

	return nil
}

// Load loads files into the tree. Which files to load and what lenses
// to use on them is specified under /augeas/load in the tree; each
// entry /augeas/load/NAME specifies a 'transform', by having itself
// exactly one child 'lens' and any number of children labelled 'incl'
// and 'excl'. The value of NAME has no meaning.
//
// The 'lens' grandchild of /augeas/load specifies which lens to use,
// and can either be the fully qualified name of a lens 'Module.lens'
// or '@Module'. The latter form means that the lens from the
// transform marked for autoloading in MODULE should be used.
//
// The 'incl' and 'excl' grandchildren of /augeas/load indicate which
// files to transform. Their value are used as glob patterns. Any file
// that matches at least one 'incl' pattern and no 'excl' pattern is
// transformed. The order of 'incl' and 'excl' entries is irrelevant.
//
// When New is first called, it populates /augeas/load with the
// transforms marked for autoloading in all the modules it finds.
//
// Before loading any files, Load will remove everything underneath
// /augeas/files and /files, regardless of whether any entries have
// been modified or not.
//
// Note that success includes the case where some files could not be
// loaded. Details of such files can be found as '/augeas//error'.
func (a Augeas) Load() error {
	ret := C.aug_load(a.handle)
	if ret == -1 {
		return a.error()
	}

	return nil
}

// Close closes the Augeas instance and frees any storage associated
// with it. After closing, the handle is invalid and can not be
// used for any more operations.
func (a Augeas) Close() {
	C.aug_close(a.handle)
}

// Version returns the Augeas version.
func (a Augeas) Version() string {
	val, _ := a.Get("/augeas/version")
	return val
}
