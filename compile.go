package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func CompileFile(name string, workdir string) error {
	slog.Info("Compiling", "filename", name, "workdir", workdir)
	var err error
	s := new(State)
	s.lineNum = 1
	s.text, err = os.ReadFile(name)
	if err != nil {
		slog.Error("Could not open file %s : %s", name, err.Error())
	}
	s.unitName = strings.TrimSuffix(filepath.Base(name), ".jkv")

	objectFile := filepath.Join(workdir, s.unitName+".asm")
	s.outputFile, err = os.OpenFile(objectFile, os.O_CREATE|os.O_TRUNC, os.ModePerm)
	defer func(s *State) {
		_ = CloseObjFile(s)
	}(s)
	EmitComment(s, "File \""+objectFile+"\"\n")

	if err != nil {
		return err
	}
	InitTypes()
	FuncInit()
	nextToken(s)
	if s.token == TOK_EOF {
		return fmt.Errorf("no program content in file")
	}

	// Top level statements can only be func, const or type.
	// Global variables are not allowed!
	for s.token != TOK_EOF && err == nil {
		if s.token == TOK_FUNC {
			err = ParseFunctionDefinition(s)
		} else if s.token == TOK_CONST {
			err = ParseConsts(s)
		} else if s.token == TOK_TYPE {
			err = ParseTypeDefs(s)
		} else {
			slog.Error("Unexpected", "token", s.tokenString)
			err = fmt.Errorf("unexpected token \"%s\"", s.tokenString)
		}
	}
	if err != nil {
		EmitError(s, err)
		return fmt.Errorf("%s Line %d: %v", name, s.lineNum, err)
	}
	return nil
}
