/* -*- coding: utf-8; -*-
 *
 * calc.c - Calculator using PEG
 *
 * Copyright (C) 2018  Lincoln Clarete
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
#include <stdio.h>
#include <math.h>
#include <readline/readline.h>
#include <readline/history.h>

#include "../../peg.h"

#define FIRST(o) CAR (CDR (o))
#define SECOND(o) CDR (CDR (o))
#define HASKEY(o,n) (strcmp (SYMBOL (CAR (o))->name, n) == 0)

Object *evNumber (Object *input)
{
  Object *first = FIRST (input);
  long int v;
  int base;
  if (HASKEY (first, "DEC")) base = 10;
  else if (HASKEY (first, "HEX")) base = 16;
  else if (HASKEY (first, "BIN")) base = 2;
  v = strtol (SYMBOL (FIRST (first))->name, NULL, base);
  return makeInt (v);
}

Object *evExpression (Object *input);

Object *evPrimary (Object *input)
{
  if (HASKEY (FIRST (input), "Number")) {
    return evNumber (FIRST (input));
  } else if (HASKEY (FIRST (input), "Expression")) {
    return evExpression (FIRST (input));
  }
  return NULL;
}

Object *evUnary (Object *input)
{
  Object *v;
  /* If the operator isn't present */
  if (!CDR (CDR (input))) return evPrimary (FIRST (input));

  v = evPrimary (CAR (SECOND (input)));

  if (HASKEY (FIRST (input), "PLUS")) {
    return makeInt (+INT (v)->value);
  } else if (HASKEY (FIRST (input), "MINUS")) {
    return makeInt (-INT (v)->value);
  }
  return NULL;
}

Object *evPower (Object *input)
{
  Object *left, *right;
  left = evUnary (FIRST (input));

  if (SECOND (input)) {
    right = evPower (FIRST (SECOND (input)));
    if (HASKEY (CAR (SECOND (input)), "POWER")) {
      return makeInt (pow (INT (left)->value, INT (right)->value));
    } else if (HASKEY (CAR (SECOND (input)), "MOD")) {
      return makeInt (INT (left)->value % INT (right)->value);
    }
    return NULL;
  } else {
    return left;
  }
}

Object *evFactor (Object *input)
{
  Object *left, *right;
  left = evPower (FIRST (input));

  if (SECOND (input)) {
    right = evFactor (FIRST (SECOND (input)));
    if (HASKEY (CAR (SECOND (input)), "STAR")) {
      return makeInt (INT (left)->value * INT (right)->value);
    } else if (HASKEY (CAR (SECOND (input)), "SLASH")) {
      return makeInt (INT (left)->value / INT (right)->value);
    }
    return Nil;
  } else {
    return left;
  }
}

Object *evTerm (Object *input)
{
  Object *left, *right;
  left = evFactor (FIRST (input));

  if (SECOND (input)) {
    right = evTerm (FIRST (SECOND (input)));
    if (HASKEY (CAR (SECOND (input)), "PLUS")) {
      return makeInt (INT (left)->value + INT (right)->value);
    } else if (HASKEY (CAR (SECOND (input)), "MINUS")) {
      return makeInt (INT (left)->value - INT (right)->value);
    }
    return Nil;
  } else {
    return left;
  }
}

Object *evExpression (Object *input)
{
  return evTerm (FIRST (input));
}

Object *calculate (Object *input)
{
  /* Unwrap "Calculator" Rule */
  return evExpression (FIRST (input));
}

/* Reads the entire content of the file under `path' into `buffer' */
void readFile (const char *path, uint8_t **buffer, size_t *size)
{
  FILE *fp = fopen (path, "rb");
  if (!fp) FATAL ("Can't open file %s", path);

  /* Read file size. */
  fseek (fp, 0, SEEK_END);
  *size = ftell (fp);
  rewind (fp);
  /* Allocate buffer and read the file into it.  The +1 is reserved
     for the NULL char. */
  if ((*buffer = calloc (*size + 1, sizeof (uint8_t))) == NULL) {
    fclose (fp);
    FATAL ("Can't read file into memory %s", path);
  }
  if ((fread (*buffer, sizeof (uint8_t), *size, fp) != *size)) {
    fclose (fp);
    FATAL ("Can't read file %s", path);
  }
  fclose (fp);
}

int main ()
{
  Machine m;
  size_t grammar_size = 0, input_size = 0;
  Bytecode *grammar = NULL;
  char *input = NULL;
  const char *output;
  bool running = true;
  Object *result;

  readFile ("calc.binx", &grammar, &grammar_size);

  while (running) {
    if ((input = readline ("calc% ")) == NULL) break;
    input_size = strlen (input);
    if (input_size > 0) add_history (input);

    mInit (&m);
    mLoad (&m, grammar, grammar_size);
    if ((output = mMatch (&m, input, input_size)) != NULL) {
      result = calculate (mExtract (&m, input));
      printObj (result); printf ("\n");
    }
    free (input);
    mFree (&m);
  }

  return 0;
}
