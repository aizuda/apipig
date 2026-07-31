package toolkit

import (
	"github.com/flosch/pongo2/v6"
	"github.com/stretchr/testify/assert"
	"testing"
)

// https://github.com/flosch/pongo2

func TestPongo2(t *testing.T) {
	// capfirst 首字母大写
	tpl, err := pongo2.FromString("Hello {{ name|capfirst }}!")
	if err != nil {
		panic(err)
	}
	out, err := tpl.Execute(pongo2.Context{"name": "florian"})
	if err != nil {
		panic(err)
	}
	assert.Equal(t, out, "Hello Florian!")

	out2, err := tpl.Execute(pongo2.Context{})
	if err != nil {
		panic(err)
	}
	assert.Equal(t, out2, "Hello !")
}
