package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
)

func EmitTo(s *state, workDir string) error {
	fn := filepath.Join(workDir, s.unitName+".tok")
	var err error
	s.outputFile, err = os.OpenFile(fn, os.O_CREATE|os.O_TRUNC, os.ModePerm)
	emit(s, "// Token file ", fn)
	return err
}

func EmitClose(s *state) error {
	return s.outputFile.Close()
}

func emit(s *state, opcode string, value string) {
	//	fmt.Printf("%s %s\n", opcode, value)
	s.outputFile.WriteString(fmt.Sprintf("%s %s\n", opcode, value))
}

func EmitStore(s *state, id string, typ string) {
	slog.Info("EmitStore: ", "name", id)
	emit(s, "   STORE_"+typ, id)
}

func EmitLoad(s *state, id string, typ string) {
	slog.Info("EmitLoad: ", "name", id)
	emit(s, "   LOAD_"+typ, id)
}

func EmitAssert(s *state) {
	slog.Info("EmitAssert")
	emit(s, "   ASSERT", "")
}

func EmitCall(s *state, id string) {
	slog.Info("EmitCall:", "name", id)
	emit(s, "   CALL", id)
}

func EmitLabel(s *state, n int) {
	slog.Info("EmitLabel: ", "no", n)
	emit(s, "L"+strconv.Itoa(n), ":")
}

func EmitFunction(s *state, id string) {
	slog.Info("EmitFunction")
	emit(s, id, "")
	emit(s, "   PROLOG", "")
}

func EmitJump(s *state, n int) {
	slog.Info("EmitJump", "no", n)
	emit(s, "   JUMP", "L"+strconv.Itoa(n))
}

func EmitJumpFalse(s *state, n int) {
	slog.Info("EmitJumpFalse", "no", n)
	emit(s, "   JUMPFALSE", "L"+strconv.Itoa(n))
}

func EmitReturn(s *state) {
	slog.Info("EmitReturn")
	emit(s, "   RETURN", "\n")
}

func EmitExit(s *state) {
	slog.Info("EmitExit")
	emit(s, "   EXIT", "")
}

func EmitModify(s *state, id string, op Token, value string) {
	slog.Info("EmitModify: ", "id", id, "op", op)
	emit(s, "   "+TokenNames[op], id+" "+value)
}

func EmitType(s *state, name string, typ int) {
	slog.Info("EmitType: "+name, strconv.Itoa(typ))
}

func EmitVar(s *state, name string, value string, typ string) {
	slog.Info("EmitVar: " + name + " value:\"" + value + "\" Func:\"" + s.currentFunc + "\" Type:" + typ)
}

func EmitConst(s *state, name string, value string, typ string) {
	emit(s, "EmitConst: "+name+"="+value+" Func:"+s.currentFunc+" Type:"+typ, "")
}

func EmitLineNo(s *state) {
	emit(s, " // Line no", strconv.Itoa(s.lineNum))
}

func EmitOp(s *state, op Token) {
	emit(s, "   "+TokenNames[op], "")
}
