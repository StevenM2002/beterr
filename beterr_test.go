package beterr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestW_StoresArgs(t *testing.T) {
	w := W(1, "two", 3.0)
	if len(w.A) != 3 {
		t.Fatalf("expected 3 args, got %d", len(w.A))
	}
	if w.A[0] != 1 || w.A[1] != "two" || w.A[2] != 3.0 {
		t.Errorf("args not stored verbatim: %+v", w.A)
	}
}

func TestW_NoArgs(t *testing.T) {
	w := W()
	if w == nil {
		t.Fatal("W() returned nil")
	}
	if len(w.A) != 0 {
		t.Errorf("expected zero args, got %d", len(w.A))
	}
}

func TestE_BasicError(t *testing.T) {
	w := W("arg1", 42)
	err := w.E(fmt.Errorf("original"), "something failed")
	if err == nil {
		t.Fatal("expected non-nil error")
	}

	var out printOutput
	if jerr := json.Unmarshal([]byte(err.Error()), &out); jerr != nil {
		t.Fatalf("output not valid JSON: %v\noutput: %s", jerr, err.Error())
	}
	if !strings.Contains(out.FnName, "TestE_BasicError") {
		t.Errorf("fn_name should reference caller, got %q", out.FnName)
	}
	if out.Msg != "something failed" {
		t.Errorf("msg = %q, want %q", out.Msg, "something failed")
	}
	if len(out.Args) != 2 {
		t.Errorf("expected 2 args, got %d: %v", len(out.Args), out.Args)
	}
	if inner, ok := out.Inner.(string); !ok || inner != "original" {
		t.Errorf("inner = %v, want %q", out.Inner, "original")
	}
}

// Regression test: w.E(nil, ...) used to panic on a nil pointer dereference
// because err.Error() was called before the nil check on the unmarshal path.
func TestE_NilError(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("E panicked on nil error: %v", r)
		}
	}()

	w := W("arg")
	err := w.E(nil, "wrapped nil")
	if err == nil {
		t.Fatal("E should still produce an error even when input is nil")
	}

	var out printOutput
	if jerr := json.Unmarshal([]byte(err.Error()), &out); jerr != nil {
		t.Fatalf("output not valid JSON: %v", jerr)
	}
	if inner, ok := out.Inner.(string); !ok || inner != "nil err" {
		t.Errorf("inner = %v, want %q", out.Inner, "nil err")
	}
}

func TestE_NestedError(t *testing.T) {
	inner := W("inner-arg").E(fmt.Errorf("root cause"), "inner failed")
	outer := W("outer-arg").E(inner, "outer failed")

	var out printOutput
	if jerr := json.Unmarshal([]byte(outer.Error()), &out); jerr != nil {
		t.Fatalf("output not valid JSON: %v", jerr)
	}
	nested, ok := out.Inner.(map[string]any)
	if !ok {
		t.Fatalf("expected nested object as inner, got %T: %v", out.Inner, out.Inner)
	}
	if nested["msg"] != "inner failed" {
		t.Errorf("nested msg = %v, want %q", nested["msg"], "inner failed")
	}
	if rootInner, _ := nested["inner"].(string); rootInner != "root cause" {
		t.Errorf("expected root cause at deepest inner, got %v", nested["inner"])
	}
}

func TestE_ContextArgRendered(t *testing.T) {
	ctx := context.Background()
	err := W(ctx, "other").E(fmt.Errorf("boom"), "ctx test")

	var out printOutput
	if jerr := json.Unmarshal([]byte(err.Error()), &out); jerr != nil {
		t.Fatalf("output not valid JSON: %v", jerr)
	}
	if len(out.Args) != 2 {
		t.Fatalf("expected 2 args, got %d: %v", len(out.Args), out.Args)
	}
	if string(out.Args[0]) != `"ctx"` {
		t.Errorf("context arg should render as JSON %q, got %s", `"ctx"`, out.Args[0])
	}
	if string(out.Args[1]) != `"other"` {
		t.Errorf("string arg should render as JSON %q, got %s", `"other"`, out.Args[1])
	}
}

