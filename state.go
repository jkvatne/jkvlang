package main

import (
	"fmt"
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
	ConstValue         float64
	noCode             int      // Used to skip code generation in constant if/else statements.
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
	LoopStack          []int
	Cfg                *Config
}

func (s *State) PushLoopBlock(n int) {
	s.LoopStack = append(s.LoopStack, n)
}

func (s *State) PopLoopBlock() {
	if len(s.LoopStack) == 0 {
		panic("loop stack underflow")
	}
	s.LoopStack = s.LoopStack[:len(s.LoopStack)-1]
}

func (s *State) LoopBlock() int {
	if len(s.LoopStack) == 0 {
		return 0
	}
	return s.LoopStack[len(s.LoopStack)-1]
}

func NewState(cfg *Config, SourceFileName string, PackageName string, isPkgFile bool) (*State, error) {
	s := new(State)
	var err error
	s.Cfg = cfg
	s.PackageName = PackageName
	s.IsPackage = PackageName != "main"
	s.IsPkgFile = isPkgFile
	s.BlockLevel = 0
	s.text, err = os.ReadFile(SourceFileName)
	if err != nil {
		return nil, fmt.Errorf("could not read source file '%s'", SourceFileName)
	}
	s.FileName = SourceFileName
	return s, err
}
