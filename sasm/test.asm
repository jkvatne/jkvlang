
extern ExitProcess
extern GetStdHandle

global main
global StdOutputHandle
global StdErrorHandle
global StdInputHandle

%define STD_INPUT_HANDLE  -10
%define STD_OUTPUT_HANDLE -11
%define STD_ERROR_HANDLE  -12
%define MAX_ERROR_LEN     40*8

section .bss
alignb 8
    StdOutputHandle resq 1
    StdErrorHandle  resq 1
    StdInputHandle  resq 1
    error_len       resq 1              ; 16 bit string length
    error           resq MAX_ERROR_LEN


section .text
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
    
    
main:
   mov rbp, rsp; for correct debugging
   
   call sysinit
   
   xor rax, rax                         ; Error code = 0
   push rbp                         ; Prologue: Save frame pointer
   mov rbp, rsp                     ; Prologue: Setup new frame pointer.
   and rsp, -16                     ; Align stack by clearing the 4 lsb
   sub rsp, 32                      ; Reserve shadow space
   mov rcx, rax
   call ExitProcess
   ret