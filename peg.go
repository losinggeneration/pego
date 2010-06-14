package main

import (
   "os"
   "strings"
   "fmt"
)

type Pattern []Instruction
func (p *Pattern) String() string {
   ret := make([]string, len(*p))
   for i, op := range *p {
      ret[i] = fmt.Sprintf("%6d  %s", i, op)
   }
   return strings.Join(ret, "\n")
}
func (p *Pattern) Or(ps ...interface{}) *Pattern {
   var ret *Pattern
   var p2 *Pattern
   for i := len(ps)-1; i >= -1; i-- {
      if i == -1 {
         p2 = p
      } else {
         p2 = Pat(ps[i])
      }
      if ret == nil {
         ret = p2
      } else {
         ret = Or(p2,ret)
      }
   }
   return ret
}
func (p *Pattern) Exc(pred *Pattern) *Pattern {
   return Seq(Not(pred), p)
}
func (p *Pattern) Rep(min, max int) *Pattern {
   return Rep(p,min,max)
}
func (p *Pattern) Resolve(name string, target *Pattern) *Pattern {
   return Grm("__start", map[string] *Pattern {
      "__start": p,
      name: target,
   })
}
func (p *Pattern) Csimple() *Pattern {
   return Csimple(p)
}
func (p *Pattern) Clist() *Pattern {
   return Clist(p)
}
func (p *Pattern) Cfunc(f func([]*CaptureResult) (interface{},os.Error)) *Pattern {
   return Cfunc(p,f)
}
func (p *Pattern) Cstring(format string) *Pattern {
   return Cstring(p,format)
}
func (p *Pattern) Csubst() *Pattern {
   return Csubst(p)
}

func Seq(args ...interface{}) *Pattern {
   return Seq2(args)
}

func Seq2(args []interface{}) *Pattern {
   size := 0
   offsets := make(map[int] int, len(args))
   for i := range args {
      offsets[i] = size
      switch v := args[i].(type) {
      case *Pattern:
         size += len(*v)-1
      case Instruction:
         size += 1
      default:
         v2 := Pat(v)
         args[i] = v2
         size += len(*v2)-1
      }
   }
   offsets[len(args)] = size
   ret := make(Pattern, size+1)
   pos := 0
   for i := range args {
      switch v := args[i].(type) {
      case *Pattern:
         copy(ret[pos:],*v)
         pos += len(*v)-1
      case *IJump:
         ret[pos] = &IJump{offsets[i+v.offset]-pos}
         pos++
      case *IChoice:
         ret[pos] = &IChoice{offsets[i+v.offset]-pos}
         pos++
      case *ICall:
         ret[pos] = &ICall{offsets[i+v.offset]-pos}
         pos++
      case *ICommit:
         ret[pos] = &ICommit{offsets[i+v.offset]-pos}
         pos++
      case Instruction:
         ret[pos] = v
         pos++
      }
   }
   ret[pos] = &IEnd{}
   return &ret
}

func Succ() *Pattern {
   return Seq()
}

func Fail() *Pattern {
   return Seq(
      &IFail{},
   )
}

func Any(n int) *Pattern {
   return Seq(
      &IAny{n},
   )
}

func Char(char byte) *Pattern {
   return Seq(
      &IChar{char},
   )
}

func isfail(p *Pattern) bool {
   _, ok := (*p)[0].(*IFail)
   return ok
}
func issucc(p *Pattern) bool {
   _, ok := (*p)[0].(*IEnd)
   return ok
}

func Or(p1, p2 *Pattern) *Pattern {
   if isfail(p1) {
      return p2
   } else if issucc(p1) || isfail(p2) {
      return p1
   }
   return Seq(
      &IChoice{3},
      p1,
      &ICommit{2},
      p2,
   )
}

func Rep(p *Pattern, min, max int) *Pattern {
   var size int
   if max < 0 {
      size = min+3
   } else {
      size = min+2*(max-min)+1
   }
   args := make([]interface{},size)
   for i := 0; i < min; i++ {
      args[i] = p
   }
   pos := min
   if max < 0 {
      args[pos+0] = &IChoice{3}
      args[pos+1] = p
      args[pos+2] = &ICommit{-2}
      pos += 3
   } else {
      args[pos+0] = &IChoice{2*(max-min)}
      pos++
      for i := min; i < max; i++ {
         args[pos+0] = p
         args[pos+1] = &IPartialCommit{1}
         pos += 2
      }
      args[pos+0] = &ICommit{1}
      pos++
   }
   return Seq2(args)
}

