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
  [OP_THROW] = "OP_THROW",
};

/* Set initial values for the machine */
void mInit (Machine *m)
{
  listInit (&m->symbols);
  m->stack = calloc (STACK_SIZE, sizeof (CaptureEntry));
  m->i = NULL;
  m->code = NULL;               /* Will be set by mLoad() */
}

/* Release the resources used by the machine */
void mFree (Machine *m)
{
  listFree (&m->symbols);
  free (m->code);
  free (m->stack);
  m->code = NULL;
  m->stack = NULL;
}

Object *mSymbol (Machine *m, const char *sym, size_t len) {
  uint32_t i;
  Object *symbol;

  for (i = 0; i < m->symbols.used; i++) {
    symbol = listItem (&m->symbols, i);
    if (SYMBOL (symbol)->len != len)
      continue;
    if (strncmp (SYMBOL (symbol)->name, sym, len) == 0)
      return symbol;
  }

  symbol = symbolNew (sym, len);
  listPush (&m->symbols, symbol);
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
  CDR (tmp) = consNew (i, OBJ (Nil));
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
uint32_t mMatch (Machine *m, const char *input, size_t input_size, Object **out)
{
  BacktrackEntry *sp = m->stack;
  Instruction *pc = m->code;
  const char *i = input;
  uint32_t btCount = 0, ltCount = 0;
  List treestk;
  const char *ffp = NULL;       /* Farther Failure Position */
  uint32_t label = PEG_SUCCESS; /* Error Label */

  /** Push data onto the machine's stack  */
#define PUSH(ii,pp) do { sp->i = ii; sp->pc = pp; sp++; } while (0)
  /** Pop data from the machine's stack. Notice it doesn't dereference
      the pointer, callers are supposed to do that when needed. */
#define POP() (--sp)
  /** The end of the input is the offset from the cursor to the end of
      the input string. */
#define THE_END (input + input_size)
  /** Update the cursor & keep track of FFP */
#define IPP() do { i++; if (i > ffp) ffp = i; } while (0)

  DEBUGLN ("   Run");

  listInit (&treestk);

  while (true) {
    /* No-op if DEBUG isn't defined */
    DEBUG_INSTRUCTION_NEXT ();
    DEBUG_STACK ();

    switch (pc->rator) {
    case OP_HALT:
    end:
      if (label > 1) {
        Symbol *lb = SYMBOL (listItem (&m->symbols, label-2));
        printf ("Match failed at pos %ld with label ",
                ffp - input + 1);
        objPrint (OBJ (lb));
        printf ("\n");
        return label;
      }
      /* We either didn't move the cursor at all or moved it and
       * backtracked on a failure */
      else if (!ffp && !i) {
        printf ("Match failed at pos 1\n");
        return PEG_FAILURE;
      } else if (ffp > i + 1) {
        printf ("Match failed at pos %ld\n", ffp - input + 1);
        return PEG_FAILURE;
      } else {
        /* Store final suffix upon successful match for testing
         * purposes. */
        m->i = i;
        /* Output captured objects */
        if (out && listLen (&treestk) > 0) {
          *out = listPop (&treestk);
          listFree (&treestk);
        }
        return label;
      }
    case OP_CAP_OPEN:
      /* printf ("OPEN[%c]: %s\n", UOPERAND1 (pc) ? 'T' : 'F', */
      /*         SYMBOL (listItem (&m->symbols, */
      /*                           UOPERAND2 (pc)))->name); */
      btCount++;
      if (UOPERAND1 (pc)) {     /* If the match is a terminal */
        listPush (&treestk, stringNew ("", 0));
      } else {                  /* If the match is a non-terminal */
        Object *node = consNew (listItem (&m->symbols, UOPERAND2 (pc)), OBJ (Nil));
        listPush (&treestk, node);
        ltCount = 0;
      }
      pc++;
      continue;
    case OP_CAP_CLOSE:
      /* printf ("CLOSE[%c]: %s\n", UOPERAND1 (pc) ? 'T' : 'F', */
      /*         SYMBOL (listItem (&m->symbols, */
      /*                           UOPERAND2 (pc)))->name); */
      if (btCount > 1) {
        append (listTop (&treestk), listPop (&treestk));
        btCount--;
        ltCount++;
      }
      pc++;
      continue;
    case OP_CAPCHAR:
      /* printf ("CAPCHAR\n"); */
      appendChar (listTop (&treestk), *(i-1));
      pc++;
      continue;
    case OP_CHAR:
      DEBUGLN ("       OP_CHAR: `%c' == `%c' ? %d", *i,
               UOPERAND0 (pc), *i == UOPERAND0 (pc));
      /* printf ("CHAR: `%c' == `%c'\n", */
      /*         *i == '\n' ? 'N' : *i, */
      /*         UOPERAND0 (pc) == '\n' ? 'N' : UOPERAND0 (pc)); */
      if (i < THE_END && *i == UOPERAND0 (pc)) { IPP (); pc++; }
      else goto fail;
      continue;
    case OP_ANY:
      DEBUGLN ("       OP_ANY: `%c' < |s| ? %d", *i, i < THE_END);
      /* printf ("ANY: %c\n", *i); */
      if (i < THE_END) { IPP (); pc++; }
      else goto fail;
      continue;
    case OP_SPAN:
      DEBUGLN ("       OP_SPAN: `%c' in [%c(%d)-%c(%d)]", *i,
               UOPERAND1 (pc), UOPERAND1 (pc),
               UOPERAND2 (pc), UOPERAND2 (pc));
      /* printf ("SPAN: %c\n", *i); */
      if (*i >= UOPERAND1 (pc) && *i <= UOPERAND2 (pc)) { IPP (); pc++; }
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
    case OP_THROW:
      label = UOPERAND0 (pc);
      goto end;
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

        /* printf (" FAIL[%u:%u-%u:%u]: %c\n", */
        /*         btCount, sp->btCount, */
        /*         ltCount, sp->ltCount, */
        /*         *i == '\n' ? 'N' : *i); */

        /* Clean capture from sequence */
        while (ltCount > sp->ltCount) {
          objFree (pop (listTop (&treestk)));
          ltCount--;
        }

        /* Clean capture stack in depth */
        while (btCount > sp->btCount) {
          objFree (listPop (&treestk));
          btCount--;
        }
      } else {
        /* 〈pc,i,e〉 ----> Fail〈e〉 */
        listFree (&treestk);
        return PEG_FAILURE;
      }
      continue;
    default:
      FATAL ("Unknown Instruction 0x%04x [%s]", pc->rator, OP_NAME (pc->rator));
    }
  }

#undef POP
#undef PUSH
}

