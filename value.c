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

Object *makeAtom (const char *p, size_t len)
{
  Atom *atom;
  atom = ATOM (makeObject (TYPE_ATOM, sizeof (Atom)));
  memcpy (atom->name, p, len);
  atom->name[len] = '\0';
  atom->len = len;
  return (Object *) atom;
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
  free (ot->items);
  oTableInit (ot);
}

void *memAlloc (uint32_t used, uint32_t capacity, uint32_t elsize, void *items)
{
  void *tmp;
  if (used == capacity) {
    capacity = capacity == 0 ? 32 : capacity * 2;
    if ((tmp = realloc (items, capacity * elsize)) == NULL) {
      free (items);
      FATAL ("Can't allocate memory");
    }
    return tmp;
  }
  return items;
}

void oTableAdjust (ObjectTable *ot, size_t osz)
{
  ot->items = memAlloc (ot->used, ot->capacity, osz, ot->items);
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

/* Print facilities */

static void printCons (Cons *obj)
{
  Cons *tmp;
  printf ("(");
  for (tmp = obj; tmp && tmp->car; tmp = CONS (tmp->cdr)) {
    printObj (tmp->car);
    if (tmp->cdr) {
      if (!CONSP (tmp->cdr)) {
        printf (" . ");
        printObj (tmp->cdr);
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
  for (size_t i = 0; i < len; i++) {
    c = s[i];
    if (escape[c] == NULL) printf("%c", c);
    else printf("%s", escape[c]);
  }
}

static void printAtom (const Object *atom)
{
  rawPrint (ATOM (atom)->name, ATOM (atom)->len);
}

void printObj (const Object *obj)
{
  if (!obj) {
    printf ("NULL");
  } else {
    switch (obj->type) {
    case TYPE_ATOM: printAtom (obj); break;
    case TYPE_NIL: printf ("nil"); break;
    case TYPE_CONS: printCons (CONS (obj)); break;
    default: FATAL ("Unknown type passed to printObj: %d\n", obj->type);
    }
  }
}
