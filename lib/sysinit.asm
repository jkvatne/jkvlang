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

_getargs:
    mov [argc],rcx
    mov [argv],rdx
    mov rsi,rdx
    mov rax,[rsi]
    mov [arg0],rax
    add rsi,8
    mov rax,[rsi]
    mov [arg1],rax
    add rsi,8
    mov rax,[rsi]
    mov [arg2],rax
    ret

_createslice:
    mov r14, [argc]  ; Len
    mov rax, r14
    imul rax, 8
    mov r12, rax      ; Cap
    add rax, 8
    call _alloc
    mov r13, rax     ; Slice location
    mov rdi, rax
    shl r12, 32
    add r12, r14
    mov [rdi], r12   ; Set len/cap in new slice
    ; Now we can add the strings
    mov rsi, argv
    mov rdi, rsi
    xor rax, rax
    mov rcx, -1
    cld
    repne scasb
    not rcx
    dec rcx
    mov rax, rcx
    mov r12, rcx
    shl r12, 32
    add r12, rcx      ; r12 is now len/cap both with string size
    add rax, 8
    call _alloc       ; Allocate new string
    mov r13, rax      ; r13 is ptr to string
    mov [rax], r12    ; Store len/cap
    mov rdi, r13
    add rdi, 8
    mov rsi, [argv]
    mov rcx, r12
    and rcx, 0x7FFFFFFF
    cld
    rep movsb
    ret

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

    ; Get command line arguments
    call _getargs
    call _createslice

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
