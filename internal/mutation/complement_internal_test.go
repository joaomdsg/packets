package mutation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComplement_pairsEachSupportedOperatorWithItsMutation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		expr     string
		original string
		mutated  string
	}{
		{"a > b", ">", ">="},
		{"a >= b", ">=", ">"},
		{"a < b", "<", "<="},
		{"a <= b", "<=", "<"},
		{"a == b", "==", "!="},
		{"a != b", "!=", "=="},
		{"a + b", "+", "-"},
		{"a - b", "-", "+"},
		{"a * b", "*", "/"},
		{"a / b", "/", "*"},
		{"a % b", "%", "*"},
		{"a << b", "<<", ">>"},
		{"a >> b", ">>", "<<"},
		{"a & b", "&", "|"},
		{"a | b", "|", "&"},
		{"a ^ b", "^", "&^"},
		{"a &^ b", "&^", "^"},
		{"a && b", "&&", "||"},
		{"a || b", "||", "&&"},
	}
	for _, c := range cases {
		t.Run(c.expr, func(t *testing.T) {
			t.Parallel()
			src := []byte("package p\n\nfunc f(a, b int) int {\n\t_ = " + c.expr + "\n\treturn 0\n}\n")
			mutants, err := GenerateMutants(src, []LineRange{{Start: 4, End: 4}})
			require.NoError(t, err)
			require.Len(t, mutants, 1)
			assert.Equal(t, c.original, mutants[0].Original)
			assert.Equal(t, c.mutated, mutants[0].Mutated)
		})
	}
}
