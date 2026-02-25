package main

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
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

func EmitPush(s *State, id string, typ string) {
	slog.Info(No(s)+" EmitPush: ", "name", id)
	typ = strings.ToUpper(typ)
	emit(s, "   PUSH_"+typ, id)
}

func EmitAssert(s *State) {
	slog.Info(No(s) + " EmitAssert")
	emit(s, "   ASSERT", "")
}

func EmitCall(s *State, id string, argNo int) {
	slog.Info(No(s)+" EmitCall:", "name", id, "argNo", argNo)
	emit(s, "   CALL", id)
	emit(s, "   ADD SP, ", strconv.Itoa(argNo)+"  // Remove arguments from stack")
}

func EmitLabel(s *State, n int) {
	slog.Info(No(s)+" EmitLabel: ", "no", n)
	emit(s, "L"+strconv.Itoa(n), ":")
}

func EmitFunction(s *State, id string) {
	slog.Info(No(s) + " EmitFunction")
	emit(s, id, "")
	emit(s, "   PUSH FP", "")
	emit(s, "   SET FP=SP", "")
}

func EmitJump(s *State, n int) {
	if s.hasReturned {
		return
	}
	slog.Info(No(s)+" EmitJump", "no", n)
	emit(s, "   JUMP", "L"+strconv.Itoa(n))
}

func EmitJumpFalse(s *State, n int) {
	slog.Info(No(s)+" EmitJumpFalse", "no", n)
	emit(s, "   JUMPFALSE", "L"+strconv.Itoa(n))
}

func EmitReturn(s *State) {
	for i := range len(s.currentFunc.returnTypes) {
		emit(s, "   POP", "[BP-"+strconv.Itoa(len(s.currentFunc.argTypes)+i)+"]  // Return value")
	}
	slog.Info(No(s) + " EmitReturn")
	emit(s, "   RETURN", "\n")
}

// EmitModify will emit a +=, -= etc. operation
func EmitModify(s *State, id string, op Token, value string) {
	slog.Info(No(s)+" EmitModify: ", "id", id, "op", op.Name(), "value", value)
	emit(s, "   "+TokenNames[op], id+" "+value)
}

func EmitType(s *State, name string, typ int) {
	slog.Info(No(s) + " EmitType: " + name + strconv.Itoa(typ))
}

func EmitLineNo(s *State) {
	emit(s, "   // Line no", strconv.Itoa(s.lineNum))
}

func EmitOp(s *State, op Token) {
	emit(s, "   "+TokenNames[op], "")
}

func ValueAsString(v ValueDef) string {
	if v.typ.pt == TYP_U8 || v.typ.pt == TYP_U16 || v.typ.pt == TYP_U32 || v.typ.pt == TYP_I16 || v.typ.pt == TYP_I32 || v.typ.pt == TYP_I64 {
		return strconv.FormatInt(v.intValue, 10)
	} else if v.typ.pt == TYP_BOOL {
		if v.boolValue {
			return "true"
		}
		return "false"
	} else if v.typ.pt == TYP_F64 {
		return strconv.FormatFloat(v.floatValue, 'g', -1, 64)
	} else if v.typ.pt == TYP_F32 {
		return strconv.FormatFloat(v.floatValue, 'g', -1, 32)
	}
	return v.stringValue
}

func EmitOpConst(s *State, op Token, c ValueDef) {
	emit(s, "   "+TokenNames[op]+"_IM", ValueAsString(c))
}

func EmitError(s *State, err error) {
	emit(s, "Error on line "+strconv.Itoa(s.lineNum)+": ", err.Error())
}

func EmitPushConst(s *State, value ValueDef) {
	slog.Info(No(s) + " EmitPushConst: " + value.stringValue)
	if !value.hasValue {
		slog.Error("EmitPushConst without value")
	}
	if value.typ.pt == TYP_BOOL {
		if value.boolValue {
			emit(s, "   PUSH_INT", "1")
		} else {
			emit(s, "   PUSH_INT", "0")
		}
	} else if value.typ.pt == TYP_U8 || value.typ.pt == TYP_U16 || value.typ.pt == TYP_I16 || value.typ.pt == TYP_U32 || value.typ.pt == TYP_I32 || value.typ.pt == TYP_I64 {
		emit(s, "   PUSH_INT", strconv.FormatInt(value.intValue, 10))
	} else if value.typ.pt == TYP_F64 || value.typ.pt == TYP_F32 {
		emit(s, "   PUSH_FLOAT", strconv.FormatFloat(value.floatValue, 'g', -1, 64))
	} else {
		emit(s, "   PUSH_PTR ", "0")
	}
}

func EmitComment(s *State, comment string) {
	emit(s, "  //", comment)
}
