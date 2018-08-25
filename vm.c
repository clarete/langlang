/* -*- coding: utf-8; -*-
 *
 * vm.c - Implementation of Parsing Machine for PEGs
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
#include <stdint.h>
#include <stdio.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>
#include <assert.h>

#include "debug.h"

#include "vm.h"
#include "value.h"

/*

Basic Functionality
===================

Matching
 * [x] Patterns
 * [x] Expressions
Extraction
 * [ ] Lists

Semantic Operations
===================

Character
 * [x] 'c'
 * [x] . (Any)
Concatenation
 * [x] a b
Ordered Choice
 * [x] a / b
Predicates
 * [x] !a
 * [x] &a (syntax suggar)
Suffixes
 * [x] a*
 * [ ] a? (syntax suggar)

Machine Instructions
====================

* [x] Char c
* [x] Any
* [x] Choice l
* [x] Commit l
* [x] Fail
* [x] FailTwice l
* [x] PartialCommit
* [x] BackCommit l
* [ ] TestChar l o
* [ ] TestAny n o
* [x] Jump l
* [x] Call l
* [x] Return
* [ ] Set
* [x] Span
* [x] CapOpen
* [x] CapClose

Bytecode Format
===============

Patterns get compiled to Bytecode objects. Bytecode objects are
sequences of instructions. Each instruction is 32 bits long.

      32bits   32bits   32bits
    -----|--------|--------|----
    | Instr1 | Instr2 | InstrN |
    ----------------------------

Instruction Format
==================

Each instruction is 32 bit long. The first 6 bits are reserved for the
opcode and the other 26 bits store parameters for the instruction.  We
have instructions that take 0, 1 or 2 parameters. Since there are only
6 bits for instructions, we can have at most 63 of them.

The utility `OP_MASK()' can be used to read the OPCODE from a 32bit
instruction data. Each argument size introduces different functions.
They're Here are the types of arguments:

Instruction with 1 parameter (Eg.: Char x)
------------------------------------------
    opcode    |Parameter #1
    ----------|------------------------------------------------------
    |0|0|0|0|1|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|1|1|0|0|0|0|1|
    ----------|------------------------------------------------------
    [   0 - 5 |                                              6 - 32 ]
    [       5 |                                                  27 ]

    * soperand() Read signed value
    * uoperand() Read unsigned value

Instruction with 2 parameters (Eg.: TestChar 4 97)
-------------------------------------------------
    opcode    | Parameter #1        | Parameter #2
    ----------|---------------------|--------------------------------
    |0|0|0|0|1|0|0|0|0|0|0|0|0|0|0|1|0|0|0|0|0|0|0|0|0|1|1|0|0|0|0|1|
    ----------|---------------------|--------------------------------
    [   0 - 5 |              6 - 16 |                       17 - 32 ]
    [       5 |                  11 |                            16 ]

    * s1operand() Read first operand as signed value
    * u1operand() Read first operand as unsigned values
    * s2operand() Read second operand as signed value
    * u2operand() Read second operand as unsigned value

Other Design Choices
====================

The core of this module tries to follow as close as possible the
design established by "A Parsing Machine for PEGs" (R. Ierusalimschy &
S. Medeiros -- 2008). However, the design doesn't really specify the
implementation details of features that aren't related to matching the
input, like the implementation of SET, SPAN and how to capture the
matches.

Features that open must provide well defined functionality but don't
have an implementation specification are documented in this section.

SPAN
----

  This is way to allow the implementation to match a range of
  characters like =a-z= instead of just expanding it to an ordered
  choice that is as big as the range E.g.: =a / b / ... / z=.

  The =OP_SPAN a b= instruction is the current way of implementing the
  semantics of =SPAN=. It takes two arguments =a= and =b= and compare
  them to the next char in the input stream =i= as follows:

    =(a >\= i) && (i <\= b)=

  This improves on the simplest case of a single range. To represent
  classes with multiple ranges e.g.: =[a-zA-Z]= the compiler currently
  has to produce an ordered choice with one choice per range.

SET
---

  This feature isn't implemented yet. It's probably going to be
  implemented as a new instruction =OP_SET l= where =l= is the
  location of a set of chars stored in a soon to be implemented string
  table.

Captures
--------

  Besides being able to tell if an input matches PEG, the parsing
  machine should also be able to extract the matched values as a tree
  where the nodes are tagged with the pieces of the grammar they
  matched.

  Two instructions were added in order to support this feature:
  =OP_CAP_OPEN t l= and =OP_CAP_CLOSE t l=. In both instructions, =t=
  is a boolean flag where false means the capture is non terminal and
  true means it's a terminal. And =l= is the location of the
  identifier of the capture in the soon to be implemented string
  table.

  The =OP_CAP_{OPEN,CLOSE}= instructions are supposed to be used in
  tandem and the compiler must generate capture pairs accounting for
  the execution model of the VM where some instructions (e.g.: Fail,
  Call, Jump, Return) can move the program counter of the virtual
  machine in non linear ways.

  To achieve the above goal, each semantic operation establishes their
  own capture rules. The predicates are the easiest ones, they never
  capture any values as they're only boolean match operators and don't
  really move the input cursor.

  Sequences require one pair of capture instructions around the whole
  set of expressions. Then each item of the expression will have their
  own capture rules applied.

  If a program doesn't match the input, the machine will get into the
  Fail state and the program counter will backtrack before the
  =OP_CAP_CLOSE= instruction can be executed. Which leads to a
  dangling =OP_CAP_OPEN= on top of the capture stack. To avoid that
  problem, the field =cap= was added to the =BacktrackEntry= struct,
  which is the format of stack entries. When the machine gets into
  fail state for backtracking, it also restores the top of the capture
  stack.
*/

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
};

