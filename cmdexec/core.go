package cmdexec

import (
	"os/exec"
	"io"
	"log"
)

type ExecutionErrorCode uint8

//Arguments for command execution
type Arguments map[string]interface{}

//Execution error code
type ExecutionError struct {
	ProcessExitCode int
	stderr []byte
}

//Represents command template
type CommandRenderer func(Arguments) (*exec.Cmd, error)

//Execute command-line tool and write result into writer
//To simplify interpretation of execution result you can use ExecutionResultRecorder interface
//and its default implementations
type CommandExecutor func(Arguments, io.Writer) error

//Used to record result of command execution and interpret this result
type ExecutionResultRecorder interface {
	io.Writer
	io.WriterTo
	io.Closer
	Len() int
}

/* CommandExecutor */

//Creates default command executor
func NewCommandExecutor(render CommandRenderer) CommandExecutor{
	if render == nil{
		log.Panicf("Command renderer is not specified")
	}
	return func(args Arguments, output io.Writer) error {
		if cmd, err := render(args); err == nil {
			cmd.Stdout = output
			if err := cmd.Run(); err == nil{
				return nil
			} else {
				switch e := err.(type) {
				case *exec.ExitError:
					return convertToError(e)
				default:
					return e
				}
			}
		} else {
			return err
		}
	}
}

/* ExecutionError */
func (self* ExecutionError) Error() string {
	if self.stderr != nil && len(self.stderr) > 0 {
		return string(self.stderr)
	} else {
		return exitCodeToString(self.ProcessExitCode)
	}
}

/* Arguments */

//creates a new empty set of arguments
func NewArguments() Arguments {
	return make(Arguments)
}

//save new argument into list of arguments
func (args Arguments) SetString(name string, value string) Arguments {
	args[name] = value
	return args
}

func (args Arguments) SetInt32(name string, value int32) Arguments {
	args[name] = value
	return args
}

func (args Arguments) SetInt64(name string, value int64) Arguments {
	args[name] = value
	return args
}

func (args Arguments) SetUInt64(name string, value uint64) Arguments {
	args[name] = value
	return args
}

func (args Arguments) SetUInt32(name string, value uint32) Arguments {
	args[name] = value
	return args
}

func (args Arguments) SetBoolean(name string, value bool) Arguments {
	args[name] = value
	return args
}

func (args Arguments) SetFloat32(name string, value float32) Arguments {
	args[name] = value
	return args
}

func (args Arguments) SetFloat64(name string, value float64) Arguments {
	args[name] = value
	return args
}