//go:build nasm

package main

import (
	"fmt"
	"log/slog"
	"math"
	"os"
	"strconv"
	"strings"
)

/*
; Register Windows ABI           JKV ABI
; 0  rax   Return value          Return value
; 1  rcx   First argument
; 2  rdx   Second argument
; 3  rbx   Preserved             Size of arguments on stack (bytes)
; 4  rsp   Stack pointer
; 5  rbp   Preserved
; 6  rsi   Preserved
; 7  rdi   Preserved             Function called for syscall
; 8  r8    Third argument
; 9  r9    Forth argument
; 10 r10   Used in syscall
; 11 r11   Used in syscall
; 12 r12   Preserved
; 13 r13   Preserved
; 14 r14   Preserved
; 15 r15   Preserved              Error pointer. 0 (nil) means ok.
*/

const (
	CarryFlag    = "0x01" // Bit 0
	ZeroFlag     = "0x40" // Bit 6
	SignFlag     = "0x80"
	OverflowFlag = "0x800"
)

var CommentIndent = 40
var spaces = "                                                                                    "

func Write(s *State, txt string, force bool) (int, error) {
	if s.nesting == 0 || force {
		// Write directly to file
		return s.outputFile.WriteString(txt)
	}
	// When parsing an argument, output text to the last element in the ArgCode slice
	s.ArgCode[len(s.ArgCode)-1] += txt
	return len(txt), nil
}

func emit(s *State, op string, dst string, src string, comment string) {
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
		txt += spaces[0:max(0, CommentIndent-len(txt))] + "; " + comment + " (" + strconv.Itoa(s.localSp) + ")"
	} else {
		txt += spaces[0:max(0, CommentIndent-len(txt))] + ";  (" + strconv.Itoa(s.localSp) + ")"
	}
	txt += "\n"
	_, err := Write(s, txt, false)
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

func EmitComment(s *State, comment string) {
	_, err := Write(s, "   ; "+comment+"\n", false)
	if err != nil {
		panic(err)
	}
}

func EmitBlankLine(s *State) {
	_, err := Write(s, "\n", false)
	if err != nil {
		panic(err)
	}
}

func EmitLineNo(s *State) {
	_, err := Write(s, "\n   ; Line "+strconv.Itoa(s.lineNum)+" "+strings.Trim(s.currentLine, "\r\n")+"\n", false)
	if err != nil {
		panic(err)
	}
}

func EmitNumericLabel(label int) string {
	return ".L" + strconv.Itoa(label)
}

func EmitLabel(s *State, label int, comment string) {
	n, _ := Write(s, ".L"+strconv.Itoa(label)+":", false)
	_, _ = Write(s, spaces[0:max(0, CommentIndent-n)]+"; "+comment+"\n", false)
}

func EmitJump(s *State, n int, comment string) {
	emit(s, "jmp", ".L"+strconv.Itoa(n), "", comment)
}

func EmitCode(s *State, code string) {
	_, _ = Write(s, code, true)
}

func EmitPushTos(s *State, argNo int, funcName string, force bool) {
	if s.RaxIsTOS {
		s.localSp++
		_, _ = Write(s, "   push rax                             ; Push arg "+strconv.Itoa(argNo)+" of "+funcName+"\n", force)
		s.RaxIsTOS = false
	}
}

func EmitCall(s *State, id string, nPar int, builtin bool) {
	if builtin {
		id = "_" + id
	}
	if nPar > 0 && s.RaxIsTOS {
		s.localSp++
		emit(s, "push", "rax", "", "Push TOS from rax to stack")
	}
	// The following is needed only for variadic functions.
	if nPar > 0 {
		emit(s, "mov", "rbx", strconv.Itoa(nPar*8), "")
	} else {
		emit(s, "xor", "rbx", "rbx", "")
	}

	emit(s, "call", id, "", "")
}

