; test_assember.asm
;
; This is a file used to verify the assmembler setup and to test 
; calling system files.

%define false 0
%define true  1

; Symbols imported from syscall.asm
extern syscall                                ; fraom syscall.asm
extern malloc
extern mfree
extern assert
extern exit
extern printf
extern sysinit
extern ExitProcess

global _start                                 ; Export symbols. The entry point

section .data                                   ; Initialized data segment
    message         db "Message from WriteFile", 0Dh, 0Ah
    messlen         EQU $-message                   ; Address of this line ($) - address of Message
    startup_msg     db "Startup code version %d.%d.%d", 0Dh, 0Ah, 00h
    test4par        db "Should be numbers 2-4 here: %d, %d, %d", 0Dh, 0Ah, 00h
    test5par        db "Should be numbers 2-5 here: %d, %d, %d, %d", 0Dh, 0Ah, 00h
    test6par        db "Should be numbers 2-6 here: %d, %d, %d, %d, %d", 0Dh, 0Ah, 00h
    test7par        db "Should be numbers 2-7 here: %d, %d, %d, %d, %d, %d", 0Dh, 0Ah, 00h
    axmess          db "... rax = 0x%X", 0Dh, 0Ah, 00h
    printbxmess     db "... rbx = 0x%X", 0Dh, 0Ah, 00h
    sp_mess         db "...  sp = 0x%X", 0Dh, 0Ah, 00h
    assert_true_mess   db "==== Assert true message, x=%d",0Dh, 0Ah, 00h
    assert_false_mess  db "==== Assert false message, x=%d",0Dh, 0Ah, 00h
    assert_args_mess  db "==== Assert false with two arguments, x=%d, y=%d",0Dh, 0Ah, 00h

section .bss                                    ; Uninitialized data segment

alignb 8
    heap            resq 1
    readback        resq 1

section .text

printax:
    push axmess
    push rax
    mov rbx, 16
    mov rdi, printf
    call syscall
    add sp, 8*2
    ret

printbx:
    push printbxmess
    push rbx
    mov rbx, 16
    mov rdi, printf
    call syscall
    add sp, 8*2
    ret

print_sp:
    push sp_mess
    push rsp
    mov rbx, 16
    mov rdi, printf
    call syscall
    add sp, 8*2
    ret

_start:
    sub   rsp, 40                                  ; Align the stack to a multiple of 16 bytes+32 bytes shadow

    call print_sp

    ; Print a startup message with integer parameters using the prinf from msvcrt.dll
    ; Must link with msvcrt.dll
    mov rcx, startup_msg  ; First argument: format string
    mov rdx, 0            ; Second argument: number
    mov r8,  0            ; Third argument: number
    mov r9,  1            ; Forth argument: number
    call printf           ; Call printf

    call print_sp

    call sysinit

    call print_sp

    ; Test using syscall
    push test4par              ; 1st parameter
    push 2                      ; 2nd parameter
    push 3                      ; 3rd parameter
    push 4                      ; 4th parameter
    mov rbx, 4*8                ; Number of parameters on stack
    mov rdi, printf             ; Address to call
    call syscall
    add sp, 8*4

    call print_sp

    ; Test using syscall
    push test5par              ; 1st parameter
    push 2                      ; 2nd parameter
    push 3                      ; 3rd parameter
    push 4                      ; 4th parameter
    push 5                      ; 5th parameter
    mov rbx, 5*8                ; Number of parameters on stack
    mov rdi, printf             ; Address to call
    call syscall
    add sp, 8*5

    call print_sp

    ; Test using syscall
    push test6par              ; 1st parameter
    push 2                      ; 2nd parameter
    push 3                      ; 3rd parameter
    push 4                      ; 4th parameter
    push 5                      ; 5th parameter
    push 6                      ; 6th parameter
    mov rbx, 6*8                ; Number of parameters on stack
    mov rdi, printf             ; Address to call
    call syscall
    add sp, 6*8

    call print_sp

    push test7par              ; 1st parameter
    push 2                      ; 2nd parameter
    push 3                      ; 3rd parameter
    push 4                      ; 4th parameter
    push 5                      ; 5th parameter
    push 6                      ; 6th parameter
    push 7                      ; 7th parameter
    mov rbx, 7*8                  ; Number of parameters on stack
    mov rdi, printf             ; Address to call
    call syscall
    add sp, 7*8

    call print_sp

    mov rax, 4096
    call malloc
    mov [heap], rax

    ; Store value to heap
    mov rdi, [heap]
    mov qword [rdi], 0x123456
    ; Read back from heap
    mov rax, [rdi]
    call printax

    ; Test mfree. Should give rax=1 after call to mfree
    mov rax, [heap]
    call mfree
    call printax

    call print_sp

    ; Test assert false
    push false
    push assert_false_mess
    push 100
    mov rbx, 3*8
    call assert
    add sp, 3*8

    call print_sp

    ; Test assert true
    push true
    push assert_true_mess
    push 101
    mov rbx, 3*8
    call assert
    add sp, 3*8

    call print_sp

    ; Test assert with two arguments
    push false
    push assert_args_mess
    push 100
    push 101
    mov rbx, 4*8
    call assert
    add sp, 4*8

    ; Test assert with one arguments (no message)
    push false
    mov rbx, 8
    call assert
    add sp, 8

    ; Exit with error code 1
    mov   rax, 1234
    call  exit
