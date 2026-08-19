; sys.asm  Contains file IO functions

%define STD_INPUT_HANDLE  -10
%define STD_OUTPUT_HANDLE -11
%define STD_ERROR_HANDLE  -12
%define FORMAT_MESSAGE_FROM_SYSTEM  4096

extern GetStdHandle
extern ExitProcess
extern GetProcessHeap
extern CreateFileA
extern CreateFileW
extern ReadFile
extern WriteFile
extern CloseHandle
extern HeapAlloc
extern HeapFree
extern printf
extern fflush
extern GetLastError
extern FormatMessageA
extern ExitProcess
extern GetKeyState

; Global variables
global argc, argv, args, env, envs, envc
global allocation_count
global f32sign_mask
global f64sign_mask
global processHeap
global alloc_size_str

; Global functions
global _exit
global _sysinit
global _cstrlen
global _create_file
global _write_file
global _read_file
global _close_file
global _lptr
global _cptr
global _len
global _bitlen
global _create_file
global _alloc
global _free_struct
global _free_slice
global _free_str
global _print
global _invert_err
global _syscall

;-------------
section .rodata
;-------------
alignb 8
sp_mess            db "...rsp=0x%X", 0Ah, 00h
crlf               db 0Ch, 0Ah, 00h
crlf_str               db 0Ah, 00h
default_assert_mess    db "Assert failed", 00h
alignb 8
alloc_size_str  dq 19
                db `--------------------------------------\nLeaked memory: %d   Error code: %d\n`, 00h

;-------------
section .bss
;-------------
alignb 8
stdOutputHandle resq 1
stdErrorHandle  resq 1
stdInputHandle  resq 1
processHeap     resq 1
error           resq 1               ; 8 byte string length/capacity
error_str       resq 32              ; 256 byte string

;-------------
section .data
;-------------
locale_str  db ".utf8", 0   ; "UTF" locale, or use "" for system default
f64sign_mask: dq 0x8000000000000000
f32sign_mask: dq 0x80000000
argc: dq 0
argv: dq 0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,
args: dq 0   ; Slice for arguments
envs: dq 0   ; Slice for environment
env: dq 0
envc: dq 0
allocation_count   dq 0

;-------------
section .text
;-------------


%define CREATE_NEW        1    ; Fail if file exists
%define CREATE_ALWAYS     2    ; Truncate old file if it exists
%define OPEN_EXISTING     3    ; Fails if file exists
%define OPEN_ALLWAYS      4
%define TRUNCATE_EXISTING 5    ; Fails if file exists

;-------------
section .text
;-------------

_print:
    mov rdi, printf
    call _syscall
    mov rax, crlf
    mov [rsp+8], rax
    mov rbx, 16
    call _syscall
    call _flush
    ret

; _printf is the local version of printf from msvcrt.dll
; All parameters are pushed on the stack. [rsp] is the format string
; All strings must be C-strings.
; Stack size should be in rbx, 8 bytes for each parameter in the format string
; Note that the format string has 8 bytes initial length/capacity
global _printf
_printf:
    mov rdi, printf
    call _syscall
    ret

global _fflush
global _flush
_fflush:
_flush:
    push rbp              ; Save old frame pointer
    mov rbp, rsp          ; Setup new frame pointer
    and rsp, -16          ; Align stack by clearing the 4 lsb
    sub rsp, 96           ; Reserve space for arguments to the called function
    xor rcx, rcx
    call fflush
    leave
    ret

global _printsp
_printsp:
    push rsp                    ; Value to be printed
    mov rax, sp_mess            ; Message at top of stack
    push rax
    mov rbx, 16                  ; Stack size is 8 bytes
    call _printf                ; system function to call
    add sp, 16
    ret


; alloc returns a pointerto the allocated memory in rax.
; One argument is needed, in rax, and that is the requested size in bytes.
; Returns the pointer in rax
_alloc:
    push rbp                         ; Prologue: Save frame pointer
    mov rbp, rsp                     ; Prologue: Setup new frame pointer.
    and rsp, -16                     ; Align stack by clearing the 4 lsb
    sub rsp, 32                      ; Reserve shadow space
    add [allocation_count], rax      ; Increment total allocated count
    mov rdi, rax                     ; Save size into rdi
    mov rcx, [processHeap]           ; Argument 1, Handle from GetProcessHeap moved into rcx
    mov rdx, 8                       ; Arbument 2, Flags into rdx, 8 means allocated memory is zeroed
    mov r8, rdi
    call HeapAlloc
    leave                            ; Epilogue: Restore old frame pointer
    ret                              ; Epilogue: Return

