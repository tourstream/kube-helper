package util

import (
	"bufio"
	"compress/gzip"
	"os"
	"github.com/spf13/afero"
)

type GzipWriterInterface interface {
	Write(s string) (int, error)
}

type gzWriter struct {
	file         afero.File
	gzWriter     *gzip.Writer
	bufferWriter *bufio.Writer
}

func CreateGzWriter(fileSystem afero.Fs,name string) (gzWriter, error) {

	file, err := fileSystem.OpenFile(name, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		return gzWriter{}, err
	}
	writer := gzip.NewWriter(file)
	bufioWriter := bufio.NewWriter(writer)

	return gzWriter{file, writer, bufioWriter}, nil
}

func (f *gzWriter) Write(s string) (int, error) {
	return f.bufferWriter.WriteString(s)
}

func (f *gzWriter) Close() {
	f.bufferWriter.Flush()
	// Close the gzip first.
	f.gzWriter.Close()
	f.file.Close()
}
