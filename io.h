#ifndef IO_GUARD
#define IO_GUARD

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>

void readFile (const char *path, uint8_t **buffer, size_t *size);

#endif  /* IO_GUARD */
