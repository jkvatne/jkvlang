section .text

global syscall


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
