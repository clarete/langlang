/* -*- coding: utf-8; -*-
 *
 * peg.c - Implementation of Parsing Machine for PEGs
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
#include <stdint.h>
#include <stdio.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>
#include <assert.h>

#include "debug.h"
#include "peg.h"
#include "value.h"
#include "io.h"

/* -- Error control & report utilities -- */

/** Retrieve name of opcode */
#define OP_NAME(o) opNames[o]

/* Helps debugging */
static const char *opNames[OP_END] = {
  [OP_CHAR] = "OP_CHAR",
  [OP_ANY] = "OP_ANY",
  [OP_CHOICE] = "OP_CHOICE",
  [OP_COMMIT] = "OP_COMMIT",
  [OP_FAIL] = "OP_FAIL",
  [OP_FAIL_TWICE] = "OP_FAIL_TWICE",
  [OP_PARTIAL_COMMIT] = "OP_PARTIAL_COMMIT",
  [OP_BACK_COMMIT] = "OP_BACK_COMMIT",
  [OP_TEST_CHAR] = "OP_TEST_CHAR",
  [OP_TEST_ANY] = "OP_TEST_ANY",
  [OP_JUMP] = "OP_JUMP",
  [OP_CALL] = "OP_CALL",
  [OP_SPAN] = "OP_SPAN",
  [OP_SET] = "OP_SET",
  [OP_CAP_OPEN] = "OP_CAP_OPEN",
  [OP_CAP_CLOSE] = "OP_CAP_CLOSE",
  [OP_RETURN] = "OP_RETURN",
  [OP_ATOM] = "OP_ATOM",
  [OP_OPEN] = "OP_OPEN",
  [OP_CLOSE] = "OP_CLOSE",
  [OP_CAPCHAR] = "OP_CAPCHAR",
};

/* Set initial values for the machine */
void mInit (Machine *m)
{
  oTableInit (&m->symbols);
  m->stack = calloc (STACK_SIZE, sizeof (CaptureEntry));
  m->captures = NULL;
  m->code = NULL;               /* Will be set by mLoad() */
  m->li = NULL;                 /* Set by Fail */
  m->cap = 0;
}

/* Release the resources used by the machine */
void mFree (Machine *m)
{
  oTableFree (&m->symbols);
  free (m->code);
  free (m->stack);
  free (m->captures);
  m->cap = 0;
  m->captures = NULL;
  m->code = NULL;
  m->stack = NULL;
}

Object *mSymbol (Machine *m, const char *sym, size_t len) {
  uint32_t i;
  Object *symbol;

  for (i = 0; i < m->symbols.used; i++) {
    symbol = oTableItem (&m->symbols, i);
    if (SYMBOL (symbol)->len != len)
      continue;
    if (strncmp (SYMBOL (symbol)->name, sym, len) == 0)
      return symbol;
  }

  symbol = makeSymbol (sym, len);
  oTableInsertObject (&m->symbols, symbol);
  return symbol;
}

#define READ_UINT8(c)  (*c++)
#define READ_UINT16(c) (c+=2, c[-2] << 8 | c[-1])
#define READ_UINT32(c) (c+=4, c[-4] << 24 | c[-3] << 16 | c[-2] << 8 | c[-1])

void mLoad (Machine *m, Bytecode *code)
{
  Instruction *tmp;
  uint32_t instr, i;
  uint8_t headerSize;
  uint16_t code_size;

  /* Header starts with number (in uint16) of entries in the string
     table. For each string in the string table we first read its size
     and then the actual string. */
  headerSize = READ_UINT16 (code);
  DEBUGLN ("   Header(%d)", headerSize);
  for (i = 0; i < headerSize; i++) {
    size_t ssize = READ_UINT8 (code);
    Object *symbol = mSymbol (m, (const char *) code, ssize);
    (void) symbol; /* Next line may not be included in output */
    DEBUGLN ("     0x%0x: String(%ld) %s", i, ssize, SYMBOL (symbol)->name);
    code += ssize; /* Push the cursor to after the string just read */
  }

  /* Code size is a 16bit integer and contains how many instructions
     the program body contains. */
  code_size = READ_UINT16 (code);
  DEBUGLN ("   Code(%d)", code_size);
  if ((tmp = m->code = calloc (code_size, sizeof (Instruction))) == NULL)
    FATAL ("Can't allocate %s", "memory");
  for (i = 0; i < code_size; i++) {
    instr = READ_UINT32 (code);
    tmp->rator = OP_MASK (instr);
    tmp->rand = instr;          /* Use SOPERAND* to access this */
    DEBUG_INSTRUCTION_LOAD ();
    tmp++;
  }
}

