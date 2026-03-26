   
; _printf is the local version of printf from msvcrt.dll
; The first parameter should be in rax (the format string)
; Stack size should be in rbx, 8 bytes for each parameter in the format string
; Note that the format string has 8 bytes initial length/capacity
;-------------
section .rodata
;-------------
crlf               db 0Dh,0Ah,00h

;-------------
section .text
;-------------

global _printf
extern printf
extern syscall

_printf:
    add rax, 8
    mov rdi, printf
    ; fallthrough to syscall
    jmp syscall
    
_println:
    add rax, 8
    mov rdi, printf
    ; fallthrough to syscall
    call syscall

    mov rax, crlf
    jmp syscall