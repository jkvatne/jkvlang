; Windows system calls


; Register Windows ABI           JKV ABI
; 0  rax   Return value          Return value
; 1  rcx   First argument
; 2  rdx   Second argument
; 3  rbx   Preserved             Size of arguments on stack (bytes)
; 4  rsp   Stack pointer
; 5  rbp   Preserved
; 6  rsi   Preserved
; 7  rdi   Preserved             Function called for syscall
; 8  r8    Third argument
; 9  r9    Forth argument
; 10 r10   Used in syscall
; 11 r11   Used in syscall
; 12 r12   Preserved
; 13 r13   Preserved
; 14 r14   Preserved
; 15 r15   Preserved

%define STD_INPUT_HANDLE -10
%define STD_OUTPUT_HANDLE -11
%define STD_ERROR_HANDLE -12

section .data
    assert_failed  db "Assert failed",0Dh, 0Ah, 00h

; Exported symbols from syscall.asm
global syscall
global malloc
global mfree
global assert
global exit
global sysinit

; Symbols from kernel32
extern ExitProcess
extern GetProcessHeap
extern HeapAlloc
extern HeapFree
extern GetStdHandle

; Symbols from msvcrt.dll
extern printf

section .bss
alignb 8
    StdOutputHandle resq 1
    StdErrorHandle  resq 1
    StdInputHandle  resq 1
    written         resq 1

section .text

writefile:
    ; Test using WriteFile
    sub   RSP, 16                                  ; 5th parameter + align stack to a multiple of 16 bytes
    mov   RCX, qword [StdOutputHandle]             ; 1st parameter is the handle
    ;lea   RDX, [message]                           ; 2nd parameter is a pointer to the text to be written
    ;mov   R8, messlen                              ; 3rd parameter is the number of bytes to write
    lea   R9, [rel written]                        ; 4th parameter is a pointer to the variable receiving the number of bytes written.
    mov   qword [RSP + 32], 0                      ; 5th parameter is a pointer to the lpOverlapped structure (or nil).
    ;call  WriteFile                                ; Call the WriteFile function found in kernel32.dll (must be linked to)
    add   RSP, 16
    ret

; assert will verify that the first arbument is true (not 0)
; if ax is null, it will print an error message using printf,
; with optional additional parameters.
; The stack will contain <bool><messageptr><arg1><arg2>..
; rbx should contain the (number of arguments) * 8.
assert:
    mov rax, [rsp+rbx]      ; Load the bool argument to be tested
    or rax, rax             ; Set flags
    jz L1                   ; Jump if the bool argument fwas false
    ret                     ; Returns if assert(true)
L1: sub rbx, 8              ; Remove the first (bool) argument
    mov rcx, assert_failed  ; Load default error message if assert has no message
    mov rdi, printf         ; Call the printf function in msvcrt.dll
    jmp syscall

; malloc returns in rax a pointer to the allocated memory or null.
; One argument is needed, in rax, and that is the requested size in bytes.
; Returns the pointer in rax
malloc:
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
mfree:
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


; syscall will call any dll function that is reachable
; The address of the function should be in rdi, arg count *8 in rbx
syscall:
    push rbp
    mov rbp, rsp          ; Setup new frame pointer

    and rsp, -16          ; Align stack by clearing the 4 lsb
    sub rsp, 80           ; Reserve space for arguments to the called function

    or rbx, rbx
    jz docall

    mov rcx, [rbx+rbp+8]    ; cx = First argument: format string
    sub rbx, 8
    jc docall

    mov rdx, [rbx+rbp+8]    ; dx = Second argument
    sub rbx, 8
    jc docall

    mov r8,  [rbx+rbp+8]    ; r8 = Third argument
    sub rbx, 8
    jc docall

    mov r9,  [rbx+rbp+8]    ; r9 = Forth argument
    sub rbx, 8
    jc docall

    mov rsi, [rbx+rbp+8]    ; Fifth argument onto stack
    mov [rsp+32], rsi
    sub rbx, 8
    jc docall

    mov rsi, [rbx+rbp+8]
    mov [rsp+40], rsi     ; Sixth argument onto stack
    sub rbx, 8
    jc docall

    mov rsi, [rbx+rbp+8]
    mov [rsp+48], rsi     ; Seventh argument onto stack
    sub rbx, 8
    jc docall

    mov rsi, [rbx+rbp+8]
    mov [rsp+56], rsi     ; Eight argument onto stack
    sub rbx, 8

docall:
    call [rdi]
    leave
    ret

; exit have one parameter - the error code, found in ax
exit:
    push rbp                         ; Prologue: Save frame pointer
    mov rbp, rsp                     ; Prologue: Setup new frame pointer.
    and rsp, -16                     ; Align stack by clearing the 4 lsb
    sub rsp, 32                      ; Reserve shadow space
    mov rcx, rax
    call ExitProcess
    leave
    ret

; sysinit will initialize the console handles
sysinit:
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

    leave
    ret

