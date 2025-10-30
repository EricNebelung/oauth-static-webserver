package main

import (
	"testing"

	"github.com/go-playground/assert/v2"
)

func TestCheckHasOneGroup(t *testing.T) {
	a1 := []string{"group1"}
	a2 := []string{"group2", "group3"}

	assert.Equal(t, checkHasOneGroup(a1, a2), false)
	assert.Equal(t, checkHasOneGroup(a1, []string{}), false)
	assert.Equal(t, checkHasOneGroup(a1, []string{"group"}), false)
	assert.Equal(t, checkHasOneGroup(a1, []string{"group1"}), true)
	assert.Equal(t, checkHasOneGroup(a1, []string{"group1", "foo"}), true)
	assert.Equal(t, checkHasOneGroup(a2, []string{"group1", "group3"}), true)
	assert.Equal(t, checkHasOneGroup(a2, []string{"group2", "group3"}), true)
	assert.Equal(t, checkHasOneGroup(a2, []string{"group1"}), false)
}
