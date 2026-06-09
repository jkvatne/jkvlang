package main

import (
	"fmt"
	"log/slog"
	"math"
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
	TOK_XOR
	TOK_SHL
	TOK_SHR
	TOK_AND_NOT
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
	TOK_XOR:         "XOR",
	TOK_SHL:         "SHL",
	TOK_SHR:         "SHR",
	TOK_AND_NOT:     "AND_NOT",
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
		t == TOK_INV_DIV || t == TOK_INV_MINUS || t == TOK_MOD || t == TOK_INV_MOD ||
		t == TOK_AND || t == TOK_OR || t == TOK_XOR || t == TOK_SHL || t == TOK_SHR || t == TOK_AND_NOT

}

func (t Token) IsLogic() bool {
	return t == TOK_AND || t == TOK_OR || t == TOK_AND_NOT
}

func isAlfaNum(ch rune) bool {
	return isNum(ch) || isAlfa(ch)
}

func nextChar(s *State) {
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
		return
	}
	n := 0
	s.ch1, n = utf8.DecodeRune(s.text[s.p:])
	if s.ch1 == '\n' {
		s.AtLineEnd = true
	}
	s.p += n
	if s.p >= len(s.text) {
		s.token = TOK_EOF
		s.ch2 = 0
		return
	}
	s.ch2, _ = utf8.DecodeRune(s.text[s.p:])
}

func eof(s *State) bool {
	return s.p >= len(s.text)
}

// TypeFromNumber will guess the type based on the value of the number
// Range for 64 bit signed integer = -9223372036854775808 ... 9223372036854775807
// Range for 64 bit unsigned integer = 0 ... 18446744073709551615
func TypeFromNumber(x int64) code.PrimaryType {
	if x >= 0 && x <= 255 {
		return code.TYP_U8
	} else if x >= -32768 && x <= 32767 {
		return code.TYP_I16
	} else if x <= 65536 {
		return code.TYP_U16
	} else if x >= -2147483648 && x <= 2147483647 {
		return code.TYP_I32
	} else if x <= 4294967296 {
		return code.TYP_U32
	}
	// Default to I64
	return code.TYP_I64
}

