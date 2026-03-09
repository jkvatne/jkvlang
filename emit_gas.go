//go:build gas

package main

import (
	"fmt"
	"log/slog"
	"strconv"
)

var CommentIndent = 40
var spaces = "                                                                                    "

func isRegister(s string) bool {
	if len(s) < 2 || len(s) > 4 {
		return false
	}
	return s[0] >= 'a' && s[0] <= 's'
}

func emit(s *State, op string, src string, dst string, comment string) {
	var pos, n int
	if s.noCode > 0 {
		return
	}
	if isRegister(src) {
		src = "%" + src
	}
	if isRegister(dst) {
		dst = "%" + dst
	}
	if len(src) > 0 && src[0] >= '0' && src[0] <= '9' {
		src = "$" + src
	}
	pos, err := s.outputFile.WriteString("   " + op + " " + src)
	if err != nil {
		panic(err)
	}
	if dst != "" {
		n, err = s.outputFile.WriteString(", " + dst)
		pos += n
		if err != nil {
			panic(err)
		}
	}
	if comment != "" {
		_, err = s.outputFile.WriteString(spaces[0:max(0, CommentIndent-pos)] + "# " + comment)
	}
	_, err = s.outputFile.WriteString("\n")
	if err != nil {
		panic(err)
	}
}

func CloseObjFile(s *State) error {
	return s.outputFile.Close()
}

func EmitError(s *State, e error) {
	_, err := s.outputFile.WriteString(e.Error() + "\n")
	if err != nil {
		panic(err)
	}
}

func EmitComment(s *State, comment string) {
	_, err := s.outputFile.WriteString("# " + comment + "\n")
	if err != nil {
		panic(err)
	}
}

func EmitBlankLine(s *State) {
	_, err := s.outputFile.WriteString("\n")
	if err != nil {
		panic(err)
	}
}

func EmitLineNo(s *State) {
	_, err := s.outputFile.WriteString("   # Line " + strconv.Itoa(s.lineNum) + "\n")
	if err != nil {
		panic(err)
	}
}

func EmitLabel(s *State, label int) {
	n, err := s.outputFile.WriteString("L" + strconv.Itoa(label) + ":")
	_, err = s.outputFile.WriteString(spaces[0:max(0, CommentIndent-n)] + "# Line " + strconv.Itoa(s.lineNum) + "\n")

	if err != nil {
		panic(err)
	}
}

func EmitJump(s *State, n int, comment string) {
	emit(s, "jmp", "L"+strconv.Itoa(n), "", comment)
}

func EmitCall(s *State, id string, argNo int) {
	emit(s, "call", id, "", "")
	if argNo > 0 {
		emit(s, "add", strconv.Itoa(argNo), "%rsp", "Reserve space for local variables")
	}
}

func EmitReturn(s *State) {
	for i := range len(s.currentFunc.returnTypes) {
		emit(s, "popq", "%rax", "", "Return value "+strconv.Itoa(i))
		s.localSp++
	}
	// Verify that the stack is now empty
	if s.localSp != 0 {
		slog.Warn(s.currentFunc.name+" returns with", "SP", s.localSp)
	}
	emit(s, "leave", "", "", "")
	emit(s, "ret", "", "", "return from "+s.currentFunc.name)
}

func EmitFunction(s *State, id string) {
	n, err := s.outputFile.WriteString("\n" + id + ":")
	_, err = s.outputFile.WriteString(spaces[0:max(0, CommentIndent-n)] + " # Line " + strconv.Itoa(s.lineNum) + "\n")
	if err != nil {
		panic(err)
	}
	emit(s, "push", "rbp", "", "")
	emit(s, "movq", "rsp", "rbp", "")
	s.localSp = 0
}

var TokenOp = map[Token]string{
	TOK_AND:        "andq",
	TOK_OR:         "orq",
	TOK_PLUS:       "addq",
	TOK_MINUS:      "subq",
	TOK_MULT:       "mulq",
	TOK_PLUS_ASGN:  "add",
	TOK_MINUS_ASGN: "sub",
	TOK_OR_ASGN:    "or",
	TOK_AND_ASGN:   "and",
	TOK_ASSIGN:     "mov",
	TOK_MULT_ASGN:  "imul",
}

