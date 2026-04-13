//go:build nasm

package main

import (
	"fmt"
	"log/slog"
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
	if s.ArgCount == 0 || force {
		return s.outputFile.WriteString(txt)
	}
	s.ArgCode[s.ArgCount-1] += txt
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
		txt += spaces[0:max(0, CommentIndent-len(txt))] + "; " + comment
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
	_, err := Write(s, "; "+comment+"\n", false)
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
	_, err := Write(s, "   ; Line "+strconv.Itoa(s.lineNum)+" "+strings.Trim(s.currentLine, "\r\n")+"\n", false)
	if err != nil {
		panic(err)
	}
}

func LabelName(label int) string {
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
	_, _ = Write(s, "   push rax                             ; Push TOS "+strconv.Itoa(argNo)+" of "+funcName+"\n", force)
}

func EmitCall(s *State, id string, nPar int) {
	emit(s, "mov", "rbx", strconv.Itoa((nPar-1)*8), "")
	emit(s, "call", id, "", "")
	if nPar > 1 {
		emit(s, "add", "rsp", strconv.Itoa(8*(nPar-1)), "Remove arguments")
		s.localSp -= nPar - 1
	}
}

func EmitReturn(s *State) {
	if !s.RaxIsTOS || s.LocalRetSize > 1 {
		for i := range len(s.currentFunc.returnTypes) {
			emit(s, "pop", "rax", "", "Return value "+strconv.Itoa(i))
			s.localSp--
		}
	}
	// Remove local variables
	if s.localSp > 0 {
		emit(s, "add", "rsp", strconv.Itoa(s.localSp*8), "")
		s.localSp -= s.localSp
	}
	// Return exit code from main
	if s.currentFunc.name == "main" {
		EmitPrintSp(s)
		emit(s, "mov", "rax", "r15", "Get error code")
		emit(s, "call", "_exit", "", "")
	}
	// Verify localsp is zero
	if s.localSp != 0 {
		panic("s.localSp != 0")
	}
	// Function epilogue. Restore frame pointer and exit
	emit(s, "leave", "", "", "")
	emit(s, "ret", "", "", "return from "+s.currentFunc.name)
}

func EmitFunction(s *State, id string) {
	_, _ = s.outputFile.WriteString("\n" + id + ":\n")
	if s.localSp != 0 {
		fmt.Printf("Local Sp: %d, should be 0\n", s.localSp)
		// panic("localSp is not 0")
	}
	// Function prologue. Set up new frame pointer.
	emit(s, "push", "rbp", "", "")
	s.localSp = 1
	emit(s, "mov", "rbp", "rsp", "")
	emit(s, "push", "rax", "", "Save first argument in rax")
	s.localSp++
	if id == "main" {
		EmitPrintSp(s)
		emit(s, "call", "_sysinit", "", "")
		EmitPrintSp(s)
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
	} else {
		panic("EmitFloatOp not implemented")
	}
	s.XmmSp--
}

func EmitPushFloat(s *State, litNo int) {
	emit(s, "movsd", "xmm"+strconv.Itoa(s.XmmSp), "[flt"+strconv.Itoa(litNo)+"]", "Load float value frm string variable")
	emit(s, "movq", "rax", "xmm"+strconv.Itoa(s.XmmSp), "")
	s.XmmSp++
	if s.XmmSp > 8 {
		panic("Floating point stack overflow")
	}
}

func EmitCompareFloats(s *State, op Token) {
	panic("EmitCompareFloats not implemented")
}

func EmitSetCompareResult(s *State, op Token) error {
	if op == TOK_EQ {
		emit(s, "sete", "al", "", "")
	} else if op == TOK_GT {
		emit(s, "setg", "al", "", "")
	} else if op == TOK_NE {
		emit(s, "setne", "al", "", "")
	} else if op == TOK_GE {
		emit(s, "setge", "al", "", "")
	} else if op == TOK_LT {
		emit(s, "setl", "al", "", "")
	} else if op == TOK_LE {
		emit(s, "setle", "al", "", "")
	} else {
		return fmt.Errorf("EmitCompareResult: invalid token %v", op)
	}
	return nil
}

