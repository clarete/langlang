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

#include "vm.h"
#include "debug.h"

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
*/

/* -- Error control & report utilities -- */

/** Retrieve name of opcode */
#define OP_NAME(o) opNames[o]

typedef struct {
  const char *pos;
  long len;
} CaptureEntry;

/* Helps debugging */
static const char *opNames[OP_HALT] = {
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
  [OP_RETURN] = "OP_RETURN",
};

/* Set initial values for the machine */
void mInit (Machine *m)
{
  memset (m->stack, 0, STACK_SIZE * sizeof (void *));
  m->code = NULL;               /* Will be set by mLoad() */
}

void mFree (Machine *m)
{
  free (m->code);
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

  DEBUGLN ("   Run");

  while (true) {
    /* No-op if TEST isn't defined */
    DEBUG_INSTRUCTION_NEXT ();
    DEBUG_STACK ();

    switch (pc->rator) {
    case 0: return i;
    case OP_CHAR:
      DEBUGLN ("       OP_CHAR: `%c' == `%c' ? %d", *i,
               UOPERAND0 (pc), *i == UOPERAND0 (pc));
      if (*i == UOPERAND0 (pc)) { i++; pc++; }
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
    case OP_RETURN: {
      assert (sp > m->stack);
      pc = POP ()->pc;
      continue;
    }
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
      } else {
        /* 〈pc,i,e〉 ----> Fail〈e〉 */
        return NULL;
      }
      continue;
    default:
      FATAL ("Unknown Instruction 0x%04x [%s]", pc->rator, OP_NAME (pc->rator));
    }
  }

#undef PUSH
#undef POP
#undef THE_END
}
