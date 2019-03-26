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

/* Static Valects */
const Value *Nil = (&(Value) { TYPE_NIL, 0 });
const Value *True = VAL ((&(Bool) {
  .o = { TYPE_BOOL, 0 },
  .value = true,
}));
const Value *False = VAL ((&(Bool) {
  .o = { TYPE_BOOL, 0 },
  .value = false,
}));

/* Memory Allocation */

#define INCR_CAPACITY(capacity)                 \
  ((capacity) < 8 ? 8 : (capacity) * 2)
#define ALLOC(type, count)                      \
  (type *) memFn (NULL, sizeof (type) * (count))
#define REALLOC(ptr, type, oldCount, count)     \
  (type *) memFn (ptr, sizeof (type) * (count))

static void *memFn (void *o, uint32_t newSize)
{
  void *alloc;
  if (newSize == 0) {
    free (o);
    return NULL;
  }
  alloc = realloc (o, newSize);
  if (!alloc) {
    free (o);
    FATAL ("Can't allocate memory");
  }
  return alloc;
}

/* ---- Value ---- */

Value *valNew (Type type, size_t size)
{
  Value *val;
  if ((val = malloc (size)) == NULL) FATAL ("Can't make new value: OOM");
  val->type = type;

  /* TODO: Receive context to associate value */
  /* val->next = c->nextValue; */
  /* c->nextValue = val; */
  return val;
}

void valFree (Value *o)
{
  Value *tmp;
  switch (o->type) {
    /* Statically allocated, don't free it! */
  case TYPE_NIL: break;
  case TYPE_BOOL: break;
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
      valFree (CAR (tmp));
      o = CDR (tmp);
      free (CONS (tmp));
    }
    break;
    /* Error Handling */
  default:
    fprintf (stderr, "Invalid value type passed to valFree\n");
    break;
  }
}

/* djb2 */
uint32_t stringHash (String *o)
{
  uint32_t i, hash = 5381;
  for (i = 0; i < stringLen (o); i++) {
    hash = ((hash << 5) + hash) + stringCharAt (o, i);
  }
  return hash;
}

uint32_t valHash (Value *o) {
  if (STRINGP (o)) {
    return stringHash (STRING (o));
  }
  return 0;
}

static void valPrintIndent (const Value *val, int level);

void valPrint (const Value *val)
{
  valPrintIndent (val, 0);
}

/* Cons */

Value *consNew (Value *car, Value *cdr)
{
  Cons *cons = CONS (valNew (TYPE_CONS, sizeof (Cons)));
  cons->car = car;
  cons->cdr = cdr;
  return (Value *) cons;
}

/* String */

Value *stringNew (const char *p, size_t len)
{
  String *str;
  str = STRING (valNew (TYPE_STRING, sizeof (String)));
  memcpy (str->value, p, len);
  str->value[len] = '\0';
  str->len = len;
  return VAL (str);
}

size_t stringLen (String *s)
{
  return s->len;
}

char stringCharAt (String *s, size_t i)
{
  return s->value[i];
}

/* Int */

Value *intNew (long int v)
{
  Int *o = INT (valNew (TYPE_INT, sizeof (Int)));
  o->value = v;
  return (Value *) o;
}

/* Symbol */

Value *symbolNew (const char *p, size_t len)
{
  Symbol *symbol;
  symbol = SYMBOL (valNew (TYPE_SYMBOL, sizeof (Symbol)));
  memcpy (symbol->name, p, len);
  symbol->name[len] = '\0';
  symbol->len = len;
  return (Value *) symbol;
}

/* ---- List ---- */

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
    valFree (lst->items[i]);
  free (lst->items);
  listInit (lst);
}

uint32_t listPush (List *lst, Value *o)
{
  uint32_t oldCapacity;
  if (lst->capacity < lst->used + 1) {
    oldCapacity = lst->capacity;
    lst->capacity = INCR_CAPACITY (oldCapacity);
    lst->items = REALLOC (lst->items, Value *,
                          oldCapacity, lst->capacity);
  }
  lst->items[lst->used++] = o;
  return lst->used;
}

Value *listPop (List *lst)
{
  Value *tmp;
  assert (listLen (lst));
  tmp = listTop (lst);
  /* TODO: Should we have something like listItemSet? */
  listItem (lst, listLen (lst)) = NULL;
  lst->used--;
  return tmp;
}

Value *listTop (List *lst)
{
  assert (listLen (lst));
  return listItem (lst, listLen (lst)-1);
}

/* Dict */

