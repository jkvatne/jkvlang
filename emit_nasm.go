//go:build nasm

package main

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/jkvatne/jkv/code"
)

/*
 Register Windows ABI           JKV ABI
 0  rax   Return value
 1  rcx   First argument
 2  rdx   Second argument
 3  rbx   Preserved             Size of arguments on stack (bytes)
 4  rsp   Stack pointer
 5  rbp   Preserved
 6  rsi   Preserved
 7  rdi   Preserved             Function called for syscall
 8  r8    Third argument
 9  r9    Forth argument
 10 r10   Used in syscall
 11 r11   Used in syscall
 12 r12   Preserved
 13 r13   Preserved
 14 r14   Preserved
 15 r15   Preserved              Error pointer. 0 (nil) means ok.
*/

const (
	CarryFlag    = "0x01" // Bit 0
	ZeroFlag     = "0x40" // Bit 6
	SignFlag     = "0x80"
	OverflowFlag = "0x800"
)

// LocalSp
// RaxIsTOS

var CommentIndent = 40
var spaces = "                                                                                    "

func includeFile(txt string, libPath string) {
	str, err := os.ReadFile(path.Join(libPath, txt))
	code.Write(string(str))
	if err != nil {
		panic("Could not read library " + txt)
	}
}

func emit(op string, dst string, src string, comment string) {
	var txt string
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
	if comment != "" && !strings.Contains(comment, "->") {
		txt += spaces[0:max(0, CommentIndent-len(txt))] + "; " + comment + Sp(0) + "\n"
	} else {
		txt += spaces[0:max(0, CommentIndent-len(txt))] + "; " + comment + "\n"
	}
	code.Write(txt)
}

func EmitLitteral(litName string, litValue string) {
	code.Write("alignb 8\n")
	code.Write(litName + " dq " + strconv.Itoa(len(litValue)) + "\n")
	code.Write("     db `" + litValue + "`, 00h\n")
}

func EmitFloatLitteral(litName string, litValue float64) {
	value := strconv.FormatFloat(litValue, 'g', 11, 64)
	if !strings.Contains(value, ".") {
		if strings.Contains(value, "e") || strings.Contains(value, "E") {
			value = strings.Replace(value, "e", ".0e", 1)
		} else {
			value = value + ".0"
		}
	}
	code.Write(litName + " dq " + value + "\n")
}

func EmitSection(section string) {
	section = strings.Trim(section, ".\n ")
	code.Write("\nsection ." + section + "\n\n")
}

func EmitTextLabel(text string) {
	text = strings.Trim(text, ":\n ")
	code.Write("\n" + text + ":\n")
}

func EmitComment(comment string) {
	_ = code.Write("   ; " + comment + "\n")
}

func EmitNumericLabel(label int) string {
	return ".L" + strconv.Itoa(label)
}

func EmitLabel(label int, comment string) {
	n := code.Write(".L" + strconv.Itoa(label) + ":")
	code.Write(spaces[0:max(0, CommentIndent-n)] + "; " + comment + "\n")
}

func EmitJump(n int, comment string) {
	emit("jmp", ".L"+strconv.Itoa(n), "", comment)
}

func Sp(delta int) string {
	if delta == 0 {
		return " (" + strconv.Itoa(code.LocalSp) + ")"
	}
	code.LocalSp += delta
	return " (" + strconv.Itoa(code.LocalSp-delta) + "->" + strconv.Itoa(code.LocalSp) + ")"
}
func EmitPushTos(argNo int, funcName string) {
	if code.RaxIsTOS {
		code.Write("   push rax                             ; Push arg " +
			strconv.Itoa(argNo) + " of " + funcName + Sp(1) + "\n")
		code.RaxIsTOS = false
	}
}

func EmitCall(id string, nPar int, builtin bool) {
	if builtin {
		id = "_" + id
	}
	if nPar > 0 && code.RaxIsTOS {
		emit("push", "rax", "", "Push TOS from rax to stack"+Sp(1))
	}
	// The following is needed only for variadic functioncode.
	if nPar > 0 {
		emit("mov", "rbx", strconv.Itoa(nPar*8), "")
	} else {
		emit("xor", "rbx", "rbx", "")
	}

	emit("call", id, "", "")
}

