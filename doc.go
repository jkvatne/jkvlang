package main

/*


BYTECODES
=========
The stack consists of quadwords, 8 bytes long.
Immediate values are allways 64bit, but can be compressed.

Data types: 8bit, 16bit, 32bit, 64bit, F32, F64 = 6 types

16 operations * 4 adr mode = 64 instructions



  The 8 basic operations on integers, operating on the two 64bit values on top of the stack
  0 ADD                                             [SP-1] = [SP]+[SP-1]; SP--
  1 SUB                                             [SP-1] = [SP]-[SP-1]; SP--
  2 MUL                                             [SP-1] = [SP]*[SP-1]; SP--
  3 DIV                                             [SP-1] = [SP]/[SP-1]; SP--
  4 MOD                                             [SP-1] = [SP]%[SP-1]; SP--
  5 AND                                             [SP-1] = [SP]&[SP-1]; SP--
  6 OR                                              [SP-1] = [SP]|[SP-1]; SP--
  7 XOR                                             [SP-1] = [SP]^[SP-1]; SP--
  8 SHR                                             [SP-1] = [SP-1]>>[SP]; SP--
  9 SHL                                             [SP-1] = [SP-1]<<[SP-1]; SP--
 10 SAR                                             [SP-1] = [SP-1]>>[SP]; SP--
 11 FADD                                            [SP-1] = [SP]+[SP-1]; SP--
 12 FSUB                                            [SP-1] = [SP]-[SP-1]; SP--
 13 FMUL                                            [SP-1] = [SP]*[SP-1]; SP--
 14 FDIV                                            [SP-1] = [SP]/[SP-1]; SP--
 15-29 TOS Op Immediate                             [SP] = [SP] op Immediate
 30-44 TOS Op [BP+Immediate]                        [SP] = [SP] op [BP+Immediate]
 45-59 TOS Op [GP+Immediate]                        [SP] = [SP] op [GP+Immediate]

// Load immediate
 60 LOAD_U                                          SP++; [SP] = immediate unsigned
 61 LOAD_I                                          SP++; [SP] = immediate sign extended
 62 LOAD_F32                                        SP++; [SP] = immediate F32;
 63 LOAD_F64                                        SP++; [SP] = immediate F64;

 64
 65
 66
 67

 // Load/store local
 68 LOADL_U8                                        SP++; [SP] = [BP+immediate] 8bit
 69 LOADL_U16                                       SP++; [SP] = [BP+immediate] 16 bit
 70 LOADL_I16                                       SP++; [SP] = [BP+immediate] 16 bit sign extend
 71 LOADL_U32                                       SP++; [SP] = [BP+immediate] 32 bit
 72 LOADL_I32                                       SP++; [SP] = [BP+immediate] 32 bit sign extend
 73 LOADL_I64                                       SP++; [SP] = [BP+immediate] 64 bit
 74 LOADL_F32                                       SP++; [SP] = [BP+immediate] F32
 75 LOADL_F64                                       SP++; [SP] = [BP+immediate] F64
 76 STOREL8                                         [BP+immediate] = [SP]; SP--
 77 STOREL16                                        [BP+immediate] = [SP]; SP--
 78 STOREL32                                        [BP+immediate] = [SP]; SP--
 79 STOREL64                                        [BP+immediate] = [SP]; SP--
 80 STORELF32                                       [BP+immediate] = [SP]; SP--
 81 STORELF64                                       [BP+immediate] = [SP]; SP--

 // Load/store global
 82 LOADG_U7                                        SP++; [SP] = [GP+immediate] 8bit
 83 LOADG_U16                                       SP++; [SP] = [GP+immediate] 16 bit
 84 LOADG_I16                                       SP++; [SP] = [GP+immediate] 16 bit
 85 LOADG_U32                                       SP++; [SP] = [GP+immediate] 32 bit
 86 LOADG_I32                                       SP++; [SP] = [GP+immediate] 32 bit
 87 LOADG_I64                                       SP++; [SP] = [GP+immediate] 64 bit
 88 LOADG_F32                                       SP++; [SP] = [GP+immediate] F32
 89 LOADG_F64                                       SP++; [SP] = [GP+immediate] F64
 90 STOREG8                                         [GP+immediate] = [SP]; SP--
 91 STOREG16                                        [GP+immediate] = [SP]; SP--
 92 STOREG32                                        [GP+immediate] = [SP]; SP--
 93 STOREG64                                        [GP+immediate] = [SP]; SP--
 94 STOREGF32                                       [GP+immediate] = [SP]; SP--
 95 STOREGF64                                       [GP+immediate] = [SP]; SP--

 // Load/store indirect
 48 LOADG8                                          SP++; [SP] = [DP+immediate] 8bit
 49 LOADG16                                         SP++; [SP] = [DP+immediate] 16 bit
 50 LOADG32                                         SP++; [SP] = [DP+immediate] 32 bit
 51 LOADG64                                         SP++; [SP] = [DP+immediate] 64 bit
 57 LOADGF32                                        SP++; [SP] = [DP+immediate] F32
 58 LOADGF32                                        SP++; [SP] = [DP+immediate] F64
 52 STOREG8                                         [DP+immediate] = [SP]; SP--
 53 STOREG16                                        [DP+immediate] = [SP]; SP--
 54 STOREG32                                        [DP+immediate] = [SP]; SP--
 55 STOREG64                                        [DP+immediate] = [SP]; SP--
 59 STOREGF32                                       [DP+immediate] = [SP]; SP--
 59 STOREGF64                                       [DP+immediate] = [SP]; SP--


 62 ADDF32                                          [SP-1] = [SP]+[SP-1]; SP--
 63 SUBF32                                          [SP-1] = [SP]-[SP-1]; SP--
 64 MULF32                                          [SP-1] = [SP]*[SP-1]; SP--
 65 DIVF32                                          [SP-1] = [SP]/[SP-1]; SP--
 66 ADDF64                                          [SP-1] = [SP]+[SP-1]; SP--
 67 SUBF64                                          [SP-1] = [SP]-[SP-1]; SP--
 68 MULF64                                          [SP-1] = [SP]*[SP-1]; SP--
 69 DIVF64                                          [SP-1] = [SP]/[SP-1]; SP--


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

** Alternative 1

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

 0 [00nnnnnn]                           data = 0..63  (0..2**6-1)
 1 [11nnnnnn]                           data = -1..-64  (-1..-2**6)
 2 [01000mmm] [d0] ... [d(m-1)]         mmm+1 bytes of data (1..8 bytes), sign extended when m<7
 3 [10000mmm] [d0] ... [d(m-1)]         mmm+1 bytes of data (1..8 bytes), no sign extended

 b = mem[pc]
 if b.bit7==b.bit6 return b
 n=b&0x07
 for i = 0..b&0x07+1 {
     result[i]=mem[pc+i+1]
 }
 return I64(result)




*/