static void append (Object *l, Object *i)
{
  Object *tmp = NULL;
  assert (l);
  assert (CONSP (l));
  for (tmp = l; !NILP (tmp) && !NILP (CDR (tmp)); tmp = CDR (tmp));
  CDR (tmp) = makeCons (i, OBJ (Nil));
}

static Object *pop (Object *l)
{
  Object *tmp, *last = NULL;
  assert (l);
  assert (CONSP (l));
  for (tmp = l;
       !NILP (tmp) && !NILP (CDR (tmp)) && !NILP (CDR (CDR (tmp)));
       tmp = CDR (tmp));
  last = CDR (tmp);
  CDR (tmp) = OBJ (Nil);
  return last;
}

static Object *appendChar (Object *s, char c)
{
  assert (STRINGP (s));
  STRING (s)->value[STRING (s)->len++] = c;
  return s;
}

/* Run the matching machine */
Object *mMatch (Machine *m, const char *input, size_t input_size)
{
  BacktrackEntry *sp = m->stack;
  Instruction *pc = m->code;
  const char *i = input;
  uint32_t btCount = 0, ltCount = 0;
  ObjectTable treestk;

  /** Push data onto the machine's stack  */
#define PUSH(ii,pp) do { sp->i = ii; sp->pc = pp; sp++; } while (0)
  /** Pop data from the machine's stack. Notice it doesn't dereference
      the pointer, callers are supposed to do that when needed. */
#define POP() (--sp)
  /** The end of the input is the offset from the cursor to the end of
      the input string. */
#define THE_END (input + input_size)

#define DEBUG_TREE() do {                                       \
    printf ("  STACK: %u:%u\n", btCount, ltCount);              \
    for (uint32_t i = oTableSize (&treestk); i > 0 ; i--) {     \
      printf ("   [%u] ", i-1);                                 \
      printObj (oTableItem (&treestk, i-1));                    \
      printf ("\n   --------------------------------------\n"); \
    }                                                           \
  } while (0)

  DEBUGLN ("   Run");

  oTableInit (&treestk);

  while (true) {
    /* No-op if DEBUG isn't defined */
    DEBUG_INSTRUCTION_NEXT ();
    DEBUG_STACK ();

    switch (pc->rator) {
    case OP_HALT:
      if (m->li && !i) {
        printf ("Match failed at pos %ld\n", m->li - input + 1);
        return NULL;
      } else {
        /* This is another use for `li'. It will store the final
           suffix upon a successful match. It's very useful for
           tests. */
        m->li = i;
        if (oTableSize (&treestk) > 0) {
          Object *tmp = oTablePop (&treestk);
          oTableFree (&treestk);
          return tmp;
        }
        /* It currently means success. */
        return OBJ (Nil);
      }
    case OP_CAP_OPEN:
      /* printf ("OPEN[%c]: %s\n", UOPERAND1 (pc) ? 'T' : 'F', */
      /*         SYMBOL (oTableItem (&m->symbols, */
      /*                             UOPERAND2 (pc)))->name); */
      btCount++;
      if (UOPERAND1 (pc)) {     /* If the match is a terminal */
        oTableInsertObject (&treestk, makeString ("", 0));
      } else {                  /* If the match is a non-terminal */
        Object *node = makeCons (oTableItem (&m->symbols, UOPERAND2 (pc)), OBJ (Nil));
        oTableInsertObject (&treestk, node);
        ltCount = 0;
      }
      pc++;
      continue;
    case OP_CAP_CLOSE:
      /* printf ("CLOSE[%c]: %s\n", UOPERAND1 (pc) ? 'T' : 'F', */
      /*         SYMBOL (oTableItem (&m->symbols, */
      /*                             UOPERAND2 (pc)))->name); */
      if (btCount > 1) {
        append (oTableTop (&treestk), oTablePop (&treestk));
        btCount--;
        ltCount++;
      }
      pc++;
      continue;
    case OP_CAPCHAR:
      /* printf ("CAPCHAR\n"); */
      appendChar (oTableTop (&treestk), *(i-1));
      pc++;
      continue;
    case OP_CHAR:
      DEBUGLN ("       OP_CHAR: `%c' == `%c' ? %d", *i,
               UOPERAND0 (pc), *i == UOPERAND0 (pc));
      /* printf ("CHAR: `%c' == `%c'\n", */
      /*         *i == '\n' ? 'N' : *i, */
      /*         UOPERAND0 (pc) == '\n' ? 'N' : UOPERAND0 (pc)); */
      if (i < THE_END && *i == UOPERAND0 (pc)) { i++; pc++; }
      else goto fail;
      continue;
    case OP_ANY:
      DEBUGLN ("       OP_ANY: `%c' < |s| ? %d", *i, i < THE_END);
      /* printf ("ANY: %c\n", *i); */
      if (i < THE_END) { i++; pc++; }
      else goto fail;
      continue;
    case OP_SPAN:
      DEBUGLN ("       OP_SPAN: `%c' in [%c(%d)-%c(%d)]", *i,
               UOPERAND1 (pc), UOPERAND1 (pc),
               UOPERAND2 (pc), UOPERAND2 (pc));
      /* printf ("SPAN: %c\n", *i); */
      if (*i >= UOPERAND1 (pc) && *i <= UOPERAND2 (pc)) { i++; pc++; }
      else goto fail;
      continue;
    case OP_CHOICE:
      sp->btCount = btCount;
      sp->ltCount = ltCount;
      PUSH (i, pc + UOPERAND0 (pc));
      pc++;
      continue;
    case OP_COMMIT:
      assert (sp > m->stack);
      POP ();                   /* Discard backtrack entry */
      pc += SOPERAND0 (pc);     /* Jump to the given position */
      continue;
    case OP_PARTIAL_COMMIT:
      assert (sp > m->stack);
      pc += SOPERAND0 (pc);
      (sp - 1)->i = i;
      continue;
    case OP_BACK_COMMIT:
      assert (sp > m->stack);
      i = POP ()->i;
      pc += SOPERAND0 (pc);
      continue;
    case OP_JUMP:
      pc = m->code + SOPERAND0 (pc);
      continue;
    case OP_CALL:
      PUSH (NULL, pc + 1);
      pc += SOPERAND0 (pc);
      continue;
    case OP_RETURN:
      assert (sp > m->stack);
      pc = POP ()->pc;
      continue;
    case OP_FAIL_TWICE:
      POP ();                   /* Drop top of stack & Fall through */
    case OP_FAIL:
    fail:
      /* No-op if DEBUG isn't defined */
      DEBUG_FAILSTATE ();

      if (sp > m->stack) {
        /* Fail〈(pc,i1):e〉 ----> 〈pc,i1,e〉 */
        do i = POP ()->i;
        while (i == NULL && sp > m->stack);
        pc = sp->pc;            /* Restore the program counter */
        /* Non-Terminals can't produce errors, so let's save the last
           known position before moving on */
        if (i) m->li = i;

        /* printf (" FAIL[%u:%u-%u:%u]: %c\n", */
        /*         btCount, sp->btCount, */
        /*         ltCount, sp->ltCount, */
        /*         *i == '\n' ? 'N' : *i); */

        /* Clean capture from sequence */
        while (ltCount > sp->ltCount) {
          objFree (pop (oTableTop (&treestk)));
          ltCount--;
        }

        /* Clean capture stack in depth */
        while (btCount > sp->btCount) {
          objFree (oTablePop (&treestk));
          btCount--;
        }
      } else {
        /* 〈pc,i,e〉 ----> Fail〈e〉 */
        oTableFree (&treestk);
        return NULL;
      }
      continue;
    default:
      FATAL ("Unknown Instruction 0x%04x [%s]", pc->rator, OP_NAME (pc->rator));
    }
  }

