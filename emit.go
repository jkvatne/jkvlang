package main

import "log/slog"

func Emit(opcode string, value string) {
	//	fmt.Printf("%s %s\n", opcode, value)
}

func GenerateOp(s *state, op int) {
	slog.Info("Generate", "Op", TokenNames[op])
	Emit("OP", TokenNames[op])
}

func PushInt(s *state, value string) {
	slog.Info("PushInt", "Value", value)
	Emit("PUSH", value)
}

func PushFloat(s *state, value string) {
	slog.Info("PushFloat", "Value", value)
	Emit("PUSH", value)
}

func PushString(s *state, value string) {
	slog.Info("PushString", "Value", value)
	Emit("PUSHSTR", value)
}
