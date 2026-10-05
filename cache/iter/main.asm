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



range_1:
   push rbp                             ; sp 0->1 
   mov rbp, rsp                         ; sp 1->1 

   ; Line 9: r = new(RangeIterator, 10)
   xor rax, rax                         ; -- 0->0 EmitAllocLocalVar Assign stack space for r
   push rax                             ; -- 0->1 
