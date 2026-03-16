; test_assember.asm
;
; This is a file used to verify the assmembler setup and to test 
; calling system files.

%define false 0
%define true  1
%define CREATE_NEW        1
%define CREATE_ALWAYS     2
%define OPEN_EXISTING     3
%define OPEN_ALWAYS       4
%define TRUNCATE_EXISTING 5


; Symbols imported from syscall.asm
extern syscall
extern malloc
extern mfree
extern assert
extern exit
extern printf
extern sysinit

extern StdOutputHandle
extern CreateFileA
extern ExitProcess
extern WriteFile
extern CloseHandle

global _start                                 ; Export symbols. The entry point

section .data                                   ; Initialized data segment
    message         db "Message from WriteFile", 0Dh, 0Ah
    startup_msg     db "Startup code version %d.%d.%d", 0Dh, 0Ah, 00h
    test4par        db "Should be numbers 2-4 here: %d, %d, %d", 0Dh, 0Ah, 00h
    test5par        db "Should be numbers 2-5 here: %d, %d, %d, %d", 0Dh, 0Ah, 00h
    test6par        db "Should be numbers 2-6 here: %d, %d, %d, %d, %d", 0Dh, 0Ah, 00h
    test8par        db "Should be numbers 2-7 here: %d, %d, %d, %d, %d, %d, %d", 0Dh, 0Ah, 00h
    axmess          db "... rax = 0x%X", 0Dh, 0Ah, 00h
    printbxmess     db "... rbx = 0x%X", 0Dh, 0Ah, 00h
    sp_mess         db "...  sp = 0x%X", 0Dh, 0Ah, 00h
    assert_true_mess   db "==== Assert true message, x=%d",0Dh, 0Ah, 00h
    assert_false_mess  db "==== Assert false message, x=%d",0Dh, 0Ah, 00h
    assert_args_mess   db "==== Assert false with 8 arguments, %d, %d, %d, %d, %d, %d",0Dh, 0Ah, 00h
    write_file_message db "This is from WriteFile using StdOutputHandle", 0Dh, 0Ah, 00h
    len1               EQU  $-write_file_message
    write_message      db "This is from WriteFile using opened file", 0Dh, 0Ah, 00h
    len2               EQU  $-write_message
    file_name          db "testfile.txt", 00h


section .bss                                    ; Uninitialized data segment

alignb 8
    heap            resq 1
    handle          resq 1
    readback        resq 1
    written         resq 1

section .text

print_ax:
    push axmess
    push rax
    mov rbx, 2*8
    mov rdi, printf
    call syscall
    add sp, 2*8
    ret

print_bx:
    push printbxmess
    push rbx
    mov rbx, 2*8
    mov rdi, printf
    call syscall
    add sp, 2*8
    ret

print_sp:
    push sp_mess
    push rsp
    mov rbx, 2*8
    mov rdi, printf
    call syscall
    add sp, 2*8
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

    push test8par              ; 1st parameter
    push 2                      ; 2nd parameter
    push 3                      ; 3rd parameter
    push 4                      ; 4th parameter
    push 5                      ; 5th parameter
    push 6                      ; 6th parameter
    push 7                      ; 7th parameter
    push 8                      ; 7th parameter
    mov rbx, 8*8                  ; Number of parameters on stack
    mov rdi, printf             ; Address to call
    call syscall
    add sp, 8*8

    call print_sp

    mov rax, 4096
    call malloc
    mov [heap], rax

    ; Store value to heap
    mov rdi, [heap]
    mov qword [rdi], 0x123456
    ; Read back from heap
    mov rax, [rdi]
    call print_ax

    ; Test mfree. Should give rax=1 after call to mfree
    mov rax, [heap]
    call mfree
    call print_ax

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

    ; Test assert with many arguments
    push false
    push assert_args_mess
    push 3
    push 4
    push 5
    push 6
    push 7
    push 8
    mov rbx, 8*8
    call assert
    add sp, 8*8

    ; Test assert with one arguments (no message)
    push false
    mov rbx, 8
    call assert
    add sp, 8

    call  print_sp

    ; Test using WriteFile
    push  qword [StdOutputHandle]        ; 1st parameter is the handle
    push  write_file_message             ; 2nd parameter is a pointer to the text to be written
    push  len1                           ; 3rd parameter is the number of bytes to write
    push  0                              ; 4th parameter is a pointer to the variable receiving the number of bytes written.
    push  0                              ; 5th parameter is a pointer to the lpOverlapped structure (or nil).
    mov   rdi, WriteFile                 ; Call the WriteFile function found in kernel32.dll (must be linked to)
    mov   rbx, 5*8
    call  syscall
    add   rsp, 5*8

    call  print_sp

    ; Test create file
    push  file_name,
    push  qword 0xc0000000  ; dwDesiredAccess, here read+write
    push  0                 ; dwShareMode, 0 = no sharing
    push  0                 ; lpSecurityAttributes, 0 = no sharing and default security
    push  CREATE_ALWAYS     ; dwCreationDisposition,
    push  0x80              ; dwFlagsAndAttributes, 0x80 is normal attributes
    push  0                 ;  hTemplateFile
    mov   rdi, CreateFileA  ; Call the WriteFile function found in kernel32.dll (must be linked to)
    mov   rbx, 7*8
    call  syscall
    add   rsp, 7*8
    mov  [handle], rax

    call  print_ax
    call  print_sp

    ; Write Write
    push qword [handle]
    push write_message
    push len2
    push 0
    push 0
    mov   rdi, WriteFile                 ; Call the WriteFile function found in kernel32.dll (must be linked to)
    mov   rbx, 5*8
    call  syscall
    add   rsp, 5*8

    ; Close file
    push rax
    mov  rdi, CloseHandle
    mov  rbx, 1*8
    call syscall
    add  rsp, 1*8

    call print_sp

    ; Exit with error code 1
    mov   rax, 1234
    call  exit
