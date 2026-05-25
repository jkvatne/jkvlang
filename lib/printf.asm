   
;-------------
section .rodata
;-------------
crlf               db 0Dh,0Ah,00h
sp_mess            db "...rsp=0x%X", 0Ah, 00h

;-------------
section .text
;-------------

extern printf
extern fflush

; _printf is the local version of printf from msvcrt.dll
; All parameters are pushed on the stack. [rsp] is the format string
; All strings must be C-strings.
; Stack size should be in rbx, 8 bytes for each parameter in the format string
; Note that the format string has 8 bytes initial length/capacity
global _printf  
_printf:
    mov rdi, printf
    call _syscall
    ret

global _fflush
_fflush:
    push rbp              ; Save old frame pointer
    mov rbp, rsp          ; Setup new frame pointer
    and rsp, -16          ; Align stack by clearing the 4 lsb
    sub rsp, 96           ; Reserve space for arguments to the called function
    xor rcx, rcx
    call fflush
    leave
    ret

global _printsp
_printsp:
    push rsp                    ; Value to be printed
    mov rax, sp_mess            ; Message at top of stack
    push rax
    mov rbx, 16                  ; Stack size is 8 bytes
    call _printf                ; system function to call
    add sp, 16
    ret
