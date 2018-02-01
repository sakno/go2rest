//+build linux darwin

package cmdexec

import (
	"os/exec"
	"syscall"
	"fmt"
)

func convertToError(err* exec.ExitError) *ExecutionError {
	result := &ExecutionError{stderr: err.Stderr}
	switch ws := err.Sys().(type) {
	case syscall.WaitStatus:
		result.ProcessExitCode = ws.ExitStatus()
	default:
		result.ProcessExitCode = -1
	}
	return result
}

func exitCodeToString(exitCode int) string{
	return fmt.Sprintf("Process was exited with code %v", exitCode)
}