//+build windows

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
	//TODO: Complete this list
	switch exitCode {
	case 0:
		return "The operation completed successfully.";
	case 1:
		return "Incorrect function.";
	case 2:
		return "The system cannot find the file specified.";
	case 3:
		return "The system cannot find the path specified.";
	case 4:
		return "The system cannot open the file.";
	case 5:
		return "Access is denied.";
	case 6:
		return "The handle is invalid.";
	case 7:
		return "The storage control blocks were destroyed.";
	case 8:
		return "Not enough storage is available to process this command.";
	case 9:
		return "The storage control block address is invalid.";
	case 10:
		return "The environment is incorrect.";
	case 11:
		return "An attempt was made to load a program with an incorrect format.";
	case 12:
		return "The access code is invalid.";
	case 13:
		return "The data is invalid.";
	case 14:
		return "Not enough storage is available to complete this operation.";
	case 15:
		return "The system cannot find the drive specified.";
	case 16:
		return "The directory cannot be removed.";
	case 17:
		return "The system cannot move the file to a different disk drive.";
	case 18:
		return "There are no more files.";
	case 19:
		return "The media is write protected.";
	case 20:
		return "The system cannot find the device specified.";
	case 21:
		return "The device is not ready.";
	case 22:
		return "The device does not recognize the command.";
	case 23:
		return "Data error (cyclic redundancy check).";
	case 24:
		return "The program issued a command but the command length is incorrect.";
	case 25:
		return "The drive cannot locate a specific area or track on the disk.";
	default:
		return fmt.Sprintf("Process was exited with code %v", exitCode)
	}
}