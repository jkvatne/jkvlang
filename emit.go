package main

import (
	"log/slog"
	"strconv"
	"strings"
)

func CloseObjFile(s *State) error {
	return s.outputFile.Close()
}

func emit(s *State, opcodes ...string) {
	if s.noCode > 0 {
		return
	}
	for _, op := range opcodes {
		_, err := s.outputFile.WriteString(op + " ")
		if err != nil {
			panic(err)
		}
	}
	_, err := s.outputFile.WriteString("\n")
	if err != nil {
		panic(err)
	}
}

func EmitStore(s *State, id string, typ string) {
	emit(s, "   STORE_"+typ, id)
}

func EmitPush(s *State, id string, typ string) {
	typ = strings.ToUpper(typ)
	emit(s, "   PUSH_"+typ, id)
}

func EmitAssert(s *State) {
	emit(s, "   ASSERT", "")
}

func EmitCall(s *State, id string, argNo int) {
	emit(s, "   CALL", id)
	emit(s, "   ADD SP, ", strconv.Itoa(argNo)+"  // Remove arguments from stack")
}

func EmitLabel(s *State, n int) {
	emit(s, "L"+strconv.Itoa(n), ":")
}

func EmitFunction(s *State, id string) {
	emit(s, id, "")
	emit(s, "   PUSH FP", "")
	emit(s, "   SET FP=SP", "")
}

func EmitJump(s *State, n int) {
	emit(s, "   JUMP", "L"+strconv.Itoa(n))
}

func EmitJumpFalse(s *State, n int) {
	emit(s, "   JUMPFALSE", "L"+strconv.Itoa(n))
}

func EmitReturn(s *State) {
	for i := range len(s.currentFunc.returnTypes) {
		emit(s, "   POP", "[BP-"+strconv.Itoa(len(s.currentFunc.argTypes)+i)+"]  // Return value")
	}
	emit(s, "   RETURN", "\n")
}

// EmitModify will emit a +=, -= etc. operation
func EmitModify(s *State, id string, op Token, value string) {
	emit(s, "   "+TokenNames[op], id+" "+value)
}

func EmitLineNo(s *State) {
	emit(s, "   // Line no", strconv.Itoa(s.lineNum))
}

func EmitOp(s *State, op Token) {
	emit(s, "   "+TokenNames[op], "")
}

func EmitOpConst(s *State, op Token, c ValueDef) {
	emit(s, "   "+TokenNames[op]+"_IM", ValueAsString(c))
}

func EmitError(s *State, err error) {
	emit(s, "Error on line "+strconv.Itoa(s.lineNum)+": ", err.Error())
}

func EmitPushConst(s *State, value ValueDef) {
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
	} else if value.typ.pt == TYP_STRING {
		emit(s, "   PUSH_STRING \"", value.stringValue, "\"")
	} else {
		emit(s, "   PUSH_PTR ", "0")
	}
}

func EmitComment(s *State, comment string) {
	emit(s, "   //", comment)
}
