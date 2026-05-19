package main

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"

	"github.com/jkvatne/jkv/code"
)

func CompileFile(name string, workdir string) error {
	err := code.New(name, workdir)
	if err != nil {
		return err
	}
	s, err := NewState(name, workdir)
	if err != nil {
		return err
	}
	defer func(s *State) {
		_ = code.CloseObjFile()
	}(s)

	LiteralInit()
	libPath, err := filepath.Abs("../lib/")
	EmitPrologue(libPath)

	InitTypes()
	FuncInit()
	nextToken(s)

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
	EmitSection("rodata")
	for i, l := range LiteralDefs {
		// ALl strings must be aligned to qword
		EmitLitteral("str"+strconv.Itoa(i), l)
	}
	for i, l := range FloatLiteralDefs {
		EmitFloatLitteral("flt"+strconv.Itoa(i+1), l)
	}
	if err != nil {
		return fmt.Errorf("%s:%d %v", name, s.lineNum, err)
	}
	if s.CommentLevel > 0 {
		return fmt.Errorf("missing end of comment")
	}
	return nil
}
