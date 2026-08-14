; exit.asm contains the exit(code) function

; Symbols from kernel32

; exit have one parameter - the error code, found in rax

section .rodata

global alloc_size_str
alignb 8
alloc_size_str  dq 19
                db `--------------------------------------\nLeaked memory: %d   Error code: %d\n`, 00h

