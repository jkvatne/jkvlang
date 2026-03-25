%include "c:\doc\compiler\lib\sysinit.asm"

; Symbols from kernel32
extern ExitProcess
extern GetProcessHeap
extern HeapAlloc
extern HeapFree
extern GetStdHandle
extern GetLastError
extern FormatMessageA
extern StdOutputHandle
extern CreateFileA
extern ExitProcess
extern WriteFile
extern CloseHandle

; Symbols from msvcrt.dll
extern printf

%define MAX_ERROR_LEN     40*8
%define FORMAT_MESSAGE_FROM_SYSTEM  4096
%define CREATE_ALWAYS     2
%define false 0
%define true  1

;-------------
section .bss
;-------------
alignb 8
error_len       resq 1              ; 16 bit string length
error           resq MAX_ERROR_LEN
heap            resq 1
handle          resq 1

;-------------
section .rodata
;-------------
crlf_str  db 0Ah, 00h
assert_mess        db "Assert failed", 00h
print_msg          db "Message from print", 0Ah, 00h
startup_msg        db "Startup code version %d.%d.%d", 0Ah, 00h
test4par           db "Should be numbers 2-4 here: %d, %d, %d", 0Ah, 00h
test5par           db "Should be numbers 2-5 here: %d, %d, %d, %d", 0Ah, 00h
test6par           db "Should be numbers 2-6 here: %d, %d, %d, %d, %d", 0Ah, 00h
test10par          db "Should be numbers 2-10 here: %d, %d, %d, %d, %d, %d, %d, %d, %d", 0Ah, 00h
free_result        db "........Free got %d, expected 1.", 0Ah, 00h
heap_readback      db "........Readback from heap, expected 0x1234, got %0X", 0Ah, 00h
axmess             db "........rax = 0x%X", 0Ah, 00h
start_sp           db "........RSP at start = 0x%X", 0Ah, 00h
end_sp             db "........RSP at end = 0x%X", 0Ah, 00h
assert_true_mess   db "........Assert true message, x=%d", 00h
assert_false_mess  db "........Assert false message, x=%d", 00h
assert_args_mess   db "........Assert false with arguments 3-10, %d, %d, %d, %d, %d, %d, %d, %d", 00h
write_file_message db "This is from WriteFile using StdOutputHandle", 0Ah, 00h
len1               EQU  $-write_file_message
write_message      db "........This is from WriteFile using opened file", 0Ah, 00h
len2               EQU  $-write_message
file_name          db "testfile.txt", 00h
str1               db "str1",0Dh,0Ah,00h


;-------------
section .text
;-------------

global main
main:       
    mov rbp, rsp; for correct debugging

    ; Print a startup message with integer parameters using the prinf from msvcrt.dll
    ; This is a direct call, must be linked with msvcrt.dll
    mov rcx, startup_msg        ; First argument: format string
    mov rdx, 0                  ; Second argument: number
    mov r8,  0                  ; Third argument: number
    mov r9,  1                  ; Fourth argument: number
    call printf                 ; Call printf

    call sysinit
   
    push rsp                    ; Value to be printed
    mov rax, start_sp           ; Message at top of stack
    mov rbx, 8                  ; Stack size is 8 bytes
    call print                  ; system function to call
    add sp, 8

    ; Test using print
    mov rax, print_msg          ; 1st parameter
    sub rax, 4                  ; Correct for length here
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

    push 10                     ; 10th parameter
    push 9                      ; 9th parameter
    push 8                      ; 8th parameter
    push 7                      ; 7th parameter
    push 6                      ; 6th parameter
    push 5                      ; 5th parameter
    push 4                      ; 4th parameter
    push 3                      ; 3rd parameter
    push 2                      ; 2nd parameter
    mov rax, test10par          ; 1st parameter
    mov rbx, 9*8                ; Number of parameters on stack
    mov rdi, printf             ; Address to call
    call syscall
    add sp, 9*8

    mov rax, 4096
    call _alloc
    mov [heap], rax

    ; Store value to heap
    mov rdi, [heap]
    mov qword [rdi], 0x123456
    ; Read back from heap
    mov rax, [rdi]
    push rax                   ; Value to be printed
    mov rax, heap_readback     ; Message at top of stack
    mov rbx, 8                 ; Stack size is 8 bytes
    call print                 ; system function to call
    add sp, 8
    
    ; Test mfree. Should give rax=1 after call to mfree
    mov rax, [heap]
    call _free
    push rax                   ; Value to be printed
    mov rax, free_result       ; Message at top of stack
    mov rbx, 8                 ; Stack size is 8 bytes
    call print                 ; system function to call
    add sp, 8

    ; Test assert false
    push 103                  ; Last parameter
    push assert_false_mess    ; String
    mov rax, 0                ; Boolean in axr
    mov rbx, 2*8
    call assert
    add sp, 2*8

    ; Test assert true
    push 101
    push assert_true_mess
    mov rax, 1
    mov rbx, 2*8
    call assert
    add sp, 2*8

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

    call _win_error


    ; Print error message
    mov rcx, error        ; First argument: format string
    call printf           ; Call printf

    mov ax, [error_len]
    ; call print_ax

