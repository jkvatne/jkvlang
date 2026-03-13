; runtime.asm

; This is the runtime for the jkv compiler, using the nasm assembler

%define STD_INPUT_HANDLE -10
%define STD_OUTPUT_HANDLE -11
%define STD_ERROR_HANDLE -12


extern WriteFile
extern ReadFile
extern CreateFileA
extern CloseHandle
extern GetStdHandle
extern ExitProcess
extern printf
extern GetCommandLineA
extern GetEnvironmentStringsA
extern GetProcessHeap

global _start          ; Exported entry point
global alloc          ; Allocate memory
global assert         ; Assert true or print message
global read_file
global write_file
global create_file
global erase_file
global make_dir
global get_std_handle

section .rodata

welcome     db "Compiler jkv v0.0.1", 0Dh, 0Ah, 00h
welcome_len EQU $-welcome 

section .bss                                    ; Uninitialized data segment
alignb 8
    StdOutputHandle resq 1
    StdErrorHandle  resq 1
    StdInputHandle  resq 1
    bytes_read      resq 1
    bytes_written   resq 1

section .text

get_std_handle:
    push rbp
    mov rbp, rsp
    and rsp, -16                  ; Align stack by clearing the 4 lsb
    sub rsp, 32                   ; Reserve shadow space
    mov rcx, [rbp+16]             ; Handle
    sub rso, 40                   ; Adjust stack
    call GetStdHandle
    add rsp, 40
    movq [rbp+24], rax            ;# Save return value to stack
    leave
    ret

;  write_file((fd int, p []byte) int
;      fd   File handle (int)
;      data Pointer to p's data
;      len  Length of p
;      return number of bytes written
write_file:
    push rbp
    movq rbp, rsp
    and rsp, -16

    mov rcx, [rbp+16]              ; Handle
    mov rdx, [rbp+24]              ; Data
    mov r8, [rbp+32]               ; len
    lea r9, [bytes_written]        ; Ptr to bytes written
    sub rsp, 80                    ; Adjust stack
    mov [rsp+32], 0                ; Arg 5
    call WriteFile
    add rsp, 80                    ; Restore stack
    mov rax,  [bytes_written]
    mov [rbp+48], rax              ; Save return value to stack
    leave
    ret

read_file:
    push rbp
    movq rbp, rsp
    and rsp, -16
    mov  rcx, [rbp+16]            ; Handle
    mov  rdx, [rbp+24]            ; Data
    movq r8, [rbp+8]              ; len
    lea  r9, [bytes_read]         ; pointer to bytes read variable
    sub rsp, 80                   ; Reserve stack
    mov [rsp+32], 0
    call ReadFile                 ; ReadFile(handle, &buffer, count, &bytesRead, &overlapped)
    pop rcx
    add rsp, 80                   ; Restore stack
    movq rax, [bytes_read]
    leave
    ret

; Windows CreateFileA
; rcx            LPCSTR                lpFileName,
; rdx            DWORD                 dwDesiredAccess, 0x4000_0000 is write
; r8             DWORD                 dwShareMode, 3 = read+write
; r9             LPSECURITY_ATTRIBUTES lpSecurityAttributes, Set to 0.
; stack          DWORD                 dwCreationDisposition, 4=Open allways
; stack          DWORD                 dwFlagsAndAttributes, 0x80 for normal files.
; stack          HANDLE                hTemplateFile = 0
; rax            HANDLE                Handle of the created file
;
; Stack upon entry, after setting up BP
; BP+72     Returned handle
; BP+64     dwDesiredAccess
; BP+56     dwShareMode
; BP+48     lpSecurityAttributes
; BP+40     dwCreateionDeisposition
; BP+32    dwFlagsAndAttributes
; BP+24    hTemplateFile

CreateFileA:
    push rbp
    movq rbp, rsp
    and  rsp, -16
    sub  rsp, 80             ; Reserve stack
    mov  rcx, [bp+16]        ; 1 lpFileName
    mov  rdx, [rbp+24]       ; 2 dwDesiredAccess (0x4000_0000 is write, 0x8000_0000 is read)
    cmp  rdx, 0
    jne cont1
    mov rdx, 0x80000000
cont1:
    mov  r8, [rbp+32]       ; 3 dwShareMode, 3 = read+write
    mov  r9, [rbp+40]       ; 4 lpSecurityAttributes attr. (0 allways)
    mov  rax, [rbp+48]      ; 5 dwCreationDisposition (4 allways)
    mov  rax, [rsp+32]      ; 5 CreateMode (4=Allways)
    mov  rax, [rbp+56]      ; 6 FlagsAndAttributes (0x80)
    mov  [rsp+40], rax      ; 6 FlagsAndAttributes (0x80)
    mov  rax, [rbp+64]      ; 7 Template file (0x00)
    mov  [rsp+48], rax)     ; 7 Template file (0x00)
    call CreateFileA        ; CreateFileA(&name, access, share, security, disp, flags, template) handle
    mov [rbp+72], rax)      ; Save return value from ax
    add rsp, 80             ; Adjust stack
    leave
    ret


close_file:
    push rbp
    movq rbp, rsp
    and  rsp, -16
    sub  rsp, 80             ; Reserve stack

    mov rcx, [rbp+16]        ; 1 Handle
    call CloseHandle
    movq [rbp+24], rax
    leave
    ret


    #mov 16(%rbp), %rax   # Prints -11
    #push %rax
    #call runtime.PrintHex64
    #pop %rax
    #call runtime.PrintLf
