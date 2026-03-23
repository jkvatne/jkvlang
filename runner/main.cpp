#include <stdio.h>
#include <stdlib.h>

extern "C" int foo(int x, int y);

/*======================================================================*
 * main()
 *======================================================================*/
int z;

int main( void ) {
    printf("============\n");
    printf("Hello World!\n");
    printf("============\n");
    z = foo(2,3);
    printf("test() returned %d\n", z);
    exit( 0 );
}

