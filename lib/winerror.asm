; winerror.asm contains _win_error which calls GetLastError

%define FORMAT_MESSAGE_FROM_SYSTEM  4096

; Symbols from kernel32
extern GetLastError
extern FormatMessageA

;-------------
section .bss
;-------------
alignb 8
error           resq 1               ; 8 byte string length/capacity
error_str       resq 32              ; 256 byte string

;-------------
section .text
;-------------

; _win_error will call GetLastError and convert it to a string
; The string is in the global variable error.
; Also, a pointer to the error string is put into r15
global _win_error
_win_error:
    ; Get last windows error into rax
    call GetLastError
    push 0                                        ; Arg 7: arguments. Not used
    push 256                                      ; Arg 6: Length of error string buffer
    mov rsi, error_str
    push rsi                                      ; Arg 5: error string buffer
    push 0                                        ; Arg 4: langID(LANG_ENGLISH, SUBLANG_ENGLISH_US),
    push rax                                      ; Arg 3: Error no
    push 0                                        ; Arg 2: modntdll.Handle(),
    mov rax, FORMAT_MESSAGE_FROM_SYSTEM | 0xFF    ; 0xFF means no crlf
    mov rdi, FormatMessageA
    mov rbx, 6*8
    call _syscall                                  ; Call the FormatMessageA function in kernel32
    add rsp, 6*8
    mov [error], rax                              ; Length of message is a 16 bit word
    ; Set pointer to error message in r15
    mov r15, error
    ret

