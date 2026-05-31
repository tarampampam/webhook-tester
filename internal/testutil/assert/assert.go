// Package assert provides test helper functions that follow testify-style naming conventions.
// All assertions call t.Fatal on failure, stopping the test immediately.
package assert

import (
	"encoding/json"
	"errors"
	"iter"
	"reflect"
	"strings"
	"testing"
)

// NoError fails the test if err is not nil.
func NoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Error fails the test if err is nil.
func Error(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ErrorIs fails the test if [errors.Is](err, target) is false.
func ErrorIs(t *testing.T, err, target error) {
	t.Helper()

	if !errors.Is(err, target) {
		t.Fatalf("expected errors.Is(%v, %v) to be true, but got false", err, target)
	}
}

// NotErrorIs fails the test if [errors.Is](err, target) is true.
func NotErrorIs(t *testing.T, err, target error) {
	t.Helper()

	if errors.Is(err, target) {
		t.Fatalf("expected errors.Is(%v, %v) to be false, but got true", err, target)
	}
}

// ErrorContains fails the test if err is nil or its message does not contain substr.
func ErrorContains(t *testing.T, err error, substr string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error containing %q, but got nil", substr)
	}

	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("expected error containing %q, but got: %v", substr, err)
	}
}

// ErrorEqual fails the test if err is nil or its message does not exactly equal msg.
func ErrorEqual(t *testing.T, err error, msg string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error %q, got nil", msg)
	}

	if err.Error() != msg {
		t.Fatalf("expected error %q, got %q", msg, err.Error())
	}
}

// Equal fails the test if expected and actual are not equal.
func Equal[T comparable](t *testing.T, expected, actual T) {
	t.Helper()

	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

// Seq2Count fails the test if the number of elements yielded by seq does not equal expectedCount.
func Seq2Count[K, V any](t *testing.T, expectedCount int, seq iter.Seq2[K, V]) {
	t.Helper()

	var n int

	for range seq {
		n++
	}

	if n != expectedCount {
		t.Fatalf("expected %d elements in sequence, got %d", expectedCount, n)
	}
}

// DeepEqual fails the test if expected and actual are not deeply equal (use for non-comparable types).
func DeepEqual(t *testing.T, expected, actual any) {
	t.Helper()

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

// Contains fails the test if s does not contain each of the given substrings.
func Contains(t *testing.T, s string, substrings ...string) {
	t.Helper()

	for _, substr := range substrings {
		if !strings.Contains(s, substr) {
			t.Fatalf("expected %q to contain %q", s, substr)
		}
	}
}

// Nil fails the test if v is not nil.
func Nil(t *testing.T, v any) {
	t.Helper()

	if v == nil {
		return
	}

	switch value := reflect.ValueOf(v); value.Kind() { //nolint:exhaustive // default covers all non-nilable kinds
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func, reflect.Interface:
		if value.IsNil() {
			return
		}

		t.Fatalf("expected nil value, got non-nil %s", value.Kind())
	default:
		t.Fatalf("expected nil value, got %v", v)
	}
}

// NotNil fails the test if v is nil.
func NotNil(t *testing.T, v any) {
	t.Helper()

	if v == nil {
		t.Fatal("expected non-nil value, got nil")
	}

	switch value := reflect.ValueOf(v); value.Kind() { //nolint:exhaustive // default covers all non-nilable kinds
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func, reflect.Interface:
		if value.IsNil() {
			t.Fatal("expected non-nil value, got nil")
		}
	default:
		// for non-pointer types, we consider any value to be non-nil, so we do nothing here
	}
}

// True fails the test if condition is false.
func True(t *testing.T, condition bool) {
	t.Helper()

	if !condition {
		t.Fatal("expected condition to be true, but it was false")
	}
}

// False fails the test if condition is true.
func False(t *testing.T, condition bool) {
	t.Helper()

	if condition {
		t.Fatal("expected condition to be false, but it was true")
	}
}

// Empty fails the test if value is not the zero value of its type.
func Empty[T comparable](t *testing.T, value T) {
	t.Helper()

	var zero T

	if value != zero {
		t.Fatalf("expected empty value, got %v", value)
	}
}

// Same fails the test if expected and actual are not the same pointer.
func Same[T any](t *testing.T, expected, actual *T) {
	t.Helper()

	if expected != actual {
		t.Fatalf("expected same pointer (%p), got different pointer (%p)", expected, actual)
	}
}

// NotSame fails the test if expected and actual are the same pointer.
func NotSame[T any](t *testing.T, expected, actual *T) {
	t.Helper()

	if expected == actual {
		t.Fatalf("expected different pointers, but both were %p", expected)
	}
}

// NotPanics fails the test if fn panics.
func NotPanics(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	fn()
}

// JSONEq fails the test if expected and actual are not equal when parsed as JSON.
func JSONEq[T ~string](t *testing.T, expected, actual T) {
	t.Helper()

	var expectedObj, actualObj any

	if err := json.Unmarshal([]byte(expected), &expectedObj); err != nil {
		t.Fatalf("invalid expected JSON: %v", err)
	}

	if actual == expected {
		return // optimization for common case where strings are identical
	}

	if err := json.Unmarshal([]byte(actual), &actualObj); err != nil {
		t.Fatalf("invalid actual JSON: %v", err)
	}

	if !reflect.DeepEqual(expectedObj, actualObj) {
		t.Fatalf("expected JSON %s, got %s", expected, actual)
	}
}
