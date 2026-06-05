; exit.asm contains the exit(code) function

; Symbols from kernel32
extern ExitProcess

; exit have one parameter - the error code, found in rax
global _exit
_exit:
    push rbp                         ; Prologue: Save frame pointer
    mov rbp, rsp                     ; Prologue: Setup new frame pointer.
    and rsp, -16                     ; Align stack by clearing the 4 lsb
    sub rsp, 32                      ; Reserve shadow space
    mov rcx, rax
    call ExitProcess
    leave
    ret   

section .rodata

global alloc_size_str
alignb 8
alloc_size_str  dq 19
                db `--------------------------------------\nLeaked memory: %d   Error code: %d\n`, 00h

