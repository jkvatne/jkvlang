
; Symbols from kernel32
extern ExitProcess

; invert_err will set err to zero if there was an error
; and sett error to 100 if there was no errors.
; This is used during testing to test expected assert errors.
global _invert_err
_invert_err:
    or r15, r15
    jnz .L1
    mov r15, 100
    ret
.L1:
    mov r15, 0
    ret

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

alignb 8
alloc_size_str  dq 19
                db `--------------------------------------\nLeaked memory: %d   Error code: %d\n`, 00h

