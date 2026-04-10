;-------------
section .text
;-------------

; syscall will call any dll function that is reachable
; The address of the function should be in rdi, arg count *8 in rbx
; rax is the first parameter
_syscall:
    push rbp              ; Save old frame pointer  
    mov rbp, rsp          ; Setup new frame pointer
    and rsp, -16          ; Align stack by clearing the 4 lsb
    sub rsp, 96           ; Reserve space for arguments to the called function

    mov rcx, rax          ; rcx = First argument: format string
    or rbx, rbx
    jz .L3

    mov rdx, [rbp+16]    ; dx = Second argument
    sub rbx, 8
    jc .L3

    mov r8,  [rbp+24]    ; r8 = Third argument
    sub rbx, 8
    jc .L3

    mov r9,  [rbp+32]    ; r9 = Forth argument
    sub rbx, 8
    jc .L3

    mov rsi, [rbp+40]    ; Fifth argument onto stack
    mov [rsp+32], rsi
    sub rbx, 8
    jc .L3

    mov rsi, [rbp+48]
    mov [rsp+40], rsi     ; Sixth argument onto stack
    sub rbx, 8
    jc .L3

    mov rsi, [rbp+56]
    mov [rsp+48], rsi     ; Seventh argument onto stack
    sub rbx, 8
    jc .L3

    mov rsi, [rbp+64]
    mov [rsp+56], rsi     ; Eight argument onto stack
    sub rbx, 8
    jc .L3

    mov rsi, [rbp+72]
    mov [rsp+64], rsi     ; Nineth argument onto stack
    sub rbx, 8
    jc .L3

    mov rsi, [rbp+80]
    mov [rsp+72], rsi     ; Tenth argument onto stack

.L3:
    call rdi
    leave
    ret


