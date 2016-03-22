package augeas

// #cgo pkg-config: libxml-2.0 augeas
// #include <augeas.h>
import "C"
import (
	"fmt"
)

// Negative codes are specific to this binding, positive codes are
// from Augeas
type ErrorCode int

const (
	CouldNotInitialize ErrorCode = -2
	NoMatch                      = -1

	// No error
	NoError = 0

	// Out of memory
	ENOMEM

	// Internal (to augeas itself) error (bug)
	EINTERNAL

	// Invalid path expression
	EPATHX

	// No match for path expression
	ENOMATCH

	// Too many matches for path expression
	EMMATCH

	// Syntax error in lens file
	ESYNTAX

	// Lens lookup failed
	ENOLENS

	// Multiple transforms
	EMXFM

	// No span for this node
	ENOSPAN

	// Cannot move node into its descendant
	EMVDESC

	// Failed to execute command
	ECMDRUN

	// Invalid argument in function call
	EBADARG
)

type Error struct {
	Code ErrorCode

	// Human-readable error message
	Message string

	// Human-readable message elaborating the error. For example, when
	// the error code is AUG_EPATHX, this will explain how the path
	// expression is invalid
	MinorMessage string

	// Details about the error. For example, for AUG_EPATHX, indicates
	// where in the path expression the error occurred.
	Details string
}

func (err Error) Error() string {
	return fmt.Sprintf("Message: %s - Minor message: %s - Details: %s",
		err.Message, err.MinorMessage, err.Details)
}

func (a Augeas) error() error {
	code := a.errorCode()
	if code == NoError {
		return nil
	}

	return Error{code, a.errorMessage(), a.errorMinorMessage(), a.errorDetails()}
}

func (a Augeas) errorCode() ErrorCode {
	return ErrorCode(C.aug_error(a.handle))
}

func (a Augeas) errorMessage() string {
	return C.GoString(C.aug_error_message(a.handle))
}

func (a Augeas) errorMinorMessage() string {
	return C.GoString(C.aug_error_minor_message(a.handle))
}

func (a Augeas) errorDetails() string {
	return C.GoString(C.aug_error_details(a.handle))
}
