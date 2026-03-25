

; Symbols from kernel32
extern ExitProcess
extern GetProcessHeap
extern HeapAlloc
extern HeapFree
extern GetStdHandle
extern GetLastError
extern FormatMessageA

; Symbols from msvcrt.dll
extern printf

%define STD_INPUT_HANDLE  -10
%define STD_OUTPUT_HANDLE -11
%define STD_ERROR_HANDLE  -12
%define MAX_ERROR_LEN     40*8
%define FORMAT_MESSAGE_FROM_SYSTEM  4096

;-------------
section .bss
;-------------
alignb 8
StdOutputHandle resq 1
StdErrorHandle  resq 1
StdInputHandle  resq 1
error_len       resq 1              ; 16 bit string length
error           resq MAX_ERROR_LEN

;-------------
section .rodata
;-------------
crlf_str  db 0Dh, 0Ah, 00h
sp_mess            db "...  sp = 0x%X", 0Dh, 0Ah, 00h


;-------------
section .text
;-------------

global main
main:       
   mov rbp, rsp; for correct debugging
   call sysinit
   call print_sp
   mov rax, 1
   ret


; Print the contents of the rsp register using printf
print_sp:
    push rbp                         ; Prologue: Save frame pointer
    mov rbp, rsp                     ; Prologue: Setup new frame pointer.
    push rsp           ; Value to be printed
    mov rax, sp_mess   ; Message at top of stack
    mov rbx, 1*8       ; Stack size is 8 bytes
    mov rdi, printf    ; system function to call
    call syscall
    add sp, 1*8
    leave
    ret
    
    
sysinit:
    ; sysinit will initialize the console handles
    push rbp                         ; Prologue: Save frame pointer
    mov rbp, rsp                     ; Prologue: Setup new frame pointer.
    and rsp, -16                     ; Align stack by clearing the 4 lsb
    sub rsp, 32                      ; Reserve shadow space

    ; Load the handle for standard output
    mov   ecx, STD_OUTPUT_HANDLE
    call  GetStdHandle
    mov   [rel StdOutputHandle], rax

    mov   ecx, STD_ERROR_HANDLE
    call  GetStdHandle
    mov   [rel StdErrorHandle], rax

    mov   ecx, STD_INPUT_HANDLE
    call  GetStdHandle
    mov   [rel StdInputHandle], rax

    mov qword [error], 0
    mov [error_len], word 0
    leave   
    ret
    
    
; print is the local version of fprintf
; Arg count should be in rbx
; The last parameter in rax
; All other parameters pushed on stack.
print:
    add rax, 4
    mov rdi, printf
    ; fallthrough to syscall

; syscall will call any dll function that is reachable
; The address of the function should be in rdi, arg count *8 in rbx
; rax is the first parameter
syscall:
    push rbp
    mov rbp, rsp          ; Setup new frame pointer
    and rsp, -16          ; Align stack by clearing the 4 lsb
    sub rsp, 96           ; Reserve space for arguments to the called function
    mov r15, 0            ; Default to no error

    mov rcx, rax          ; rcx = First argument: format string
    or rbx, rbx
    jz _L3

    mov rdx, [rbp+16]    ; dx = Second argument
    sub rbx, 8
    jc _L3

    mov r8,  [rbp+24]    ; r8 = Third argument
    sub rbx, 8
    jc _L3

    mov r9,  [rbp+32]    ; r9 = Forth argument
    sub rbx, 8
    jc _L3

    mov rsi, [rbp+40]    ; Fifth argument onto stack
    mov [rsp+32], rsi
    sub rbx, 8
    jc _L3

    mov rsi, [rbp+48]
    mov [rsp+40], rsi     ; Sixth argument onto stack
    sub rbx, 8
    jc _L3

    mov rsi, [rbp+56]
    mov [rsp+48], rsi     ; Seventh argument onto stack
    sub rbx, 8
    jc _L3

    mov rsi, [rbp+64]
    mov [rsp+56], rsi     ; Eight argument onto stack
    sub rbx, 8
    jc _L3

    mov rsi, [rbp+72]
    mov [rsp+64], rsi     ; Nineth argument onto stack
    sub rbx, 8
    jc _L3

    mov rsi, [rbp+80]
    mov [rsp+72], rsi     ; Tenth argument onto stack

_L3:
    call rdi
    leave
    ret
