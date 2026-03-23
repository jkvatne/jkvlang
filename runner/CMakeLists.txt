cmake_minimum_required(VERSION 3.10)

project(runner)
set(CMAKE_CXX_STANDARD 17)
enable_language(ASM_NASM)
set(CMAKE_ASM_NASM_FLAGS "-f win64 -g -F cv8" CACHE STRING "NASM Flags" FORCE)

add_executable(runner
        main.cpp
        test.asm
)
