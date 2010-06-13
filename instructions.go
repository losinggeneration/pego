package main

import (
   "fmt"
   "container/vector"
   "strings"
)

type Instruction interface {
   fmt.Stringer
}

type Char struct {
   char byte
}
func (op *Char) String() string {
   return fmt.Sprintf("Char %#02x", op.char)
}

type Jump struct { offset int }
func (op *Jump) String() string {
   return fmt.Sprintf("Jump %+d", op.offset)
}

type Choice struct { offset int }
func (op *Choice) String() string {
   return fmt.Sprintf("Choice %+d", op.offset)
}

type Call struct { offset int }
func (op *Call) String() string {
   return fmt.Sprintf("Call %+d", op.offset)
}

type Commit struct { offset int }
func (op *Commit) String() string {
   return fmt.Sprintf("Commit %+d", op.offset)
}

type Return struct { }
func (op *Return) String() string { return "Return" }
type Fail struct { }
func (op *Fail) String() string { return "Fail" }
type End struct { }
func (op *End) String() string { return "End" }

type OpenCapture struct {
   capOffset int
   handler CaptureHandler
}
func (op *OpenCapture) String() string {
   return fmt.Sprintf("Capture open %+d (%v)", -op.capOffset, op.handler)
}

type CloseCapture struct {
   capOffset int
}
func (op *CloseCapture) String() string {
   return fmt.Sprintf("Capture close %+d", -op.capOffset)
}

type FullCapture struct {
   capOffset int
}
func (op *FullCapture) String() string {
   return fmt.Sprintf("Capture full %+d", -op.capOffset)
}

type EmptyCapture struct {
   capOffset int
}
func (op *EmptyCapture) String() string {
   return "Capture empty"
}

type RuntimeCapture struct {
   capOffset int
}
func (op *RuntimeCapture) String() string {
   return fmt.Sprintf("Capture close/rt %+d", -op.capOffset)
}

type Charset struct {
   chars [8]uint32
}
func (op *Charset) String() string {
   def := uint32(op.chars[0] & 1)
   ranges := new(vector.StringVector)
   ranges.Push("Charset")
   if def == 0 {
      ranges.Push("[")
   } else {
      ranges.Push("[^")
   }
   start := -1
   fmtChar := func(char int) string {
      if 32 < char && char < 127 {
         return fmt.Sprintf("%c", char)
      }
      return fmt.Sprintf("\\x%02X", char)
   }
   for i := 0; i <= 256; i++ {
      switch {
      case i < 256 && (op.chars[i>>5] >> uint(i&0x1F)) & 1 != def:
         if start == -1 { start = i }
      case start == -1:
      case start == i-1:
         ranges.Push(fmtChar(start))
         start = -1
      case start == i-2:
         ranges.Push(fmtChar(start))
         ranges.Push(fmtChar(start+1))
         start = -1
      default:
         ranges.Push(fmt.Sprintf("%s-%s", fmtChar(start), fmtChar(i-1)))
         start = -1
      }
   }
   ranges.Push("]")
   return strings.Join(ranges.Data(), " ")
}

func (op *Charset) Has(char byte) bool {
   return op.chars[int(char>>5)] & (1 << uint(char & 0x1F)) != 0
}
func (op *Charset) add(lo, hi byte) {
   lobyt, hibyt := int(lo>>5), int(hi>>5)
   lobit, hibit := uint(lo & 0x1F), uint(hi & 0x1F)
   for i := lobyt; i <= hibyt; i++ {
      if lobyt <= i && i <= hibyt {
         mask := ^uint32(0)
         if i == lobyt { mask &^= (1 << lobit) - 1 }
         if i == hibyt { mask &^= (^uint32(1)) << hibit }
         op.chars[i] |= mask
      }
   }
}
func (op *Charset) negate() {
   for i := range op.chars {
      op.chars[i] = ^op.chars[i]
   }
}

type Any struct {
   count int
}
func (op *Any) String() string {
   return fmt.Sprintf("Any x %d", op.count)
}

