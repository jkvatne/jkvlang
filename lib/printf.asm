   
;-------------
section .rodata
;-------------
crlf               db 0Dh,0Ah,00h

;-------------
section .text
;-------------

extern printf
extern fflush

; _printf is the local version of printf from msvcrt.dll
; The first parameter should be in rax (the format string)
; Stack size should be in rbx, 8 bytes for each parameter in the format string
; Note that the format string has 8 bytes initial length/capacity
global _printf  
_printf:
    add rax, 8
    mov rdi, printf
    jmp _syscall

_fflush:
    push rbp              ; Save old frame pointer
    mov rbp, rsp          ; Setup new frame pointer
    and rsp, -16          ; Align stack by clearing the 4 lsb
    sub rsp, 96           ; Reserve space for arguments to the called function
    call fflush
    leave
    ret
