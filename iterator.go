package main

import (
	"fmt"
	"strconv"

	"github.com/jkvatne/jkv/code"
)

var StartLabelStack []int
var EndLabelStack []int

func PushLabel(start, end int) {
	StartLabelStack = append(StartLabelStack, start)
	EndLabelStack = append(EndLabelStack, end)
}
func PopLabels() {
	StartLabelStack = StartLabelStack[:len(StartLabelStack)-1]
	EndLabelStack = EndLabelStack[:len(EndLabelStack)-1]
}

func GetTopStartLabel() int {
	return StartLabelStack[len(StartLabelStack)-1]
}

func GetTopEndLabel() int {
	return EndLabelStack[len(EndLabelStack)-1]
}

func ParseBreak(s *State) error {
	EmitJump(GetTopEndLabel(), "Break: Jump to end of loop")
	return nil
}

func ParseFail(s *State) error {
	if !s.found(TOK_LPAR) {
		return fmt.Errorf("Expected '(' after 'fail'")
	}
	if s.token == TOK_ID {
		id := s.tokenString
		s.next()
		v := VarDefs[id]
		if v != nil && v.Typ.Pt.IsInteger() {
			EmitStoreErr(int(v.Value.IntValue))
			EmitJump(s.returnLbl, "Failed with const var="+strconv.Itoa(int(v.Value.IntValue)))
		}
	} else if s.token == TOK_INT {
		c := VarDefs[s.tokenString]
		if !c.Typ.Pt.IsInteger() {
			return fmt.Errorf("Expected integer parameter for 'fail'")
		}
		EmitStoreErr(int(c.Value.IntValue))
		EmitJump(s.returnLbl, "Failed with const")
	}
	if !s.found(TOK_RPAR) {
		return fmt.Errorf("Expected ')' after 'fail'")
	}
	return nil
}

// ParseLoop is a simple loop depending on break to exit.
func ParseFor(s *State) error {
	startLabel := code.NewLabel()
	endLabel := code.NewLabel()
	EmitLabel(startLabel, "Start of loop")
	PushLabel(startLabel, endLabel)
	if !s.found(TOK_LBRACE) {
		return fmt.Errorf("expected { but got %s", s.tokenString)
	}
	err := ParseBlock(s, false)
	if err != nil {
		return err
	}
	if !s.found(TOK_RBRACE) {
		return fmt.Errorf("expected } after loop block, but got %s", s.tokenString)
	}
	EmitJump(GetTopStartLabel(), "Jump to start of loop")
	EmitLabel(endLabel, "Exit from loop")
	PopLabels()
	return err
}

/*
func ParseFor(s *State) error {
	startLabel := code.NewLabel()
	endLabel := code.NewLabel()
	PushLabel(startLabel, endLabel)
	id := s.tokenString
	s.next()
	lvalues, err := ParseLvalueList(s, id) // Args to yield
	if len(lvalues) == 0 {
		return fmt.Errorf("expected at least one variable in for loop, but got %s", s.tokenString)
	}
	if !s.found(TOK_ASSIGN) {
		return fmt.Errorf("expected '=' but got %s", s.tokenString)
	}
	// Now we should have the iterator function call
	if s.token != TOK_ID {
		return fmt.Errorf("expected iterator but got %s", s.tokenString)
	}
	// id is iterator function name.
	id = s.tokenString
	s.next()
	if !s.found(TOK_LPAR) {
		return fmt.Errorf("expected '(', but got %s", s.tokenString)
	}

	EmitPushLabel(startLabel)
	EmitPushFramePointer()

	code.OutputArgCode()
	code.NewArgCode()
	values, results, err := ParseFuncCall(s, id, true)
	if err != nil {
		return err
	}
	code.OutputArgCode()
	if len(results) != len(lvalues) {
		return fmt.Errorf("expected %d results, but got %d", len(lvalues), len(results))
	}
	for i := 0; i < len(lvalues); i++ {
		if lvalues[i].Typ == nil {
			lvalues[i].Typ = results[i]
			lvalues[i].Value.Typ = results[i]
		}
	}
	fmt.Printf("%d %d\n", len(values), len(results))
	if !s.found(TOK_LBRACE) {
		return fmt.Errorf("expected { but got %s", s.tokenString)
	}
	EmitLabel(startLabel, "Start of loop")
	err = ParseBlock(s, false)
	if err != nil {
		return err
	}
	if !s.found(TOK_RBRACE) {
		return fmt.Errorf("expected } after loop block, but got %s", s.tokenString)
	}
	EmitJump(GetTopStartLabel(), "Jump to start of loop")
	EmitLabel(endLabel, "Exit from loop")
	PopLabels()
	return err
}
*/

func ParseContinue(s *State) error {
	return fmt.Errorf("Continue not implemented")
}
