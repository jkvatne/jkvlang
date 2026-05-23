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
section .data
;-------------
    locale_str  db ".utf8", 0   ; "UTF" locale, or use "" for system default

;-------------
section .text
;-------------

extern GetStdHandle
extern ExitProcess
extern setlocale
extern SetConsoleOutputCP
extern SetConsoleCP

global _sysinit
_sysinit:
    ; sysinit will initialize the console handles
    push rbp                         ; Prologue: Save frame pointer
    mov rbp, rsp                     ; Prologue: Setup new frame pointer.
    and rsp, -16                     ; Prologue: Align stack by clearing the 4 lsb
    sub rsp, 48                      ; Prologue: Reserve shadow space

    mov rcx, 0                ; First argument in RCX (
    mov rdx, locale_str       ; Second argument in RDX
    ;call setlocale

    mov rcx, 65001
    ;call SetConsoleOutputCP

    mov rcx, 65001
    ;call SetConsoleCP

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

    ; Initialize the error code
    mov  r15, 0

    leave   
    ret
