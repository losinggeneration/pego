package main

import (
   "container/vector"
   "fmt"
   "os"
   "regexp"
   "strconv"
   "strings"
)


type CaptureHandler interface {
   Process(input string, start, end int, captures *CapStack, subcaps int) (interface{}, os.Error)
}

type SimpleCapture struct {}
func (h *SimpleCapture) String() string { return "simple" }
func (h *SimpleCapture) Process(input string, start, end int, captures *CapStack, subcaps int) (interface{}, os.Error) {
   return input[start:end], nil
}

type PositionCapture struct {}
func (h *PositionCapture) String() string { return "position" }
func (h *PositionCapture) Process(input string, start, end int, captures *CapStack, subcaps int) (interface{}, os.Error) {
   return start, nil
}

type ConstCapture struct {
   value interface{}
}
func (h *ConstCapture) String() string {
   return fmt.Sprintf("const(%v)", h.value)
}
func (h *ConstCapture) Process(input string, start, end int, captures *CapStack, subcaps int) (interface{}, os.Error) {
   return h.value, nil
}

type ListCapture struct {}
func (h *ListCapture) String() string { return "list" }
func (h *ListCapture) Process(input string, start, end int, captures *CapStack, subcaps int) (interface{}, os.Error) {
   subs := captures.Pop(subcaps)
   ret := make([]interface{},len(subs))
   for i := range subs {
      ret[i] = subs[i].value
   }
   return ret, nil
}

type FunctionCapture struct {
   function func([]*CaptureResult) (interface{},os.Error)
}
func (h *FunctionCapture) String() string { return "function" }
func (h *FunctionCapture) Process(input string, start, end int, captures *CapStack, subcaps int) (interface{}, os.Error) {
   subs := captures.Pop(subcaps)
   return h.function(subs)
}

type StringCapture struct {
   pattern string
}
func (h *StringCapture) String() string {
   return fmt.Sprintf("string(%q)", h.pattern)
}
func (h *StringCapture) Process(input string, start, end int, captures *CapStack, subcaps int) (interface{}, os.Error) {
   subs := captures.Pop(subcaps)
   p := regexp.MustCompile(`{[0-9]+}|{{|{}`)
   var err os.Error
   ret := p.ReplaceAllStringFunc(h.pattern, func(s string) string {
      switch s[1] {
      case '{': return "{"
      case '}': return "}"
      }
      if err != nil { return "<ERROR>" }
      var i int
      i,err = strconv.Atoi(s[1:len(s)-1])
      if err == nil && i >= len(subs) { err = os.ErrorString("String format number out of range") }
      if err != nil { return "<ERROR>" }
      return fmt.Sprintf("%v", subs[i].value)
   })
   return ret, nil
}

type SubstCapture struct {}
func (h *SubstCapture) String() string { return "subst" }
func (h *SubstCapture) Process(input string, start, end int, captures *CapStack, subcaps int) (interface{}, os.Error) {
   subs := captures.Pop(subcaps)
   ret := new(vector.StringVector)
   pos := start
   for _,c := range subs {
      if c.start > pos { ret.Push(input[pos:c.start]) }
      ret.Push(fmt.Sprintf("%v",c.value))
      pos = c.end
   }
   if pos < end {
      ret.Push(input[pos:end])
   }
   return strings.Join(ret.Data(), ""), nil
}



