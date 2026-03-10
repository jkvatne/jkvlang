//go:build nasm

package main

import (
	"fmt"
	"log/slog"
	"strconv"
)

var CommentIndent = 40
var spaces = "                                                                                    "

func emit(s *State, op string, src string, dst string, comment string) {
	var pos, n int
	if s.noCode > 0 {
		return
	}
	pos, _ = s.outputFile.WriteString("   " + op)
	if dst != "" {
		n, _ = s.outputFile.WriteString(" " + dst)
		pos += n
	}
	if src != "" && dst != "" {
		n, _ = s.outputFile.WriteString(",")
		pos += n
	}
	if src != "" {
		n, _ = s.outputFile.WriteString(" " + src)
		pos += n
	}
	if comment != "" {
		_, _ = s.outputFile.WriteString(spaces[0:max(0, CommentIndent-pos)] + "; " + comment)
	}
	_, err := s.outputFile.WriteString("\n")
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

func EmitSp(s *State) {
	_, err := s.outputFile.WriteString("   ; Sp=" + strconv.Itoa(s.localSp) + "\n")
	if err != nil {
		panic(err)
	}
}

func EmitComment(s *State, comment string) {
	_, err := s.outputFile.WriteString("   ; " + comment + "\n")
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
	_, err := s.outputFile.WriteString("   ; Line " + strconv.Itoa(s.lineNum) + "\n")
	if err != nil {
		panic(err)
	}
}

func EmitLabel(s *State, label int) {
	n, err := s.outputFile.WriteString("L" + strconv.Itoa(label) + ":")
	_, err = s.outputFile.WriteString(spaces[0:max(0, CommentIndent-n)] + "; Line " + strconv.Itoa(s.lineNum) + "\n")

	if err != nil {
		panic(err)
	}
}

func EmitJump(s *State, n int, comment string) {
	emit(s, "jmp", "L"+strconv.Itoa(n), "", comment)
}

func EmitCall(s *State, id string, argNo int) {
	emit(s, "call", id, "", "")
	if argNo > 1 {
		emit(s, "add", strconv.Itoa(8*(argNo-1)), "rsp", "Remove arguments")
		s.localSp -= argNo - 1
	}
}

func EmitReturn(s *State) {
	if !s.RaxIsTOS || s.LocalRetSize > 1 {
		for i := range len(s.currentFunc.returnTypes) {
			emit(s, "pop", "rax", "", "Return value "+strconv.Itoa(i))
			s.localSp++
			EmitSp(s)
		}
	}
	// Verify that the stack is now empty, except for the local arguments
	if s.localSp != s.VarCount[0] {
		slog.Warn(s.currentFunc.name+" returns with", "SP", s.VarCount[0])
	}
	EmitSp(s)
	// Function epilogue. Restore frame pointer and exit
	emit(s, "leave", "", "", "")
	emit(s, "ret", "", "", "return from "+s.currentFunc.name)
}

func EmitFunction(s *State, id string) {
	n, err := s.outputFile.WriteString("\n" + id + ":")
	_, err = s.outputFile.WriteString(spaces[0:max(0, CommentIndent-n)] + " ; Line " + strconv.Itoa(s.lineNum) + "\n")
	if err != nil {
		panic(err)
	}
	emit(s, "push", "rbp", "", "")
	emit(s, "mov", "rsp", "rbp", "")
	s.localSp = 0
	EmitSp(s)
	s.RaxIsTOS = false
}

var TokenOp = map[Token]string{
	TOK_AND:        "and",
	TOK_OR:         "or",
	TOK_PLUS:       "add",
	TOK_MINUS:      "sub",
	TOK_MULT:       "mul",
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
		emit(s, "pop", "rbx", "", "Get divisor from stack into RBX")
		emit(s, "idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
	} else if op == TOK_MOD {
		emit(s, "cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit(s, "pop", "rbx", "", "Get divisor from stack into RBX")
		emit(s, "idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
		emit(s, "mov", "rdx", "rax", "Move reminder to AX (top of stack)")
	} else {
		emit(s, "pop", "%rbx", "", "")
		instruction := TokenOp[op]
		if instruction == "" {
			slog.Error("EmitIntegerOp called with invalid token", "op", op.Name())
		}
		emit(s, instruction, "%rbx", "%rax", "")
	}
	s.localSp--
	EmitSp(s)
}

// EmitOpConst will evaluate tos=tos op <constant>
// It uses 64bit integer values on the 64 bit rax register
func EmitOpConst(s *State, op Token, value int64, comment string) {
	if op == TOK_DIV {
		emit(s, "cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit(s, "mov", strconv.FormatInt(value, 10), "rbx", "Get divisor from stack into RBX")
		emit(s, "idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
	} else if op == TOK_MOD {
		emit(s, "cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit(s, "mov", "strconv.FormatInt(value,10)", "rbx", "RBX=constant divisor")
		emit(s, "idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
		emit(s, "mov", "rdx", "rax", "Move reminder to AX (top of stack)")
	} else if op == TOK_ASSIGN {
		emit(s, "mov", "strconv.FormatInt(value,10)", "rbx", "RBX=constant divisor")
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
	if instr == "imul" {
		emit(s, "mov", strconv.FormatInt(value, 10), "rax", "")
		emit(s, "movzx", DataType(size)+BpRel(adr), "rbx", comment)
		emit(s, "imul", "rbx", "", "")
	} else {
		emit(s, instr, strconv.FormatInt(value, 10), DataType(size)+BpRel(adr), comment)
	}
	return nil
}

// Number interface defines the constraint for types that can be used
// with the generic Abs function.
type Number interface {
	int | int8 | int16 | int32 | int64 | float32 | float64
}

// Abs returns the absolute value of a number of any type that satisfies the Number constraint.
func Abs[T Number](x T) T {
	if x < 0 {
		return -x
	}
	return x
}

func BpRel(offset int) string {
	ofs := strconv.Itoa(Abs(offset))
	if offset < 0 {
		ofs = "-" + ofs
	} else {
		ofs = "+" + ofs
	}
	return "[rbp" + ofs + "]"
}

func DataType(size int) string {
	switch Abs(size) {
	case 1:
		return "byte "
	case 2:
		return "word "
	case 4:
		return "dword "
	default:
		return "qword "
	}
}

func AxRegName(size int) string {
	switch Abs(size) {
	case 1:
		return "al"
	case 2:
		return "rax"
	case 4:
		return "eax"
	default:
		return "rax"
	}
}

func MovOpcode(size int) string {
	if size == 1 {
		// Zero extend bytes
		return "movzx"
	}
	if size < 0 {
		// Unsigned - zero extend
		return "movzx"
	}
	// Sign extend
	return "movsx"
}

// EmitStoreConst will store a constant of given size into a local variable at [BP+offset]
func EmitStoreConst(s *State, size int, value int64, offset int, comment string) {
	num := strconv.FormatInt(value, 10)
	emit(s, "mov", num, DataType(size)+BpRel(offset), "")
}

// EmitLoad will push a local variable onto the stack (into AX)
func EmitLoad(s *State, size int, adr int, comment string) {
	if s.RaxIsTOS {
		emit(s, "push", "rax", "", "1 Push TOS")
		s.localSp++
		EmitSp(s)
	}
	s.RaxIsTOS = true
	emit(s, MovOpcode(size), DataType(size)+BpRel(adr), "rax", comment)
}

// EmitStore will save the Top of Stack (AX) into a local variable of given size.
// It will then increment the stack pointer
func EmitStore(s *State, size int, adr int, comment string) {
	emit(s, "mov", AxRegName(size), BpRel(adr), comment)
	emit(s, "add", "8", "rsp", "")
	s.RaxIsTOS = false
	s.localSp--
	EmitSp(s)
}

// EmitAddSp will drop the top "count" 64-bit words.
func EmitAddSp(s *State, count int, comment string) {
	if count != 0 {
		emit(s, "add", strconv.Itoa(-count*8), "rsp", comment)
		s.localSp += count
		EmitSp(s)
	}
	s.RaxIsTOS = false
}

func EmitPushString(s *State, txt string) {
	// emit(s, "   PUSH_STRING", txt)
	s.localSp++
	EmitSp(s)
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
	s.RaxIsTOS = false
}

func EmitAllocLocalVar(s *State, size int, comment string) {
	emit(s, "xor", "rax", "rax", "Clear rax")
	emit(s, "push", "rax", "", "New variable, "+comment)
}

func EmitPushConst(s *State, value int64, comment string) {
	if s.RaxIsTOS {
		emit(s, "push", "rax", "", "2 Push TOS")
		s.localSp++
		EmitSp(s)
	}
	if value == 0 {
		emit(s, "xor", "rax", "rax", comment)
	} else {
		emit(s, "mov", strconv.FormatInt(value, 10), "rax", comment)
	}
	s.RaxIsTOS = true
}
