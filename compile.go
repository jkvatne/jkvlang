package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type state = struct {
	text []byte
	p    int
}

func Compile(workdir string, inputPath string, outputName string) error {
	entries, err := os.ReadDir(inputPath)
	if err != nil {
		return fmt.Errorf("Fatal error " + err.Error())
	}
	s := new(state)
	for _, entry := range entries {
		if !entry.IsDir() {
			slog.Info("Compiling", "filename", entry.Name())
			s.text, err = os.ReadFile(filepath.Join(inputPath, entry.Name()))
			if err != nil {
				slog.Error("Could not open file %s : %s", entry.Name(), err.Error())
			}
			CompileFile(s, workdir)
		}
	}
	return fmt.Errorf("Work in progress")
}

func CompileFile(s *state, workdir string) {
	token := next(s)
	slog.Info("Compiling", "token", token)
}

func next(s *state) string {
	return "0"
}
