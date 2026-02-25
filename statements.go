package main

import (
	"errors"
	"fmt"
)

func ParseReturn(s *State) error {
	f := s.currentFunc
	requireRpar := s.found(TOK_LPAR)
	i := 0
	for {
		v, err := ParseExpression(s)
		if err != nil {
			return err
		}
		if !CanAssign(f.returnTypes[i].pt, v.typ.pt) {
			return fmt.Errorf("returns wrong type")
		}
		if v.hasValue {
			EmitPushConst(s, v)
		}
		if !s.found(TOK_COMMA) {
			break
		}
		i++
	}
	if requireRpar && !s.found(TOK_RPAR) {
		return errors.New("expected )")
	}
	if len(f.returnTypes) == 0 {
		return fmt.Errorf("function '%s' has no return_type declaration", f.name)
	}
	EmitReturn(s)
	return nil
}

// ParseStatement will parse the statements inside a {} block or similar.
// retured is true if the statemend emited a return instruction
func ParseStatement(s *State) (returned bool, err error) {
	if s.token == TOK_RETURN {
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
	} else if s.token == TOK_ASSERT {
		nextToken(s)
		var v ValueDef
		v, err = ParseExpression(s)
		if err != nil {
			return false, err
		}
		if v.hasValue {
			if !v.boolValue {
				return false, fmt.Errorf("assert failed")
			}
			emit(s, "Assert succeeded", "")
		} else {
			EmitAssert(s)
		}
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
		if returned {
			if s.hasReturned {
				return fmt.Errorf("statements afer return")
			}
			s.hasReturned = true
		}
		if s.token == TOK_SEMICOLON {
			nextToken(s)
		}
	}
	return nil
}
