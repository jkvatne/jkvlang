package main

import (
	"fmt"
	"strconv"
	"strings"

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
	if len(EndLabelStack) == 0 {
		fmt.Printf("Empty EndLabelStack on line %d\n", code.LineNum)
		return 0
	}
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
		} else {
			return fmt.Errorf("fail() needs an integer argument")
		}
	} else if s.token == TOK_NUM {
		e, _ := strconv.Atoi(s.tokenString)
		EmitStoreErr(e)
		EmitJump(s.returnLbl, "Failed with const")
		s.next()
	}
	if !s.found(TOK_RPAR) {
		return fmt.Errorf("expected ')' after 'fail'")
	}
	return nil
}

// ParseHash will catch errors
func ParseHash(s *State) error {
	s.next()
	err := ParseStatement(s)
	return err
}

// ParseLoopVars will parse the variables receiveng values from next(r).
// Theese variables are f.ex. "i" in " i=range(1,2)"
// They are returned in the lvalues[] array.
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
		EmitAllocLocalVar("Allocate loop variable " + lvalue.Name)
		lvalues = append(lvalues, lvalue)
		s.next()
		if !s.found(TOK_COMMA) {
			break
		}
	}
	return lvalues, nil
}

var rangeCount = 1

func ParseFor(s *State) error {
	startLabel := code.NewLabel()
	endLabel := code.NewLabel()
	PushLabel(startLabel, endLabel)
	var lvalues []*VarDef
	var err error
	rangeName := "___range" + strconv.Itoa(rangeCount)
	rangeCount++
	if !s.found(TOK_LBRACE) {
		s.BlockLevel++
		s.LoopLevel++
		lvalues, err = ParseLoopVars(s)
		if err != nil {
			return err
		}
		if len(lvalues) == 0 {
			return fmt.Errorf("expected at least one variable in for loop, but got %s", s.tokenString)
		}
		if !s.found(TOK_ASSIGN) {
			return fmt.Errorf("expected '=' but got %s", s.tokenString)
		}
		if s.token != TOK_ID {
			return fmt.Errorf("expected function name but got %s", s.tokenString)
		}
		// Now parse the function returning the range
		id, err2 := HandleImportedName(s)
		if err2 != nil {
			return err2
		}

		if !s.found(TOK_LPAR) {
			return fmt.Errorf("expected '(' but got %s", s.tokenString)
		}
		code.NewArgCode()
		// Call the range function
		AddLocalVar(s, rangeName, &StructType)
		results, err := ParseFuncCall(s, id, true)
		StructType.StructSize = 8

		if err != nil {
			return err
		}
		if len(results) != 1 {
			return fmt.Errorf("expected a single state in for-loop")
		}
		code.OutputArgCode()
		p := &ParDef{name: "rec", typ: &StructType}
		fun := "next"
		if strings.Contains(id, "@") {
			pkg, _ := DeMangleFun(id)
			if pkg != "" {
				fun = MangleFun(pkg, fun)
			}
		}
		f := FindFuncDefFromParList(fun, []*ParDef{p})
		if f == nil {
			return fmt.Errorf("range must have a next function")
		}
		lvalues[0].Typ = f.returnTypes[0]
		VarDefs[lvalues[0].Name].Typ = f.returnTypes[0]

		EmitLabel(startLabel, "Start of loop")

		// Call the next function
		EmitCall(f.label, 1, false)
		// Assign result to loop variable
		if !s.found(TOK_LBRACE) {
			return fmt.Errorf("expected '{' but got %s", s.tokenString)
		}
		EmitJumpTrue("r15", endLabel, "")

		err = ParseBlock(s, false, false)

		if err != nil {
			return err
		}
		if !s.found(TOK_RBRACE) {
			return fmt.Errorf("expected } after loop block, but got %s", s.tokenString)
		}
		EmitJump(GetTopStartLabel(), "Jump to start of loop")
		EmitLabel(endLabel, "Exit from loop")
		PopLabels()
		EmitPopAx("Pop address of loop state")
		DeleteLocalVar(s, rangeName)
		// Remove local loop variables
		for _, v := range lvalues {
			EmitPopAx("Pop loop variable")
			DeleteLocalVar(s, v.Name)
		}
		// DeleteBlockVars(s)
		EmitReturnIfErr2(s.returnLbl)
		// Clear err if it is 1 as this is used to signal break using pull iterators
		EmitClearBreakErr()
		s.BlockLevel--
		s.LoopLevel--
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
	s.LoopLevel++
	err := ParseBlock(s, false, false)
	if err != nil {
		return err
	}
	if !s.found(TOK_RBRACE) {
		return fmt.Errorf("expected } after loop block, but got %s", s.tokenString)
	}
	// Remove local loop variables
	// DeleteBlockVars(s)
	EmitJump(GetTopStartLabel(), "Jump to start of loop")
	EmitLabel(endLabel, "Exit from loop")
	// Clear err if it is 1 as this is used to signal break using pull iterators
	EmitClearBreakErr()
	PopLabels()
	s.LoopLevel--
	return err
}
