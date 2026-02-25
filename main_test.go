package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		unitName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		name := filepath.Join(inputPath, entry.Name())
		fmt.Printf("Compiling %s\n", entry.Name())
		err = CompileFile(name, outputPath)
		if strings.HasPrefix(unitName, "err_") {
			if err == nil {
				t.Errorf("Expected error in %s", unitName)
			}
			continue
		} else if err != nil {
			t.Errorf("Error in file \"%s.jkv\" : %v", unitName, err)
		} else {
			targetFile := "./test/targets/" + unitName + ".tok"
			objectFile := "./test/objectfiles/" + unitName + ".tok"
			err := FilesAreEqual(objectFile, targetFile)
			if err != nil {
				fmt.Printf("Object file not correct: %s\n", err.Error())
			}
		}
	}
}
