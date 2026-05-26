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
	if len(f.returnTypes) > 0 {
		for {
			code.NewArgCode()
			values, err := ParseExpression(s)
			if err != nil {
				return err
			}
			for _, v := range values {
				if v.IsConst {
					if v.Typ.Pt.IsInteger() {
						EmitPushConst(v.IntValue, "Returned const value number "+strconv.Itoa(i))
					} else if v.Typ.Pt == TYP_STRING {
						EmitPushStringLit(v.StringLitNo, "Returned string lit number "+strconv.Itoa(i))
					} else {
						panic("Not implemented")
					}
				} else if v.LocalVar != nil {
					v.LocalVar.Value.IsTempObj = false
				}
				// Save returned value into reserved slot before BP.
				EmitStoreBpOfs(len(f.parameters)+len(f.returnTypes)-i+1, "Save returned value nr "+strconv.Itoa(i+1))
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

/*
if i >= len(f.returnTypes) && len(f.returnTypes) > 0 {
return fmt.Errorf("too many return values")
}
if !CanAssign(f.returnTypes[i].Pt, v.Typ.Pt) {
return fmt.Errorf("returns wrong type")
}
*/
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
			code.NewArgCode()
			values, _, err1 := ParseFuncCall(s, id, false)
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
