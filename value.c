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

Object *makeObject (Type type, size_t size)
{
  Object *obj = malloc (size);
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

void printObj (const Object *obj)
{
  if (!obj) {
    printf ("NULL");
  } else {
    switch (obj->type) {
    case TYPE_ATOM: printf ("%s", ATOM (obj)->name); break;
    case TYPE_NIL: printf ("nil"); break;
    case TYPE_CONS: printCons (CONS (obj)); break;
    default: FATAL ("Unknown type passed to printObj: %d\n", obj->type);
    }
  }
}
