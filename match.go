// vim: ff=unix ts=3 sw=3 noet

package main

import (
	"fmt"
	"container/vector"
	"strings"
	"os"
)

// Call/fallback stack

type StackEntry struct {
	p, i, c int
}

type Stack struct {
	*vector.Vector
}

func (s *Stack) String() string {
	ret := new(vector.StringVector)
	ret.Push("[")
	for v := range s.Iter() {
		switch v := v.(type) {
		case *StackEntry:
			ret.Push(fmt.Sprintf("%v", *v))
		default:
			ret.Push(fmt.Sprintf("%v", v))
		}
	}
	ret.Push("]")
	return strings.Join(ret.Data(), " ")
}

// === Capture stack ===

type CapStack struct {
	data []*CaptureEntry
	top  int
}

type CaptureEntry struct {
	p, start, end int
	handler       CaptureHandler
	value         interface{}
}

func NewCapStack() *CapStack {
	return &CapStack{}
}

func (s *CapStack) String() string {
	ret := new(vector.StringVector)
	ret.Push("[")
	var i int
	for i = 0; i < s.top; i++ {
		ret.Push(fmt.Sprintf("%v", s.data[i]))
	}
	ret.Push("<|")
	for ; i < len(s.data); i++ {
		ret.Push(fmt.Sprintf("%v", s.data[i]))
	}
	ret.Push("]")
	return strings.Join(ret.Data(), " ")
}

// Open and return an new capture
func (s *CapStack) Open(p int, start int) *CaptureEntry {
	if s.data == nil {
		s.data = make([]*CaptureEntry, 8)
	} else if len(s.data) == s.top {
		newData := make([]*CaptureEntry, 2*len(s.data)+1)
		copy(newData, s.data)
		s.data = newData
	}
	s.data[s.top] = &CaptureEntry{p: p, start: start, end: -1}
	s.top++
	return s.data[s.top-1]
}

// Close and return the closest open capture
func (s *CapStack) Close(end int) (*CaptureEntry, int) {
	var i int
	for i = s.top - 1; i >= 0; i-- {
		if s.data[i].end == -1 {
			s.data[i].end = end
			return s.data[i], s.top - i - 1
		}
	}
	return nil, 0
}

// Used when returning the values
// Similar to CaptureEntry, but without some internal values
type CaptureResult struct {
	start, end int
	value      interface{}
}

// Pop and return the top `count` captures
func (s *CapStack) Pop(count int) []*CaptureResult {
	subcaps := make([]*CaptureResult, count)
	i := s.top - count
	for j := 0; j < count; j++ {
		subcaps[j] = &CaptureResult{s.data[i+j].start, s.data[i+j].end, s.data[i+j].value}
	}
	s.top -= count
	return subcaps
}

// Create and return a mark
func (s *CapStack) Mark() int {
	return s.top
}

// Rollback to a previous mark
func (s *CapStack) Rollback(mark int) {
	s.top = mark
}

