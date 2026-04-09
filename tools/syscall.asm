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

section .rodata
crlf_str  db 0Dh, 0Ah, 00h

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

;------------
section .data
;------------
next32: dq 1234567890     ; 32 bits seed
next64  dq 1234567890     ; 64 bit seed. (Initial seed should be set using current time for better results)

;------------
section .bss
;------------
alignb 8
    StdOutputHandle resq 1
    StdErrorHandle  resq 1
    StdInputHandle  resq 1
    error_len       resq 1              ; 16 bit string length
    error           resq MAX_ERROR_LEN

;------------
section .text
;------------

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

    mov qword [error], 0
    mov [error_len], word 0
    leave
    ret

; get_win_error will call GetLastError and convert it to a string
; The string is in the global variable error.
; Also, a pointer to the error string is put into r15
get_win_error:
    ; Get last windows error into rax
    call GetLastError
  	push 0         ; Arg 7: arguments. Not used
  	push 300       ; Arg 6: Length of error string buffer
  	push error     ; Arg 5: error string buffer
  	push 0         ; Arg 4: langID(LANG_ENGLISH, SUBLANG_ENGLISH_US),
  	push rax       ; Arg 3: Error no
  	push 0         ; Arg 2: modntdll.Handle(),
  	mov rax, FORMAT_MESSAGE_FROM_SYSTEM | 0xFF ; |FORMAT_MESSAGE_FROM_HMODULE|FORMAT_MESSAGE_ARGUMENT_ARRAY ; 0xFF means no crlf
    mov rdi, FormatMessageA
    mov rbx, 6*8
    call syscall
    add rsp, 6*8
    mov [error_len], ax   ; 16 bit word
    ; Set pointer to error message in r15
    mov r15, error
    ret

; Generate a 32 bit quasi-random integer
; Warning: This generates a deterministic quasi-random sequence with a fixed seed
; It can be used for tests where a repeatable squence is needed
global qrand
qrand:
    mov eax, [next32]
    imul eax, 1103515245 ; Common multiplier (used by glibc)
    add eax, 12345       ; Common increment
    mov [next32], eax      ; Save new seed for next time
    ret

    ; --- 64-bit RDRAND implementation ---
global rand64:
rand64:
    rdrand rax          ; Try to generate a 64-bit random number into RAX
    jnc rand64          ; The Carry Flag (CF) is 0 if the generator was not ready; retry if so.
    ret                 ; RAX now contains a valid 64-bit random number

global qrand64:
qrand64:
    mov rax, [next64]              ; Load current seed
    mov rdx, 6364136223846793005   ; Multiplier 'a'
    mul rdx                        ; RAX = seed * a (lower 64 bits of result)
    mov rdx, 1442695040888963407   ; Add increment 'c'
    add rax,rdx
    mov [next64], rax              ; Update seed for next call
    ret