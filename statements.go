package main

import (
	"fmt"
	"strconv"
)

func ParseReturn(s *State) error {
	f := s.currentFunc
	// requireRpar := s.found(TOK_LPAR)
	i := 0
	if len(f.returnTypes) > 0 {
		for {
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
				EmitPushConst(s, v.IntValue, "Return value "+strconv.Itoa(i))
			}
			if !s.found(TOK_COMMA) {
				break
			}
			i++
		}
		if len(f.returnTypes) == 0 {
			return fmt.Errorf("function '%s' has no return_type declaration", f.name)
		}
	}
	EmitReturn(s)
	return nil
}

// ParseStatement will parse the statements inside a {} block or similar.
// returned is true if the statement emitted a return instruction
func ParseStatement(s *State) (returned bool, err error) {
	if s.XmmSp != 0 || s.localSp > 2 {
		fmt.Printf("Line no %d: XmmSp=%d  localSp=%d\n", s.lineNum, s.XmmSp, s.localSp)
	}
	s.XmmSp = 0
	s.localSp = 1
	if s.token == TOK_ID {
		id := s.tokenString
		s.next()
		err = ParseAssignOrCall(s, id)
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
	} else if s.token == TOK_ID {
		id := s.tokenString
		s.next()
		err = ParseAssignOrCall(s, id)
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
	s.hasReturned = false
	for s.token != TOK_RBRACE && s.token != TOK_COLON {
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
