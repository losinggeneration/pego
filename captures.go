// vim: ff=unix ts=3 sw=3 noet

package main

import (
	"container/vector"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)


// Interface for all capture handlers
type CaptureHandler interface {
	Process(input string, start, end int, captures *CapStack, subcaps int) (interface{}, os.Error)
}

// Captures the matched substring
type SimpleCapture struct{}

func (h *SimpleCapture) String() string { return "simple" }
func (h *SimpleCapture) Process(input string, start, end int, captures *CapStack, subcaps int) (interface{}, os.Error) {
	return input[start:end], nil
}

// Captures the current input position
type PositionCapture struct{}

func (h *PositionCapture) String() string { return "position" }
func (h *PositionCapture) Process(input string, start, end int, captures *CapStack, subcaps int) (interface{}, os.Error) {
	return start, nil
}

// Captures a constant value
type ConstCapture struct {
	value interface{}
}

func (h *ConstCapture) String() string {
	return fmt.Sprintf("const(%v)", h.value)
}
func (h *ConstCapture) Process(input string, start, end int, captures *CapStack, subcaps int) (interface{}, os.Error) {
	return h.value, nil
}

// Captures a list of all sub-captures
type ListCapture struct{}

func (h *ListCapture) String() string { return "list" }
func (h *ListCapture) Process(input string, start, end int, captures *CapStack, subcaps int) (interface{}, os.Error) {
	subs := captures.Pop(subcaps)
	ret := make([]interface{}, len(subs))
	for i := range subs {
		ret[i] = subs[i].value
	}
	return ret, nil
}

// Calls a function with all sub-captures, and captures the return value.
// If functions reports an error, let it bubble up.
type FunctionCapture struct {
	function func([]*CaptureResult) (interface{}, os.Error)
}

func (h *FunctionCapture) String() string { return "function" }
func (h *FunctionCapture) Process(input string, start, end int, captures *CapStack, subcaps int) (interface{}, os.Error) {
	subs := captures.Pop(subcaps)
	return h.function(subs)
}

// Capture a string created from a format applied to the sub-captures.
type StringCapture struct {
	format string
}

func (h *StringCapture) String() string {
	return fmt.Sprintf("string(%q)", h.format)
}
func (h *StringCapture) Process(input string, start, end int, captures *CapStack, subcaps int) (interface{}, os.Error) {
	subs := captures.Pop(subcaps)
	p := regexp.MustCompile(`{[0-9]+}|{{|{}`)
	var err os.Error
	ret := p.ReplaceAllStringFunc(h.format, func(s string) string {
		switch s[1] {
		case '{':
			return "{"
		case '}':
			return "}"
		}
		if err != nil {
			return "<ERROR>"
		}
		var i int
		i, err = strconv.Atoi(s[1 : len(s)-1])
		if err == nil && i >= len(subs) {
			err = os.ErrorString("String format number out of range")
		}
		if err != nil {
			return "<ERROR>"
		}
		return fmt.Sprintf("%v", subs[i].value)
	})
	return ret, err
}

// Capture a string, with all sub-captures replaced by their string-representation.
// XXX(mizardx): Better explaination?
type SubstCapture struct{}

func (h *SubstCapture) String() string { return "subst" }
func (h *SubstCapture) Process(input string, start, end int, captures *CapStack, subcaps int) (interface{}, os.Error) {
	subs := captures.Pop(subcaps)
	ret := new(vector.StringVector)
	pos := start
	for _, c := range subs {
		if c.start > pos {
			ret.Push(input[pos:c.start])
		}
		ret.Push(fmt.Sprintf("%v", c.value))
		pos = c.end
	}
	if pos < end {
		ret.Push(input[pos:end])
	}
	return strings.Join(ret.Data(), ""), nil
}
