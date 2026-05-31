package main

import (
	"fmt"
	"log/slog"
	"strconv"
	"unicode/utf8"

	"github.com/jkvatne/jkv/code"
)

type Token int

//goland:noinspection GoSnakeCaseUsage
const (
	TOK_UNDEF Token = iota
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
	TOK_OR_ASGN
	TOK_LOG_OR
	TOK_AND
	TOK_AND_ASGN
	TOK_LOG_AND
	TOK_VAR
	TOK_FUNC
	TOK_CONST
	TOK_IF
	TOK_ELSE
	TOK_FOR
	TOK_RETURN
	TOK_DIV_ASGN
	TOK_MULT_ASGN
	TOK_SEMICOLON
	TOK_COLON
	TOK_STRUCT
	TOK_QMARK
	TOK_DOT
	TOK_AT
	TOK_MAX
	TOK_MIN
	TOK_ABS
	TOK_TYPE
	TOK_TRUE
	TOK_FALSE
	TOK_INVALID
	TOK_INV_MINUS
	TOK_INV_DIV
	TOK_INV_MOD
	TOK_NEW
	TOK_NOT
	TOK_LOOP
	TOK_BREAK
	TOK_CONTINUE
	TOK_FAIL
	TOK_SIZE
)

//goland:noinspection GoSnakeCaseUsage
var TokenNames = [...]string{
	TOK_UNDEF:       "UNDEF",
	TOK_PLUS:        "+",
	TOK_PLUS_PLUS:   "++",
	TOK_PLUS_ASGN:   "+=",
	TOK_MINUS:       "-",
	TOK_MINUS_MINUS: "--",
	TOK_MINUS_ASGN:  "-=",
	TOK_MULT:        "*",
	TOK_DIV:         "/",
	TOK_MOD:         "%",
	TOK_FLOAT:       "float",
	TOK_INT:         "int",
	TOK_STRING:      "string",
	TOK_ID:          "ID",
	TOK_EOF:         "EOF",
	TOK_LBRACE:      "{",
	TOK_RBRACE:      "}",
	TOK_LPAR:        "(",
	TOK_RPAR:        ")",
	TOK_LBRACK:      "[",
	TOK_RBRACK:      "]",
	TOK_COMMA:       ",",
	TOK_ASSIGN:      "=",
	TOK_GE:          ">=",
	TOK_GT:          ">",
	TOK_LE:          "<=",
	TOK_LT:          "<",
	TOK_EQ:          "==",
	TOK_NE:          "!=",
	TOK_OR:          "|",
	TOK_OR_ASGN:     "|=",
	TOK_LOG_OR:      "||",
	TOK_AND:         "&",
	TOK_AND_ASGN:    "&=",
	TOK_LOG_AND:     "&&",
	TOK_VAR:         "VAR",
	TOK_FUNC:        "FUNC",
	TOK_CONST:       "CONST",
	TOK_IF:          "if",
	TOK_ELSE:        "else",
	TOK_FOR:         "for",
	TOK_RETURN:      "return",
	TOK_DIV_ASGN:    "/=",
	TOK_MULT_ASGN:   "*=",
	TOK_SEMICOLON:   ";",
	TOK_COLON:       ":",
	TOK_STRUCT:      "struct",
	TOK_QMARK:       "?",
	TOK_DOT:         ".",
	TOK_AT:          "@",
	TOK_MIN:         "min",
	TOK_MAX:         "max",
	TOK_ABS:         "abs",
	TOK_TYPE:        "type",
	TOK_TRUE:        "true",
	TOK_FALSE:       "false",
	TOK_INVALID:     "INVALID",
	TOK_INV_MINUS:   "INV_MINUS",
	TOK_INV_DIV:     "INV_DIV",
	TOK_INV_MOD:     "INV_MOD",
	TOK_NEW:         "NEW",
	TOK_NOT:         "NOT",
	TOK_LOOP:        "LOOP",
	TOK_BREAK:       "BREAK",
	TOK_CONTINUE:    "CONTINUE",
	TOK_FAIL:        "FAIL",
	TOK_SIZE:        "SIZE",
}

