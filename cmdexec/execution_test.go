//+build linux windows

package cmdexec

import (
	"testing"
	"bytes"
)

func TestCommandRendering(test *testing.T){
	templ, err := NewDefaultRenderer("echo", "echo \"{{.message}}\"")
	if err != nil {
		test.Fatal(err)
	}
	args := NewArguments().SetString("message", "Hello, world!")
	cmd, err := templ(args)
	if err != nil {
		test.Fatal(err)
	}
	if len(cmd.Args) != 2 {
		test.Fatal("Invalid number of parsed args")
	}
}

func readAll(result ExecutionResultRecorder) ([]byte, error) {
	buf := new(bytes.Buffer)
	_, err := result.WriteTo(buf)
	return buf.Bytes(), err
}

func TestTextExecution(test *testing.T) {
	renderer, err := NewDefaultRenderer("echo", "echo \"{{.message}}\"")
	if err != nil {
		test.Fatal(err)
	}
	executor := NewCommandExecutor(renderer)
	args := NewArguments().SetString("message", "Hello, world!")
	result := NewTextRecorder()
	defer result.Close()
	if err := executor(args, result); err != nil {
		test.Fatal(err)
	}

	if out, err := readAll(result); err != nil || string(out) != "Hello, world!\n" {
		test.Fatal("Unexpected stdout")
	}
}

func TestFileExecution(test *testing.T) {
	renderer, err := NewDefaultRenderer("echo", "echo {{.message}}")
	if err != nil {
		test.Fatal(err)
	}
	executor := NewCommandExecutor(renderer)
	args := NewArguments().SetString("message", "Hello, world!")
	result, _ := NewTempFileRecorder(true)
	defer result.Close()
	if err := executor(args, result); err != nil {
		test.Fatal(err)
	}
	if out, err := readAll(result); err != nil || string(out) != "Hello, world!\n" {
		test.Fatal("Unexpected stdout")
	}
}