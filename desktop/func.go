package main

// #include <stdlib.h>
// #include <locale.h>
import "C"

import (
	"os"
	"runtime"
	"unsafe"
)

const LC_NUMERIC = int(C.LC_NUMERIC)

func setLocale(lc int, locale string) {
	l := C.CString(locale)
	defer C.free(unsafe.Pointer(l))
	C.setlocale(C.int(lc), l)
}

func inSlice(a string, b []string) bool {
	for _, i := range b {
		if a == i {
			return true
		}
	}
	return false
}

func homeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}
