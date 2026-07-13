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
	if !strings.Contains(comment, "->") {
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

func EmitF64Litteral(litName string, litValue float64) {
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

func EmitF32Litteral(litName string, litValue float32) {
	value := strconv.FormatFloat(float64(litValue), 'g', 11, 32)
	if !strings.Contains(value, ".") {
		if strings.Contains(value, "e") || strings.Contains(value, "E") {
			value = strings.Replace(value, "e", ".0e", 1)
		} else {
			value = value + ".0"
		}
	}
	code.Write(litName + " dd " + value + "\n")
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

func Label(n int) string {
	return ".L" + strconv.Itoa(n)
}

func EmitJump(n int, comment string) {
	emit("jmp", Label(n), "", comment)
}

func Sp(delta int) string {
	ss := "-" + code.StackState()
	if delta == 0 {
		return " (" + strconv.Itoa(code.LocalSp) + ")" + ss
	}
	code.LocalSp += delta
	return " (" + strconv.Itoa(code.LocalSp-delta) + "->" + strconv.Itoa(code.LocalSp) + ")" + ss
}

func EmitPushTos(argNo int, funcName string) {
	if code.AxIsTos() {
		code.Write("   push rax                             ; Push arg " +
			strconv.Itoa(argNo) + " of " + funcName + Sp(1) + "\n")
		code.SetSp()
	}
}

func EmitCall(id string, nPar int, builtin bool) {
	EmitComment("Call function")
	if builtin {
		id = "_" + id
	}
	if nPar > 0 && code.AxIsTos() {
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
	EmitComment("Setting localsp=0")
	if id != "main" {
		emit("push", "rbp", "", "")
	}
	emit("mov", "rbp", "rsp", "")
	code.LocalSp = 0
	if id == "main" {
		EmitPrintSp()
		emit("call", "_sysinit", "", "")
	}
	code.SetUndef()
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
	TOK_SHL:        "shl",
	TOK_SHR:        "shr",
	TOK_AND_NOT:    "andnot",
}

func xmm(sp int) string {
	return "xmm" + strconv.Itoa(sp)
}

func EmitPushF64Lit(litNo int) {
	emit("mov", "rax", "[f64_"+strconv.Itoa(litNo)+"]", "EmitPushF64Lit()")
	emit("push", "rax", "", "Push old tos in rax"+Sp(1))
	code.SetSp()
}

func EmitPushF32Lit(litNo int) {
	emit("mov", "eax", "dword [f32_"+strconv.Itoa(litNo)+"]", "EmitPushF32Lit()")
	emit("push", "rax", "", "Push old tos in rax"+Sp(1))
	code.SetSp()
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
	code.SetAx()
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

// EmitOpAssignF64 constant float value to variable
func EmitOpAssignF64(op Token, adr int, litNo int, comment string) error {
	if op == TOK_ASSIGN {
		code.SetAx()
		emit("mov", "rax", "[f64_"+strconv.Itoa(litNo)+"]", comment)
		code.SetUndef()
		emit("mov", BpRel(adr), "rax", "")
	} else {
		return fmt.Errorf("F64 assign operation %s not implemented", op.Name())
	}
	return nil
}

// EmitOpAssignF32 constant float value to variable
func EmitOpAssignF32(op Token, adr int, litNo int, comment string) error {
	if op == TOK_ASSIGN {
		code.SetAx()
		emit("mov", "rax", "[f32_"+strconv.Itoa(litNo)+"]", comment)
		code.SetUndef()
		emit("mov", BpRel(adr), "rax", "")
	} else {
		return fmt.Errorf("F32 assign operation %s not implemented", op.Name())
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
		if value > 0x7FFFFFFF || value < -0x7FFFFFFF {
			if instr == "mov" {
				emit(instr, DataType(4)+BpRel(adr), strconv.FormatInt(value&0xFFFFFFFF, 10), comment)
				emit(instr, DataType(4)+BpRel(adr+4), strconv.FormatInt((value>>32)&0xFFFFFFFF, 10), comment)
			} else {
				return fmt.Errorf("value out of range")
			}
		} else {
			emit(instr, DataType(size)+BpRel(adr), strconv.FormatInt(value, 10), comment)
		}
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
	}
	panic("Bp relative addressing with zero offset")
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
	if size >= 8 {
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
	emit("mov", DataType(size)+BpRel(offset), num, comment)
}

func EmitLoadFloat(size int, adr int, comment string, fun string) {
	EmitFlushRax("Before LoadFloat")
	code.SetAx()
	if size == 8 {
		emit("mov", "rax", BpRel(adr), comment)
	} else if size == 4 {
		// if fun == "print" || fun == "printf" {
		//	emit("cvtss2sd", "xmm0", "dword "+BpRel(adr), "Load F32 and convert to F64")
		//	emit("movq", "rax", "xmm0", "Set rax to 64bit float value")
		// } else {
		emit("mov", "eax", "dword "+BpRel(adr), comment)
		// }
	}
}

func EmitLoadField(size int, localVarOfs int, fieldOffset int) {
	EmitFlushRax("Before LoadField")
	code.SetAx()
	emit("mov", "rax", BpRel(localVarOfs), "EmitLoadField")
	emit("add", "rax", strconv.Itoa(fieldOffset), "Struct field offset")
	emit(MovOpcode(size), "rax", DataType(size)+" [rax]", "Load value from field")
}

// EmitLoad will push a local variable onto the stack (into AX)
func EmitLoad(size int, adr int, comment string) {
	EmitFlushRax("EmitLoad: push TOS")
	code.SetAx()
	emit(MovOpcode(size), "rax", DataType(size)+BpRel(adr), "EmitLoad: "+comment)
}

// EmitStoreToLocal will save the Top of Stack (AX) into a local variable of given size and offset.
// It will then clear RaxIsTos, effectively doing a pop
func EmitStoreToLocal(opcode string, size int, adr int, comment string) {
	emit(opcode, BpRel(adr), AxName(size), comment)
	code.SetSp()
}

func EmitStoreF64(adr int, comment string) {
	emit("mov", BpRel(adr), "rax", comment)
}

func EmitStoreF32(adr int, comment string) {
	emit("mov", BpRel(adr), "eax", "Store F32")
}

// EmitJumpFalse will emit an instruction to jump if top of stack is false.
// Top of stack is typically already in AX
func EmitJumpFalse(n int, comment string) {
	if !code.AxIsTos() {
		panic("TOS not in AX")
	}
	emit("or", "al", "al", comment)
	emit("jz", Label(n), "", "")
	// Implicit pop of TOS
	code.SetUndef()
}

// EmitJumpTrue will emit an instruction to jump if top of stack is false.
// Top of stack is typically already in AX
func EmitJumpTrue(n int, comment string) {
	if !code.AxIsTos() {
		panic("TOS not in AX")
	}
	emit("or", "al", "al", comment)
	emit("jnz", Label(n), "", "")
	// Implicit pop of TOS
	code.SetUndef()
}

// EmitAllocLocalVar will allocate a local variable
// TODO Allow for types larger than 8 byte. For now, use 8 bytes for all local variables
func EmitAllocLocalVar(comment string) int {
	emit("sub", "rsp", "8", comment+Sp(1))
	return -8 * code.LocalSp
}

func EmitPushStringLit(lit int, comment string) {
	EmitFlushRax("Before PushStringLit")
	code.SetAx()
	emit("mov", "rax", "str"+strconv.Itoa(lit), comment)
}

func EmitPushConst(value int64, comment string) {
	EmitFlushRax("Before PushConst")
	code.SetAx()
	if value == 0 {
		emit("xor", "rax", "rax", comment)
	} else {
		emit("mov", "rax", strconv.FormatInt(value, 10), "PushConst "+comment)
	}
}

func EmitFlushRax(comment string) {
	if code.AxIsTos() {
		emit("push", "rax", "", comment+Sp(1))
		code.SetUndef()
	}
}

func EmitAssertTosInRax(comment string) {
	if !code.AxIsTos() {
		code.SetAx()
		emit("pop", "rax", "", comment+Sp(-1))
	}
}

// EmitConcat will concatenate the two strings at the top of the stack
// First string pointer in [rsp], second string pointer in [rax]
// It uses registers r12, r13, r14, rbx, rcx, rdx, rsi, rdi.
// Calls _alloc to allocate a new string with size for both the input strings + 32 bytes extra.
func EmitConcat(free1 bool, free2 bool) {
	EmitComment("Start of EmitConcat")
	EmitAssertTosInRax("Get TOS before concat string")
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
	emit("add", "rsi", "32", "Add 32 for extra bytes")
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
	emit("add", "rsi", "8", "Skip len/ca string 2")
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
	case TOK_SHL:
		return TOK_INV_SHL
	case TOK_SHR:
		return TOK_INV_SHR
	default:
		return op
	}
}

// EmitCompareStrToLit : The pointer to the first string (val1) is found in AX. Compare it to the known constant in val2
func EmitCompareStrToLit(op Token, stringValue string, stringLitNo int, isTemp bool) (err error) {
	EmitAssertTosInRax("Get TOS before compare string")
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
		emit("mov", "r13", "1", "Strings was equal, set r13=true")
		EmitLabel(lbl, "")
		if isTemp {
			emit("mov", "rbx", "[r14]", "isTem=true, check if it is a const string with cap=0")
			emit("shr", "rbx", "32", "")
			emit("or", "rbx", "rbx", "")
			lb := code.NewLabel()
			emit("jz", EmitNumericLabel(lb), "", "")
			emit("mov", "rax", "r14", "")
			emit("call", "_free_str", "", "EmitCompareStrToLit")
			EmitLabel(lb, "")
		}
		emit("mov", "rax", "r14", "Result to TOS (rax)")
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
		return nil
	}
	return fmt.Errorf("EmitCompareStrings not implemented for " + op.Name())
}

func EmitCompareStringsEq(temp1 bool, temp2 bool) {
	// Compare two strings, one in rax, and one on top of stack, and drop top of stack
	lbl := code.NewLabel()
	EmitAssertTosInRax("Get TOS before compare strings eq")
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
}

// EmitCompareStringsNe compares two strings, one in rax, and one on top of stack, and drop top of stack
func EmitCompareStringsNe(temp1 bool, temp2 bool) {
	lbl := code.NewLabel()
	EmitAssertTosInRax("Get TOS before compare strings NE")
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
}

// EmitFreeString assumes the full address exists in rax.
func EmitFreeString(comment string) {
	lbl := code.NewLabel()
	// Verify that rax is not nil
	emit("or", "rax", "rax", "EmitFreeString")
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
	code.SetUndef()
}

// EmitFreeStruct assumes the full address exists in rax.
// It will free the pointer in rax and decrement allocation_count by the size given.
func EmitFreeStruct(size int, comment string) {
	emit("mov", "rcx", strconv.Itoa(size), comment)
	// _free_struct assumes pointer in rax and size in rcx
	emit("call", "_free_struct", "", "")
	code.SetUndef()
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
	EmitFlushRax("Before NewStruct")
	code.SetAx()
	emit("mov", "rax", "str"+strconv.Itoa(litNo), "PushConstString")
}

// EmitEpilogue - restores frame pointer and exit
func EmitEpilogue(name string) {
	if name == "main" {
		EmitPrintSp()
		oklbl := code.NewLabel()
		errlbl := code.NewLabel()
		emit("or", "r15", "r15", "")
		emit("jnz", Label(errlbl), "", "Jump if zero flag is set")
		emit("mov", "rax", "[allocation_count]", "")
		emit("or", "rax", "rax", "")
		emit("jz", Label(oklbl), "", "Jump if zero flag is set")
		EmitLabel(errlbl, "We had either err!=0 or allocationcount!=0")
		EmitComment("main() returning. Printing allocation count end err.")
		emit("push", "r15", "", ""+Sp(1))
		emit("mov", "rax", "[allocation_count]", "Printing allocation count")
		emit("push", "rax", "", "d"+Sp(1))
		emit("mov", "rax", "alloc_size_str+8", "")
		emit("push", "rax", "", "c"+Sp(1))
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
		emit("ret", "", "", "return from "+name)
	}
}

func EmitLoadErr() {
	EmitFlushRax("Before NewStruct")
	code.SetAx()
	emit("mov", "rax", "r15", "Load err")
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

func EmitAddToRsi(ofs int) {
	emit("add", "rsi", strconv.Itoa(ofs), "")
}

func EmitLoadIndirect() {
	emit("mov", "rax", "[rsi]", "")
}

// EmitStoreIndirect has Pointer on stack, value in rax
func EmitStoreIndirect(op string, size int) {
	emit("pop", "rsi", "", Sp(-1))
	if size == 8 {
		emit(op, "[rsi]", "rax", "EmitStoreIndirect quad")
	} else if size == 4 {
		emit(op, "dword [rsi]", "eax", "EmitStoreIndirect dword")
	} else if size == 2 {
		emit(op, "word [rsi]", "ax", "EmitStoreIndirect word")
	} else if size == 1 {
		emit(op, "byte [rsi]", "al", "EmitStoreIndirect byte")
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

func EmitAssignIndirectConstInt(size int, unsigned bool, value int64, comment string) {
	emit("mov", DataType(size)+"[rax]", strconv.Itoa(int(value)), comment)
}

func EmitGetAddrOfLocal(ofs int) {
	emit("lea", "rax", BpRel(ofs), "")
	emit("push", "rax", "", "b"+Sp(1))
}

func EmitNewString() {
	// Allocate string
	EmitAssertTosInRax("Before NewString")
	emit("mov", "r12", "rax", "new string capacity")
	emit("add", "rax", "8", "Add space for cap/len")
	emit("call", "_alloc", "", "Allocate new string")
	emit("mov", "rsi", "rax", "Save rax")
	emit("mov", "rdi", "rax", "Then clear the new string")
	emit("xor", "rax", "rax", "")
	emit("mov", "rcx", "r12", "")
	emit("add", "rcx", "8", "Add space for cap/len befor clearing")
	emit("cld", "", "", "")
	emit("rep", "stosb", "", "")
	emit("shl", "r12", "32", "")
	emit("mov", "[rsi]", "r12", "Store capacity")
	code.SetAx()
	emit("mov", "rax", "rsi", "Restore rax pointing to string")
}

// EmitNewStruct will create a new struct object on the heap
// The pointer will be in the TOS (i.e. rax)
func EmitNewStruct(t *TypeDef) {
	EmitFlushRax("Before NewStruct")
	code.SetAx()
	emit("mov", "rax", strconv.Itoa(t.Size()), "")
	emit("call", "_alloc", "", "Allocate new struct")
}

func EmitNewSlice(t *TypeDef, elementSize int, hasLen bool) {
	EmitAssertTosInRax("Before NewSlice")
	if hasLen {
		emit("mov", "r14", "rax", "new slice length")
		emit("pop", "rax", "", Sp(-1))
	} else {
		emit("mov", "r14", "0", "new slice length is zero")
	}
	emit("mov", "r12", "rax", "new slice capacity (in elements)")
	emit("imul", "rax", strconv.Itoa(elementSize), "")
	emit("add", "rax", "8", "Add space for len/cap")
	emit("call", "_alloc", "", "Allocate new slice")
	emit("mov", "r13", "rax", "Save rax")
	emit("mov", "rdi", "rax", "Then clear the new slice")
	emit("xor", "rax", "rax", "")
	emit("mov", "rcx", "r12", "")
	emit("cld", "", "", "")
	emit("rep", "stosb", "", "")
	emit("shl", "r12", "32", "")
	emit("add", "r12", "r14", "")
	emit("mov", "[r13]", "r12", "Store capacity")
	code.SetAx()
	emit("mov", "rax", "r13", "Restore rax pointing to slice")

}

func EmitNot() {
	EmitAssertTosInRax("Value to 'Not'")
	emit("xor", "rax", "1", "")
}

func EmitJumpOnError(label int) {
	emit("or", "r15", "r15", "Check err")
	emit("jz", Label(label), "", "")
}

func EmitClearBreakErr() {
	emit("mov", "rax", "r15", "Clear r15 if it was 1")
	emit("dec", "rax", "", "")
	emit("or", "rax", "rax", "")
	emit("cmovz", "r15", "rax", "")
}

func EmitNegate() {
	EmitAssertTosInRax("Value to 'Negate'")
	emit("neg", "rax", "", "")
}

// EmitFreeIfExists must preserve rax, because it contains pointer to the new struct
func EmitFreeIfExists(offset int, size int, txt string) {
	emit("mov", "r12", "rax", "")
	emit("mov", "rax", BpRel(offset), txt)
	emit("or", "rax", "rax", "Is pointer nil?")
	lbl := code.NewLabel()
	emit("jz", Label(lbl), "", "")
	emit("mov", "rcx", strconv.Itoa(size), "")
	emit("call", "_free_struct", "", "")
	EmitLabel(lbl, "")
	emit("mov", "rax", "r12", "")
}

// EmitAppend expects NOS=pointer to slice, TOS=value to append
// size is the element size in bytes
func EmitAppend(size int) {
	EmitAssertTosInRax("")
	emit("mov", "rdi", "[rsp]", "")
	emit("mov", DataType(size)+"[rdi]", AxName(size), "")
	// TOS is no longer in rax
	code.SetUndef()
}

func EmitLoadGlobalConst(name string) {
	EmitFlushRax("Before EmitLoadGlobalConst")
	emit("mov", "rax", name, "")
	code.SetAx()
}
