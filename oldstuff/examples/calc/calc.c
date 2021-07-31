/* -*- coding: utf-8; -*-
 *
 * calc.c - Calculator using PEG
 *
 * Copyright (C) 2018-2019  Lincoln Clarete
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
#include "../../io.h"

#define FIRST(o) CAR (CDR (o))
#define SECOND(o) CDR (CDR (o))
#define HASKEY(o,n) (strcmp (SYMBOL (CAR (o))->name, n) == 0)
#define BINOP(a,op,b) (INT (a)->value op INT (b)->value)

Value *evNumber (Value *input)
{
  Value *first = FIRST (input);
  long int v;
  int base = -1;
  if (HASKEY (first, "DEC")) base = 10;
  else if (HASKEY (first, "HEX")) base = 16;
  else if (HASKEY (first, "BIN")) base = 2;
  v = strtol (SYMBOL (FIRST (first))->name, NULL, base);
  return intNew (v);
}

Value *evTerm (Value *input);

Value *evPrimary (Value *input)
{
  if (HASKEY (FIRST (input), "Number")) {
    return evNumber (FIRST (input));
  } else if (HASKEY (FIRST (input), "Term")) {
    return evTerm (FIRST (input));
  }
  return NULL;
}

Value *evUnary (Value *input)
{
  Value *v;
  /* If the operator isn't present */
  if (NILP (CDR (CDR (input)))) return evPrimary (FIRST (input));

  v = evPrimary (CAR (SECOND (input)));

  if (HASKEY (FIRST (input), "PLUS")) {
    return intNew (+INT (v)->value);
  } else if (HASKEY (FIRST (input), "MINUS")) {
    return intNew (-INT (v)->value);
  }
  return NULL;
}

Value *evPower (Value *input)
{
  Value *left, *right;
  left = evUnary (FIRST (input));
  right = SECOND (input);

  while (!NILP (right)) {
    if (HASKEY (CAR (right), "POWER")) {
      right = CDR (right);
      left = intNew (pow (INT (left)->value,
                          INT (evUnary (CAR (right)))->value));
    } else if (HASKEY (CAR (right), "MOD")) {
      right = CDR (right);
      left = intNew (BINOP (left, &, evUnary (CAR (right))));
    }
    right = CDR (right);
  }
  return left;
}

Value *evFactor (Value *input)
{
  Value *left, *right;
  left = evPower (FIRST (input));
  right = SECOND (input);

  while (!NILP (right)) {
    if (HASKEY (CAR (right), "STAR")) {
      right = CDR (right);
      left = intNew (BINOP (left, *, evPower (CAR (right))));
    } else if (HASKEY (CAR (right), "SLASH")) {
      right = CDR (right);
      left = intNew (BINOP (left, /, evPower (CAR (right))));
    }
    right = CDR (right);
  }
  return left;
}

Value *evTerm (Value *input)
{
  Value *left, *right;
  left = evFactor (FIRST (input));
  right = SECOND (input);

  while (!NILP (right)) {
    if (HASKEY (CAR (right), "PLUS")) {
      right = CDR (right);
      left = intNew (BINOP (left, +, evFactor (CAR (right))));
    } else if (HASKEY (CAR (right), "MINUS")) {
      right = CDR (right);
      left = intNew (BINOP (left, -, evFactor (CAR (right))));
    }
    right = CDR (right);
  }
  return left;
}

Value *calculate (Value *input)
{
  /* Unwrap "Calculator" Rule */
  return evTerm (FIRST (input));
}

int main ()
{
  Machine m;
  size_t grammar_size = 0, input_size = 0;
  Bytecode *grammar = NULL;
  char *input = NULL;
  bool running = true;
  Value *result, *tree;

  readFile ("calc.binx", &grammar, &grammar_size);

  while (running) {
    if ((input = readline ("calc% ")) == NULL) break;
    input_size = strlen (input);
    if (input_size > 0) {
      add_history (input);

      mInit (&m);
      mLoad (&m, grammar);
      if ((mMatch (&m, input, input_size, &tree)) == 0) {
        result = calculate (tree);
        valPrint (result); printf ("\n");
      }
      mFree (&m);
    }
    free (input);
  }

  return 0;
}