func EmitFunction(s *State, id string) {
	_, _ = s.outputFile.WriteString("\n" + id + ":\n")
	if s.localSp != 0 {
		panic("localSp is not 0")
	}
	// Function prologue. Set up new frame pointer.
	if id != "main" {
		emit(s, "push", "rbp", "", "")
	}
	emit(s, "mov", "rbp", "rsp", "")
	s.localSp = 0
	if id == "main" {
		EmitPrintSp(s)
		emit(s, "call", "_sysinit", "", "")
	}
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

func xmm(sp int) string {
	return "xmm" + strconv.Itoa(sp)
}

// EmitFloatOp will generate a stack operation on the top two stack entries, like fadd or fsub
// The stack pointer will be incremented (pop), and the result will now be on top of the stack (xmm0)
// Assumes TOS is in xmm+sp and NOS in xmm+sp-1
func EmitF64Op(s *State, op Token) {
	if s.XmmSp < 2 {
		panic("EmitF64OP requires two values on the floating point stack")
	}
	if op == TOK_PLUS {
		emit(s, "addsd", xmm(s.XmmSp-2), xmm(s.XmmSp-1), "Add the two top xmm stack values")
	} else if op == TOK_MINUS {
		emit(s, "subsd", xmm(s.XmmSp-2), xmm(s.XmmSp-1), "Subtract the two top xmm stack values")
	} else if op == TOK_MULT {
		emit(s, "mulsd", xmm(s.XmmSp-2), xmm(s.XmmSp-1), "Multiply the two top xmm stack values")
	} else if op == TOK_DIV {
		emit(s, "divsd", xmm(s.XmmSp-2), xmm(s.XmmSp-1), "Divide the two top xmm stack values")
	} else if op == TOK_INV_DIV {
		emit(s, "divsd", xmm(s.XmmSp-1), xmm(s.XmmSp-2), "Divide the two top xmm stack values inverted")
		emit(s, "movq", xmm(s.XmmSp-2), xmm(s.XmmSp-1), "")
	} else {
		panic("EmitFloatOp not implemented for " + op.Name())
	}
	s.XmmSp--
	if s.XmmSp < 0 {
		panic("Floating point stack underflow")
	}
}

func EmitPushFloat(s *State, litNo int) {
	emit(s, "movsd", "xmm"+strconv.Itoa(s.XmmSp), "[flt"+strconv.Itoa(litNo)+"]", "Load float value from literal")
	emit(s, "movq", "rax", "xmm"+strconv.Itoa(s.XmmSp), "")
	s.XmmSp++
	s.RaxIsTOS = true
	if s.XmmSp > 8 {
		panic("Floating point stack overflow")
	}
}

func EmitJumpCond(s *State, op Token, unsignedOrFloat bool) error {
	lbl := NewLabel(s)
	emit(s, "mov", "rax", "1", "Default to true")
	if op == TOK_EQ {
		emit(s, "je", EmitNumericLabel(lbl), "", "")
	} else if op == TOK_NE {
		emit(s, "jne", EmitNumericLabel(lbl), "", "")
	} else {
		if unsignedOrFloat {
			if op == TOK_GT {
				emit(s, "ja", EmitNumericLabel(lbl), "", "")
			} else if op == TOK_LE {
				emit(s, "jbe", EmitNumericLabel(lbl), "", "")
			} else if op == TOK_GE {
				emit(s, "jae", EmitNumericLabel(lbl), "", "")
			} else if op == TOK_LT {
				emit(s, "jb", EmitNumericLabel(lbl), "", "")
			} else {
				return fmt.Errorf("EmitJumpCond not implemented")
			}
		} else {
			if op == TOK_GT {
				emit(s, "jg", EmitNumericLabel(lbl), "", "")
			} else if op == TOK_LE {
				emit(s, "jle", EmitNumericLabel(lbl), "", "")
			} else if op == TOK_GE {
				emit(s, "jge", EmitNumericLabel(lbl), "", "")
			} else if op == TOK_LT {
				emit(s, "jl", EmitNumericLabel(lbl), "", "")
			} else {
				return fmt.Errorf("EmitJumpCond not implemented")
			}

		}
	}
	emit(s, "mov", "rax", "0", "Return false if we did not jump")
	EmitLabel(s, lbl, "")
	s.RaxIsTOS = true
	return nil
}

// EmitCompareFloats compares two floats. TOS is in xmm<sp>. NOS is in xmm<sp-1>
func EmitCompareFloats(s *State, op Token) (err error) {
	emit(s, "ucomisd", xmm(s.XmmSp-2), xmm(s.XmmSp-1), "Compare two floats equal")
	err = EmitJumpCond(s, op, true)
	s.XmmSp -= 2
	if s.XmmSp < 0 {
		panic("Floating point stack underflow")
	}
	return err
}

// EmitCompareIntegers will compare the top two stack entries
func EmitCompareIntegers(s *State, op Token, unsigned bool) (err error) {
	s.localSp--
	emit(s, "pop", "rbx", "", "Pop next on stack into RBX")
	emit(s, "cmp", "rax", "rbx", "Compare and set flags")
	return EmitJumpCond(s, op, unsigned)
}

// EmitCompareIntConst will compare top of stack with a constant
func EmitCompareIntConst(s *State, op Token, value int64, unsigned bool) error {
	sval := strconv.FormatInt(value, 10)
	emit(s, "cmp", "rax", sval, "Compare and set flags")
	return EmitJumpCond(s, op, unsigned)
}

// EmitIntegerOp will generate a stack operation on the top two stack entries, like add or sub
// The stack pointer will be incremented (pop), and the result will now be on top of the stack (AX)
func EmitIntegerOp(s *State, op Token) {
	if op == TOK_DIV {
		emit(s, "xchg", "rbx", "rax", "Exchange RAX and RBX since we calculate NOS/TOS")
		emit(s, "cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		s.localSp--
		emit(s, "pop", "rbx", "", "Get divisor from stack into RBX")
		emit(s, "idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
	} else if op == TOK_MOD {
		emit(s, "cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		s.localSp--
		emit(s, "pop", "rbx", "", "Get divisor from stack into RBX")
		emit(s, "idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
		emit(s, "mov", "rax", "rdx", "Move reminder to AX (top of stack)")
	} else {
		s.localSp--
		emit(s, "pop", "rbx", "", "")
		instruction := TokenOp[op]
		if instruction == "" {
			slog.Error("EmitIntegerOp called with invalid token", "op", op.Name())
		}
		if op == TOK_MULT {
			emit(s, "mul", "rbx", "", "")
		} else {
			emit(s, instruction, "rax", "rbx", "")
		}
	}
}

// EmitOpConst will evaluate tos=tos op <constant>
// It uses 64bit integer values on the 64 bit rax register
func EmitOpIntConst(s *State, op Token, value int64, comment string) error {
	sval := strconv.FormatInt(value, 10)
	if op == TOK_DIV {
		emit(s, "cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit(s, "mov", "rbx", sval, "Get divisor from stack into RBX")
		emit(s, "idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
	} else if op == TOK_MOD {
		emit(s, "cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit(s, "mov", "rbx", sval, "RBX=constant divisor")
		emit(s, "idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
		emit(s, "mov", "rax", "rdx", "Move reminder to AX (top of stack)")
	} else if op == TOK_ASSIGN {
		emit(s, "mov", "rax", sval, "")
	} else {
		instr := TokenOp[op]
		if instr == "" {
			return fmt.Errorf("invalid operation %s", op.Name())
		}
		emit(s, instr, "rax", "$"+strconv.FormatInt(value, 10), comment)
	}
	return nil
}

func AxName(size int) string {
	if size == 1 {
		return "al"
	} else if size == 2 {
		return "ax"
	} else if size == 4 {
		return "eax"
	} else if size == 8 {
		return "rax"
	} else {
		panic("AxName with invalid size")
	}
}

// Assign constant float value to variable
func EmitOpAssignFloat(s *State, op Token, adr int, litNo int, comment string) error {
	if op == TOK_ASSIGN {
		emit(s, "mov", "rax", "[flt"+strconv.Itoa(litNo)+"]", "")
		emit(s, "mov", BpRel(adr), "rax", "")
	} else {
		panic("Float assign operation not implemented")
	}
	return nil
}

// EmitOpAssign will set variable at <adr> to <adr> op <value>
func EmitOpAssign(s *State, op Token, adr int, size int, value int64, comment string) error {
	instr := TokenOp[op]
	if instr == "" {
		return fmt.Errorf("EmitOpAssign called with invalid token %s", op.Name())
	}
	if instr == "imul" {
		emit(s, "mov", "rax", strconv.FormatInt(value, 10), "")
		emit(s, "movzx", "rbx", DataType(size)+BpRel(adr), comment)
		emit(s, "imul", "rbx", "", "")
		// Move result to local variable at BpRel(adr)
		emit(s, "mov", DataType(size)+BpRel(adr), AxName(size), "move result of *= to local variable")

	} else {
		emit(s, instr, DataType(size)+BpRel(adr), strconv.FormatInt(value, 10), comment)
	}
	return nil
}

func EmitOpAssignString(s *State, offset int, litno int) error {
	emit(s, "mov", DataType(8)+BpRel(offset), "str"+strconv.Itoa(litno), "")
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
	// We need to use the abs value in order to allways have either + or - in the instruction.
	// Bp can never be zero.
	ofs := strconv.Itoa(Abs(offset))
	if offset < 0 {
		return "[rbp-" + ofs + "]"
	} else if offset > 15 {
		return "[rbp+" + ofs + "]"
	} else {
		panic("Bp relative addressing with zero offset")
	}
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

func MovOpcode(size int) string {
	if size == 8 {
		return "mov"
	}
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
	emit(s, "mov", DataType(size)+BpRel(offset), num, "")
}

func EmitLoadFloat64(s *State, size int, adr int, comment string) {
	emit(s, "movq", xmm(s.XmmSp), BpRel(adr), comment)
	emit(s, "mov", "rax", BpRel(adr), "")
	s.XmmSp++
}

// EmitLoad will push a local variable onto the stack (into AX)
func EmitLoad(s *State, size int, adr int, comment string) {
	if s.RaxIsTOS {
		s.localSp++
		emit(s, "push", "rax", "", "1 Push TOS")
	}
	s.RaxIsTOS = true
	emit(s, MovOpcode(size), "rax", DataType(size)+BpRel(adr), comment)
}

// EmitStore will save the Top of Stack (AX) into a local variable of given size.
// It will then clear RaxIssTos, effectively doing a pop
func EmitStore(s *State, opcode string, size int, adr int, comment string) {
	emit(s, opcode, BpRel(adr), AxName(size), comment)
	s.RaxIsTOS = false
}

func EmitStoreF64(s *State, adr int, comment string) {
	s.XmmSp--
	if s.XmmSp < 0 {
		panic("Floating point stack underflow")
	}
	emit(s, "movq", BpRel(adr), xmm(s.XmmSp), comment)
}

func EmitPushString(s *State, litno int) {
	if s.RaxIsTOS {
		s.localSp++
		emit(s, "push", "rax", "", "EmitPushString() Push TOS")
	}
	emit(s, "mov", "rax", "str"+strconv.Itoa(litno), "Push pointer to literal string")
	s.localSp++
	s.RaxIsTOS = true
}

// EmitJumpFalse will emit an instruction to jump if top of stack is false.
// Top of stack is typically already in AX
func EmitJumpFalse(s *State, n int, comment string) {
	if !s.RaxIsTOS {
		panic("TOS not in AX")
	}
	emit(s, "or", "al", "al", "Set zero flag if rax is zero")
	emit(s, "jz", ".L"+strconv.Itoa(n), "", comment)
	// Implicit pop of TOS
	s.RaxIsTOS = false
}

// EmitJumpTrue will emit an instruction to jump if top of stack is false.
// Top of stack is typically already in AX
func EmitJumpTrue(s *State, n int, comment string) {
	if !s.RaxIsTOS {
		panic("TOS not in AX")
	}
	emit(s, "or", "al", "al", "Set zero flag if rax is zero")
	emit(s, "jnz", ".L"+strconv.Itoa(n), "", "Jump if zero flag is set")
	// Implicit pop of TOS
	s.RaxIsTOS = false
}

// TODO Allow for types larger than 8 bytes. For now, use 8 bytes for all locals.
func EmitAllocLocalVar(s *State, comment string) int {
	s.localSp++
	emit(s, "sub", "rsp", "8", comment)
	return -8 * s.localSp
}

func EmitPushStringLit(s *State, lit int, comment string) {
	if s.RaxIsTOS {
		s.localSp++
		emit(s, "push", "rax", "", "2 Push TOS")
	}
	s.RaxIsTOS = true
	emit(s, "mov", "rax", "str"+strconv.Itoa(lit), comment)
}

func EmitSkipLenCap(s *State) {
	emit(s, "add", "dword [rsp]", "8", "Skip len/cap")
}

func EmitPushConst(s *State, value int64, comment string) {
	if s.RaxIsTOS {
		s.localSp++
		emit(s, "push", "rax", "", "EmitPushConst() Push TOS")
	}
	if value == 0 {
		emit(s, "xor", "rax", "rax", comment)
	} else {
		emit(s, "mov", "rax", strconv.FormatInt(value, 10), comment)
	}
	if len(s.ArgCode) == 0 {
		s.RaxIsTOS = true
	}
}

func EmitPrintHello(s *State, format string) {
	emit(s, "mov", "ecx", "-11", "STD_OUTPUT_HANDLE (.11)")
	emit(s, "call", "GetStdHandle", "", "Handle returned in rax")
	emit(s, "mov", "rcx", "rax", "1.arg - console handle")
	emit(s, "mov", "rdx", "[rel msg]", "2.arg - pointer to message")
	emit(s, "mov", "r8", "20", "3.arg - console handle")
	emit(s, "xor", "r9", "r9", "4.arg - console handle")
	emit(s, "mov", "qword [3sp+32]", "0", "5.arg - console handle")
}

func EmitLitteral(s *State, litName string, litValue string) {
	_, _ = s.outputFile.WriteString(litName + " dq " + strconv.Itoa(len(litValue)) + "\n")
	_, _ = s.outputFile.WriteString("     db `" + litValue + "`, 00h\n")
}

func EmitFloatLitteral(s *State, litName string, litValue float64) {
	value := strconv.FormatFloat(litValue, 'g', 11, 64)
	if !strings.Contains(value, ".") {
		if strings.Contains(value, "e") || strings.Contains(value, "E") {
			value = strings.Replace(value, "e", ".0e", 1)
		} else {
			value = value + ".0"
		}
	}
	_, _ = s.outputFile.WriteString(litName + " dq " + value + "\n")
}

func EmitSection(s *State, section string) {
	section = strings.Trim(section, ".\n ")
	_, err := s.outputFile.WriteString("\nsection ." + section + "\n\n")
	if err != nil {
		panic(err)
	}
}

// EmitConcat will concatenate the two strings at the top of the stack
// First string pointer in [rsp], second string pointer in [rax]
// It uses registers r12, r13, r14, rbx, rcx, rdx, rsi, rdi.
// Calls _alloc to allocate a new string with size for both the input strings + 32 bytes extra.
func EmitConcat(s *State, free1 bool, free2 bool) {
	EmitComment(s, "")
	EmitComment(s, "Start of EmitConcat")
	// Get string 1 sizes/ptr into r14, rbx from [rsp]
	emit(s, "mov", "rdx", "[rsp]", "Get string 1 ptr into rdx")
	emit(s, "mov", "rbx", "rdx", "Get string 1 ptr into rbx")
	emit(s, "mov", "r14d", "dword [rdx]", "String 1 size into r14")
	// Get string 2 sizes/ptr into r12, r13 from rax
	emit(s, "mov", "r12d", "dword [rax]", "Get string 2 size into r12d from TOS (rax)")
	emit(s, "mov", "r13", "rax", "Save string 2 ptr in r13")
	// Calculate new size to allocate, including 32 extra bytes
	emit(s, "mov", "rax", "r12", "Calculate new size to allocate, including 32 extra bytes")
	emit(s, "add", "rax", "r14", "")
	emit(s, "add", "rax", "40", "Add 32+8 to include len/cap")
	// Allocate string
	emit(s, "call", "_alloc", "", "Allocate new string")
	// Save pointer in r9 and rdi for later use
	emit(s, "mov", "rdi", "rax", "Save pointer in rdi for later use")
	s.localSp++
	emit(s, "push", "rax", "", "Save pointer on stack for later use")
	// Save new capacity/length
	emit(s, "mov", "rsi", "r12", "First string length")
	emit(s, "add", "rsi", "r14", "Add second length")
	emit(s, "mov", "rax", "rsi", "New length")
	emit(s, "add", "rsi", "40", "Add 32 for extra bytes and 8 for len/cap")
	emit(s, "shl", "rsi", "32", "Move to cap (msw)")
	emit(s, "or", "rax", "rsi", "")
	emit(s, "mov", "[rdi]", "rax", "Save len/cap")
	emit(s, "add", "rdi", "8", "move pointer to actual string data")
	// Copy string 1
	emit(s, "mov", "rsi", "rbx", "Copy string 1")
	emit(s, "add", "rsi", "8", "")
	emit(s, "mov", "rcx", "r14", "")
	emit(s, "cld", "", "", "")
	emit(s, "rep", "movsb", "", "")
	// Copy string 2
	emit(s, "mov", "rsi", "r13", "Copy string 2")
	emit(s, "add", "rsi", "8", "Skip len/cap")
	emit(s, "mov", "rcx", "r12", "")
	emit(s, "rep", "movsb", "", "")
	if free1 {
		emit(s, "mov", "rax", "rbx", "Free first argument to Concatenate")
		emit(s, "call", "_free_str", "", "")
	}
	if free2 {
		emit(s, "mov", "rax", "r13", "Free second argument to Concatenate")
		emit(s, "call", "_free_str", "", "")
	}
	// Copy the allocated buffer address from r9 to rax. Now rax points to the new string.
	s.localSp--
	emit(s, "pop", "rax", "", "Now AX should point to the string")
	// Remove the top of stack. New TOS is the pointer in rax. Arguments in rbx and r13.
	s.localSp--
	emit(s, "add", "rsp", "8", "Remove the top of stack. New TOS is the pointer in rax")
	EmitComment(s, "End of EmitConcat")
	EmitComment(s, "")
}

func includeFile(s *State, txt string) error {
	// _, _ = Write(s, "%include \""+s.LibPath+txt+"\"\n", false)
	str, err := os.ReadFile(s.LibPath + txt)
	_, _ = Write(s, string(str), false)
	if err != nil {
		return err
	}
	return nil
}

func EmitPrologue(s *State) {
	includeFile(s, "sysinit.asm")
	includeFile(s, "syscall.asm")
	includeFile(s, "assert.asm")
	includeFile(s, "printf.asm")
	includeFile(s, "alloc.asm")
	includeFile(s, "exit.asm")
	// includeFile(s, "winerror.asm")
	EmitSection(s, "text")
	emit(s, "global", "main", "", "")
	EmitBlankLine(s)
	EmitBlankLine(s)
}

func EmitPrintSp(s *State) {
	if *PrintSp {
		emit(s, "call", "_printsp", "", "")
		emit(s, "call", "_fflush", "", "")
	}
}

func Inverse(op Token) Token {
	switch op {
	case TOK_LT:
		return TOK_GT
	case TOK_LE:
		return TOK_GE
	case TOK_GT:
		return TOK_LT
	case TOK_GE:
		return TOK_LE
	case TOK_MINUS:
		return TOK_INV_MINUS
	case TOK_DIV:
		return TOK_INV_DIV
	default:
		return op
	}
}

// EmitConstOpConst will calculate the result of the operation on the two constant values
// and return the constant result.
func EmitConstOpConst(op Token, val1 *ValueDef, val2 *ValueDef) (*ValueDef, error) {
	var result ValueDef
	result.Typ = widest(val1, val2).Typ
	result.HasValue = true
	switch op {
	case TOK_PLUS:
		result.IntValue = val1.IntValue + val2.IntValue
		result.FloatValue = val1.FloatValue + val2.FloatValue
	case TOK_MINUS:
		result.IntValue = val1.IntValue - val2.IntValue
		result.FloatValue = val1.FloatValue - val2.FloatValue
	case TOK_MULT:
		result.IntValue = val1.IntValue * val2.IntValue
		result.FloatValue = val1.FloatValue * val2.FloatValue
	case TOK_DIV:
		result.IntValue = val1.IntValue / val2.IntValue
		result.FloatValue = val1.FloatValue / val2.FloatValue
	case TOK_AND:
		result.IntValue = val1.IntValue & val2.IntValue
	case TOK_OR:
		result.IntValue = val1.IntValue | val2.IntValue
	case TOK_LOG_OR:
		result.Typ = &BoolType
		result.BoolValue = val1.BoolValue || val2.BoolValue
	case TOK_LOG_AND:
		result.Typ = &BoolType
		result.BoolValue = val1.BoolValue && val2.BoolValue
	case TOK_EQ:
		result.Typ = &BoolType
		result.BoolValue = math.Abs(val1.FloatValue-val2.FloatValue)/max(val1.FloatValue, val2.FloatValue, 1e-30) < 1e-7
	case TOK_NE:
		result.Typ = &BoolType
		result.BoolValue = math.Abs(val1.FloatValue-val2.FloatValue)/max(val1.FloatValue, val2.FloatValue, 1e-30) >= 1e-7
	case TOK_LT:
		result.Typ = &BoolType
		result.BoolValue = val1.FloatValue < val2.FloatValue
	case TOK_LE:
		result.Typ = &BoolType
		result.BoolValue = val1.FloatValue <= val2.FloatValue
	case TOK_GT:
		result.Typ = &BoolType
		result.BoolValue = val1.FloatValue > val2.FloatValue
	case TOK_GE:
		result.Typ = &BoolType
		result.BoolValue = val1.FloatValue >= val2.FloatValue
	default:
		// Invalid operand
		return &NoValue, fmt.Errorf("invalid operation: %s", TokenNames[op])
	}
	return &result, nil
}

// EmitCompareStrToLit : The pointer to the first string (val1) is found in AX. Compare it to the known constant in val2
func EmitCompareStrToLit(s *State, op Token, stringValue string, stringLitNo int, isTemp bool) (err error) {
	if !s.RaxIsTOS {
		s.localSp--
		emit(s, "pop", "rax", "", "EmitCompareStrToLit, pop first argument into rax")
	}
	if op == TOK_EQ {
		emit(s, "mov", "r14", "rax", "CompareStrings, save rax to r14")
		emit(s, "mov", "rdi", "rax", "Save rax to rdi")
		// First check lengths
		emit(s, "mov", "eax", "[rax]", "")
		emit(s, "cmp", "eax", strconv.Itoa(len(stringValue)), "Compare string lengths")
		lbl := NewLabel(s)
		emit(s, "mov", "rbx", "0", "Initialize result to false")
		emit(s, "jne", EmitNumericLabel(lbl), "", "If not equal, jump to unequal end")
		emit(s, "mov", "ecx", "eax", "")
		emit(s, "mov", "rsi", "str"+strconv.Itoa(stringLitNo), "Pointer to literal string")
		emit(s, "add", "rsi", "8", "Skip size of literal string")
		emit(s, "add", "rdi", "8", "Skip size of string object")
		emit(s, "cld", "", "", "")
		emit(s, "repe", "cmpsb", "", "")
		emit(s, "jne", EmitNumericLabel(lbl), "", "If not equal, jump to unequal end")
		emit(s, "mov", "rbx", "1", "Strings was equal, set rax=true")
		EmitLabel(s, lbl, "")
		if isTemp {
			emit(s, "mov", "rax", "r14", "")
			emit(s, "call", "_free_str", "", "")
		}
		emit(s, "mov", "rax", "rbx", "Result to TOS (rax)")
		s.RaxIsTOS = true
		return nil
	} else if op == TOK_NE {
		lbl := NewLabel(s)
		emit(s, "mov", "rbx", "1", "Initialize result to true")
		emit(s, "mov", "rdi", "rax", "Save tos")
		emit(s, "mov", "r14", "rax", "Save tos")
		emit(s, "mov", "rsi", "str"+strconv.Itoa(stringLitNo), "Pointer to literal string")
		// First check lengths
		emit(s, "cmp", "word [rax]", strconv.Itoa(len(stringValue)), "Compare string lengths")
		emit(s, "jne", EmitNumericLabel(lbl), "", "If lengths not equal, jump to unequal end")
		emit(s, "mov", "ecx", "[rax]", "Get nos length")
		emit(s, "add", "rsi", "8", "Start of string 1")
		emit(s, "add", "rdi", "8", "Start of string 2")
		emit(s, "cld", "", "", "")
		emit(s, "repe", "cmpsb", "", "")
		emit(s, "jne", EmitNumericLabel(lbl), "", "If not equal, jump to unequal end")
		emit(s, "mov", "rbx", "0", "Strings was equal, set rax=false")
		EmitLabel(s, lbl, "unequal")
		emit(s, "mov", "rax", "rbx", "Result to TOS (rax)")
		s.RaxIsTOS = true
		return nil
	} else {
		return fmt.Errorf("EmitCompareStrings not implemented for " + op.Name())
	}
}

func EmitCompareStringsEq(s *State, temp1 bool, temp2 bool) {
	// Compare two strings, one in rax, and one on top of stack, and drop top of stack
	lbl := NewLabel(s)
	emit(s, "mov", "rdi", "rax", "Save tos")
	emit(s, "mov", "rsi", "[rsp]", "Get nos")
	emit(s, "mov", "rcx", "4", "Compare first 4 bytes")
	emit(s, "cld", "", "", "")
	emit(s, "repe", "cmpsb", "", "")
	s.localSp--
	emit(s, "pop", "rax", "", "Get nos ptr")
	emit(s, "mov", "rbx", "0", "Initialize result to false")
	emit(s, "jne", EmitNumericLabel(lbl), "", "If lengths not equal, jump to unequal end")
	emit(s, "mov", "ecx", "[rax]", "Get nos length")
	emit(s, "add", "rsi", "4", "Start of string 1")
	emit(s, "add", "rdi", "4", "Start of string 2")
	emit(s, "cld", "", "", "")
	emit(s, "repe", "cmpsb", "", "")
	emit(s, "jne", EmitNumericLabel(lbl), "", "If not equal, jump to unequal end")
	emit(s, "mov", "rbx", "1", "Strings was equal, set rax=true")
	EmitLabel(s, lbl, "unequal")
	if temp1 {
		emit(s, "mov", "rax", "rsi", "EmitCompareStringsEq 1")
		emit(s, "call", "_free_str", "", "")
	}
	if temp2 {
		emit(s, "mov", "rax", "rdi", "EmitCompareStringsEq 2")
		emit(s, "call", "_free_str", "", "")
	}
	emit(s, "mov", "rax", "rbx", "Result to TOS (rax)")
}

// Compare two strings, one in rax, and one on top of stack, and drop top of stack
func EmitCompareStringsNe(s *State) {
	lbl := NewLabel(s)
	emit(s, "mov", "rbx", "1", "Initialize result to true")
	emit(s, "mov", "rdi", "rax", "Save tos")
	emit(s, "mov", "rsi", "[rsp]", "Get nos")
	emit(s, "mov", "rcx", "4", "Compare first 4 bytes")
	emit(s, "cld", "", "", "")
	emit(s, "repe", "cmpsb", "", "")
	s.localSp--
	emit(s, "pop", "rax", "", "Get nos ptr")
	emit(s, "jne", EmitNumericLabel(lbl), "", "If lengths not equal, jump to unequal end")
	emit(s, "mov", "ecx", "[rax]", "Get nos length")
	emit(s, "add", "rsi", "4", "Start of string 1")
	emit(s, "add", "rdi", "4", "Start of string 2")
	emit(s, "cld", "", "", "")
	emit(s, "repe", "cmpsb", "", "")
	emit(s, "jne", EmitNumericLabel(lbl), "", "If not equal, jump to unequal end")
	emit(s, "mov", "rbx", "0", "Strings was equal, set rax=false")
	EmitLabel(s, lbl, "unequal")
	emit(s, "mov", "rax", "rbx", "Result to TOS (rax)")
}

// EmitFreeLocalVariables will free an object in a local variable
func EmitFreeLocalVariables(s *State, adr int, pt PrimaryType, comment string) error {
	if pt == TYP_STRING {
		// Decrement allocation count, first load size given in offset +4 (capacity)
		emit(s, "mov", "rax", BpRel(adr), "Load cap")
		emit(s, "mov", "rax", "[rax]", "")
		emit(s, "shr", "rax", "32", "")
		// Skip free if cap=0
		lbl := NewLabel(s)
		emit(s, "or", "rax", "rax", "")
		emit(s, "jz", EmitNumericLabel(lbl), "", "")
		// Load the offset from the variable in local stack frame with offset given by adr
		emit(s, "mov", "rax", BpRel(adr), "")
		emit(s, "call", "_free_str", "", comment)
		EmitLabel(s, lbl, "")
		return nil
	} else {
		return fmt.Errorf("Can not free %s", TokenNames[pt])
	}
}

func EmitPopAx(s *State) {
	s.localSp--
	emit(s, "pop", "rax", "", "")
}

func EmitPop(s *State) {
	s.localSp--
	emit(s, "add", "rsp", "8", "Remove one argument")
}

// EmitAddToSp adjusts stack pointer. Count is in qwords.
// Positive count to reserve space (push)
// Negative count to remove entries (pop)
func EmitAddToSp(s *State, count int, comment string) {
	s.localSp += count
	if count > 0 {
		// Stack grows downward
		emit(s, "sub", "rsp", strconv.Itoa(count*8), comment)
	} else if count < 0 {
		emit(s, "add", "rsp", strconv.Itoa(-count*8), comment)
	}
}

func EmitPushConstString(s *State, litNo int) {
	emit(s, "mov", "rax", "str"+strconv.Itoa(litNo), "")
	s.RaxIsTOS = true
}

func CheckLocalSp(s *State, text string) {
	if s.localSp != 0 {
		panic("Function stack is " + strconv.Itoa(s.localSp) + " at end of " + text)
	}
}
