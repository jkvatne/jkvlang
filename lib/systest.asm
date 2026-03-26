%include "c:\doc\compiler\lib\syscall.asm"
%include "c:\doc\compiler\lib\sysinit.asm"
%include "c:\doc\compiler\lib\winerror.asm"
%include "c:\doc\compiler\lib\assert.asm"
%include "c:\doc\compiler\lib\alloc.asm"
%include "c:\doc\compiler\lib\exit.asm"
%include "c:\doc\compiler\lib\printf.asm"

; Symbols from kernel32
extern CreateFileA
extern ExitProcess
extern WriteFile
extern CloseHandle

%define CREATE_ALWAYS   2

;-------------
section .bss
;-------------
alignb 8
heap            resq 1
handle          resq 1
    
;-------------
section .rodata
;-------------
print_msg          db "........Message from print", 0Ah, 00h
startup_msg        db "Startup code version %d.%d.%d", 0Ah, 00h
test10par          db "........Should be numbers 2-10 here: %d, %d, %d, %d, %d, %d, %d, %d, %d", 0Ah, 00h
free_result        db "........Free got %d, expected 1.", 0Ah, 00h
heap_readback      db "........Readback from heap, expected 0x1234, got %0X", 0Ah, 00h
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

;-------------
section .text
;-------------

global main
main:       
    mov rbp, rsp; for correct debugging
    call print_startup_message   
    call sysinit
   
    ; Test the library's _printf() function found in printf.asm
    push rsp                    ; Value to be printed
    mov rax, start_sp           ; Message at top of stack
    mov rbx, 8                  ; Stack size is 8 bytes
    call _printf                ; system function to call
    add sp, 8                   ; Restore stack

    ; Test using _printf()
    mov rax, print_msg          ; 1st parameter
    mov rbx, 0                  ; Stack size is zero, only rax is used for format string
    call _printf

    ; Test using syscall directly, printing 10 parameters
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
    add rax, 8
    mov rbx, 9*8                ; Number of parameters on stack
    mov rdi, printf             ; Address to call
    call syscall
    add sp, 9*8

    ; Test alloc, allocating 4096 bytes
    mov rax, 4096
    call _alloc
    mov [heap], rax

    ; Store a value to the heap allocated area
    mov rdi, [heap]
    mov qword [rdi], 0x123456
    
    ; Read back the same value from heap and print it
    mov rax, [rdi]
    push rax                    ; Value to be printed
    mov rax, heap_readback      ; Message at top of stack
    mov rbx, 8                  ; Stack size is 8 bytes
    call _printf                ; System function to call
    add sp, 8                   ; Restore stack
    
    ; Test _free(). Should return rax=1 when successfull.
    mov rax, [heap]
    call _free
    push rax                    ; Value to be printed
    mov rax, free_result        ; Message at top of stack
    mov rbx, 8                  ; Stack size is 8 bytes
    call _printf                ; System function to call
    add sp, 8

    ; Test assert false
    push 103                    ; Last parameter
    push assert_false_mess      ; Format string
    mov rax, 0                  ; Boolean value to assert, in rax
    mov rbx, 2*8                ; Stack size is 16 bytes
    call _assert                ; Call assert
    add sp, 2*8                 ; Restore stack

    ; Test assert true
    push 101
    push assert_true_mess
    mov rax, 1
    mov rbx, 2*8
    call _assert
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
    mov rax, 0
    mov rbx, 9*8
    call _assert
    add sp, 9*8

    ; Test assert fail with no message
    mov rax, 0
    mov rbx,0
    call _assert

    ; Test using WriteFile
    push  0                              ; 5th parameter is a pointer to the lpOverlapped structure (or nil).
    push  0                              ; 4th parameter is a pointer to the variable receiving the number of bytes written.
    push  len1                           ; 3rd parameter is the number of bytes to write
    push  write_file_message             ; 2nd parameter is a pointer to the text to be written
    mov   rax,  qword [StdOutputHandle]  ; 1st parameter is the handle
    mov   rdi, WriteFile                 ; Call the WriteFile function found in kernel32.dll (must be linked to)
    mov   rbx, 4*8
    call  syscall
    add   rsp, 4*8

    ; Test create file
    push  0                     ;  hTemplateFile
    push  0x80                  ; dwFlagsAndAttributes, 0x80 is normal attributes
    push  CREATE_ALWAYS         ; dwCreationDisposition,
    push  0                     ; lpSecurityAttributes, 0 = no sharing and default security
    push  0                     ; dwShareMode, 0 = no sharing
    mov   rdi, 0xc0000000       ; dwDesiredAccess, here read+write 
    push  rdi                   ; push value (could not push value directly because of argument size)
    mov   rax, file_name
    mov   rdi, CreateFileA      ; Call the WriteFile function found in kernel32.dll
    mov   rbx, 6*8
    call  syscall
    add   rsp, 6*8
    mov   [handle], rax

    ; Test write to file
    push 0
    push 0
    push len2
    push write_message
    mov rax, qword [handle]
    mov   rdi, WriteFile        ; Call the WriteFile function found in kernel32.dll
    mov   rbx, 4*8
    call  syscall
    add   rsp, 4*8

    ; Close file
    mov rax, qword [handle]
    mov  rdi, CloseHandle   ; Call the CloseHandle function found in kernel32.dll
    mov  rbx, 0
    call syscall

    ; Test create file with error (filename=nil)
    push  0                 ; hTemplateFile
    push  0x80              ; dwFlagsAndAttributes, 0x80 is normal attributes
    push  CREATE_ALWAYS     ; dwCreationDisposition,
    push  0                 ; lpSecurityAttributes, 0 = no sharing and default security
    push  0                 ; dwShareMode, 0 = no sharing
    mov   rdi, 0xc0000000
    push  rdi               ; dwDesiredAccess, here read+write
    mov   rax, 0
    mov   rdi, CreateFileA  ; Call the WriteFile function found in kernel32.dll
    mov   rbx, 6*8
    call  syscall
    add   rsp, 6*8
    mov   [handle], rax

    add   rax, 1
    jnz   create_was_ok
    call  _win_error

    ; Print error message
    mov   rax, error         ; First argument: format string
    call  _println           ; Call println

create_was_ok:

    ; Close file
    push rax
    mov  rdi, CloseHandle   ; Call the CloseHandle function found in kernel32.dll
    mov  rbx, 1*8
    call syscall
    add  rsp, 1*8

    push rsp               ; rsp is value to be printed
    mov  rax, end_sp       ; Format string
    mov  rbx, 8            ; Stack size is 8 bytes
    call _printf           ; system function to call
    add  sp, 8

    ; Exit with error code 1234
    mov  rax, 1234
    call exit

       
print_startup_message:
    ; Print a startup message with integer parameters using the prinf from msvcrt.dll
    ; This is a direct call, must be linked with msvcrt.dll
    ; Note that stack prologue/epilogue is needed, or printf will crash.
    push rbp                     ; Prologue: Save frame pointer
    mov  rbp, rsp                ; Prologue: Setup new frame pointer.
    and  rsp, -16                ; Prologue: Align stack by clearing the 4 lsb
    sub  rsp, 32                 ; Prologue: Reserve shadow space
    mov  rcx, startup_msg        ; First argument: format string
    mov  rdx, 0                  ; Second argument: number
    mov  r8,  0                  ; Third argument: number
    mov  r9,  1                  ; Fourth argument: number
    call printf                  ; Call printf
    leave                        ; Epilogue: Restore rbp
    ret
