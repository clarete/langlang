package langlang

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVM(t *testing.T) {
	t.Run("I guess I will just die", func(t *testing.T) {
		bytecode := Encode(&Program{code: []Instruction{
			IHalt{},
		}})
		assert.Equal(t, uint8(0), bytecode.code[0])

		vm := NewVirtualMachine(bytecode)

		_, cur, err := vm.Match(strings.NewReader(""))

		require.NoError(t, err)
		assert.Equal(t, 0, cur)
	})

	t.Run("did the cursor move", func(t *testing.T) {
		cur, err := match("G <- .", "foo")
		require.NoError(t, err)
		assert.Equal(t, 1, cur)
	})

	t.Run("any", func(t *testing.T) {
		val, cur, err := exec("G <- .", "foo")
		require.NoError(t, err)
		assert.Equal(t, 1, cur)

		span := NewSpan(NewLocation(0, 0, 0), NewLocation(0, 1, 1))
		assert.Equal(t, val, NewNode("G", NewString("f", span), span))
	})

	t.Run("any star", func(t *testing.T) {
		val, cur, err := exec("G <- .*", "foo")
		require.NoError(t, err)
		assert.Equal(t, 3, cur)

		span := NewSpan(NewLocation(0, 0, 0), NewLocation(0, 1, 3))
		assert.Equal(t, val, NewNode("G", NewString("foo", span), span))
	})

	t.Run("char", func(t *testing.T) {
		val, cur, err := exec("G <- 'f'", "foo")
		require.NoError(t, err)
		assert.Equal(t, 1, cur)

		span := NewSpan(NewLocation(0, 0, 0), NewLocation(0, 1, 1))
		assert.Equal(t, val, NewNode("G", NewString("f", span), span))
	})

	t.Run("or", func(t *testing.T) {
		val, cur, err := exec("G <- 'f' / 'g' / 'h'", "g")
		require.NoError(t, err)
		assert.Equal(t, 1, cur)

		span := NewSpan(NewLocation(0, 0, 0), NewLocation(0, 1, 1))
		assert.Equal(t, val, NewNode("G", NewString("g", span), span))
	})

	t.Run("or on words", func(t *testing.T) {
		val, cur, err := exec("G <- 'avocado' / 'avante' / 'aviador'", "avante")
		require.NoError(t, err)
		assert.Equal(t, 6, cur)

		span := NewSpan(NewLocation(0, 0, 0), NewLocation(0, 1, 6))
		assert.Equal(t, val, NewNode("G", NewString("avante", span), span))
	})

	t.Run("class with range", func(t *testing.T) {
		val, cur, err := exec("G <- [0-9]+", "42")
		require.NoError(t, err)
		assert.Equal(t, 2, cur)

		span := NewSpan(NewLocation(0, 0, 0), NewLocation(0, 1, 2))
		assert.Equal(t, val, NewNode("G", NewString("42", span), span))
	})

	t.Run("class with range and literal", func(t *testing.T) {
		val, cur, err := exec("G <- [a-z_]+", "my_id")
		require.NoError(t, err)
		assert.Equal(t, 5, cur)

		span := NewSpan(NewLocation(0, 0, 0), NewLocation(0, 1, 5))
		assert.Equal(t, val, NewNode("G", NewString("my_id", span), span))
	})

	t.Run("optional matches", func(t *testing.T) {
		val, cur, err := exec("G <- 'f'?", "foo")
		require.NoError(t, err)
		assert.Equal(t, 1, cur)

		span := NewSpan(NewLocation(0, 0, 0), NewLocation(0, 1, 1))
		assert.Equal(t, val, NewNode("G", NewString("f", span), span))
	})

	t.Run("optional does not match", func(t *testing.T) {
		val, cur, err := exec("G <- 'f'?", "bar")
		require.NoError(t, err)
		assert.Equal(t, 0, cur)
		assert.Equal(t, val, nil)
	})

	t.Run("optional does not match followed by something else", func(t *testing.T) {
		val, cur, err := exec("G <- 'f'? 'bar'", "bar")
		require.NoError(t, err)
		assert.Equal(t, 3, cur)

		span := NewSpan(NewLocation(0, 0, 0), NewLocation(0, 1, 3))
		assert.Equal(t, val, NewNode("G", NewString("bar", span), span))
	})

	t.Run("not predicate", func(t *testing.T) {
		val, cur, err := exec("G <- (!';' .)*", "foo; bar")
		require.NoError(t, err)
		assert.Equal(t, 3, cur)

		span := NewSpan(NewLocation(0, 0, 0), NewLocation(0, 1, 3))
		assert.Equal(t, val, NewNode("G", NewString("foo", span), span))
	})

	t.Run("not any and star", func(t *testing.T) {
		val, cur, err := exec(`G <- "'" (!"'" .)* "'"`, "'foo'")
		require.NoError(t, err)
		assert.Equal(t, 5, cur)

		span := NewSpan(NewLocation(0, 0, 0), NewLocation(0, 1, 5))
		assert.Equal(t, val, NewNode("G", NewString("'foo'", span), span))
	})

	t.Run("and predicate", func(t *testing.T) {
		val, cur, err := exec("G <- &'a' .", "avocado")
		require.NoError(t, err)
		assert.Equal(t, 1, cur)

		span := NewSpan(NewLocation(0, 0, 0), NewLocation(0, 1, 1))
		assert.Equal(t, val, NewNode("G", NewString("a", span), span))
	})

	t.Run("parse hex number", func(t *testing.T) {
		val, cur, err := exec("G <- '0x' [0-9a-fA-F]+ / '0'", "0xff")
		require.NoError(t, err)
		assert.Equal(t, 4, cur)

		span := NewSpan(NewLocation(0, 0, 0), NewLocation(0, 1, 4))
		assert.Equal(t, val, NewNode("G", NewString("0xff", span), span))
	})
}

func match(expr, input string) (int, error) {
	vm := NewVirtualMachine(compile(expr, false))
	_, cur, err := vm.Match(strings.NewReader(input))
	return cur, err
}

func exec(expr, input string) (Value, int, error) {
	vm := NewVirtualMachine(compile(expr, true))
	return vm.Match(strings.NewReader(input))
}

func compile(expr string, addCap bool) *Bytecode {
	ast, err := NewGrammarParser(expr).Parse()
	if err != nil {
		panic(err)
	}
	if addCap {
		ast, err = AddCaptures(ast)
		if err != nil {
			panic(err)
		}
	}
	fmt.Printf("ast\n%s\n", ast.PrettyPrint())

	asm, err := Compile(ast, CompilerConfig{Optimize: 1})
	if err != nil {
		panic(err)
	}

	fmt.Printf("asm\n%s\n", asm.PrettyPrint())
	return Encode(asm)
}
