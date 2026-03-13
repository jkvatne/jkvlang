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
extern GetEnvironmentStringsA
extern CreateFileA
extern CloseHandle
extern GetProcessHeap

global _start                                    ; Export symbols. The entry point
global _alloc
global _assert

section .data                                   ; Initialized data segment
    message         db "Message from WriteFile", 0Dh, 0Ah
    messlen         EQU $-message                   ; Address of this line ($) - address of Message
    startup_msg     db "Startup code version %d.%d.%d", 0Dh, 0Ah, 00h

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

_start:
    sub   rsp, 40                                  ; Align the stack to a multiple of 16 bytes+32 bytes shadow

    ; Print a startup message with integer parameters using the prinf from msvcrt.dll
    ; Must link with msvcrt.dll
    mov rcx, startup_msg  ; First argument: format string
    mov rdx, 0            ; Second argument: number
    mov r8,  0            ; Third argument: number
    mov r9,  1            ; Forth argument: number
    call printf           ; Call printf

    mov   ecx, STD_OUTPUT_HANDLE
    call  GetStdHandle
    mov   qword [rel StdOutputHandle], RAX

    mov   ecx, STD_ERROR_HANDLE
    call  GetStdHandle
    mov   qword [rel StdErrorHandle], RAX

    mov   ecx, STD_INPUT_HANDLE
    call  GetStdHandle
    mov   qword [rel StdInputHandle], RAX

    sub   RSP, 16                                  ; 5th parameter + align stack to a multiple of 16 bytes
    mov   RCX, qword [StdOutputHandle]             ; 1st parameter is the handle
    lea   RDX, [message]                           ; 2nd parameter is a pointer to the text to be written
    mov   R8, messlen                              ; 3rd parameter is the number of bytes to write
    lea   R9, [rel Written]                        ; 4th parameter is a pointer to the variable receiving the number of bytes written.
    mov   qword [RSP + 32], 0                      ; 5th parameter is a pointer to the lpOverlapped structure (or nil).
    call  WriteFile                                ; Call the WriteFile function found in kernel32.dll (must be linked to)
    add   RSP, 48                                  ; Remove the 48 bytes
    
    push  rax
    xor   ECX, ECX
    call  ExitProcess