void dictInit (Dict *dct)
{
  dct->values = NULL;
  dct->capacity = 0;
  dct->used = 0;
  dct->o.type = TYPE_DICT;
}

void dictFree (Dict *dct)
{
  for (size_t i = 0; i < dct->used; i++)
    valFree (dct->values[i]);
  free (dct->values);
  dictInit (dct);
}

Value *dictFind (Dict *dct, Value *k)
{
  uint64_t hash;
  uint32_t index;
  Value *tmp, *entries;

  if (dictLen (dct) == 0) return NULL;

  hash = valHash (k);
  index = hash % dct->capacity;
  entries = dictItem (dct, index);

  for (tmp = entries; !NILP (tmp); tmp = CDR (tmp)) {
    if (valEqual (CAR (CAR (tmp)), k)) {
      return CAR (tmp);
    }
  }
  return NULL;
}

bool dictSet (Dict *dct, Value *k, Value *v)
{
  uint32_t index, hash, i, oldCapacity;
  Value **values, *tmp, *o;
  if ((o = dictFind (dct, k)) != NULL) {
    CDR (o) = v;
    return false;               /* Not created */
  }
  if (dct->capacity < dct->used + 1) {
    oldCapacity = dct->capacity;
    /* Allocate larger space */
    dct->capacity = INCR_CAPACITY (oldCapacity);
    values = ALLOC (Value *, dct->capacity);
    for (i = 0; i < dct->capacity; i++) {
      values[i] = VAL (Nil);
    }
    /* Re-hash all existing keys */
    for (i = 0; i < oldCapacity; i++) {
      o = dictItem (dct, i);
      for (tmp = o; !NILP (tmp); tmp = CDR (tmp)) {
        hash = valHash (CAR (CAR (tmp)));
        index = hash % dct->capacity;
        values[index] = consNew (CAR (tmp), values[index]);
      }
    }
    dct->values = values;
  }
  /* Insert the new item into the table */
  index = valHash (k) % dct->capacity;
  dct->values[index] = consNew (consNew (k, v), dct->values[index]);

  /* printf ("hash: %10d, index: %2d, ", stringHash (k), index); */
  /* valPrint (k); */
  /* printf ("\n"); */

  dct->used++;
  return true;                  /* Just Created */
}

bool dictGet (Dict *dct, Value *k, Value **v)
{
  Value *o = dictFind (dct, k);
  if (o) *v = CDR (o);
  return o != NULL;
}

bool dictDel (Dict *dct, Value *k)
{
  uint64_t hash;
  uint32_t index;
  Value *tmp, *prev, *next, *entries;

  if (dictLen (dct) == 0) return false;

  hash = valHash (k);
  index = hash % dct->capacity;
  entries = dictItem (dct, index);

  prev = NULL;
  for (tmp = entries; !NILP (tmp); prev = tmp, tmp = CDR (tmp)) {
    if (valEqual (CAR (CAR (tmp)), k)) {
      next = CDR (tmp);
      if (prev) CDR (prev) = next;
      else dct->values[index] = next;
      dct->used--;
      return true;
    }
  }
  return false;
}

/* Value Equality */

