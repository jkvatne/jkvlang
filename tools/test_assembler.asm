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

global _start                                    ; Export symbols. The entry point
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

; _printf is a function that can be called from the compiled code.
; It assumes parameters are on the stack
; rax contains the number of parameters
_printf:
    push rbp
    inc rax
    mov rbp, rsp
    shl rax, 3
    mov rcx, [rax+rbp]   ; First argument: format string
    sub rax, 8
    mov rdx, [rax+rbp]    ; Second argument
    sub rax, 8
    mov r8,  [rax+rbp]    ; Third argument
    sub rax, 8
    mov r9,  [rax+rbp]    ; Forth argument
    sub rsp, 80           ; Reserve stack
    sub rax, 8
    mov rbx, [rax+rbp]
    and rsp, -16          ; Align stack by clearing the 4 lsb
    sub rsp, 80           ; Reserve shadow space
    mov [rsp+32], rbx     ; Fifth argument onto stack
    sub rax, 8
    mov rbx, [rax+rbp]
    mov [rsp+40], rbx     ; Sixth argument onto stack
    call printf
    leave
    ret

; _syscall will call any dll function that is reachable
; The address of the function should be in r10, arg count in rax
syscall1:
    push rbp
    inc rax
    mov rbp, rsp
    shl rax, 3
    mov rcx, [rax+rbp]   ; First argument: format string
    sub rax, 8
    mov rdx, [rax+rbp]    ; Second argument
    sub rax, 8
    mov r8,  [rax+rbp]    ; Third argument
    sub rax, 8
    mov r9,  [rax+rbp]    ; Forth argument
    sub rsp, 80           ; Reserve stack
    sub rax, 8
    mov rbx, [rax+rbp]
    and rsp, -16          ; Align stack by clearing the 4 lsb
    sub rsp, 80           ; Reserve shadow space
    mov [rsp+32], rbx     ; Fifth argument onto stack
    sub rax, 8
    mov rbx, [rax+rbp]
    mov [rsp+40], rbx     ; Sixth argument onto stack
    call [rdi]
    leave
    ret

; syscall will call any dll function that is reachable
; The address of the function should be in r10, arg count in rax
syscall:
    push rbp
    mov rbp, rsp          ; Setup new frame pointer

    and rsp, -16          ; Align stack by clearing the 4 lsb
    sub rsp, 80           ; Reserve space for arguments to the called function

    dec rax
    shl rax, 3

    mov rcx, [rax+rbp+16]    ; cx = First argument: format string
    sub rax, 8
    jc docall

    mov rdx, [rax+rbp+16]    ; dx = Second argument
    sub rax, 8
    jc docall

    mov r8,  [rax+rbp+16]    ; r8 = Third argument
    sub rax, 8
    jc docall

    mov r9,  [rax+rbp+16]    ; r9 = Forth argument
    sub rax, 8
    jc docall

    mov rbx, [rax+rbp+16]    ; Fifth argument onto stack
    mov [rsp+32], rbx
    sub rax, 8
    jc docall

    mov rbx, [rax+rbp+16]
    mov [rsp+40], rbx     ; Sixth argument onto stack
    sub rax, 8
    jc docall

    mov rbx, [rax+rbp+16]
    mov [rsp+48], rbx     ; Seventh argument onto stack
    sub rax, 8
    jc docall

    mov rbx, [rax+rbp+16]
    mov [rsp+56], rbx     ; Eight argument onto stack
    sub rax, 8

docall:
    call [rdi]
    leave
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

    ; Test using WriteFile
    sub   RSP, 16                                  ; 5th parameter + align stack to a multiple of 16 bytes
    mov   RCX, qword [StdOutputHandle]             ; 1st parameter is the handle
    lea   RDX, [message]                           ; 2nd parameter is a pointer to the text to be written
    mov   R8, messlen                              ; 3rd parameter is the number of bytes to write
    lea   R9, [rel Written]                        ; 4th parameter is a pointer to the variable receiving the number of bytes written.
    mov   qword [RSP + 32], 0                      ; 5th parameter is a pointer to the lpOverlapped structure (or nil).
    call  WriteFile                                ; Call the WriteFile function found in kernel32.dll (must be linked to)
    add   RSP, 48                                  ; Remove the 48 bytes

    ; Test using _prinf
    push test6par               ; 1st parameter
    push 2                      ; 2nd parameter
    push 3                      ; 3rd parameter
    push 4                      ; 4th parameter
    push 5                      ; 5th parameter
    push 6                      ; 6th parameter
    mov rax, 6                  ; Number of parameters on stack
    call _printf
    add sp, -8*6

    ; Test using syscall
    push test4par              ; 1st parameter
    push 2                      ; 2nd parameter
    push 3                      ; 3rd parameter
    push 4                      ; 4th parameter
    mov rax, 4                  ; Number of parameters on stack
    mov rdi, printf             ; Address to call
    call syscall
    add sp, -8*4

    ; Test using syscall
    push test5par              ; 1st parameter
    push 2                      ; 2nd parameter
    push 3                      ; 3rd parameter
    push 4                      ; 4th parameter
    push 5                      ; 5th parameter
    mov rax, 5                  ; Number of parameters on stack
    mov rdi, printf             ; Address to call
    call syscall
    add sp, -8*5

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
    add sp, -8*6

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
    add sp, -8*7

    ; Exit with error code 1
    mov   rcx, 123
    call  ExitProcess
