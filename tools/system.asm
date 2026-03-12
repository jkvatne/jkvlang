
section .text

extern GetStdHandle                             ; Import external symbols
extern WriteFile                                ; Windows API functions, not decorated
extern ExitProcess
extern GetStdHandle
extern WriteFile
extern ExitProcess
extern HeapAlloc
extern printf
extern GetCommandLineA
extern GetEnvironmentStringsA
extern CreateFileA
extern CloseHandle
extern GetProcessHeap

global _start                                    ; Export symbols. The entry point
global _alloc
global _assert

section .data                                   ; Initialized data segment
    Message        db "Hello from system.asm", 0Dh, 0Ah
    MessageLength  EQU $-Message                   ; Address of this line ($) - address of Message

section .bss                                    ; Uninitialized data segment

alignb 8
    StandardHandle resq 1
    Written        resq 1

section .text

; Assert will print error message if [sp] is zero (false)
_assert:

; HeapAlloc(Handle,Flags,Size), using Microsoft ABI
_alloc:
    push rbp
    mov rbp, rsp
    and rsp, -16                        ; Align stack by clearing the 4 lsb
    sub rsp, 32                         ; Reserve shadow space
    call GetProcessHeap
    mov rcx, rax                     ; Handle into rcx
    mov rdx, 8                       ; Flags into rdx
    mov r8, 32                       ; Size into r8
    sub rsp, 48                      ;
    call HeapAlloc
    add rsp, 48
    mov rsp, rbp
    pop rbp
    ret

_start:
    sub   rsp, 40                                  ; Align the stack to a multiple of 16 bytes+32 bytes shadow
    mov   ecx, -11                                 ; Std output
    call  GetStdHandle
    mov   qword [rel StandardHandle], RAX
    sub   RSP, 16                                  ; 5th parameter + align stack to a multiple of 16 bytes
    mov   RCX, qword [rel StandardHandle]          ; 1st parameter
    lea   RDX, [rel Message]                       ; 2nd parameter
    mov   R8, MessageLength                        ; 3rd parameter
    lea   R9, [rel Written]                        ; 4th parameter
    mov   qword [RSP + 32], 0                      ; 5th parameter
    call  WriteFile                                ; Output can be redirect to a file using >
    ;; call  main
    add   RSP, 48                                  ; Remove the 48 bytes
    xor   ECX, ECX
    call  ExitProcess
