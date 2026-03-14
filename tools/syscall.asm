section .text

global syscall

; syscall will call any dll function that is reachable
; The address of the function should be in r10, arg count in rax
syscall:
    push rbp
    mov rbp, rsp
    inc rax
    shl rax, 3
    mov rcx, [rax+rbp]   ; First argument: format string
    sub rax, 8
    jc docall
    mov rdx, [rax+rbp]    ; Second argument
    sub rax, 8
    jc docall
    mov r8,  [rax+rbp]    ; Third argument
    sub rax, 8
    jc docall
    mov r9,  [rax+rbp]    ; Forth argument
    sub rsp, 80           ; Reserve stack
    sub rax, 8
    jc docall
    mov rbx, [rax+rbp]
    and rsp, -16          ; Align stack by clearing the 4 lsb
    sub rsp, 80           ; Reserve shadow space
    mov [rsp+32], rbx     ; Fifth argument onto stack
    sub rax, 8
    jc docall
    mov rbx, [rax+rbp]
    mov [rsp+40], rbx     ; Sixth argument onto stack
docall:
    call [rdi]
    leave
    ret
