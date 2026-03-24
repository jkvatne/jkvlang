
global main
global malloc
global mfree
global assert
global exit
global sysinit
global StdOutputHandle
global StdErrorHandle
global StdInputHandle
global error
global get_win_error
global print


; Symbols from kernel32
extern ExitProcess
extern GetProcessHeap
extern HeapAlloc
extern HeapFree
extern GetStdHandle
extern GetLastError
extern FormatMessageA

; Symbols from msvcrt.dll
extern printf
%define STD_INPUT_HANDLE  -10
%define STD_OUTPUT_HANDLE -11
%define STD_ERROR_HANDLE  -12
%define MAX_ERROR_LEN     40*8

section .rodata
    crlf_str  db 0Dh, 0Ah, 00h

section .bss
    alignb 8
    StdOutputHandle resq 1
    StdErrorHandle  resq 1
    StdInputHandle  resq 1
    error_len       resq 1              ; 16 bit string length
    error           resq MAX_ERROR_LEN


section .text
sysinit:
    ; sysinit will initialize the console handles
    push rbp                         ; Prologue: Save frame pointer
    mov rbp, rsp                     ; Prologue: Setup new frame pointer.
    and rsp, -16                     ; Align stack by clearing the 4 lsb
    sub rsp, 32                      ; Reserve shadow space

    ; Load the handle for standard output
    mov   ecx, STD_OUTPUT_HANDLE
    call  GetStdHandle
    mov   [rel StdOutputHandle], rax

    mov   ecx, STD_ERROR_HANDLE
    call  GetStdHandle
    mov   [rel StdErrorHandle], rax

    mov   ecx, STD_INPUT_HANDLE
    call  GetStdHandle
    mov   [rel StdInputHandle], rax

    mov qword [error], 0
    mov [error_len], word 0
    leave   
    ret
    
main:       
   mov rbp, rsp; for correct debugging
   call sysinit
   
   xor rax, rax                     ; Error code = 0
   ret
   

; exit have one parameter - the error code, found in rax
exit:
    push rbp                         ; Prologue: Save frame pointer
    mov rbp, rsp                     ; Prologue: Setup new frame pointer.
    and rsp, -16                     ; Align stack by clearing the 4 lsb
    sub rsp, 32                      ; Reserve shadow space
    mov rcx, rax
    call ExitProcess
    leave
    ret   
    
    
; print is the local version of fprintf
; Arg count should be in rbx
; The last parameter in rax
; All other parameters pushed on stack.
print:
    add rax, 4
    mov rdi, printf
    ; fallthrough to syscall

; syscall will call any dll function that is reachable
; The address of the function should be in rdi, arg count *8 in rbx
; rax is the first parameter
syscall:
    push rbp
    mov rbp, rsp          ; Setup new frame pointer
    and rsp, -16          ; Align stack by clearing the 4 lsb
    sub rsp, 96           ; Reserve space for arguments to the called function
    mov r15, 0            ; Default to no error

    mov rcx, rax          ; rcx = First argument: format string
    or rbx, rbx
    jz _L3

    mov rdx, [rbp+16]    ; dx = Second argument
    sub rbx, 8
    jc _L3

    mov r8,  [rbp+24]    ; r8 = Third argument
    sub rbx, 8
    jc _L3

    mov r9,  [rbp+32]    ; r9 = Forth argument
    sub rbx, 8
    jc _L3

    mov rsi, [rbp+40]    ; Fifth argument onto stack
    mov [rsp+32], rsi
    sub rbx, 8
    jc _L3

    mov rsi, [rbp+48]
    mov [rsp+40], rsi     ; Sixth argument onto stack
    sub rbx, 8
    jc _L3

    mov rsi, [rbp+56]
    mov [rsp+48], rsi     ; Seventh argument onto stack
    sub rbx, 8
    jc _L3

    mov rsi, [rbp+64]
    mov [rsp+56], rsi     ; Eight argument onto stack
    sub rbx, 8
    jc _L3

    mov rsi, [rbp+72]
    mov [rsp+64], rsi     ; Nineth argument onto stack
    sub rbx, 8
    jc _L3

    mov rsi, [rbp+80]
    mov [rsp+72], rsi     ; Tenth argument onto stack

