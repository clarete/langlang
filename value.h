/* value.h - Representation of values
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
#ifndef VALUE_GUARD
#define VALUE_GUARD

#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>

/* Constants */
#define MAX_SYMBOL_SIZE    256
#define INIT_LIST_SIZE     32

/* Type cast shortcuts */
#define OBJ(x)      ((Object *) x)
#define SYMBOL(x)   ((Symbol *) x)
#define CONS(x)     ((Cons *) x)
#define INT(x)      ((Int *) x)
#define STRING(x)   ((String *) x)
#define LIST(x)     ((List *) x)
#define DICT(x)     ((Dict *) x)

/* Predicates used in the C environment. */
#define SYMBOLP(o)  (OBJ (o)->type == TYPE_SYMBOL)
#define CONSP(o)    (OBJ (o)->type == TYPE_CONS)
#define NILP(o)     (OBJ (o)->type == TYPE_NIL)
#define INTP(o)     (OBJ (o)->type == TYPE_INT)
#define STRINGP(o)  (OBJ (o)->type == TYPE_STRING)
#define LISTP(o)    (OBJ (o)->type == TYPE_LIST)
#define DICTP(o)    (OBJ (o)->type == TYPE_DICT)

/* Utilities */
#define CAR(o) (CONS (o)->car)
#define CDR(o) (CONS (o)->cdr)

typedef enum {
  TYPE_SYMBOL = 1,
  TYPE_CONS,
  TYPE_NIL,
  TYPE_INT,
  TYPE_STRING,
  TYPE_LIST,
  TYPE_DICT,
  TYPE_END
} Type;

typedef struct obj {
  Type type;
  struct obj *next;
} Object;

typedef struct {
  Object o;
  Object *car;
  Object *cdr;
} Cons;

typedef struct {
  Object o;
  uint32_t len;
  char name[MAX_SYMBOL_SIZE];
} Symbol;

typedef struct {
  Object o;
  uint32_t len;
  char value[MAX_SYMBOL_SIZE];
} String;

typedef struct {
  Object o;
  long int value;
} Int;

typedef struct {
  Object o;
  uint32_t used;
  uint32_t capacity;
  Object **items;
} List;

typedef struct {
  Object o;
  uint32_t used;
  uint32_t capacity;
  Object **values;
} Dict;

extern const Object *Nil;

void objPrint (const Object *o);
bool objEqual (const Object *o1, const Object *o2);
void objFree (Object *o);
uint32_t objHash (Object *o);

Object *consNew (Object *car, Object *cdr);
Object *symbolNew (const char *p, size_t len);
Object *intNew (long int v);

Object *stringNew (const char *p, size_t len);
size_t stringLen (String *s);
char stringCharAt (String *s, size_t i);

void listInit (List *lst);
void listFree (List *lst);
uint32_t listPush (List *lst, Object *o);
Object *listPop (List *lst);
Object *listTop (List *lst);
#define listItem(o,i) ((o)->items[i])
#define listLen(o) ((o)->used)

void dictInit (Dict *d);
void dictFree (Dict *d);
bool dictSet (Dict *d, Object *k, Object *v);
bool dictGet (Dict *d, Object *k, Object **v);
bool dictDel (Dict *d, Object *k);
#define dictLen(d) ((d)->used)
#define dictItem(d,i) ((d)->values[i])

#endif  /* VALUE_GUARD */
