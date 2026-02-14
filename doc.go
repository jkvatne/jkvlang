package main

/*


BYTECODES
=========
The stack consists of quadwords, 8 bytes long.
Immediate values are allways 64bit, but can be compressed.



  The 8 basic operations on integers
  0 ADD                                             [SP-1] = [SP]+[SP-1]; SP--
  1 SUB                                             [SP-1] = [SP]-[SP-1]; SP--
  2 MUL                                             [SP-1] = [SP]*[SP-1]; SP--
  3 DIV                                             [SP-1] = [SP]/[SP-1]; SP--
  4 MOD                                             [SP-1] = [SP]%[SP-1]; SP--
  5 AND                                             [SP-1] = [SP]&[SP-1]; SP--
  6 OR                                              [SP-1] = [SP]|[SP-1]; SP--
  7 XOR                                             [SP-1] = [SP]^[SP-1]; SP--

  8-15 Op Immediate                                 [SP] = [SP] op Immediate
 16-31 Op BP+Immediate                              [SP] = [SP] op Immediate
 32-39 Op GP+Immediate                              [SP] = [SP] op Immediate

 40 LOADI8                                          SP++; [SP] = immediate;
 41 LOADI16                                         SP++; [SP] = immediate;
 42 LOADI32                                         SP++; [SP] = immediate;
 43 LOADI64                                         SP++; [SP] = immediate;
 44 LOADL8                                          SP++; [SP] = [BP+immediate]
 45 LOADL16                                         SP++; [SP] = [BP+immediate]
 46 LOADL32                                         SP++; [SP] = [BP+immediate]
 47 LOADL64                                         SP++; [SP] = [BP+immediate]
 48 LOADG8                                          SP++; [SP] = [BP+immediate]
 49 LOADG16                                         SP++; [SP] = [BP+immediate]
 50 LOADG32                                         SP++; [SP] = [BP+immediate]
 51 LOADG64                                         SP++; [SP] = [BP+immediate]

 52 LOADIF32                                        SP++; [SP] = immediate F32;
 53 LOADLF32                                        SP++; [SP] = [BP+immediate]
 54 LOADGF32                                        SP++; [SP] = [BP+immediate]
 55 LOADIF64                                        SP++; [SP] = immediate;
 56 LOADLF64                                        SP++; [SP] = [BP+immediate]
 57 LOADGF64                                        SP++; [SP] = [BP+immediate]

 58 ADDF32                                          [SP-1] = [SP]+[SP-1]; SP--
 59 SUBF32                                          [SP-1] = [SP]-[SP-1]; SP--
 60 MULF32                                          [SP-1] = [SP]*[SP-1]; SP--
 61 DIVF32                                          [SP-1] = [SP]/[SP-1]; SP--
 62 ADDF64                                          [SP-1] = [SP]+[SP-1]; SP--
 63 SUBF64                                          [SP-1] = [SP]-[SP-1]; SP--
 64 MULF64                                          [SP-1] = [SP]*[SP-1]; SP--
 65 DIVF64                                          [SP-1] = [SP]/[SP-1]; SP--


 227 CALL <offset>
 228 JMP <offset>
 229 JZ <offset>
 230 JNZ <offset>
 231 JEQ <offset>
 232 JNE <offset>
 233 JGT <offset>
 234 JGE <offset>
 235 JLT <offset>
 236 JLE <offset>
 237 JZF <offset>
 238 JNZF <offset>
 239 JEQF <offset>
 240 JNEF <offset>
 241 JGTF <offset>
 242 JGEF <offset>
 243 JLTF <offset>
 244 JLEF <offset>
 245 JMP <TOS>

 246 CALL <TOS>
 247 SYSCALL <immediate>
 248 NEG                                             [SP] = -[SP]
 249 SWAP                                            [SP] = [SP-1]; [SP-1] = [SP]
 250 DUP                                             [SP+1] = [SP]; SP++
 251 PROLOG <local variable size>
 252 RETURN
 253
 254 Extended opcode (not used yet)
 255 NOP                                             No operation

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

* Compressed immediate data

** Alternatvie 1

[00nnnnnn] data = 0..63
[010nnnnn] [nnnnnnnn] data = 0..8191
[011nnnnn] [nnnnnnnn] [nnnnnnnn]  data = 0..2097151  (2**13-1)
[11nnnnn] data = -1..-64
[101nnnnn] [nnnnnnnn] data = -1..-8192
[100nnnnn] [nnnnnnnn] [nnnnnnnn]  data = -1..-2097152  (-2**13)
[11111111] [d0] [d1] [d2] [d3] [d4] [d5] [d6] [d7] [d8]

** Alternative 2
 0 [000nnnnn]                           data = 0..31  (0..2**6)
 1 [001nnnnn] [nnnnnnnn]                data = 0..16383   (0..2**14-1)
 2 [010nnnnn] [nnnnnnnn] [nnnnnnnn]     data = 0..41944303  (2**22-1)
 3 [01100mmm] [d0] ... [d(m-1)]         mmm+1 bytes of data (1..8 bytes), sign extended when m<7
 3 [01101xxx]                           Invalid
 3 [01010xxx]                           Invalid
 3 [01011xxx]                           Invalid
 3 [100xxxxx]                           Invalid
 5 [101nnnnn] [nnnnnnnn] [nnnnnnnn]     data = -1..-41944304  (-1..-2**22)
 6 [110nnnnn] [nnnnnnnn]                data = 0..-16384  (-1..-2**14)
 7 [111nnnnn]                           data = -1---32  (-1..-2**6)

** Alternative 3
 0 [00snnnnn]                           data = -32..31  (-2**6..2**6-1)
 1 [01snnnnn] [nnnnnnnn]                data = 0..16383   (0..2**14-1)
 2 [10snnnnn] [nnnnnnnn] [nnnnnnnn]     data = 0..41944303  (2**22-1)
 3 [11   mmm] [d0]..[d(m)]

 ** Alternative 4
 0 [00nnnnnn]                           data = 0..31  (0..2**6)
 1 [11nnnnnn]                           data = -1---32  (-1..-2**6)
 2 [01xxxmmm] [d0] ... [d(m-1)]         mmm+1 bytes of data (1..8 bytes), sign extended when m<7
 3 [10xxxmmm] [d0] ... [d(m-1)]         mmm+1 bytes of data (1..8 bytes), no sign extended

 (xxx must be 0)



*/
