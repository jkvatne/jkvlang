; File "C:\Doc\compiler\test\build\hello.asm"

%include "C:\Doc\compiler\test\syscall.asm"

section .text

   global _start
   global WinMain
   extern assert
   extern syscall
   extern exit
   extern malloc
   extern mfree
   extern sysinit
   extern print

WinMain:
main:
   mov rbp, rsp; for correct debugging
   call sysinit
   call main                            ; Call the main procedure
   xor eax, eax                         ; Error code = 0
   call exit


main2:
   push rbp
   mov rbp, rsp
   ; Line 2     s = "HelloHello"
   xor rdx, rdx
   push rdx                             ; New variable, s
   ; Line 3     t = "World....."
   xor rdx, rdx
   push rdx                             ; New variable, t
   ; Line 4     v = s + t
   mov rax, qword [rbp-16]              ; Load variable t
   ; String concatenation. First allocate string   push rax
   push 50
   call malloc
   ; Skip string length and set up destination in rdi
   mov rdi, rax
   add rdi, 4
   ; Set si to point to first string and skip size
   mov rsi,[rsp+8]
   add rsi, 4
   ; Set count and direction flag
   mov rcx, 3
   cld
   ; Do copy
   rep movsb
   ; Copy second string
   pop rsi
   add rsi, 4
   mov rcx, 3
   rep movsb
   ; now AX should point to the string. Set resulting length
   mov dword [rax], 6
   xor rdx, rdx
   push rdx                             ; New variable, v
   ; Line 5     print(v)
   mov rax, qword [rbp-16]              ; Load variable v
   push rax                             ; 2 Push TOS
   mov rax, str0
   mov rbx, 0
   call print
   add rsp, 24
   leave
   ret                                  ; return from main

section .rodata

alignb 8
str0 dd 10
     db `HelloHello`, 00h
str1 dd 10
     db `World.....`, 00h