/* Set initial values for the machine */
void mInit (Machine *m)
{
  m->stack = calloc (STACK_SIZE, sizeof (CaptureEntry *));
  m->captures = calloc (STACK_SIZE, sizeof (CaptureEntry *));
  /* memset (m->stack, 0, STACK_SIZE * sizeof (void *)); */
  /* memset (m->captures, 0, STACK_SIZE * sizeof (void *)); */
  m->code = NULL;               /* Will be set by mLoad() */
  m->cap = m->captures;
}

void mFree (Machine *m)
{
  free (m->code);
  m->cap = NULL;
  m->code = NULL;
}

void mLoad (Machine *m, Bytecode *code, size_t code_size)
{
  Instruction *tmp;
  uint32_t instr, i;

  /* Code size is in uint8_t and each instruction is 16bits */
  DEBUGLN ("   Load");
  if ((tmp = m->code = calloc (sizeof (Instruction), code_size / 4 + 2)) == NULL)
    FATAL ("Can't allocate %s", "memory");
  for (i = 0; i < code_size; i += 4) {
    instr  = *code++ << 24;
    instr |= *code++ << 16;
    instr |= *code++ << 8;
    instr |= *code++;

    tmp->rator = OP_MASK (instr);
    tmp->rand = instr;          /* Use SOPERAND* to access this */
    DEBUG_INSTRUCTION_LOAD ();
    tmp++;
  }
}

