package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/jkvatne/jkv/code"
)

const SepChar = "#"

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
	MangledName        string
	IsPackage          bool
	LoopStack          []LoopInfo
	Cfg                *Config
	GeneratePkg        bool
	Level              int
	packages           map[string]string
	ScanPkg            bool
	buildLib           bool
}

type LoopInfo struct {
	StartLabel int
	ExitLabel  int
	BreakLabel int
}

func (s *State) LoopLevel() int {
	return len(s.LoopStack)
}

func (s *State) PushLoopBlock() {
	s.LoopStack = append(s.LoopStack, LoopInfo{StartLabel: code.NewLabel(), ExitLabel: code.NewLabel(), BreakLabel: code.NewLabel()})
}

func (s *State) PushForBlock() {
	startLabel := code.NewLabel()
	endLabel := code.NewLabel()
	s.LoopStack = append(s.LoopStack, LoopInfo{StartLabel: startLabel, ExitLabel: endLabel, BreakLabel: startLabel})
}

func (s *State) PopLoopBlock() {
	if len(s.LoopStack) == 0 {
		panic("loop stack underflow")
	}
	s.LoopStack = s.LoopStack[:len(s.LoopStack)-1]
}

func (s *State) GetLoopLevel() int {
	if len(s.LoopStack) == 0 {
		return 0
	}
	return len(s.LoopStack)
}

func (s *State) GetLoopStartLabel() int {
	if len(s.LoopStack) == 0 {
		return 0
	}
	return s.LoopStack[len(s.LoopStack)-1].StartLabel
}

func (s *State) GetLoopExitLabel() int {
	if len(s.LoopStack) == 0 {
		return 0
	}
	return s.LoopStack[len(s.LoopStack)-1].ExitLabel
}

func (s *State) GetLoopBreakLabel() int {
	if len(s.LoopStack) == 0 {
		return 0
	}
	return s.LoopStack[len(s.LoopStack)-1].BreakLabel
}

func NewState(cfg *Config, sourceFilePath string, mangledName string, level int) (*State, error) {
	s := new(State)
	var err error
	s.Cfg = cfg
	s.Level = level
	s.MangledName = mangledName
	s.IsPackage = mangledName != "main"
	s.text, err = os.ReadFile(sourceFilePath)
	code.NextLineNum = 1
	code.LineNum = 1
	if err != nil {
		return nil, fmt.Errorf("could not read source file '%s'", sourceFilePath)
	}
	s.SourceFilePath = sourceFilePath
	s.packages = make(map[string]string)
	return s, err
}

func (s *State) Mangle(id string) string {
	if strings.Contains(id, SepChar) {
		return id
	}
	if s.MangledName != "" && s.MangledName != "main" {
		return s.MangledName + SepChar + id
	}
	return id
}
