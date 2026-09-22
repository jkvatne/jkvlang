//go:build nasm

package main

import (
	"fmt"
	"log/slog"
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
	startSp := code.LocalSp
	if op == "push" {
		code.LocalSp++
	} else if op == "pop" {
		code.LocalSp--
	} else if (op == "add" || op == "sub") && dst == "rsp" {
		n, err := strconv.Atoi(src)
		if err != nil {
			panic(err)
		}
		if n%8 != 0 {
			panic("Stack add/sub must be multiple of 8")
		}
		if op == "add" {
			code.LocalSp -= n / 8
		} else {
			code.LocalSp += n / 8
		}
	}
	endSp := code.LocalSp
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
	ss := fmt.Sprintf("%d->%d", startSp, endSp)
	txt += spaces[0:max(0, CommentIndent-len(txt))] + "; " + ss + " " + comment + "\n"
	code.Write(txt)
}

func EmitStringLitteral(litName string, litValue string) {
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

func EmitExtern(name string) {
	code.Write("extern " + name + "\n")
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

func SpTxt() string {
	ss := "-" + code.StackState()
	return " (" + strconv.Itoa(code.LocalSp) + "->" + strconv.Itoa(code.LocalSp) + ")" + ss
}

func EmitPushTos(argNo int, funcName string) {
	if code.AxIsTos() {
		// code.Write("   push rax                             ; Push arg " + strconv.Itoa(argNo) + " of " + funcName + "\n")
		emit("push", "rax", "", "Push arg "+strconv.Itoa(argNo)+" of "+funcName)
		code.SetSp()
	}
}

func EmitCall(id string, nPar int, builtin bool) {
	EmitComment("Call function " + id)
	if builtin {
		id = "_" + id
	}
	if nPar > 0 && code.AxIsTos() {
		emit("push", "rax", "", "Push TOS from rax to stack")
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
	code.SetUndef()
}

var TokenOp = map[Token]string{
	TOK_AND:        "and",
	TOK_OR:         "or",
	TOK_XOR:        "xor",
	TOK_PLUS:       "add",
	TOK_MINUS:      "sub",
	TOK_MULT:       "mul",
	TOK_DIV:        "div",
	TOK_PLUS_ASGN:  "add",
	TOK_MINUS_ASGN: "sub",
	TOK_OR_ASGN:    "or",
	TOK_AND_ASGN:   "and",
	TOK_ASSIGN:     "mov",
	TOK_MULT_ASGN:  "imul",
	TOK_DIV_ASGN:   "idiv",
	TOK_SHL:        "shl",
	TOK_SHR:        "shr",
	TOK_AND_NOT:    "andnot",
}

func xmm(sp int) string {
	return "xmm" + strconv.Itoa(sp)
}

func EmitPushF64Lit(x float64) {
	litNo := AddF64Lit(x)
	emit("mov", "rax", "[f64_"+strconv.Itoa(litNo)+"]", "EmitPushF64Lit()")
	emit("push", "rax", "", "Push old tos in rax")
	code.SetSp()
}

func EmitPushF32Lit(x float32) {
	litNo := AddF32Lit(x)
	emit("mov", "eax", "dword [f32_"+strconv.Itoa(litNo)+"]", "EmitPushF32Lit()")
	emit("push", "rax", "", "Push old tos in rax")
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

func EmitLoadFloat(size int, adr int, comment string) {
	EmitFlushRax("Before LoadFloat")
	code.SetAx()
	if size == 8 {
		emit("mov", "rax", BpRel(adr), comment)
	} else if size == 4 {
		emit("mov", "eax", "dword "+BpRel(adr), comment)
	}
}

// EmitLoad will push a local variable onto the stack (into AX)
func EmitLoad(size int, adr int, comment string) {
	EmitFlushRax("EmitLoad, flush rax onto stack")
	emit(MovOpcode(size), "rax", DataType(size)+BpRel(adr), comment)
	code.SetAx()
}

// EmitJumpFalse will emit an instruction to jump if top of stack is false.
// Top of stack is typically already in AX
func EmitJumpFalse(reg string, lbl int, comment string) {
	emit("or", reg, reg, comment)
	emit("jz", Label(lbl), "", "")
	// Implicit pop of TOS
	code.SetUndef()
}

// EmitJumpTrue will emit an instruction to jump if top of stack is false.
// Top of stack is typically already in AX
func EmitJumpTrue(reg string, lbl int, comment string) {
	emit("or", reg, reg, comment)
	emit("jnz", Label(lbl), "", "")
	// Implicit pop of TOS
	code.SetUndef()
}

// EmitAllocLocalVar will allocate a local variable
// TODO Allow for types larger than 8 byte. For now, use 8 bytes for all local variables
func EmitAllocLocalVar(comment string) int {
	emit("xor", "rax", "rax", "EmitAllocLocalVar "+comment)
	emit("push", "rax", "", "")
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
		emit("push", "rax", "", comment)
		code.SetUndef()
	}
}

func EmitAssertTosInRax(comment string) {
	if !code.AxIsTos() {
		code.SetAx()
		emit("pop", "rax", "", comment)
	}
}

func EmitPrologue(libPath string, inc bool) {
	if inc {
		EmitComment("File \"" + code.UnitName + ".asm\"\n")
		includeFile("sys.asm", libPath)
	}

	if !inc {
		EmitExtern("_sysinit")
		EmitExtern("_assert")
		EmitExtern("allocation_count")
		EmitExtern("_printf")
		EmitExtern("_print")
		EmitExtern("_printsp")
		EmitExtern("_fflush")
		EmitExtern("_exit")
		EmitExtern("_invert_err")
		EmitExtern("_alloc")
		EmitExtern("_free_struct")
		EmitExtern("_free_str")
		EmitExtern("_free_slice")
		EmitExtern("_len")
		EmitExtern("_lptr")
		EmitExtern("_cptr")
		EmitExtern("_bitlen")
		EmitExtern("alloc_size_str")
		EmitExtern("ExitProcess")
		EmitExtern("processHeap")
		EmitExtern("f32sign_mask")
		EmitExtern("f64sign_mask")
		EmitExtern("argv")
		EmitExtern("argc")
		EmitExtern("arg0")
		EmitExtern("arg1")
		EmitExtern("arg2")
		EmitExtern("args")
		EmitExtern("_cstrlen")
		EmitSection("text")
	}

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

// EmitCompareStrToLit : The pointer to the first string (val1) is found in TOS. Compare it to the known constant in val2
func EmitCompareStrToLit(op Token, stringValue string, stringLitNo int, isTemp bool) (err error) {
	EmitAssertTosInRax("Get TOS before compare string")
	if op == TOK_EQ || op == TOK_NE {
		lbl := code.NewLabel()
		emit("mov", "r13", "0", "Initialize result to false")
		emit("mov", "rdi", "rax", "Save rax to rdi")
		emit("mov", "r14", "rax", "CompareStrings, save rax to r14")
		// Make sure string is not nil
		emit("or", "rax", "rax", "Check for nil")
		emit("jz", EmitNumericLabel(lbl), "", "")
		// First check lengths
		emit("mov", "eax", "[rax]", "")
		emit("cmp", "eax", strconv.Itoa(len(stringValue)), "Compare string lengths")
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
		emit("mov", "rax", "r13", "Result to TOS (rax)")
		if op == TOK_NE {
			emit("xor", "rax", "1", "Invert result for TOK_NE")
		}
		return nil
	}
	return fmt.Errorf("EmitCompareStrings not implemented for " + op.Name())
}

func emitCompareString() int {
	lbl := code.NewLabel()
	emit("jne", EmitNumericLabel(lbl), "", "If lengths not equal, jump to unequal end")
	emit("mov", "ecx", "[rax]", "Get nos length")
	emit("add", "rsi", "4", "Start of string 1")
	emit("add", "rdi", "4", "Start of string 2")
	emit("cld", "", "", "")
	emit("repe", "cmpsb", "", "")
	emit("jne", EmitNumericLabel(lbl), "", "If not equal, jump to unequal end")
	return lbl
}

func EmitCompareStringsEq(temp1 bool, temp2 bool) {
	// Compare two strings, one in rax, and one on top of stack, and drop top of stack

	EmitAssertTosInRax("Get TOS before compare strings eq")
	emit("mov", "rdi", "rax", "Save tos")
	emit("mov", "rsi", "[rsp]", "Get nos")
	emit("mov", "rcx", "4", "Compare first 4 bytes")
	emit("cld", "", "", "")
	emit("repe", "cmpsb", "", "")
	emit("pop", "rax", "", "Get nos ptr")
	emit("mov", "rbx", "0", "Initialize result to false")
	lbl := emitCompareString()
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
	EmitAssertTosInRax("Get TOS before compare strings NE")
	emit("mov", "rdi", "rax", "Save tos")
	emit("mov", "rsi", "[rsp]", "Get nos")
	emit("mov", "rcx", "4", "Compare first 4 bytes")
	emit("cld", "", "", "")
	emit("repe", "cmpsb", "", "")
	emit("pop", "rax", "", "Get nos ptr")
	emit("mov", "rbx", "1", "Initialize result to true")
	lbl := emitCompareString()
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
	emit("or", "rax", "rax", "EmitFreeString "+comment)
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
	EmitLabel(lbl, "End of EmitFreeString")
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
	emit("push", "rax", "", txt)
}

func EmitPopAx(txt string) {
	emit("pop", "rax", "", txt)
}

// EmitAddToSp adjusts stack pointer. Count is in qwordcode.
// Positive count to reserve space (push)
// Negative count to remove entries (pop)
func EmitAddToSp(count int, comment string) {
	if count > 0 {
		// Stack grows downward
		emit("sub", "rsp", strconv.Itoa(count*8), comment)
		emit("mov", "qword [rsp]", "0", "Clear")
	} else if count < 0 {
		emit("add", "rsp", strconv.Itoa(-count*8), comment)
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
		emit("push", "r15", "", "")
		emit("mov", "rax", "[allocation_count]", "Printing allocation count")
		emit("push", "rax", "", "")
		emit("mov", "rax", "alloc_size_str+8", "")
		emit("push", "rax", "", "")
		emit("mov", "rbx", "24", "")
		emit("call", "_printf", "", "")
		emit("call", "_fflush", "", "")
		emit("add", "rsp", "24", "")
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
	for _, name = range Externals {
		EmitExtern(name)
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
	emit("pop", "rbx", "", comment)
}

func EmitGetAddrOfLocal(ofs int) {
	emit("lea", "rax", BpRel(ofs), "")
	emit("push", "rax", "", "b")
}

func EmitNewString(hasLen bool) {
	// Allocate string
	EmitAssertTosInRax("Before NewString")
	if hasLen {
		emit("mov", "r13", "rax", "save new string length")
		emit("pop", "rax", "", ""+"get capacity into rax")
	}
	emit("mov", "r12", "rax", "save new string capacity")
	emit("add", "rax", "8", "Add space for cap/len")
	emit("call", "_alloc", "", "Allocate new string")
	emit("mov", "rsi", "rax", "Save pointer to new string")
	emit("mov", "rdi", "rax", "Then clear the new string")
	emit("xor", "rax", "rax", "")
	emit("mov", "rcx", "r12", "")
	emit("add", "rcx", "8", "Add space for cap/len befor clearing")
	emit("cld", "", "", "")
	emit("rep", "stosb", "", "")
	emit("shl", "r12", "32", "")
	if hasLen {
		emit("add", "r12", "r13", "Insert length")
	}
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
	// Zero struct
	emit("mov", "rdx", "rax", "")
	emit("mov", "rdi", "rax", "")
	emit("mov", "rcx", strconv.Itoa(t.Size()), "")
	emit("cld", "", "", "")
	emit("xor", "rax", "rax", "")
	emit("rep", "stosb", "", "")
	emit("mov", "rax", "rdx", "")
}

func EmitNewSlice(elementSize int, hasLen bool) {
	EmitAssertTosInRax("Before NewSlice")
	if hasLen {
		emit("mov", "r14", "rax", "new slice length")
		emit("pop", "rax", "", "")
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

// EmitNegateF64 will negate the 64bit float value in rax
func EmitNegateF64() {
	emit("mov", "rcx", "[f64sign_mask]", "")
	emit("xor", "rax", "rcx", "")
}

func EmitNegateF32() {
	emit("mov", "rcx", "[f32sign_mask]", "")
	emit("xor", "rax", "rcx", "")
}

func EmitNegate() {
	EmitAssertTosInRax("Value to 'Negate'")
	emit("neg", "rax", "", "")
}

func EmitLoadGlobalConst(name string) {
	EmitFlushRax("Before EmitLoadGlobalConst")
	emit("mov", "rax", name, "")
	code.SetAx()
}

// EmitCopyStringToRam will copy a read-only string (with cap=0) to RAM
// Assumes the string pointer is given in rax
// Uses R12, R13
// On return, rax is pointer to new string
func EmitCopyStringToRam() {
	lbl := code.NewLabel()
	EmitComment("EmitCopyStringToRam -copy string to RAM if it is read-only. rax is string pointer")
	emit("mov", "rdx", "rax", "Load pointer to constant string")
	emit("mov", "rdx", "[rdx]", "Load string len/cap")
	emit("shr", "rdx", "32", "")
	emit("or", "rdx", "rdx", "Test if capacity is zero")
	emit("jnz", Label(lbl), "", "")
	emit("mov", "r13", "rax", "Load pointer to constant string")
	emit("mov", "rax", "[r13]", "Fetch length of string (cap should be zero)")
	emit("add", "rax", "32", "Add space for cap and spare bytes")
	emit("mov", "r12", "rax", "cap to r12")
	emit("sub", "r12", "8", "not include cap/len word")
	emit("shl", "r12", "32", "")
	emit("call", "_alloc", "", "Allocate new string")
	emit("mov", "rdi", "rax", "")
	emit("add", "rdi", "8", "Skip len/cap when moving string")
	emit("mov", "r14", "rax", "")
	emit("add", "r12", "[r13]", "Add len to len/cap in r12")
	emit("mov", "rsi", "r13", "")
	emit("mov", "rcx", "[rsi]", "")
	emit("add", "rsi", "8", "Skip len/cap when moving string")
	emit("cld", "", "", "")
	emit("rep", "movsb", "", "copy old string")
	emit("mov", "[r14]", "r12", "Mov len/cap into string")
	emit("mov", "rax", "r14", "")
	EmitLabel(lbl, "EmitCopyStringToRam done, Writeable string now in rax")
}

func EmitLea(ofs int, comment string) {
	emit("lea", "rsi", BpRel(ofs), comment)
	emit("push", "rsi", "", "")
}

// EmitModifyConstIndexedCharIndirect assumes pointer to string in rax
func EmitModifyConstIndexedCharIndirect(offset int) {
	EmitComment("EmitModifyConstIndexedCharIndirect")
	emit("push", "rax", "", "Save rax before copying string")
	emit("mov", "rax", "[rax]", "")
	EmitCopyStringToRam()
	emit("pop", "rdi", "", "")
	emit("mov", "[rdi]", "rax", "")
	emit("add", "rax", strconv.Itoa(offset), "EmitModifyConstIndexedCharIndirect")
	emit("add", "rax", "8", "Skip len/cap of string not const")
}

func EmitModifyConstIndexedChar(addr int, offset int) {
	EmitComment("EmitModifyConstIndexedChar")
	emit("mov", "rax", BpRel(addr), "EmitModifyConstIndexedChar")
	EmitCopyStringToRam()
	emit("mov", BpRel(addr), "rax", "")
	emit("add", "rax", strconv.Itoa(offset), "EmitModifyConstIndexedChar")
	emit("add", "rax", "8", "Skip len/cap of string not const")
}

// EmitModifyIndexedCharIndirect
// TOS is value of index in rax,  NOS is pointer to string pointer
func EmitModifyIndexedCharIndirect() {
	EmitComment("EmitModifyIndexedCharIndirect")
	EmitFlushRax("Flush rax")
	emit("mov", "rax", "[rsp+8]", "Load pointer to string pointer")
	emit("mov", "rax", "[rax]", "Get string itself")
	EmitCopyStringToRam()
	emit("mov", "rbx", "[rsp+8]", "")
	emit("mov", "[rbx]", "rax", "")
	emit("add", "rax", "[rsp]", "")
	emit("add", "rax", "8", "Skip len/cap of string not const")
	emit("add", "rsp", "8", "Done EmitModifyIndexedCharIndirect")
	emit("mov", "[rsp]", "rax", "")
}

// EmitModifyIndexedChar
// String pointer in variable at <addr>
// TOS is new value
func EmitModifyIndexedChar(addr int) {
	EmitComment("EmitModifyIndexedChar")
	emit("push", "rax", "", ""+"EmitModifyIndexedChar")
	emit("mov", "rax", BpRel(addr), "")
	EmitCopyStringToRam()
	emit("mov", BpRel(addr), "rax", "Update variable to point at new string in case it has changed")
	emit("pop", "rbx", "", "")
	emit("add", "rax", "rbx", "Add index")
	emit("add", "rax", "8", "Skip len/cap of string not const")
}

func EmitModifyConstIndexedSlice(offset int, size int, returnLbl int) {
	emit("mov", "rsi", "[rax]", "Load slice pointer for const")
	ofs := 8 + offset*size
	emit("mov", "eax", "dword [rsi]", "Load len/cap")
	emit("cmp", "eax", strconv.Itoa(offset), "Check for index out of bounds")
	lbl := code.NewLabel()
	emit("jg", Label(lbl), "", "Jump if ok")
	emit("mov", "r15", "96", "Error code")
	emit("jmp", Label(returnLbl), "", "return with error")
	EmitLabel(lbl, "")
	emit("add", "rsi", strconv.Itoa(ofs), "Index into slice, skipping len/cap")
	emit("mov", "rax", "rsi", "")
}

func EmitModifyIndexedSlice(size int) {
	emit("mov", "rax", "[rax]", "Load slice pointer 1")
	emit("add", "rax", "8", "Skip len/cap in slice")
	emit("pop", "rbx", "", "Get index")
	emit("shl", "rbx", ShiftFromSize(size), "")
	emit("add", "rax", "rbx", "Index into lvalue slice not const")
}

func EmitLoadField(lvalueOffset int, indirect bool, fieldOffset int, varName string, fieldName string) {
	if !indirect {
		EmitFlushRax("Flush rax before EmitLoadField of " + fieldName)
		emit("mov", "rax", BpRel(lvalueOffset), "Load local variable "+varName)
	}
	if fieldOffset != 0 {
		emit("add", "rax", strconv.Itoa(fieldOffset), "LoadField: Add field offset for field '"+fieldName+"'")
	}
	code.SetAx()
}

func EmitLoadWithOffset(ofs int, comment string) {
	EmitPushAx("")
	emit("mov", "rax", "[rax+"+strconv.Itoa(ofs)+"]", comment)
	code.SetAx()
}

func EmitFreeSlice(t *TypeDef) {
	emit("mov", "rcx", strconv.Itoa(t.Element.Size()), "Load element size")
	// _free_slice assumes pointer in rax and element size in rcx
	emit("call", "_free_slice", "", "")
	code.SetUndef()
}

func EmitStartAppend(length int) {
	emit("mov", "rsi", "[rax]", "Load slice pointer for append")
	emit("mov", "eax", "[rsi]", "Get length")
	emit("imul", "rax", strconv.Itoa(length), "")
	emit("add", "rsi", "rax", "")
	emit("add", "rsi", "8", "Add slice offset")
	emit("push", "rsi", "", "")
	code.SetUndef()
}

func EmitDoAppend(length int) {
	emit("pop", "rdi", "", "")
	emit("mov", DataType(length)+"[rdi]", AxName(length), "")
	emit("add", "rdi", strconv.Itoa(length), "")
	emit("push", "rdi", "", "")
	code.SetUndef()
}

func EmitUpdateAppendLength(n int) {
	emit("pop", "rdi", "", "")
	emit("mov", "rax", "[rdi]", "Get length")
	emit("add", "rax", strconv.Itoa(n), "")
	emit("mov", "[rdi]", "rax", "")
}

func EmitConvertF32toF64() {
	if code.AxIsTos() {
		emit("cvtss2sd", "xmm0", "eax", "convert rax F32 to F64")
	} else {
		emit("cvtss2sd", "xmm0", "dword [rsp]", "convert F32 to F64")
	}
	emit("movq", "rax", "xmm0", "Set rax to 64bit float value")
	emit("mov", "[rsp]", "rax", "")
}

func EmitLoadIndexedVar(frameOfs int, index int64, size int) {
	ofs := int(index) * size
	emit("mov", "rax", BpRel(frameOfs), "")
	emit("add", "rax", strconv.Itoa(ofs), "")
	if size == 1 {
		emit("movzx", "eax", "byte [rax+8]", "")
	}
	code.SetAx()
}

func EmitLoadTosIndirect(size int, fieldName string) {
	EmitComment("EmitLoadTosIndirect")
	code.SetAx()
	emit(MovOpcode(size), "rax", DataType(size)+" [rax]", "Load value in field '"+fieldName+"'")
	EmitComment("")
}

// EmitLoadGlobal TOS is index. Pointer is in global variable <id>
func EmitLoadGlobal(id string, size int, index int, isConst bool) {
	if isConst {
		emit("mov", "rax", "["+id+"]", "EmitLoadGlobal")
		emit("add", "rax", strconv.Itoa(index*size+8), "Index element "+strconv.Itoa(index)+" of string/slice")
	} else {
		EmitAssertTosInRax("Assure tos (index) is in rax")
		if size > 1 {
			emit("imul", "rax", strconv.Itoa(size), "")
		}
		emit("add", "rax", "8", "")
		emit("add", "rax", "["+id+"]", "EmitLoadGlobal")
	}
	if size == 1 {
		emit("movzx", "rax", "byte [rax]", "Get char from string in EmitLoadGlobal")
	} else if size == 2 || size == 4 {
		emit("movzx", "rax", DataType(size)+"[rax]", "")
	} else if size == 8 {
		emit("mov", "rax", "[rax]", "")
	} else {
		panic("TODO")
	}
	code.SetAx()
}

// LoadIndexedValue assumes TOS is the index (after parsing index)
// If isIndirect, NOS will be the pointer (address of string/slice)
// if !isIndirect, we use the offset to find the local variable. TOS is still index.
// isConst==true means the index is in the parameter "index", and not on stack.
func LoadIndexedValue(isIndirect bool, isConst bool, offset int, index int64, size int) {
	if !isIndirect && !isConst {
		// Load index into rax
		EmitAssertTosInRax("LoadIndexedValue: Assure index in rax")
		emit("mov", "rbx", BpRel(offset), "LoadIndexedValue: Get local var string address into rbx")
		if size > 1 {
			emit("imul", "rax", strconv.Itoa(size), "")
		}
		emit("add", "rax", "8", "Skip len/cap")
		emit("add", "rax", "rbx", "Calculate address by adding offset 1")
	} else if !isIndirect {
		// Local variable with constant index
		emit("mov", "rax", BpRel(offset), "LoadIndexedValue: Get local var string address into rbx")
		emit("add", "rax", strconv.Itoa(int(index)*size+8), "LoadIndexedValue: Index element "+strconv.Itoa(int(index))+" of string/slice")
	} else if !isConst {
		// TOS is index, NOS is pointer
		EmitAssertTosInRax("LoadIndexedValue: Assure tos in rax")
		emit("pop", "rbx", "", "LoadIndexedValue. Get pointer in NOS into rbx")
		// Check for nil pointer
		emit("or", "rbx", "rbx", "Check for nil pointer")
		lbl := code.NewLabel()
		emit("jnz", Label(lbl), "", "")
		emit("mov", "r15", "105", "")
		// emit("jmp", ".L9999", "", "Panic termination")
		EmitLabel(lbl, "")
		if size > 1 {
			emit("imul", "rax", strconv.Itoa(size), "")
		}
		emit("add", "rax", "8", "Skip len/cap")
		emit("add", "rax", "rbx", "Calculate address by adding offset 2")
	} else {
		// Tos is index. Pointer in local variable at offset, Const offset into string.
		EmitAssertTosInRax("LoadIndexedValue: Assure index in rax")
		emit("mov", "rbx", BpRel(offset), "LoadIndexedValue: Get local var string address into rbx")
		emit("add", "rax", strconv.Itoa(int(index)*size+8), "LoadIndexedValue: Index element "+strconv.Itoa(int(index))+" of string/slice")
		// emit("add", "rax", "rbx", "Calculate address by adding offset 3")
	}
	code.SetAx()
	if size == 1 {
		emit("movzx", "rax", "byte [rax]", "LoadIndexedValue: Get char from string")
	} else if size == 2 || size == 4 {
		emit("movzx", "rax", DataType(size)+"[rax]", "LoadIndexedValue: Get word/dword")
	} else if size == 8 {
		emit("mov", "rax", "[rax]", "LoadIndexedValue: Get qword")
	} else {
		panic("TODO")
	}
}

func EmitClearErr() {
	emit("xor", "r15", "r15", "Clear error")
}

func EmitSkipLenCapForPrint() {
	emit("add", "dword [rsp]", "8", "Skip len/cap of print argument literal string")
}

// EmitCompareIntegers will compare the top two stack entries
func EmitCompareIntegers(op Token, unsigned bool) (err error) {
	EmitPopBx("Pop next on stack into RBX")
	emit("cmp", "rbx", "rax", "Compare two ints")
	return EmitJumpCond(op, unsigned)
}

// EmitCompareIntConst will compare top of stack with a constant
func EmitCompareIntConst(op Token, value int64, unsigned bool) error {
	sval := strconv.FormatInt(value, 10)
	if value > 0x7fffffff || value < -0x7fffffff {
		// value = -0xffffffff + value
		emit("mov", "rbx", sval, "")
		emit("cmp", "rax", "rbx", "Compare int with 64bit const")
	} else {
		emit("cmp", "rax", sval, "Compare int with const")
	}
	return EmitJumpCond(op, unsigned)
}

// EmitIntegerOp will generate a stack operation on the top two stack entries, like add or sub
// The stack pointer will be incremented (pop), and the result will now be on top of the stack (AX)
// We assume TOS is in rax. Then NOS will be popped to rcx.
// For subtraction, we should calculate NOS-TOS or rcx-rax
func EmitIntegerOp(op Token) {
	if !code.AxIsTos() {
		panic("emitIntegerOp assumes RaxIsTOS=true")
	}
	if op == TOK_DIV {
		emit("pop", "rcx", "", "")
		emit("xchg", "rax", "rcx", "")
		emit("cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit("idiv", "rcx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
	} else if op == TOK_MOD {
		emit("pop", "rcx", "", "")
		emit("xchg", "rax", "rcx", "")
		emit("cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit("idiv", "rcx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
		emit("mov", "rax", "rdx", "Move reminder to AX (top of stack)")
	} else if op == TOK_MINUS {
		emit("pop", "rcx", "", "")
		emit("sub", "rax", "rcx", "Integer op minus")
		emit("neg", "rax", "", "")
	} else if op == TOK_AND_NOT {
		emit("pop", "rcx", "", "")
		emit("not", "rax", "", "")
		emit("and", "rax", "rcx", "AndNot")
	} else {
		instruction := TokenOp[op]
		if instruction == "" {
			slog.Error("EmitIntegerOp called with invalid token", "op", op.Name())
		}
		if op == TOK_MULT {
			emit("pop", "rcx", "", "")
			emit("mul", "rcx", "", "Integer op mul")
		} else if op == TOK_SHL || op == TOK_SHR {
			emit("mov", "rcx", "rax", "Integer op shift")
			emit("pop", "rax", "", "")
			emit(instruction, "rax", "cl", "Integer op shift")
		} else {
			emit("pop", "rcx", "", "")
			emit(instruction, "rax", "rcx", "Integer op other")
		}
	}
}

// EmitOpIntConst will evaluate tos=tos op <constant>
// It uses 64bit integer values on the 64 bit rax register
func EmitOpIntConst(op Token, value int64, comment string) error {
	if !code.AxIsTos() {
		panic("emitOpIntConst assumes RaxIsTOS=true")
	}
	if value < -0x7FFFFFFF || value > 0x7FFFFFFFF {
		return fmt.Errorf("value out of range: %d", value)
	}
	sval := strconv.FormatInt(value, 10)
	if op == TOK_DIV {
		emit("cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit("mov", "rbx", sval, "Get constant divisor into RBX")
		emit("idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
	} else if op == TOK_INV_DIV {
		emit("mov", "rbx", sval, "Get constant divisor into RBX")
		emit("xchg", "rax", "rbx", "")
		emit("cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit("idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
	} else if op == TOK_MOD {
		emit("mov", "rbx", sval, "RBX=constant divisor")
		emit("cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit("idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
		emit("mov", "rax", "rdx", "Move reminder to AX (top of stack)")
	} else if op == TOK_INV_MOD {
		emit("mov", "rbx", sval, "RBX=constant divisor")
		emit("xchg", "rax", "rbx", "")
		emit("cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit("idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
		emit("mov", "rax", "rdx", "Move reminder to AX (top of stack)")
	} else if op == TOK_MULT {
		emit("imul", "rax", "rax, "+sval, "")
	} else if op == TOK_INV_MINUS {
		emit("sub", "rax", strconv.FormatInt(value, 10), comment)
		emit("neg", "rax", "", "")
	} else if op == TOK_MINUS {
		emit("sub", "rax", strconv.FormatInt(value, 10), comment)
	} else if op == TOK_INV_SHL {
		emit("mov", "rcx", "rax", "TOK_INV_SHL")
		emit("mov", "rax", strconv.FormatInt(value, 10), comment)
		emit("shl", "rax", "cl", "")
	} else if op == TOK_INV_SHR {
		emit("mov", "rcx", "rax", "TOK_INV_SHR")
		emit("mov", "rax", strconv.FormatInt(value, 10), comment)
		emit("shr", "rax", "cl", "")
	} else {
		instr := TokenOp[op]
		if instr == "" {
			return fmt.Errorf("invalid operation %s", op.Name())
		}
		emit(instr, "rax", strconv.FormatInt(value, 10), instr+" "+comment)
	}
	return nil
}

// Float operations

// emitFloatOp assumes the operands are already in xmm1 and xmm2
// The result will be in xmm1
func emitFloatOp(op Token, size int) error {
	sufix := "d"
	if size == 32 {
		sufix = "s"
	}
	if op == TOK_PLUS || op == TOK_PLUS_ASGN {
		emit("adds"+sufix, xmm(1), xmm(2), "Add tos to nos")
	} else if op == TOK_MINUS || op == TOK_MINUS_ASGN {
		emit("subs"+sufix, xmm(1), xmm(2), "Subtract nos from tos")
	} else if op == TOK_INV_MINUS {
		emit("subs"+sufix, xmm(2), xmm(1), "Subtract nos from tos")
		if size == 64 {
			emit("movq", xmm(1), xmm(2), "")
		} else {
			emit("movq", xmm(1), xmm(2), "")
		}
	} else if op == TOK_MULT || op == TOK_MULT_ASGN {
		emit("muls"+sufix, xmm(1), xmm(2), "Multiply nos by tos")
	} else if op == TOK_DIV || op == TOK_DIV_ASGN {
		emit("divs"+sufix, xmm(1), xmm(2), "Divide tos by nos")
	} else if op == TOK_INV_DIV {
		emit("divs"+sufix, xmm(2), xmm(1), "Divide nos by tos")
		if size == 64 {
			emit("movq", xmm(1), xmm(2), "")
		} else {
			emit("movq", xmm(1), xmm(2), "")
		}
	} else {
		return fmt.Errorf("float operation not implemented for " + op.Name())
	}
	code.SetAx()
	return nil
}

func EmitOpF64Const(op Token, x float64) error {
	EmitAssertTosInRax("Get TOS before float op const")
	litNo := AddF64Lit(x)
	emit("movq", xmm(1), "rax", "emitOpF64Const move tos in rax to xmm1")
	emit("mov", "rax", "[f64_"+strconv.Itoa(litNo)+"]", "emitOpF64Const")
	emit("movq", xmm(2), "rax", "emitOpF64Const mov nos to xmm2")
	err := emitFloatOp(op, 64)
	emit("movq", "rax", xmm(1), "EmitOpF64Const: Move float result into rax")
	return err
}

func EmitOpF32Const(op Token, x float32) error {
	litNo := AddF32Lit(x)
	EmitAssertTosInRax("Get TOS before float op const")
	emit("movd", xmm(1), "eax", "emitOpF32Const move tos in rax to xmm1")
	emit("mov", "eax", "[f32_"+strconv.Itoa(litNo)+"]", "emitOpF32Const")
	emit("movd", xmm(2), "eax", "emitOpF32Const mov nos to xmm2")
	err := emitFloatOp(op, 32)
	code.SetAx()
	emit("movd", "eax", xmm(1), "EmitOpF32Const: Move float result into rax")
	return err
}

func EmitF64Op(op Token, typ1 code.PrimaryType, typ2 code.PrimaryType) error {
	// load TOS into xmm1 and convert to F64 if necessary
	if typ2.IsInteger() {
		emit("cvtsi2sd", xmm(2), "rax", "convert integer into xmm2")
	} else if typ2 == code.TYP_F32 {
		emit("cvtss2sd", xmm(2), "rax", "convert F32  into xmm2")
	} else if typ2 == code.TYP_F64 {
		emit("movq", xmm(2), "rax", "EmitFloatOp mov nos to xmm2")
	} else {
		return fmt.Errorf("EmitFloatOp not implemented for " + op.Name())
	}
	// Load NOS into xmm2 and convert to F64 if necessary
	emit("pop", "rax", "", "EmitFloatOp pop nos")
	if typ1.IsInteger() {
		emit("cvtsi2sd", xmm(1), "rax", "convert integer into xmm1")
	} else if typ1 == code.TYP_F32 {
		emit("cvtss2sd", xmm(1), "rax", "convert F32  into xmm1")
	} else if typ1 == code.TYP_F64 {
		emit("movq", xmm(1), "rax", "EmitFloatOp mov nos to xmm1")
	} else {
		return fmt.Errorf("EmitFloatOp not implemented for " + op.Name())
	}
	// Do F64 opertion
	err := emitFloatOp(op, 64)
	emit("movq", "rax", xmm(1), "EmitF64Op: Move float result into rax")
	return err
}

func EmitF32Op(op Token, typ1 code.PrimaryType, typ2 code.PrimaryType) error {
	// F32 operations
	// load TOS into xmm1 and convert to F32 if necessary
	if typ1.IsInteger() {
		emit("cvtsi2ss", xmm(2), "rax", "convert integer into xmm2")
	} else if typ1 == code.TYP_F32 {
		emit("movd", xmm(2), "eax", "EmitFloatOp mov nos to xmm2")
	} else {
		return fmt.Errorf("EmitFloatOp not implemented for " + op.Name())
	}
	// Load NOS into xmm2 and convert to F32 if necessary
	emit("pop", "rax", "", "EmitFloatOp pop nos")
	if typ2.IsInteger() {
		emit("cvtsi2ss", xmm(1), "rax", "convert integer into xmm1")
	} else if typ2 == code.TYP_F32 {
		emit("movd", xmm(1), "eax", "EmitFloatOp mov nos to xmm1")
	} else {
		return fmt.Errorf("EmitFloatOp not implemented for " + op.Name())
	}
	err := emitFloatOp(op, 32)
	emit("movq", "rax", xmm(1), "EmitF32Op: Move float result into rax")
	return err
}

// EmitCompareF64Const compares float in TOS with float constant
func EmitCompareF64Const(op Token, x float64) error {
	litNo := AddF64Lit(x)
	emit("movq", xmm(1), "rax", "")
	emit("mov", "rax", "[f64_"+strconv.Itoa(litNo)+"]", "Load float value from literal")
	emit("movq", xmm(2), "rax", "")
	emit("ucomisd", xmm(1), xmm(2), "Compare two F64 "+op.Name())
	return EmitJumpCond(op, true)
}

// EmitCompareF32Const compares float in TOS with float constant
func EmitCompareF32Const(op Token, x float32) (err error) {
	litNo := AddF32Lit(x)
	emit("movd", xmm(1), "eax", "")
	emit("mov", "eax", "[f32_"+strconv.Itoa(litNo)+"]", "Load float value from literal")
	emit("movd", xmm(2), "eax", "")
	emit("ucomiss", xmm(1), xmm(2), "Compare two F32 "+op.Name())
	err = EmitJumpCond(op, true)
	return err
}

// EmitCompareF64 compares two floats in TOS and NOS.
func EmitCompareF64(op Token) (err error) {
	emit("movq", xmm(2), "rax", "")
	EmitPopAx("")
	emit("movq", xmm(1), "rax", "")
	emit("ucomisd", xmm(1), xmm(2), "Compare two floats "+op.Name())
	err = EmitJumpCond(op, true)
	return err
}

// EmitCompareF32 compares two floats in TOS and NOS.
func EmitCompareF32(op Token) (err error) {
	emit("movd", xmm(2), "eax", "")
	EmitPopAx("")
	emit("movd", xmm(1), "eax", "")
	emit("ucomiss", xmm(1), xmm(2), "Compare two floats "+op.Name())
	err = EmitJumpCond(op, true)
	return err
}

func EmitLoadBool(value bool) {
	if value {
		emit("mov", "rax", "1", "")
	} else {
		emit("xor", "rax", "rax", "")
	}
	code.SetAx()
}

func EmitLoadGlobalVar(name string, pt code.PrimaryType) {
	// Todo : Use type to determine size to move
	emit("mov", "rax", "["+name+"]", "Load variable "+name)
	code.SetAx()
}

// ====================================================================================
//  ASSIGN
// ====================================================================================

// EmitAssignVariableExpressionInt will save the Top of Stack (AX) into a local variable of given size and offset.
// It will then clear RaxIsTos, effectively doing a pop
func EmitAssignVariableExpressionInt(op Token, size int, adr int, comment string) error {
	if op == TOK_MULT_ASGN {
		emit("imul", "rax", BpRel(adr), "")
		emit("mov", BpRel(adr), "rax", "")
		return nil
	} else if op == TOK_DIV_ASGN {
		emit("mov", "rcx", BpRel(adr), "")
		emit("cdq", "", "", "")
		emit("idiv", "ecx", "", "")
		emit("mov", BpRel(adr), "rax", "")
		return nil
	}
	EmitAssertTosInRax("")
	emit(TokenOp[op], BpRel(adr), AxName(size), "EmitAssignVariableExpressionInt "+comment)
	code.SetUndef()
	return nil
}

func EmitAssignVariableExpressionF64(op Token, adr int, comment string) error {
	if op == TOK_ASSIGN {
		EmitAssertTosInRax("")
		emit("mov", BpRel(adr), "rax", comment)
		return nil
	}
	emit("movq", "xmm2", "rax", comment)
	emit("mov", "rax", BpRel(adr), comment)
	emit("movq", "xmm1", "rax", comment)
	err := emitFloatOp(op, 64)
	emit("movq", "rax", "xmm1", "")
	emit("mov", BpRel(adr), "rax", comment)
	return err
}

func EmitAssignVariableExpressionF32(op Token, adr int, comment string) error {
	if op == TOK_ASSIGN {
		EmitAssertTosInRax("")
		emit("mov", BpRel(adr), "eax", comment)
		return nil
	}
	emit("movd", "xmm2", "eax", comment)
	emit("mov", "eax", BpRel(adr), comment)
	emit("movd", "xmm1", "eax", comment)
	err := emitFloatOp(op, 32)
	emit("movd", "eax", "xmm1", "")
	emit("mov", BpRel(adr), "eax", comment)
	return err
}

func EmitOpAssignIndirectConstF64(op Token, value float64) error {
	litNo := AddF64Lit(value)
	code.SetAx()
	emit("pop", "rdi", "", "EmitOpAssignIndirectF64Const")
	emit("mov", "rax", "[f64_"+strconv.Itoa(litNo)+"]", "")
	if op == TOK_ASSIGN {
		code.SetUndef()
		emit("mov", "[rdi]", "rax", "")
		return nil
	} else if op == TOK_PLUS_ASGN || op == TOK_MINUS_ASGN || op == TOK_DIV_ASGN || op == TOK_MULT_ASGN {
		emit("movq", xmm(2), "rax", "move tos in rax to xmm1")
		emit("mov", "rax", "[rdi]", "")
		emit("movq", xmm(1), "rax", "")
		err := emitFloatOp(op, 64)
		emit("movq", "rax", xmm(1), "EmitAssignF64ConstToLocal: Move float result into rax")
		emit("mov", "[rdi]", "rax", "")
		return err
	}
	return fmt.Errorf("float operation not implemented")
}

func EmitOpAssignIndirectConstF32(op Token, value float32) error {
	litNo := AddF32Lit(value)
	emit("pop", "rdi", "", "EmitOpAssignIndirectF32Const ")
	emit("mov", "eax", "dword [f32_"+strconv.Itoa(litNo)+"]", "")
	if op == TOK_ASSIGN {
		code.SetUndef()
		emit("mov", "dword [rdi]", "eax", "")
		return nil
	} else if op == TOK_PLUS_ASGN || op == TOK_MINUS_ASGN || op == TOK_DIV_ASGN || op == TOK_MULT_ASGN {
		emit("movd", xmm(2), "eax", "move tos in rax to xmm1")
		emit("mov", "eax", "dword [rdi]", "")
		emit("movd", xmm(1), "eax", "")
		err := emitFloatOp(op, 32)
		emit("movd", "eax", xmm(1), "EmitAssignF64ConstToLocal: Move float result into rax")
		emit("mov", "dword [rdi]", "eax", "")
		return err
	}
	return fmt.Errorf("float operation not implemented")
}

// EmitAssignIndirectExpressionInt has Pointer on stack, value in rax
func EmitAssignIndirectExpressionInt(op Token, size int) error {
	EmitAssertTosInRax("")
	emit("pop", "rsi", "", "Pop lvalue pointer into rsi")
	if op == TOK_MULT_ASGN {
		emit("imul", "rax", "[rsi]", "")
		emit("mov", "[rsi]", "rax", "")
		return nil
	} else if op == TOK_DIV_ASGN {
		emit("mov", "rcx", "rax", "")
		emit("mov", "rax", "[rsi]", "")
		emit("cdq", "", "", "")
		emit("idiv", "ecx", "", "")
		emit("mov", "[rsi]", "rax", "")
		return nil
	}
	if size == 8 {
		emit(TokenOp[op], "[rsi]", "rax", "EmitStoreIndirect quad")
	} else if size == 4 {
		emit(TokenOp[op], "dword [rsi]", "eax", "EmitStoreIndirect dword")
	} else if size == 2 {
		emit(TokenOp[op], "word [rsi]", "ax", "EmitStoreIndirect word")
	} else if size == 1 {
		emit(TokenOp[op], "byte [rsi]", "al", "EmitStoreIndirect byte")
	} else {
		return fmt.Errorf("store indirect with wrong size")
	}
	return nil
}

// EmitAssignIndirectConstInt assumes pointer in TOS and constant in parameter "value"
func EmitAssignIndirectConstInt(op Token, size int, value int64, comment string) error {
	EmitComment("EmitAssignIndirectConstInt")
	EmitFlushRax("")
	emit("pop", "rdi", "", "pop EmitAssignIndirectConstInt")
	instr := TokenOp[op]
	if instr == "" {
		return fmt.Errorf("EmitIntegerOp called with invalid token %s", op.Name())
	}
	if op == TOK_ASSIGN {
		emit("mov", DataType(size)+" [rdi]", strconv.Itoa(int(value)), "")
	} else {
		// Do a read modify write operation, f.e.x +=
		if size == 4 {
			emit("mov", "eax", "[rdi]", comment)
		} else if size == 8 {
			emit("mov", "rax", "[rdi]", "")
		} else if size == 1 {
			emit("mov", "al", "byte [rdi]", "")
		} else {
			return fmt.Errorf("%s not implemented for size %d", op.Name(), size)
		}
		if instr == "idiv" {
			emit("mov", "rcx", strconv.FormatInt(value, 10), "idiv load divisor")
			if size != 4 {
				return fmt.Errorf("only 32 bit integer divide currently supported")
			}
			emit("cdq", "", "", "")
			emit("idiv", "ecx", "", "")
		} else {
			emit(TokenOp[op], "rax", strconv.Itoa(int(value)), "Integer op other")
		}
		emit("mov", "[rdi]", AxName(size), "")
	}
	return nil
}

// EmitAssignIndirectExpressionF64 assumes pointer to F64 on stack and operand in rax
func EmitAssignIndirectExpressionF64(op Token) error {
	emit("pop", "rdi", "", "")
	if op == TOK_ASSIGN {
		code.SetUndef()
		emit("mov", "[rdi]", "rax", "")
		return nil
	} else if op == TOK_PLUS_ASGN || op == TOK_MINUS_ASGN || op == TOK_DIV_ASGN || op == TOK_MULT_ASGN {
		emit("movq", xmm(2), "rax", "move tos in rax to xmm1")
		emit("mov", "rax", "[rdi]", "")
		emit("movq", xmm(1), "rax", "")
		err := emitFloatOp(op, 64)
		emit("movq", "rax", xmm(1), "EmitAssignF64ConstToLocal: Move float result into rax")
		emit("mov", "[rdi]", "rax", "")
		return err
	}
	return fmt.Errorf("%s not implemented for indirect assign F64", op.Name())

}

func EmitAssignIndirectExpressionF32(op Token) error {
	emit("pop", "rdi", "", "")
	if op == TOK_ASSIGN {
		code.SetUndef()
		emit("mov", "dword [rdi]", "eax", "")
		return nil
	} else if op == TOK_PLUS_ASGN || op == TOK_MINUS_ASGN || op == TOK_DIV_ASGN || op == TOK_MULT_ASGN {
		emit("movd", xmm(2), "eax", "move tos in rax to xmm1")
		emit("mov", "eax", "dword [rdi]", "")
		emit("movd", xmm(1), "eax", "")
		err := emitFloatOp(op, 32)
		emit("movd", "eax", xmm(1), "EmitAssignTosF32ToIndirect: Move float result into rax")
		emit("mov", "[rdi]", "rax", "")
		return err
	}
	return fmt.Errorf("%s not implemented for indirect assign F64", op.Name())
}

// EmitAssignVariableConstF64 constant float value to variable
func EmitAssignVariableConstF64(op Token, adr int, x float64, comment string) error {
	if op == TOK_ASSIGN {
		litNo := AddF64Lit(x)
		code.SetAx()
		emit("mov", "rax", "[f64_"+strconv.Itoa(litNo)+"]", comment)
		code.SetUndef()
		emit("mov", BpRel(adr), "rax", "")
		return nil
	} else if op == TOK_PLUS_ASGN || op == TOK_MINUS_ASGN || op == TOK_DIV_ASGN || op == TOK_MULT_ASGN {
		litNo := AddF64Lit(x)
		code.SetAx()
		emit("mov", "rax", BpRel(adr), comment)
		emit("movq", xmm(1), "rax", "move tos in rax to xmm1")
		emit("mov", "rax", "[f64_"+strconv.Itoa(litNo)+"]", "")
		emit("movq", xmm(2), "rax", "mov nos to xmm2")
		err := emitFloatOp(op, 64)
		emit("movq", "rax", xmm(1), "EmitAssignF64ConstToLocal: Move float result into rax")
		emit("mov", BpRel(adr), "rax", "")
		return err
	}
	return fmt.Errorf("type F64 assign operation %s not implemented", op.Name())
}

// EmitAssignVariableConstF32 constant float value to variable
func EmitAssignVariableConstF32(op Token, adr int, x float32, comment string) error {
	if op == TOK_ASSIGN {
		litNo := AddF32Lit(x)
		code.SetAx()
		emit("mov", "rax", "[f32_"+strconv.Itoa(litNo)+"]", comment)
		code.SetUndef()
		emit("mov", BpRel(adr), "rax", "")
		return nil
	} else if op == TOK_PLUS_ASGN || op == TOK_MINUS_ASGN || op == TOK_DIV_ASGN || op == TOK_MULT_ASGN {
		litNo := AddF32Lit(x)
		code.SetAx()
		emit("mov", "eax", "dword "+BpRel(adr), comment)
		emit("movd", xmm(1), "eax", "move tos in rax to xmm1")
		emit("mov", "eax", "dword [f32_"+strconv.Itoa(litNo)+"]", "")
		emit("movd", xmm(2), "eax", "mov nos to xmm2")
		err := emitFloatOp(op, 32)
		emit("movd", "eax", xmm(1), "EmitAssignF64ConstToLocal: Move float result into rax")
		emit("mov", "dword "+BpRel(adr), "eax", "")
		return err
	}
	return fmt.Errorf("type F32 assign operation %s not implemented", op.Name())
}

// EmitAssignVariableConstInt will set variable at <adr> to <adr> op <value>
func EmitAssignVariableConstInt(op Token, adr int, size int, value int64, comment string) error {
	instr := TokenOp[op]
	if instr == "" {
		return fmt.Errorf("EmitOpAssign called with invalid token %s", op.Name())
	}
	if instr == "idiv" {
		emit("mov", "rcx", strconv.FormatInt(value, 10), "idiv load divisor")
		if size == 4 {
			emit("mov", "eax", DataType(size)+BpRel(adr), comment)
		} else {
			return fmt.Errorf("only 32 bit integer divide currently supported")
		}
		emit("cdq", "", "", "")
		emit("idiv", "ecx", "", "")
		// Move result to local variable at BpRel(adr)
		emit("mov", DataType(size)+BpRel(adr), AxName(size), "move result of *= to local variable")
	} else if instr == "imul" {
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

// EmitAssignVariableExpressionStruct assigns rax to the variable given and frees old variable contents.
func EmitAssignVariableExpressionStruct(op Token, size int, adr int, comment string) error {
	EmitFlushRax("")
	// Check for existing struct - free it if needed
	emit("mov", "rax", BpRel(adr), "EmitAssignVariableExpressionStruct, Get old value")
	emit("or", "rax", "rax", "")
	lbl := code.NewLabel()
	emit("jz", Label(lbl), "", "")
	// Now free old struct
	EmitFreeStruct(size, "")
	EmitLabel(lbl, "")
	emit("pop", "rax", "", "")
	emit(TokenOp[op], BpRel(adr), "rax", "EmitStoreToLocal "+comment)
	code.SetUndef()
	return nil
}

// EmitConcat will concatenate the two strings at the top of the stack
// First string pointer in [rsp], second string pointer in rax
// It uses registers r12, r13, r14, rbx, rcx, rdx, rsi, rdi.
// Calls _alloc to allocate a new string with size for both the input strings + 32 bytes extra.
// This works fine
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
	emit("push", "rax", "", "Save pointer on stack for later use")
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
		lbl := code.NewLabel()
		emit("mov", "rax", "[rbx]", "Free first argument to Concatenate")
		emit("shr", "rax", "32", "")
		emit("or", "rax", "rax", "")
		emit("jz", Label(lbl), "", "")
		emit("mov", "rax", "rbx", "Free first argument to Concatenate")
		emit("call", "_free_str", "", "")
		EmitLabel(lbl, "")
	}
	if free2 {
		lbl := code.NewLabel()
		emit("mov", "rax", "[r13]", "Free first argument to Concatenate")
		emit("shr", "rax", "32", "")
		emit("or", "rax", "rax", "")
		emit("jz", Label(lbl), "", "")
		emit("mov", "rax", "r13", "Free second argument to Concatenate")
		emit("call", "_free_str", "", "")
		EmitLabel(lbl, "")
	}

	// Copy the allocated buffer address from r9 to rax. Now rax points to the new string.
	EmitPopAx("Now AX should point to the string")
	// Remove the top of stack. New TOS is the pointer in rax. Arguments in rbx and r13.
	emit("add", "rsp", "8", "Remove the top of stack. New TOS is the pointer in rax")
}

// ExtendStringCapacity of a string by copying it into new memory (uses r12)
// * ebx should contain the required extra length, f.ex. the length of the appended string
// * rsi should point to the old string, so [rsi] is the old len/cap
// Extend if old length + required extra > old capacity  (low([rsi])+rbx > [rsi]>>32
// The new capacity will be <ebx> + <old len> + <bytesExtra> (or possibly (ebx+oldcap)*2)
// * At exit, rdi points to the first empty character of the new string (ready for move)
// * At exit, rdx points to the extended string's len/cap or the old string's len/cap
func ExtendStringCapacity(bytesExtra int) {
	lbl1 := code.NewLabel()
	lbl2 := code.NewLabel()
	lbl3 := code.NewLabel()
	emit("push", "rsi", "", "")
	// Check if old string was nil.
	emit("mov", "rax", "rbx", "")
	emit("or", "rsi", "rsi", "")
	emit("jz", Label(lbl2), "", "")
	// Check if old len + new len (rbx) is more than old cap (rcx)
	emit("mov", "rax", "[rsi]", "Start ExtendStringCapacity, load old len/cap")
	emit("mov", "rcx", "rax", "Old len/cap into rcx")
	emit("shr", "rcx", "32", "Get only old cap in rcx")
	emit("mov", "eax", "eax", "Clear cap, leave old length in rax")
	emit("mov", "rdi", "rsi", "")
	emit("add", "rdi", "rax", "")
	emit("add", "rdi", "8", "")
	emit("add", "rax", "rbx", "Add extra length to old length")
	emit("cmp", "rax", "rcx", "Compare len to cap")
	emit("jb", Label(lbl1), "", "jump if we have enough space")
	// Extend capacity, including extra bytes.
	EmitLabel(lbl2, "")
	emit("add", "rax", strconv.Itoa(bytesExtra+16), "Add extra and space for len/cap")
	emit("mov", "r12", "rax", "Save new cap in r12")
	emit("shl", "r12", "32", "Save new cap in correct half of len/cap")
	emit("add", "rax", "8", "Allocate 8 bytes more than capacity, to store len/cap")
	emit("call", "_alloc", "", "Allocate new string")
	// Copy old string
	emit("mov", "rdi", "rax", "Pointer to new string")
	emit("mov", "rdx", "rax", "Save pointer to new string")
	emit("xor", "rcx", "rcx", "")
	emit("add", "rdi", "8", "Skip len/cap when moving string")
	emit("or", "rsi", "rsi", "")
	emit("jz", Label(lbl3), "", "")
	emit("mov", "rcx", "[rsi]", "Get old length")
	EmitLabel(lbl3, "")
	emit("mov", "ecx", "ecx", "Clear cap, leave old length in rcx")
	emit("add", "r12", "rcx", "Add old length to len/cap in r12")
	emit("mov", "rbx", "rsi", "Save pointer to old string in order to free it if needed")
	emit("add", "rsi", "8", "Skip len/cap when moving string")
	emit("cld", "", "", "")
	emit("rep", "movsb", "", "copy old string")
	// Update new string
	emit("mov", "[rdx]", "r12", "Mov new len/cap into string")
	emit("mov", "rsi", "rdx", "rdx now points to the new string's len/cap")
	// Free old string
	emit("mov", "[rsp]", "rdx", "")
	emit("mov", "rax", "rbx", "rbx points to the old string")
	emit("call", "_free_str", "", "")
	EmitLabel(lbl1, "End of ExtendStringCapacity")
	emit("pop", "rdx", "", "")
}

// =======   APPEND STR-STR ===========

// EmitAppendVariableExpressionStrStr appends the string on stack to the variable at adr.
// Ok
func EmitAppendVariableExpressionStrStr(adr int) error {
	emit("mov", "rbx", BpRel(adr), "Get pointer to first part")
	emit("push", "rbx", "", "and save it to stack")
	emit("mov", "r13", "rax", "Save second part to r13")
	// Set bx to the appended length (on stack)
	emit("mov", "rbx", "[r13]", "Get len/cap of second part")
	emit("mov", "ebx", "ebx", "Clear capacity. ")
	emit("mov", "r14", "rbx", "Save length of second part")
	// Set si to point to len/cap of string to be possibly extended
	emit("mov", "rsi", "[rsp]", "")
	ExtendStringCapacity(4)
	// rdi points to the first empty character of the new string (ready for move)
	// rdx points to the extended string's len/cap or the old string's len/cap
	emit("add", "[rdx]", "r14", "Add length of second part to length/cap of first part")
	emit("mov", "rcx", "r14", "Get length of second part")
	emit("mov", "ecx", "ecx", "Clear cap, added length in rcx")
	emit("mov", "rsi", "r13", "Get appended string")
	emit("add", "rsi", "8", "")
	emit("rep", "movsb", "", "copy appended string 1")
	// Now update local variable
	emit("mov", BpRel(adr), "rdx", "")
	emit("pop", "rax", "", "")
	return nil
}

// EmitAppendIndirectExpressionStrStr appends string in rax (second part) to  [rsp] (first part)
// OK
func EmitAppendIndirectExpressionStrStr() error {
	// Set si to point to len/cap of string to be possibly extended
	EmitAssertTosInRax("")
	EmitComment(">> EmitAppendIndirectExpressionStrStr")
	emit("mov", "rsi", "[rsp]", "First part")
	emit("mov", "rsi", "[rsi]", "Get string len/cap pointer for first part")
	emit("mov", "r13", "rax", "Save second part to r13")
	emit("mov", "rbx", "[r13]", "Get len/cap of second part")
	emit("mov", "ebx", "ebx", "Clear capacity. Ready to extend.")
	emit("mov", "r14", "rbx", "Save length of second part")
	ExtendStringCapacity(4)
	// rdi points to the first empty character of the new string (ready for move)
	// rdx points to the extended string's len/cap or the old string's len/cap
	emit("add", "[rsi]", "r14", "Add length of second part to length/cap of first part")
	emit("mov", "rcx", "r14", "Get length of second part")
	emit("mov", "ecx", "ecx", "Clear cap, added length in rcx")
	emit("mov", "rsi", "r13", "Get appended string")
	emit("add", "rsi", "8", "")
	emit("rep", "movsb", "", "copy appended string 2")
	// Now update indirect variable
	emit("mov", "rdi", "[rsp]", "")
	emit("mov", "qword [rdi]", "rdx", "")
	emit("pop", "rax", "", "")
	return nil
}

// EmitAppendIndirectConstStrStr appends a constant character value to string in NOS.
func EmitAppendIndirectConstStrStr(strLitNo int) error {
	emit("mov", "rsi", "[rsp]", "Load pointer to string (EmitAppendIndirectConstStrStr)")
	emit("mov", "rsi", "[rsi]", "Get string len/cap pointer")
	emit("mov", "rax", "str"+strconv.Itoa(strLitNo), "")
	emit("mov", "rbx", "[rax]", "Get part 2 len/cap")
	emit("mov", "ebx", "ebx", "Clear upper 32 bits - keep length")
	emit("mov", "r14", "rbx", "Save length of second part")
	ExtendStringCapacity(4)
	// rdi points to the first empty character of the new string (ready for move)
	// rdx points to the extended string's len/cap or the old string's len/cap
	emit("add", "[rsi]", "r14", "Add length of second part to length/cap of first part")
	emit("mov", "rcx", "r14", "Get length of second part")
	emit("mov", "ecx", "ecx", "Clear cap, added length in rcx")
	emit("mov", "rsi", "str"+strconv.Itoa(strLitNo), "")
	emit("add", "rsi", "8", "")
	emit("rep", "movsb", "", "copy appended string 3")
	// Now update indirect variable
	emit("mov", "rdi", "[rsp]", "")
	emit("mov", "qword [rdi]", "rdx", "")
	emit("pop", "rax", "", "")
	return nil
}

func EmitAppendVariableConstStrStr(adr int, strLitNo int) error {
	// EmitConcat will concatenate the two strings at the top of the stack
	emit("mov", "rax", BpRel(adr), "")
	emit("push", "rax", "", "")
	emit("mov", "rax", "str"+strconv.Itoa(strLitNo), "")
	emit("push", "rax", "", "")
	code.SetSp()
	EmitConcat(true, false)
	emit("mov", BpRel(adr), "rax", "")
	return nil
}

// =======   ASSIGN STR-STR ===========

// EmitAssignIndirectConstStrStr ok
func EmitAssignIndirectConstStrStr(strLitNo int) error {
	EmitAssertTosInRax("")
	// Free old string in [rax] if it exists
	lbl := code.NewLabel()
	emit("mov", "rbx", "[rax]", "Free existing in EmitAssignIndirectConstStrStr")
	emit("mov", "r14", "rax", "")
	emit("or", "rbx", "rbx", "")
	emit("jz", Label(lbl), "", "")
	emit("mov", "rbx", "[rbx]", "")
	emit("shr", "rbx", "32", "")
	emit("or", "rbx", "rbx", "")
	emit("jz", Label(lbl), "", "")
	emit("mov", "rax", "[rax]", "")
	emit("call", "_free_str", "", "")
	EmitLabel(lbl, "")

	emit("mov", "rbx", "str"+strconv.Itoa(strLitNo), "")
	emit("mov", "[r14]", "rbx", "")
	return nil
}

// EmitAssignVariableConstStrStr will append a litteral string to the tstring in local variabl at adr
// The old string may be replaced with a bigger string if needed.
func EmitAssignVariableConstStrStr(adr int, strLitNo int) error {
	code.SetAx()
	// Check if we must free old variable
	emit("mov", "rax", BpRel(adr), "EmitAssignVariableConstStrStr")
	emit("call", "_free_str", "", "")
	emit("mov", "rax", "str"+strconv.Itoa(strLitNo), "")
	emit("mov", BpRel(adr), "rax", "")
	return nil
}

// EmitAssignIndirectExpressionStrStr assigns  string in  [rax] (right side) to string pointed to by [rsp] (left side)
func EmitAssignIndirectExpressionStrStr() error {
	// Free old string in [rax] if it exists
	EmitAssertTosInRax("")
	emit("mov", "r12", "rax", "")
	lbl := code.NewLabel()
	EmitComment("EmitAssignIndirectExpressionStrStr")
	emit("mov", "rbx", "[rsp]", "Free existing string pointed to by indirect expression if needed.")
	emit("or", "rbx", "rbx", "")
	emit("jz", Label(lbl), "", "")
	emit("mov", "rbx", "[rbx]", "")
	emit("or", "rbx", "rbx", "")
	emit("jz", Label(lbl), "", "")
	emit("mov", "rdi", "rbx", "Save pointer to string that might be freed")
	emit("mov", "rbx", "[rbx]", "Now rbx should be len/cap")
	emit("shr", "rbx", "32", "")
	emit("or", "rbx", "rbx", "")
	emit("jz", Label(lbl), "", "")
	emit("mov", "rax", "rdi", "")
	emit("call", "_free_str", "", "")
	EmitLabel(lbl, "")

	emit("mov", "rdi", "[rsp]", "Get indirect pointer")
	emit("mov", "qword [rdi]", "r12", "Save new string")
	emit("pop", "rax", "", "")
	return nil
}

// EmitAssignVariableExpressionStrStr assigns string at rax to variable at adr
func EmitAssignVariableExpressionStrStr(adr int) error {
	EmitAssertTosInRax("")
	emit("mov", BpRel(adr), "rax", "")
	return nil
}

// ========== APPEND STR-CHAR ===========

func EmitAppendIndirectConstStrChar(value int) error {
	if value > 128 {
		return fmt.Errorf("only ascii values <128 is supported for now")
	}
	emit("mov", "rsi", "[rsp]", "Load indirect pointer (EmitAppendIndirectConstStrChar)")
	emit("mov", "rsi", "[rsi]", "Load string pointer")
	emit("mov", "rbx", "1", "Load needed extra space")
	// Now ebx should contain the required extra length (1) and rsi should point to the old string, so [rsi] is the old len/cap
	ExtendStringCapacity(4)
	// rdi points to the first empty character of the new string (ready for move)
	// rdx points to the extended string's len/cap or the old string's len/cap
	emit("inc", "dword [rdx]", "", "Incr original length by one")
	emit("mov", "byte [rdi]", strconv.Itoa(value), "Append character")
	// Now update indirect variable
	emit("mov", "rdi", "[rsp]", "")
	emit("mov", "qword [rdi]", "rdx", "")
	emit("pop", "rax", "", "")
	return nil
}

func EmitAssignIndirectExpressionStrChar() error {
	return fmt.Errorf("EmitAssignIndirectExpressionStrChar not implemented")
}

// EmitAssignVariableConstStrChar will append a litteral character to a string in a variable.
func EmitAssignVariableConstStrChar(op Token, adr int, value int) error {
	if value > 128 {
		return fmt.Errorf("only ascii values <128 is supported for now")
	}
	if op != TOK_PLUS_ASGN {
		return fmt.Errorf("only += supported for string += int")
	}
	emit("mov", "rdi", BpRel(adr), "Load pointer to string from local variable")
	emit("mov", "rsi", "rdi", "save copy of pointer")
	emit("mov", "rbx", "1", "Load needed extra space")
	// Now ebx should contain the required extra length (1) and rsi should point to the old string, so [rsi] is the old len/cap
	ExtendStringCapacity(4)
	// rdi points to the first empty character of the new string (ready for move)
	// rdx points to the extended string's len/cap or the old string's len/cap
	emit("inc", "dword [rdx]", "", "Incr original length by one")
	emit("mov", "byte [rdi]", strconv.Itoa(value), "Append character")
	// Update variable
	emit("mov", BpRel(adr), "rdx", "")
	return nil
}

// EmitAppendVariableExpressionStrChar appends a character in rax to the string in the variable at <adr>
func EmitAppendVariableExpressionStrChar(adr int) error {
	emit("mov", "r14", "rax", "Save character value")
	emit("mov", "rdi", BpRel(adr), "Load pointer to string from local variable")
	emit("mov", "rsi", "rdi", "save copy of pointer")
	emit("mov", "rax", "[rdi]", "Load len/cap")
	emit("shr", "rax", "32", "Get cap")
	emit("mov", "rbx", "[rdi]", "Load len/cap")
	emit("mov", "ebx", "ebx", "Clear upper 32 bits - keep length")
	emit("cmp", "rbx", "rax", "Compare len to cap")
	lbl := code.NewLabel()
	emit("jnz", Label(lbl), "", "jump if we have enought space")
	// Extend capacity, including 64 extra bytes
	emit("mov", "r13", "rax", "Old len")
	emit("add", "rax", "64", "Add 64+8 to include len/cap")
	emit("mov", "r12", "rax", "")
	emit("add", "rax", "8", "")
	emit("call", "_alloc", "", "Allocate new string")
	// Save new string pointer
	emit("mov", BpRel(adr), "rax", "")
	EmitLabel(lbl, "")
	emit("add", "rdi", "rbx", "Add length to pointer - we will save to end of string")
	emit("add", "rdi", "8", "Skip len/cap also")
	// TODO Handle longer characters (UTF)
	emit("mov", "rax", "r14", "Char to rax")
	emit("mov", "byte [rdi]", "al", "Add char to string")
	emit("inc", "qword [rsi]", "", "")
	return nil
}
