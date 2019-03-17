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
#include <sys/types.h>
#include <dirent.h>

#include "../debug.h"
#include "../peg.h"
#include "../value.h"
#include "../io.h"

#define NUM_RUNS 13

void runFiles (const char *grammar_file,
               const char *input_file)
{
  struct timespec start, stop;
  double time_taken;
  double total = 0;
  size_t i, grammar_size, input_size;

  Machine m;
  Bytecode *grammar = NULL;
  unsigned char *input = NULL;
  Object *output = NULL;

  readFile (grammar_file, &grammar, &grammar_size);
  readFile (input_file, &input, &input_size);

  printf ("Input: g: %s, i: %s\n", grammar_file, input_file);

  for (i = 0; i < NUM_RUNS; i++) {
    mInit (&m);
    mLoad (&m, grammar);

    clock_gettime (CLOCK_MONOTONIC, &start);
    assert (mMatch (&m, (const char *) input, input_size, &output) == 0);
    clock_gettime (CLOCK_MONOTONIC, &stop);
    if (output) objFree (output);

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

static bool endsWith (const char *str, const char *suffix)
{
  size_t lenS = strlen (str);
  size_t lenSuffix = strlen (suffix);
  if (lenS < lenSuffix) return false;
  return strncmp (str + (lenS - lenSuffix), suffix, lenSuffix) == 0;
}

void run ()
{
  DIR *dp;
  struct dirent *de;
  char fpath[PATH_MAX];

  if ((dp = opendir ("./data")) == NULL) {
    fprintf (stderr, "Directory data doesn't seem to exist\n");
    fprintf (stderr, "the `make' command should put it back there\n");
    fprintf (stderr, "so long\n");
    exit (2);
  }
  while ((de = readdir (dp)) != NULL) {
    memset (fpath, 0, PATH_MAX);
    memcpy (fpath, "./data/", 7);
    memcpy (fpath+7, de->d_name, strlen (de->d_name));
    if (endsWith (de->d_name, ".csv")) {
      runFiles ("csv0.nc.binx", (const char *) fpath);
      runFiles ("csv0.binx", (const char *) fpath);
    } else if (endsWith (de->d_name, ".json")) {
      runFiles ("json0.nc.binx", (const char *) fpath);
      runFiles ("json0.binx", (const char *) fpath);
    }
  }
  closedir (dp);
}

int main ()
{
  run ();
  return EXIT_SUCCESS;
}
