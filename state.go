package main

import (
	"os"
)

type State struct {
	ch1                rune
	ch2                rune
	text               []byte // The whole current file being compiled
	p                  int    // Points to the current character in text
	currentLine        string // The content of the current source code text line
	token              Token  // The current token as a number
	tokenString        string // The current token as a string
	ConstValue         ConstValue
	noCode             int      // Used to skip code generation in constant if/else statements.
	LocalVarCount      int      // The number of local variables in each level.
	HasReturned        bool     // Used to avoid jumps after return statement and checking for dead code
	currentFuncDef     *FuncDef // The current function being compiled. Nested function definitions is not allowed.
	currentFuncCall    string
	ParCount           int // The number of formal parameters to the current function
	LocalRetSize       int // The number of return values from the current function
	CommentLevel       int
	returnLbl          int
	BlockLevel         int
	FileName           string
	ParsingReturnValue bool
	Cleanup            string
	LoopLevel          int
	PackageName        string
	IsPackage          bool
	IsPkgFile          bool
}

func NewState(SourceFile string, PackageName string, isPkgFile bool) (*State, error) {
	s := new(State)
	var err error
	s.PackageName = PackageName
	s.IsPackage = PackageName != "main"
	s.IsPkgFile = isPkgFile
	s.text, err = os.ReadFile(SourceFile)
	s.FileName = SourceFile
	return s, err
}
