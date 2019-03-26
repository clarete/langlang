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
#define VAL(x)      ((Value *) x)
#define SYMBOL(x)   ((Symbol *) x)
#define CONS(x)     ((Cons *) x)
#define INT(x)      ((Int *) x)
#define BOOL(x)     ((Bool *) x)
#define STRING(x)   ((String *) x)
#define LIST(x)     ((List *) x)
#define DICT(x)     ((Dict *) x)

/* Predicates used in the C environment. */
#define SYMBOLP(o)  (VAL (o)->type == TYPE_SYMBOL)
#define CONSP(o)    (VAL (o)->type == TYPE_CONS)
#define NILP(o)     (VAL (o)->type == TYPE_NIL)
#define BOOLP(o)    (VAL (o)->type == TYPE_BOOL)
#define INTP(o)     (VAL (o)->type == TYPE_INT)
#define STRINGP(o)  (VAL (o)->type == TYPE_STRING)
#define LISTP(o)    (VAL (o)->type == TYPE_LIST)
#define DICTP(o)    (VAL (o)->type == TYPE_DICT)

/* Utilities */
#define CAR(o) (CONS (o)->car)
#define CDR(o) (CONS (o)->cdr)

typedef enum {
  TYPE_SYMBOL = 1,
  TYPE_CONS,
  TYPE_NIL,
  TYPE_INT,
  TYPE_BOOL,
  TYPE_STRING,
  TYPE_LIST,
  TYPE_DICT,
  TYPE_END
} Type;

typedef struct val {
  Type type;
  struct val *next;
} Value;

typedef struct {
  Value o;
  Value *car;
  Value *cdr;
} Cons;

typedef struct {
  Value o;
  uint32_t len;
  char name[MAX_SYMBOL_SIZE];
} Symbol;

typedef struct {
  Value o;
  uint32_t len;
  char value[MAX_SYMBOL_SIZE];
} String;

typedef struct {
  Value o;
  long int value;
} Int;

typedef struct {
  Value o;
  bool value;
} Bool;

typedef struct {
  Value o;
  uint32_t used;
  uint32_t capacity;
  Value **items;
} List;

typedef struct {
  Value o;
  uint32_t used;
  uint32_t capacity;
  Value **values;
} Dict;

extern const Value *Nil;
extern const Value *True;
extern const Value *False;

void valPrint (const Value *o);
bool valEqual (const Value *o1, const Value *o2);
void valFree (Value *o);
uint32_t valHash (Value *o);

Value *consNew (Value *car, Value *cdr);
Value *symbolNew (const char *p, size_t len);
Value *intNew (long int v);

Value *stringNew (const char *p, size_t len);
size_t stringLen (String *s);
char stringCharAt (String *s, size_t i);
#define stringAsCharArr(s) ((s)->value)

void listInit (List *lst);
void listFree (List *lst);
uint32_t listPush (List *lst, Value *o);
Value *listPop (List *lst);
Value *listTop (List *lst);
#define listItem(o,i) ((o)->items[i])
#define listLen(o) ((o)->used)

void dictInit (Dict *d);
void dictFree (Dict *d);
bool dictSet (Dict *d, Value *k, Value *v);
bool dictGet (Dict *d, Value *k, Value **v);
bool dictDel (Dict *d, Value *k);
#define dictLen(d) ((d)->used)
#define dictItem(d,i) ((d)->values[i])

#endif  /* VALUE_GUARD */
