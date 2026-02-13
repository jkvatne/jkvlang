package main

/*
BYTECODES
=========
The stack consists of quadwords, 8 bytes long.

General instructions without extra data (max 56)
  0 NOP                                             No operation
  1 NEG                                             [SP] = -[SP]
  2 SWAP                                            [SP]=[SP-1]; [SP-1]=[SP]
  3 DUP                                             [SP+1] =  [SP]; SP++
  4 ADD                                             [SP-1] = [SP]+[SP-1]; SP--
  5 SUB                                             [SP-1] = [SP]-[SP-1]; SP--
  6 MUL                                             [SP-1] = [SP]*[SP-1]; SP--
  7 DIV                                             [SP-1] = [SP]/[SP-1]; SP--
  8 MOD                                             [SP-1] = [SP]%[SP-1]; SP--
  9 RETURN
 10 CALL <offset>
 11 JMP <offset>
 12 JZ <offset>
 13 JNZ <offset>
 14 JEQ <offset>
 15 JNE <offset>
 16 JGT <offset>
 17 JGE <offset>
 18 JLT <offset>
 19 JLE <offset>
 20 JZF <offset>
 21 JNZF <offset>
 22 JEQF <offset>
 23 JNEF <offset>
 24 JGTF <offset>
 25 JGEF <offset>
 26 JLTF <offset>
 27 JLEF <offset>
 28 JMP <TOS>
 29 CALL <TOS>
 30 SYSCALL <immediate>
 31 PROLOG
 32
 ...
 56 Extended opcode (not used yet)

* Data types

Note that I8 is not possible. Allways use U8 = byte
Also, U64 is not possible. All 64 bit values are signed 2-complement.
0x00 U8
0x01 U16
0x02 I16
0x03 U32
0x04 I32
0x05 I64
0x06 F32
0x07 F64

* Operators

1 Move
2 Add
3 Sub
4 Mult
5 Div

* Operands

1 Local = Local op TOS
2 Global = Global op TOS
3 TOS = TOS op Local
4 TOS = TOS op Global
5 TOS = TOS op Immediate

* Operations

A total of 5x5=25 operations on 8 datatypes = 200 opcodes
The remaining 56 are for special operations.
Examples:

ADD_I16_DL <ofs>  Adds a 16bit value from top of stack to the local variable at <ofs>
MOV_U8_DG <ofs>   Moves the lower 8 bits of the top of the stac to the global byte variable at ofs.

* Additional data
Byte 1 : +-32
Byte 1&2: +-
*/
