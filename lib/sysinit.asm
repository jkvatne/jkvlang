; sysinit.asm   Initializes the program

%define STD_INPUT_HANDLE  -10
%define STD_OUTPUT_HANDLE -11
%define STD_ERROR_HANDLE  -12

global ProcessHeap

;-------------
section .bss
;-------------
alignb 8
StdOutputHandle resq 1
StdErrorHandle  resq 1
StdInputHandle  resq 1
ProcessHeap     resq 1

;-------------
section .data
;-------------
    locale_str  db ".utf8", 0   ; "UTF" locale, or use "" for system default
    f64sign_mask: dq 0x8000000000000000
    f32sign_mask: dq 0x80000000
    global f32sign_mask
    global f64sign_mask
    argc: dq 0
    argv: dq 0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,
    arg0: dq 0
    arg1: dq 0
    arg2: dq 0

;-------------
section .text
;-------------

extern GetStdHandle
extern ExitProcess
extern GetProcessHeap

global argc, argv, arg0, arg1, arg2
global _sysinit
global _cstrlen

; strlen will calculate the length of a 0-terminated C-string at [rax]
_cstrlen:
    push rbp                         ; Prologue: Save frame pointer
    mov rbp, rsp                     ; Prologue: Setup new frame pointer.
    mov rax, [rbp+16]
    mov rdi, [rax]
    xor   rax,rax      ; compare to zero
    mov   rcx,-1       ;limit scan length
    cld
    repne scasb
    not rcx
    dec   rcx          ;minus one for rep going too far by one
    mov [rbp+24], rcx
    leave
    ret

_sysinit:
    ; sysinit will initialize the console handles
    push rbp                         ; Prologue: Save frame pointer
    mov rbp, rsp                     ; Prologue: Setup new frame pointer.
    and rsp, -16                     ; Prologue: Align stack by clearing the 4 lsb
    sub rsp, 48                      ; Prologue: Reserve shadow space

    ; Get this threads local allocation heap
    call GetProcessHeap
    mov [ProcessHeap], rax

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
