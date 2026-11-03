package main

import (
	"fmt"
	"os"
	"strings"
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
	HasReturned        bool     // Used to avoid jumps after return statement and checking for dead code
	currentFuncDef     *FuncDef // The current function being compiled. Nested function definitions is not allowed.
	currentFuncCall    string
	ParCount           int // The number of formal parameters to the current function
	LocalRetSize       int // The number of return values from the current function
	CommentLevel       int
	returnLbl          int
	BlockLevel         int
	SourceFilePath     string
	ParsingReturnValue bool
	Cleanup            string
	LoopLevel          int
	MangledName        string
	IsPackage          bool
	LoopStack          []int
	Cfg                *Config
	GeneratePkg        bool
	Level              int
	packages           map[string]string
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

func NewState(cfg *Config, sourceFilePath string, mangledName string, level int) (*State, error) {
	s := new(State)
	var err error
	s.Cfg = cfg
	s.Level = level
	s.MangledName = mangledName
	s.IsPackage = mangledName != "main"
	s.text, err = os.ReadFile(sourceFilePath)
	if err != nil {
		return nil, fmt.Errorf("could not read source file '%s'", sourceFilePath)
	}
	s.SourceFilePath = sourceFilePath
	s.packages = make(map[string]string)
	return s, err
}

func (s *State) Mangle(id string) string {
	if strings.Contains(id, "@") {
		return id
	}
	if s.MangledName != "" && s.MangledName != "main" {
		return s.MangledName + "@" + id
	}
	return id
}
