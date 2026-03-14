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
extern GetStdHandle
extern WriteFile
extern ExitProcess
extern HeapAlloc
extern printf
extern GetCommandLineA
extern GetCommandLineW
extern GetEnvironmentStringsA
extern GetEnvironmentStringsW
extern CreateFileA
extern CreateFileW
extern CloseHandle
extern GetProcessHeap
extern syscall                                ; fraom syscall.asm

global _start                                 ; Export symbols. The entry point
global _alloc
global _assert

section .data                                   ; Initialized data segment
    message         db "Message from WriteFile", 0Dh, 0Ah
    messlen         EQU $-message                   ; Address of this line ($) - address of Message
    startup_msg     db "Startup code version %d.%d.%d", 0Dh, 0Ah, 00h
    test4par        db "Should be numbers 2-4 here: %d, %d, %d", 0Dh, 0Ah, 00h
    test5par        db "Should be numbers 2-5 here: %d, %d, %d, %d", 0Dh, 0Ah, 00h
    test6par        db "Should be numbers 2-6 here: %d, %d, %d, %d, %d", 0Dh, 0Ah, 00h
    test7par        db "Should be numbers 2-7 here: %d, %d, %d, %d, %d, %d", 0Dh, 0Ah, 00h
    spmess          db "... rsp = 0x%X", 0Dh, 0Ah, 00h

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

printsp:
    push spmess
    push rsp
    mov rax, 2
    mov rdi, printf
    call syscall
    add sp, 8*2
    ret

_start:
    sub   rsp, 40                                  ; Align the stack to a multiple of 16 bytes+32 bytes shadow

    call printsp

    ; Print a startup message with integer parameters using the prinf from msvcrt.dll
    ; Must link with msvcrt.dll
    mov rcx, startup_msg  ; First argument: format string
    mov rdx, 0            ; Second argument: number
    mov r8,  0            ; Third argument: number
    mov r9,  1            ; Forth argument: number
    call printf           ; Call printf

    call printsp

    mov   ecx, STD_OUTPUT_HANDLE
    call  GetStdHandle
    mov   qword [rel StdOutputHandle], RAX

    mov   ecx, STD_ERROR_HANDLE
    call  GetStdHandle
    mov   qword [rel StdErrorHandle], RAX

    mov   ecx, STD_INPUT_HANDLE
    call  GetStdHandle
    mov   qword [rel StdInputHandle], RAX

    call printsp

    ; Test using WriteFile
    sub   RSP, 16                                  ; 5th parameter + align stack to a multiple of 16 bytes
    mov   RCX, qword [StdOutputHandle]             ; 1st parameter is the handle
    lea   RDX, [message]                           ; 2nd parameter is a pointer to the text to be written
    mov   R8, messlen                              ; 3rd parameter is the number of bytes to write
    lea   R9, [rel Written]                        ; 4th parameter is a pointer to the variable receiving the number of bytes written.
    mov   qword [RSP + 32], 0                      ; 5th parameter is a pointer to the lpOverlapped structure (or nil).
    call  WriteFile                                ; Call the WriteFile function found in kernel32.dll (must be linked to)
    add   RSP, 16
    call printsp

    ; Test using syscall
    push test4par              ; 1st parameter
    push 2                      ; 2nd parameter
    push 3                      ; 3rd parameter
    push 4                      ; 4th parameter
    mov rax, 4                  ; Number of parameters on stack
    mov rdi, printf             ; Address to call
    call syscall
    add sp, 8*4

    call printsp

    ; Test using syscall
    push test5par              ; 1st parameter
    push 2                      ; 2nd parameter
    push 3                      ; 3rd parameter
    push 4                      ; 4th parameter
    push 5                      ; 5th parameter
    mov rax, 5                  ; Number of parameters on stack
    mov rdi, printf             ; Address to call
    call syscall
    add sp, 8*5

    call printsp

    ; Test using syscall
    push test6par              ; 1st parameter
    push 2                      ; 2nd parameter
    push 3                      ; 3rd parameter
    push 4                      ; 4th parameter
    push 5                      ; 5th parameter
    push 6                      ; 6th parameter
    mov rax, 6                  ; Number of parameters on stack
    mov rdi, printf             ; Address to call
    call syscall
    add sp, 8*6

    call printsp

    push test7par              ; 1st parameter
    push 2                      ; 2nd parameter
    push 3                      ; 3rd parameter
    push 4                      ; 4th parameter
    push 5                      ; 5th parameter
    push 6                      ; 6th parameter
    push 7                      ; 6th parameter
    mov rax, 7                  ; Number of parameters on stack
    mov rdi, printf             ; Address to call
    call syscall
    add sp, 8*7

    call printsp

    ; Exit with error code 1
    mov   rcx, 123
    call  ExitProcess
