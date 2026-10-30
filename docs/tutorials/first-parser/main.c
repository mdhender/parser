#include "calc.c"
#include "calc.h"

int main(void) {
    void *parser = ParseAlloc(malloc);
    if (parser == NULL) return 1;
    Parse(parser, INTEGER, 3);
    Parse(parser, PLUS, 0);
    Parse(parser, INTEGER, 4);
    Parse(parser, 0, 0);
    ParseFree(parser, free);
    return 0;
}