func EmitFunction(id string) {
	EmitTextLabel(id)
	if code.LocalSp != 0 {
		panic("LocalSp is not 0")
	}
	// Function prologue. Set up new frame pointer.
	code.LocalSp = 0
	EmitComment("Setting localsp=0")
	if id != "main" {
		emit("push", "rbp", "", ""+Sp(1))
	}
	emit("mov", "rbp", "rsp", "")
	if id == "main" {
		EmitPrintSp()
		emit("call", "_sysinit", "", "")
	}
	code.RaxIsTOS = false
}

var TokenOp = map[Token]string{
	TOK_AND:        "and",
	TOK_OR:         "or",
	TOK_XOR:        "xor",
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

func EmitPushFloatLit(litNo int) {
	emit("mov", "rax", "[flt"+strconv.Itoa(litNo)+"]", "EmitPushFloatLit()")
	emit("push", "rax", "", "Push old tos in rax"+Sp(1))
	code.RaxIsTOS = false
}

func EmitJumpCond(op Token, unsignedOrFloat bool) error {
	lbl := code.NewLabel()
	emit("mov", "rax", "1", "Default to true")
	if op == TOK_EQ {
		emit("je", EmitNumericLabel(lbl), "", "")
	} else if op == TOK_NE {
		emit("jne", EmitNumericLabel(lbl), "", "")
	} else {
		if unsignedOrFloat {
			if op == TOK_GT {
				emit("ja", EmitNumericLabel(lbl), "", "")
			} else if op == TOK_LE {
				emit("jbe", EmitNumericLabel(lbl), "", "")
			} else if op == TOK_GE {
				emit("jae", EmitNumericLabel(lbl), "", "")
			} else if op == TOK_LT {
				emit("jb", EmitNumericLabel(lbl), "", "")
			} else {
				return fmt.Errorf("EmitJumpCond not implemented for " + op.Name())
			}
		} else {
			if op == TOK_GT {
				emit("jg", EmitNumericLabel(lbl), "", "")
			} else if op == TOK_LE {
				emit("jle", EmitNumericLabel(lbl), "", "")
			} else if op == TOK_GE {
				emit("jge", EmitNumericLabel(lbl), "", "")
			} else if op == TOK_LT {
				emit("jl", EmitNumericLabel(lbl), "", "")
			} else {
				return fmt.Errorf("EmitJumpCond not implemented for " + op.Name())
			}

		}
	}
	emit("mov", "rax", "0", "Return false if we did not jump")
	EmitLabel(lbl, "")
	code.RaxIsTOS = true
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
	}
	panic("AxName with invalid size")
}

// EmitOpAssignFloat constant float value to variable
func EmitOpAssignFloat(op Token, adr int, litNo int, comment string) error {
	if op == TOK_ASSIGN {
		emit("mov", "rax", "[flt"+strconv.Itoa(litNo)+"]", "")
		emit("mov", BpRel(adr), "rax", "")
	} else {
		panic("Float assign operation not implemented")
	}
	return nil
}

// EmitOpAssign will set variable at <adr> to <adr> op <value>
func EmitOpAssign(op Token, adr int, size int, value int64, comment string) error {
	instr := TokenOp[op]
	if instr == "" {
		return fmt.Errorf("EmitOpAssign called with invalid token %s", op.Name())
	}
	if instr == "imul" {
		emit("mov", "rax", strconv.FormatInt(value, 10), "OpAssign imul")
		if size == 4 {
			emit("mov", "ebx", DataType(size)+BpRel(adr), comment)
		} else {
			emit("mov", "rbx", DataType(size)+BpRel(adr), comment)
		}
		emit("imul", "rbx", "", "")
		// Move result to local variable at BpRel(adr)
		emit("mov", DataType(size)+BpRel(adr), AxName(size), "move result of *= to local variable")

	} else {
		emit(instr, DataType(size)+BpRel(adr), strconv.FormatInt(value, 10), comment)
	}
	return nil
}
func EmitOpAssignStringLitToField(offset int, fieldOfs int, litno int) error {
	emit("mov", "rax", DataType(8)+BpRel(offset), "")
	emit("add", "rax", strconv.Itoa(fieldOfs), "")
	emit("mov", "qword [rax]", "str"+strconv.Itoa(litno), "")
	return nil
}

