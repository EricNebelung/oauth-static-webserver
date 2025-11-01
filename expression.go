package main

import (
	"errors"
	"fmt"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"

	log "github.com/sirupsen/logrus"
)

var (
	ErrExpressionAddVariable = errors.New("error adding variable to expression script before compile")
	ErrExpressionCompile     = errors.New("error compiling expression")
	ErrExpressionSetVariable = errors.New("error setting variable in expression script")
	ErrExpressionRun         = errors.New("error running expression script")
	ErrExpressionResMissing  = errors.New("variable __res__ not found after expression evaluation")
	ErrExpressionResNotBool  = errors.New("variable __res__ is not a bool after expression evaluation")
)

// The Expression contains a compiled Tengo script for evaluating expressions.
type Expression struct {
	compiled *tengo.Compiled
}

// newExpression creates a new Expression by compiling the given expression string.
// It returns ErrExpressionCompile if the compilation fails.
func newExpression(expression string) (*Expression, error) {
	script := tengo.NewScript([]byte(fmt.Sprintf(`
		math := import("math")
		text := import("text")
		times := import("times")
		__res__ := (%s)
	`, expression)))
	// import standard libraries
	script.SetImports(stdlib.GetModuleMap("math", "text", "times"))
	// set predefined variables
	err := script.Add("user", map[string]any{})
	if err != nil {
		log.WithError(err).Error(ErrExpressionAddVariable.Error())
		return nil, ErrExpressionAddVariable
	}
	compiled, err := script.Compile()
	if err != nil {
		log.WithError(err).Error(ErrExpressionCompile.Error())
		return nil, ErrExpressionCompile
	}
	return &Expression{
		compiled: compiled,
	}, nil
}

// Eval evaluates the expression against the provided user value map.
// It returns the boolean result of the evaluation or an error if the evaluation fails.
func (e *Expression) Eval(value map[string]any) (bool, error) {
	cloned := e.compiled.Clone()
	err := cloned.Set("user", value)
	if err != nil {
		log.WithError(err).Error(ErrExpressionSetVariable.Error())
		return false, ErrExpressionSetVariable
	}
	err = cloned.Run()
	if err != nil {
		log.WithError(err).Error(ErrExpressionRun.Error())
		return false, ErrExpressionRun
	}
	v := cloned.Get("__res__")
	if v == nil {
		log.Error(ErrExpressionResMissing.Error())
		return false, ErrExpressionResMissing
	}
	result, ok := v.Value().(bool)
	if !ok {
		log.Error(ErrExpressionResNotBool.Error())
		return false, ErrExpressionResNotBool
	}
	return result, nil
}
