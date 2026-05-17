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
// arguments, message, and any nested chain), plus a pre-built shallow view
// of the immediate previous error for Top to return.
type Error struct {
	inner error       // full underlying err, returned by Unwrap so errors.Is/As traverse
	out   printOutput // this layer's own structured data (with recursively expanded Inner)
	top   error       // shallow rendition of inner: just the immediate previous, no further beterr chain
	json  string      // cached StructString(out)
}

// Error returns the JSON-serialised wrap including any nested chain.
func (e *Error) Error() string {
	return e.json
}

// Unwrap returns the underlying error so errors.Is and errors.As traverse
// beterr wraps the same way they traverse fmt.Errorf("%w", ...) wraps.
// Sentinel checks against the original error continue to work after any
// number of beterr layers have been added.
func (e *Error) Unwrap() error {
	return e.inner
}

// Top returns the immediate previous error this wrap was applied to, with
// any further beterr chain stripped. If the wrapped error was itself a
// beterr Error, Top returns a shallow rendition of it — that layer's own
// fn_name, args, and msg, with its inner set to null. If the wrapped error
// was a plain error, Top returns it as-is. If the wrap was created over a
// nil error, Top returns nil.
//
// Use Top when forwarding an error to a caller who shouldn't see the full
// nested error stack — for example, returning errors from an RPC handler:
//
//	wrapped := beterr.W(userID).E(err, "failed to process request")
//	log.Println(wrapped)                            // full nested debug JSON
//	return connect.NewError(code, wrapped.Top())    // only the last error
func (e *Error) Top() error {
	return e.top
}

// E formats an error with debugging context including function name, arguments, and message.
// It wraps the original error with structured debugging information that can be chained.
// The returned *Error implements the error interface, exposes Unwrap so
// errors.Is/As traverse the chain, and exposes Top to release only the
// immediate previous error without leaking the deeper stack.
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

	var topErr error
	if be, ok := err.(*Error); ok {
		shallowOut := printOutput{
			FnName: be.out.FnName,
			Args:   be.out.Args,
			Msg:    be.out.Msg,
			Inner:  nil,
		}
		topErr = &Error{
			out:  shallowOut,
			json: StructString(shallowOut),
		}
	} else if err != nil {
		topErr = err
	}

	return &Error{
		inner: err,
		out:   o,
		top:   topErr,
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
