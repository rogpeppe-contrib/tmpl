package main_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	main "github.com/benbjohnson/tmpl"
)

// Ensure paths can be parsed from command line flags.
func TestMain_ParseFlags_Paths(t *testing.T) {
	m := NewMain()
	if err := m.ParseFlags([]string{"a", "b", "c"}); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(m.Paths, []string{"a", "b", "c"}) {
		t.Fatalf("unexpected paths: %+v", m.Paths)
	}
}

// Ensure data can be parsed from command line flags as JSON.
func TestMain_ParseFlags_Data_JSON(t *testing.T) {
	m := NewMain()
	if err := m.ParseFlags([]string{"-data", `{"foo":"bar"}`}); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(m.Data, map[string]interface{}{"foo": "bar"}) {
		t.Fatalf("unexpected data: %#v", m.Data)
	}
}

// Ensure data can be parsed from command line flags as a filename.
func TestMain_ParseFlags_Data_File(t *testing.T) {
	m := NewMain()
	m.FileReadWriter.ReadFileFn = func(filename string) ([]byte, error) {
		if filename != "path/to/data" {
			t.Fatalf("unexpected filename: %s", filename)
		}
		return []byte(`{"foo":"bar"}`), nil
	}

	if err := m.ParseFlags([]string{"-data", `@path/to/data`}); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(m.Data, map[string]interface{}{"foo": "bar"}) {
		t.Fatalf("unexpected data: %#v", m.Data)
	}
}

// Ensure a basic template file can be processed.
func TestMain_Run(t *testing.T) {
	m := NewMain()
	m.OS.StatFn = func(filename string) (os.FileInfo, error) {
		if filename != "a.tmpl" {
			t.Fatalf("unexpected filename: %s", filename)
		}
		return &fileInfo{mode: 0666}, nil
	}
	m.FileReadWriter.ReadFileFn = func(filename string) ([]byte, error) {
		if filename != "a.tmpl" {
			t.Fatalf("unexpected filename: %s", filename)
		}
		return []byte(`hi {{.name}}, you are {{.age}}`), nil
	}
	m.FileReadWriter.WriteFileFn = func(filename string, data []byte, perm os.FileMode) error {
		if filename != "a" {
			t.Fatalf("unexpected filename: %s", filename)
		} else if string(data) != `hi bob, you are 12` {
			t.Fatalf("unexpected data: %s", data)
		} else if perm != 0666 {
			t.Fatalf("unexpected perm: %s", perm)
		}
		return nil
	}
	m.Paths = []string{"a.tmpl"}
	m.Data = map[string]interface{}{"name": "bob", "age": 12}
	if err := m.Run(); err != nil {
		t.Fatal(err)
	}
}

// Ensure a file can be processed against array data.
func TestMain_Run_Array(t *testing.T) {
	m := NewMain()
	m.FileReadWriter.ReadFileFn = func(filename string) ([]byte, error) {
		return []byte(`I like{{range .}} ({{.}}){{end}}`), nil
	}
	m.FileReadWriter.WriteFileFn = func(filename string, data []byte, perm os.FileMode) error {
		if string(data) != `I like (apple) (pear)` {
			t.Fatalf("unexpected data: %s", data)
		}
		return nil
	}

	m.Paths = []string{"a.tmpl"}
	m.Data = []interface{}{"apple", "pear"}
	if err := m.Run(); err != nil {
		t.Fatal(err)
	}
}

// Ensure a file will add a comment header if generating a Go file.
func TestMain_Run_Header_Go(t *testing.T) {
	m := NewMain()
	m.FileReadWriter.ReadFileFn = func(filename string) ([]byte, error) {
		return []byte("\n\n\n\n\n\npackage foo"), nil
	}
	m.FileReadWriter.WriteFileFn = func(filename string, data []byte, perm os.FileMode) error {
		if string(data) != `
// Code generated by tmpl; DO NOT EDIT.
// https://github.com/benbjohnson/tmpl
//
// Source: x.go.tmpl

package foo
`[1:] {
			t.Fatalf("unexpected data: %s", data)
		}
		return nil
	}

	m.Paths = []string{"x.go.tmpl"}
	m.Data = []interface{}{"apple", "pear"}
	if err := m.Run(); err != nil {
		t.Fatal(err)
	}
}

// Main is a test wrapper for main.Main.
type Main struct {
	*main.Main

	OS             MainOS
	FileReadWriter MainFileReadWriter

	Stdin  bytes.Buffer
	Stdout bytes.Buffer
	Stderr bytes.Buffer
}

// NewMain returns a new instance of Main.
// If the verbose command line flag is set then stdout/stderr also go to the terminal.
func NewMain() *Main {
	m := &Main{Main: main.NewMain()}
	m.Main.OS = &m.OS
	m.Main.FileReadWriter = &m.FileReadWriter
	m.Main.Stdin = &m.Stdin
	m.Main.Stdout = &m.Stdout
	m.Main.Stderr = &m.Stderr

	if testing.Verbose() {
		m.Main.Stdout = io.MultiWriter(os.Stdout, m.Main.Stdout)
		m.Main.Stderr = io.MultiWriter(os.Stderr, m.Main.Stderr)
	}

	// Default stat() to use 0666.
	m.OS.StatFn = DefaultOSStat

	return m
}

// MainOS is a mockable implementation of Main.OS.
type MainOS struct {
	StatFn func(filename string) (os.FileInfo, error)
}

func (os *MainOS) Stat(filename string) (os.FileInfo, error) {
	return os.StatFn(filename)
}

func DefaultOSStat(filename string) (os.FileInfo, error) { return &fileInfo{mode: 0666}, nil }

// MainFileReadWriter is a mockable implementation of Main.FileReadWriter.
type MainFileReadWriter struct {
	ReadFileFn  func(filename string) ([]byte, error)
	WriteFileFn func(filename string, data []byte, perm os.FileMode) error
}

func (r *MainFileReadWriter) ReadFile(filename string) ([]byte, error) {
	return r.ReadFileFn(filename)
}

func (r *MainFileReadWriter) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return r.WriteFileFn(filename, data, perm)
}

type fileInfo struct {
	mode os.FileMode
}

func (fi *fileInfo) Name() string       { return "" }
func (fi *fileInfo) Size() int64        { return 0 }
func (fi *fileInfo) Mode() os.FileMode  { return fi.mode }
func (fi *fileInfo) ModTime() time.Time { return time.Time{} }
func (fi *fileInfo) IsDir() bool        { return false }
func (fi *fileInfo) Sys() interface{}   { return nil }

// MustTempDir returns a temporary directory. Panic on error.
func MustTempDir() string {
	path, err := ioutil.TempDir("", "tmpl-")
	if err != nil {
		panic(err)
	}
	return path
}

// MustRemoveAll recursively removes a path. Panic on error.
func MustRemoveAll(path string) {
	if err := os.RemoveAll(path); err != nil {
		panic(err)
	}
}

// MustWriteFile writes data to filename. Panic on error.
func MustWriteFile(filename string, data []byte, perm os.FileMode) {
	if err := ioutil.WriteFile(filename, data, perm); err != nil {
		panic(err)
	}
}

// MustReadFile reads all data from filename. Panic on error.
func MustReadFile(filename string) []byte {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return data
}