func isNum(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isHex(ch rune) bool {
	return ch >= 'a' && ch <= 'f' || ch >= 'A' && ch <= 'F'
}

func isAlfa(ch rune) bool {
	return ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z'
}

func (t Token) Name() string {
	return TokenNames[t]
}

func (t Token) IsCompare() bool {
	return t == TOK_EQ || t == TOK_NE || t == TOK_GT || t == TOK_LE || t == TOK_LT || t == TOK_GE
}

func (t Token) IsAritmetic() bool {
	return t == TOK_PLUS || t == TOK_MINUS || t == TOK_DIV || t == TOK_MULT ||
		t == TOK_INV_DIV || t == TOK_INV_MINUS || t == TOK_MOD || t == TOK_INV_MOD
}

func (t Token) IsLogic() bool {
	return t == TOK_AND || t == TOK_OR
}

func isAlfaNum(ch rune) bool {
	return isNum(ch) || isAlfa(ch)
}

func nextChar(s *State) (rune, rune) {
	if s.AtLineEnd {
		s.AtLineEnd = false
		code.LineNum++
		s.currentLine = ""
		for i := s.p; i < len(s.text); {
			ch, n := utf8.DecodeRune(s.text[i:])
			if ch == '\n' {
				break
			}
			s.currentLine += string(ch)
			i += n
		}
	}
	if eof(s) {
		return 0, 0
	}
	ch1, n := utf8.DecodeRune(s.text[s.p:])
	if ch1 == '\n' {
		s.AtLineEnd = true
	}
	s.p += n
	if s.p >= len(s.text) {
		s.token = TOK_EOF
		return ch1, 0
	}
	ch2, _ := utf8.DecodeRune(s.text[s.p:])
	return ch1, ch2
}

func eof(s *State) bool {
	return s.p >= len(s.text)
}

func parseNumber(s *State, ch1 rune, ch2 rune) (rune, rune) {
	// Parse number
	var hasDp bool
	var hasExp bool
	var hasExpSgn bool
	hex := false
	if ch1 == '0' && (ch2 == 'x' || ch2 == 'X') {
		hex = true
		ch1, ch2 = nextChar(s)
		ch1, ch2 = nextChar(s)
	}
	num := string(ch1)
	for {
		if isNum(ch2) || hex && isHex(ch2) {
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
	var err error
	if hex {
		s.tokenIntValue, err = strconv.ParseInt(num, 16, 64)
		s.token = TOK_INT
		if err == nil && s.tokenIntValue > 0 {
			s.tokenUintValue = uint64(s.tokenIntValue)
		}
		if err != nil {
			s.tokenUintValue, err = strconv.ParseUint(num, 16, 64)
		}
		if err != nil {
			slog.Error("invalid integer")
			s.token = TOK_EOF
		}
	} else if hasExp || hasDp {
		s.tokenFloatValue, err = strconv.ParseFloat(num, 64)
		if err == nil {
			s.token = TOK_FLOAT
		}
	} else {
		s.tokenIntValue, err = strconv.ParseInt(num, 10, 64)
		if err != nil {
			s.tokenUintValue, err = strconv.ParseUint(num, 16, 64)
		}
		if err != nil {
			slog.Error("invalid integer")
			s.token = TOK_EOF
		}
		s.token = TOK_INT
	}
	if err != nil {
		s.token = TOK_INVALID
	}
	return ch1, ch2
}

func (s *State) found(tokens ...Token) bool {
	for _, t := range tokens {
		if s.token == t {
			nextToken(s)
			return true
		}
	}
	return false
}

func (s *State) foundId() (string, error) {
	if s.token == TOK_ID {
		id := s.tokenString
		nextToken(s)
		return id, nil
	}
	return "", fmt.Errorf("expected ID but found %s", s.tokenString)
}

func (s *State) next() {
	nextToken(s)
}

func nextToken(s *State) {
	s.token = TOK_EOF
	for s.token == TOK_EOF {
		if eof(s) {
			return
		}
		ch1, ch2 := nextChar(s)
		s.tokenString = string(ch1)
		switch {
		case ch1 == '\r':
			s.tokenString = "<cr>"
			continue
		case ch1 == '\n':
			s.tokenString = "<lf>"
			continue
		case ch1 == '\t':
			s.tokenString = "<tab>"
			continue
		case ch1 == '\f':
			continue
		case ch1 == ' ':
			continue
		case ch1 == '!' && ch2 == '=':
			s.tokenString = "!="
			ch1, ch2 = nextChar(s)
			s.token = TOK_NE
		case ch1 == '"':
			s.tokenString = ""
			for {
				ch1, ch2 = nextChar(s)
				if ch1 == '"' || ch1 == 0 {
					break
				}
				s.tokenString += string(ch1)
			}
			s.token = TOK_STRING
		case ch1 == '&' && ch2 == '&':
			ch1, ch2 = nextChar(s)
			s.token = TOK_LOG_AND
			s.tokenString = "&&"
		case ch1 == '&':
			s.token = TOK_AND
			s.tokenString = "&"
		case ch1 == '(':
			s.token = TOK_LPAR
			s.tokenString = "("
		case ch1 == ')':
			s.token = TOK_RPAR
			s.tokenString = ")"
		case ch1 == '*' && ch2 == '=':
			ch1, ch2 = nextChar(s)
			s.tokenString = "*="
			s.token = TOK_MULT_ASGN
		case ch1 == '*':
			s.token = TOK_MULT
			s.tokenString = "*"
		case ch1 == '%':
			s.token = TOK_MOD
			s.tokenString = "%"
		case ch1 == '+' && ch2 == '+':
			ch1, ch2 = nextChar(s)
			s.token = TOK_PLUS_PLUS
			s.tokenString = "++"
		case ch1 == '+' && ch2 == '=':
			ch1, ch2 = nextChar(s)
			s.tokenString = "+="
			s.token = TOK_PLUS_ASGN
		case ch1 == '+':
			s.token = TOK_PLUS
			s.tokenString = "+"
		case ch1 == '!':
			s.token = TOK_NOT
		case ch1 == ',':
			s.token = TOK_COMMA
		case ch1 == '-' && isNum(ch2):
			ch1, ch2 = parseNumber(s, ch1, ch2)
		case ch1 == '-' && ch2 == '-':
			ch1, ch2 = nextChar(s)
			s.tokenString = "--"
			s.token = TOK_MINUS_MINUS
		case ch1 == '-' && ch2 == '=':
			ch1, ch2 = nextChar(s)
			s.token = TOK_MINUS_ASGN
			s.tokenString = "-="
		case ch1 == '-':
			s.token = TOK_MINUS
		case ch1 == '.':
			s.token = TOK_DOT
		case ch1 == '/' && ch2 == '/':
			// Skip comment
			for ch1 != '\n' && !eof(s) {
				ch1, ch2 = nextChar(s)
			}
			continue
		case ch1 == '/' && ch2 == '*':
			// Skip /* */ comment
			s.CommentLevel = 1
			for !eof(s) && s.CommentLevel > 0 {
				ch1, ch2 = nextChar(s)
				if ch1 == '/' && ch2 == '*' {
					s.CommentLevel++
				} else if ch1 == '*' && ch2 == '/' {
					s.CommentLevel--
				}
			}
			ch1, ch2 = nextChar(s)
			continue
		case ch1 == '/' && ch2 == '=':
			ch1, ch2 = nextChar(s)
			s.tokenString = "/="
			s.token = TOK_DIV_ASGN
		case ch1 == '/':
			s.tokenString = "/"
			s.token = TOK_DIV
		case isNum(ch1):
			ch1, ch2 = parseNumber(s, ch1, ch2)
		case ch1 == ':':
			s.tokenString = ":"
			s.token = TOK_COLON
		case ch1 == ';':
			s.tokenString = ";"
			s.token = TOK_SEMICOLON
			continue
		case ch1 == '<' && ch2 == '=':
			s.tokenString = "<="
			ch1, ch2 = nextChar(s)
			s.token = TOK_LE
		case ch1 == '<':
			s.token = TOK_LT
			s.tokenString = "<"
		case ch1 == '=' && ch2 == '=':
			s.tokenString = "=="
			ch1, ch2 = nextChar(s)
			s.token = TOK_EQ
		case ch1 == '=':
			s.tokenString = "="
			s.token = TOK_ASSIGN
		case ch1 == '>' && ch2 == '=':
			ch1, ch2 = nextChar(s)
			s.tokenString = ">="
			s.token = TOK_GE
		case ch1 == '>':
			s.tokenString = ">"
			s.token = TOK_GT
		case ch1 == '?':
			s.tokenString = "?"
			s.token = TOK_QMARK
		case ch1 == '@':
			s.tokenString = "@"
			s.token = TOK_AT
		case isAlfa(ch1):
			value := string(ch1)
			for isAlfaNum(ch2) || ch2 == '_' {
				ch1, ch2 = nextChar(s)
				value += string(ch1)
			}
			s.tokenString = value
			s.token = TOK_ID
			switch value {
			case "func":
				s.token = TOK_FUNC
			case "true":
				s.token = TOK_TRUE
			case "false":
				s.token = TOK_FALSE
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
			case "min":
				s.token = TOK_MIN
			case "max":
				s.token = TOK_MAX
			case "abs":
				s.token = TOK_ABS
			case "type":
				s.token = TOK_TYPE
			case "struct":
				s.token = TOK_STRUCT
			case "new":
				s.token = TOK_NEW
			case "loop":
				s.token = TOK_LOOP
			case "continue":
				s.token = TOK_CONTINUE
			case "break":
				s.token = TOK_BREAK
			case "fail":
				s.token = TOK_FAIL
			}
		case ch1 == '[':
			s.token = TOK_LBRACK
			s.tokenString = "["
		case ch1 == ']':
			s.tokenString = "]"
			s.token = TOK_RBRACK
		case ch1 == '{':
			s.token = TOK_LBRACE
		case ch1 == '|' && ch2 == '|':
			ch1, ch2 = nextChar(s)
			s.token = TOK_LOG_OR
			s.tokenString = "||"
		case ch1 == '|':
			s.token = TOK_OR
			s.tokenString = "|"
		case ch1 == '}':
			s.token = TOK_RBRACE
			s.tokenString = "}"
		default:
			slog.Error("Unknown", "char", fmt.Sprintf("0x%02x", ch1))
		}
		break
	}
}

func Expect(s *State, token Token) error {
	if s.token != token {
		return fmt.Errorf("expected token '%s' but got '%s'", token.Name(), s.tokenString)
	}
	nextToken(s)
	return nil
}
