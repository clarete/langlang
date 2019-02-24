/* value.c - Representation of values
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
#include <assert.h>
#include <string.h>
#include <stdio.h>

#include "error.h"
#include "value.h"

/* Static Objects */
const Object *Nil = (&(Object) { TYPE_NIL, 0 });

/* ---- Object ---- */

Object *objNew (Type type, size_t size)
{
  Object *obj;
  if ((obj = malloc (size)) == NULL) FATAL ("Can't make new object: OOM");
  obj->type = type;

  /* TODO: Receive context to associate object */
  /* obj->next = c->nextObject; */
  /* c->nextObject = obj; */
  return obj;
}

void objFree (Object *o)
{
  Object *tmp;
  switch (o->type) {
    /* Statically allocated, don't free it! */
  case TYPE_NIL: break;
    /* Won't be freed til the end when symbol table is freed */
  case TYPE_SYMBOL: break;
    /* Leaf-node, just free it */
  case TYPE_INT: free (INT (o)); break;
    /* Leaf-node, just free it */
  case TYPE_STRING: free (STRING (o)); break;
    /* Recursive case */
  case TYPE_CONS:
    while (CONSP (o)) {
      tmp = o;
      objFree (CAR (tmp));
      o = CDR (tmp);
      free (CONS (tmp));
    }
    break;
    /* Error Handling */
  default:
    fprintf (stderr, "Invalid Object Type\n");
    break;
  }
}

static void objPrintIndent (const Object *obj, int level);

void objPrint (const Object *obj)
{
  objPrintIndent (obj, 0);
}

/* Cons */

Object *consNew (Object *car, Object *cdr)
{
  Cons *cons = CONS (objNew (TYPE_CONS, sizeof (Cons)));
  cons->car = car;
  cons->cdr = cdr;
  return (Object *) cons;
}

/* String */

Object *stringNew (const char *p, size_t len)
{
  String *str;
  str = STRING (objNew (TYPE_STRING, sizeof (String)));
  memcpy (str->value, p, len);
  str->value[len] = '\0';
  str->len = len;
  return OBJ (str);
}

size_t stringLen (Object *s)
{
  return STRING (s)->len;
}

char stringCharAt (Object *s, size_t i)
{
  return STRING (s)->value[i];
}

/* Int */

Object *intNew (long int v)
{
  Int *o = INT (objNew (TYPE_INT, sizeof (Int)));
  o->value = v;
  return (Object *) o;
}

/* Symbol */

Object *symbolNew (const char *p, size_t len)
{
  Symbol *symbol;
  symbol = SYMBOL (objNew (TYPE_SYMBOL, sizeof (Symbol)));
  memcpy (symbol->name, p, len);
  symbol->name[len] = '\0';
  symbol->len = len;
  return (Object *) symbol;
}

/* ---- List ---- */

static void listAdjust (List *lst, size_t osz);

void listInit (List *lst)
{
  lst->items = NULL;
  lst->capacity = 0;
  lst->used = 0;
  lst->o.type = TYPE_LIST;
}

void listFree (List *lst)
{
  for (size_t i = 0; i < lst->used; i++)
    free (lst->items[i]);
  free (lst->items);
  listInit (lst);
}

static void listAdjust (List *lst, size_t osz)
{
  void *tmp;
  if (lst->used == lst->capacity) {
    lst->capacity = lst->capacity == 0 ? INIT_LIST_SIZE : lst->capacity * 2;
    if ((tmp = realloc (lst->items, lst->capacity * osz)) == NULL) {
      free (lst->items);
      FATAL ("Can't allocate memory");
    }
    lst->items = tmp;
  }
}

uint32_t listPush (List *lst, Object *o)
{
  listAdjust (lst, sizeof (Object *));
  lst->items[lst->used++] = o;
  return lst->used;
}

Object *listPop (List *lst)
{
  Object *tmp;
  assert (listLen (lst));
  tmp = listTop (lst);
  /* TODO: Should we have something like listItemSet? */
  listItem (lst, listLen (lst)) = NULL;
  lst->used--;
  return tmp;
}

Object *listTop (List *lst)
{
  assert (listLen (lst));
  return listItem (lst, listLen (lst)-1);
}

/* Object Equality */

bool consEqual (const Object *o1, const Object *o2)
{
  /* TODO: Add dynamically allocated stack */
  const Cons *stack[1048] = { 0 };
  const Cons **stack_top = stack;
  const Cons *current = NULL;
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
         different. Both should also be cons cells. */
      if (NILP (tmp1) && NILP (tmp2)) return true;
      else if (NILP (tmp1) || NILP (tmp2)) return false;
      if (!CONSP (tmp1) || !CONSP (tmp2)) return false;
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
  case TYPE_SYMBOL: return SYMBOL (o1) == SYMBOL (o2);
  case TYPE_STRING: return strcmp (STRING (o1)->value, STRING (o2)->value) == 0;
  case TYPE_CONS: return consEqual (o1, o2);
  default: FATAL ("Unknown type passed to objPrint: %d\n", o1->type);
  }
}

/* Print facilities */

#define INDENTED(n, s)                                          \
  do {                                                          \
    for (int i = 0; i < n; i++) printf (" ");                   \
    printf (s);                                                 \
  } while (0)

static void symbolPrint (const Object *symbol);

static void consPrint (Cons *obj, int level)
{
  Cons *tmp;
  if (level > 0) printf ("\n");
  INDENTED (level, "(");
  for (tmp = obj; tmp && tmp->car; tmp = CONS (tmp->cdr)) {
    objPrintIndent (tmp->car, level+1);
    if (tmp->cdr) {
      if (NILP (tmp->cdr))
        break;
      if (!CONSP (tmp->cdr)) {
        printf (" . ");
        objPrintIndent (tmp->cdr, level+1);
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

static void symbolPrint (const Object *symbol)
{
  printf ("\"");
  rawPrint (SYMBOL (symbol)->name, SYMBOL (symbol)->len);
  printf ("\"");
}

static void stringPrint (const Object *symbol)
{
  printf ("\"");
  rawPrint (STRING (symbol)->value, STRING (symbol)->len);
  printf ("\"");
}

static void objPrintIndent (const Object *obj, int level)
{
  if (!obj) {
    printf ("NULL");
  } else {
    switch (obj->type) {
    case TYPE_SYMBOL: symbolPrint (obj); break;
    case TYPE_STRING: stringPrint (obj); break;
    case TYPE_NIL: printf ("nil"); break;
    case TYPE_CONS: consPrint (CONS (obj), level); break;
    case TYPE_INT: printf ("%ld", INT (obj)->value); break;
    default: FATAL ("Unknown type passed to objPrint: %d\n", obj->type);
    }
  }
}