// Args should land at their real JSON type — numbers as numbers, objects as
// objects, strings as strings — with no double-escaping pile-up.
func TestE_ArgsRetainJSONTypes(t *testing.T) {
	type payload struct {
		Retries int `json:"retries"`
	}
	err := W(42, "hello", payload{Retries: 3}).E(fmt.Errorf("x"), "")

	var out printOutput
	if jerr := json.Unmarshal([]byte(err.Error()), &out); jerr != nil {
		t.Fatalf("output not valid JSON: %v", jerr)
	}
	if len(out.Args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(out.Args))
	}
	if string(out.Args[0]) != "42" {
		t.Errorf("number arg = %s, want 42", out.Args[0])
	}
	if string(out.Args[1]) != `"hello"` {
		t.Errorf("string arg = %s, want \"hello\"", out.Args[1])
	}
	if string(out.Args[2]) != `{"retries":3}` {
		t.Errorf("struct arg = %s, want {\"retries\":3}", out.Args[2])
	}
}

// Nested args from a previous level should survive a round-trip through E
// without re-stringification.
func TestE_NestedArgsCleanThroughRoundTrip(t *testing.T) {
	inner := W(map[string]int{"retries": 3}).E(fmt.Errorf("disk full"), "inner")
	outer := W("outer-arg").E(inner, "outer")

	// The raw output should contain the inner args as a real JSON object,
	// not as an escaped string like "{\"retries\":3}".
	raw := outer.Error()
	if strings.Contains(raw, `"{\"retries\":3}"`) {
		t.Errorf("nested args were re-stringified: %s", raw)
	}
	if !strings.Contains(raw, `{"retries":3}`) {
		t.Errorf("expected raw object form in nested args, got: %s", raw)
	}
}

func TestE_NoMsg(t *testing.T) {
	err := W().E(fmt.Errorf("x"))
	var out printOutput
	if jerr := json.Unmarshal([]byte(err.Error()), &out); jerr != nil {
		t.Fatalf("output not valid JSON: %v", jerr)
	}
	if out.Msg != "" {
		t.Errorf("expected empty msg, got %q", out.Msg)
	}
}

func TestE_MultipleMsgPartsJoined(t *testing.T) {
	err := W().E(fmt.Errorf("x"), "failed", "to", "do thing")
	var out printOutput
	if jerr := json.Unmarshal([]byte(err.Error()), &out); jerr != nil {
		t.Fatalf("output not valid JSON: %v", jerr)
	}
	if out.Msg != "failed to do thing" {
		t.Errorf("expected joined msg, got %q", out.Msg)
	}
}

func TestStructString_Marshalable(t *testing.T) {
	type S struct {
		A int    `json:"a"`
		B string `json:"b"`
	}
	got := StructString(S{A: 1, B: "x"})
	want := `{"a":1,"b":"x"}`
	if got != want {
		t.Errorf("StructString = %q, want %q", got, want)
	}
}

func TestStructString_FallbackOnUnmarshalable(t *testing.T) {
	ch := make(chan int)
	got := StructString(ch)
	if strings.HasPrefix(got, "{") || strings.HasPrefix(got, "\"") {
		t.Errorf("expected non-JSON fallback for channel, got %q", got)
	}
	if got == "" {
		t.Error("expected non-empty fallback string")
	}
}

func TestTop_ReturnsErrorPassedToE(t *testing.T) {
	original := fmt.Errorf("root cause")
	wrapped := W("arg1", 42).E(original, "user-friendly message")

	top := wrapped.Top()
	if top != original {
		t.Errorf("Top = %v, want the same error instance passed to E", top)
	}
}

func TestTop_ReturnsNilWhenEWrappedNil(t *testing.T) {
	wrapped := W().E(nil, "no underlying err")
	if top := wrapped.Top(); top != nil {
		t.Errorf("Top = %v, want nil", top)
	}
}

func TestTop_StripsDeeperBeterrChainFromImmediatePrevious(t *testing.T) {
	root := fmt.Errorf("root cause")
	inner := W("inner-arg").E(root, "inner msg")
	outer := W("outer-arg").E(inner, "outer msg")

	top := outer.Top()
	if top == nil {
		t.Fatal("Top returned nil")
	}

	if strings.Contains(top.Error(), "root cause") {
		t.Errorf("Top should not leak deeper chain (root cause), got: %s", top.Error())
	}
	if strings.Contains(top.Error(), "outer msg") {
		t.Errorf("Top should not include this layer's own msg, got: %s", top.Error())
	}
	if !strings.Contains(top.Error(), "inner msg") {
		t.Errorf("Top should include immediate previous msg, got: %s", top.Error())
	}

	var o printOutput
	if err := json.Unmarshal([]byte(top.Error()), &o); err != nil {
		t.Fatalf("Top output is not JSON: %v\noutput: %s", err, top.Error())
	}
	if o.Msg != "inner msg" {
		t.Errorf("Top out.Msg = %q, want %q", o.Msg, "inner msg")
	}
	if o.Inner != nil {
		t.Errorf("Top out.Inner should be nil to indicate stripped chain, got: %v", o.Inner)
	}
	if !strings.Contains(o.FnName, "TestTop_StripsDeeperBeterrChain") {
		t.Errorf("Top out.FnName should be the immediate previous fn, got %q", o.FnName)
	}
}

