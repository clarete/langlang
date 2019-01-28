/* -*- coding: utf-8; -*-
 *
 * bench.c - Benchmark stuff
 *
 * Copyright (C) 2019  Lincoln Clarete
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */
#include <assert.h>
#include <stdio.h>
#include <string.h>
#include <time.h>

#include "../debug.h"
#include "../peg.h"
#include "../value.h"
#include "../io.h"

#define NUM_RUNS 13

void run (const char *grammar_file,
          const char *input_file)
{
  struct timespec start, stop;
  double time_taken;
  double total = 0;
  size_t i, grammar_size, input_size;

  Machine m;
  Bytecode *grammar = NULL;
  unsigned char *input = NULL;

  readFile (grammar_file, &grammar, &grammar_size);
  readFile (input_file, &input, &input_size);

  for (i = 0; i < NUM_RUNS; i++) {
    mInit (&m);
    mLoad (&m, grammar);

    clock_gettime (CLOCK_MONOTONIC, &start);
    assert (mMatch (&m, (const char *) input, input_size));
    /* mExtract (&m, (const char *) input); */
    clock_gettime (CLOCK_MONOTONIC, &stop);

    time_taken =
      (stop.tv_sec - start.tv_sec) +
      ((stop.tv_nsec - start.tv_nsec) / 1e9);
    total += time_taken;

    printf ("[%ld:%lfs] %s %s\n", i, time_taken, grammar_file, input_file);
    mFree (&m);
  }

  printf ("Result: %s ran against %s in %lfs (%ld)\n",
          grammar_file, input_file, total / i, i);
}

int main ()
{
  /* 1000 lines & 500 columns */
  run ("csv0.binx", "./data/1.a.csv");
  /* 500 lines & 1000 columns */
  run ("csv0.binx", "./data/1.b.csv");
  /* 1000 lines & 1000 columns */
  run ("csv0.binx", "./data/1.c.csv");
  return EXIT_SUCCESS;
}
