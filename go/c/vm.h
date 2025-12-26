#ifndef _LANGLANG_VM
#define _LANGLANG_VM

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>

typedef struct ll_parsing_error {
  char *message;
  char *label;
  int start;
  int end;
} ll_parsing_error;

void ll_parsing_error_free(ll_parsing_error *e);

#endif
