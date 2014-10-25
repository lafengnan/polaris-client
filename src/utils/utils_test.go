package utils

import (
    "testing"
)

func TestTrace(t *testing.T) {
    r, _ := Trace("hello", "world")

    if r != "hello" {
        t.Errorf("Trace(\"hello\", \"world\") failed. Got %s, expected \"hello\"", r)
    }
}

func BenchmarkTrace(b *testing.B) {
    for i := 0; i < b.N; i++ {
        Trace("hello", "world")
    }
}

func TestGetFunctionName(t *testing.T) {

    name := GetFunctionName(TestTrace)

    if name != "utils.TestTrace" {
        t.Errorf("GetFunctionName(TestTrace) failed. Got %s, expected utils.TestTrace", name)
    }
}

func BenchmarkGetFunctionName(b *testing.B) {
    for i := 0; i < b.N; i++ {
        GetFunctionName(TestTrace)
    }
}