// Main match function
func match(program *Pattern, input string) (interface{}, os.Error, int) {
	const FAIL = -1
	var p, i, c int
	stack := &Stack{new(vector.Vector)}
	captures := NewCapStack()
	for p = 0; p < len(*program); {
		if p == FAIL {
			// Unroll stack until a fallback point is reached
			if stack.Len() == 0 {
				return nil, os.ErrorString("Stack is empty"), i
			}
			switch e := stack.Pop().(type) {
			case *StackEntry:
				p, i, c = e.p, e.i, e.c
				captures.Rollback(c)
			case int:
			}
			continue
		}
		fmt.Printf("%6d  %s\n", p, (*program)[p])
		switch op := (*program)[p].(type) {
		default:
			return nil, os.ErrorString(fmt.Sprintf("Unimplemented: %#v", (*program)[p])), i
		case nil:
			// Noop
			p++
		case *IChar:
			if i < len(input) && input[i] == op.char {
				p++
				i++
			} else {
				p = FAIL
			}
		case *ICharset:
			if i < len(input) && op.Has(input[i]) {
				p++
				i++
			} else {
				p = FAIL
			}
		case *IAny:
			if i+op.count > len(input) {
				p = FAIL
			} else {
				p++
				i += op.count
			}
		case *IJump:
			p += op.offset
		case *IChoice:
			stack.Push(&StackEntry{p + op.offset, i, captures.Mark()})
			p++
		case *IOpenCall:
			return nil, os.ErrorString(fmt.Sprintf("Unresolved name: %q", op.name)), i
		case *ICall:
			stack.Push(p + 1)
			p += op.offset
		case *IReturn:
			if stack.Len() == 0 {
				return nil, os.ErrorString("Return with empty stack"), i
			}
			switch e := stack.Pop().(type) {
			case *StackEntry:
				return nil, os.ErrorString("Expecting return address on stack; Found failure address"), i
			case int:
				p = e
			}
		case *ICommit:
			if stack.Len() == 0 {
				return nil, os.ErrorString("Commit with empty stack"), i
			}
			switch stack.Pop().(type) {
			case *StackEntry:
				p += op.offset
			case int:
				return nil, os.ErrorString("Expecting failure address on stack; Found return address"), i
			}
		case *IOpenCapture:
			e := captures.Open(p, i-op.capOffset)
			if op.handler == nil {
				e.handler = &SimpleCapture{}
			} else {
				e.handler = op.handler
			}
			p++
		case *ICloseCapture:
			e, count := captures.Close(i - op.capOffset)
			v, err := e.handler.Process(input, e.start, e.end, captures, count)
			if err != nil {
				return nil, err, i
			}
			e.value = v
			p++
		case *IFullCapture:
			e := captures.Open(p, i-op.capOffset)
			if op.handler == nil {
				e.handler = &SimpleCapture{}
			} else {
				e.handler = op.handler
			}
			captures.Close(i)
			v, err := e.handler.Process(input, e.start, e.end, captures, 0)
			if err != nil {
				return nil, err, i
			}
			e.value = v
			p++
		case *IEmptyCapture:
			e := captures.Open(p, i-op.capOffset)
			if op.handler == nil {
				e.handler = &SimpleCapture{}
			} else {
				e.handler = op.handler
			}
			captures.Close(i - op.capOffset)
			v, err := e.handler.Process(input, e.start, e.end, captures, 0)
			if err != nil {
				return nil, err, i
			}
			e.value = v
			p++
		case *IFail:
			p = FAIL
		case *IEnd:
			caps := captures.Pop(captures.top)
			var ret interface{}
			if len(caps) > 0 && caps[0] != nil {
				ret = caps[0].value
			}
			return ret, nil, i
		}
	}
	return nil, os.ErrorString("Invalid jump or missing End instruction."), i
}

// Test code
func main() {
	pat := Grm("S", map[string]*Pattern{
		"S": Ref("A").Clist(),
		"A": Seq(
			NegSet("()").Rep(0, -1),
			Seq(
				Ref("B"),
				NegSet("()").Rep(0, -1),
			).Rep(0, -1)).Csimple(),
		"B": Seq(
			"(", Ref("A"), ")"),
	})
	fmt.Println("Compiled pattern:")
	fmt.Println(pat)
	fmt.Println()
	tests := []string{
		"x", "(x)", "a(b(c)d(e)f)g", ")",
	}
	for _, s := range tests {
		fmt.Printf("\n\n=== MATCHING %q ===\n", s)
		fmt.Println("Trace:")
		r, err, pos := match(pat, s)
		fmt.Println()
		if r != nil {
			fmt.Printf("Return value: %v\n", r)
		}
		if err != nil {
			fmt.Printf("Error: %#v\n", err)
		}
		fmt.Printf("End position: %d\n", pos)
		if pos != len(s) {
			fmt.Println("Failed to match whole input")
		}
	}
}
