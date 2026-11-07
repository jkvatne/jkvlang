package main

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/jkvatne/jkv/code"
)

func ParseBreak(s *State) error {
	slog.Debug("Break", "level", s.BlockLevel)
	EmitStoreErr(1)
	err := FreeBlockVarsNoDelete(s, s.BlockLevel)
	if err != nil {
		return err
	}
	EmitJump(s.GetLoopBreakLabel(), "Jump to break label")
	return nil
}

func ParseContinue(s *State) error {
	sl := s.GetLoopStartLabel()
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

// ParseLoopVars will parse the variables receiving values from next(r).
// These variables are f.ex. "i" in " i=range(1,2)"
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
	s.PushForBlock()
	var lvalues []*VarDef
	var err error
	rangeName := "___range" + strconv.Itoa(rangeCount)
	rangeCount++
	s.BlockLevel++
	slog.Debug("-- Setup for-loop, increasing block level", "Func", s.currentFuncDef.name, "Line", code.LineNum, "BlockLevel", s.BlockLevel)
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
	AddLocalVar(s, rangeName, &StructType)
	code.NewArgCode()
	// Call the range function
	results, err := ParseFuncCall(s, id, true)
	StructType.StructSize = 8

	if err != nil {
		return err
	}
	if len(results) != 1 {
		return fmt.Errorf("expected a single state in for-loop")
	}
	code.ConsArgCode(len(code.ArgCode), false)
	code.OutputArgCode()
	p := &ParDef{name: "rec", typ: &StructType}
	fun := "next"
	if strings.Contains(id, SepChar) {
		pkg, _ := DeMangleFun(id)
		if pkg != "" {
			fun = MangleFun(pkg, fun)
		}
	}
	f := FindFuncDefFromParList(fun, []*ParDef{p}, false)
	if f == nil {
		return fmt.Errorf("range must have a next function")
	}
	lvalues[0].Typ = f.returnTypes[0]
	VarDefs[lvalues[0].Name].Typ = f.returnTypes[0]

	slog.Debug("-- Start of for-loop", "Func", s.currentFuncDef.name, "Line", code.LineNum, "BlockLevel", s.BlockLevel)
	EmitLabel(s.GetLoopStartLabel(), "Start of loop")
	EmitLabel(s.GetLoopBreakLabel(), "Start of loop")

	// Call the next function
	EmitCall(f.label, 1, false)
	// Assign result to loop variable
	if !s.found(TOK_LBRACE) {
		return fmt.Errorf("expected '{' but got %s", s.tokenString)
	}
	EmitJumpTrue("r15", s.GetLoopExitLabel(), "Exit loop if err==1 after call to next")

	s.BlockLevel++
	slog.Debug("ParseFor: Increase BlockLevel", "BlockLevel", s.BlockLevel)

	err = ParseStatements(s)
	if err != nil {
		return err
	}
	err = FreeBlockVars(s, s.BlockLevel)
	if err != nil {
		return err
	}
	s.BlockLevel--
	slog.Debug("ParseFor: Decrease BlockLevel 1", "BlockLevel", s.BlockLevel)

	if !s.found(TOK_RBRACE) {
		return fmt.Errorf("expected } after loop block, but got %s", s.tokenString)
	}
	EmitJump(s.GetLoopStartLabel(), "Jump to start of loop")

	EmitComment("End of loop")
	EmitLabel(s.GetLoopExitLabel(), "")
	slog.Debug("-- End for-loop", "Func", s.currentFuncDef.name, "Line", code.LineNum, "BlockLevel", s.BlockLevel)
	s.PopLoopBlock()
	err = FreeBlockVars(s, s.BlockLevel)
	if err != nil {
		return err
	}
	s.BlockLevel--
	slog.Debug("ParseFor: Decrease BlockLevel 2", "BlockLevel", s.BlockLevel)
	EmitReturnIfErr2(s.returnLbl)
	// Clear err if it is 1 as this is used to signal break using pull iterators
	EmitClearBreakErr()
	return err
}

// ParseLoop is a simple loop depending on break to exit.
func ParseLoop(s *State) error {
	s.BlockLevel++
	s.PushLoopBlock()
	slog.Debug("-- Start of loop, increasing block level", "Func", s.currentFuncDef.name, "Line", code.LineNum, "BlockLevel", s.BlockLevel)
	EmitLabel(s.GetLoopStartLabel(), "Start of loop")
	if !s.found(TOK_LBRACE) {
		return fmt.Errorf("expected { but got %s", s.tokenString)
	}
	err := ParseStatements(s)
	if err != nil {
		return err
	}
	if !s.found(TOK_RBRACE) {
		return fmt.Errorf("expected } after loop block, but got %s", s.tokenString)
	}
	err = FreeBlockVarsNoDelete(s, s.BlockLevel)
	if err != nil {
		return err
	}
	EmitJump(s.GetLoopStartLabel(), "Jump to start of loop")

	// Note that we come here only from a break.
	EmitComment("End of loop")
	slog.Debug("-- End of loop 1", "Func", s.currentFuncDef.name, "Line", code.LineNum, "BlockLevel", s.BlockLevel)
	EmitLabel(s.GetLoopBreakLabel(), "")
	err = FreeBlockVars(s, s.BlockLevel)
	if err != nil {
		return err
	}
	EmitLabel(s.GetLoopExitLabel(), "")

	// Clear err if it is 1 as this is used to signal break using pull iterators
	EmitClearBreakErr()
	s.PopLoopBlock()
	err = FreeBlockVars(s, s.BlockLevel)
	if err != nil {
		return err
	}
	s.BlockLevel--
	slog.Debug("-- End of loop 2: done, decrease BlockLevel", "BlockLevel", s.BlockLevel)
	return err
}
