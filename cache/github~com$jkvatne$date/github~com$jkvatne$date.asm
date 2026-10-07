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

section .text

extern tick_frequency
extern _get_ticks
extern time_used_str
extern alloc_size_str


   ; Global function time@now
extern time@now_1
   global main                          ; -- 0->0 

github~com$jkvatne$date@date_1:
   push rbp                             ; -- 0->1 
   mov rbp, rsp                         ; -- 1->1 

   ; Line 9: return t/86400
   mov rax, qword [rbp+16]              ; -- 0->0 EmitLoad: Load variable t
   cqo                                  ; ax 0->0 Sign-extend dividend in RAX into RDX:RAX
   mov rbx, 86400                       ; ax 0->0 Get constant divisor into RBX
   idiv rbx                             ; ax 0->0 RAX = RDX:RAX/RBX; RDX=Reminder
   mov [rbp+24], rax                    ; ax 0->0 Save returned value nr 1
   jmp .L11                             ; sp 0->0 Return
.L11:                                   ; Return label for date
   push rax                             ; -- 0->1 Save rax before freeing local variables from date
   pop rax                              ; sp 1->0 Restore rax after freeing local variables
   leave                                ; sp 0->0 
   ret                                  ; sp 0->0 return from date
   ; --------------------------------------------
   ; External symbols

section .rodata

global github~com$jkvatne$date@time@now_1
global github~com$jkvatne$date@date_1
