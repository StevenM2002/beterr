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
	FnName string   `json:"fn_name"`
	Args   []string `json:"args"`
	Msg    string   `json:"msg"`
	Inner  any      `json:"inner"`
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
		Args:   []string{},
		Msg:    m,
		Inner:  errStr,
	}

	// See if we can unmarshal inner into PrintOutput
	var prevO printOutput
	myErr := json.Unmarshal([]byte(err.Error()), &prevO)
	if myErr == nil {
		o.Inner = prevO
	}

	for _, c := range w.A {
		if _, ok := c.(context.Context); ok {
			o.Args = append(o.Args, "ctx")
			continue
		}
		o.Args = append(o.Args, StructString(c))
	}
	return fmt.Errorf("%s", StructString(o))
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