// EmitIntegerOp will generate a stack operation on the top two stack entries, like add or sub
// The stack pointer will be incremented (pop), and the result will now be on top of the stack (AX)
func EmitIntegerOp(s *State, op Token) {
	if op == TOK_DIV {
		emit(s, "xchg", "rax", "rbx", "Exchange RAX and RBX since we calculate NOS/TOS")
		emit(s, "cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit(s, "popq", "rbx", "", "Get divisor from stack into RBX")
		emit(s, "idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
	} else if op == TOK_MOD {
		emit(s, "cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit(s, "popq", "rbx", "", "Get divisor from stack into RBX")
		emit(s, "idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
		emit(s, "mov", "rdx", "rax", "Move reminder to AX (top of stack)")
	} else {
		emit(s, "popq", "%rbx", "", "")
		instruction := TokenOp[op]
		if instruction == "" {
			slog.Error("EmitIntegerOp called with invalid token", "op", op.Name())
		}
		emit(s, instruction, "%rbx", "%rax", "")
	}
	s.localSp--
}

// EmitOpConst will evaluate tos=tos op <constant>
// It uses 64bit integer values on the 64 bit rax register
func EmitOpConst(s *State, op Token, value int64, comment string) {
	if op == TOK_DIV {
		emit(s, "cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit(s, "movq", strconv.FormatInt(value, 10), "rbx", "Get divisor from stack into RBX")
		emit(s, "idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
	} else if op == TOK_MOD {
		emit(s, "cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit(s, "movq", "strconv.FormatInt(value,10)", "rbx", "RBX=constant divisor")
		emit(s, "idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
		emit(s, "mov", "rdx", "rax", "Move reminder to AX (top of stack)")
	} else if op == TOK_ASSIGN {
		emit(s, "movq", "strconv.FormatInt(value,10)", "rbx", "RBX=constant divisor")
	} else {
		instr := TokenOp[op]
		if instr == "" {
			slog.Error("EmitIntegerOp called with invalid token", "op", op.Name())
		}
		emit(s, TokenOp[op], "$"+strconv.FormatInt(value, 10), "rax", comment)
	}
}

// EmitOpAssign will set variable at <adr> to <adr> op <value>
func EmitOpAssign(s *State, op Token, adr int, size int, value int64, comment string) error {
	instr := TokenOp[op]
	if instr == "" {
		return fmt.Errorf("EmitOpAssign called with invalid token %s", op.Name())
	}
	if size == 1 {
		emit(s, instr+"b", strconv.FormatInt(value, 10), strconv.Itoa(adr)+"(%rbp)", comment)
	} else if Abs(size) == 2 {
		emit(s, instr+"w", strconv.FormatInt(value, 10), strconv.Itoa(adr)+"(%rbp)", comment)
	} else if Abs(size) == 4 {
		emit(s, instr+"l", strconv.FormatInt(value, 10), strconv.Itoa(adr)+"(%rbp)", comment)
	} else {
		emit(s, instr+"q", strconv.FormatInt(value, 10), strconv.Itoa(adr)+"(%rbp)", comment)
	}
	return nil
}

func EmitStoreConst(s *State, size int, value int64, address int, comment string) {
	c := strconv.FormatInt(value, 10)
	if size == 1 {
		emit(s, "movb", c, strconv.Itoa(address)+"(%rbp)", comment)
	} else if Abs(size) == 2 {
		emit(s, "movw", c, strconv.Itoa(address)+"(%rbp)", comment)
	} else if Abs(size) == 4 {
		emit(s, "movzl", c, strconv.Itoa(address)+"(%rbp)", comment)
	} else {
		emit(s, "movq", c, strconv.Itoa(address)+"(%rbp)", comment)
	}
}

// EmitLoad will push a local variable onto the stack (into AX)
func EmitLoad(s *State, size int, adr int, comment string) {
	if s.RaxIsTOS {
		emit(s, "pushq", "rax", "", "Push TOS")
	}
	s.RaxIsTOS = true
	if size == 1 {
		emit(s, "movzbq", strconv.Itoa(adr)+"(%rbp)", "rax", comment)
	} else if size == -2 {
		emit(s, "movzwq", strconv.Itoa(adr)+"(%rbp)", "rax", comment)
	} else if size == 2 {
		emit(s, "movswq", strconv.Itoa(adr)+"(%rbp)", "rax", comment)
	} else if size == -4 {
		emit(s, "movzlq", strconv.Itoa(adr)+"(%rbp)", "rax", comment)
	} else if size == 4 {
		emit(s, "movslq", strconv.Itoa(adr)+"(%rbp)", "rax", comment)
	}
	s.localSp++
}

// EmitStore will save the Top of Stack (AX) into a local variable of given size.
// It will then increment the stack pointer
func EmitStore(s *State, size int, address int, comment string) {
	if size == 1 {
		emit(s, "movb", "al", strconv.Itoa(address)+"(%rbp)", comment)
	} else if Abs(size) == 2 {
		emit(s, "movw", "ax", strconv.Itoa(address)+"(%rbp)", comment)
	} else if Abs(size) == 4 {
		emit(s, "movzl", "eax", strconv.Itoa(address)+"(%rbp)", comment)
	} else {
		emit(s, "movq", "rax", strconv.Itoa(address)+"(%rbp)", comment)
	}
	emit(s, "addq", "8", "rsp", "")
	s.RaxIsTOS = false
	s.localSp--
}

// EmitPop will drop the top "count" 64-bit words.
func EmitPop(s *State, count int, comment string) {
	emit(s, "addq", strconv.Itoa(count*8), "rsp", comment)
	s.localSp += count
	s.RaxIsTOS = false
}

func EmitPushString(s *State, txt string) {
	// emit(s, "   PUSH_STRING", txt)
	s.localSp++
}

func EmitAssert(s *State) {
	// emit(s, "   ASSERT", "")
}

// EmitJumpFalse will emit an instruction to jump if top of stack is false.
// Top of stack is typically already in AX
func EmitJumpFalse(s *State, n int, comment string) {
	if !s.RaxIsTOS {
		panic("TOS not in AX")
	}
	emit(s, "or", "rax", "rax", "Set zero flag if rax is zero")
	emit(s, "jz", "L"+strconv.Itoa(n), "", "Jump if zero flag is set")
	// Implicit pop of TOS
	s.localSp--
	s.RaxIsTOS = false
}

func EmitPushConst(s *State, value int64, comment string) {
	if s.RaxIsTOS {
		emit(s, "pushq", "rax", "", "Push TOS")
	}
	if value == 0 {
		emit(s, "xorq", "rax", "rax", comment)
	} else {
		emit(s, "movq", strconv.FormatInt(value, 10), "rax", comment)
	}
	s.RaxIsTOS = true
	s.localSp++
}

func Abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
