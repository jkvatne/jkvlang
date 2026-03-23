
extern printf

;---------------------------------------------
section .bss          ; Uninitialized data segment
;---------------------------------------------

;---------------------------------------------
section .rodata        ;  Read only data
;---------------------------------------------
msg          db "Message from print", 0Dh, 0Ah, 00h

;---------------------------------------------
section .text
global foo
foo:
    push rbp
    mov rbp, rsp
    mov ax, cx
    add ax, dx
    leave
    ret

