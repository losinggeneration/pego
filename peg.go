// vim: ff=unix ts=3 sw=3 noet

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

// List of alternatives.
func (p *Pattern) Or(ps ...interface{}) *Pattern {
	var ret *Pattern
	var p2 *Pattern
	for i := len(ps) - 1; i >= -1; i-- {
		if i == -1 {
			p2 = p
		} else {
			p2 = Pat(ps[i])
		}
		if ret == nil {
			ret = p2
		} else {
			ret = Or(p2, ret)
		}
	}
	return ret
}

// Match this pattern, except for when `pred` matches.
func (p *Pattern) Exc(pred *Pattern) *Pattern {
	return Seq(Not(pred), p)
}

// Match this pattern between `min` and `max` times.
// max == -1 means unlimited.
func (p *Pattern) Rep(min, max int) *Pattern {
	return Rep(p, min, max)
}

// Resolve an open reference. Consider using a grammar instead.
func (p *Pattern) Resolve(name string, target *Pattern) *Pattern {
	return Grm("__start", map[string]*Pattern{
		"__start": p,
		name:      target,
	})
}

// A simple capture of this pattern.
func (p *Pattern) Csimple() *Pattern {
	return Csimple(p)
}

// A list capture of this pattern.
func (p *Pattern) Clist() *Pattern {
	return Clist(p)
}

// A function capture of this pattern.
func (p *Pattern) Cfunc(f func([]*CaptureResult) (interface{}, os.Error)) *Pattern {
	return Cfunc(p, f)
}

// A string capture of this pattern.
func (p *Pattern) Cstring(format string) *Pattern {
	return Cstring(p, format)
}

// A substition capture of this pattern.
func (p *Pattern) Csubst() *Pattern {
	return Csubst(p)
}

// A sequence of values, instructions and other patterns.
// (See Seq2)
func Seq(args ...interface{}) *Pattern {
	return Seq2(args)
}

// A sequence of values, instructions and other patterns
// Instructions that represent jumps are updated to match
// the sizes of the argument.
func Seq2(args []interface{}) *Pattern {
	size := 0
	// Figure out where each argument will be placed in
	// the final pattern.
	offsets := make(map[int]int, len(args))
	for i := range args {
		offsets[i] = size
		switch v := args[i].(type) {
		case *Pattern:
			size += len(*v) - 1
		case Instruction:
			size += 1
		default:
			// Convert a value to a pattern.
			v2 := Pat(v)
			args[i] = v2
			size += len(*v2) - 1
		}
	}
	offsets[len(args)] = size
	// Construct the final pattern.
	ret := make(Pattern, size+1)
	pos := 0
	for i := range args {
		switch v := args[i].(type) {
		case *Pattern:
			copy(ret[pos:], *v)
			pos += len(*v) - 1
		case *IJump:
			ret[pos] = &IJump{offsets[i+v.offset] - pos}
			pos++
		case *IChoice:
			ret[pos] = &IChoice{offsets[i+v.offset] - pos}
			pos++
		case *ICall:
			ret[pos] = &ICall{offsets[i+v.offset] - pos}
			pos++
		case *ICommit:
			ret[pos] = &ICommit{offsets[i+v.offset] - pos}
			pos++
		case Instruction:
			ret[pos] = v
			pos++
		}
	}
	ret[pos] = &IEnd{}
	return &ret
}

// Always succeeds (an empty pattern).
func Succ() *Pattern {
	return Seq()
}

// Always fails.
func Fail() *Pattern {
	return Seq(
		&IFail{},
	)
}

// Matches `n` of any character.
func Any(n int) *Pattern {
	return Seq(
		&IAny{n},
	)
}

// Matches `char`
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

// Ordered choice of p1 and p2
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

