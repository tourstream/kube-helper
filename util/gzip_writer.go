package util

import (
	"bufio"
	"compress/gzip"
	"os"
)

type GzipWriterInterface interface {
	Write(s string) (int, error)
}

type gzWriter struct {
	file         *os.File
	gzWriter     *gzip.Writer
	bufferWriter *bufio.Writer
}

func CreateGzWriter(name string) (gzWriter, error) {

	file, err := os.OpenFile(name, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		return nil, err
	}
	gzWriter := gzip.NewWriter(file)

	return gzWriter{file, gzWriter, bufio.NewWriter(gzWriter)}, nil
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
