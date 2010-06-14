package main

import (
   "fmt"
   "container/vector"
   "strings"
)

type Instruction interface {
   fmt.Stringer
}

type IChar struct {
   char byte
}
func (op *IChar) String() string {
   return fmt.Sprintf("Char %#02x", op.char)
}

type IJump struct { offset int }
func (op *IJump) String() string {
   return fmt.Sprintf("Jump %+d", op.offset)
}

type IChoice struct { offset int }
func (op *IChoice) String() string {
   return fmt.Sprintf("Choice %+d", op.offset)
}

type IOpenCall struct { name string }
func (op *IOpenCall) String() string {
   return fmt.Sprintf("OpenCall %q", op.name)
}

type ICall struct { offset int }
func (op *ICall) String() string {
   return fmt.Sprintf("Call %+d", op.offset)
}

type ICommit struct { offset int }
func (op *ICommit) String() string {
   return fmt.Sprintf("Commit %+d", op.offset)
}

type IPartialCommit struct { offset int }
func (op *IPartialCommit) String() string {
   return fmt.Sprintf("PartialCommit %+d", op.offset)
}

type IReturn struct { }
func (op *IReturn) String() string { return "Return" }

type IFail struct { }
func (op *IFail) String() string { return "Fail" }

type IEnd struct { }
func (op *IEnd) String() string { return "End" }

type IOpenCapture struct {
   capOffset int
   handler CaptureHandler
}
func (op *IOpenCapture) String() string {
   return fmt.Sprintf("Capture open %+d (%v)", -op.capOffset, op.handler)
}

type ICloseCapture struct {
   capOffset int
}
func (op *ICloseCapture) String() string {
   return fmt.Sprintf("Capture close %+d", -op.capOffset)
}

type IFullCapture struct {
   capOffset int
   handler CaptureHandler
}
func (op *IFullCapture) String() string {
   return fmt.Sprintf("Capture full %+d (%s)", -op.capOffset, op.handler)
}

type IEmptyCapture struct {
   capOffset int
   handler CaptureHandler
}
func (op *IEmptyCapture) String() string {
   return fmt.Sprintf("Capture empty (%s)", op.handler)
}

type ICharset struct {
   chars [8]uint32
}
func (op *ICharset) String() string {
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

func (op *ICharset) Has(char byte) bool {
   return op.chars[int(char>>5)] & (1 << uint(char & 0x1F)) != 0
}
func (op *ICharset) add(lo, hi byte) {
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
func (op *ICharset) negate() {
   for i := range op.chars {
      op.chars[i] = ^op.chars[i]
   }
}

type IAny struct {
   count int
}
func (op *IAny) String() string {
   return fmt.Sprintf("Any x %d", op.count)
}