#undef POP
#undef PUSH
#undef PUSH_CAP
}

Object *mMatchList (Machine *m, Object *input)
{
  BacktrackEntry *sp = m->stack;
  Instruction *pc = m->code;
  Object *l = input;
  ObjectTable parents;
  Symbol *sym;

  /** Push data onto the machine's stack  */
#define PUSH(ll,pp) do { sp->l = ll; sp->pc = pp; sp++; } while (0)
  /** Pop data from the machine's stack. Notice it doesn't dereference
      the pointer, callers are supposed to do that when needed. */
#define POP() (--sp)
  /** Parent node stack item starts from top */
#define PARENT(n) (oTableItem (&parents, oTableSize (&parents)-1-n))

  DEBUGLN ("   Run");

  oTableInit (&parents);

  while (true) {
    /* No-op if DEBUG isn't defined */
    DEBUG_INSTRUCTION_NEXT ();
    DEBUG_STACK ();

    switch (pc->rator) {
    case OP_HALT:
      oTableFree (&parents);
      return l;
    case OP_OPEN:
      if (!CONSP (l) || !CONSP (CAR (l))) goto fail;
      PUSH (CDR (l), pc++); l = CAR (l);
      oTableInsertObject (&parents, NULL);
      continue;
    case OP_CLOSE:
      if (!NILP (l)) goto fail;
      l = POP ()->l; pc++;
      if (oTableSize (&parents) > 1 && PARENT (0))
        append (PARENT (1), PARENT (0));
      if (oTableSize (&parents) == 1)
        l = oTableItem (&parents, 0);
      parents.used--;
      continue;
    case OP_ATOM:
      /* Did the machine receive the right parameter? */
      sym = SYMBOL (oTableItem (&m->symbols, UOPERAND0 (pc)));
      if (!sym) goto fail;
      DEBUGLN ("       OP_ATOM: `%s' == `%s'",
               sym->name, SYMBOL (CAR (l))->name);
      /* Is it a valid subject? */
      if (!CONSP (l)) goto fail;
      if (CONSP (CAR (l))) goto fail;
      /* Does it match with the atom we're looking for? */
      if (strncmp (SYMBOL (CAR (l))->name, sym->name, sym->len)) goto fail;
      /* Append match to the output list */
      if (oTableSize (&parents)) {
        if (PARENT (0)) append (PARENT (0), CAR (l));
        else PARENT (0) = makeCons (CAR (l), OBJ (Nil));
      }
      /* Crank the machine to go to the next element & instruction */
      l = CDR (l); pc++;
      continue;
    case OP_ANY:
      DEBUGLN ("       OP_ANY: %d", l != NULL && l != Nil);
      if (!l || NILP (l)) goto fail;
      if (oTableSize (&parents)) {
        if (PARENT (0)) append (PARENT (0), CAR (l));
        else PARENT (0) = makeCons (CAR (l), OBJ (Nil));
      }
      l = CDR (l); pc++;
      continue;
    case OP_SPAN:
      WARN ("SPAN instruction is noop for lists");
      continue;
    case OP_CHOICE:
      PUSH (l, pc + UOPERAND0 (pc));
      pc++;
      continue;
    case OP_COMMIT:
      assert (sp > m->stack);
      POP ();                   /* Discard backtrack entry */
      pc += SOPERAND0 (pc);     /* Jump to the given position */
      continue;
    case OP_PARTIAL_COMMIT:
      assert (sp > m->stack);
      pc += SOPERAND0 (pc);
      (sp - 1)->l = l;
      continue;
    case OP_BACK_COMMIT:
      assert (sp > m->stack);
      l = POP ()->l;
      pc += SOPERAND0 (pc);
      continue;
    case OP_JUMP:
      pc = m->code + SOPERAND0 (pc);
      continue;
    case OP_CALL:
      PUSH (NULL, pc + 1);
      pc += SOPERAND0 (pc);
      continue;
    case OP_RETURN:
      assert (sp > m->stack);
      pc = POP ()->pc;
      continue;
    case OP_FAIL_TWICE:
      POP ();                   /* Drop top of stack & Fall through */
    case OP_FAIL:
    fail:
      /* No-op if DEBUG isn't defined */
      DEBUG_FAILSTATE2 ();

      if (sp > m->stack) {
        /* Fail〈(pc,i1):e〉 ----> 〈pc,i1,e〉 */
        do l = POP ()->l;
        while (l == NULL && sp > m->stack);
        pc = sp->pc;            /* Restore the program counter */
        if (oTableSize (&parents) > 1) {
          /* Clean entries created by OPEN and not used because of
             backtracking */
          while (!PARENT (0)) parents.used--;
        }
      } else {
        /* 〈pc,i,e〉 ----> Fail〈e〉 */
        oTableFree (&parents);
        return NULL;
      }
      continue;
    default:
      FATAL ("Unknown Instruction 0x%04x [%s]", pc->rator, OP_NAME (pc->rator));
    }
  }
#undef POP
#undef PUSH
#undef PARENT
}

Object *mRunFile (Machine *m, const char *grammar_file, const char *input_file)
{
  size_t grammar_size = 0, input_size = 0;
  Bytecode *grammar = NULL;
  char *input = NULL;
  Object *output = NULL;

  readFile (grammar_file, &grammar, &grammar_size);
  readFile (input_file, (uint8_t **) &input, &input_size);

  mLoad (m, grammar);
  output = mMatch (m, input, input_size);

  free (grammar);
  free (input);
  return output;
}
