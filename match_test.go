package pego

import (
	"fmt"
	"testing"
)

func TestSimpleMatch(t *testing.T) {
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

	tests := []string{
		"x", "(x)", "a(b(c)d(e)f)g", ")",
	}

	for _, s := range tests {
		fmt.Printf("\n\n=== MATCHING %q ===\n", s)
		fmt.Println("Trace:")
		r, err, pos := match(pat, s)

		if r != nil {
			fmt.Printf("Return value: %v\n", r)
		}
		if err != nil {
			t.Errorf("Error: %#v\n", err)
		}

		fmt.Printf("End position: %d\n", pos)
		if pos != len(s) {
			t.Error("Failed to match whole input")
		}
	}
}
