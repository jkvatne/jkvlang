package main

import (
	"fmt"
	"log/slog"
	"strconv"
)

func CloseObjFile(s *State) error {
	return s.outputFile.Close()
}

func emit(s *State, opcode string, value string) {
	_, err := s.outputFile.WriteString(fmt.Sprintf("%s %s\n", opcode, value))
	if err != nil {
		panic(err)
	}
}

func EmitStore(s *State, id string, typ string) {
	slog.Info(No(s)+" EmitStore: ", "name", id)
	emit(s, "   STORE_"+typ, id)
}

func EmitLoad(s *State, id string, typ string) {
	slog.Info(No(s)+" EmitLoad: ", "name", id)
	emit(s, "   LOAD_"+typ, id)
}

func EmitAssert(s *State) {
	slog.Info(No(s) + " EmitAssert")
	emit(s, "   ASSERT", "")
}

func EmitCall(s *State, id string) {
	slog.Info(No(s)+" EmitCall:", "name", id)
	emit(s, "   CALL", id)
}

func EmitLabel(s *State, n int) {
	slog.Info(No(s)+" EmitLabel: ", "no", n)
	emit(s, "L"+strconv.Itoa(n), ":")
}

func EmitFunction(s *State, id string) {
	slog.Info(No(s) + " EmitFunction")
	emit(s, id, "")
	emit(s, "   PROLOG", "")
}

func EmitJump(s *State, n int) {
	slog.Info(No(s)+" EmitJump", "no", n)
	emit(s, "   JUMP", "L"+strconv.Itoa(n))
}

func EmitJumpFalse(s *State, n int) {
	slog.Info(No(s)+" EmitJumpFalse", "no", n)
	emit(s, "   JUMPFALSE", "L"+strconv.Itoa(n))
}

func EmitReturn(s *State) {
	slog.Info(No(s) + " EmitReturn")
	emit(s, "   RETURN", "\n")
}

func EmitModify(s *State, id string, op Token, value string) {
	slog.Info(No(s)+" EmitModify: ", "id", id, "op", op)
	emit(s, "   "+TokenNames[op], id+" "+value)
}

func EmitType(s *State, name string, typ int) {
	slog.Info(No(s)+" EmitType: "+name, strconv.Itoa(typ))
}

func EmitVar(s *State, name string, value string, typ string) {
	slog.Info(No(s) + " EmitVar: " + name + " value:\"" + value + "\" Func:\"" + s.currentFunc + "\" Type:" + typ)
}

func EmitConst(s *State, name string, value string, typ string) {
	slog.Info(No(s) + " EmitConst: " + name + " value:\"" + value + "\" Type:" + typ)
	emit(s, "EmitConst: "+name+"="+value+" Func:"+s.currentFunc+" Type:"+typ, "")
}

func EmitLineNo(s *State) {
	emit(s, " // Line no", strconv.Itoa(s.lineNum))
}

func EmitOp(s *State, op Token) {
	emit(s, "   "+TokenNames[op], "")
}

func EmitError(s *State, err error) {
	emit(s, "Error on line "+strconv.Itoa(s.lineNum)+": ", err.Error())
}
