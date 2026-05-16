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

// E formats an error with debugging context including function name, arguments, and message.
// It wraps the original error with structured debugging information that can be chained.
func (w *Wrap) E(err error, msg ...string) error {
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
	return fmt.Errorf("%s", StructString(o))
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
