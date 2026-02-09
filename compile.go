package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type state = struct {
	text        []byte
	p           int
	lineNum     int
	token       int
	tokenString string
}

const (
	TOK_UNDEF = iota
	TOK_PLUS
	TOK_PLUS_PLUS
	TOK_PLUS_ASGN
	TOK_MINUS
	TOK_MINUS_MINUS
	TOK_MINUS_ASGN
	TOK_FLOAT
	TOK_INT
	TOK_STRING
	TOK_NAME
	TOK_EOF
	TOK_LBRACE
	TOK_RBRACE
	TOK_LPAR
	TOK_RPAR
	TOK_LBRACK
	TOK_RBRACK
	TOK_GE
	TOK_GT
	TOK_LE
	TOK_LT
	TOK_EQ
	TOK_NE
)

var usedToken [24]bool

func Compile(workdir string, inputPath string, outputName string) error {
	entries, err := os.ReadDir(inputPath)
	if err != nil {
		return fmt.Errorf("Fatal error " + err.Error())
	}
	s := new(state)
	s.lineNum = 1
	for _, entry := range entries {
		if !entry.IsDir() {
			slog.Info("Compiling", "filename", entry.Name())
			s.text, err = os.ReadFile(filepath.Join(inputPath, entry.Name()))
			if err != nil {
				slog.Error("Could not open file %s : %s", entry.Name(), err.Error())
			}
			CompileFile(s, workdir)
		}
	}
	for i, t := range usedToken {
		if t == false && i > 0 {
			slog.Error("Missing", "token", i)
		}
	}
}

func CompileFile(s *state, workdir string) {
	for s.token != TOK_EOF {
		nextToken(s)
		usedToken[s.token] = true
	}
}

func isNum(ch uint8) bool {
	return ch >= uint8('0') && (ch <= uint8('9'))
}
func isAlfa(ch uint8) bool {
	return ch >= uint8('A') && ch <= 'Z' || ch >= 'a' && ch <= 'z'
}

func isAlfaNum(ch uint8) bool {
	return isNum(ch) || isAlfa(ch)
}

func nextChar(s *state) (uint8, uint8) {
	ch1 := s.text[s.p]
	if ch1 == '\n' {
		s.lineNum++
	}
	s.p++
	if s.p >= len(s.text) {
		s.token = TOK_EOF
		return ch1, 0
	}
	ch2 := s.text[s.p]
	return ch1, ch2
}

func eof(s *state) bool {
	return s.p >= len(s.text)
}

func nextToken(s *state) {
	s.token = TOK_EOF
	for s.token == TOK_EOF {
		if eof(s) {
			return
		}
		ch1, ch2 := nextChar(s)
		s.tokenString = string(ch1)
		switch {
		case ch1 == '/' && ch2 == '/':
			// Skip comment
			slog.Info("Skipping comment")
			for ch1 != '\n' && !eof(s) {
				ch1, ch2 = nextChar(s)
			}
		case ch1 == '/' && ch2 == '*':
			// Skip /* */ comment
			slog.Info("Skipping long comment")
			for (ch1 != '*' || ch2 != '/') && !eof(s) {
				ch1, ch2 = nextChar(s)
			}
		case ch1 == '>' && ch2 == '=':
			ch1, ch2 = nextChar(s)
			s.token = TOK_GE
			break
		case ch1 == '>':
			ch1, ch2 = nextChar(s)
			s.token = TOK_GT
			break
		case ch1 == '<' && ch2 == '=':
			ch1, ch2 = nextChar(s)
			s.token = TOK_LE
			break
		case ch1 == '<':
			ch1, ch2 = nextChar(s)
			s.token = TOK_LT
			break
		case ch1 == '=' && ch2 == '=':
			ch1, ch2 = nextChar(s)
			s.token = TOK_EQ
			break
		case ch1 == '!' && ch2 == '=':
			ch1, ch2 = nextChar(s)
			s.token = TOK_NE
			break
		case ch1 == '(':
			s.token = TOK_LPAR
			break
		case ch1 == ')':
			s.token = TOK_RPAR
			break
		case ch1 == '{':
			s.token = TOK_LBRACE
			break
		case ch1 == '}':
			s.token = TOK_RBRACE
			break
		case ch1 == '[':
			s.token = TOK_LBRACK
			break
		case ch1 == ']':
			s.token = TOK_RBRACK
			break
		case ch1 == '+' && ch2 != '+' && ch2 != '=':
			s.token = TOK_PLUS
			break
		case ch1 == '+' && ch2 == '+':
			ch1, ch2 = nextChar(s)
			s.token = TOK_PLUS_PLUS
			s.tokenString = "++"
			break
		case ch1 == '+' && ch2 == '=':
			ch1, ch2 = nextChar(s)
			s.tokenString = "+="
			s.token = TOK_PLUS_ASGN
			break
		case ch1 == '-' && ch2 != '-' && ch2 != '=':
			s.token = TOK_MINUS
			break
		case ch1 == '-' && ch2 == '-':
			ch1, ch2 = nextChar(s)
			s.tokenString = "--"
			s.token = TOK_MINUS_MINUS
			break
		case ch1 == '-' && ch2 == '=':
			ch1, ch2 = nextChar(s)
			s.token = TOK_MINUS_ASGN
			s.tokenString = "-="
			break
		case ch1 == ' ':
		case ch1 == '\f':
		case ch1 == '\v':
		case ch1 == '\r':
			ch1, ch2 = nextChar(s)
		case ch1 == '\n':
			s.lineNum++
		case isAlfa(ch1):
			value := string(ch1)
			for isAlfaNum(ch2) {
				ch1, ch2 = nextChar(s)
				value += string(ch1)
			}
			s.tokenString = value
			s.token = TOK_NAME
		case isNum(ch1) || ch1 == '-' && isNum(ch2):
			// Parse number
			var hasDp bool
			var hasExp bool
			var hasExpSgn bool
			num := string(ch1)
			for {
				if isNum(ch2) {
					num = num + string(ch2)
				} else if ch2 == '.' && !hasDp {
					num = num + string(ch2)
					hasDp = true
				} else if ch2 == 'e' || ch2 == 'E' {
					num = num + string(ch2)
					hasExp = true
				} else if (ch2 == '+') || (ch2 == '-') && hasExp && !hasExpSgn {
					num = num + string(ch2)
					hasExpSgn = true
				} else {
					break
				}
				ch1, ch2 = nextChar(s)
			}
			s.tokenString = num
			if hasExp || hasDp {
				s.token = TOK_FLOAT
			} else {
				s.token = TOK_INT
			}

		case ch1 == '"':
			s.tokenString = ""
			for {
				ch1, ch2 = nextChar(s)
				if ch1 == '\\' {
					ch1, ch2 = nextChar(s)
					if ch1 == 'n' {
						s.tokenString += string('\n')
						continue
					}
				}
				if ch1 == '"' {
					break
				}
				s.tokenString += string(ch1)
				s.token = TOK_STRING
			}
		}

	}
	slog.Info("Token", "Lno", s.lineNum, "Value", s.token, "String", s.tokenString)
}
