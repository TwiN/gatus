package chart

import (
	"bufio"
	"io"
	"os"
)

// ReadLines reads a file and calls the handler for each line.
func ReadLines(filePath string, handler func(string) error) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		err = handler(line)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReadChunks reads a file in `chunkSize` pieces, dispatched to the handler.
func ReadChunks(filePath string, chunkSize int, handler func([]byte) error) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	chunk := make([]byte, chunkSize)
	for {
		readBytes, err := f.Read(chunk)
		if err == io.EOF {
			break
		}
		readData := chunk[:readBytes]
		err = handler(readData)
		if err != nil {
			return err
		}
	}
	return nil
}
