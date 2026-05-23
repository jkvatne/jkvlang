extern GetProcessHeap
extern HeapAlloc
extern HeapFree

section .data
dummy1 dq 0
allocation_count   dq 0
dummy2 dq 0

section .rodata
allocstr db `Allocated %d bytes at 0x%X\n`, 00h
freestr  db `Freed     %d bytes at 0x%X\n`, 00h

section .text

; alloc returns a pointerto the allocated memory in rax.
; One argument is needed, in rax, and that is the requested size in bytes.
; Returns the pointer in rax
global _alloc
_alloc:
    push rbp                         ; Prologue: Save frame pointer
    mov rbp, rsp                     ; Prologue: Setup new frame pointer.
    and rsp, -16                     ; Align stack by clearing the 4 lsb
    sub rsp, 32                      ; Reserve shadow space
    add [allocation_count], rax      ; Increment total allocated count
    mov rdi, rax                     ; Save size into rdi
    call GetProcessHeap
    mov rcx, rax                     ; Argument 1, Handle from GetProcessHeap moved into rcx
    mov rdx, 8                       ; Arbument 2, Flags into rdx, 8 means allocated memory is zeroed
    mov r8, rdi
    call HeapAlloc

    ; Print debug message with allocation size
    ; mov rcx, allocstr                ; First argument: format string
    ; mov rdx, rdi                     ; Second argument: size
    ; mov r8, rax                      ; Third argument: address
    ; mov rdi, rax                     ; Save rax
    ; call printf
    ; mov rax, rdi                     ; Restore rax

    leave                            ; Epilogue: Restore old frame pointer
    ret                              ; Epilogue: Return

; _free_struct will free memory with the given size..L1
; size is actualy only used to decrement allocation_count.
; rax is pointer to heap
; rcx is size
global _free_struct
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

; _free_str will free the string pointed to by rax, assuming it is a string with len/cap.
; It assumes it is from the default Process Heap returned from GetProcessHeap
; No return value.
global _free_str
_free_str:
    push rbp
    mov rbp, rsp
    and rsp, -16                     ; Align stack by clearing the 4 lsb
    sub rsp, 40                      ; Reserve shadow space
    mov rdi, rax                     ; Save objecgt pointer in rdi

    mov rax, [rax]                   ; Load len/cap qword
    shr rax, 32                      ; Extract capacity in the high 32bits
    jz .L1                           ; Do not free if cap is zero
    sub [allocation_count], rax,     ; Decrement allocated count

    mov rax, rdi
    call GetProcessHeap
    mov rcx, rax                     ; Argument 1, Handle from GetProcessHeap moved into rcx
    mov rdx, 0                       ; Argument 2, flags into rdx, 0 must be used
    mov r8, rdi                      ; Argument 3, move memory pointer into r8
    call HeapFree                    ; Do the actual freeing of the memory
    or rax, rax                      ; Check that Free returned 1
    jnz .L1
    mov r15, 102

.L1:
    leave
    ret