_L3:
    call [rdi]
    leave
    ret


; assert will verify that the first arbument (rax) is true (not 0)
; with optional additional parameters.
; The stack will contain <messageptr><arg1><arg2>..
; rbx should contain the size of the stack. (number of arguments-1) * 8.
; rax is already the value to be tested
; NB: Assert will append CRLF after the message.
assert:
    push rbp
    mov rbp, rsp          ; Setup new frame pointer
    and rsp, -16          ; Align stack by clearing the 4 lsb
    sub rsp, 96           ; Reserve space for arguments to the called function

    or rax, rax           ; Set z-flag if rax is zero
    jz _L1                 ; Jump if the bool argument was false
    leave
    ret                   ; Returns if assert(true)
_L1:
    mov rcx, [rbp+16]    ; rcx = First argument: format string
    add rcx, 4
    sub rbx, 8
    or rbx, rbx
    jz _L2

    mov rdx, [rbp+24]    ; dx = Second argument
    sub rbx, 8
    jc _L2

    mov r8,  [rbp+32]    ; r8 = Third argument
    sub rbx, 8
    jc _L2

    mov r9,  [rbp+40]    ; r9 = Forth argument
    sub rbx, 8
    jc _L2

    mov rsi, [rbp+48]    ; Fifth argument onto stack
    mov [rsp+32], rsi
    sub rbx, 8
    jc _L2

    mov rsi, [rbp+56]
    mov [rsp+40], rsi     ; Sixth argument onto stack
    sub rbx, 8
    jc _L2

    mov rsi, [rbp+64]
    mov [rsp+48], rsi     ; Seventh argument onto stack
    sub rbx, 8
    jc _L2

    mov rsi, [rbp+72]
    mov [rsp+56], rsi     ; Eight argument onto stack
    sub rbx, 8
    jc _L2

    mov rsi, [rbp+80]
    mov [rsp+64], rsi     ; Nineth argument onto stack
    sub rbx, 8
    jc _L2

    mov rsi, [rbp+88]
    mov [rsp+72], rsi     ; Tenth argument onto stack
    jmp _L2

_L2:
    call printf

    mov rcx, crlf_str
    call printf

    leave
    ret


; malloc returns in rax a pointer to the allocated memory or null.
; One argument is needed, in rax, and that is the requested size in bytes.
; Returns the pointer in rax
_alloc:
    push rbp                         ; Prologue: Save frame pointer
    mov rbp, rsp                     ; Prologue: Setup new frame pointer.
    and rsp, -16                     ; Align stack by clearing the 4 lsb
    sub rsp, 32                      ; Reserve shadow space
    mov rdi, rax                     ; Save size into rdi
    call GetProcessHeap
    mov rcx, rax                     ; Argument 1, Handle from GetProcessHeap moved into rcx
    mov rdx, 8                       ; Arbument 2, Flags into rdx, 8 means allocated memory is zeroed
    mov r8, rdi
    call HeapAlloc
    leave                            ; Epilogue: Restore old frame pointer
    ret                              ; Epilogue: Return

; free will free the memory pointed to by rax.
; It assumes it is from the default Process Heap returned from GetProcessHeap
; No return value.
_free:
    push rbp
    mov rbp, rsp
    and rsp, -16                     ; Align stack by clearing the 4 lsb
    sub rsp, 40                      ; Reserve shadow space
    mov rdi, rax                     ; Save ptr in rdi
    call GetProcessHeap
    mov rcx, rax                     ; Argument 1, Handle from GetProcessHeap moved into rcx
    mov rdx, 0                       ; Argument 2, flags into rdx, 0 must be used
    mov r8, rdi                      ; Argument 3, move memory pointer into r8
    call HeapFree
    leave
    ret
