package main

import (
	"errors"
	"testing"
)

func TestExpression(t *testing.T) {
	expr := `user.level > 3 || (user.level < 2 && user.name == "admin")`
	e, err := newExpression(expr)
	if err != nil {
		t.Fatalf("failed to create expression: %v", err)
	}

	tests := []struct {
		user     map[string]any
		expected bool
	}{
		{map[string]any{"name": "alice", "level": 5}, true},
		{map[string]any{"name": "bob", "level": 1}, false},
		{map[string]any{"name": "admin", "level": 1}, true},
		{map[string]any{"name": "charlie", "level": 2}, false},
	}

	for _, test := range tests {
		result, err := e.Eval(test.user)
		if err != nil {
			t.Errorf("failed to evaluate expression for user %v: %v", test.user, err)
			continue
		}
		if result != test.expected {
			t.Errorf("unexpected result for user %v: got %v, want %v", test.user, result, test.expected)
		}
	}
}

func TestExpressionWrongTypes(t *testing.T) {
	test := []struct {
		user        map[string]any
		expression  string
		expectedErr error
	}{
		// wrong operation resulting in non-bool
		{
			user:        map[string]any{"name": "alice", "level": 3},
			expression:  `user.level + 3`,
			expectedErr: ErrExpressionResNotBool,
		},
		// wrong type (string) resulting in non-bool
		{
			user:        map[string]any{"name": "alice", "level": "three"},
			expression:  `user.level + 3`,
			expectedErr: ErrExpressionResNotBool,
		},
		// wrong type (string) for minus operation
		{
			user:        map[string]any{"name": "alice", "level": "three"},
			expression:  `user.level - 3`,
			expectedErr: ErrExpressionRun,
		},
	}

	for _, testCase := range test {
		e, err := newExpression(testCase.expression)
		if err != nil {
			t.Errorf("failed to create expression: %v", err)
			continue
		}
		_, err = e.Eval(testCase.user)
		if !errors.Is(err, testCase.expectedErr) {
			t.Errorf("unexpected error for user %v and expression %q: got %v, want %v", testCase.user, testCase.expression, err, testCase.expectedErr)
			t.FailNow()
		}
	}
}

func TestExpressionWrongSyntax(t *testing.T) {
	expression := `user.level > 3 ||| user.name == "admin"` // triple | is invalid
	_, err := newExpression(expression)
	if !errors.Is(err, ErrExpressionCompile) {
		t.Errorf("unexpected error for expression %q: got %v, want %v", expression, err, ErrExpressionCompile)
		t.FailNow()
	}
}
