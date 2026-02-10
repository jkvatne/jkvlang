package main

import (
	"fmt"
	"log/slog"
	"strconv"
)

func emit(opcode string, value string) {
	if !*noCode {
		fmt.Printf("%s %s\n", opcode, value)
	}
}

func EmitPop(id string) {
	slog.Info("Emit pop", "name", id)
	emit("   POP", id)
}

func EmitPush(id string) {
	slog.Info("Emit push", "name", id)
	emit("   PUSH", id)
}

func EmitCall(id string) {
	slog.Info("Emit call", "name", id)
	emit("   CALL", id)
}

func GenerateOp(s *state, op int) {
	slog.Info("Generate", "Op", TokenNames[op])
	emit("   OP", TokenNames[op])
}

func PushInt(s *state, value string) {
	slog.Info("PushInt", "Value", value)
	emit("   PUSH", value)
}

func PushFloat(s *state, value string) {
	slog.Info("PushFloat", "Value", value)
	emit("   PUSH", value)
}

func PushString(s *state, value string) {
	slog.Info("PushString", "Value", value)
	emit("   PUSHSTR", "\""+value+"\"")
}

func EmitLabel(n int) {
	slog.Info("EmitLabel", "no", n)
	emit("L"+strconv.Itoa(n), ":")
}

func EmitFunction(id string) {
	slog.Info("EmitFunction")
	emit(id, ":")
}

func EmitJump(n int) {
	slog.Info("EmitJump", "no", n)
	emit("   JUMP", "L"+strconv.Itoa(n))
}

func EmitJumpFalse(n int) {
	slog.Info("EmitJump", "no", n)
	emit("   JUMPFALSE", "L"+strconv.Itoa(n))
}

func EmitReturn() {
	slog.Info("EmitReturn")
	emit("   RETURN", "")
}
