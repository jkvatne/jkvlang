
; Symbols from kernel32
extern CreateFileA
extern CreateFileW
extern ReadFile
extern WriteFile
extern CloseHandle

%define CREATE_ALWAYS     2    ; Truncate old file if it exists
%define CREATE_NEW        1    ; Fail if file exists
%define OPEN_ALLWAYS      4
%define OPEN_EXISTING     3    ; Fails if file exists
%define TRUNCATE_EXISTING 5    ; Fails if file exists

global _create_file
global _write_file
global _read_file
global _close_file


;-------------
section .text
;-------------

global _create_file

_create_file:
    mov rdi, CreateFileA
    mov bx, 8*7
    jmp _syscall

_read_file:
    ; Set pointer to the number of bytes read
    lea rax, [rsp+48]
    mov [rsp+24], rax
    mov rdi, ReadFile
    mov bx, 8*5
    call _syscall

_write_file:
    mov rdi, WriteFile
    mov bx, 8*5
    jmp _syscall

_close_file:
    mov rdi, CloseHandle
    mov bx, 8
    jmp _syscall

_cptr:
    mov rax, [rsp+8]
    add rax, 8            ; cptr(). Point to the string itself
    mov [rsp+16], rax
    ret

_len:
    mov rax, [rax]
    and rax, 0x7FFFFFFF
    ret

