package cmdexec

import (
	"text/template"
	"os/exec"
	"bytes"
	"errors"
	"strings"
)

//Creates default command renderer based on Go template language
func NewDefaultRenderer(name string, text string) (CommandRenderer, error) {
	if tpl, err := template.New(name).Parse(text); err == nil {
		renderer := func(args Arguments) (*exec.Cmd, error) {
			buf := new(bytes.Buffer)
			if err := tpl.Execute(buf, args); err == nil {
				args := parseCommandLine(buf.String())
				if len(args) < 1 {
					return nil, errors.New("Invalid command-line template")
				}
				result := exec.Command(args[0], args[1:]...)
				return result, nil
			} else {
				return nil, err
			}
		}
		return renderer, nil
	} else {
		return nil, err
	}
}

func NewAutoNamedRenderer(text string) (CommandRenderer, error) {
	switch patternName := strings.Fields(text); len(patternName) {
	case 0:
		return NewDefaultRenderer("<noname>", text)
	default:
		return NewDefaultRenderer(patternName[0], text)
	}
}
