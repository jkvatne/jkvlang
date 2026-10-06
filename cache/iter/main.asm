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



range_1:
   push rbp                             ; sp 0->1 
   mov rbp, rsp                         ; sp 1->1 

   ; Line 9: r = new(RangeIterator)
   xor rax, rax                         ; -- 0->0 EmitAllocLocalVar Assign stack space for r
   push rax                             ; -- 0->1 
   mov rax, 0                           ; ax 1->1 
   call _alloc                          ; ax 1->1 Allocate new struct
   mov rdx, rax                         ; ax 1->1 Zero new struct
   mov rdi, rax                         ; ax 1->1 
   push rax                             ; ax 1->2 Save new struct
   mov rcx, 0                           ; ax 2->2 
   cld                                  ; ax 2->2 
   xor rax, rax                         ; ax 2->2 
   rep stosb                            ; ax 2->2 
   mov rax, rdx                         ; ax 2->2 
   mov rax, [rbp-8]                     ; ax 2->2 EmitAssignVariableExpressionStruct, Get old value
   or rax, rax                          ; ax 2->2 
   jz .L3                               ; ax 2->2 
   mov rcx, 0                           ; ax 2->2 Now free the struct RangeIterator itself
   call _free_struct                    ; ax 2->2 
.L3:                                    ; 
   pop rax                              ; -- 2->1 Get new struct
   mov [rbp-8], rax                     ; -- 1->1 EmitStoreToLocal Assign struct to r

   ; Line 10: r start = start
   mov rax, [rbp-8]                     ; -- 1->1 EmitLoadField: Load local variable r
   push rax                             ; ax 1->2 Assure pointer is on stack
   movsx rax, dword [rbp+16]            ; -- 2->2 EmitLoad: Load variable start
   pop rdi                              ; ax 2->1 
   mov dword [rdi], eax                 ; ax 1->1 EmitAssignIndirectExpressionInt

   ; Line 11: r end = end
   mov rax, [rbp-8]                     ; -- 1->1 EmitLoadField: Load local variable r
   push rax                             ; ax 1->2 Assure pointer is on stack
   movsx rax, dword [rbp+24]            ; -- 2->2 EmitLoad: Load variable end
   pop rdi                              ; ax 2->1 
   mov dword [rdi], eax                 ; ax 1->1 EmitAssignIndirectExpressionInt

   ; Line 12: return r
   mov rax, qword [rbp-8]               ; -- 1->1 EmitLoad: Load struct/string variable r
   mov [rbp+32], rax                    ; ax 1->1 Save returned value nr 1
   jmp .L2                              ; sp 1->1 Return
.L2:                                    ; Return label for range
   push rax                             ; -- 1->2 Save rax before freeing local variables from range
   pop rax                              ; sp 2->1 Restore rax after freeing local variables
   add rsp, 8                           ; sp 1->0 Delete block vars
   leave                                ; sp 0->0 
   ret                                  ; sp 0->0 return from range
   ; --------------------------------------------
   ; External symbols
extern Sleep

range_2:
   push rbp                             ; sp 0->1 
   mov rbp, rsp                         ; sp 1->1 

   ; Line 16: r = new(RangeIterator)
   xor rax, rax                         ; -- 0->0 EmitAllocLocalVar Assign stack space for r
   push rax                             ; -- 0->1 
   mov rax, 0                           ; ax 1->1 
   call _alloc                          ; ax 1->1 Allocate new struct
   mov rdx, rax                         ; ax 1->1 Zero new struct
   mov rdi, rax                         ; ax 1->1 
   push rax                             ; ax 1->2 Save new struct
   mov rcx, 0                           ; ax 2->2 
   cld                                  ; ax 2->2 
   xor rax, rax                         ; ax 2->2 
   rep stosb                            ; ax 2->2 
   mov rax, rdx                         ; ax 2->2 
   mov rax, [rbp-8]                     ; ax 2->2 EmitAssignVariableExpressionStruct, Get old value
   or rax, rax                          ; ax 2->2 
   jz .L5                               ; ax 2->2 
   mov rcx, 0                           ; ax 2->2 Now free the struct RangeIterator itself
   call _free_struct                    ; ax 2->2 
.L5:                                    ; 
   pop rax                              ; -- 2->1 Get new struct
   mov [rbp-8], rax                     ; -- 1->1 EmitStoreToLocal Assign struct to r

   ; Line 17: r start = 0
   mov rax, [rbp-8]                     ; -- 1->1 EmitLoadField: Load local variable r
   push rax                             ; ax 1->2 Assure pointer is on stack
   pop rdi                              ; -- 2->1 pop EmitAssignIndirectConstInt
   mov dword  [rdi], 0                  ; -- 1->1 

   ; Line 18: r end = count-1
   mov rax, [rbp-8]                     ; -- 1->1 EmitLoadField: Load local variable r
   push rax                             ; ax 1->2 Assure pointer is on stack
   movsx rax, dword [rbp+16]            ; -- 2->2 EmitLoad: Load variable count
   sub rax, 1                           ; ax 2->2 TosOpConst
   pop rdi                              ; ax 2->1 
   mov dword [rdi], eax                 ; ax 1->1 EmitAssignIndirectExpressionInt

   ; Line 19: return r
   mov rax, qword [rbp-8]               ; -- 1->1 EmitLoad: Load struct/string variable r
   mov [rbp+24], rax                    ; ax 1->1 Save returned value nr 1
   jmp .L4                              ; sp 1->1 Return
.L4:                                    ; Return label for range
   push rax                             ; -- 1->2 Save rax before freeing local variables from range
   pop rax                              ; sp 2->1 Restore rax after freeing local variables
   add rsp, 8                           ; sp 1->0 Delete block vars
   leave                                ; sp 0->0 
   ret                                  ; sp 0->0 return from range
   ; --------------------------------------------
   ; External symbols
extern Sleep

next_1:
   push rbp                             ; sp 0->1 
   mov rbp, rsp                         ; sp 1->1 

   ; Line 23: if r.start>r.end ? fail(1)