/* Run the matching machine */
const char *mMatch (Machine *m, const char *input, size_t input_size)
{
  BacktrackEntry *sp = m->stack;
  Instruction *pc = m->code;
  const char *i = input;

  /** Push data onto the machine's stack  */
#define PUSH(ii,pp) do { sp->i = ii; sp->pc = pp; sp++; } while (0)
  /** Pop data from the machine's stack. Notice it doesn't dereference
      the pointer, callers are supposed to do that when needed. */
#define POP() (--sp)
  /** The end of the input is the offset from the cursor to the end of
      the input string. */
#define THE_END (input + input_size)

  /** Push data to the capture buffer  */
#define PUSH_CAP(_p,_ty,_id,_tr) do {                                   \
    m->cap->pos = _p;                                                 \
    m->cap->type = _ty;                                               \
    m->cap->term = _id;                                               \
    m->cap->idx = _tr;                                                \
    m->cap++;                                                         \
  } while (0)

  DEBUGLN ("   Run");

  while (true) {
    /* No-op if TEST isn't defined */
    DEBUG_INSTRUCTION_NEXT ();
    DEBUG_STACK ();

    switch (pc->rator) {
    case 0: return i;
    case OP_CAP_OPEN:
      PUSH_CAP (i, CapOpen, UOPERAND1 (pc), UOPERAND2 (pc));
      pc++;
      continue;
    case OP_CAP_CLOSE:
      PUSH_CAP (i, CapClose, UOPERAND0 (pc), UOPERAND2 (pc));
      pc++;
      continue;
    case OP_CHAR:
      DEBUGLN ("       OP_CHAR: `%c' == `%c' ? %d", *i,
               UOPERAND0 (pc), *i == UOPERAND0 (pc));
      if (i < THE_END && *i == UOPERAND0 (pc)) { i++; pc++; }
      else goto fail;
      continue;
    case OP_ANY:
      DEBUGLN ("       OP_ANY: `%c' < |s| ? %d", *i, i < THE_END);
      if (i < THE_END) { i++; pc++; }
      else goto fail;
      continue;
    case OP_SPAN:
      DEBUGLN ("       OP_SPAN: `%c' in [%c(%d)-%c(%d)]", *i,
               UOPERAND1 (pc), UOPERAND1 (pc),
               UOPERAND2 (pc), UOPERAND2 (pc));
      if (*i >= UOPERAND1 (pc) && *i <= UOPERAND2 (pc)) { i++; pc++; }
      else goto fail;
      continue;
    case OP_CHOICE:
      sp->cap = m->cap;
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
      /* No-op if TEST isn't defined */
      DEBUG_FAILSTATE ();

      if (sp > m->stack) {
        /* Fail〈(pc,i1):e〉 ----> 〈pc,i1,e〉 */
        do i = POP ()->i;
        while (i == NULL && sp > m->stack);
        pc = sp->pc;            /* Restore the program counter */
        /* Non-Terminals can't produce errors */
        m->cap = sp->cap;
      } else {
        /* 〈pc,i,e〉 ----> Fail〈e〉 */
        return NULL;
      }
      continue;
    default:
      FATAL ("Unknown Instruction 0x%04x [%s]", pc->rator, OP_NAME (pc->rator));
    }
  }
}

const char *cap_type[2] = { " Open", "Close" };

void printCaptures (Machine *m)
{
  CaptureEntry match, *cp = m->cap;

  (void) match;                 /* In case TEST isn't set */

  while (cp > m->captures) {
    match = *--cp;              /* POP () */
    DEBUGLN ("     CAP: %s[%d]", cap_type[match.type], match.idx);
  }
}

Object *mExtract (Machine *m, const char *input)
{
  uint16_t start, end;
  CaptureEntry close, match, *stack;
  CaptureEntry *cp, *sp;
  Object *item = NULL, *out = NULL; /* Output list to be filled in bottom-up */

  printCaptures (m);

  stack = calloc (STACK_SIZE, sizeof (CaptureEntry *));

  sp = stack;
  cp = m->cap;

  DEBUGLN ("  Extract: %p %p", (void*) cp, (void*) m->captures);

  while (cp > m->captures) {
    match = *--cp;              /* POP () */

    DEBUGLN ("     MATCH: %s[%d]", cap_type[match.type], match.idx);

    if (match.type == CapClose) {
      *sp++ = match;
      continue;
    }

    /* Pop the last entry from capture stack */
    close = *--sp;

    if (match.idx != close.idx) {
      DEBUGLN ("Closing on the wrong capture %d:%d", close.idx, match.idx);
    }

    char key[256];
    sprintf (key, "%d", match.idx);

    if (match.term) {
      /* Terminal */
      start = match.pos - input;
      end = close.pos - input;
      item = makeAtom (input + start, end - start);
      out = makeCons (makeCons (makeAtom (key, strlen (key)), item), out);
    } else {
      /* Non-Terminal */
      out = makeCons (makeCons (makeAtom (key, strlen (key)), out), NULL);
    }

    printObj (out);
    printf ("\n\n");
  }

  return out;
}
