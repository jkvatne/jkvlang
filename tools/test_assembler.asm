; test_assember.asm
;
; This is a file used to verify the assmembler setup and to test 
; calling system files.



    
    
%define STD_INPUT_HANDLE -10
%define STD_OUTPUT_HANDLE -11
%define STD_ERROR_HANDLE -12

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
    Message         db "Hello from system.asm %d %d", 0Dh, 0Ah
    MessageLength   EQU $-Message                   ; Address of this line ($) - address of Message
    Message2        db "Hello again", 0Dh, 0Ah, 00h
    MessageLength2  EQU $-Message2                   ; Address of this line ($) - address of Message
    fmt             db "Hello, number: %d %d", 0Dh, 0Ah

section .bss                                    ; Uninitialized data segment

alignb 8
    StdOutputHandle resq 1
    StdErrorHandle  resq 1
    StdInputHandle  resq 1
    Written         resq 1

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

_print:
    push  rbp
    mov   rbp, rsp
    sub   RSP, 16                                  ; 5th parameter + align stack to a multiple of 16 bytes
    mov   RCX, qword [StdOutputHandle]             ; 1st parameter
    lea   RDX, [rax]                               ; 2nd parameter
    mov   R8, MessageLength                        ; 3rd parameter
    lea   R9, [rel Written]                        ; 4th parameter
    mov   qword [RSP + 32], 0                      ; 5th parameter
    call  WriteFile                                ; Output can be redirect to a file using >
    add   RSP, 48                                  ; Remove the 48 bytes
    ret

_start:
    sub   rsp, 40                                  ; Align the stack to a multiple of 16 bytes+32 bytes shadow

    mov rcx, fmt         ; First argument: format string
    mov rdx, 42          ; Second argument: number
    mov r8,  45          ; Second argument: number
    mov r9,  Message2
    call printf          ; Call printf

    mov   ecx, STD_OUTPUT_HANDLE
    call  GetStdHandle
    mov   qword [rel StdOutputHandle], RAX

    mov   ecx, STD_ERROR_HANDLE
    call  GetStdHandle
    mov   qword [rel StdErrorHandle], RAX

    mov   ecx, STD_INPUT_HANDLE
    call  GetStdHandle
    mov   qword [rel StdInputHandle], RAX

    ;mov   rax, Message
    ;call _print
    ;add   rsp, 8

    mov   rax, Message2
    call _print
    add   rsp, 8
    
    push  rax
    xor   ECX, ECX
    call  ExitProcess
