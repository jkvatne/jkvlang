package main

import (
	"fmt"
	"log/slog"
	"strconv"
)

func CompileFile(name string, workdir string) error {
	// slog.Info("Compiling", "filename", name, "workdir", workdir)
	s, err := NewState(name, workdir)
	if err != nil {
		return err
	}
	defer func(s *State) {
		_ = CloseObjFile(s)
	}(s)

	LiteralInit()
	EmitPrologue(s)

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
	EmitSection(s, "rodata")
	for i, l := range LiteralDefs {
		// ALl strings must be aligned to qword
		EmitLitteral(s, "str"+strconv.Itoa(i), l)
	}
	for i, l := range FloatLiteralDefs {
		EmitFloatLitteral(s, "flt"+strconv.Itoa(i+1), l)
	}
	if err != nil {
		return fmt.Errorf("%s:%d %v", name, s.lineNum, err)
	}
	if s.CommentLevel > 0 {
		return fmt.Errorf("Missing end of comment")
	}
	return nil
}
