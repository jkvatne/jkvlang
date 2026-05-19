package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jkvatne/jkv/code"
)

func TestTokens(t *testing.T) {
	if len(TokenNames) != int(TOK_SIZE)+1 {
		t.Errorf("length of TokenNames is wrong")
	}
}

func TestCompile(t *testing.T) {
	slog.SetLogLoggerLevel(8)
	inputPath := "./test"
	outputPath := "./test/objectfiles"
	entries, err := os.ReadDir(inputPath)
	if err != nil {
		t.Errorf("Error opening test dir: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		code.UnitName = strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		name := filepath.Join(inputPath, entry.Name())
		fmt.Printf("Compiling %s\n", entry.Name())
		err = CompileFile(name, outputPath)
		if strings.HasPrefix(code.UnitName, "err_") {
			if err == nil {
				t.Errorf("Expected error in %s", code.UnitName)
			}
			continue
		} else if err != nil {
			t.Errorf("Error in file \"%s.jkv\" : %v", unitName, err)
		} else {
			targetFile := "./test/targets/" + code.UnitName + ".asm"
			objectFile := "./test/objectfiles/" + code.UnitName + ".asm"
			err := FilesAreEqual(objectFile, targetFile)
			if err != nil {
				fmt.Printf("Object file not correct: %s\n", err.Error())
			}
		}
	}
}