func TestTop_ChainOfThreeWrapsExposesOnlyMiddle(t *testing.T) {
	root := fmt.Errorf("root cause")
	layer1 := W().E(root, "layer1 msg")
	layer2 := W().E(layer1, "layer2 msg")
	layer3 := W().E(layer2, "layer3 msg")

	top := layer3.Top()
	if strings.Contains(top.Error(), "layer3 msg") {
		t.Errorf("Top should not contain this layer's own msg, got: %s", top.Error())
	}
	if !strings.Contains(top.Error(), "layer2 msg") {
		t.Errorf("Top should contain immediate previous (layer2) msg, got: %s", top.Error())
	}
	if strings.Contains(top.Error(), "layer1 msg") {
		t.Errorf("Top should not contain deeper layer1 msg, got: %s", top.Error())
	}
	if strings.Contains(top.Error(), "root cause") {
		t.Errorf("Top should not contain root cause, got: %s", top.Error())
	}
}

func TestTop_DoesNotLeakWrapJSON(t *testing.T) {
	original := fmt.Errorf("plain message")
	wrapped := W("secret-arg").E(original, "top msg")

	top := wrapped.Top()
	if strings.Contains(top.Error(), "secret-arg") {
		t.Errorf("Top should not leak wrap args: %q", top.Error())
	}
	if strings.Contains(top.Error(), "top msg") {
		t.Errorf("Top should not leak wrap msg: %q", top.Error())
	}
	if strings.Contains(top.Error(), "fn_name") {
		t.Errorf("Top should not leak fn_name: %q", top.Error())
	}
}

func TestErrorImplementsErrorInterface(t *testing.T) {
	var _ error = (*Error)(nil)
	var err error = W().E(fmt.Errorf("x"), "msg")
	if err == nil {
		t.Fatal("E result should not be nil when assigned to error interface")
	}
}

func TestErrorsIsTraversesSingleBeterrWrap(t *testing.T) {
	sentinel := errors.New("the sentinel")
	wrapped := W("arg").E(sentinel, "context")

	if !errors.Is(wrapped, sentinel) {
		t.Errorf("errors.Is should find sentinel through a single beterr wrap")
	}
}

func TestErrorsIsTraversesNestedBeterrWraps(t *testing.T) {
	sentinel := errors.New("the sentinel")
	layer1 := W().E(sentinel, "layer 1")
	layer2 := W().E(layer1, "layer 2")
	layer3 := W().E(layer2, "layer 3")

	if !errors.Is(layer3, sentinel) {
		t.Errorf("errors.Is should find sentinel through three beterr wraps")
	}
}

func TestErrorsIsTraversesMixedFmtAndBeterrChain(t *testing.T) {
	sentinel := errors.New("the sentinel")
	fmtWrapped := fmt.Errorf("%w: with context", sentinel)
	beterrWrapped := W("arg").E(fmtWrapped, "outer")

	if !errors.Is(beterrWrapped, sentinel) {
		t.Errorf("errors.Is should traverse beterr -> fmt.Errorf(%%w) -> sentinel")
	}
}

func TestErrorsAsRetrievesUnderlyingTypedError(t *testing.T) {
	wantErr := &typedTestErr{msg: "hello"}
	wrapped := W().E(wantErr, "outer")

	var got *typedTestErr
	if !errors.As(wrapped, &got) {
		t.Fatal("errors.As should retrieve underlying *typedTestErr through beterr wrap")
	}
	if got != wantErr {
		t.Errorf("errors.As recovered = %v, want %v", got, wantErr)
	}
}

type typedTestErr struct{ msg string }

func (e *typedTestErr) Error() string { return e.msg }
