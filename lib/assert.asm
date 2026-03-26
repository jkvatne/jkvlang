
; Symbols from msvcrt.dll
extern printf

;-------------
section .rodata
;-------------
crlf_str  db 0Ah, 00h
assert_mess        db "Assert failed", 00h

;-------------
section .text
;-------------

; assert will verify that the first arbument (rax) is true (not 0)
; with optional additional parameters.
; The stack will contain <messageptr><arg1><arg2>..
; rbx should contain the size of the stack. (number of arguments-1) * 8.
; rax is already the value to be tested
; NB: Assert will append CRLF after the message.
global _assert
_assert:
    push rbp
    mov rbp, rsp          ; Setup new frame pointer
    and rsp, -16          ; Align stack by clearing the 4 lsb
    sub rsp, 96           ; Reserve space for arguments to the called function

    or rax, rax           ; Set z-flag if rax is zero
    jz .L1                 ; Jump if the bool argument was false
    leave
    ret                   ; Returns if assert(true)
.L1:
    or bx, bx            ; Check if bx=0 (no string given)
    jnz .L5 
    mov bx, 8
    mov rcx, assert_mess    
    jmp .L4
.L5:    

    mov rcx, [rbp+16]    ; rcx = First argument: format string
    add rcx, 8           ; Skip length/capacity of string
    sub rbx, 8
    or rbx, rbx
    jz .L2
.L4:
    mov rdx, [rbp+24]    ; dx = Second argument
    sub rbx, 8
    jc .L2

    mov r8,  [rbp+32]    ; r8 = Third argument
    sub rbx, 8
    jc .L2

    mov r9,  [rbp+40]    ; r9 = Forth argument
    sub rbx, 8
    jc .L2

    mov rsi, [rbp+48]    ; Fifth argument onto stack
    mov [rsp+32], rsi
    sub rbx, 8
    jc .L2

    mov rsi, [rbp+56]
    mov [rsp+40], rsi     ; Sixth argument onto stack
    sub rbx, 8
    jc .L2

    mov rsi, [rbp+64]
    mov [rsp+48], rsi     ; Seventh argument onto stack
    sub rbx, 8
    jc .L2

    mov rsi, [rbp+72]
    mov [rsp+56], rsi     ; Eight argument onto stack
    sub rbx, 8
    jc .L2

    mov rsi, [rbp+80]
    mov [rsp+64], rsi     ; Nineth argument onto stack
    sub rbx, 8
    jc .L2

    mov rsi, [rbp+88]
    mov [rsp+72], rsi     ; Tenth argument onto stack
    jmp .L2

.L2:
    call printf

    mov rcx, crlf_str
    call printf

    leave
    ret