// Repeat pattern between `min` and `max` times.
// max == -1 means unlimited.
func Rep(p *Pattern, min, max int) *Pattern {
	var size int
	if max < 0 {
		size = min + 3
	} else {
		size = min + 2*(max-min) + 1
	}
	args := make([]interface{}, size)
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
		args[pos+0] = &IChoice{2 * (max - min)}
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

// Negative look-ahead for the pattern.
func Not(p *Pattern) *Pattern {
	return Seq(
		&IChoice{4},
		p,
		&ICommit{1},
		&IFail{},
	)
}

// Positive look-ahead for the pattern.
func And(p *Pattern) *Pattern {
	return Seq(
		&IChoice{5},
		&IChoice{2},
		p,
		&ICommit{1},
		&IFail{},
	)
}

// Open reference to a name. Use with grammars.
func Ref(name string) *Pattern {
	return Seq(
		&IOpenCall{name},
	)
}

// Match the text litteraly.
func Lit(text string) *Pattern {
	args := make([]interface{}, len(text))
	for i := 0; i < len(text); i++ {
		args[i] = &IChar{text[i]}
	}
	return Seq2(args)
}

// Match a grammar.
// start: name of the first pattern
// grammar: map of names to patterns
func Grm(start string, grammar map[string]*Pattern) *Pattern {
	// Figure out where each pattern begins, so that open
	// references can be resolved
	refs := map[string]int{"": 0}
	size := 2
	order := make([]string, len(grammar))
	i := 0
	for name, p := range grammar {
		if len(name) == 0 {
			panic("Invalid name")
		}
		order[i] = name
		i += 1
		refs[name] = size
		size += len(*p)
	}
	// Construct the final pattern
	ret := make(Pattern, size+1)
	ret[0] = &ICall{refs[start] - 0}
	ret[1] = &IJump{size - 1}
	for _, name := range order {
		copy(ret[refs[name]:], *grammar[name])
		ret[refs[name]+len(*grammar[name])-1] = &IReturn{}
	}
	ret[len(ret)-1] = &IEnd{}
	// Update references
	for i, op := range ret {
		if op2, ok := op.(*IOpenCall); ok {
			if offset, ok := refs[op2.name]; ok {
				ret[i] = &ICall{offset - i}
			}
		}
	}
	return &ret
}

// Match a set of characters.
func Set(chars string) *Pattern {
	mask := [...]uint32{0, 0, 0, 0, 0, 0, 0, 0}
	for i := 0; i < len(chars); i++ {
		c := chars[i]
		mask[c>>5] |= 1 << (c & 0x1F)
	}
	return Seq(&ICharset{mask})
}

// Match a negated set of characters. Opposite of Set()
func NegSet(chars string) *Pattern {
	const N = ^uint32(0)
	mask := [...]uint32{N, N, N, N, N, N, N, N}
	for i := 0; i < len(chars); i++ {
		c := chars[i]
		mask[c>>5] &^= 1 << (c & 0x1F)
	}
	return Seq(&ICharset{mask})
}

// Resolve a value to a pattern.
// Patterns are return unmodified.
// * true gives a pattern that always succeeds. Equivalent to Succ().
// * false gives a pattern that always fails. Equivalent to Fail().
// * n < 0 asserts that there are at least -n characters.
//   Equivalent to And(Any(n)).
// * n >= 0 matches n characters. Equivalent to Any(n).
// * A string matches itself. Equivalent to Lit(value)
func Pat(value interface{}) *Pattern {
	switch v := value.(type) {
	case *Pattern:
		return v
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
			return And(Any(-v))
		}
	case string:
		return Lit(v)
	}
	// TODO(mizardx): Proper error handling.
	return nil
}

// Does a simple capture of the pattern.
func Csimple(p *Pattern) *Pattern {
	return Seq(
		&IOpenCapture{0, &SimpleCapture{}},
		p,
		&ICloseCapture{},
	)
}

// Does a position capture.
func Cposition() *Pattern {
	return Seq(
		&IEmptyCapture{0, &PositionCapture{}},
	)
}

// Does a constant capture.
func Cconst(value interface{}) *Pattern {
	return Seq(
		&IEmptyCapture{0, &ConstCapture{value}},
	)
}

// Does a list capture.
func Clist(p *Pattern) *Pattern {
	return Seq(
		&IOpenCapture{0, &ListCapture{}},
		p,
		&ICloseCapture{},
	)
}

// Does a function capture.
func Cfunc(p *Pattern, f func([]*CaptureResult) (interface{}, os.Error)) *Pattern {
	return Seq(
		&IOpenCapture{0, &FunctionCapture{f}},
		p,
		&ICloseCapture{},
	)
}

// Does a string capture.
func Cstring(p *Pattern, format string) *Pattern {
	return Seq(
		&IOpenCapture{0, &StringCapture{format}},
		p,
		&ICloseCapture{},
	)
}

// Does a substitution capture.
func Csubst(p *Pattern) *Pattern {
	return Seq(
		&IOpenCapture{0, &SubstCapture{}},
		p,
		&ICloseCapture{},
	)
}
