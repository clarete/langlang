#include <stdio.h>
#include <string.h>

#include "../../peg.h"
#include "../../value.h"
#include "../../io.h"

int main (int argc, char **argv)
{
  Machine m;
  size_t grammar_size = 0, input_size = 0;
  Bytecode *grammar = NULL;
  char *input = NULL;
  Object *tree;
  String *label;
  Dict errors;
  uint32_t e;

  if (argc != 2) exit (-1);

  readFile ("tiny.binx", &grammar, &grammar_size);
  readFile (argv[1], (uint8_t **) &input, &input_size);

  mInit (&m);
  mLoad (&m, grammar);

  dictInit (&errors);
  dictSet (&errors, mSymbol (&m, "sc", 2),
           stringNew ("missing ';' at the end of the statement", 39));
  dictSet (&errors, mSymbol (&m, "eif", 3),
           stringNew ("missing expression after if", 27));
  dictSet (&errors, mSymbol (&m, "then", 4),
           stringNew ("missing 'then' after if", 23));
  dictSet (&errors, mSymbol (&m, "cs1", 3),
           stringNew ("missing Expression after then", 29));
  dictSet (&errors, mSymbol (&m, "cs2", 3),
           stringNew ("missing Expression after else", 29));
  dictSet (&errors, mSymbol (&m, "end", 3),
           stringNew ("missing 'END' after if", 22));

  if ((e = mMatch (&m, input, input_size, &tree)) == 0) {
    objPrint (tree);
    printf ("\n");
  } else if (e > 1) {
    label = STRING (listItem (&m.symbols, e-2));
    Object *msg = NULL,
      *sym = mSymbol (&m, stringAsCharArr (label), stringLen (label));
    printf ("Syntax error: ");
    if (dictGet (&errors, sym, &msg))
      printf ("%s", stringAsCharArr (STRING (msg)));
    else
      objPrint (OBJ (label));
    printf ("\n");
  }

  free (input);
  mFree (&m);

  return 0;
}