bool consEqual (const Value *o1, const Value *o2)
{
  /* TODO: Add dynamically allocated stack */
  const Cons *stack[1048] = { 0 };
  const Cons **stack_top = stack;
  const Cons *current = NULL;
  Value *tmp1, *tmp2;

  if (!CONSP (o1) || !CONSP (o2)) return false;

  /* Add the root of the tree to the top of the stack */
  *stack_top++ = &(Cons){
    .o = { .type = TYPE_CONS },
    .car = VAL (o1),
    .cdr = VAL (o2)
  };

  /* Loop til the stack is empty */
  while (stack_top > stack) {
    current = *--stack_top;
    /* If current pair in the stack has different type, bail */
    if (VAL (CAR (current))->type != VAL (CDR (current))->type)
      return false;
    /* If current pair isn't a Cons and isn't equal, bail */
    if (!CONSP (CAR (current)) && !valEqual (CAR (current), CDR (current)))
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

bool listEqual (const Value *o1, const Value *o2)
{
  /* TODO: Add dynamically allocated stack */
  const Value *stack[1048] = { 0 };
  const Value **stack_top = stack;
  const Value *current = NULL, *left = NULL, *right = NULL;

  /* Add both values cons cell wrapped to the top of the stack */
  *stack_top++ = VAL (consNew (VAL (o1), VAL (o2)));

  while (stack_top > stack) {
    current = *--stack_top;
    left = CAR (current);
    right = CDR (current);
    free (CONS (current));      /* Don't free left & right */

    if (LISTP (left) && LISTP (left)) {
      /* Both are lists but have different length */
      if (listLen (LIST (left)) != listLen (LIST (right)))
        return false;
      /* Add each item to the stack so they can be tested */
      for (uint32_t i = 0; i < listLen (LIST (left)); i++)
        *stack_top++ = VAL (consNew (listItem (LIST (left), i),
                                     listItem (LIST (right), i)));
    } else if (!LISTP (left) && !LISTP (right)) {
      /* None are lists, we can compare them with the recursive
       * function */
      if (!valEqual (left, right)) return false;
    }
  }

  return true;
}

bool valEqual (const Value *o1, const Value *o2)
{
  if (o1->type != o2->type) return false;
  switch (o1->type) {
  case TYPE_NIL: return true;
  case TYPE_INT: return INT (o1)->value == INT (o2)->value;
  case TYPE_BOOL: return o1 == o2;
  /* TODO: Should compare pointer, will fix after adding lookup to
     symbol factory */
  case TYPE_SYMBOL: return SYMBOL (o1) == SYMBOL (o2);
  case TYPE_STRING: return strcmp (STRING (o1)->value, STRING (o2)->value) == 0;
  case TYPE_CONS: return consEqual (o1, o2);
  case TYPE_LIST: return listEqual (o1, o2);
  default: FATAL ("Unknown type passed to valEqual: %d\n", o1->type);
  }
}

/* Print facilities */

#define INDENTED(n, s)                                          \
  do {                                                          \
    for (int i = 0; i < n; i++) printf (" ");                   \
    printf (s);                                                 \
  } while (0)

static void symbolPrint (const Value *symbol);

static void consPrint (Cons *val, int level)
{
  Cons *tmp;
  if (level > 0) printf ("\n");
  INDENTED (level, "(");
  for (tmp = val; tmp && tmp->car; tmp = CONS (tmp->cdr)) {
    valPrintIndent (tmp->car, level+1);
    if (tmp->cdr) {
      if (NILP (tmp->cdr))
        break;
      if (!CONSP (tmp->cdr)) {
        printf (" . ");
        valPrintIndent (tmp->cdr, level+1);
        break;
      } else {
        printf (" ");
      }
    }
  }
  printf (")");
}

static void listPrint (List *lst, int level)
{
  if (level > 0) printf ("\n");
  INDENTED (level, "[");
  for (uint32_t i = 0; i < listLen (lst); i++) {
    valPrintIndent (listItem (lst, i), level+1);
    if (i != listLen (lst)-1) printf (", ");
  }
  printf ("]");
}

static void dictPrint (Dict *dict, int level)
{
  Value *o, *tmp;
  uint32_t found = 0;
  INDENTED (level, "{");
  for (uint32_t i = 0; i < dict->capacity; i++) {
    if (NILP (dictItem (dict, i))) continue;
    o = dictItem (dict, i);
    for (tmp = o; !NILP (tmp); tmp = CDR (tmp)) {
      valPrint (CAR (CAR (tmp)));
      printf (": ");
      valPrint (CDR (CAR (tmp)));
      if (found++ != dictLen (dict)-1) printf (", ");
    }
  }
  printf ("}");
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

static void symbolPrint (const Value *symbol)
{
  printf ("\"");
  rawPrint (SYMBOL (symbol)->name, SYMBOL (symbol)->len);
  printf ("\"");
}

static void stringPrint (const Value *symbol)
{
  printf ("\"");
  rawPrint (STRING (symbol)->value, STRING (symbol)->len);
  printf ("\"");
}

static void valPrintIndent (const Value *val, int level)
{
  if (!val) {
    printf ("NULL");
  } else {
    switch (val->type) {
    case TYPE_NIL: printf ("nil"); break;
    case TYPE_INT: printf ("%ld", INT (val)->value); break;
    case TYPE_BOOL: printf ("%s", BOOL (val)->value ? "true" : "false"); break;
    case TYPE_SYMBOL: symbolPrint (val); break;
    case TYPE_STRING: stringPrint (val); break;
    case TYPE_CONS: consPrint (CONS (val), level); break;
    case TYPE_LIST: listPrint (LIST (val), level); break;
    case TYPE_DICT: dictPrint (DICT (val), level); break;
    default: FATAL ("Unknown type passed to valPrint: %d\n", val->type);
    }
  }
}
