pego
====

This is a pattern matching library for Go. It is based on lpeg, which uses a flavor of PEG.

##Example
```go
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
```

##More information
* [LPeg - Parsing Expression Grammars For Lua](http://www.inf.puc-rio.br/~roberto/lpeg/lpeg.html) - Source of inspiration
* [A Text Pattern-Matching Tool based on Parsing Expression Grammars](http://www.inf.puc-rio.br/~roberto/docs/peg.pdf) - Paper on the implementation of LPeg.
