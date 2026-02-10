package main

import (
	"fmt"
	"log/slog"
)

type state = struct {
	text        []byte
	p           int
	lineNum     int
	token       int
	tokenString string
	labelNo     int
}

const (
	TOK_UNDEF = iota
	TOK_PLUS
	TOK_PLUS_PLUS
	TOK_PLUS_ASGN
	TOK_MINUS
	TOK_MINUS_MINUS
	TOK_MINUS_ASGN
	TOK_MULT
	TOK_DIV
	TOK_MOD
	TOK_FLOAT
	TOK_INT
	TOK_STRING
	TOK_ID
	TOK_EOF
	TOK_LBRACE
	TOK_RBRACE
	TOK_LPAR
	TOK_RPAR
	TOK_LBRACK
	TOK_RBRACK
	TOK_COMMA
	TOK_ASSIGN
	TOK_GE
	TOK_GT
	TOK_LE
	TOK_LT
	TOK_EQ
	TOK_NE
	TOK_OR
	TOK_LOG_OR
	TOK_AND
	TOK_LOG_AND
	TOK_VAR
	TOK_FUNC
	TOK_CONST
	TOK_IF
	TOK_ELSE
	TOK_FOR
	TOK_RETURN
	TOK_SIZE
)

var usedToken [TOK_SIZE]bool
var TokenNames = [...]string{
	TOK_UNDEF:       "UNDEF",
	TOK_PLUS:        "PLUS",
	TOK_PLUS_PLUS:   "PLUS_PLUS",
	TOK_PLUS_ASGN:   "PLUS_ASGN",
	TOK_MINUS:       "MINUS",
	TOK_MINUS_MINUS: "MINUS_MINUS",
	TOK_MINUS_ASGN:  "MINUS_ASGN",
	TOK_MULT:        "MULT",
	TOK_DIV:         "DIV",
	TOK_MOD:         "MOD",
	TOK_FLOAT:       "FLOAT",
	TOK_INT:         "INT",
	TOK_STRING:      "STRING",
	TOK_ID:          "ID",
	TOK_EOF:         "EOF",
	TOK_LBRACE:      "LBRACE",
	TOK_RBRACE:      "RBRACE",
	TOK_LPAR:        "LPAR",
	TOK_RPAR:        "RPAR",
	TOK_LBRACK:      "LBRACK",
	TOK_RBRACK:      "RBRACK",
	TOK_COMMA:       "COMMA",
	TOK_ASSIGN:      "ASSIGN",
	TOK_GE:          "GE",
	TOK_GT:          "GT",
	TOK_LE:          "LE",
	TOK_LT:          "LT",
	TOK_EQ:          "EQ",
	TOK_NE:          "NE",
	TOK_OR:          "OR",
	TOK_LOG_OR:      "LOG_OR",
	TOK_AND:         "AND",
	TOK_LOG_AND:     "LOG_AND",
	TOK_VAR:         "VAR",
	TOK_FUNC:        "FUNC",
	TOK_CONST:       "CONST",
	TOK_IF:          "IF",
	TOK_ELSE:        "ELSE",
	TOK_FOR:         "FOR",
	TOK_RETURN:      "RETURN",
	TOK_SIZE:        "SIZE",
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
	s1 := string(ch1)
	s2 := string(ch2)
	slog.Debug("Got", "s1:", s1, "s2:", s2)

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
		case ch1 == ' ':

		case ch1 == ',':
			s.token = TOK_COMMA
			break
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
			ch1, ch2 = nextChar(s)
		case ch1 == '>' && ch2 == '=':
			ch1, ch2 = nextChar(s)
			s.token = TOK_GE
			break
		case ch1 == '>':
			s.token = TOK_GT
			break
		case ch1 == '<' && ch2 == '=':
			ch1, ch2 = nextChar(s)
			s.token = TOK_LE
			break
		case ch1 == '<':
			s.token = TOK_LT
			break
		case ch1 == '=' && ch2 == '=':
			ch1, ch2 = nextChar(s)
			s.token = TOK_EQ
			break
		case ch1 == '=':
			s.token = TOK_ASSIGN
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
		case ch1 == '&' && ch2 == '&':
			ch1, ch2 = nextChar(s)
			s.token = TOK_LOG_AND
			s.tokenString = "&&"
			break
		case ch1 == '&':
			s.token = TOK_AND
			s.tokenString = "&"
			break
		case ch1 == '|' && ch2 == '|':
			ch1, ch2 = nextChar(s)
			s.token = TOK_LOG_OR
			s.tokenString = "||"
			break
		case ch1 == '|':
			s.token = TOK_OR
			s.tokenString = "|"
			break
		case ch1 == '*':
			s.token = TOK_MULT
			s.tokenString = "*"
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
			s.token = TOK_ID
			switch value {
			case "func":
				s.token = TOK_FUNC
			case "if":
				s.token = TOK_IF
			case "for":
				s.token = TOK_FOR
			case "else":
				s.token = TOK_ELSE
			case "var":
				s.token = TOK_VAR
			case "const":
				s.token = TOK_CONST
			case "return":
				s.token = TOK_RETURN
			}
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
					ch1, ch2 = nextChar(s)
					hasExp = true
					hasExpSgn = ch1 == '-' || ch2 == '+'
				} else if ((ch2 == '+') || (ch2 == '-')) && hasExp && !hasExpSgn {
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
		default:
			slog.Error("Unknown", "char", fmt.Sprintf("0x%02x", ch1))
		}
	}
}
