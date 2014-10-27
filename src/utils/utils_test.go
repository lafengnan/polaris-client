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

func TestGetDirAndFileList(t *testing.T) {
    path1 := "/home/panzhongbin/test_files/210_files/"
    path2 := "/hahah"
    w1, err1 := GetDirAndFileList(path1)
    w2, err2 := GetDirAndFileList(path2)
    if len(w1.Files) != 210 || err1 != nil {
        t.Errorf("GetDirAndFileList(\"/home/panzhongbin/test_files/210_files\") failed", w1, err1)
    }

    if err2 == nil {
        t.Errorf("GetDirAndFileList(\"/hahah/\") failed", w2, err2)
    }

}

func BenchmarkGetDirAndFileList(b *testing.B) {
    path := "/home/panzhongbin/test_files/210_files/"
    for i := 0; i < b.N; i++ {
        GetDirAndFileList(path)
    }
}
