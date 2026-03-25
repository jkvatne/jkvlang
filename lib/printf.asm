    
; print is the local version of fprintf
; Arg count should be in rbx
; The first parameter in rax, that is the format string
; Note that the format string has 8 bytes initial length/capacity
; All other parameters pushed on stack.
global _printf
_printf:
    add rax, 8
    mov rdi, printf
    ; fallthrough to syscall
    jmp syscall
    