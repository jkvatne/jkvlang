; sys.asm  Contains file IO functions

extern CreateFileA
extern CreateFileW
extern ReadFile
extern WriteFile
extern CloseHandle

%define CREATE_NEW        1    ; Fail if file exists
%define CREATE_ALWAYS     2    ; Truncate old file if it exists
%define OPEN_EXISTING     3    ; Fails if file exists
%define OPEN_ALLWAYS      4
%define TRUNCATE_EXISTING 5    ; Fails if file exists

global _create_file
global _write_file
global _read_file
global _close_file
global _lptr
global _cptr
global _len

;-------------
section .text
;-------------

global _create_file

_create_file:
    mov rdi, CreateFileA
    mov bx, 8*7
    call _syscall
    add rax, 1
    jnz .L2
    mov r15, 107
.L2:
    sub rax, 1
    mov [rsp+8*8], rax
    ret

_read_file:
    mov rdi, ReadFile
    mov bx, 8*5
    call _syscall
    mov [rsp+8*6], rax
    or rax,rax
    jnz .L1
    mov r15,106
.L1:
    ret

_write_file:
    mov rdi, WriteFile
    mov bx, 8*5
    call _syscall
    mov [rsp+8*6], rax
    ret

_close_file:
    mov rdi, CloseHandle
    mov bx, 8
    call _syscall
    ret

_cptr:
    mov rax, [rsp+8]
    add rax, 8
    mov [rsp+16], rax
    ret

_lptr:
    mov rax, [rsp+8]
    mov [rsp+16], rax
    ret

_len:
    mov rax, [rax]
    and rax, 0x7FFFFFFF
    ret

