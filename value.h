/* value.h - Representation of values
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
#ifndef VALUE_GUARD
#define VALUE_GUARD

#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>

/* Constants */
#define MAX_SYMBOL_SIZE    128
#define INIT_OTABLE_SIZE   32

/* Type cast shortcuts */
#define OBJ(x)      ((Object *) x)
#define SYMBOL(x)   ((Symbol *) x)
#define CONS(x)     ((Cons *) x)
#define INT(x)      ((Int *) x)

/* Predicates used in the C environment. */
#define SYMBOLP(o)  (OBJ (o)->type == TYPE_SYMBOL)
#define CONSP(o)    (OBJ (o)->type == TYPE_CONS)
#define NILP(o)     (OBJ (o)->type == TYPE_NIL)
#define INTP(o)     (OBJ (o)->type == TYPE_INT)

/* Utilities */
#define CAR(o) (CONS (o)->car)
#define CDR(o) (CONS (o)->cdr)

typedef enum {
  TYPE_SYMBOL = 1,
  TYPE_CONS,
  TYPE_NIL,
  TYPE_INT,
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
  int8_t len;
  char name[MAX_SYMBOL_SIZE];
} Symbol;

typedef struct {
  Object o;
  long int value;
} Int;

typedef struct {
  void **items;
  uint32_t used;
  uint32_t capacity;
} ObjectTable;

void printObj (const Object *o);
Object *makeCons (Object *car, Object *cdr);
Object *makeSymbol (const char *p, size_t len);
Object *makeInt (long int v);
bool objEqual (const Object *o1, const Object *o2);
void objFree (Object *o);

void oTableInit (ObjectTable *ot);
void oTableFree (ObjectTable *ot);
void oTableAdjust (ObjectTable *ot, size_t osz);
uint32_t oTableInsert (ObjectTable *ot, void *o, size_t osz);
uint32_t oTableInsertObject (ObjectTable *ot, Object *o);
#define oTableItem(o,i) ((o)->items[i])

/* Static object */
#define Nil (&(Object) { TYPE_NIL, 0 })

#endif  /* VALUE_GUARD */
