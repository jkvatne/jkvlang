; test_assember.asm
;
; This is a file used to verify the assmembler setup and to test 
; calling system files.

%define false 0
%define true  1
;%define CREATE_NEW        1
%define CREATE_ALWAYS     2
;%define OPEN_EXISTING     3
;%define OPEN_ALWAYS       4
;%define TRUNCATE_EXISTING 5
;%define FORMAT_MESSAGE_FROM_SYSTEM  4096

; Symbols imported from syscall.asm
extern syscall
extern malloc
extern mfree
extern assert
extern exit
extern printf
extern sysinit
extern error
extern error_len
extern get_win_error
extern print

; Symbols from kernel32
extern StdOutputHandle
extern CreateFileA
extern ExitProcess
extern WriteFile
extern CloseHandle
extern GetLastError
extern FormatMessageA

; Export symbols
global _start          ; The entry point

;---------------------------------------------
section .bss          ; Uninitialized data segment
;---------------------------------------------

alignb 8
heap            resq 1
handle          resq 1

;---------------------------------------------
section .rodata        ;  Read only data
;---------------------------------------------


print_msg          db "Message from print", 0Dh, 0Ah, 00h
startup_msg        db "Startup code version %d.%d.%d", 0Dh, 0Ah, 00h
test4par           db "Should be numbers 2-4 here: %d, %d, %d", 0Dh, 0Ah, 00h
test5par           db "Should be numbers 2-5 here: %d, %d, %d, %d", 0Dh, 0Ah, 00h
test6par           db "Should be numbers 2-6 here: %d, %d, %d, %d, %d", 0Dh, 0Ah, 00h
test10par          db "Should be numbers 2-10 here: %d, %d, %d, %d, %d, %d, %d, %d, %d", 0Dh, 0Ah, 00h
axmess             db "... rax = 0x%X", 0Dh, 0Ah, 00h
sp_mess            db "...  sp = 0x%X", 0Dh, 0Ah, 00h
assert_true_mess   db "==== Assert true message, x=%d",0Dh, 0Ah, 00h
assert_false_mess  db "==== Assert false message, x=%d",0Dh, 0Ah, 00h
assert_args_mess   db "==== Assert false with arguments 3-10, %d, %d, %d, %d, %d, %d, %d, %d",0Dh, 0Ah, 00h
write_file_message db "This is from WriteFile using StdOutputHandle", 0Dh, 0Ah, 00h
len1               EQU  $-write_file_message
write_message      db "This is from WriteFile using opened file", 0Dh, 0Ah, 00h
len2               EQU  $-write_message
file_name          db "testfile.txt", 00h
str1               db "!!!!!!!!!!!!!!!!!!!!!!! str1",0Dh,0Ah,00h

;---------------------------------------------
section .text
;---------------------------------------------

; Print the contents of the rax register using printf
print_ax:
    push rax            ; Value to be printed
    mov rax, axmess
    mov rbx, 1*8
    mov rdi, printf
    call syscall
    add sp, 1*8
    ret

; Print the contents of the rsp register using printf
print_sp:
    push rsp           ; Value to be printed
    mov rax, sp_mess   ; Message at top of stack
    mov rbx, 1*8       ; Stack size is 8 bytes
    mov rdi, printf    ; system function to call
    call syscall
    add sp, 1*8
    ret