; _free_struct will free memory with the given size..L1
; size is actualy only used to decrement allocation_count.
; rax is pointer to heap
; rcx is size
_free_struct:
    push rbp
    mov rbp, rsp
    and rsp, -16                     ; Align stack by clearing the 4 lsb
    sub rsp, 40                      ; Reserve shadow space
    mov rdi, rax                     ; Save objecgt pointer in rdi

    sub [allocation_count], rcx,     ; Decrement allocated count

    call GetProcessHeap
    mov rcx, rax                     ; Argument 1, Handle from GetProcessHeap moved into rcx
    mov rdx, 0                       ; Argument 2, flags into rdx, 0 must be used
    mov r8, rdi                      ; Argument 3, move memory pointer into r8
    call HeapFree                    ; Do the actual freeing of the memory
    or rax, rax                      ; Check that Free returned 1
    jnz .L2
    mov r15, 103
.L2:
    leave
    ret

; _free_slice will free memory used by the slice
; rax is pointer to the slice on the heap (points to len/cap)
; rcx is element size in bytes
_free_slice:
    push rbp
    mov rbp, rsp
    and rsp, -16                     ; Align stack by clearing the 4 lsb
    sub rsp, 40                      ; Reserve shadow space
    mov rdi, rax                     ; Save objecgt pointer in rdi

    mov rax, [rdi]                   ; Load len/cap
    shr rax, 32                      ; Extract cap
    imul rax, rcx                    ; Multiply capacity by elementsize
    add rax, 8                       ; Add space for len/cap
    mov rcx, rax
    sub [allocation_count], rcx,     ; Decrement allocated count

    call GetProcessHeap
    mov rcx, rax                     ; Argument 1, Handle from GetProcessHeap moved into rcx
    mov rdx, 0                       ; Argument 2, flags into rdx, 0 must be used
    mov r8, rdi                      ; Argument 3, move memory pointer into r8
    call HeapFree                    ; Do the actual freeing of the memory
    or rax, rax                      ; Check that Free returned 1
    jnz .L2
    mov r15, 103
.L2:
    leave
    ret

; _free_str will free the string pointed to by rax, assuming it is a string with len/cap.
; It assumes it is from the default Process Heap returned from GetProcessHeap
; No return value.
_free_str:
    push rbp
    mov rbp, rsp
    and rsp, -16                     ; Align stack by clearing the 4 lsb
    sub rsp, 40                      ; Reserve shadow space
    mov r12,  rax                    ; Save object pointer

    mov rcx, [rax]                   ; Load len/cap qword
    shr rcx, 32                      ; Extract capacity in the high 32bits
    add rcx, 8
    jz .L1                           ; Do not free if cap is zero
    sub [allocation_count], rcx      ; Decrement allocated count

    ; Clear area to avoid double use
    mov rdi, rax                     ; Destination pointer (buffer address)
    xor eax, eax                     ; Value to store (0)
    cld                              ; Clear direction flag (process forward)
    rep stosb                        ; Repeat storing AL into [RDI] (use stosd for dwords)
    mov rax, r12

    call GetProcessHeap
    mov rcx, rax                     ; Argument 1, Handle from GetProcessHeap moved into rxx
    mov rdx, 0                       ; Argument 2, flags into rdx, 0 must be used
    mov r8, r12                      ; Argument 3, move memory pointer into r8
    call HeapFree                    ; Do the actual freeing of the memory
    or rax, rax                      ; Check that Free returned 1
    jnz .L1
    mov r15, 102

.L1:
    leave
    ret


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

