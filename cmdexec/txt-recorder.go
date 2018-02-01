package cmdexec

import (
	"bytes"
	"io"
)

//text recorder interprets process output as text
type textRecorder struct {
	buffer bytes.Buffer
}

func (self *textRecorder) Len() int {
	return self.buffer.Len()
}

func (self *textRecorder) Close() error{
	self.buffer.Reset()
	return nil
}

func (self *textRecorder) Write(p []byte) (int, error) {
	return self.buffer.Write(p)
}

func (self *textRecorder) WriteTo(output io.Writer) (int64, error) {
	return self.buffer.WriteTo(output)
}

//Creates a new recorder of command result which interprets
//process output as text
func NewTextRecorder() ExecutionResultRecorder {
	return new(textRecorder)
}
