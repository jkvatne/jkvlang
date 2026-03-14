section .text

global syscall
global malloc
global mfree

extern GetProcessHeap
extern HeapAlloc
extern HeapFree

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
; The address of the function should be in r10, arg count in rax
syscall:
    push rbp
    mov rbp, rsp          ; Setup new frame pointer

    and rsp, -16          ; Align stack by clearing the 4 lsb
    sub rsp, 80           ; Reserve space for arguments to the called function

    dec rax
    shl rax, 3

    mov rcx, [rax+rbp+16]    ; cx = First argument: format string
    sub rax, 8
    jc docall

    mov rdx, [rax+rbp+16]    ; dx = Second argument
    sub rax, 8
    jc docall

    mov r8,  [rax+rbp+16]    ; r8 = Third argument
    sub rax, 8
    jc docall

    mov r9,  [rax+rbp+16]    ; r9 = Forth argument
    sub rax, 8
    jc docall

    mov rbx, [rax+rbp+16]    ; Fifth argument onto stack
    mov [rsp+32], rbx
    sub rax, 8
    jc docall

    mov rbx, [rax+rbp+16]
    mov [rsp+40], rbx     ; Sixth argument onto stack
    sub rax, 8
    jc docall

    mov rbx, [rax+rbp+16]
    mov [rsp+48], rbx     ; Seventh argument onto stack
    sub rax, 8
    jc docall

    mov rbx, [rax+rbp+16]
    mov [rsp+56], rbx     ; Eight argument onto stack
    sub rax, 8

docall:
    call [rdi]
    leave
    ret
