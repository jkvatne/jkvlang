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



iter@range_1:
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
   mov rcx, 0                           ; ax 2->2 Now free the struct iter@RangeIterator itself
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
   add rax, 4                           ; -- 1->1 LoadField: Add field offset for field 'end'
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

iter@range_2:
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
   mov rcx, 0                           ; ax 2->2 Now free the struct iter@RangeIterator itself
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
   add rax, 4                           ; -- 1->1 LoadField: Add field offset for field 'end'
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

iter@next_1:
   push rbp                             ; sp 0->1 
   mov rbp, rsp                         ; sp 1->1 

   ; Line 23: if r.start>r.end ? fail(1)
   mov rax, [rbp+16]                    ; -- 0->0 EmitLoadField: Load local variable r
   ; EmitLoadTosIndirect
   movsx rax, dword  [rax]              ; ax 0->0 Load value in field 'start'
   ; 
   push rax                             ; ax 0->1 Flush rax before EmitLoadField of end
   mov rax, [rbp+16]                    ; -- 1->1 EmitLoadField: Load local variable r
   add rax, 4                           ; -- 1->1 LoadField: Add field offset for field 'end'
   ; EmitLoadTosIndirect
   movsx rax, dword  [rax]              ; ax 1->1 Load value in field 'end'
   ; 
   pop rbx                              ; ax 1->0 Pop next on stack into RBX
   cmp rbx, rax                         ; ax 0->0 Compare two ints
   mov rax, 1                           ; ax 0->0 Default to true
   jg .L7                               ; ax 0->0 
   mov rax, 0                           ; ax 0->0 Return false if we did not jump
.L7:                                    ; 
   or al, al                            ; ax 0->0 Skip block 1 if false
   jz .L8                               ; ax 0->0 
   mov r15, 1                           ; -- 0->0 Set tos to r15 = error value
   jmp .L6                              ; -- 0->0 Failed with const
.L8:                                    ; 

   ; Line 24: x = r.start
   xor rax, rax                         ; -- 0->0 EmitAllocLocalVar Assign stack space for x
   push rax                             ; -- 0->1 
   mov rax, [rbp+16]                    ; -- 1->1 EmitLoadField: Load local variable r
   ; EmitLoadTosIndirect
   movsx rax, dword  [rax]              ; ax 1->1 Load value in field 'start'
   ; 
   mov [rbp-8], eax                     ; ax 1->1 EmitAssignVariableExpressionInt Assign int to x

   ; Line 25: r start = x+1
   mov rax, [rbp+16]                    ; -- 1->1 EmitLoadField: Load local variable r
   push rax                             ; ax 1->2 Assure pointer is on stack
   movsx rax, dword [rbp-8]             ; -- 2->2 EmitLoad: Load variable x
   add rax, 1                           ; ax 2->2 add TosOpConst
   pop rdi                              ; ax 2->1 
   mov dword [rdi], eax                 ; ax 1->1 EmitAssignIndirectExpressionInt

   ; Line 26: return x
   movsx rax, dword [rbp-8]             ; -- 1->1 EmitLoad: Load variable x
   mov [rbp+24], rax                    ; ax 1->1 Save returned value nr 1
   jmp .L6                              ; sp 1->1 Return
.L6:                                    ; Return label for next
   push rax                             ; -- 1->2 Save rax before freeing local variables from next
   pop rax                              ; sp 2->1 Restore rax after freeing local variables
   add rsp, 8                           ; sp 1->0 Delete block vars
   leave                                ; sp 0->0 
   ret                                  ; sp 0->0 return from next
   ; --------------------------------------------
   ; External symbols

section .rodata

global iter@range_1
global iter@range_2
global iter@next_1
