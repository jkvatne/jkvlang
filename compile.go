package main

import (
	"fmt"
	"strconv"

	"github.com/jkvatne/jkv/code"
)

func CompileFile(name string, workdir string, libPath string) error {
	err := code.New(name, workdir)
	if err != nil {
		return err
	}
	InitVardefs()
	InitTypes()
	s, err := NewState(name)
	if err != nil {
		return err
	}
	defer func(s *State) {
		_ = code.CloseObjFile()
	}(s)

	LiteralInit()
	EmitPrologue(libPath, true)

	InitTypes()
	FuncInit()
	nextToken(s)

	// Top level statements can only be func, const or type.
	// Global variables are not allowed!
	for s.token != TOK_EOF {
		if s.token == TOK_FUNC {
			err = ParseFuncDef(s)
		} else if s.token == TOK_CONST {
			s.next()
			err = ParseConsts(s)
		} else if s.token == TOK_TYPE {
			s.next()
			err = ParseTypeDefs(s)
		} else if s.token == TOK_VAR {
			s.next()
			err = ParseVars(s)
		} else {
			err = fmt.Errorf("unexpected token \"%s\"", s.tokenString)
		}
		if err != nil {
			return fmt.Errorf("%s:%d %v", name, code.LineNum, err)
		}
	}
	EmitSection("rodata")
	for i, l := range StringLiteralDefs {
		// ALl strings must be aligned to qword
		EmitStringLitteral("str"+strconv.Itoa(i), l)
	}
	for i, l := range F64LiteralDefs {
		EmitF64Litteral("f64_"+strconv.Itoa(i+1), l)
	}
	for i, l := range F32LiteralDefs {
		EmitF32Litteral("f32_"+strconv.Itoa(i+1), l)
	}
	if s.CommentLevel > 0 {
		return fmt.Errorf("missing end of comment")
	}
	return nil
}
