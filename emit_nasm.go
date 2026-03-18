//go:build nasm

package main

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

const (
	CarryFlag    = "0x01" // Bit 0
	ZeroFlag     = "0x40" // Bit 6
	SignFlag     = "0x80"
	OverflowFlag = "0x800"
)

var CommentIndent = 40
var spaces = "                                                                                    "

func Write(s *State, txt string) (int, error) {
	if s.ArgCount == 0 {
		return s.outputFile.WriteString(txt)
	}
	s.ArgCode[s.ArgCount-1] += txt
	return len(txt), nil
}

func emit(s *State, op string, src string, dst string, comment string) {
	var txt string
	if s.noCode > 0 {
		return
	}
	txt = "   " + op
	if dst != "" {
		txt = txt + " " + dst
	}
	if src != "" && dst != "" {
		txt = txt + ","
	}
	if src != "" {
		txt = txt + " " + src
	}
	if comment != "" {
		txt += spaces[0:max(0, CommentIndent-len(txt))] + "; " + comment
	}
	txt += "\n"
	_, err := Write(s, txt)
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

func EmitTextLabel(s *State, text string) {
	text = strings.Trim(text, ":\n ")
	_, err := s.outputFile.WriteString(text + ":\n")
	if err != nil {
	}
}

func EmitSpComment(s *State) {
	// _, _ := Write(s, "   ; Sp="+strconv.Itoa(s.localSp)+"\n")
}

func EmitComment(s *State, comment string) {
	_, err := Write(s, "; "+comment+"\n")
	if err != nil {
		panic(err)
	}
}

func EmitBlankLine(s *State) {
	_, err := Write(s, "\n")
	if err != nil {
		panic(err)
	}
}

func EmitLineNo(s *State) {
	_, err := Write(s, "   ; Line "+strconv.Itoa(s.lineNum)+" "+strings.Trim(s.currentLine, "\r\n")+"\n")
	if err != nil {
		panic(err)
	}
}

func EmitLabel(s *State, label int) {
	_, _ = s.outputFile.WriteString(".L" + strconv.Itoa(label) + ":\n")
	// _, err = s.outputFile.WriteString(spaces[0:max(0, CommentIndent-n)] + "; Line " + strconv.Itoa(s.lineNum) + "\n")
}

func EmitJump(s *State, n int, comment string) {
	emit(s, "jmp", ".L"+strconv.Itoa(n), "", comment)
}

func EmitCall(s *State, id string, nPar int) {
	s.ArgCount = 0
	for i := len(s.ArgCode) - 1; i >= 0; i-- {
		_, _ = Write(s, s.ArgCode[i])
		if i == 0 {
			break
		}
		emit(s, "push", "rax", "", "Argument "+strconv.Itoa(i+1))
		s.localSp++
	}
	emit(s, "mov", strconv.Itoa((nPar-1)*8), "rbx", "")
	emit(s, "call", id, "", "")
	if nPar > 1 {
		emit(s, "add", strconv.Itoa(8*(nPar-1)), "rsp", "Remove arguments")
		s.localSp -= nPar - 1
	}
}

func EmitReturn(s *State) {
	if !s.RaxIsTOS || s.LocalRetSize > 1 {
		for i := range len(s.currentFunc.returnTypes) {
			emit(s, "pop", "rax", "", "Return value "+strconv.Itoa(i))
			s.localSp++
			EmitSpComment(s)
		}
	}
	// Remove local variables
	if s.localSp > 0 {
		emit(s, "add", strconv.Itoa(s.localSp*8), "rsp", "")
		s.localSp -= s.localSp
	}
	// Function epilogue. Restore frame pointer and exit
	emit(s, "leave", "", "", "")
	emit(s, "ret", "", "", "return from "+s.currentFunc.name)
}

func EmitFunction(s *State, id string) {
	_, _ = s.outputFile.WriteString("\n" + id + ":\n")
	// Function prologue. Set up new frame pointer.
	emit(s, "push", "rbp", "", "")
	emit(s, "mov", "rsp", "rbp", "")
	s.localSp = 0
	s.ArgCount = 0
	EmitSpComment(s)
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

// EmitFloatOp will generate a stack operation on the top two stack entries, like fadd or fsub
// The stack pointer will be incremented (pop), and the result will now be on top of the stack (MMX0)
func EmitFloatOp(s *State, op Token) {

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
	} else if op == TOK_EQ {
		emit(s, "pop", "rbx", "", "Pop next on stack into RBX")
		emit(s, "cmp", "rbx", "rax", "Compare and set flags")
		emit(s, "pushf", "", "", "Push flags")
		emit(s, "and", ZeroFlag, "[rsp]", "Mask zero flag")
	} else if op == TOK_NE {
		emit(s, "pop", "rbx", "", "Pop next on stack into RBX")
		emit(s, "cmp", "rbx", "rax", "Compare and set flags")
		emit(s, "pushf", "", "", "Push flags")
		emit(s, "and", ZeroFlag, "[rsp]", "Mask zero flag")
		emit(s, "xor", ZeroFlag, "[rsp]", "Invert zero flag")
	} else if op == TOK_GT {
		emit(s, "pop", "rbx", "", "Pop next on stack into RBX")
		emit(s, "cmp", "rbx", "rax", "Compare and set flags")
		emit(s, "pushf", "", "", "Push flags")
		emit(s, "and", SignFlag, "[rsp]", "Mask zero flag")
	} else if op == TOK_LE {
		emit(s, "pop", "rbx", "", "Pop next on stack into RBX")
		emit(s, "cmp", "rbx", "rax", "Compare and set flags")
		emit(s, "pushf", "", "", "Push flags")
		emit(s, "and", SignFlag, "[rsp]", "Mask sign flag")
		emit(s, "xor", SignFlag, "[rsp]", "Invert sign flag")
	} else {
		emit(s, "pop", "%rbx", "", "")
		instruction := TokenOp[op]
		if instruction == "" {
			slog.Error("EmitIntegerOp called with invalid token", "op", op.Name())
		}
		emit(s, instruction, "%rbx", "%rax", "")
	}
	s.localSp--
	EmitSpComment(s)
}

// EmitOpConst will evaluate tos=tos op <constant>
// It uses 64bit integer values on the 64 bit rax register
func EmitOpIntConst(s *State, op Token, value int64, comment string) error {
	sval := strconv.FormatInt(value, 10)
	if op == TOK_DIV {
		emit(s, "cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit(s, "mov", sval, "rbx", "Get divisor from stack into RBX")
		emit(s, "idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
	} else if op == TOK_MOD {
		emit(s, "cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit(s, "mov", sval, "rbx", "RBX=constant divisor")
		emit(s, "idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
		emit(s, "mov", "rdx", "rax", "Move reminder to AX (top of stack)")
	} else if op == TOK_ASSIGN {
		emit(s, "mov", sval, "rbx", "")
	} else if op == TOK_EQ {
		emit(s, "cmp", sval, "rax", "Compare and set flags")
		// emit(s, "cmovz", "1", "[rsp]", "Set TOS if zero")
		emit(s, "pushf", "", "", "Push flags")
		emit(s, "and", ZeroFlag, "qword [rsp]", "Mask zero flag")
	} else {
		instr := TokenOp[op]
		if instr == "" {
			return fmt.Errorf("invalid operation %s", op.Name())
		}
		emit(s, instr, "$"+strconv.FormatInt(value, 10), "rax", comment)
	}
	return nil
}

// EmitOpFloatConst will evaluate tos=tos op <constant>
func EmitOpFloatConst(s *State, op Token, value float64, comment string) error {
	return fmt.Errorf("float operation not implemented")
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
		EmitSpComment(s)
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
	EmitSpComment(s)
}

// EmitAddSp will drop the top "count" 64-bit words.
func EmitAddSp(s *State, count int, comment string) {
	if count != 0 {
		emit(s, "add", strconv.Itoa(-count*8), "rsp", comment)
		s.localSp += count
		EmitSpComment(s)
	}
	s.RaxIsTOS = false
}

func EmitPushString(s *State, litno int) {
	if s.RaxIsTOS {
		emit(s, "push", "rax", "", "3 Push TOS")
		s.localSp++
		EmitSpComment(s)
	}
	emit(s, "mov", "str"+strconv.Itoa(litno), "rax", "Push pointer to literal string")
	s.localSp++
	s.RaxIsTOS = true
	EmitSpComment(s)
}

func EmitAssert(s *State) {
	emit(s, "push", strconv.Itoa(s.lineNum), "", "")
	emit(s, "call", "_assert", "", "")
	emit(s, "pop", "", "cx", "")
}

// EmitJumpFalse will emit an instruction to jump if top of stack is false.
// Top of stack is typically already in AX
func EmitJumpFalse(s *State, n int, comment string) {
	if !s.RaxIsTOS {
		panic("TOS not in AX")
	}
	emit(s, "or", "rax", "rax", "Set zero flag if rax is zero")
	emit(s, "jz", ".L"+strconv.Itoa(n), "", "Jump if zero flag is set")
	// Implicit pop of TOS
	s.RaxIsTOS = false
}

func EmitAllocLocalVar(s *State, size int, comment string) {
	emit(s, "xor", "rdx", "rdx", "")
	emit(s, "push", "rdx", "", "New variable, "+comment)
}

func EmitPushStringLit(s *State, lit int) {
	if s.RaxIsTOS {
		emit(s, "push", "rax", "", "2 Push TOS")
		s.localSp++
		EmitSpComment(s)
	}
	emit(s, "mov", "str"+strconv.Itoa(lit), "rax", "")
}

func EmitPushConst(s *State, value int64, comment string) {
	if s.RaxIsTOS {
		emit(s, "push", "rax", "", "3 Push TOS")
		s.localSp++
		EmitSpComment(s)
	}
	if value == 0 {
		emit(s, "xor", "rax", "rax", comment)
	} else {
		emit(s, "mov", strconv.FormatInt(value, 10), "rax", comment)
	}
	if s.ArgCount == 0 {
		s.RaxIsTOS = true
	}
}

func EmitPrologue(s *State) {
	EmitSection(s, "text")
	emit(s, "global", "_start", "", "")
	emit(s, "extern", "assert", "", "")
	emit(s, "extern", "syscall", "", "")
	emit(s, "extern", "exit", "", "")
	emit(s, "extern", "malloc", "", "")
	emit(s, "extern", "mfree", "", "")
	emit(s, "extern", "sysinit", "", "")
	emit(s, "extern", "print", "", "")
	EmitBlankLine(s)
	EmitTextLabel(s, "_start")
	emit(s, "call", "sysinit", "", "")
	emit(s, "call", "main", "", "Call the main procedure")
	emit(s, "xor", "eax", "eax", "Error code = 0")
	emit(s, "call", "exit", "", "")
	EmitBlankLine(s)
}

func EmitPrintHello(s *State, format string) {
	emit(s, "mov", "-11", "ecx", "STD_OUTPUT_HANDLE (.11)")
	emit(s, "call", "GetStdHandle", "", "Handle returned in rax")
	emit(s, "mov", "rax", "rcx", "1.arg - console handle")
	emit(s, "mov", "[rel msg]", "rdx", "2.arg - pointer to message")
	emit(s, "mov", "20", "r8", "3.arg - console handle")
	emit(s, "xor", "r9", "r9", "4.arg - console handle")
	emit(s, "mov", "0", "qword [3sp+32]", "5.arg - console handle")
}

func EmitLitteral(s *State, litName string, litValue string) {
	_, _ = s.outputFile.WriteString(litName + " db \"" + litValue + "\", 0Ah, 0Dh, 00h\n")
}

func EmitSection(s *State, section string) {
	section = strings.Trim(section, ".\n ")
	_, err := s.outputFile.WriteString("\nsection ." + section + "\n\n")
	if err != nil {
		panic(err)
	}
}