// EmitCompareIntegers will compare the top two stack entries
func EmitCompareIntegers(s *State, op Token) error {
	emit(s, "pop", "rbx", "", "Pop next on stack into RBX")
	s.localSp--
	emit(s, "cmp", "rax", "rbx", "Compare and set flags")
	return EmitSetCompareResult(s, op)
}

// EmitCompareIntConst will compare top of stack with a constant
func EmitCompareIntConst(s *State, op Token, value int64) error {
	sval := strconv.FormatInt(value, 10)
	emit(s, "cmp", "rax", sval, "Compare and set flags")
	return EmitSetCompareResult(s, op)
}

// EmitIntegerOp will generate a stack operation on the top two stack entries, like add or sub
// The stack pointer will be incremented (pop), and the result will now be on top of the stack (AX)
func EmitIntegerOp(s *State, op Token) {
	if op == TOK_DIV {
		emit(s, "xchg", "rbx", "rax", "Exchange RAX and RBX since we calculate NOS/TOS")
		emit(s, "cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit(s, "pop", "rbx", "", "Get divisor from stack into RBX")
		s.localSp--
		emit(s, "idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
	} else if op == TOK_MOD {
		emit(s, "cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit(s, "pop", "rbx", "", "Get divisor from stack into RBX")
		s.localSp--
		emit(s, "idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
		emit(s, "mov", "rax", "rdx", "Move reminder to AX (top of stack)")
	} else {
		emit(s, "pop", "rbx", "", "")
		s.localSp--
		instruction := TokenOp[op]
		if instruction == "" {
			slog.Error("EmitIntegerOp called with invalid token", "op", op.Name())
		}
		emit(s, instruction, "rax", "rbx", "")
	}
	s.localSp--
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

// EmitOpFloatConst will evaluate tos=tos op <constant>
func EmitOpFloatConst(s *State, op Token, value float64, comment string) error {
	return fmt.Errorf("float operation not implemented")
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
		emit(s, "mov", "[rbp+"+strconv.Itoa(adr)+"]", "rax", "")
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
	} else if offset > 0 {
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
	emit(s, "movq", xmm(s.XmmSp), "[rbp+"+strconv.Itoa(adr)+"]", "")
	s.XmmSp++
}

// EmitLoad will push a local variable onto the stack (into AX)
func EmitLoad(s *State, size int, adr int, comment string) {
	if s.RaxIsTOS {
		emit(s, "push", "rax", "", "1 Push TOS")
		s.localSp++
	}
	s.RaxIsTOS = true
	emit(s, MovOpcode(size), "rax", DataType(size)+BpRel(adr), comment)
}

// EmitStore will save the Top of Stack (AX) into a local variable of given size.
// It will then clear RaxIssTos, effectively doing a pop
func EmitStore(s *State, opcode string, size int, adr int, comment string) {
	emit(s, opcode, BpRel(adr), AxName(size), comment)
	s.RaxIsTOS = false
	s.localSp--
}

func EmitStoreF64(s *State, adr int, comment string) {
	s.XmmSp--
	emit(s, "movq", BpRel(adr), xmm(s.XmmSp), comment)
}

// EmitAddSp will drop the top "count" 64-bit words.
func EmitAddSp(s *State, count int, comment string) {
	if count != 0 {
		emit(s, "sub", "rsp", strconv.Itoa(count*8), comment)
		s.localSp += count
	}
	s.RaxIsTOS = false
}

func EmitPushString(s *State, litno int) {
	if s.RaxIsTOS {
		emit(s, "push", "rax", "", "EmitPushString() Push TOS")
		s.localSp++
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
func EmitAllocLocalVar(s *State, size int, comment string) {
	emit(s, "xor", "rdx", "rdx", "")
	emit(s, "push", "rdx", "", "New variable, "+comment)
	s.localSp++
}

func EmitPushStringLit(s *State, lit int) {
	if s.RaxIsTOS {
		emit(s, "push", "rax", "", "2 Push TOS")
		s.localSp++
	}
	emit(s, "mov", "rax", "str"+strconv.Itoa(lit), "")
}

func EmitSkipLenCap(s *State) {
	emit(s, "add", "rax", "8", "Load string value frm string variable")
}

func EmitPushConst(s *State, value int64, comment string) {
	if s.RaxIsTOS {
		emit(s, "push", "rax", "", "EmitPushConst() Push TOS")
		s.localSp++
	}
	if value == 0 {
		emit(s, "xor", "rax", "rax", comment)
	} else {
		emit(s, "mov", "rax", strconv.FormatInt(value, 10), comment)
	}
	if s.ArgCount == 0 {
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
	_, _ = s.outputFile.WriteString(litName + " dq " + strconv.FormatFloat(litValue, 'g', 11, 64) + "\n")
}

func EmitSection(s *State, section string) {
	section = strings.Trim(section, ".\n ")
	_, err := s.outputFile.WriteString("\nsection ." + section + "\n\n")
	if err != nil {
		panic(err)
	}
}

// EmitAlloc will allocate a memory area of given length and return a pointer in rax
func EmitAlloc(s *State, len int) {
	_, _ = Write(s, "   push rax\n", false)
	// Allocate result string and assign pointer to di
	_, _ = Write(s, "   push 50\n", false)
	_, _ = Write(s, "   call malloc\n", false)
}

func out(s *State, str string) {
	_, _ = Write(s, "   "+str+"\n", false)
}

// EmitConcat will concatenate the two strings at the top of the stack
// First string pointer in [rsp], second string pointer in [rax]
// It uses registers r12, r13, r14, rbx, rcx, rdx, rsi, rdi.
func EmitConcat(s *State) {
	// Get string 1 sizes/ptr into r14, rbx from [rsp]
	emit(s, "mov", "rdx", "[rsp]", " Get string 1 sizes/ptr into r14, rbx")
	emit(s, "mov", "r14d", "dword [rdx]", " Get string 1 sizes/ptr into r14, rbx")
	emit(s, "mov", "rbx", "rdx", "")
	emit(s, "add", "rbx", "8", "")
	// Get string 2 sizes/ptr into r12, r13 from rax
	emit(s, "mov", "r12d", "dword [rax]", " Get string 2 sizes/ptr into r12, r13")
	emit(s, "mov", "r13", "rax", "")
	emit(s, "add", "r13", "8", "")
	// Calculate new size to allocate, including 32 extra bytes
	emit(s, "mov", "rax", "r12", " Calculate new size to allocate, including 32 extra bytes")
	emit(s, "add", "rax", "r14", "")
	emit(s, "add", "rax", "32", "")
	// Allocate string
	emit(s, "call", "_alloc", "", "Allocate new string")
	// Save pointer in rdx and rdi for later use
	emit(s, "mov", "rdx", "rax", "Save pointer in rdx and rdi for later use")
	emit(s, "mov", "rdi", "rax", "")
	// Save new length
	emit(s, "mov", "rax", "r12", "Save new length")
	emit(s, "add", "rax", "r14", "")
	emit(s, "mov", "[rdi]", "rax", "")
	emit(s, "add", "rdi", "8", "")
	// Copy string 1
	emit(s, "mov", "rsi", "rbx", "Copy string 1")
	emit(s, "mov", "rcx", "r14", "")
	emit(s, "cld", "", "", "")
	emit(s, "rep", "movsb", "", "")
	// Copy string 2
	emit(s, "mov", "rsi", "r13", "Copy string 2")
	emit(s, "mov", "rcx", "r12", "")
	emit(s, "rep", "movsb", "", "")
	// Remove the top of stack. New TOS is the pointer in rax
	emit(s, "pop", "rax", "", "Remove the top of stack. New TOS is the pointer in rax")
	s.localSp--
	// Now AX should point to the string
	emit(s, "mov", "rax", "rdx", "Now AX should point to the string")
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
	}
}
