package main

import (
	"bufio"
	"fmt"
	"os"
)

// FilesAreEqual will return nil if the files exists and are equal.
// The error will either be a file open error, or a description of the
// difference with a line number
func FilesAreEqual(file1Path, file2Path string) error {
	f1, err := os.Open(file1Path)
	if err != nil {
		return err
	}
	defer func(f1 *os.File) {
		_ = f1.Close()
	}(f1)

	f2, err := os.Open(file2Path)
	if err != nil {
		return err
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
			return nil
		}
		if err1 != nil {
			return err1
		}
		if err2 != nil {
			return err2
		}
		if line1 != line2 {
			return fmt.Errorf("difference between %s and %s in line %d",
				file1Path, file2Path, lineNo)
		}
	}
}
