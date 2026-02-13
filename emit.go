package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
)

var emitPath string

func EmitTo(s *state, path string) error {
	var err error
	emitPath = path
	s.outputFile, err = os.OpenFile(s.unitName+".tok", os.O_CREATE, os.ModePerm)
	s.outputFile.WriteString("Token file\n")
	return err
}

func EmitStop(s *state) error {
	return s.outputFile.Close()
}

func emit(s *state, opcode string, value string) {
	if !*noCode {
		fmt.Printf("%s %s\n", opcode, value)
	}
	s.outputFile.WriteString(fmt.Sprintf("%s %s\n", opcode, value))
}

func EmitStore(s *state, id string, typ string) {
	slog.Info("Pop stack and store value into", "name", id)
	emit(s, "   STORE_"+typ, id)
}

func EmitLoad(s *state, id string, typ string) {
	slog.Info("Emit load", "name", id)
	emit(s, "   LOAD_"+typ, id)
}

func EmitCall(s *state, id string) {
	slog.Info("Emit call", "name", id)
	emit(s, "   CALL", id)
}

func EmitLabel(s *state, n int) {
	slog.Info("EmitLabel", "no", n)
	emit(s, "L"+strconv.Itoa(n), ":")
}

func EmitFunction(s *state, id string) {
	slog.Info("EmitFunction")
	emit(s, id, ":  // Line "+strconv.Itoa(s.lineNum))
	emit(s, "   PROLOG", "")
}

func EmitJump(s *state, n int) {
	slog.Info("EmitJump", "no", n)
	emit(s, "   JUMP", "L"+strconv.Itoa(n))
}

func EmitJumpFalse(s *state, n int) {
	slog.Info("EmitJump", "no", n)
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
	slog.Info("EmitModify", "id", id, "op", op)
	emit(s, "   "+TokenNames[op], id+" "+value)
}

func EmitType(s *state, name string, typ int) {
	slog.Info("Type "+name, strconv.Itoa(typ))
}

func EmitVar(s *state, name string, value string, typ string) {
	slog.Info("Var:"+name+" value:\""+value+"\" Func:\""+s.currentFunc+"\" Type:"+typ, "")
}

func EmitConst(s *state, name string, value string, typ string) {
	emit(s, "Const:"+name+"="+value+" Func:"+s.currentFunc+" Type:"+typ, "")
}
