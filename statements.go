package main

import (
	"fmt"
	"strconv"
)

func ParseReturn(s *State) error {
	f := s.currentFuncDef
	i := 0
	if len(s.ArgCode) > 0 {
		panic("ArgCode was not empty")
	}
	if len(f.returnTypes) > 0 {
		for {
			PushArgCode(s)
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
					EmitPushConst(s, v.IntValue, "Returned value number "+strconv.Itoa(i))
				} else if v.Typ.Pt == TYP_STRING {
					EmitPushStringLit(s, v.StringLitNo, "Returned value number "+strconv.Itoa(i))
				} else {
					panic("Not implemented")
				}
			} else {
				// Copy return values to stack are where the caller expects them
				if !s.RaxIsTOS {
					// emit(s, "pop", "rax", "", "Pop return value to rax")
				}
			}
			emit(s, "mov", BpRel(16+i*8+len(f.parameters)*8), "rax", "")
			if !s.found(TOK_COMMA) {
				break
			}
			i++
			ConsArgCode(s, 2, false)
		}
		if len(f.returnTypes) == 0 {
			return fmt.Errorf("function '%s' has no return_type declaration", f.name)
		}
	}
	OutputArgCode(s)
	s.DidReturn = true
	s.Returning = false
	return nil
}

// ParseStatement will parse the statements inside a {} block or similar.
// returned is true if the statement emitted a return instruction
func ParseStatement(s *State) (returned bool, err error) {
	s.DidReturn = false
	if s.XmmSp != 0 || s.localSp > 2 {
		// fmt.Printf("Line no %d: XmmSp=%d  localSp=%d\n", s.lineNum, s.XmmSp, s.localSp)
	}
	s.XmmSp = 0
	if s.token == TOK_ID {
		id := s.tokenString
		s.next()
		if s.found(TOK_LPAR) {
			PushArgCode(s)
			values, err1 := ParseFuncCall(s, id, false)
			if err1 != nil {
				return false, err1
			}
			if len(values) > 0 {
				return false, fmt.Errorf("function '%s' has returns a value that is never used", id)
			}
		} else {
			err = ParseAssign(s, id)
		}
		OutputArgCode(s)
	} else if s.token == TOK_RETURN {
		nextToken(s)
		if s.hasReturned {
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
	// CheckLocalSp(s, "Line "+strconv.Itoa(s.lineNum))
	return returned, err
}

func ParseStatements(s *State) error {
	s.hasReturned = false
	for s.token != TOK_RBRACE && s.token != TOK_COLON {
		if s.DidReturn {
			emit(s, "jmp", ".L"+strconv.Itoa(s.returnLbl), "", "Jump to return")
			s.DidReturn = false
		}
		EmitLineNo(s)
		returned, err := ParseStatement(s)
		if err != nil {
			return err
		}
		EmitPrintSp(s)
		if returned {
			if s.hasReturned {
				return fmt.Errorf("statements after return")
			}
			s.hasReturned = true
		}
		if s.token == TOK_SEMICOLON {
			nextToken(s)
		}
		s.RaxIsTOS = false
	}
	EmitLineNo(s)
	return nil
}
