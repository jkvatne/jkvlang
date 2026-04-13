package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func CompileFile(name string, workdir string) error {
	// slog.Info("Compiling", "filename", name, "workdir", workdir)
	var err error
	s := new(State)
	s.LibPath, err = filepath.Abs("../lib/")
	s.LibPath += string(os.PathSeparator)
	s.ArgCode = make([]string, 0, 64)
	LiteralInit()
	s.lineNum = 1
	s.text, err = os.ReadFile(name)
	if err != nil {
		slog.Error("Could not open file %s : %s", name, err.Error())
	}
	s.unitName = strings.TrimSuffix(filepath.Base(name), ".jkv")

	objectFile := filepath.Join(workdir, s.unitName+".asm")
	s.outputFile, err = os.Create(objectFile)
	defer func(s *State) {
		_ = CloseObjFile(s)
	}(s)
	EmitComment(s, "File \""+objectFile+"\"\n")
	EmitPrologue(s)

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
			err = ParseFuncDef(s)
		} else if s.token == TOK_CONST {
			err = ParseConsts(s)
		} else if s.token == TOK_TYPE {
			err = ParseTypeDefs(s)
		} else {
			slog.Error("Unexpected", "token", s.tokenString)
			err = fmt.Errorf("unexpected token \"%s\"", s.tokenString)
		}
	}

	EmitSection(s, "rodata")
	for i, l := range LiteralDefs {
		// ALl strings must be aligned to qword
		EmitCode(s, "alignb 8\n")
		EmitLitteral(s, "str"+strconv.Itoa(i), l)
	}
	if err != nil {
		EmitError(s, err)
		return fmt.Errorf("%s:%d %v", name, s.lineNum, err)
	}
	return nil
}
