[BITS 64]
mov r8, -1            ; annonymous fd
mov rax, 9            ; mmap number
mov rdi, 0            ; operating system will choose mapping destination
mov rsi, 12582912     ; page size
mov rdx, 0x7          ; PROT_READ|PROT_WRITE|PROT_EXEC
mov r10, 0x8022       ; MAP_PRIVATE|MAP_ANON|MAP_POPULATE             
mov r9, 0             ; offset inside test.txt
syscall               ; now rax will point to mapped location 