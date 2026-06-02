package main

import (
	"log/slog"
	"os"
)

type State struct {
	text            []byte  // The whole current file being compiled
	p               int     // Points to the current character in text
	currentLine     string  // The content of the current source code text line
	AtLineEnd       bool    // Flag used for lineNum calculation
	token           Token   // The current token as a number
	tokenString     string  // The current token as a string
	tokenFloatValue float64 // The current token as a float (if it is a number)
	tokenIntValue   int64
	tokenUintValue  uint64
	noCode          int      // Used to skip code generation in constant if/else statements.
	LocalVarCount   int      // The number of local variables in each level.
	HasReturned     bool     // Used to avoid jumps after return statement and checking for dead code
	currentFuncDef  *FuncDef // The current function being compiled. Nested function definitions is not allowed.
	currentFuncCall string
	ParCount        int // The number of formal parameters to the current function
	LocalRetSize    int // The number of return values from the current function
	CommentLevel    int
	returnLbl       int
	DidReturn       bool
	IsBinary        bool
	BlockLevel      int
}

func NewState(name string) (*State, error) {
	s := new(State)
	var err error
	s.text, err = os.ReadFile(name)
	if err != nil {
		slog.Error("Could not open file %s : %s", name, err.Error())
	}
	return s, err
}
