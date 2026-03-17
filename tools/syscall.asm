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
; 15 r15   Preserved              Error pointer. 0 (nil) means ok.


; Errors
; CloseHandle  error=GetLastError when result=0
; WriteFile    error=GetLastError when result=0
; CreateFileA  error=GetLastError when result=-1
; ReadFile     error=GetLastError when result=0
; printf       error=GetLastError when result=-1 (or <0)

%define STD_INPUT_HANDLE  -10
%define STD_OUTPUT_HANDLE -11
%define STD_ERROR_HANDLE  -12
%define MAX_ERROR_LEN     40*8
%define FORMAT_MESSAGE_FROM_SYSTEM  4096

section .data
alignb 8
assert_failed  db "Assert failed without message",0Dh, 0Ah, 00h

; Exported symbols from syscall.asm
global syscall
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

section .bss
alignb 8
    StdOutputHandle resq 1
    StdErrorHandle  resq 1
    StdInputHandle  resq 1
    error           resb MAX_ERROR_LEN

section .text

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

; assert will verify that the first arbument (rax) is true (not 0)
; with optional additional parameters.
; The stack will contain <messageptr><arg1><arg2>..
; rbx should contain the size of the stack. (number of arguments-1) * 8.
; rax is already the value to be tested
assert:
    or rax, rax             ; Set z-flag if rax is zero
    jz L1                   ; Jump if the bool argument was false
    ret                     ; Returns if assert(true)
L1: mov rcx, assert_failed  ; Load default error message to be used if assert has no message
    ; fallthrough to print

; print is the local version of fprintf
; Arg count should be in rbx
; The last parameter in rax
; All other parameters pushed on stack.
print:
    mov rdi, printf
    ; fallthrough to syscall

; syscall will call any dll function that is reachable
; The address of the function should be in rdi, arg count *8 in rbx
; rax is the first parameter
syscall:
    push rbp
    mov rbp, rsp          ; Setup new frame pointer
    mov r15, 0            ; Default to no error
    and rsp, -16          ; Align stack by clearing the 4 lsb
    sub rsp, 80           ; Reserve space for arguments to the called function

    or rbx, rbx
    jz docall

    mov rcx, rax          ; cx = First argument: format string
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

get_win_error:
    push rbp                         ; Prologue: Save frame pointer
    mov rbp, rsp                     ; Prologue: Setup new frame pointer.
    and rsp, -16                     ; Align stack by clearing the 4 lsb
    sub rsp, 32                      ; Reserve shadow space
    ; Get last windows error
    call GetLastError
    ; Get text for errornumber in rax
  	push FORMAT_MESSAGE_FROM_SYSTEM ; |FORMAT_MESSAGE_FROM_HMODULE|FORMAT_MESSAGE_ARGUMENT_ARRAY
  	push 0         ; modntdll.Handle(),
  	push rax       ; Error no
  	push 0         ; langID(LANG_ENGLISH, SUBLANG_ENGLISH_US),
  	push error     ; error string buffer
  	push 300       ; Length of error string buffer
  	push 0         ; Arguments
    mov rdi, FormatMessageA
    mov rbx, 7*8
    call syscall
    add rsp, 7*8
    ; Set pointer to error message in r15
    mov r15, error
    ; call GetLastError   ; Check if FormatMessageA failed
    ; Epilogue
    mov rsp, rbp
    pop rbp
    ret