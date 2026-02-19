package main

import "fmt"

// ParseStatement will parse the statements inside a {} block or similar.
// retured is true if the statemend emited a return instruction
func ParseStatement(s *State) (returned bool, err error) {
	if s.token == TOK_RETURN {
		if s.hasReturned {
			return true, fmt.Errorf("More than one return in block")
		}
		nextToken(s)
		_, err = ParseExpression(s)
		EmitReturn(s)
		returned = true
	} else if s.token == TOK_IF {
		err = ParseIf(s)
	} else if s.token == TOK_FOR {
		nextToken(s)
	} else if s.token == TOK_ASSERT {
		nextToken(s)
		var v ValueDef
		v, err = ParseExpression(s)
		if v.hasValue {
			if !v.boolValue {
				return false, fmt.Errorf("Assert failed")
			} else {
				emit(s, "Assert succeeded", "")
			}
		} else {
			EmitAssert(s)
		}
	} else if s.token == TOK_ID {
		_, err = ParseAssignOrCall(s)
	} else if s.token == TOK_SEMICOLON {
		// Ignore
		nextToken(s)
	} else if s.token == TOK_VAR {
		return false, ParseVars(s)
	} else if s.token == TOK_TYPE {
		return false, ParseTypeDefs(s)
	} else {
		return false, fmt.Errorf("Unknown statement starting with %s", s.tokenString)
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
				return fmt.Errorf("Statements afer return")
			}
			s.hasReturned = true
		}
		if s.token == TOK_SEMICOLON {
			nextToken(s)
		}
	}
	return nil
}
