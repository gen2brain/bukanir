package main

// #include <stdlib.h>
// #include <locale.h>
import "C"

import (
	"os"
	"path/filepath"
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

func cacheDir() string {
	dir := os.Getenv("XDG_CACHE_HOME")
	if dir == "" {
		dir = filepath.Join(homeDir(), ".cache", "bukanir")
	} else {
		dir = filepath.Join(dir, "bukanir")
	}
	return dir
}
