package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type State struct {
	LibPath         string
	unitName        string   // The name of the current unit without extension
	outputFile      *os.File // File where the assembly is put
	text            []byte   // The whole current file being compiled
	p               int      // Points to the current character in text
	lineNum         int      // The current line number in text
	currentLine     string   // The content of the current source code text line
	AtLineEnd       bool     // Flag used for lineNum calculation
	token           Token    // The current token as a number
	tokenString     string   // The current token as a string
	tokenFloatValue float64  // The current token as a float (if it is a number)
	labelNo         int      // The current label number in this file, to make all local labels unique.
	noCode          int      // Used to skip code generation in constant if/else statements.
	VarCount        int      // The number of local variables in each level.
	hasReturned     bool     // Used to avoid jumps after return statement and checking for dead code
	localSp         int      // Tracks the stack pointer. Used to pop arguments and verify correct code
	RaxIsTOS        bool     // False if there is no value in rax. Normally rax is top of stack.
	currentFuncDef  *FuncDef // The current function being compiled. Nested function definitions is not allowed.
	currentFuncCall string
	ParCount        int      // The number of formal parameters to the current function
	LocalRetSize    int      // The number of return values from the current function
	ArgCode         []string // Temporary storage of assembly code. needed because we evaluate arguments in reverse order
	CleanupCode     []string
	CommentLevel    int
	XmmSp           int // Stack pointer into SSE registers
	returnLbl       int
	DidReturn       bool
	Returning       bool
}

func NewState(name string, workdir string) (*State, error) {
	s := new(State)
	s.LibPath, _ = filepath.Abs("../lib/")
	s.LibPath += string(os.PathSeparator)
	s.ArgCode = make([]string, 0, 64)
	s.CleanupCode = make([]string, 0, 64)
	s.lineNum = 1
	var err error
	s.text, err = os.ReadFile(name)
	if err != nil {
		slog.Error("Could not open file %s : %s", name, err.Error())
	}
	s.unitName = strings.TrimSuffix(filepath.Base(name), ".jkv")
	objectFile := filepath.Join(workdir, s.unitName+".asm")
	s.outputFile, err = os.Create(objectFile)
	return s, err
}

func PushArgCode(s *State) {
	s.ArgCode = append(s.ArgCode, "")
}

func PushCleanupCode(s *State) {
	s.CleanupCode = append(s.CleanupCode, "")
}

func SetCleanupCode(s *State, txt string) {
	if txt != "" {
		s.CleanupCode[len(s.CleanupCode)-1] = txt
	}
}

func OutputCleanupCode(s *State, n int) {
	na := len(s.ArgCode) - 1
	nc := len(s.CleanupCode) - 1
	for ; n > 0; n-- {
		if s.CleanupCode[nc] != "" {
			s.ArgCode[na] = s.ArgCode[na] + s.CleanupCode[nc]
		}
		nc--
	}
	if nc < n {
		panic("CleanupCode error")
	}
	s.CleanupCode = s.CleanupCode[0 : len(s.CleanupCode)-n]
}

func ConsArgCode(s *State, count int, reverse bool) {
	if count == 0 {
		return
	}
	txt := ""
	startArgNo := len(s.ArgCode) - count
	if startArgNo < 0 {
		panic("ArgCode error")
	}
	if reverse {
		for i := len(s.ArgCode) - 1; i >= startArgNo; i-- {
			txt += s.ArgCode[i]
		}
	} else {
		for i := startArgNo; i < len(s.ArgCode); i++ {
			txt += s.ArgCode[i]
		}
	}
	if len(s.ArgCode) > startArgNo {
		s.ArgCode[startArgNo] = txt
	}
	if len(s.ArgCode) > startArgNo {
		s.ArgCode = s.ArgCode[0 : startArgNo+1]
	}
}

func OutputArgCode(s *State) {
	if len(s.ArgCode) > 1 {
		panic("Line " + strconv.Itoa(s.lineNum) + ": OutputArgCode should have only one entry in ArgCode")
	}
	if len(s.ArgCode) == 0 {
		return
	}
	_, _ = Write(s, s.ArgCode[0], true)
	s.ArgCode = nil
}