func parseNumber(s *State) {
	var err error
	var hasDp bool
	var hasExp bool
	var hasExpSgn bool
	var hex bool
	var base = 10
	if s.ch1 == '0' && (s.ch2 == 'x' || s.ch2 == 'X') {
		base = 16
		nextChar(s)
		nextChar(s)
		hex = true
	}
	num := string(s.ch1)
	for {
		if isNum(s.ch2) || hex && isHex(s.ch2) {
			num = num + string(s.ch2)
		} else if s.ch2 == '.' && !hasDp {
			num = num + string(s.ch2)
			hasDp = true
		} else if s.ch2 == 'e' || s.ch2 == 'E' {
			num = num + string(s.ch2)
			nextChar(s)
			hasExp = true
			hasExpSgn = s.ch1 == '-' || s.ch2 == '+'
		} else if ((s.ch2 == '+') || (s.ch2 == '-')) && hasExp && !hasExpSgn {
			num = num + string(s.ch2)
			hasExpSgn = true
		} else {
			break
		}
		nextChar(s)
	}
	s.tokenString = num
	if hasExp || hasDp {
		var f float64
		f, err = strconv.ParseFloat(num, 64)
		if err != nil {
			s.token = TOK_INVALID
		}
		s.token = TOK_FLOAT
		s.ConstValue.Bits = math.Float64bits(f)
		s.ConstValue.Pt = code.TYP_F64
	} else {
		var i int64
		var u uint64
		i, err = strconv.ParseInt(num, base, 64)
		s.ConstValue.Bits = uint64(i)
		s.ConstValue.Pt = TypeFromNumber(i)
		s.token = TOK_INT
		// If conversion failed, try parsing it as an unsigned number
		if err != nil {
			u, err = strconv.ParseUint(num, 16, 64)
			if err != nil {
				fmt.Printf("Error parsing %s as integer: %s\n", num, err)
				s.token = TOK_INVALID
			}
			// We have a constant that is out of range for I64.
			s.ConstValue.Bits = u
			s.ConstValue.Pt = code.TYP_U64
		}
	}
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
		nextChar(s)
		s.tokenString = string(s.ch1)
		switch {
		case s.ch1 == '\r':
			s.tokenString = "<cr>"
			continue
		case s.ch1 == '\n':
			s.tokenString = "<lf>"
			continue
		case s.ch1 == '\t':
			s.tokenString = "<tab>"
			continue
		case s.ch1 == '\f':
			continue
		case s.ch1 == ' ':
			continue
		case s.ch1 == '!' && s.ch2 == '=':
			s.tokenString = "!="
			nextChar(s)
			s.token = TOK_NE
		case s.ch1 == '"':
			s.tokenString = ""
			for {
				nextChar(s)
				if s.ch1 == '"' || s.ch1 == 0 {
					break
				}
				s.tokenString += string(s.ch1)
			}
			s.token = TOK_STRING
		case s.ch1 == '&' && s.ch2 == '&':
			nextChar(s)
			s.token = TOK_LOG_AND
			s.tokenString = "&&"
		case s.ch1 == '&' && s.ch2 == '^':
			nextChar(s)
			s.token = TOK_AND_NOT
			s.tokenString = "&^"
		case s.ch1 == '&':
			s.token = TOK_AND
			s.tokenString = "&"
		case s.ch1 == '(':
			s.token = TOK_LPAR
			s.tokenString = "("
		case s.ch1 == ')':
			s.token = TOK_RPAR
			s.tokenString = ")"
		case s.ch1 == '*' && s.ch2 == '=':
			nextChar(s)
			s.tokenString = "*="
			s.token = TOK_MULT_ASGN
		case s.ch1 == '*':
			s.token = TOK_MULT
			s.tokenString = "*"
		case s.ch1 == '%':
			s.token = TOK_MOD
			s.tokenString = "%"
		case s.ch1 == '+' && s.ch2 == '+':
			nextChar(s)
			s.token = TOK_PLUS_PLUS
			s.tokenString = "++"
		case s.ch1 == '+' && s.ch2 == '=':
			nextChar(s)
			s.tokenString = "+="
			s.token = TOK_PLUS_ASGN
		case s.ch1 == '+':
			s.token = TOK_PLUS
			s.tokenString = "+"
		case s.ch1 == '!':
			s.token = TOK_NOT
		case s.ch1 == ',':
			s.token = TOK_COMMA
		// case s.ch1 == '-' && isNum(s.ch2):
		//	s.ch1, s.ch2 = parseNumber(s, s.ch1, s.ch2)
		case s.ch1 == '-' && s.ch2 == '-':
			nextChar(s)
			s.tokenString = "--"
			s.token = TOK_MINUS_MINUS
		case s.ch1 == '-' && s.ch2 == '=':
			nextChar(s)
			s.token = TOK_MINUS_ASGN
			s.tokenString = "-="
		case s.ch1 == '-':
			s.token = TOK_MINUS
		case s.ch1 == '.':
			s.token = TOK_DOT
		case s.ch1 == '/' && s.ch2 == '/':
			// Skip comment
			for s.ch1 != '\n' && !eof(s) {
				nextChar(s)
			}
			continue
		case s.ch1 == '/' && s.ch2 == '*':
			// Skip /* */ comment
			s.CommentLevel = 1
			for !eof(s) && s.CommentLevel > 0 {
				nextChar(s)
				if s.ch1 == '/' && s.ch2 == '*' {
					s.CommentLevel++
				} else if s.ch1 == '*' && s.ch2 == '/' {
					s.CommentLevel--
				}
			}
			nextChar(s)
			continue
		case s.ch1 == '/' && s.ch2 == '=':
			nextChar(s)
			s.tokenString = "/="
			s.token = TOK_DIV_ASGN
		case s.ch1 == '/':
			s.tokenString = "/"
			s.token = TOK_DIV
		case isNum(s.ch1):
			parseNumber(s)
		case s.ch1 == ':':
			s.tokenString = ":"
			s.token = TOK_COLON
		case s.ch1 == ';':
			s.tokenString = ";"
			s.token = TOK_SEMICOLON
			continue
		case s.ch1 == '<' && s.ch2 == '=':
			s.tokenString = "<="
			nextChar(s)
			s.token = TOK_LE
		case s.ch1 == '<' && s.ch2 == '<':
			s.tokenString = "<<"
			nextChar(s)
			s.token = TOK_SHL
		case s.ch1 == '<':
			s.token = TOK_LT
			s.tokenString = "<"
		case s.ch1 == '=' && s.ch2 == '=':
			s.tokenString = "=="
			nextChar(s)
			s.token = TOK_EQ
		case s.ch1 == '=':
			s.tokenString = "="
			s.token = TOK_ASSIGN
		case s.ch1 == '>' && s.ch2 == '=':
			nextChar(s)
			s.tokenString = ">="
			s.token = TOK_GE
		case s.ch1 == '>' && s.ch2 == '>':
			nextChar(s)
			s.tokenString = ">>"
			s.token = TOK_SHR
		case s.ch1 == '>':
			s.tokenString = ">"
			s.token = TOK_GT
		case s.ch1 == '?':
			s.tokenString = "?"
			s.token = TOK_QMARK
		case s.ch1 == '@':
			s.tokenString = "@"
			s.token = TOK_AT
		case isAlfa(s.ch1):
			value := string(s.ch1)
			for isAlfaNum(s.ch2) || s.ch2 == '_' {
				nextChar(s)
				value += string(s.ch1)
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
		case s.ch1 == '[':
			s.token = TOK_LBRACK
			s.tokenString = "["
		case s.ch1 == ']':
			s.tokenString = "]"
			s.token = TOK_RBRACK
		case s.ch1 == '{':
			s.token = TOK_LBRACE
		case s.ch1 == '|' && s.ch2 == '|':
			nextChar(s)
			s.token = TOK_LOG_OR
			s.tokenString = "||"
		case s.ch1 == '|':
			s.token = TOK_OR
			s.tokenString = "|"
		case s.ch1 == '^':
			s.token = TOK_XOR
			s.tokenString = "^"
		case s.ch1 == '}':
			s.token = TOK_RBRACE
			s.tokenString = "}"
		default:
			slog.Error("Unknown", "char", fmt.Sprintf("0x%02x", s.ch1))
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
