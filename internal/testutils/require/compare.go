package require

import (
	"cmp"
	"fmt"
	"reflect"
	"testing"
)

func GreaterOrEqual(t *testing.T, expected, actual any, msgAndArgs ...any) {
	t.Helper()

	if compare(t, expected, actual) < 0 {
		FailNow(t, fmt.Sprintf("expected: %v to be greater or equal to: %v", expected, actual), msgAndArgs...)
	}
}

func Greater(t *testing.T, expected, actual any, msgAndArgs ...any) {
	t.Helper()

	if compare(t, expected, actual) <= 0 {
		FailNow(t, fmt.Sprintf("expected: %v to be greater than: %v", expected, actual), msgAndArgs...)
	}
}

func LessOrEqual(t *testing.T, expected, actual any, msgAndArgs ...any) {
	t.Helper()

	if compare(t, expected, actual) > 0 {
		FailNow(t, fmt.Sprintf("expected: %v to be less or equal to: %v", expected, actual), msgAndArgs...)
	}
}

func Less(t *testing.T, expected, actual any, msgAndArgs ...any) {
	t.Helper()

	if compare(t, expected, actual) >= 0 {
		FailNow(t, fmt.Sprintf("expected: %v to be less than: %v", expected, actual), msgAndArgs...)
	}
}

func compare(t *testing.T, expected, actual any) int {
	t.Helper()

	e := reflect.ValueOf(expected)
	a := reflect.ValueOf(actual)

	if e.Kind() != a.Kind() {
		FailNow(t, "type mismatch: expected %T, got %T", expected, actual)
	}

	if !e.Comparable() {
		FailNow(t, "expected value is not comparable")
	}

	if !a.Comparable() {
		FailNow(t, "actual value is not comparable")
	}

	if e.Kind() != a.Kind() {
		FailNow(t, "type mismatch: expected %T, got %T", expected, actual)
	}

	switch e.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		{
			ev := e.Convert(reflect.TypeOf(int64(0))).Interface().(int64)
			av := a.Convert(reflect.TypeOf(int64(0))).Interface().(int64)
			return cmp.Compare(av, ev)
		}

	case reflect.Uint8, reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		{
			ev := e.Convert(reflect.TypeOf(uint64(0))).Interface().(uint64)
			av := a.Convert(reflect.TypeOf(uint64(0))).Interface().(uint64)
			return cmp.Compare(av, ev)
		}

	case reflect.Float32, reflect.Float64:
		{
			ev := e.Convert(reflect.TypeOf(float64(0))).Interface().(float64)
			av := a.Convert(reflect.TypeOf(float64(0))).Interface().(float64)
			return cmp.Compare(av, ev)
		}
	case reflect.String:
		{
			ev := e.Convert(reflect.TypeOf(string(""))).Interface().(string)
			av := a.Convert(reflect.TypeOf(string(""))).Interface().(string)
			return cmp.Compare(av, ev)
		}
	case reflect.Uintptr:
		{
			ev := e.Convert(reflect.TypeOf(uintptr(0))).Interface().(uintptr)
			av := a.Convert(reflect.TypeOf(uintptr(0))).Interface().(uintptr)
			return cmp.Compare(av, ev)
		}
	}

	FailNow(t, "type not supported: %T", expected)
	return 0 // should not be reached
}
