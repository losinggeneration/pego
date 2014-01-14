// vim: ff=unix ts=3 sw=3 noet

package pego

import (
	"errors"
	"fmt"
	"strings"
)

// Call/fallback stack

type StackEntry struct {
	p, i, c int
}

type Stack struct {
	slice []interface{}
}

func (s *Stack) Len() int {
	return len(s.slice)
}

func (s *Stack) Pop() interface{} {
	var i interface{}
	i, s.slice = s.slice[len(s.slice)-1], s.slice[:len(s.slice)-1]
	return i
}

func (s *Stack) Push(i interface{}) {
	s.slice = append(s.slice, i)
}

func (s *Stack) At(i int) interface{} {
	return s.slice[i]
}

func (s *Stack) String() string {
	ret := make([]string, 0)
	//ret.Push("[")
	ret = append(ret, "[")
	for _, v := range s.slice {
		switch v := v.(type) {
		case *StackEntry:
			//ret.Push(fmt.Sprintf("%v", *v))
			ret = append(ret, fmt.Sprintf("%v", *v))
		default:
			//ret.Push(fmt.Sprintf("%v", v))
			ret = append(ret, fmt.Sprintf("%v", v))
		}
	}
	//ret.Push("]")
	ret = append(ret, "]")
	return strings.Join(ret, " ")
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
	ret := make([]string, 0)
	//ret.Push("[")
	ret = append(ret, "[")
	var i int
	for i = 0; i < s.top; i++ {
		//ret.Push(fmt.Sprintf("%v", s.data[i]))
		ret = append(ret, fmt.Sprintf("%v", s.data[i]))
	}
	//ret.Push("<|")
	for ; i < len(s.data); i++ {
		//ret.Push(fmt.Sprintf("%v", s.data[i]))
		ret = append(ret, fmt.Sprintf("%v", s.data[i]))
	}
	//ret.Push("]")
	ret = append(ret, "]")
	return strings.Join(ret, " ")
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
func Match(program *Pattern, input string) (interface{}, error, int) {
	const FAIL = -1
	var p, i, c int
	stack := &Stack{make([]interface{}, 0)}
	captures := NewCapStack()
	for p = 0; p < len(*program); {
		if p == FAIL {
			// Unroll stack until a fallback point is reached
			if stack.Len() == 0 {
				return nil, errors.New("Stack is empty"), i
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
			return nil, errors.New(fmt.Sprintf("Unimplemented: %#v", (*program)[p])), i
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
		case *ISpan:
			for i < len(input) && op.Has(input[i]) {
				i++
			}
			p++
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
			return nil, errors.New(fmt.Sprintf("Unresolved name: %q", op.name)), i
		case *ICall:
			stack.Push(p + 1)
			p += op.offset
		case *IReturn:
			if stack.Len() == 0 {
				return nil, errors.New("Return with empty stack"), i
			}
			e, ok := stack.Pop().(int)
			if !ok {
				return nil, errors.New("Expecting return address on stack; Found failure address"), i
			}
			p = e
		case *ICommit:
			if stack.Len() == 0 {
				return nil, errors.New("Commit with empty stack"), i
			}
			_, ok := stack.Pop().(*StackEntry)
			if !ok {
				return nil, errors.New("Expecting failure address on stack; Found return address"), i
			}
			p += op.offset
		case *IPartialCommit:
			if stack.Len() == 0 {
				return nil, errors.New("PartialCommit with empty stack"), i
			}
			e, ok := stack.At(stack.Len() - 1).(*StackEntry)
			if !ok {
				return nil, errors.New("Expecting failure address on stack; Found return address"), i
			}
			e.i = i
			e.c = captures.Mark()
			p += op.offset
		case *IBackCommit:
			if stack.Len() == 0 {
				return nil, errors.New("BackCommit with empty stack"), i
			}
			e, ok := stack.Pop().(*StackEntry)
			if !ok {
				return nil, errors.New("Expecting failure address on stack; Found return address"), i
			}
			i = e.i
			captures.Rollback(e.c)
			p += op.offset
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
		case *IFailTwice:
			if stack.Len() == 0 {
				return nil, errors.New("IFailTwice with empty stack"), i
			}
			e, ok := stack.Pop().(*StackEntry)
			if !ok {
				return nil, errors.New("Expecting failure address on stack; Found return address"), i
			}
			i = e.i
			captures.Rollback(e.c)
			p = FAIL
		case *IGiveUp:
			return nil, nil, i
		case *IEnd:
			caps := captures.Pop(captures.top)
			var ret interface{}
			if len(caps) > 0 && caps[0] != nil {
				ret = caps[0].value
			}
			return ret, nil, i
		}
	}
	return nil, errors.New("Invalid jump or missing End instruction."), i
}