; Primary entry point for exe file
_start:
    push rbp                    ; Prologue: Save frame pointer
    mov rbp, rsp                ; Prologue: Setup new frame pointer.
    and rsp, -16                ; Align stack by clearing the 4 lsb
    sub rsp, 32                 ; Reserve shadow space

    call print_sp

    ; Print a startup message with integer parameters using the prinf from msvcrt.dll
    ; This is a direct call, not using syscall.asm, must be linked with msvcrt.dll
    mov rcx, startup_msg        ; First argument: format string
    mov rdx, 0                  ; Second argument: number
    mov r8,  0                  ; Third argument: number
    mov r9,  1                  ; Fourth argument: number
    call printf                 ; Call printf

    call sysinit

    call print_sp

    ; Test using print
    mov rax, print_msg          ; 1st parameter
    mov rbx, 0
    call print

    ; Test using syscall
    push 4                      ; 4th parameter
    push 3                      ; 3rd parameter
    push 2                      ; 2nd parameter
    mov rax, test4par           ; 1st parameter
    mov rbx, 3*8                ; Number of parameters on stack
    mov rdi, printf             ; Address to call
    call syscall
    add sp, 3*8

    call print_sp

    ; Test using syscall
    push 5                      ; 5th parameter
    push 4                      ; 4th parameter
    push 3                      ; 3rd parameter
    push 2                      ; 2nd parameter
    mov rax, test5par           ; 1st parameter
    mov rbx, 4*8                ; Number of parameters on stack
    mov rdi, printf             ; Address to call
    call syscall
    add sp, 4*8

    call print_sp

    ; Test using syscall
    push 6                      ; 6th parameter
    push 5                      ; 5th parameter
    push 4                      ; 4th parameter
    push 3                      ; 3rd parameter
    push 2                      ; 2nd parameter
    mov rax, test6par           ; 1st parameter
    mov rbx, 5*8                ; Number of parameters on stack
    mov rdi, printf             ; Address to call
    call syscall
    add sp, 5*8

    call print_sp

    push 10                     ; 10th parameter
    push 9                      ; 9th parameter
    push 8                      ; 8th parameter
    push 7                      ; 7th parameter
    push 6                      ; 6th parameter
    push 5                      ; 5th parameter
    push 4                      ; 4th parameter
    push 3                      ; 3rd parameter
    push 2                      ; 2nd parameter
    mov rax, test10par           ; 1st parameter
    mov rbx, 9*8                  ; Number of parameters on stack
    mov rdi, printf             ; Address to call
    call syscall
    add sp, 9*8

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
    push 103                 ; Last parameter
    push assert_false_mess   ; String
    mov rax, 0               ; Boolean in axr
    mov rbx, 2*8
    call assert
    add sp, 2*8

    call print_sp

    ; Test assert true
    push 101
    push assert_true_mess
    mov rax, 1
    mov rbx, 2*8
    call assert
    add sp, 2*8

    call print_sp

    ; Test assert with 10 arguments
    push 10
    push 9
    push 8
    push 7
    push 6
    push 5
    push 4
    push 3
    push assert_args_mess
    mov rax, false
    mov rbx, 9*8
    call assert
    add sp, 9*8

    ; Test assert fail with no message
    mov rax, 0
    mov rbx,0
    call assert
    call  print_sp

    ; Test assert as in hello.jkv
    mov rax, 1234
    push rax                                 ; Argument 3
    mov rax, str1
    push rax                                 ; Argument 2
    mov rax, false
    mov rbx, 16
    call assert
    add rsp, 16                             ; Remove arguments

    ; Test using WriteFile
    push  0                              ; 5th parameter is a pointer to the lpOverlapped structure (or nil).
    push  0                              ; 4th parameter is a pointer to the variable receiving the number of bytes written.
    push  len1                           ; 3rd parameter is the number of bytes to write
    push  write_file_message             ; 2nd parameter is a pointer to the text to be written
    mov rax,  qword [StdOutputHandle]        ; 1st parameter is the handle
    mov   rdi, WriteFile                 ; Call the WriteFile function found in kernel32.dll (must be linked to)
    mov   rbx, 4*8
    call  syscall
    add   rsp, 4*8

    call  print_sp

    ; Test create file
    push  0                 ;  hTemplateFile
    push  0x80              ; dwFlagsAndAttributes, 0x80 is normal attributes
    push  CREATE_ALWAYS     ; dwCreationDisposition,
    push  0                 ; lpSecurityAttributes, 0 = no sharing and default security
    push  0                 ; dwShareMode, 0 = no sharing
    mov rdi, 0xc0000000
    push  rdi               ; dwDesiredAccess, here read+write
    mov rax, file_name
    mov   rdi, CreateFileA  ; Call the WriteFile function found in kernel32.dll
    mov   rbx, 6*8
    call  syscall
    add   rsp, 6*8
    mov  [handle], rax

    call  print_ax
    call  print_sp

    ; Write to file
    push 0
    push 0
    push len2
    push write_message
    mov rax, qword [handle]
    mov   rdi, WriteFile     ; Call the WriteFile function found in kernel32.dll
    mov   rbx, 4*8
    call  syscall
    add   rsp, 4*8

    ; Close file
    mov rax, qword [handle]
    mov  rdi, CloseHandle   ; Call the CloseHandle function found in kernel32.dll
    mov  rbx, 0
    call syscall

    ; Test create file with error (filename=nil)
    push  0                 ;  hTemplateFile
    push  0x80              ; dwFlagsAndAttributes, 0x80 is normal attributes
    push  CREATE_ALWAYS     ; dwCreationDisposition,
    push  0                 ; lpSecurityAttributes, 0 = no sharing and default security
    push  0                 ; dwShareMode, 0 = no sharing
    mov rdi, 0xc0000000
    push  rdi               ; dwDesiredAccess, here read+write
    mov rax, 0
    mov   rdi, CreateFileA  ; Call the WriteFile function found in kernel32.dll
    mov   rbx, 6*8
    call  syscall
    add   rsp, 6*8
    mov  [handle], rax

    add rax, 1
    jnz create_was_ok

    call get_win_error


    ; Print error message
    mov rcx, error        ; First argument: format string
    call printf           ; Call printf

    mov ax, [error_len]
    call print_ax

create_was_ok:
    ; Close file
    push rax
    mov  rdi, CloseHandle   ; Call the CloseHandle function found in kernel32.dll
    mov  rbx, 1*8
    call syscall
    add  rsp, 1*8


    call print_sp

    ; Exit with error code 1234
    mov   rax, 1234
    call  exit

