extern _sysinit
extern _syscall
extern _assert
extern allocation_count
extern _printf
extern _print
extern _fflush
extern _exit
extern _invert_err
extern _alloc
extern _free_struct
extern _free_str
extern _free_slice
extern _len
extern _lptr
extern _cptr
extern _bitlen
extern alloc_size_str
extern ExitProcess
extern processHeap
extern f32sign_mask
extern f64sign_mask
extern argv
extern argc
extern args
extern _cstrlen
extern envc
extern envs

section .text

extern tick_frequency
extern _get_ticks
extern time_used_str
extern alloc_size_str
extern _low_byte
extern _floor
extern _int64
extern _uint64
extern _int32
extern _mul64



time@now_1:
   push rbp                             ; sp 0->1 Function prologue
   mov rbp, rsp                         ; sp 0->0 Function prologue

   ; Line 9: return 1234
   mov rax, 1234                        ; ax 0->0 PushConst Returned const value number 0
   mov [rbp+16], rax                    ; ax 0->0 Save returned value nr 1
   jmp .L8                              ; sp 0->0 Return
.L8:                                    ; Return label for now
   push rax                             ; -- 0->1 Save rax before freeing local variables from now
   pop rax                              ; sp 1->0 Restore rax after freeing local variables
   leave                                ; sp 0->0 
   ret                                  ; sp 0->0 return from now

section .rodata

global time@now_1
