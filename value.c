/* value.c - Representation of values
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
#include <string.h>
#include <stdio.h>

#include "error.h"
#include "value.h"

/* ---- Object Factories ---- */

Object *makeObject (Type type, size_t size)
{
  Object *obj;
  if ((obj = malloc (size)) == NULL) FATAL ("Can't make new object: OOM");
  obj->type = type;

  /* TODO: Receive context to associate object */
  /* obj->next = c->nextObject; */
  /* c->nextObject = obj; */
  return obj;
}

Object *makeCons (Object *car, Object *cdr)
{
  Cons *cons = CONS (makeObject (TYPE_CONS, sizeof (Cons)));
  cons->car = car;
  cons->cdr = cdr;
  return (Object *) cons;
}

Object *makeSymbol (const char *p, size_t len)
{
  Symbol *symbol;
  symbol = SYMBOL (makeObject (TYPE_SYMBOL, sizeof (Symbol)));
  memcpy (symbol->name, p, len);
  symbol->name[len] = '\0';
  symbol->len = len;
  return (Object *) symbol;
}

Object *makeInt (long int v)
{
  Int *o = INT (makeObject (TYPE_INT, sizeof (Int)));
  o->value = v;
  return (Object *) o;
}

/* ---- Object Table ---- */

void oTableInit (ObjectTable *ot)
{
  ot->items = NULL;
  ot->capacity = 0;
  ot->used = 0;
}

void oTableFree (ObjectTable *ot)
{
  for (size_t i = 0; i < ot->used; i++)
    free (ot->items[i]);
  free (ot->items);
  oTableInit (ot);
}

void oTableAdjust (ObjectTable *ot, size_t osz)
{
  void *tmp;
  if (ot->used == ot->capacity) {
    ot->capacity = ot->capacity == 0 ? INIT_OTABLE_SIZE : ot->capacity * 2;
    if ((tmp = realloc (ot->items, ot->capacity * osz)) == NULL) {
      free (ot->items);
      FATAL ("Can't allocate memory");
    }
    ot->items = tmp;
  }
}

uint32_t oTableInsertObject (ObjectTable *ot, Object *o)
{
  oTableAdjust (ot, sizeof (Object *));
  ot->items[ot->used++] = o;
  return ot->used;
}

uint32_t oTableInsert (ObjectTable *ot, void *o, size_t osz)
{
  oTableAdjust (ot, osz);
  ot->items[ot->used++] = o;
  return ot->used;
}

bool objConsEqual (const Object *o1, const Object *o2)
{
  /* TODO: Add dynamically allocated stack */
  const Cons *stack[1048] = { 0 };
  const Cons **stack_top = stack;
  const Cons *current;
  Object *tmp1, *tmp2;

  if (!CONSP (o1) || !CONSP (o2)) return false;

  /* Add the root of the tree to the top of the stack */
  *stack_top++ = &(Cons){
    .o = { .type = TYPE_CONS },
    .car = OBJ (o1),
    .cdr = OBJ (o2)
  };

  /* Loop til the stack is empty */
  while (stack_top > stack) {
    current = *--stack_top;
    /* If current pair in the stack has different type, bail */
    if (OBJ (CAR (current))->type != OBJ (CDR (current))->type)
      return false;
    /* If current pair isn't a Cons and isn't equal, bail */
    if (!CONSP (CAR (current)) && !objEqual (CAR (current), CDR (current)))
      return false;
    /* Iterate over Cons items */
    for (tmp1 = CAR (current), tmp2 = CDR (current); ;
         tmp1 = CDR (tmp1), tmp2 = CDR (tmp2)) {
      /* If both are null, they're equal, if just one is null they're
         different */
      if (!tmp1 && !tmp2) return true;
      else if (!tmp1 || !tmp2) return false;
      /* Push the new pair into the stack */
      *stack_top++ = &(Cons){
        .o = { .type = TYPE_CONS },
        .car = CAR (tmp1),
        .cdr = CAR (tmp2)
      };
    }
  }
  return true;
}

bool objEqual (const Object *o1, const Object *o2)
{
  if (o1->type != o2->type) return false;
  switch (o1->type) {
  case TYPE_NIL: return true;
  case TYPE_INT: return INT (o1)->value == INT (o2)->value;
  /* TODO: Should compare pointer, will fix after adding lookup to
     symbol factory */
  case TYPE_SYMBOL: return strcmp (SYMBOL (o1)->name, SYMBOL (o2)->name) == 0;
  case TYPE_CONS: return objConsEqual (o1, o2);
  default: FATAL ("Unknown type passed to printObj: %d\n", o1->type);
  }
}

/* Print facilities */

#define INDENTED(n, s)                                          \
  do {                                                          \
    for (int i = 0; i < n; i++) printf (" ");                   \
    printf (s);                                                 \
  } while (0)

static void printObjIndent (const Object *obj, int level);

static void printCons (Cons *obj, int level)
{
  Cons *tmp;
  if (level > 0) printf ("\n");
  INDENTED (level, "(");
  for (tmp = obj; tmp && tmp->car; tmp = CONS (tmp->cdr)) {
    printObjIndent (tmp->car, level+1);
    if (tmp->cdr) {
      if (!CONSP (tmp->cdr)) {
        printf (" . ");
        printObjIndent (tmp->cdr, level+1);
        break;
      } else {
        printf (" ");
      }
    }
  }
  printf (")");
}

static void rawPrint (const char *s, size_t len)
{
  const char *escape[256] = { NULL }; /* Only good for ascii. */
  int c;
  escape['\0'] = "\\0";
  escape['\r'] = "\\r";
  escape['\n'] = "\\n";
  escape['\\'] = "\\\\";
  escape['"'] = "\\\"";
  for (size_t i = 0; i < len; i++) {
    c = s[i];
    if (escape[c] == NULL) printf("%c", c);
    else printf("%s", escape[c]);
  }
}

static void printSymbol (const Object *symbol)
{
  printf ("\"");
  rawPrint (SYMBOL (symbol)->name, SYMBOL (symbol)->len);
  printf ("\"");
}

static void printObjIndent (const Object *obj, int level)
{
  if (!obj) {
    printf ("NULL");
  } else {
    switch (obj->type) {
    case TYPE_SYMBOL: printSymbol (obj); break;
    case TYPE_NIL: printf ("nil"); break;
    case TYPE_CONS: printCons (CONS (obj), level); break;
    case TYPE_INT: printf ("%ld", INT (obj)->value); break;
    default: FATAL ("Unknown type passed to printObj: %d\n", obj->type);
    }
  }
}

void printObj (const Object *obj)
{
  printObjIndent (obj, 0);
}