func EmitOpAssignString(offset int, litno int) error {
	emit("mov", DataType(8)+BpRel(offset), "str"+strconv.Itoa(litno), "")
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
func EmitStoreConst(size int, value int64, offset int, comment string) {
	num := strconv.FormatInt(value, 10)
	emit("mov", DataType(size)+BpRel(offset), num, "")
}

func EmitLoadFloat64(size int, adr int, comment string) {
	EmitFlushRax("")
	code.RaxIsTOS = true
	emit("mov", "rax", BpRel(adr), comment)
}

func EmitLoadField(size int, localVarOfs int, fieldOffset int) {
	EmitFlushRax("")
	code.RaxIsTOS = true
	emit("mov", "rax", BpRel(localVarOfs), "EmitLoadField")
	emit("add", "rax", strconv.Itoa(fieldOffset), "Struct field offset")
	emit(MovOpcode(size), "rax", DataType(size)+" [rax]", "Move value to field")
}

// EmitLoad will push a local variable onto the stack (into AX)
func EmitLoad(size int, adr int, comment string) {
	EmitFlushRax("EmitLoad push TOS")
	code.RaxIsTOS = true
	emit(MovOpcode(size), "rax", DataType(size)+BpRel(adr), comment)
}

// EmitStoreToLocal will save the Top of Stack (AX) into a local variable of given size and offset.
// It will then clear RaxIsTos, effectively doing a pop
func EmitStoreToLocal(opcode string, size int, adr int, comment string) {
	emit(opcode, BpRel(adr), AxName(size), comment)
	code.RaxIsTOS = false
}

func EmitStoreF64(adr int, comment string) {
	emit("mov", BpRel(adr), "rax", comment)
}

// EmitJumpFalse will emit an instruction to jump if top of stack is false.
// Top of stack is typically already in AX
func EmitJumpFalse(n int, comment string) {
	if !code.RaxIsTOS {
		panic("TOS not in AX")
	}
	emit("or", "al", "al", "Set zero flag if rax is zero")
	emit("jz", ".L"+strconv.Itoa(n), "", comment)
	// Implicit pop of TOS
	code.RaxIsTOS = false
}

// EmitJumpTrue will emit an instruction to jump if top of stack is false.
// Top of stack is typically already in AX
func EmitJumpTrue(n int, comment string) {
	if !code.RaxIsTOS {
		panic("TOS not in AX")
	}
	emit("or", "al", "al", "Set zero flag if rax is zero")
	emit("jnz", ".L"+strconv.Itoa(n), "", "Jump if zero flag is set")
	// Implicit pop of TOS
	code.RaxIsTOS = false
}

// EmitAllocLocalVar will allocate a local variable
// TODO Allow for types larger than 8 byte. For now, use 8 bytes for all local variables
func EmitAllocLocalVar(comment string) int {
	emit("push", "0", "", comment+Sp(1))
	return -8 * code.LocalSp
}

func EmitPushStringLit(lit int, comment string) {
	EmitFlushRax("")
	emit("mov", "rax", "str"+strconv.Itoa(lit), comment)
	code.RaxIsTOS = true
}

func EmitSkipLenCap() {
	emit("add", "dword [rsp]", "8", "Skip len/cap")
}

func EmitPushConst(value int64, comment string) {
	EmitFlushRax("")
	if value == 0 {
		emit("xor", "rax", "rax", comment)
	} else {
		emit("mov", "rax", strconv.FormatInt(value, 10), "PushConst "+comment)
	}
	code.RaxIsTOS = true
}

func EmitFlushRax(comment string) {
	if code.RaxIsTOS {
		emit("push", "rax", "", comment+Sp(1))
		code.RaxIsTOS = false
	}
}

func EmitAssertTosInRax(comment string) {
	if !code.RaxIsTOS {
		emit("pop", "rax", "", comment+Sp(-1))
		code.RaxIsTOS = true
	}
}

func EmitPrintHello(format string) {
	emit("mov", "ecx", "-11", "STD_OUTPUT_HANDLE (.11)")
	emit("call", "GetStdHandle", "", "Handle returned in rax")
	emit("mov", "rcx", "rax", "1.arg - console handle")
	emit("mov", "rdx", "[rel msg]", "2.arg - pointer to message")
	emit("mov", "r8", "20", "3.arg - console handle")
	emit("xor", "r9", "r9", "4.arg - console handle")
	emit("mov", "qword [3sp+32]", "0", "5.arg - console handle")
}

// EmitConcat will concatenate the two strings at the top of the stack
// First string pointer in [rsp], second string pointer in [rax]
// It uses registers r12, r13, r14, rbx, rcx, rdx, rsi, rdi.
// Calls _alloc to allocate a new string with size for both the input strings + 32 bytes extra.
func EmitConcat(free1 bool, free2 bool) {
	EmitComment("")
	EmitComment("Start of EmitConcat")
	EmitAssertTosInRax("Get TOS")
	// Get string 1 sizes/ptr into r14, rbx from [rsp]
	emit("mov", "rdx", "[rsp]", "Get string 1 ptr into rdx")
	emit("mov", "rbx", "rdx", "Get string 1 ptr into rbx")
	emit("mov", "r14d", "dword [rdx]", "String 1 size into r14")
	// Get string 2 sizes/ptr into r12, r13 from rax
	emit("mov", "r12d", "dword [rax]", "Get string 2 size into r12d from TOS (rax)")
	emit("mov", "r13", "rax", "Save string 2 ptr in r13")
	// Calculate new size to allocate, including 32 extra bytes
	emit("mov", "rax", "r12", "Calculate new size to allocate, including 32 extra bytes")
	emit("add", "rax", "r14", "")
	emit("add", "rax", "40", "Add 32+8 to include len/cap")
	// Allocate string
	emit("call", "_alloc", "", "Allocate new string")
	// Save pointer in r9 and rdi for later use
	emit("mov", "rdi", "rax", "Save pointer in rdi for later use")
	emit("push", "rax", "", "Save pointer on stack for later use"+Sp(1))
	// Save new capacity/length
	emit("mov", "rsi", "r12", "First string length")
	emit("add", "rsi", "r14", "Add second length")
	emit("mov", "rax", "rsi", "New length")
	emit("add", "rsi", "40", "Add 32 for extra bytes and 8 for len/cap")
	emit("shl", "rsi", "32", "Move to cap (msw)")
	emit("or", "rax", "rsi", "")
	emit("mov", "[rdi]", "rax", "Save len/cap")
	emit("add", "rdi", "8", "move pointer to actual string data")
	// Copy string 1
	emit("mov", "rsi", "rbx", "Copy string 1")
	emit("add", "rsi", "8", "")
	emit("mov", "rcx", "r14", "")
	emit("cld", "", "", "")
	emit("rep", "movsb", "", "")
	// Copy string 2
	emit("mov", "rsi", "r13", "Copy string 2")
	emit("add", "rsi", "8", "Skip len/cap")
	emit("mov", "rcx", "r12", "")
	emit("rep", "movsb", "", "")
	if free1 {
		emit("mov", "rax", "rbx", "Free first argument to Concatenate")
		emit("call", "_free_str", "", "")
	}
	if free2 {
		emit("mov", "rax", "r13", "Free second argument to Concatenate")
		emit("call", "_free_str", "", "")
	}
	// Copy the allocated buffer address from r9 to rax. Now rax points to the new string.
	EmitPopAx("Now AX should point to the string")
	// Remove the top of stack. New TOS is the pointer in rax. Arguments in rbx and r13.
	emit("add", "rsp", "8", "Remove the top of stack. New TOS is the pointer in rax"+Sp(-1))
	code.RaxIsTOS = true
	EmitComment("End of EmitConcat")
	EmitComment("")
}

func EmitPrologue(libPath string) {
	EmitComment("File \"" + code.UnitName + ".asm\"\n")
	includeFile("sysinit.asm", libPath)
	includeFile("syscall.asm", libPath)
	includeFile("assert.asm", libPath)
	includeFile("printf.asm", libPath)
	includeFile("alloc.asm", libPath)
	includeFile("exit.asm", libPath)
	includeFile("sys.asm", libPath)
	EmitSection("text")
	emit("global", "main", "", "")
	code.EmitBlankLine()
	code.EmitBlankLine()
}

func EmitPrintSp() {
	if *PrintSp {
		emit("call", "_printsp", "", "")
		emit("call", "_fflush", "", "")
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
	case TOK_MOD:
		return TOK_INV_MOD
	default:
		return op
	}
}

// EmitCompareStrToLit : The pointer to the first string (val1) is found in AX. Compare it to the known constant in val2
func EmitCompareStrToLit(op Token, stringValue string, stringLitNo int, isTemp bool) (err error) {
	EmitAssertTosInRax("Get TOS")
	if op == TOK_EQ {
		emit("mov", "r14", "rax", "CompareStrings, save rax to r14")
		emit("mov", "rdi", "rax", "Save rax to rdi")
		// First check lengths
		emit("mov", "eax", "[rax]", "")
		emit("cmp", "eax", strconv.Itoa(len(stringValue)), "Compare string lengths")
		lbl := code.NewLabel()
		emit("mov", "rbx", "0", "Initialize result to false")
		emit("jne", EmitNumericLabel(lbl), "", "If not equal, jump to unequal end")
		emit("mov", "ecx", "eax", "")
		emit("mov", "rsi", "str"+strconv.Itoa(stringLitNo), "Pointer to literal string")
		emit("add", "rsi", "8", "Skip size of literal string")
		emit("add", "rdi", "8", "Skip size of string object")
		emit("cld", "", "", "")
		emit("repe", "cmpsb", "", "")
		emit("jne", EmitNumericLabel(lbl), "", "If not equal, jump to unequal end")
		emit("mov", "rbx", "1", "Strings was equal, set rax=true")
		EmitLabel(lbl, "")
		if isTemp {
			emit("mov", "rax", "r14", "isTemp")
			emit("call", "_free_str", "", "EmitCompareStrToLit")
		}
		emit("mov", "rax", "rbx", "Result to TOS (rax)")
		code.RaxIsTOS = true
		return nil
	} else if op == TOK_NE {
		lbl := code.NewLabel()
		emit("mov", "rbx", "1", "Initialize result to true")
		emit("mov", "rdi", "rax", "Save tos")
		emit("mov", "r14", "rax", "Save tos")
		emit("mov", "rsi", "str"+strconv.Itoa(stringLitNo), "Pointer to literal string")
		// First check lengths
		emit("cmp", "word [rax]", strconv.Itoa(len(stringValue)), "Compare string lengths")
		emit("jne", EmitNumericLabel(lbl), "", "If lengths not equal, jump to unequal end")
		emit("mov", "ecx", "[rax]", "Get nos length")
		emit("add", "rsi", "8", "Start of string 1")
		emit("add", "rdi", "8", "Start of string 2")
		emit("cld", "", "", "")
		emit("repe", "cmpsb", "", "")
		emit("jne", EmitNumericLabel(lbl), "", "If not equal, jump to unequal end")
		emit("mov", "rbx", "0", "Strings was equal, set rax=false")
		EmitLabel(lbl, "unequal")
		emit("mov", "rax", "rbx", "Result to TOS (rax)")
		code.RaxIsTOS = true
		return nil
	} else {
		return fmt.Errorf("EmitCompareStrings not implemented for " + op.Name())
	}
}

func EmitCompareStringsEq(temp1 bool, temp2 bool) {
	// Compare two strings, one in rax, and one on top of stack, and drop top of stack
	lbl := code.NewLabel()
	EmitAssertTosInRax("Get TOS")
	emit("mov", "rdi", "rax", "Save tos")
	emit("mov", "rsi", "[rsp]", "Get nos")
	emit("mov", "rcx", "4", "Compare first 4 bytes")
	emit("cld", "", "", "")
	emit("repe", "cmpsb", "", "")
	emit("pop", "rax", "", "Get nos ptr"+Sp(-1))
	emit("mov", "rbx", "0", "Initialize result to false")
	emit("jne", EmitNumericLabel(lbl), "", "If lengths not equal, jump to unequal end")
	emit("mov", "ecx", "[rax]", "Get nos length")
	emit("add", "rsi", "4", "Start of string 1")
	emit("add", "rdi", "4", "Start of string 2")
	emit("cld", "", "", "")
	emit("repe", "cmpsb", "", "")
	emit("jne", EmitNumericLabel(lbl), "", "If not equal, jump to unequal end")
	emit("mov", "rbx", "1", "Strings was equal, set rax=true")
	EmitLabel(lbl, "unequal")
	if temp1 {
		emit("mov", "rax", "rsi", "EmitCompareStringsEq 1")
		emit("call", "_free_str", "", "")
	}
	if temp2 {
		emit("mov", "rax", "rdi", "EmitCompareStringsEq 2")
		emit("call", "_free_str", "", "")
	}
	emit("mov", "rax", "rbx", "Result to TOS (rax)")
	code.RaxIsTOS = true
}

// EmitCompareStringsNe compares two strings, one in rax, and one on top of stack, and drop top of stack
func EmitCompareStringsNe(temp1 bool, temp2 bool) {
	lbl := code.NewLabel()
	EmitAssertTosInRax("Get TOS")
	emit("mov", "rdi", "rax", "Save tos")
	emit("mov", "rsi", "[rsp]", "Get nos")
	emit("mov", "rcx", "4", "Compare first 4 bytes")
	emit("cld", "", "", "")
	emit("repe", "cmpsb", "", "")
	emit("pop", "rax", "", "Get nos ptr"+Sp(-1))
	emit("mov", "rbx", "1", "Initialize result to true")
	emit("jne", EmitNumericLabel(lbl), "", "If lengths not equal, jump to unequal end")
	emit("mov", "ecx", "[rax]", "Get nos length")
	emit("add", "rsi", "4", "Start of string 1")
	emit("add", "rdi", "4", "Start of string 2")
	emit("cld", "", "", "")
	emit("repe", "cmpsb", "", "")
	emit("jne", EmitNumericLabel(lbl), "", "If not equal, jump to unequal end")
	emit("mov", "rbx", "0", "Strings was equal, set rax=false")
	EmitLabel(lbl, "unequal")
	if temp1 {
		emit("mov", "rax", "rsi", "EmitCompareStringsEq 1")
		emit("call", "_free_str", "", "")
	}
	if temp2 {
		emit("mov", "rax", "rdi", "EmitCompareStringsEq 2")
		emit("call", "_free_str", "", "")
	}
	emit("mov", "rax", "rbx", "Result to TOS (rax)")
	code.RaxIsTOS = true
}

// EmitFreeString assumes the full address exists in rax.
func EmitFreeString(comment string) {
	lbl := code.NewLabel()
	// Verify that rax is not nil
	emit("or", "rax", "rax", "")
	emit("jz", EmitNumericLabel(lbl), "", "")
	// Load len/cap
	emit("mov", "rbx", "[rax]", "")
	// Extract cap only
	emit("shr", "rbx", "32", "")
	// Skip free if cap=0
	emit("or", "rbx", "rbx", "")
	emit("jz", EmitNumericLabel(lbl), "", "")
	// Address is in rax. Just call _free_str
	emit("call", "_free_str", "", comment)
	// Exit label
	EmitLabel(lbl, "")
}

// EmitFreeStruct assumes the full address exists in rax.
// It will free the pointer in rax and decrement allocation_count by the size given.
func EmitFreeStruct(size int, comment string) {
	emit("mov", "rcx", strconv.Itoa(size), "")
	// _free_struct assumes pointer in rax and size in rcx
	emit("call", "_free_struct", "", "")
}

func EmitPushAx(txt string) {
	emit("push", "rax", "", txt+Sp(1))
}

func EmitPopAx(txt string) {
	emit("pop", "rax", "", txt+Sp(-1))
}

// EmitAddToSp adjusts stack pointer. Count is in qwordcode.
// Positive count to reserve space (push)
// Negative count to remove entries (pop)
func EmitAddToSp(count int, comment string) {
	if count > 0 {
		// Stack grows downward
		emit("sub", "rsp", strconv.Itoa(count*8), comment+Sp(count))
	} else if count < 0 {
		emit("add", "rsp", strconv.Itoa(-count*8), comment+Sp(count))
	}
}

func EmitPushConstString(litNo int) {
	emit("mov", "rax", "str"+strconv.Itoa(litNo), "PushConstString")
	code.RaxIsTOS = true
}

// EmitEpilogue - restores frame pointer and exit
func EmitEpilogue(name string) {
	if name == "main" {
		EmitPrintSp()
		oklbl := code.NewLabel()
		errlbl := code.NewLabel()
		emit("or", "r15", "r15", "")
		emit("jnz", ".L"+strconv.Itoa(errlbl), "", "Jump if zero flag is set")
		emit("mov", "rax", "[allocation_count]", "")
		emit("or", "rax", "rax", "")
		emit("jz", ".L"+strconv.Itoa(oklbl), "", "Jump if zero flag is set")
		EmitLabel(errlbl, "We had either err!=0 or allocationcount!=0")
		EmitComment("main() returning. Printing allocation count end err.")
		emit("push", "r15", "", ""+Sp(1))
		emit("mov", "rax", "[allocation_count]", "Printing allocation count")
		emit("push", "rax", "", ""+Sp(1))
		emit("mov", "rax", "alloc_size_str+8", "")
		emit("push", "rax", "", ""+Sp(1))
		emit("mov", "rbx", "24", "")
		emit("call", "_printf", "", "")
		emit("call", "_fflush", "", "")
		emit("add", "rsp", "24", ""+Sp(-3))
		EmitLabel(oklbl, "End of printing errors, returning error code via _exit()")
		emit("mov", "rax", "[allocation_count]", "Check that allocation count is zero")
		emit("or", "rax", "rax", "")
		emit("jz", ".L9999", "", "")
		emit("mov", "r15", "97", "If not zero, exit code=97")
		EmitLabel(9999, "")
		emit("mov", "rax", "r15", "Get error code")
		emit("call", "_exit", "", "")
	} else {
		emit("leave", "", "", "")
		code.LocalSp--
		emit("ret", "", "", "return from "+name)
	}
}

func EmitLoadErr() {
	emit("mov", "rax", "r15", "Load err")
	code.RaxIsTOS = true
}

func EmitStoreBpOfs(ofs int, comment string) {
	emit("mov", BpRel(ofs*8), "rax", comment)
}

func EmitStoreErr(err int) {
	emit("mov", "r15", strconv.Itoa(err), "Set tos to r15 = error value")
}

func EmitPopBx(comment string) {
	emit("pop", "rbx", "", comment+Sp(-1))
}

// EmitNewStruct will create a new struct object on the heap
// The pointer will be in the TOS (i.e. rax)
func EmitNewStruct(s *State, t *TypeDef) {
	emit("mov", "rax", strconv.Itoa(t.Size()), "")
	emit("call", "_alloc", "", "Allocate new struct")
	code.RaxIsTOS = true
}

func EmitAddToRsi(s *State, ofs int) {
	emit("add", "rsi", strconv.Itoa(ofs), "")
}

func EmitLoadIndirect() {
	emit("mov", "rax", "[rsi]", "")
}
func EmitStoreIndirect(size int) {
	if size == 8 {
		emit("mov", "[rsi]", "rax", "")
	} else if size == 4 {
		emit("mov", "dword [rsi]", "eax", "")
	} else if size == 2 {
		emit("mov", "word [rsi]", "ax", "")
	} else if size == 1 {
		emit("mov", "byte [rsi]", "al", "")
	} else {
		panic("Internal error - store indirect with wrong size")
	}
}

func EmitLoadEa(localOfs int) {
	emit("mov", "rsi", BpRel(localOfs), "EmitLoadEa")
}

func EmitAssignIndirectStrLit(litNo int, size int, comment string) {
	emit("mov", DataType(size)+"[rax]", "str"+strconv.Itoa(litNo), "11 "+comment)
}

func EmitAssignIndirectInt(size int, value int64, comment string) {
	emit("mov", DataType(size)+"[rsi]", strconv.Itoa(int(value)), comment)
}

func EmitGetAddrOfLocal(ofs int) {
	emit("lea", "rax", BpRel(ofs), "")
	emit("push", "rax", "", ""+Sp(1))
}

func EmitNewString() {
	// Allocate string
	EmitAssertTosInRax("")
	emit("mov", "r12", "rax", "new string capacity")
	emit("call", "_alloc", "", "Allocate new string")
	emit("mov", "rsi", "rax", "Save rax")
	emit("mov", "rdi", "rax", "Then clear the new string")
	emit("xor", "rax", "rax", "")
	emit("mov", "rcx", "r12", "")
	emit("cld", "", "", "")
	emit("rep", "stosb", "", "")
	emit("shl", "r12", "32", "")
	emit("mov", "[rsi]", "r12", "Store capacity")
	emit("mov", "rax", "rsi", "Restore rax pointing to string")
	code.RaxIsTOS = true
}

func EmitNot() {
	EmitAssertTosInRax("")
	emit("xor", "rax", "1", "")
}

func EmitPushLabel(label int) {
	emit("lea", "rax", "[rel .L"+strconv.Itoa(label)+"]", "")
	emit("push", "rax", "", ""+Sp(1))
}

func EmitPushFramePointer() {
	emit("push", "rbp", "", ""+Sp(1))
}

func EmitJumpOnError(label int) {
	emit("or", "r15", "r15", "Check err")
	emit("jz", ".L"+strconv.Itoa(label), "", "")
}

func EmitClearBreakErr() {
	emit("mov", "rax", "r15", "Clear r15 if it was 1")
	emit("dec", "rax", "", "")
	emit("or", "rax", "rax", "")
	emit("cmovz", "r15", "rax", "")
}

func EmitNegate() {
	EmitAssertTosInRax("")
	emit("neg", "rax", "", "")
}