func Not(p *Pattern) *Pattern {
   return Seq(
      &IChoice{4},
      p,
      &ICommit{1},
      &IFail{},
   )
}

func And(p *Pattern) *Pattern {
   return Seq(
      &IChoice{5},
      &IChoice{2},
      p,
      &ICommit{1},
      &IFail{},
   )
}

func Ref(name string) *Pattern {
   return Seq(
      &IOpenCall{name},
   )
}

func Lit(text string) *Pattern {
   args := make([]interface{}, len(text))
   for i := 0; i < len(text); i++ {
      args[i] = &IChar{text[i]}
   }
   return Seq2(args)
}

func Grm(start string, grammar map[string] *Pattern) *Pattern {
   refs := map[string] int { "": 0 }
   size := 2
   order := make([]string, len(grammar))
   i := 0
   for name, p := range grammar {
      if len(name) == 0 { panic("Invalid name") }
      order[i] = name
      i += 1
      refs[name] = size
      //fmt.Printf("Mapping %q to %d\n", name, size)
      size += len(*p)
   }
   ret := make(Pattern, size+1)
   ret[0] = &ICall{refs[start] - 0}
   ret[1] = &IJump{size-1}
   for _, name := range order {
      copy(ret[refs[name]:], *grammar[name])
      ret[refs[name]+len(*grammar[name])-1] = &IReturn{}
   }
   ret[len(ret)-1] = &IEnd{}
   for i, op := range ret {
      if op2, ok := op.(*IOpenCall); ok {
         if offset, ok := refs[op2.name]; ok {
            ret[i] = &ICall{offset-i}
         }
      }
   }
   return &ret
}

func Set(chars string) *Pattern {
   mask := [...]uint32{0,0,0,0,0,0,0,0}
   for i := 0; i < len(chars); i++ {
      c := chars[i]
      mask[c>>5] |= 1 << (c & 0x1F)
   }
   return Seq(&ICharset{mask})
}

func NegSet(chars string) *Pattern {
   const N = ^uint32(0)
   mask := [...]uint32{N,N,N,N,N,N,N,N}
   for i := 0; i < len(chars); i++ {
      c := chars[i]
      mask[c>>5] &^= 1 << (c & 0x1F)
   }
   return Seq(&ICharset{mask})
}

func Pat(value interface{}) *Pattern {
   switch v := value.(type) {
   case *Pattern: return v
   case bool:
      if v {
         return Succ()
      } else {
         return Fail()
      }
   case int:
      if v >= 0 {
         return Any(v)
      } else {
         return Not(Any(v))
      }
   case string:
      return Lit(v)
   }
   return nil
}

func Csimple(p *Pattern) *Pattern {
   return Seq(
      &IOpenCapture{0,&SimpleCapture{}},
      p,
      &ICloseCapture{},
   )
}

func Cposition() *Pattern {
   return Seq(
      &IEmptyCapture{0,&PositionCapture{}},
   )
}

func Cconst(value interface{}) *Pattern {
   return Seq(
      &IEmptyCapture{0,&ConstCapture{value}},
   )
}

func Clist(p *Pattern) *Pattern {
   return Seq(
      &IOpenCapture{0,&ListCapture{}},
      p,
      &ICloseCapture{},
   )
}

func Cfunc(p *Pattern, f func([]*CaptureResult) (interface{},os.Error)) *Pattern {
   return Seq(
      &IOpenCapture{0,&FunctionCapture{f}},
      p,
      &ICloseCapture{},
   )
}

func Cstring(p *Pattern, format string) *Pattern {
   return Seq(
      &IOpenCapture{0,&StringCapture{format}},
      p,
      &ICloseCapture{},
   )
}

func Csubst(p *Pattern) *Pattern {
   return Seq(
      &IOpenCapture{0,&SubstCapture{}},
      p,
      &ICloseCapture{},
   )
}