create_was_ok:
    ; Close file
    push rax
    mov  rdi, CloseHandle   ; Call the CloseHandle function found in kernel32.dll
    mov  rbx, 1*8
    call syscall
    add  rsp, 1*8

    push rsp           ; rsp is value to be printed
    mov rax, end_sp    ; Format string
    mov rbx, 8         ; Stack size is 8 bytes
    call print         ; system function to call
    add sp, 8

    ; Exit with error code 1234
    mov   rax, 1234
    call  exit

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
; The first parameter in rax, that is the format string
; Note that the format string has 8 bytes initial length/capacity
; All other parameters pushed on stack.
print:
    add rax, 8
    mov rdi, printf
    ; fallthrough to syscall

; syscall will call any dll function that is reachable
; The address of the function should be in rdi, arg count *8 in rbx
; rax is the first parameter
syscall:
    push rbp              ; Save old frame pointer  
    mov rbp, rsp          ; Setup new frame pointer
    and rsp, -16          ; Align stack by clearing the 4 lsb
    sub rsp, 96           ; Reserve space for arguments to the called function
    mov r15, 0            ; Default to no error

    mov rcx, rax          ; rcx = First argument: format string
    or rbx, rbx
    jz .L3

    mov rdx, [rbp+16]    ; dx = Second argument
    sub rbx, 8
    jc .L3

    mov r8,  [rbp+24]    ; r8 = Third argument
    sub rbx, 8
    jc .L3

    mov r9,  [rbp+32]    ; r9 = Forth argument
    sub rbx, 8
    jc .L3

    mov rsi, [rbp+40]    ; Fifth argument onto stack
    mov [rsp+32], rsi
    sub rbx, 8
    jc .L3

    mov rsi, [rbp+48]
    mov [rsp+40], rsi     ; Sixth argument onto stack
    sub rbx, 8
    jc .L3

    mov rsi, [rbp+56]
    mov [rsp+48], rsi     ; Seventh argument onto stack
    sub rbx, 8
    jc .L3

    mov rsi, [rbp+64]
    mov [rsp+56], rsi     ; Eight argument onto stack
    sub rbx, 8
    jc .L3

    mov rsi, [rbp+72]
    mov [rsp+64], rsi     ; Nineth argument onto stack
    sub rbx, 8
    jc .L3

    mov rsi, [rbp+80]
    mov [rsp+72], rsi     ; Tenth argument onto stack

.L3:
    call rdi
    leave
    ret



; _alloc returns in rax a pointer to the allocated memory or null.
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

; _free will free the memory pointed to by rax.
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


; _win_error will call GetLastError and convert it to a string
; The string is in the global variable error.
; Also, a pointer to the error string is put into r15
_win_error:
    ; Get last windows error into rax
    call GetLastError
    push 0         ; Arg 7: arguments. Not used
    push 300       ; Arg 6: Length of error string buffer
    mov rsi, error
    push rsi       ; Arg 5: error string buffer
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
    or bx, bx            ; Check if bx=0 (no string given)
    jnz _L5 
    mov bx, 8
    mov rcx, assert_mess    
    jmp _L4
_L5:    

    mov rcx, [rbp+16]    ; rcx = First argument: format string
    add rcx, 8           ; Skip length/capacity of string
    sub rbx, 8
    or rbx, rbx
    jz _L2
_L4:
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

