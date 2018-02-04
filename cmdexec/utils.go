package cmdexec

import (
	"os"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"strconv"
)

const tempFilePrefix = "pycliw-"

var rnd = rand.New(rand.NewSource(0xEDB88320))

func NewTempFileName(extension string) string {
	return filepath.Join(os.TempDir(), strconv.Itoa(rnd.Int())) + extension
}

func NewTempFile() (*os.File, error) {
	if file, err := ioutil.TempFile("", tempFilePrefix); err == nil {
		return file, nil
	} else {
		return nil, err
	}
}

func parseCommandLine(command string) []string {
	type ParserState uint8
	const stateStart = ParserState(0)
	const stateQuotes = ParserState(1)
	const stateArgs = ParserState(2)

	var args []string
	state := stateStart
	current := ""
	quote := "\""
	escapeNext := true
	for i := 0; i < len(command); i++ {
		c := command[i]

		if state == stateQuotes {
			if string(c) != quote {
				current += string(c)
			} else {
				args = append(args, current)
				current = ""
				state = stateStart
			}
			continue
		}

		if escapeNext {
			current += string(c)
			escapeNext = false
			continue
		}

		if c == '\\' {
			escapeNext = true
			continue
		}

		if c == '"' || c == '\'' {
			state = stateQuotes
			quote = string(c)
			continue
		}

		if state == stateArgs {
			if c == ' ' || c == '\t' {
				args = append(args, current)
				current = ""
				state = stateStart
			} else {
				current += string(c)
			}
			continue
		}

		if c != ' ' && c != '\t' {
			state = stateArgs
			current += string(c)
		}
	}

	if state == stateQuotes {
		return []string{}
	}

	if current != "" {
		args = append(args, current)
	}

	return args
}