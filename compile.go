package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func CheckFile(s *State, workdir string) {
	for s.token != TOK_EOF {
		nextToken(s)
		slog.Info("Token", "Lno", s.lineNum, "Value", s.token, "String", s.tokenString)
		usedToken[s.token] = true
	}
	for i, t := range usedToken {
		if t == false && i > 0 {
			slog.Error("Missing", "token", i)
		}
	}
}

func CompileFile(name string, workdir string) error {
	slog.Info("Compiling", "filename", name)
	var err error
	s := new(State)
	s.lineNum = 1
	s.text, err = os.ReadFile(name)
	if err != nil {
		slog.Error("Could not open file %s : %s", name, err.Error())
	}
	s.unitName = strings.TrimSuffix(filepath.Base(name), ".jkv")

	objectFile := filepath.Join(workdir, s.unitName+".tok")
	s.outputFile, err = os.OpenFile(objectFile, os.O_CREATE|os.O_TRUNC, os.ModePerm)
	defer CloseObjFile(s)
	emit(s, "   // Token file ", objectFile)

	if err != nil {
		return err
	}
	InitTypes()
	FuncInit()
	nextToken(s)
	if s.token == TOK_EOF {
		return fmt.Errorf("No program content in file")
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
			err = fmt.Errorf("Unexpected token \"%s\"", s.tokenString)
		}
	}
	if err != nil {
		EmitError(s, err)
	}
	return fmt.Errorf("%s Line %d: %v", name, s.lineNum, err)
}
