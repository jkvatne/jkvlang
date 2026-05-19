//go:build nasm

package main

import (
	"fmt"
	"math"
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
	if comment != "" {
		txt += spaces[0:max(0, CommentIndent-len(txt))] + "; " + comment + "\n"
	} else {
		txt += spaces[0:max(0, CommentIndent-len(txt))] + "\n"
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
	code.Write(text + ":\n")
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

func EmitPushTos(argNo int, funcName string) {
	if code.RaxIsTOS {
		code.Write("   push rax                             ; Push arg " +
			strconv.Itoa(argNo) + " of " + funcName + " (" + strconv.Itoa(code.LocalSp) + ")\n")
		code.LocalSp++
		code.RaxIsTOS = false
	}
}

func EmitCall(id string, nPar int, builtin bool) {
	if builtin {
		id = "_" + id
	}
	if nPar > 0 && code.RaxIsTOS {
		emit("push", "rax", "", "Push TOS from rax to stack")
		code.LocalSp++
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
	if id != "main" {
		emit("push", "rbp", "", "")
	}
	emit("mov", "rbp", "rsp", "")
	code.LocalSp = 0
	if id == "main" {
		EmitPrintSp()
		emit("call", "_sysinit", "", "")
	}
	code.RaxIsTOS = false
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

func EmitPushFloatLit(litNo int) {
	emit("mov", "rax", "[flt"+strconv.Itoa(litNo)+"]", "EmitPushFloatLit()")
	emit("push", "rax", "", "Push old tos in rax")
	code.LocalSp++
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
				return fmt.Errorf("EmitJumpCond not implemented")
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
				return fmt.Errorf("EmitJumpCond not implemented")
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
		emit("movzx", "rbx", DataType(size)+BpRel(adr), comment)
		emit("imul", "rbx", "", "")
		// Move result to local variable at BpRel(adr)
		emit("mov", DataType(size)+BpRel(adr), AxName(size), "move result of *= to local variable")

	} else {
		emit(instr, DataType(size)+BpRel(adr), strconv.FormatInt(value, 10), comment)
	}
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
	if code.RaxIsTOS {
		emit("push", "rax", "", "Push TOS loading float")
		code.LocalSp++
	}
	code.RaxIsTOS = true
	emit("mov", "rax", BpRel(adr), comment)
}

// EmitLoad will push a local variable onto the stack (into AX)
func EmitLoad(size int, adr int, comment string) {
	if code.RaxIsTOS {
		emit("push", "rax", "", "1 Push TOS")
		code.LocalSp++
	}
	code.RaxIsTOS = true
	emit(MovOpcode(size), "rax", DataType(size)+BpRel(adr), comment)
}

// EmitStore will save the Top of Stack (AX) into a local variable of given size.
// It will then clear RaxIssTos, effectively doing a pop
func EmitStore(opcode string, size int, adr int, comment string) {
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
// TODO Allow for types larger than 8 bytecode. For now, use 8 bytes for all localcode.
func EmitAllocLocalVar(comment string) int {
	// emit("sub", "rsp", "8", comment)
	emit("push", "0", "", comment)
	code.LocalSp++
	return -8 * code.LocalSp
}

func EmitPushStringLit(lit int, comment string) {
	if code.RaxIsTOS {
		emit("push", "rax", "", "2 Push TOS")
		code.LocalSp++
	}
	emit("mov", "rax", "str"+strconv.Itoa(lit), comment)
	code.RaxIsTOS = true
}

func EmitSkipLenCap() {
	emit("add", "dword [rsp]", "8", "Skip len/cap")
}

func EmitPushConst(value int64, comment string) {
	if code.RaxIsTOS {
		emit("push", "rax", "", "EmitPushConst() Push TOS")
		code.LocalSp++
	}
	if value == 0 {
		emit("xor", "rax", "rax", comment)
	} else {
		emit("mov", "rax", strconv.FormatInt(value, 10), "PushConst "+comment)
	}
	code.RaxIsTOS = true
}

func EmitFlushRax(comment string) {
	if code.RaxIsTOS {
		emit("push", "rax", "", comment)
		code.LocalSp++
		code.RaxIsTOS = false
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
	if !code.RaxIsTOS {
		emit("pop", "rax", "", "TOS was on stack. Pop it into rax")
		code.LocalSp--
	}
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
	emit("push", "rax", "", "Save pointer on stack for later use")
	code.LocalSp++
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
	emit("pop", "rax", "", "Now AX should point to the string")
	code.LocalSp--
	code.RaxIsTOS = true
	// Remove the top of stack. New TOS is the pointer in rax. Arguments in rbx and r13.
	emit("add", "rsp", "8", "Remove the top of stack. New TOS is the pointer in rax")
	code.LocalSp--
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
		if val2.Typ.Pt.IsInteger() {
			if val2.IntValue == 0 {
				return &NoValue, fmt.Errorf("can not divide by zero")
			}
			result.IntValue = val1.IntValue / val2.IntValue
		} else if val2.Typ.Pt.IsFloat() {
			result.FloatValue = val1.FloatValue / val2.FloatValue
		}
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
func EmitCompareStrToLit(op Token, stringValue string, stringLitNo int, isTemp bool) (err error) {
	if !code.RaxIsTOS {
		emit("pop", "rax", "", "EmitCompareStrToLit, pop first argument into rax")
		code.LocalSp--
	}
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
			emit("mov", "rax", "r14", "")
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
	if !code.RaxIsTOS {
		emit("pop", "rax", "", "CompareStrings, get TOS to rax")
		code.LocalSp--
	}
	emit("mov", "rdi", "rax", "Save tos")
	emit("mov", "rsi", "[rsp]", "Get nos")
	emit("mov", "rcx", "4", "Compare first 4 bytes")
	emit("cld", "", "", "")
	emit("repe", "cmpsb", "", "")
	emit("pop", "rax", "", "Get nos ptr")
	code.LocalSp--
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
	if !code.RaxIsTOS {
		emit("pop", "rax", "", "CompareStrings, get TOS to rax")
		code.LocalSp--
	}
	emit("mov", "rdi", "rax", "Save tos")
	emit("mov", "rsi", "[rsp]", "Get nos")
	emit("mov", "rcx", "4", "Compare first 4 bytes")
	emit("cld", "", "", "")
	emit("repe", "cmpsb", "", "")
	emit("pop", "rax", "", "Get nos ptr")
	code.LocalSp--
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

// EmitFreeLocalVariables will free an object in a local variable
func EmitFreeLocalVariables(adr int, pt PrimaryType, comment string) error {
	if pt == TYP_STRING {
		// Decrement allocation count, first load size given in offset +4 (capacity)
		emit("mov", "rax", BpRel(adr), "Load cap")
		emit("mov", "rax", "[rax]", "")
		emit("shr", "rax", "32", "")
		// Skip free if cap=0
		lbl := code.NewLabel()
		emit("or", "rax", "rax", "")
		emit("jz", EmitNumericLabel(lbl), "", "")
		// Load the offset from the variable in local stack frame with offset given by adr
		emit("mov", "rax", BpRel(adr), "")
		emit("call", "_free_str", "", comment)
		EmitLabel(lbl, "")
		return nil
	}
	return fmt.Errorf("can not free %s", TokenNames[pt])
}

func EmitPushAx(txt string) {
	emit("push", "rax", "", txt)
	code.LocalSp++
}

func EmitPopAx(txt string) {
	emit("pop", "rax", "", txt)
	code.LocalSp--
}

// EmitAddToSp adjusts stack pointer. Count is in qwordcode.
// Positive count to reserve space (push)
// Negative count to remove entries (pop)
func EmitAddToSp(count int, comment string) {
	if count > 0 {
		// Stack grows downward
		emit("sub", "rsp", strconv.Itoa(count*8), comment)
	} else if count < 0 {
		emit("add", "rsp", strconv.Itoa(-count*8), comment)
	}
	code.LocalSp += count
}

func EmitPushConstString(litNo int) {
	emit("mov", "rax", "str"+strconv.Itoa(litNo), "PushConstString")
	code.RaxIsTOS = true
}

// EmitEpilogue - restores frame pointer and exit
func EmitEpilogue(name string) {
	if name == "main" {
		EmitPrintSp()
		// Print remaining allocation
		EmitComment("main() returning. Printing allocation count.")
		emit("push", "r15", "", "")
		code.LocalSp++
		emit("mov", "rax", "[allocation_count]", "Printing allocation count")
		emit("push", "rax", "", "")
		code.LocalSp++
		emit("mov", "rax", "alloc_size_str+8", "")
		emit("push", "rax", "", "")
		code.LocalSp++
		emit("mov", "rbx", "24", "")
		emit("call", "_printf", "", "")
		emit("call", "_fflush", "", "")
		emit("add", "rsp", "16", "")
		code.LocalSp -= 2
		EmitComment("Returning error code via _exit()")
		emit("mov", "rax", "r15", "Get error code")
		emit("call", "_exit", "", "")
	} else {
		emit("leave", "", "", "")
		emit("ret", "", "", "return from "+name)
	}
}

func EmitLoadErr() {
	emit("mov", "rax", "r15", "Load err")
	code.RaxIsTOS = true
}

func EmitStoreReturnValue(i int) {
	emit("mov", BpRel(16+8*i), "rax", "")
}

func EmitStoreErr(err int, comment string) {
	emit("mov", "r15", strconv.Itoa(err), "Set tos to r15 = error value")
}
