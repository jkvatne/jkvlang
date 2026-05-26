package code

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	LabelNo     int
	RaxIsTOS    bool
	LocalSp     int
	LineNum     int
	UnitName    string
	OutputFile  *os.File // File where the assembly is put
	ArgCode     []string // Temporary storage of assembly code. needed because we evaluate arguments in reverse order
	CleanupCode []string
)

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
	// _, _ = Write(s, ArgCode[0], true)
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

func EmitLineNo(currentLine string, localSp int) {
	Write("\n   ; Line " + strconv.Itoa(LineNum) + " " + strings.Trim(currentLine, "\r\n") + "  (SP=" + strconv.Itoa(localSp) + ")\n")
}

func EmitBlankLine() {
	Write("\n")
}
