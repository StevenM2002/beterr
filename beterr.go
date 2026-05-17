// Package beterr provides structured error handling and debugging utilities for Go applications.
// It offers enhanced error formatting with function call context and argument inspection.
package beterr

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
)

// printOutput represents the structured output format for debug information.
type printOutput struct {
	FnName string            `json:"fn_name"`
	Args   []json.RawMessage `json:"args"`
	Msg    string            `json:"msg"`
	Inner  any               `json:"inner"`
}

// Wrap provides debugging functionality with argument tracking.
// The A field stores arguments that will be included in error output.
type Wrap struct {
	// A holds arguments to be included in debug output
	A []any
}

// Error is the error returned by Wrap.E. It carries the original underlying
// error alongside a JSON-serialised description of the wrap (function name,
// arguments, message, and any nested chain).
type Error struct {
	inner error
	json  string
}

// Error returns the JSON-serialised wrap including any nested chain.
func (e *Error) Error() string {
	return e.json
}

// Top returns the error that was passed to E to produce this wrap.
// It peels off the current beterr layer, returning the underlying error
// exactly as it was given. If the underlying error is itself a beterr wrap,
// call Top again to peel the next layer. If the wrap was created over a
// nil error, Top returns nil.
//
// Use Top when forwarding an error to a caller who shouldn't see the
// structured debug JSON the wrap adds — for example, returning errors from
// an RPC handler:
//
//	wrapped := beterr.W(userID).E(err, "failed to process request")
//	log.Println(wrapped)                            // full debug JSON
//	return connect.NewError(code, wrapped.Top())    // underlying err only
func (e *Error) Top() error {
	return e.inner
}

// Unwrap returns the underlying error so errors.Is and errors.As traverse
// beterr wraps the same way they traverse fmt.Errorf("%w", ...) wraps.
// Sentinel checks against the original error continue to work after any
// number of beterr layers have been added.
func (e *Error) Unwrap() error {
	return e.inner
}

// E formats an error with debugging context including function name, arguments, and message.
// It wraps the original error with structured debugging information that can be chained.
// The returned *Error implements the error interface and exposes Top to peel
// the wrap and recover the underlying error.
func (w *Wrap) E(err error, msg ...string) *Error {
	m := strings.Join(msg, " ")
	pc, _, _, ok := runtime.Caller(1)
	fnName := "unknown"
	if ok {
		fnName = runtime.FuncForPC(pc).Name()
	}
	errStr := "nil err"
	if err != nil {
		errStr = err.Error()
	}
	o := printOutput{
		FnName: fnName,
		Args:   []json.RawMessage{},
		Msg:    m,
		Inner:  errStr,
	}

	if err != nil {
		var prevO printOutput
		if json.Unmarshal([]byte(errStr), &prevO) == nil {
			o.Inner = prevO
		}
	}

	for _, c := range w.A {
		if _, ok := c.(context.Context); ok {
			o.Args = append(o.Args, json.RawMessage(`"ctx"`))
			continue
		}
		b, jerr := json.Marshal(c)
		if jerr != nil {
			b, _ = json.Marshal(fmt.Sprintf("%+v", c))
		}
		o.Args = append(o.Args, b)
	}
	return &Error{
		inner: err,
		json:  StructString(o),
	}
}

// W creates a new Wrap instance with the provided arguments.
// This is a convenience function that internally calls Wrap{A: []any{...}}.
// It accepts any number of arguments which will be included in error output for debugging.
//
// Example usage:
//   w := W(userID, requestData, config)
//   return w.E(err, "failed to process request")
func W(args ...any) *Wrap {
	return &Wrap{A: args}
}

// StructString converts any value to a JSON string representation.
// If JSON marshaling fails, it falls back to the default string format.
func StructString(v any) string {
	s, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%+v", v) // Fallback to default string representation
	}
	return string(s)
}
