%define STD_INPUT_HANDLE  -10
%define STD_OUTPUT_HANDLE -11
%define STD_ERROR_HANDLE  -12

;-------------
section .bss
;-------------
alignb 8
StdOutputHandle resq 1
StdErrorHandle  resq 1
StdInputHandle  resq 1

;-------------
section .text
;-------------

extern GetStdHandle

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
    leave   
    ret
