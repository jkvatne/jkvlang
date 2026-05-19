package main

import (
	"fmt"
	"strconv"

	"github.com/jkvatne/jkv/code"
)

func ParseReturn(s *State) error {
	f := s.currentFuncDef
	i := 0
	if len(code.ArgCode) > 0 {
		panic("ArgCode was not empty")
	}
	if len(f.returnTypes) > 0 {
		for {
			code.PushArgCode()
			v, err := ParseExpression(s)
			if err != nil {
				return err
			}
			if len(f.returnTypes) <= i && len(f.returnTypes) > 0 {
				return fmt.Errorf("too many return values")
			}
			if !CanAssign(f.returnTypes[i].Pt, v.Typ.Pt) {
				return fmt.Errorf("returns wrong type")
			}
			if v.HasValue {
				if v.Typ.Pt.IsInteger() {
					EmitPushConst(v.IntValue, "Returned value number "+strconv.Itoa(i))
				} else if v.Typ.Pt == TYP_STRING {
					EmitPushStringLit(v.StringLitNo, "Returned value number "+strconv.Itoa(i))
				} else {
					panic("Not implemented")
				}
			}
			EmitStoreReturnValue(i + len(f.parameters))
			if !s.found(TOK_COMMA) {
				break
			}
			i++
			code.ConsArgCode(2, false)
		}
		if len(f.returnTypes) == 0 {
			return fmt.Errorf("function '%s' has no return_type declaration", f.name)
		}
	}
	code.OutputArgCode()
	s.DidReturn = true
	return nil
}

// ParseStatement will parse the statements inside a {} block or similar.
// returned is true if the statement emitted a return instruction
func ParseStatement(s *State) (returned bool, err error) {
	s.DidReturn = false
	if s.token == TOK_ID {
		id := s.tokenString
		s.next()
		if s.found(TOK_LPAR) {
			if len(code.ArgCode) > 0 {
				panic("ArgCode was not empty")
			}
			code.PushArgCode()
			values, err1 := ParseFuncCall(s, id, false)
			if err1 != nil {
				return false, err1
			}
			if len(values) > 0 {
				return false, fmt.Errorf("function '%s' has returns a value that is never used", id)
			}
			code.OutputArgCode()
		} else {
			err = ParseAssign(s, id)
		}
	} else if s.token == TOK_RETURN {
		nextToken(s)
		if s.HasReturned {
			return true, fmt.Errorf("more than one return in block")
		}
		err = ParseReturn(s)
		returned = true
	} else if s.token == TOK_IF {
		err = ParseIf(s)
	} else if s.token == TOK_FOR {
		nextToken(s)
	} else if s.token == TOK_SEMICOLON {
		// Ignore
		nextToken(s)
	} else if s.token == TOK_VAR {
		return false, ParseVars(s)
	} else if s.token == TOK_TYPE {
		return false, ParseTypeDefs(s)
	} else {
		return false, fmt.Errorf("unknown statement starting with %s", s.tokenString)
	}
	return returned, err
}

func ParseStatements(s *State) error {
	s.HasReturned = false
	for s.token != TOK_RBRACE && s.token != TOK_COLON {
		if s.DidReturn {
			EmitJump(s.returnLbl, "Jump to return")
			s.DidReturn = false
		}
		code.EmitLineNo(s.currentLine, code.LocalSp)
		returned, err := ParseStatement(s)
		if err != nil {
			return err
		}
		EmitPrintSp()
		if returned {
			if s.HasReturned {
				return fmt.Errorf("statements after return")
			}
			s.HasReturned = true
		}
		if s.token == TOK_SEMICOLON {
			nextToken(s)
		}
		code.RaxIsTOS = false
	}
	code.EmitLineNo(s.currentLine, code.LocalSp)
	return nil
}
