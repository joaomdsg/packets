// Package mutation implements a diff-scoped mutation-testing oracle.
//
// It mutates supported binary operators within a set of changed lines,
// runs the project's test suite against each mutant, and reports the
// mutants that SURVIVE (tests still pass) as findings — the lines where
// a green test suite fails to actually constrain behaviour.
package mutation

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
)

// LineRange is an inclusive range of 1-based line numbers.
type LineRange struct {
	Start, End int
}

// Mutant is a single operator mutation applied to a source file.
type Mutant struct {
	Line     int    // 1-based line of the mutated operator
	Original string // original operator text, e.g. ">="
	Mutated  string // replacement operator text, e.g. ">"
	Source   []byte // full file contents with the mutation applied
}

// complement maps each supported binary operator to the operator it
// mutates into. Membership in this map is what makes an operator
// "supported"; anything absent is left untouched.
var complement = map[token.Token]token.Token{
	token.GTR:  token.GEQ, // >  -> >=
	token.GEQ:  token.GTR, // >= -> >
	token.LSS:  token.LEQ, // <  -> <=
	token.LEQ:  token.LSS, // <= -> <
	token.EQL:  token.NEQ, // == -> !=
	token.NEQ:  token.EQL, // != -> ==
	token.ADD:  token.SUB, // +  -> -
	token.SUB:  token.ADD, // -  -> +
	token.LAND: token.LOR, // && -> ||
	token.LOR:  token.LAND, // || -> &&
}

// GenerateMutants parses src and returns one Mutant per supported binary
// operator site that falls within any of the given changed line ranges.
// If lines is empty, every site in the file is considered. Mutants are
// ordered by line.
func GenerateMutants(src []byte, lines []LineRange) ([]Mutant, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "src.go", src, 0)
	if err != nil {
		return nil, fmt.Errorf("parse source: %w", err)
	}

	var mutants []Mutant
	ast.Inspect(file, func(n ast.Node) bool {
		be, ok := n.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		to, ok := complement[be.Op]
		if !ok {
			return true
		}
		pos := fset.Position(be.OpPos)
		if !withinLines(pos.Line, lines) {
			return true
		}

		original := be.Op.String()
		mutated := to.String()
		offset := pos.Offset
		newSrc := make([]byte, 0, len(src)+len(mutated)-len(original))
		newSrc = append(newSrc, src[:offset]...)
		newSrc = append(newSrc, mutated...)
		newSrc = append(newSrc, src[offset+len(original):]...)

		mutants = append(mutants, Mutant{
			Line:     pos.Line,
			Original: original,
			Mutated:  mutated,
			Source:   newSrc,
		})
		return true
	})

	sort.SliceStable(mutants, func(i, j int) bool {
		return mutants[i].Line < mutants[j].Line
	})
	return mutants, nil
}

// withinLines reports whether line falls within any range, treating an
// empty ranges slice as "every line".
func withinLines(line int, ranges []LineRange) bool {
	if len(ranges) == 0 {
		return true
	}
	for _, r := range ranges {
		if line >= r.Start && line <= r.End {
			return true
		}
	}
	return false
}
