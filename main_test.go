package main_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

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

// Ensure data can be parsed from command line flags.
func TestMain_ParseFlags_Data(t *testing.T) {
	m := NewMain()
	if err := m.ParseFlags([]string{"-data", `{"foo":"bar"}`}); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(m.Data, map[string]interface{}{"foo": "bar"}) {
		t.Fatalf("unexpected data: %#v", m.Data)
	}
}

// Ensure a file can be processed against map data.
func TestMain_Run_Map(t *testing.T) {
	path := MustTempDir()
	defer MustRemoveAll(path)

	MustWriteFile(filepath.Join(path, "a.tmpl"), []byte(`hi {{.name}}, you are {{.age}}`), 0666)

	m := NewMain()
	m.Paths = []string{filepath.Join(path, "a.tmpl")}
	m.Data = map[string]interface{}{"name": "bob", "age": 12}
	if err := m.Run(); err != nil {
		t.Fatal(err)
	} else if data := MustReadFile(filepath.Join(path, "a")); string(data) != `hi bob, you are 12` {
		t.Fatalf("unexpected content: %s", data)
	}
}

// Ensure a file can be processed against array data.
func TestMain_Run_File(t *testing.T) {
	path := MustTempDir()
	defer MustRemoveAll(path)

	MustWriteFile(filepath.Join(path, "a.tmpl"), []byte(`I like{{range .}} ({{.}}){{end}}`), 0666)

	m := NewMain()
	m.Paths = []string{filepath.Join(path, "a.tmpl")}
	m.Data = []interface{}{"apple", "pear"}
	if err := m.Run(); err != nil {
		t.Fatal(err)
	} else if data := MustReadFile(filepath.Join(path, "a")); string(data) != `I like (apple) (pear)` {
		t.Fatalf("unexpected content: %s", data)
	}
}

// Main is a test wrapper for main.Main.
type Main struct {
	*main.Main

	Stdin  bytes.Buffer
	Stdout bytes.Buffer
	Stderr bytes.Buffer
}

// NewMain returns a new instance of Main.
// If the verbose command line flag is set then stdout/stderr also go to the terminal.
func NewMain() *Main {
	m := &Main{Main: main.NewMain()}
	m.Main.Stdin = &m.Stdin
	m.Main.Stdout = &m.Stdout
	m.Main.Stderr = &m.Stderr

	if testing.Verbose() {
		m.Main.Stdout = io.MultiWriter(os.Stdout, m.Main.Stdout)
		m.Main.Stderr = io.MultiWriter(os.Stderr, m.Main.Stderr)
	}

	return m
}

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
