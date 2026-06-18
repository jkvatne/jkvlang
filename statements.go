package main

import (
	"fmt"
	"strconv"

	"github.com/jkvatne/jkv/code"
)

func ParseReturn(s *State) error {
	f := s.currentFuncDef
	if len(code.ArgCode) > 0 {
		panic("ArgCode was not empty")
	}
	i := 0
	if len(f.returnTypes) == 0 {
		EmitJump(s.returnLbl, "Return")
	} else {
		for {
			code.NewArgCode()
			values, err := ParseExpression(s)
			if err != nil {
				return err
			}
			for _, v := range values {
				if !CanAssign(f.returnTypes[i].Pt, v.Typ.Pt) {
					return fmt.Errorf("returns wrong type")
				}
				if v.IsConst {
					if v.Typ.Pt.IsInteger() {
						EmitPushConst(v.IntValue, "Returned const value number "+strconv.Itoa(i))
					} else if v.Typ.Pt == code.TYP_STRING {
						EmitPushStringLit(v.StringLitNo, "Returned string lit number "+strconv.Itoa(i))
					} else if v.Typ.Pt == code.TYP_BOOL {
						EmitPushConst(v.IntValue, "Bool const")
					} else {
						panic("Not implemented")
					}

				}
				// Save returned value into reserved slot before BP.
				EmitStoreBpOfs(len(f.parameters)+2+i, "Save returned value nr "+strconv.Itoa(i+1))
				code.SetSp()
				i++
			}
			if !s.found(TOK_COMMA) {
				break
			}
		}
		if len(f.returnTypes) != i {
			return fmt.Errorf("expected %d returns but got %d", len(f.returnTypes), i)
		}
	}
	code.ConsArgCode(i, false)
	code.OutputArgCode()
	s.DidReturn = true
	return nil
}

// ParseStatement will parse the statements inside a {} block or similar.
// returned is true if the statement emitted a return instruction
func ParseStatement(s *State) (err error) {
	switch s.token {
	case TOK_ID:
		id := s.tokenString
		s.next()
		if s.found(TOK_LPAR) {
			if len(code.ArgCode) > 0 {
				panic("ArgCode was not empty")
			}
			code.NewArgCode()
			values, err1 := ParseFuncCall(s, id, false)
			if err1 != nil {
				return err1
			}
			if len(values) > 0 {
				return fmt.Errorf("function '%s' has returns a value that is never used", id)
			}
			code.OutputArgCode()
		} else {
			err = ParseAssign(s, id)
		}
		if s.token == TOK_ELSE {
			s.next()
			lbl := code.NewLabel()
			EmitJumpOnError(lbl)
			err = ParseStatement(s)
			EmitLabel(lbl, "")
		}
	case TOK_RETURN:
		s.next()
		err = ParseReturn(s)
	case TOK_IF:
		err = ParseIf(s)
	case TOK_SEMICOLON:
		// Ignore
		nextToken(s)
	case TOK_VAR:
		s.next()
		err = ParseVars(s)
	case TOK_TYPE:
		s.next()
		err = ParseTypeDefs(s)
	case TOK_BREAK:
		s.next()
		err = ParseBreak(s)
	case TOK_FAIL:
		s.next()
		err = ParseFail(s)
	case TOK_CONTINUE:
		s.next()
		err = ParseContinue()
	case TOK_FOR:
		s.next()
		err = ParseFor(s)
	case TOK_LOOP:
		s.next()
		err = ParseLoop(s)
	default:
		err = fmt.Errorf("unknown statement starting with %s", s.tokenString)
	}
	return err
}

func ParseStatements(s *State) error {
	for s.token != TOK_RBRACE && s.token != TOK_COLON {
		code.EmitLineNo(s.currentLine)
		err := ParseStatement(s)
		if err != nil {
			return err
		}
		EmitPrintSp()
		if s.token == TOK_SEMICOLON {
			nextToken(s)
		}
		code.SetUndef()
	}
	code.EmitLineNo(s.currentLine)
	return nil
}
