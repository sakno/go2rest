package cmdexec

import (
	"os"
	"io"
)

type fileRecorder struct {
	file os.File
	deleteOnClose bool
	written int
}

func (self *fileRecorder) Len() int {
	return self.written
}

func (self *fileRecorder) Close() error {
	fileName := self.file.Name()
	if err := self.file.Close(); err == nil {
		if self.deleteOnClose {
			return os.Remove(fileName)
		} else {
			return nil
		}
	} else {
		return err
	}
}

func (self *fileRecorder) Write(p []byte) (int, error) {
	written, err := self.file.Write(p)
	self.written += written
	return written, err
}

func (self *fileRecorder) WriteTo(output io.Writer) (int64, error) {
	self.file.Seek(0, io.SeekStart)
	return io.Copy(output, &self.file)
}

func NewTempFileRecorder(deleteOnClose bool) (ExecutionResultRecorder, error) {
	if file, err := NewTempFile(); err == nil {
		return &fileRecorder{file: *file, deleteOnClose: deleteOnClose}, nil
	} else {
		return nil, err
	}
}