void enclose (List *ot)
{
  Object *out = OBJ (Nil);

  while (CONSP (listTop (ot))) {
    out = consNew (listPop (ot), out);
  }
  while (!NILP (listTop (ot))) {
    out = consNew (listPop (ot), out);
  }
  /* POP the NIL value that marks the beginning of the list being
     enclosed */
  assert (NILP (listPop (ot)));
  listPush (ot, out);
}

Object *mMatchList (Machine *m, Object *input)
{
  BacktrackEntry *sp = m->stack;
  Instruction *pc = m->code;
  Object *l = input;
  List treestk;
  Symbol *sym;
  uint32_t btCount = 0, ltCount = 0;

  /** Push data onto the machine's stack  */
#define PUSH(ll,pp) do { sp->l = ll; sp->pc = pp; \
    sp->btCount = btCount;                        \
    sp->ltCount = ltCount;                        \
    sp++;                                         \
  } while (0)
  /** Pop data from the machine's stack. Notice it doesn't dereference
      the pointer, callers are supposed to do that when needed. */
#define POP() (--sp)

#define DEBUG_TREE() do {                                       \
    printf ("%u:%u,%u:%u\t\t%02u: [",                           \
            btCount, sp->btCount,                               \
            ltCount, sp->ltCount,                               \
            listLen (&treestk));                                \
    for (uint32_t i = listLen (&treestk); i > 0 ; i--) {        \
      objPrint (listItem (&treestk, i-1));                      \
      if (i > 1) printf (", ");                                 \
    }                                                           \
    printf ("]\n");                                             \
  } while (0)

  DEBUGLN ("   Run");

  listInit (&treestk);

  while (true) {
    /* No-op if DEBUG isn't defined */
    DEBUG_INSTRUCTION_NEXT ();
    DEBUG_STACK ();

    switch (pc->rator) {
    case OP_HALT:
      if (l) {
        if (listLen (&treestk) > 0) {
          Object *result = listPop (&treestk);
          /* listFree (&treestk); */
          return result;
        } else {
          return l;
        }
      } else {
        listFree (&treestk);
        return NULL;
      }
    case OP_OPEN:
      if (!CONSP (l) || !CONSP (CAR (l))) goto fail;
      PUSH (CDR (l), pc++); l = CAR (l);
      btCount++;
      listPush (&treestk, OBJ (Nil));
      continue;
    case OP_CLOSE:
      if (!NILP (l)) goto fail;
      enclose (&treestk);
      l = POP ()->l; pc++;
      ltCount = sp->ltCount;
      btCount = sp->btCount;
      continue;
    case OP_ATOM:
      /* Did the machine receive the right parameter? */
      sym = SYMBOL (listItem (&m->symbols, UOPERAND0 (pc)));
      if (!sym) goto fail;
      /* printf ("ATOM: `%s' == `%s'\t", */
      /*         sym->name, SYMBOL (CAR (l))->name); */
      /* Is it a valid subject? */
      if (!CONSP (l)) goto fail;
      if (CONSP (CAR (l))) goto fail;
      /* Does it match with the atom we're looking for? */
      if (strncmp (SYMBOL (CAR (l))->name, sym->name, sym->len)) goto fail;
      listPush (&treestk, CAR (l));
      ltCount++;
      /* Crank the machine to go to the next element & instruction */
      l = CDR (l); pc++;
      continue;
    case OP_ANY:
      if (!l || NILP (l)) goto fail;
      listPush (&treestk, CAR (l));
      ltCount++;
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

        while (ltCount > sp->ltCount) {
          listPop (&treestk);
          ltCount--;
        }
        while (btCount > sp->btCount) {
          listPop (&treestk);
          btCount--;
        }
      } else {
        /* 〈pc,i,e〉 ----> Fail〈e〉 */
        listFree (&treestk);
        return NULL;
      }
      continue;
    default:
      FATAL ("Unknown Instruction 0x%04x [%s]", pc->rator, OP_NAME (pc->rator));
    }
  }
#undef POP
#undef PUSH
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
  mMatch (m, input, input_size, &output);

  free (grammar);
  free (input);
  return output;
}
