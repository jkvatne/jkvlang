package code

import (
	"go/constant"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type stackState uint8

const (
	undef stackState = iota
	sp
	ax
	cc
)

type PrimaryType int

//goland:noinspection ALL,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage
const (
	TYP_NONE PrimaryType = iota
	TYP_BOOL
	TYP_U8
	TYP_I16
	TYP_U16

	TYP_I32
	TYP_U32
	TYP_RUNE
	TYP_I64
	TYP_U64

	TYP_F32
	TYP_F64
	TYP_STRING
	TYP_STRUCT
	TYP_FUNC

	TYP_MAP
	TYP_SET
	TYP_PTR
	TYP_ERROR
	TYP_SLICE

	TYP_COUNT
)

var PrimaryTypeNames = [...]string{
	"None", "Bool", "U8", "I16", "U16",
	"I32", "U32", "Rune", "I64", "U64",
	"F32", "F64", "String", "Struct", "Func",
	"Map", "Set", "Ptr", "Error", "Slice",
	"<unused>"}

var PrimaryTypeSizes = [...]int{
	0, 1, 1, 2, 2,
	4, 4, 4, 8, 8,
	4, 8, 8, 8, 8,
	8, 8, 8, 8, 8,
	0}

type Value struct {
	Bits        uint64
	Pt          PrimaryType
	StringValue string
}

var (
	constValue  constant.Value
	state       stackState
	LabelNo     int
	LocalSp     int
	LineNum     int
	UnitName    string
	OutputFile  *os.File // File where the assembly is put
	ArgCode     []string // Temporary storage of assembly code. needed because we evaluate arguments in reverse order
	CleanupCode []string
)

func (t PrimaryType) IsObject() bool {
	return t == TYP_STRUCT || t == TYP_STRING || t == TYP_MAP || t == TYP_SET
}

func (t PrimaryType) Name() string {
	return PrimaryTypeNames[t]
}

func (t PrimaryType) Size() int {
	return PrimaryTypeSizes[t]
}

func (t PrimaryType) IsInteger() bool {
	return t == TYP_I32 || t == TYP_U32 || t == TYP_U16 || t == TYP_I16 || t == TYP_U8 || t == TYP_I64 || t == TYP_U64
}

func (t PrimaryType) IsUnsigned() bool {
	return t == TYP_U32 || t == TYP_U16 || t == TYP_U8 || t == TYP_U64
}

func (t PrimaryType) IsFloat() bool {
	return t == TYP_F32 || t == TYP_F64
}

func (t PrimaryType) IsNumber() bool {
	return t.IsFloat() || t.IsInteger()
}

func StackState() string {
	if state == ax {
		return "ax"
	} else if state == sp {
		return "sp"
	} else {
		return "??"
	}
}

func SetAx() {
	state = ax
}

func SetSp() {
	state = sp
}

func SetUndef() {
	state = undef
}

func AxIsTos() bool {
	return state == ax
}

func SpIsTos() bool {
	return state == sp
}

func New(name string, workdir string) (err error) {
	ArgCode = make([]string, 0, 64)
	CleanupCode = make([]string, 0, 64)
	UnitName = strings.TrimSuffix(filepath.Base(name), ".jkv")
	fn := filepath.Join(workdir, UnitName+".asm")
	OutputFile, err = os.Create(fn)
	LineNum = 1
	return err
}

func NewLabel() int {
	LabelNo++
	return LabelNo
}
func CloseObjFile() error {
	return OutputFile.Close()
}

func NewArgCode() {
	ArgCode = append(ArgCode, "")
}

func PushCleanupCode() {
	CleanupCode = append(CleanupCode, "")
}

func SetCleanupCode(txt string) {
	if txt != "" {
		CleanupCode[len(CleanupCode)-1] = txt
	}
}

func OutputCleanupCode(n int) {
	na := len(ArgCode) - 1
	nc := len(CleanupCode) - 1
	if nc+1 < n {
		panic("CleanupCode error")
	}
	for ; n > 0; n-- {
		if CleanupCode[nc] != "" {
			ArgCode[na] = ArgCode[na] + CleanupCode[nc]
		}
		nc--
	}
	CleanupCode = CleanupCode[0 : len(CleanupCode)-n]
}

func ConsArgCode(count int, reverse bool) {
	if count == 0 {
		return
	}
	txt := ""
	startArgNo := len(ArgCode) - count
	if startArgNo < 0 {
		panic("ArgCode error")
	}
	if reverse {
		for i := len(ArgCode) - 1; i >= startArgNo; i-- {
			txt += ArgCode[i]
		}
	} else {
		for i := startArgNo; i < len(ArgCode); i++ {
			txt += ArgCode[i]
		}
	}
	if len(ArgCode) > startArgNo {
		ArgCode[startArgNo] = txt
	}
	if len(ArgCode) > startArgNo {
		ArgCode = ArgCode[0 : startArgNo+1]
	}
}

func OutputArgCode() {
	if len(ArgCode) > 1 {
		panic("Line " + strconv.Itoa(LineNum) + ": OutputArgCode should have only one entry in ArgCode")
	}
	if len(ArgCode) == 0 {
		return
	}
	_, _ = OutputFile.WriteString(ArgCode[0])
	ArgCode = nil
}

func Write(txt string) int {
	if len(ArgCode) == 0 {
		// Write directly to file
		n, err := OutputFile.WriteString(txt)
		if err != nil {
			panic("Could not write to file " + OutputFile.Name() + ": " + err.Error())
		}
		return n
	}

	// When parsing an argument, output text to the last element in the ArgCode slice
	ArgCode[len(ArgCode)-1] += txt
	return len(txt)
}

func EmitLineNo(currentLine string) {
	Write("\n   ; Line " + strconv.Itoa(LineNum) + " " + strings.Trim(currentLine, "\r\n") + "\n")
}

func EmitBlankLine() {
	Write("\n")
}