;  BOOL ReadFile(
;  [in]                HANDLE       hFile,
;  [out]               LPVOID       lpBuffer,
;  [in]                DWORD        nNumberOfBytesToRead,
;  [out, optional]     LPDWORD      lpNumberOfBytesRead,
;  [in, out, optional] LPOVERLAPPED lpOverlapped
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
    mov [rsp+16], rax
    ret

_bitlen:
    mov rcx, [rsp+8]
    bsr rax, rcx
    jnz  .L1
    mov rax, -1
.L1:
    inc rax
    mov [rsp+16], rax
    ret

_sysinit:
    ; sysinit will initialize the console handles
    push rbp                         ; Prologue: Save frame pointer
    mov rbp, rsp                     ; Prologue: Setup new frame pointer.
    and rsp, -16                     ; Prologue: Align stack by clearing the 4 lsb
    sub rsp, 48                      ; Prologue: Reserve shadow space

    ; Copy argc/argv pointers to global variables
    mov [argc],rcx
    mov [argv],rdx
    mov [env], r8
    ; Get this threads local allocation heap
    call GetProcessHeap
    mov [processHeap], rax

    ; Get command line arguments
    call _create_args
    call _create_envs

    ; Load the handle for standard output
    mov   ecx, STD_OUTPUT_HANDLE
    call  GetStdHandle
    mov   [rel stdOutputHandle], rax

    mov   ecx, STD_ERROR_HANDLE
    call  GetStdHandle
    mov   [rel stdErrorHandle], rax

    mov   ecx, STD_INPUT_HANDLE
    call  GetStdHandle
    mov   [rel stdInputHandle], rax

    ; Initialize the error code
    mov  r15, 0
    mov qword [allocation_count], 0
    leave
    ret

; assert will verify that the first arbument [rsp] is true (not 0)
; with optional additional parameters in [rsp+8], [rsp+16]...
; Stack; <bool><messageptr><arg1><arg2>..
; rbx should contain the size of the stack. (number of arguments-1) * 8.
; NB: Assert will append LF after the message.
global _assert
_assert:
    mov rax, [rsp+8]
    or rax, rax           ; Set z-flag if rax is zero
    jz .L1                ; Jump if the bool argument was false
    ret                   ; Returns if assert(true)
.L1:

    push rbp
    mov rbp, rsp          ; Setup new frame pointer
    and rsp, -16          ; Align stack by clearing the 4 lsb
    sub rsp, 96           ; Reserve space for arguments to the called function

    mov r15, 99           ; Set error code - assert failed
    sub rbx, 8
    or bx, bx             ; Check if bx=0 (no string given)
    jnz .L5
    mov bx, 8
    mov rcx, default_assert_mess
    jmp .L4
.L5:

    mov rcx, [rbp+24]    ; rcx = First argument: format string
    add rcx, 8           ; Skip length/capacity of string
    sub rbx, 8
    or rbx, rbx
    jz .L2
.L4:
    mov rdx, [rbp+32]    ; dx = Second argument
    sub rbx, 8
    jc .L2

    mov r8,  [rbp+40]    ; r8 = Third argument
    sub rbx, 8
    jc .L2

    mov r9,  [rbp+48]    ; r9 = Forth argument
    sub rbx, 8
    jc .L2

    mov rsi, [rbp+56]    ; Fifth argument onto stack
    mov [rsp+32], rsi
    sub rbx, 8
    jc .L2

    mov rsi, [rbp+64]
    mov [rsp+40], rsi     ; Sixth argument onto stack
    sub rbx, 8
    jc .L2

    mov rsi, [rbp+72]
    mov [rsp+48], rsi     ; Seventh argument onto stack
    sub rbx, 8
    jc .L2

    mov rsi, [rbp+80]
    mov [rsp+56], rsi     ; Eight argument onto stack
    sub rbx, 8
    jc .L2

    mov rsi, [rbp+88]
    mov [rsp+64], rsi     ; Ninth argument onto stack
    sub rbx, 8
    jc .L2

.L2:
    call printf

    mov rcx, crlf_str
    call printf

    leave
    ret

; invert_err will set err to zero if there was an error
; and sett error to 100 if there was no errors.
; This is used during testing to test expected assert errors.
_invert_err:
    or r15, r15
    jnz .L1
    mov r15, 100
    ret
.L1:
    mov r15, 0
    ret

_GetKeyState:
    mov rdi, GetKeyState
    mov rbx, 8
    call _syscall
    mov rdi, rsp
    add rdi, 16
    mov [rdi], rax
    ret

; syscall will call any dll function that is reachable
; The address of the function should be in rdi, arg count *8 in rbx
; All parameters should be on the stack
_syscall:
    push rbp              ; Save old frame pointer
    mov rbp, rsp          ; Setup new frame pointer
    and rsp, -16          ; Align stack by clearing the 4 lsb
    sub rsp, 96           ; Reserve space for arguments to the called function
    mov r14, rbx
    mov rcx, [rbp+24]     ; rcx = First argument
    sub rbx, 9
    jc .L3

    mov rdx, [rbp+32]    ; dx = Second argument
    sub rbx, 8
    jc .L3

    mov r8,  [rbp+40]    ; r8 = Third argument
    sub rbx, 8
    jc .L3

    mov r9,  [rbp+48]    ; r9 = Forth argument
    sub rbx, 8
    jc .L3

    mov rsi, [rbp+56]    ; Fifth argument onto stack
    mov [rsp+32], rsi
    sub rbx, 8
    jc .L3

    mov rsi, [rbp+64]
    mov [rsp+40], rsi     ; Sixth argument onto stack
    sub rbx, 8
    jc .L3

    mov rsi, [rbp+72]
    mov [rsp+48], rsi     ; Seventh argument onto stack
    sub rbx, 8
    jc .L3

    mov rsi, [rbp+80]
    mov [rsp+56], rsi     ; Eight argument onto stack
    sub rbx, 8
    jc .L3

    mov rsi, [rbp+88]
    mov [rsp+64], rsi     ; Nineth argument onto stack
    sub rbx, 8
    jc .L3

    mov rsi, [rbp+96]
    mov [rsp+72], rsi     ; Tenth argument onto stack

.L3:
    mov rbx, r14
    call rdi
    leave
    ret

_exit:
    push rbp                         ; Prologue: Save frame pointer
    mov rbp, rsp                     ; Prologue: Setup new frame pointer.
    and rsp, -16                     ; Align stack by clearing the 4 lsb
    sub rsp, 32                      ; Reserve shadow space
    mov rcx, rax
    call ExitProcess
    leave
    ret

; Calculate length of C-string pointed to by rax and return length in rax
; Will destroy rcx an rdi
_cstrlen:
    mov rdi, rax
    xor rax, rax
    mov rcx, -1
    cld
    repne scasb
    not rcx
    dec rcx
    mov rax, rcx
    ret

; strlen will calculate the length of a 0-terminated C-string at [rax]
; This function uses jkv calling conventions with parameters on the stack
cstrlen:
    push rbp                         ; Prologue: Save frame pointer
    mov rbp, rsp                     ; Prologue: Setup new frame pointer.
    mov rax, [rbp+16]
    mov rdi, [rax]
    xor   rax,rax      ; compare to zero
    mov   rcx,-1       ;limit scan length
    cld
    repne scasb
    not rcx
    dec   rcx          ;minus one for rep going too far by one
    mov [rbp+24], rcx
    leave
    ret

; Convert a C-string pointed to by rax into a standard const string with cap=0
; Return pointer to new string in rax. Uses r13, r14, rsi, rdi, rcx
_cstr_to_string:
    mov r13, rax
    call _cstrlen    ; Will destroy rcx an rdi
    mov r14, rax     ; Save length in r14
    add rax, 8
    call _alloc      ; Returns pointer in rax
    push rax
    mov rdi, rax
    mov [rdi], r14   ; Copy length into first 8 bytes of new string
    add rdi, 8
    mov rcx, r14          ; Initialzie rcx with string count
    mov rsi, r13
    cld
    rep movsb             ; Copy bytes into new string in rdi
    pop rax
    ret

; Create a slice of strings with command line arguments.
; r12 counts arguments to be copied
; r11 points to output slice elements
; r10 points to input c-string array in argv[]
_create_args:
    ; Read in command line arguments to argc and argv variables.
    mov r12, [argc]  ; Number of arguments will be length of slize
    mov rax, r12     ; Len into rax
    imul rax, 8      ; Calculate slize size
    add rax, 8       ; Calculate slize size
    call _alloc      ; Allocate slize
    mov [args], rax  ; Save slice in global variable args
    mov r11, rax     ; r11 points to the new slice's content.
    mov [r11], r12   ; Set slice length
    add r11, 8       ; Point to first string in slice
    mov r10, [argv]  ; r10 points to the first argv string pointer
    ; Now we can add the strings in a loop over the argv list
.loop:
    mov rax, [r10]         ; Point to the argv string itself
    push r10
    push r11
    call _cstr_to_string   ; Convert to string. New string in rax. Uses  r13, r14, rsi, rdi. Rax will point to new string
    pop r11
    pop r10
    mov [r11], rax         ; Save new string in slice
    add r10, 8             ; Next string pointer in slice
    add r11, 8             ; next argv
    dec r12                ; Count args
    jnz .loop
    ret

; Create a slice of strings with environment string
; r12 counts arguments to be copied
; r11 points to output slice elements
; r10 points to input c-string array in argv[]
_create_envs:
    mov r10, [env]   ; r10 points to the first env string pointer
    xor r12, r12
.countenv:
    mov rax, [r10]
    or rax, rax
    jz .endenvcount
    inc r12
    add r10, 8
    jmp .countenv
.endenvcount:
    mov [envc], r12

    mov r10, [env]   ; r10 points to the first env string pointer
    ; mov r12, 5       ; Number of arguments will be length of slize
    mov rax, r12     ; Len into rax
    imul rax, 8      ; Calculate slize size
    add rax, 8       ; Calculate slize size
    call _alloc      ; Allocate slize
    mov [envs], rax  ; Save slice in global variable args
    mov r11, rax     ; r11 points to the new slice's content.
    mov [r11], r12   ; Set slice length
    add r11, 8       ; Point to first string in slice
    mov r10, [env]   ; r10 points to the first argv string pointer
    ; Now we can add the strings in a loop over the argv list
.loop:
    mov rax, [r10]         ; Point to the argv string itself
    push r10
    push r11
    call _cstr_to_string   ; Convert to string. New string in rax. Uses  r13, r14, rsi, rdi. Rax will point to new string
    pop r11
    pop r10
    mov [r11], rax         ; Save new string in slice
    add r10, 8             ; Next string pointer in slice
    add r11, 8             ; next argv
    dec r12                ; Count args
    jnz .loop
    ret

