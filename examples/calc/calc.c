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

Object *actionDEC (Object *args, void *data)
{
  const char *s = stringAsCharArr (STRING (CADR (args)));
  return intNew (strtol (s, NULL, 10));
}

Object *actionBIN (Object *args, void *data)
{
  const char *s = stringAsCharArr (STRING (CADR (args)));
  return intNew (strtol (s, NULL, 2));
}

Object *actionHEX (Object *args, void *data)
{
  const char *s = stringAsCharArr (STRING (CADR (args)));
  return intNew (strtol (s, NULL, 16));
}

Object *actionNumber (Object *args, void *data)
{
  return CADR (args);
}

Object *actionPrimary (Object *args, void *data)
{
  return CADR (args);
}

Object *actionUnary (Object *args, void *data)
{
  return CADR (args);
}

Object *actionPower (Object *args, void *data)
{
  Object *right, *left = CADR (args);
  const char *s;
  if (NILP (CDR (CDR (args))))
    return left;
  s = stringAsCharArr (STRING (CAR (CADR (CDR (args)))));
  right = CADR (CDR (CDR (args)));
  if (strcmp (s, "POWER") == 0)
    return intNew (pow (INT (left)->value, INT (right)->value));
  else if (strcmp (s, "MOD") == 0)
    return intNew (INT (left)->value % INT (right)->value);
  return NULL;
}

Object *actionFactor (Object *args, void *data)
{
  Object *right, *left = CADR (args);
  const char *s;
  if (NILP (CDR (CDR (args))))
    return left;
  s = stringAsCharArr (STRING (CAR (CADR (CDR (args)))));
  right = CADR (CDR (CDR (args)));
  if (strcmp (s, "STAR") == 0)
    return intNew (INT (left)->value * INT (right)->value);
  else if (strcmp (s, "SLASH") == 0)
    return intNew (INT (left)->value / INT (right)->value);
  return NULL;
}

Object *actionTerm (Object *args, void *data)
{
  Object *right, *left = CADR (args);
  const char *s;
  if (NILP (CDR (CDR (args))))
    return left;
  s = stringAsCharArr (STRING (CAR (CADR (CDR (args)))));
  right = CADR (CDR (CDR (args)));
  if (strcmp (s, "PLUS") == 0)
    return intNew (INT (left)->value + INT (right)->value);
  else if (strcmp (s, "MINUS") == 0)
    return intNew (INT (left)->value - INT (right)->value);
  return NULL;
}

int main ()
{
  Machine m;
  size_t grammar_size = 0, input_size = 0;
  Bytecode *grammar = NULL;
  char *input = NULL;
  bool running = true;
  Object *tree;

  readFile ("calc.binx", &grammar, &grammar_size);

  mInit (&m);
  mLoad (&m, grammar);

  mPrim (&m, mSymbol (&m, "Number", 6), OBJ (primNew (actionNumber)));
  mPrim (&m, mSymbol (&m, "Primary", 7), OBJ (primNew (actionPrimary)));
  mPrim (&m, mSymbol (&m, "Unary", 5), OBJ (primNew (actionUnary)));
  mPrim (&m, mSymbol (&m, "Power", 5), OBJ (primNew (actionPower)));
  mPrim (&m, mSymbol (&m, "Factor", 6), OBJ (primNew (actionFactor)));
  mPrim (&m, mSymbol (&m, "Term", 4), OBJ (primNew (actionTerm)));
  mPrim (&m, mSymbol (&m, "DEC", 3), OBJ (primNew (actionDEC)));
  mPrim (&m, mSymbol (&m, "BIN", 3), OBJ (primNew (actionBIN)));
  mPrim (&m, mSymbol (&m, "HEX", 3), OBJ (primNew (actionHEX)));

  while (running) {
    if ((input = readline ("calc% ")) == NULL) break;
    input_size = strlen (input);
    if (input_size > 0) {
      add_history (input);
      if (mMatch (&m, input, input_size, &tree) == 0) {
        objPrint (tree); printf ("\n");
      }
    }
    free (input);
  }

  mFree (&m);

  return 0;
}
