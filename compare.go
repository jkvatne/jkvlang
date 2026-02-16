package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
)

func FilesAreEqual(file1Path, file2Path string) (bool, error) {
	f1, err := os.Open(file1Path)
	if err != nil {
		return false, err
	}
	defer func(f1 *os.File) {
		_ = f1.Close()
	}(f1)

	f2, err := os.Open(file2Path)
	if err != nil {
		return false, err
	}
	defer func(f2 *os.File) {
		_ = f2.Close()
	}(f2)

	reader1 := bufio.NewReader(f1)
	reader2 := bufio.NewReader(f2)
	var line1, line2 string
	lineNo := 0
	var err1, err2 error
	for {
		lineNo++
		line1, err1 = reader1.ReadString('\n')
		line2, err2 = reader2.ReadString('\n')
		if err1 != nil && err2 != nil && err1.Error() == "EOF" && err2.Error() == "EOF" {
			return true, nil
		}
		if err1 != nil {
			return false, err1
		}
		if err2 != nil {
			return false, err2
		}
		if line1 != line2 {
			return false, fmt.Errorf("Difference between %s and %s in line %d",
				file1Path, file2Path, lineNo)
		}
	}
}

func xxFilesAreEqual(file1Path, file2Path string) (bool, error) {
	f1, err := os.Open(file1Path)
	if err != nil {
		return false, err
	}
	defer func(f1 *os.File) {
		_ = f1.Close()
	}(f1)

	f2, err := os.Open(file2Path)
	if err != nil {
		return false, err
	}
	defer func(f2 *os.File) {
		_ = f2.Close()
	}(f2)

	// Check file sizes first for a quick fail
	fi1, err := f1.Stat()
	if err != nil {
		return false, err
	}
	fi2, err := f2.Stat()
	if err != nil {
		return false, err
	}
	if fi1.Size() != fi2.Size() {
		return false, nil
	}
	const chunkSize = 64000
	b1 := make([]byte, chunkSize)
	b2 := make([]byte, chunkSize)
	for {
		n1, err1 := f1.Read(b1)
		n2, err2 := f2.Read(b2)
		if err1 != nil || err2 != nil {
			if err1 == io.EOF && err2 == io.EOF {
				return true, nil // Reached end of both files, they match
			} else if err1 == io.EOF || err2 == io.EOF {
				return false, nil // Files have different lengths
			} else {
				return false, fmt.Errorf("error reading files: %v, %v", err1, err2)
			}
		}
		b1 = b1[0:n1]
		b2 = b2[0:n2]
		if !bytes.Equal(b1, b2) {
			return false, nil // Content differs
		}
	}
}
