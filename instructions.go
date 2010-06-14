// vim: ff=unix ts=3 sw=3 noet

package main

import (
	"fmt"
	"container/vector"
	"strings"
)

// Interface for instructions.
// XXX(mizardx): Make this more specific?
type Instruction interface {
	fmt.Stringer
}

// Match a single character.
type IChar struct {
	char byte
}

func (op *IChar) String() string {
	return fmt.Sprintf("Char %#02x", op.char)
}

// Relative jump.
type IJump struct {
	offset int
}

func (op *IJump) String() string {
	return fmt.Sprintf("Jump %+d", op.offset)
}

// Add a fallback point at offset, and continue on next instruction.
type IChoice struct {
	offset int
}

func (op *IChoice) String() string {
	return fmt.Sprintf("Choice %+d", op.offset)
}

// Unresolved call. Used by grammars.
type IOpenCall struct {
	name string
}

func (op *IOpenCall) String() string {
	return fmt.Sprintf("OpenCall %q", op.name)
}

// Push return address to stack, and do a relative jump.
type ICall struct {
	offset int
}

func (op *ICall) String() string {
	return fmt.Sprintf("Call %+d", op.offset)
}

// Pop a fallback point, and do a relative jump.
type ICommit struct {
	offset int
}

func (op *ICommit) String() string {
	return fmt.Sprintf("Commit %+d", op.offset)
}

// Update top stack entry, and do a relative jump.
type IPartialCommit struct {
	offset int
}

func (op *IPartialCommit) String() string {
	return fmt.Sprintf("PartialCommit %+d", op.offset)
}

// Pop a return address from the stack and jump to it.
type IReturn struct{}

func (op *IReturn) String() string { return "Return" }

// Roll back to closest fallback point.
type IFail struct{}

func (op *IFail) String() string { return "Fail" }

// End of program (Last instuction only)
type IEnd struct{}

func (op *IEnd) String() string { return "End" }

// Open a new capture
type IOpenCapture struct {
	capOffset int
	handler   CaptureHandler
}

func (op *IOpenCapture) String() string {
	return fmt.Sprintf("Capture open %+d (%v)", -op.capOffset, op.handler)
}

// Close the nearest open capture
type ICloseCapture struct {
	capOffset int
}

func (op *ICloseCapture) String() string {
	return fmt.Sprintf("Capture close %+d", -op.capOffset)
}

// Open and close a new capture of fixed size
type IFullCapture struct {
	capOffset int
	handler   CaptureHandler
}

func (op *IFullCapture) String() string {
	return fmt.Sprintf("Capture full %+d (%s)", -op.capOffset, op.handler)
}

// Open and close a new empty capture
type IEmptyCapture struct {
	capOffset int
	handler   CaptureHandler
}

func (op *IEmptyCapture) String() string {
	return fmt.Sprintf("Capture empty (%s)", op.handler)
}

// Match a character from a set
// TODO(mizardx): Unicode?
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
		case i < 256 && (op.chars[i>>5]>>uint(i&0x1F))&1 != def:
			if start == -1 {
				start = i
			}
		case start == -1:
		case start == i-1:
			ranges.Push(fmtChar(start))
			start = -1
		case start == i-2:
			ranges.Push(fmtChar(start))
			ranges.Push(fmtChar(start + 1))
			start = -1
		default:
			ranges.Push(fmt.Sprintf("%s-%s", fmtChar(start), fmtChar(i-1)))
			start = -1
		}
	}
	ranges.Push("]")
	return strings.Join(ranges.Data(), " ")
}

// Charset contains character?
func (op *ICharset) Has(char byte) bool {
	return op.chars[int(char>>5)]&(1<<uint(char&0x1F)) != 0
}
// Add a range of characters
func (op *ICharset) add(lo, hi byte) {
	lobyt, hibyt := int(lo>>5), int(hi>>5)
	lobit, hibit := uint(lo&0x1F), uint(hi&0x1F)
	for i := lobyt; i <= hibyt; i++ {
		if lobyt <= i && i <= hibyt {
			mask := ^uint32(0)
			if i == lobyt {
				mask &^= (1 << lobit) - 1
			}
			if i == hibyt {
				mask &^= (^uint32(1)) << hibit
			}
			op.chars[i] |= mask
		}
	}
}
// Negate the whole set
func (op *ICharset) negate() {
	for i := range op.chars {
		op.chars[i] = ^op.chars[i]
	}
}

// Match `count` of any character
type IAny struct {
	count int
}

func (op *IAny) String() string {
	return fmt.Sprintf("Any x %d", op.count)
}
