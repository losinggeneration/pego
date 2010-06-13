package main

import (
   "fmt"
)

type Pattern []Instruction
func (p *Pattern) Or(ps ...interface{}) *Pattern {
   var ret *Pattern
   var p2 *Pattern
   for i := len(ps)-1; i >= -1; i-- {
      if i == -1 {
         p2 = p
      } else {
         p2 = P(ps[i])
      }
      if ret == nil {
         ret = p2
      } else {
         ret = Or(p2,ret)
      }
   }
   return ret
}
func (p *Pattern) Except(pred *Pattern) *Pattern {
   return Sequence(Not(pred), p)
}

func Sequence(args ...interface{}) *Pattern {
   return Sequence2(args)
}

func Sequence2(args []interface{}) *Pattern {
   size := 0
   offsets := make(map[int] int, len(args))
   for i := range args {
      offsets[i] = size
      switch v := args[i].(type) {
      case *Pattern:
         size += len(*v)-1
      case Instruction:
         size += 1
      }
   }
   offsets[len(args)] = size
   ret := make(Pattern, size+1)
   pos := 0
   for i := range args {
      switch v := args[i].(type) {
      default:
         panic(fmt.Sprintf("Not a *Pattern or Instruction: %#v",v))
      case *Pattern:
         copy(ret[pos:],*v)
         pos += len(*v)-1
      case *IJump:
         ret[pos] = &IJump{offsets[i+v.offset]}
         pos++
      case *IChoice:
         ret[pos] = &IChoice{offsets[i+v.offset]}
         pos++
      case *ICall:
         ret[pos] = &ICall{offsets[i+v.offset]}
         pos++
      case *ICommit:
         ret[pos] = &ICommit{offsets[i+v.offset]}
         pos++
      case Instruction:
         ret[pos] = v
         pos++
      }
   }
   ret[pos] = &IEnd{}
   return &ret
}

func Succeed() *Pattern {
   return Sequence()
}

func Fail() *Pattern {
   return Sequence(
      &IFail{},
   )
}

func Any(n int) *Pattern {
   return Sequence(
      &IAny{n},
   )
}

func Char(char byte) *Pattern {
   return Sequence(
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
   return Sequence(
      &IChoice{3},
      p1,
      &ICommit{2},
      p2,
   )
}

func Repeat(p *Pattern, min, max int) *Pattern {
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
   return Sequence2(args)
}

func Not(p *Pattern) *Pattern {
   return Sequence(
      &IChoice{4},
      p,
      &ICommit{1},
      &IFail{},
   )
}

func And(p *Pattern) *Pattern {
   return Sequence(
      &IChoice{5},
      &IChoice{2},
      p,
      &ICommit{1},
      &IFail{},
   )
}



func P(value interface{}) *Pattern {
   switch v := value.(type) {
   case *Pattern: return v
   case bool:
      if v {
         return Succeed()
      } else {
         return Fail()
      }
   case int:
      if v >= 0 {
         return Any(v)
      } else {
         return Not(Any(v))
      }
   }
   return nil
}

