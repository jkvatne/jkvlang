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
	FreeBlockVars(s)
	EmitJump(GetTopEndLabel(), "Break: Jump to end of loop")
	return nil
}

func ParseContinue() error {
	sl := GetTopStartLabel()
	EmitJump(sl, "Continue")
	return fmt.Errorf("continue not implemented")
}

func ParseFail(s *State) error {
	if !s.found(TOK_LPAR) {
		return fmt.Errorf("expected '(' after 'fail'")
	}
	if s.token == TOK_ID {
		id := s.tokenString
		s.next()
		v := VarDefs[id]
		if v != nil && v.Typ.Pt.IsInteger() {
			EmitStoreErr(1) // TODO
			EmitJump(s.returnLbl, "Failed with const var="+strconv.Itoa(1))
		}
	} else if s.token == TOK_INT {
		c := VarDefs[s.tokenString]
		if !c.Typ.Pt.IsInteger() {
			return fmt.Errorf("expected integer parameter for 'fail'")
		}
		// EmitStoreErr(int(c.Value.IntValue))
		EmitJump(s.returnLbl, "Failed with const")
	}
	if !s.found(TOK_RPAR) {
		return fmt.Errorf("expected ')' after 'fail'")
	}
	return nil
}

func ParseLoopVars(s *State) (lvalues []*VarDef, err error) {
	for {
		if s.token != TOK_ID {
			return nil, fmt.Errorf("loop variables expected")
		}
		id := s.tokenString
		lvalue := VarDefs[id]
		if lvalue != nil {
			return nil, fmt.Errorf("loop variable shadowing " + id)
		}
		lvalue = AddLocalVar(s, id, nil)
		lvalues = append(lvalues, lvalue)
		if !s.found(TOK_COMMA) {
			break
		}
	}
	return lvalues, nil
}

func ParseFor(s *State) error {
	startLabel := code.NewLabel()
	endLabel := code.NewLabel()
	PushLabel(startLabel, endLabel)
	var lvalues []*VarDef
	var err error
	if !s.found(TOK_LBRACE) {
		s.BlockLevel++
		lvalues, err = ParseLoopVars(s)
		if err != nil {
			return err
		}
		if len(lvalues) == 0 {
			return fmt.Errorf("expected at least one variable in for loop, but got %s", s.tokenString)
		}
		for _, l := range lvalues {
			ofs := EmitAllocLocalVar("Allocate loop variable " + l.Name)
			l.Offset = ofs
		}
		s.next()
		if !s.found(TOK_ASSIGN) {
			return fmt.Errorf("expected '=' but got %s", s.tokenString)
		}
		// Now parse the function returning the range
		id := s.tokenString
		if !s.found(TOK_ID) {
			return fmt.Errorf("expected function name but got %s", s.tokenString)
		}
		if !s.found(TOK_LPAR) {
			return fmt.Errorf("expected '(' but got %s", s.tokenString)
		}
		code.NewArgCode()
		results, err := ParseFuncCall(s, id, true)
		if err != nil {
			return err
		}
		if len(results) != 1 {
			return fmt.Errorf("expected a single state in for-loop")
		}
		code.OutputArgCode()
		f := FuncDefs["next"]
		if f == nil {
			return fmt.Errorf("range must have a next function")
		}
		lvalues[0].Typ = f.returnTypes[0]
		VarDefs[lvalues[0].Name].Typ = f.returnTypes[0]
		// Insert call next() before for block
		EmitLabel(startLabel, "Start of loop")
		EmitCall("next", 1, false)
		// Assign result to loop variable
		if !s.found(TOK_LBRACE) {
			return fmt.Errorf("expected '{' but got %s", s.tokenString)
		}
		emit("or", "r15", "r15", "")
		emit("jnz", Label(endLabel), "", "")
		err = ParseBlock(s, false)
		if err != nil {
			return err
		}
		if !s.found(TOK_RBRACE) {
			return fmt.Errorf("expected } after loop block, but got %s", s.tokenString)
		}
		EmitJump(GetTopStartLabel(), "Jump to start of loop")
		EmitLabel(endLabel, "Exit from loop")
		emit("mov", "r15", "0", "")
		// Clear err if it is 1 as this is used to signal break using pull iterators
		EmitClearBreakErr()
		PopLabels()
		// EmitFreeStruct assumes the full address exists in rax. So just pop it as the state is now TOS.
		EmitPopAx("Pop address of loop state")
		EmitFreeStruct(results[0].Typ.Size(), "Free loop state")
		// Remove local loop variables
		for _, v := range lvalues {
			EmitPopAx("Pop loop variable")
			DeleteLocalVar(s, v.Name)
		}
		DeleteBlockVars(s)
		s.BlockLevel--
	}
	return err
}

// ParseLoop is a simple loop depending on break to exit.
func ParseLoop(s *State) error {
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
	// Clear err if it is 1 as this is used to signal break using pull iterators
	EmitClearBreakErr()
	PopLabels()
	return err
}
