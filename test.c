/* -*- coding: utf-8; -*-
 *
 * test.c - Implementation of Parsing Machine for PEGs
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
#include <stdio.h>
#include <string.h>

#include "debug.h"
#include "peg.h"

/* Utilities to write bytecode for testing */

#define GEN0(o)       (enc (gen0 (o)))
#define GEN1(o,a0)    (enc (gen1 (o, a0)))
#define GEN2(o,a0,a1) (enc (gen2 (o, a0, a1)))

static inline uint32_t gen0 (OpCode opc) {
  return opc << OPERATOR_OFFSET;
}

static inline uint32_t gen1 (OpCode opc, uint32_t arg0) {
  return (arg0 & 0x7ffffff)  | (opc << OPERATOR_OFFSET);
}

static inline uint32_t gen2 (OpCode opc, uint16_t arg0, uint16_t arg1) {
  return ((arg1 & 0xffff) | (arg0 << S2_OPERAND_SIZE) | opc << OPERATOR_OFFSET);
}

static inline uint32_t enc (uint32_t in)
{
  return
    ((in << 24) & 0xff000000) |
    ((in << 8 ) & 0x00ff0000) |
    ((in >> 8 ) & 0x0000ff00) |
    ((in >> 24) & 0x000000ff) ;
}

static inline size_t progSize (size_t nInstructions)
{
  /* An empty header contains two 16bit fields, one for the number of
     symbols within the header and the other one for the number of
     instructions within the code. */
  return 32 + (nInstructions * INSTRUCTION_SIZE);
}

static inline void writeHeader (uint32_t *prog, size_t nInstructions)
{
  union { uint32_t hh; uint16_t h[2]; } header;
  header.h[0] = 0;
  header.h[1] = nInstructions << 8;
  prog[0] = header.hh;
}

/* ---- TESTS ---- */

static void test_gen_args ()
{
  printf (" * gen args\n");
  printf ("     gen0arg (OP_ANY): 0x%02x\n", gen0 (OP_ANY));
  assert (gen0 (OP_ANY) == 0x10000000);
  printf ("     gen1arg (OP_CHAR, 'a'): 0x%02x\n", gen1 (OP_CHAR, 'a'));
  assert (gen1 (OP_CHAR, 'a') == 0x8000061);
  printf ("     gen1arg (OP_COMMIT, -2): 0x%02x\n", gen1 (OP_COMMIT, -2));
  assert (gen1 (OP_COMMIT, -2) == 0x27fffffe);
  printf ("     gen2arg (OP_SPAN, 'a', 'e'): 0x%02x\n", gen2 (OP_SPAN, 'a', 'e'));
  assert (gen2 (OP_SPAN, 'a', 'e') == 0x70610065);
  assert (gen2 (OP_CALL, 2, 1) == 0x60020001);
}

static void test_read_instr ()
{
  Instruction instr;
  uint32_t code;
  printf ("* read_instr\n");
  printf ("     gen0arg (OP_ANY): 0x%02x\n", gen0 (OP_ANY));

  assert (OP_MASK (gen0 (OP_ANY)) == OP_ANY);
  assert (OP_MASK (gen2 (OP_CALL, 2, 3)) == OP_CALL);
  assert (RN_MASK (gen0 (OP_ANY)) == 0);

  code = gen2 (OP_SPAN, 'a', 'e');
  instr.rator = OP_MASK (code);
  instr.u32 = RN_MASK (code);

  printf ("     gen2arg (OP_SPAN, 'a', 'e'): 0x%02x\n", code);
  assert (instr.u.r1 == 'a');
  assert (instr.u.r2 == 'e');
}

/*
  s[i] = ‘c’
  -------------------
  match ‘c’ s i = i+1 (ch.1)
*/
static void test_ch1 ()
{
  Machine m;
  int instructions = 2;
  uint32_t b[progSize (instructions)];
  const char *i = "a";
  DEBUGLN (" * t:ch.1");

  writeHeader (b, instructions);
  /* S <- 'a' */
  b[1] = GEN1 (OP_CHAR, 'a');   /* 0x0: Char 'a' */
  b[2] = GEN0 (OP_HALT);        /* 0x1: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, i, strlen (i), NULL) == 0);
  assert (m.i - i == 1);        /* Match */

  mFree (&m);
}

/*
  s[i] != ‘c’
  -------------------
  match ‘c’ s i = nil (ch.2)
 */
static void test_ch2 ()
{
  Machine m;
  int instructions = 2;
  uint32_t b[progSize (instructions)];
  DEBUGLN (" * t:ch.2");

  writeHeader (b, instructions);
  /* S <- 'a' */
  b[1] = GEN1 (OP_CHAR, 'a');   /* 0x0: Char 'a' */
  b[2] = GEN0 (OP_HALT);        /* 0x1: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, "x", 1, NULL) == 1); /* Failed */
  mFree (&m);
}

/*
  i ≤ |s|
  -----------------
  match . s i = i+1 (any.1)
*/
static void test_any1 ()
{
  Machine m;
  int instructions = 2;
  uint32_t b[progSize (instructions)];
  const char *i = "a";
  DEBUGLN (" * t:any.1");

  writeHeader (b, instructions);
  /* S <- . */
  b[1] = GEN0 (OP_ANY);         /* 0x0: Any */
  b[2] = GEN0 (OP_HALT);        /* 0x1: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);

  assert (mMatch (&m, i, strlen (i), NULL) == 0); /* Didn't fail */
  assert (m.i - i == 1);                          /* Match */

  mFree (&m);
}

/*
  i > |s|
  -----------------
  match . s i = nil (any.2)
*/
static void test_any2 ()
{
  Machine m;
  int instructions = 2;
  uint32_t b[progSize (instructions)];
  DEBUGLN (" * t:any.2");

  writeHeader (b, instructions);
  /* S <- . */
  b[1] = GEN0 (OP_ANY);         /* 0x0: Any */
  b[2] = GEN0 (OP_HALT);        /* 0x1: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, "", 0, NULL) > 0); /* Failed */
  mFree (&m);
}

/*
  match p s i = nil
  -----------------
  match !p s i = i (not.1)
*/
static void test_not1 ()
{
  Machine m;
  int instructions = 5;
  uint32_t b[progSize (instructions)];
  const char *i = "b";
  DEBUGLN (" * t:not.1");

  writeHeader (b, instructions);
  /* S <- !'a' */
  b[1] = GEN1 (OP_CHOICE, 4);   /* 0x0: Choice 0x4 */
  b[2] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_COMMIT, 1);   /* 0x2: Commit 1 */
  b[4] = GEN0 (OP_FAIL);        /* 0x3: Fail */
  b[5] = GEN0 (OP_HALT);        /* 0x4: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, i, strlen (i), NULL) == 0); /* Didn't fail */
  assert (m.i - i == 0);        /* But didn't match anything */

  mFree (&m);
}

void test_not1_fail_twice ()
{
  Machine m;
  int instructions = 4;
  uint32_t b[progSize (instructions)];
  const char *i = "b";
  DEBUGLN (" * t:not.1 (fail-twice)");

  writeHeader (b, instructions);
  /* S <- !'a' */
  b[1] = GEN1 (OP_CHOICE, 4);   /* 0x0: Choice 0x4 */
  b[2] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[3] = GEN0 (OP_FAIL_TWICE);  /* 0x2: FailTwice */
  b[4] = GEN0 (OP_HALT);        /* 0x3: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, i, strlen (i), NULL) == 0); /* Didn't fail */
  assert (m.i - i == 0);        /* But didn't match any char */

  mFree (&m);
}

/*
  match p s i = i+j
  ------------------
  match !p s i = nil (not.2)
*/
void test_not2 ()
{
  Machine m;
  int instructions = 5;
  uint32_t b[progSize (instructions)];
  DEBUGLN (" * t:not.2");

  writeHeader (b, instructions);
  /* S <- !'a' */
  b[1] = GEN1 (OP_CHOICE, 4);   /* 0x0: Choice 0x4 */
  b[2] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_COMMIT, 1);   /* 0x2: Commit 1 */
  b[4] = GEN0 (OP_FAIL);        /* 0x3: Fail */
  b[5] = GEN0 (OP_HALT);        /* 0x4: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, "a", 1, NULL) > 0); /* Failed */
  mFree (&m);
}

void test_not2_fail_twice ()
{
  Machine m;
  int instructions = 4;
  uint32_t b[progSize (instructions)];
  DEBUGLN (" * t:not.2 (fail-twice)");

  writeHeader (b, instructions);
  /* !'a' */
  b[1] = GEN1 (OP_CHOICE, 4);   /* 0x0: Choice 0x4 */
  b[2] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[3] = GEN0 (OP_FAIL_TWICE);  /* 0x2: FailTwice */
  b[4] = GEN0 (OP_HALT);        /* 0x3: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, "a", 1, NULL) > 0); /* Failed */
  mFree (&m);
}

/*
  match g p s i = i+j
  -------------------
  match g &p s i = i (and.1)
*/
void test_and1 ()
{
  Machine m;
  int instructions = 8;
  uint32_t b[progSize (instructions)];
  const char *i = "a";
  DEBUGLN (" * t:and.1");

  writeHeader (b, instructions);
  /* S <- &'a' */
  b[1] = GEN1 (OP_CHOICE, 7);   /* 0x0: Choice 0x7 */
  b[2] = GEN1 (OP_CHOICE, 4);   /* 0x1: Choice 0x4 */
  b[3] = GEN1 (OP_CHAR, 'a');   /* 0x2: Char 'a' */
  b[4] = GEN1 (OP_COMMIT, 1);   /* 0x3: Commit 1 */
  b[5] = GEN0 (OP_FAIL);        /* 0x4: Fail */
  b[6] = GEN1 (OP_COMMIT, 1);   /* 0x5: Commit 1 */
  b[7] = GEN0 (OP_FAIL);        /* 0x6: Fail */
  b[8] = GEN0 (OP_HALT);        /* 0x7: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, i, strlen (i), NULL) == 0); /* Didn't fail */
  assert (m.i - i == 0);        /* But didn't match anything */

  mFree (&m);
}

void test_and1_back_commit ()
{
  Machine m;
  int instructions = 5;
  uint32_t b[progSize (instructions)];
  const char *i = "a";
  DEBUGLN (" * t:and.1 (back-commit)");

  writeHeader (b, instructions);
  /* S <- &'a' */
  b[1] = GEN1 (OP_CHOICE, 3);      /* 0x0: Choice 0x3 */
  b[2] = GEN1 (OP_CHAR, 'a');      /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_BACK_COMMIT, 2); /* 0x2: BackCommit 2 */
  b[4] = GEN0 (OP_FAIL);           /* 0x3: Fail */
  b[5] = GEN0 (OP_HALT);           /* 0x4: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, i, strlen (i), NULL) == 0); /* Didn't fail */
  assert (m.i - i == 0);        /* But didn't match anything */

  mFree (&m);
}

/*
match g p s i = nil
--------------------
match g &p s i = nil (and.2)
*/
void test_and2 ()
{
  Machine m;
  int instructions = 8;
  uint32_t b[progSize (instructions)];
  DEBUGLN (" * t:and.2");

  writeHeader (b, instructions);
  /* &'a' */
  b[1] = GEN1 (OP_CHOICE, 7);   /* 0x0: Choice 0x7 */
  b[2] = GEN1 (OP_CHOICE, 4);   /* 0x1: Choice 0x4 */
  b[3] = GEN1 (OP_CHAR, 'a');   /* 0x2: Char 'a' */
  b[4] = GEN1 (OP_COMMIT, 1);   /* 0x3: Commit 1 */
  b[5] = GEN0 (OP_FAIL);        /* 0x4: Fail */
  b[6] = GEN1 (OP_COMMIT, 1);   /* 0x5: Commit 1 */
  b[7] = GEN0 (OP_FAIL);        /* 0x6: Fail */
  b[8] = GEN0 (OP_HALT);        /* 0x7: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, "b", 1, NULL) > 0); /* Failed */
  mFree (&m);
}

void test_and2_back_commit ()
{
  Machine m;
  int instructions = 5;
  uint32_t b[progSize (instructions)];
  DEBUGLN (" * t:and.2 (back-commit)");

  writeHeader (b, instructions);
  /* S <- &'a' */
  b[1] = GEN1 (OP_CHOICE, 3);      /* 0x0: Choice 0x3 */
  b[2] = GEN1 (OP_CHAR, 'a');      /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_BACK_COMMIT, 2); /* 0x2: BackCommit 2 */
  b[4] = GEN0 (OP_FAIL);           /* 0x3: Fail */
  b[5] = GEN0 (OP_HALT);           /* 0x4: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, "b", 1, NULL) > 0); /* Failed */
  mFree (&m);
}

/*
  match p1 s i = i+j    match p2 s i + j = i+j+k
  ----------------------------------------------
         match p1 p2 s i = i+j+k (con.1)
*/
void test_con1 ()
{
  Machine m;
  int instructions = 4;
  uint32_t b[progSize (instructions)];
  const char *i = "abc";
  DEBUGLN (" * t:con.1");

  writeHeader (b, instructions);
  /* 'a' . 'c' */
  b[1] = GEN1 (OP_CHAR, 'a');   /* 0x0: Char 'a' */
  b[2] = GEN0 (OP_ANY);         /* 0x1: Any */
  b[3] = GEN1 (OP_CHAR, 'c');   /* 0x2: Char 'c' */
  b[4] = GEN0 (OP_HALT);        /* 0x3: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, i, strlen (i), NULL) == 0); /* Didn't fail */
  assert (m.i - i == 3);        /* Matched all 3 chars */

  mFree (&m);
}

/*
  match p1 s i = i+j    match p2 s i + j = nil
  ----------------------------------------------
         match p1 p2 s i = nil (con.2)
 */
void test_con2 ()
{
  Machine m;
  int instructions = 4;
  uint32_t b[progSize (instructions)];
  DEBUGLN (" * t:con.2");

  writeHeader (b, instructions);
  /* S <- 'a' 'c' . */
  b[1] = GEN1 (OP_CHAR, 'a');   /* 0x0: Char 'a' */
  b[2] = GEN1 (OP_CHAR, 'c');   /* 0x1: Char 'c' */
  b[3] = GEN0 (OP_ANY);         /* 0x2: Any */
  b[4] = GEN0 (OP_HALT);        /* 0x3: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, "abc", 3, NULL) > 0); /* Failed */
  mFree (&m);
}

/*
  match p1 s i = nil
  ---------------------
  match p1 p2 s i = nil (con.3)
 */
void test_con3 ()
{
  Machine m;
  int instructions = 4;
  uint32_t b[progSize (instructions)];
  DEBUGLN (" * t:con.3");

  writeHeader (b, instructions);
  /* S <- 'a' 'c' . */
  b[1] = GEN1 (OP_CHAR, 'a');   /* 0x0: Char 'a' */
  b[2] = GEN1 (OP_CHAR, 'c');   /* 0x1: Char 'c' */
  b[3] = GEN0 (OP_ANY);         /* 0x2: Any */
  b[4] = GEN0 (OP_HALT);        /* 0x3: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, "cba", 3, NULL) > 0); /* Failed */
  mFree (&m);
}

/*
  match p1 s i = nil    match p2 s i = nil
  ----------------------------------------
      match p1 / p2 s i = nil  (ord.1)
 */
void test_ord1 ()
{
  Machine m;
  int instructions = 5;
  uint32_t b[progSize (instructions)];
  DEBUGLN (" * t:ord.1");

  writeHeader (b, instructions);
  /* 'a' / 'b' */
  b[1] = GEN1 (OP_CHOICE, 3);   /* 0x0: Choice 0x3 */
  b[2] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_COMMIT, 2);   /* 0x2: Commit 0x2 */
  b[4] = GEN1 (OP_CHAR, 'b');   /* 0x3: Char 'c' */
  b[5] = GEN0 (OP_HALT);        /* 0x4: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, "c", 1, NULL) > 0); /* Failed */
  mFree (&m);
}

/*
  match p1 s i = i+j
  -----------------------
  match p1 / p2 s i = i+j (ord.2)
 */
void test_ord2 ()
{
  Machine m;
  int instructions = 5;
  uint32_t b[progSize (instructions)];
  const char *i = "a";
  DEBUGLN (" * t:ord.2");

  writeHeader (b, instructions);
  /* 'a' / 'b' */
  b[1] = GEN1 (OP_CHOICE, 3);   /* 0x0: Choice 0x3 */
  b[2] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_COMMIT, 2);   /* 0x2: Commit 0x2 */
  b[4] = GEN1 (OP_CHAR, 'b');   /* 0x3: Char 'c' */
  b[5] = GEN0 (OP_HALT);        /* 0x4: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, i, strlen (i), NULL) == 0); /* Didn't fail */
  assert (m.i - i == 1);        /* Match the first char */

  mFree (&m);
}

/*
  match p1 s i = nil    match p2 s i = i+k
  ----------------------------------------
      match p1 / p2 s i = i+k (ord.3)
 */
void test_ord3 ()
{
  Machine m;
  int instructions = 5;
  uint32_t b[progSize (instructions)];
  const char *i = "b";
  DEBUGLN (" * t:ord.3");

  writeHeader (b, instructions);
  /* 'a' / 'b' */
  b[1] = GEN1 (OP_CHOICE, 3);   /* 0x0: Choice 0x3 */
  b[2] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_COMMIT, 2);   /* 0x2: Commit 0x2 */
  b[4] = GEN1 (OP_CHAR, 'b');   /* 0x3: Char 'c' */
  b[5] = GEN0 (OP_HALT);        /* 0x4: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, i, strlen (i), NULL) == 0); /* Didn't fail */
  assert (m.i - i == 1);        /* Match the first char */

  mFree (&m);
}

/*
  match p s i = i+j    match p∗ s i + j = i+j+k
  ----------------------------------------------
          match p∗ s i = i+j+k (rep.1)
*/
void test_rep1 ()
{
  Machine m;
  int instructions = 4;
  uint32_t b[progSize (instructions)];
  const char *i = "aab";
  DEBUGLN (" * t:rep.1");

  writeHeader (b, instructions);
  /* S <- 'a*' */
  b[1] = GEN1 (OP_CHOICE, 3);   /* 0x0: Choice 3 */
  b[2] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_COMMIT, -2);  /* 0x2: Commit -2 */
  b[4] = GEN0 (OP_HALT);        /* 0x3: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, i, strlen (i), NULL) == 0); /* Didn't fail */
  assert (m.i - i == 2);        /* Matched two chars */

  mFree (&m);
}

void test_rep1_partial_commit ()
{
  Machine m;
  int instructions = 4;
  uint32_t b[progSize (instructions)];
  const char *i = "aab";
  DEBUGLN (" * t:rep.1 (partial-commit)");

  writeHeader (b, instructions);
  /* S <- 'a*' */
  b[1] = GEN1 (OP_CHOICE, 3);          /* 0x0: Choice 3 */
  b[2] = GEN1 (OP_CHAR, 'a');          /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_PARTIAL_COMMIT, -1); /* 0x3: PartialCommit -1 */
  b[4] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, i, strlen (i), NULL) == 0); /* Didn't fail */
  assert (m.i - i == 2);        /* Matched two chars */

  mFree (&m);
}

/*
  match p s i = nil
  -----------------
  match p∗ s i = i (rep.2)
*/
void test_rep2 ()
{
  Machine m;
  int instructions = 4;
  uint32_t b[progSize (instructions)];
  const char *i = "b";
  DEBUGLN (" * t:rep.2");

  writeHeader (b, instructions);
  /* S <- 'a*' */
  b[1] = GEN1 (OP_CHOICE, 3);   /* 0x0: Choice 3 */
  b[2] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_COMMIT, -2);  /* 0x2: Commit -2 */
  b[4] = GEN0 (OP_HALT);        /* 0x3: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, i, strlen (i), NULL) == 0); /* Didn't fail */
  assert (m.i - i == 0);        /* But didn't match any char */

  mFree (&m);
}

void test_rep2_partial_commit ()
{
  Machine m;
  int instructions = 4;
  uint32_t b[progSize (instructions)];
  const char *i = "b";
  DEBUGLN (" * t:rep.2 (partial-commit)");

  writeHeader (b, instructions);
  /* S <- 'a*' */
  b[1] = GEN1 (OP_CHOICE, 3);          /* 0x0: Choice 3 */
  b[2] = GEN1 (OP_CHAR, 'a');          /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_PARTIAL_COMMIT, -1); /* 0x3: PartialCommit -1 */
  b[4] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, i, strlen (i), NULL) == 0); /* Didn't fail */
  assert (m.i - i == 0);        /* But didn't match any char */

  mFree (&m);
}

/*
  match g g(Ak) s i = i+j
  -----------------------
  match g Ak s i = i+j (var.1)
*/
void test_var1 ()
{
  Machine m;
  int instructions = 0xc;
  uint32_t b[progSize (instructions)];
  const char *i = "1+1";
  DEBUGLN (" * t:var.1");

  writeHeader (b, instructions);
  /* Start */
  b[0x1] = GEN1 (OP_CALL, 0x2); /* 0x0: Call 0x2 */
  b[0x2] = GEN1 (OP_JUMP, 0xb); /* 0x1: Jump 0xb */
  /* S <- D '+' D */
  b[0x3] = GEN1 (OP_CALL, 0x4); /* 0x2: Call 0x4 */
  b[0x4] = GEN1 (OP_CHAR, '+'); /* 0x3: Char '+' */
  b[0x5] = GEN1 (OP_CALL, 0x2); /* 0x4: Call 0x2 */
  b[0x6] = GEN0 (OP_RETURN);    /* 0x5: Return */
  /* D <- '0' / '1' */
  b[0x7] = GEN1 (OP_CHOICE, 3); /* 0x6: Choice 0x3 */
  b[0x8] = GEN1 (OP_CHAR, '0'); /* 0x7: Char '0' */
  b[0x9] = GEN1 (OP_COMMIT, 2); /* 0x8: Commit 0x2 */
  b[0xa] = GEN1 (OP_CHAR, '1'); /* 0x9: Char '1' */
  b[0xb] = GEN0 (OP_RETURN);    /* 0xa: Return */
  /* End */
  b[0xc] = GEN0 (OP_HALT);      /* 0xb: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, i, strlen (i), NULL) == 0); /* Didn't fail */
  assert (m.i - i == 3);        /* Matched the whole input */

  mFree (&m);
}

/*
  match g g(Ak) s i = nil
  -----------------------
  match g Ak s i = nil (var.2)
*/
void test_var2 ()
{
  Machine m;
  int instructions = 0xc;
  uint32_t b[progSize (instructions)];
  DEBUGLN (" * t:var.2");

  writeHeader (b, instructions);
  /* Start */
  b[0x1] = GEN1 (OP_CALL, 0x2); /* 0x0: Call 0x2 */
  b[0x2] = GEN1 (OP_JUMP, 0xb); /* 0x1: Jump 0xb */
  /* S <- D '+' D */
  b[0x3] = GEN1 (OP_CALL, 0x4); /* 0x2: Call 0x4 */
  b[0x4] = GEN1 (OP_CHAR, '+'); /* 0x3: Char '+' */
  b[0x5] = GEN1 (OP_CALL, 0x2); /* 0x4: Call 0x2 */
  b[0x6] = GEN0 (OP_RETURN);    /* 0x5: Return */
  /* D <- '0' / '1' */
  b[0x7] = GEN1 (OP_CHOICE, 3); /* 0x6: Choice 0x3 */
  b[0x8] = GEN1 (OP_CHAR, '0'); /* 0x7: Char '0' */
  b[0x9] = GEN1 (OP_COMMIT, 2); /* 0x8: Commit 0x2 */
  b[0xa] = GEN1 (OP_CHAR, '1'); /* 0x9: Char '1' */
  b[0xb] = GEN0 (OP_RETURN);    /* 0xa: Return */
  /* End */
  b[0xc] = GEN0 (OP_HALT);      /* 0xb: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, "1+2", 3, NULL) > 0); /* Failed */
  mFree (&m);
}

void test_span1 ()
{
  Machine m;
  int instructions = 4;
  uint32_t b[progSize (instructions)];
  const char *i = "abcdefgh";
  DEBUGLN (" * t:span.1");

  writeHeader (b, instructions);
  /* S <- '[a-e]*' */
  b[0x1] = GEN1 (OP_CHOICE, 3);      /* 0x0: Choice 0x3 */
  b[0x2] = GEN2 (OP_SPAN, 'a', 'e'); /* 0x1: Span 'a'-'e' */
  b[0x3] = GEN1 (OP_COMMIT, -2);     /* 0x2: Commit -0x2 */
  b[0x4] = GEN0 (OP_HALT);           /* 0x3: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, i, strlen (i), NULL) == 0); /* Didn't fail */
  assert (m.i - i == 5);                          /* Matched chars */

  mFree (&m);
}

/* ---- Captures ---- */

/* TODO: Commented out for now :( */
void test_cap1 ()
{
  Machine m;
  int instructions = 10;
  uint32_t b[progSize (instructions)];
  Value *out = NULL;
  const char *i = "a";
  DEBUGLN (" * t:cap.1");

  writeHeader (b, instructions);
  /* S <- %{ 'a' } */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x9);
  b[0x3] = GEN2 (OP_CAP_OPEN, 0x0, 0x0); /* CapOpen 0 (Main) */
  b[0x4] = GEN2 (OP_CAP_OPEN, 0x1, 0x1);
  b[0x5] = GEN1 (OP_CHAR, 0x61);
  b[0x6] = GEN0 (OP_CAPCHAR);
  b[0x7] = GEN2 (OP_CAP_CLOSE, 0x1, 0x1);
  b[0x8] = GEN2 (OP_CAP_CLOSE, 0x0, 0x0); /* CapClose 0 (Main) */
  b[0x9] = GEN0 (OP_RETURN);
  b[0xa] = GEN0 (OP_HALT);

  mInit (&m);
  mSymbol (&m, "Main", 4);
  mSymbol (&m, "Char", 4);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, i, strlen (i), &out) == 0); /* It worked */

  assert (out);

  valPrint (out);
  printf ("\n");

  assert (out);                 /* Isn't empty */
  assert (CONSP (out));         /* Is a list */
  /* assert (CONSP (CAR (out))); */
  /* assert (SYMBOLP (CAR (CAR (out))));   /\* Has an symbol within it *\/ */
  /* assert (strcmp (SYMBOL (CAR (CAR (out)))->name, "a") == 0); /\* Has the right value *\/ */

  valFree (out);
  mFree (&m);
}

/* ---- Error handling ---- */

static void test_error0 ()
{
  Machine m;
  int instructions = 12;
  uint32_t b[progSize (instructions)];
  const char *i = "c";
  DEBUGLN (" * t:err.0");

  writeHeader (b, instructions);
  /* S <- A / B; A <- 'a'; B <- 'b' */
  b[0x1] = GEN1 (OP_CALL, 0x2);   /* CALL L(S) */
  b[0x2] = GEN1 (OP_JUMP, 0xB);
  b[0x3] = GEN1 (OP_CHOICE, 0x3); /* L(S) = 0x3 */
  b[0x4] = GEN1 (OP_CALL, 0x4);   /* CALL L(A) */
  b[0x5] = GEN1 (OP_COMMIT, 0x2);
  b[0x6] = GEN1 (OP_CALL, 0x4);   /* CALL L(B) */
  b[0x7] = GEN0 (OP_RETURN);
  b[0x8] = GEN1 (OP_CHAR, 0x61);  /* 'a' LS(A) = 0x8 */
  b[0x9] = GEN0 (OP_RETURN);
  b[0xA] = GEN1 (OP_CHAR, 0x62);  /* 'b' LS(B) = 0xA */
  b[0xB] = GEN0 (OP_RETURN);
  b[0xC] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  assert (mMatch (&m, i, strlen (i), NULL) == 1);

  mFree (&m);
}

/* G[⇑l] */
static void test_throw ()
{
  Machine m;
  int instructions = 12;
  uint32_t b[progSize (instructions)];
  const char *i = "c";
  DEBUGLN (" * t:throw.0");

  writeHeader (b, instructions);

  b[0x1] = GEN1 (OP_CALL, 0x2);   /* CALL L(S) */
  b[0x2] = GEN1 (OP_JUMP, 0x4);
  b[0x3] = GEN1 (OP_THROW, 0x2);
  b[0x4] = GEN0 (OP_RETURN);
  b[0x5] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  mSymbol (&m, "test", 4);
  assert (mMatch (&m, i, strlen (i), NULL) == 2);

  mFree (&m);
}

/*
          PEG
   G[p] x ---> l
   -------------, l != fail (label,not.3)
           PEG
   G[!p] x ---> l
 */
static void test_label_not_3 ()
{
  Machine m;
  int instructions = 5;
  uint32_t b[progSize (instructions)];
  DEBUGLN (" * t:label.not.3.1");

  writeHeader (b, instructions);
  /* S <- ![⇑l] */
  b[1] = GEN1 (OP_CHOICE, 4);   /* 0x0: Choice 0x4 */
  b[2] = GEN1 (OP_THROW, 2);    /* 0x1: Throw 0x2 */
  b[3] = GEN1 (OP_COMMIT, 1);   /* 0x2: Commit 1 */
  b[4] = GEN0 (OP_FAIL);        /* 0x3: Fail */
  b[5] = GEN0 (OP_HALT);        /* 0x4: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b);
  mSymbol (&m, "error", 5);
  assert (mMatch (&m, "a", 1, NULL) == 2); /* Failed \w label 0x2 */
  assert (!m.i);                /* And didn't match anything */

  mFree (&m);
}


/* ---- List Matching ---- */

/*
         PEG
  G[ε] l ---> l (empty.1)
*/
static void test_lst_empty1 ()
{
  Machine m;
  int instructions = 4;
  uint32_t b[progSize (instructions)];
  Value *input = NULL;
  DEBUGLN (" * t:empty.1");

  writeHeader (b, instructions);
  /* *Empty* */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x4);
  b[0x3] = GEN0 (OP_RETURN);
  b[0x4] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b);

  input = consNew (mSymbol (&m, "a", 1), VAL (Nil));
  assert (mMatchList (&m, input) == input);

  valFree (input);
  mFree (&m);
}

/*
           PEG
  G[.] x:l ---> l (any.1)
*/
static void test_lst_any1 ()
{
  Machine m;
  int instructions = 5;
  uint32_t b[progSize (instructions)];
  Value *input, *output;
  DEBUGLN (" * t:lst.any.1");

  writeHeader (b, instructions);
  /* S <- . */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x4);
  b[0x3] = GEN0 (OP_ANY);
  b[0x4] = GEN0 (OP_RETURN);
  b[0x5] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b);

  input = consNew (mSymbol (&m, "a", 1), VAL (Nil));
  output = mMatchList (&m, input);
  assert (valEqual (output, mSymbol (&m, "a", 1)));

  valFree (input);
  mFree (&m);
}

/*
         PEG
  G[.] ε ---> fail (any.2)
*/
static void test_lst_any2 ()
{
  Machine m;
  int instructions = 5;
  uint32_t b[progSize (instructions)];
  DEBUGLN (" * t:lst.any.2");

  writeHeader (b, instructions);
  /* S <- . */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x4);
  b[0x3] = GEN0 (OP_ANY);
  b[0x4] = GEN0 (OP_RETURN);
  b[0x5] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b);

  assert (!mMatchList (&m, NULL));

  mFree (&m);
}

/*
           PEG
  G[a] a:l ---> l (term.1)
*/
static void test_lst_term1 ()
{
  Machine m;
  int instructions = 5;
  uint32_t b[progSize (instructions)];
  Value *input, *output;
  DEBUGLN (" * t:lst.term.1");

  writeHeader (b, instructions);
  /* S <- "MyTerm" */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x4);
  b[0x3] = GEN1 (OP_ATOM, 0x0);
  b[0x4] = GEN0 (OP_RETURN);
  b[0x5] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b);

  input = consNew (mSymbol (&m, "MyTerm", 6), VAL (Nil));
  output = mMatchList (&m, input);
  assert (valEqual (output, mSymbol (&m, "MyTerm", 6)));

  valFree (input);
  mFree (&m);
}

/*
           PEG
  G[a] b:l ---> fail (term.2)
*/
static void test_lst_term2 ()
{
  Machine m;
  int instructions = 5;
  uint32_t b[progSize (instructions)];
  Value *input;
  DEBUGLN (" * t:lst.term.2");

  writeHeader (b, instructions);
  /* S <- "aTerm" */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x4);
  b[0x3] = GEN1 (OP_ATOM, 0x0);
  b[0x4] = GEN0 (OP_RETURN);
  b[0x5] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b);

  /* Create the 0th term in the symbol table */
  mSymbol (&m, "aTerm", 5);

  /* Create the input with another symbol */
  input = consNew (mSymbol (&m, "myTerm", 6), VAL (Nil));
  assert (!mMatchList (&m, input));

  valFree (input);
  mFree (&m);
}

/*
         PEG
  G[a] ε ---> fail (term.3)
*/
static void test_lst_term3 ()
{
  Machine m;
  int instructions = 5;
  uint32_t b[progSize (instructions)];
  DEBUGLN (" * t:lst.term.3");

  writeHeader (b, instructions);
  /* S <- "aTerm" */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x4);
  b[0x3] = GEN1 (OP_ATOM, 0x0);
  b[0x4] = GEN0 (OP_RETURN);
  b[0x5] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b);

  /* Create the 0th term referenced in the bytecode grammar within the
   * symbol table */
  mSymbol (&m, "aTerm", 5);

  /* Run the above grammar on an empty input */
  assert (!mMatchList (&m, VAL (Nil)));

  /* Cleanup */
  mFree (&m);
}

/*
             PEG
  G[a] l1:l2 ---> fail (term.4)
*/
static void test_lst_term4 ()
{
  Machine m;
  int instructions = 5;
  uint32_t b[progSize (instructions)];
  Value *input;
  DEBUGLN (" * t:lst.term.4");

  writeHeader (b, instructions);
  /* S <- "aTerm" */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x4);
  b[0x3] = GEN1 (OP_ATOM, 0x0);
  b[0x4] = GEN0 (OP_RETURN);
  b[0x5] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b);

  /* Create the 0th term referenced in the bytecode grammar within the
   * symbol table */
  mSymbol (&m, "aTerm", 5);

  /* Run the above grammar on an input with a list within a list as
   * the first element */
  input = consNew (consNew (mSymbol (&m, "test", 4), VAL (Nil)), VAL (Nil));
  assert (!mMatchList (&m, input));

  /* Cleanup */
  valFree (input);
  mFree (&m);
}

/*
           PEG
   G[p] l1 ---> ε
--------------------- (list.1)
             PEG
G[{p}] l1:l2 ---> l2
*/
static void test_lst_list1 ()
{
  Machine m;
  int instructions = 7;
  uint32_t b[progSize (instructions)];
  Value *input, *output;
  DEBUGLN (" * t:lst.list.1");

  writeHeader (b, instructions);
  /* S <- { "test" } */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x6);
  b[0x3] = GEN0 (OP_OPEN);
  b[0x4] = GEN1 (OP_ATOM, 0x0);
  b[0x5] = GEN0 (OP_CLOSE);
  b[0x6] = GEN0 (OP_RETURN);
  b[0x7] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b);

  /* Create the 0th term referenced in the bytecode grammar within the
   * symbol table */
  mSymbol (&m, "test", 4);

  /* Run the above grammar on an input with "test" as the first (and
   * only) element of a list */
  input = consNew (consNew (mSymbol (&m, "test", 4), VAL (Nil)), VAL (Nil));
  output = mMatchList (&m, input);

  /* It does match! */
  assert (output);
  assert (valEqual (output, consNew (mSymbol (&m, "test", 4), VAL (Nil))));

  /* Cleanup */
  valFree (input);
  mFree (&m);
}

/*
           PEG
   G[p] l1 ---> X
----------------------, X != ε (list.2)
             PEG
G[{p}] l1:l2 ---> fail
  */
static void test_lst_list2 ()
{
  Machine m;
  int instructions = 7;
  uint32_t b[progSize (instructions)];
  Value *input, *output;
  DEBUGLN (" * t:lst.list.2");

  writeHeader (b, instructions);
  /* S <- { "foo" } */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x6);
  b[0x3] = GEN0 (OP_OPEN);
  b[0x4] = GEN1 (OP_ATOM, 0x0);
  b[0x5] = GEN0 (OP_CLOSE);
  b[0x6] = GEN0 (OP_RETURN);
  b[0x7] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b);

  /* Create the 0th term referenced in the bytecode grammar within the
   * symbol table */
  mSymbol (&m, "foo", 3);

  /* Run the above grammar on an input with "test" as the first (and
   * only) element of a list */
  input = consNew (consNew (mSymbol (&m, "test", 4), VAL (Nil)), VAL (Nil));
  output = mMatchList (&m, input);

  /* It DOES NOT match! */
  assert (!output);

  /* Cleanup */
  valFree (input);
  mFree (&m);
}

/*
             PEG
  G[{p}] a:l ---> fail (list.3)
*/
static void test_lst_list3 ()
{
  Machine m;
  int instructions = 7;
  uint32_t b[progSize (instructions)];
  Value *input, *output;
  DEBUGLN (" * t:lst.list.3");

  writeHeader (b, instructions);
  /* S <- { "foo" } */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x6);
  b[0x3] = GEN0 (OP_OPEN);
  b[0x4] = GEN1 (OP_ATOM, 0x0);
  b[0x5] = GEN0 (OP_CLOSE);
  b[0x6] = GEN0 (OP_RETURN);
  b[0x7] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b);

  /* Create the 0th term referenced in the bytecode grammar within the
   * symbol table */
  mSymbol (&m, "foo", 3);

  /* Run the above grammar on an input with "test" as the first (and
   * only) element of a list */
  input = consNew (mSymbol (&m, "test", 4), VAL (Nil));
  output = mMatchList (&m, input);

  /* It DOES NOT match! */
  assert (!output);

  /* Cleanup */
  valFree (input);
  mFree (&m);
}

/*
           PEG
  G[{p}] ε ---> fail
*/
static void test_lst_list4 ()
{
  Machine m;
  int instructions = 7;
  uint32_t b[progSize (instructions)];
  Value *output;
  DEBUGLN (" * t:lst.list.4");

  writeHeader (b, instructions);
  /* S <- { "foo" } */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x6);
  b[0x3] = GEN0 (OP_OPEN);
  b[0x4] = GEN1 (OP_ATOM, 0x0);
  b[0x5] = GEN0 (OP_CLOSE);
  b[0x6] = GEN0 (OP_RETURN);
  b[0x7] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b);

  /* Create the 0th term referenced in the bytecode grammar within the
   * symbol table */
  mSymbol (&m, "foo", 3);

  /* Run the above grammar on a empty input */
  output = mMatchList (&m, VAL (Nil));

  /* It DOES NOT match! */
  assert (!output);

  /* Cleanup */
  mFree (&m);
}

void test_val_equal_int ()
{
  Value *o1 = intNew (2);
  Value *o2 = intNew (3);
  Value *o3 = intNew (2);

  assert (!valEqual (o1, o2));
  assert (!valEqual (o2, o3));
  assert (valEqual (o1, o3));

  valFree (o1);
  valFree (o2);
  valFree (o3);
}

void test_val_equal_symbol ()
{
  Machine m;
  Value *o1, *o2, *o3;

  mInit (&m);
  o1 = mSymbol (&m, "Hi!", 3);
  o2 = mSymbol (&m, "Oi!", 3);
  o3 = mSymbol (&m, "Hi!", 3);

  assert (!valEqual (o1, o2));
  assert (!valEqual (o2, o3));
  assert (valEqual (o1, o3));

  mFree (&m);
}

void test_val_equal_bool ()
{
  assert (True == True);
  assert (False == False);
  assert (True != False);
  assert (valEqual (True, True));
  assert (valEqual (False, False));
  assert (!valEqual (True, False));
}

void test_val_equal_cons ()
{
  Machine m;
  Value *o1, *o2, *o3;

  mInit (&m);
  o1 = consNew (mSymbol (&m, "Hi!", 3), consNew (intNew (5), VAL (Nil)));
  o2 = consNew (mSymbol (&m, "Hi!", 3), intNew (5));
  o3 = consNew (mSymbol (&m, "Hi!", 3), consNew (intNew (5), VAL (Nil)));

  assert (!valEqual (o1, o2));
  assert (!valEqual (o2, o3));
  assert (valEqual (o1, o3));

  valFree (o1);
  valFree (o2);
  valFree (o3);
  mFree (&m);
}

void test_val_equal_list ()
{
  List arr1, arr2, arr3;

  listInit (&arr1);
  listInit (&arr2);
  listInit (&arr3);

  listPush (&arr1, intNew (1));
  listPush (&arr1, intNew (2));
  listPush (&arr1, intNew (3));

  listPush (&arr2, intNew (3));
  listPush (&arr2, intNew (2));
  listPush (&arr2, intNew (1));

  listPush (&arr3, intNew (1));
  listPush (&arr3, intNew (2));
  listPush (&arr3, intNew (3));

  assert (!valEqual (VAL (&arr1), VAL (&arr2)));
  assert (!valEqual (VAL (&arr2), VAL (&arr3)));
  assert (valEqual (VAL (&arr1), VAL (&arr3)));
}

void test_val_equal_sublist ()
{
  List arr1, arr2;
  List sarr1, sarr2;
  List sarr3, sarr4;
  int i;

  listInit (&arr1); listInit (&arr2);
  listInit (&sarr1); listInit (&sarr2);
  listInit (&sarr3); listInit (&sarr4);

  for (i = 1; i < 4; i++) {
    listPush (&sarr1, intNew (i));
    listPush (&sarr2, intNew (i));
    listPush (&sarr3, intNew (i*2));
    listPush (&sarr4, intNew (i*2));
  }

  listPush (&arr1, VAL (&sarr1));
  listPush (&arr2, VAL (&sarr2));
  listPush (&arr1, VAL (&sarr3));
  listPush (&arr2, VAL (&sarr4));

  assert (valEqual (VAL (&arr1), VAL (&arr2)));

  listPush (&arr1, intNew (99));
  listPush (&arr2, VAL (True));
  listPush (&arr2, VAL (False));
  listPush (&arr2, VAL (Nil));

  assert (!valEqual (VAL (&arr1), VAL (&arr2)));

  printf ("The lists: ");
  valPrint (consNew (VAL (&arr1), VAL (&arr2)));
  printf ("\n");
}

void test_dict0 ()
{
  Dict d0;
  Value *output;

  dictInit (&d0);

  /* Can't delete anything from an empty dict */
  assert (!dictDel (&d0, stringNew ("seven", 5)));

  /* Add a bunch of entries */
  assert (dictSet (&d0, stringNew ("one", 3), intNew (1)));
  assert (dictSet (&d0, stringNew ("two", 3), intNew (2)));
  assert (dictSet (&d0, stringNew ("three", 5), intNew (3)));
  assert (dictSet (&d0, stringNew ("four", 4), intNew (4)));
  assert (dictSet (&d0, stringNew ("five", 4), intNew (5)));
  assert (dictSet (&d0, stringNew ("six", 3), intNew (6)));
  assert (dictSet (&d0, stringNew ("seven", 5), intNew (7)));
  assert (dictSet (&d0, stringNew ("eight", 5), intNew (8)));
  assert (dictSet (&d0, stringNew ("nine", 4), intNew (9)));
  assert (dictSet (&d0, stringNew ("ten", 3), intNew (10)));

  /* Make sure the newly inserted all there */
  assert (dictLen (&d0) == 10);

  /* Try to insert key `eight' again, shouldn't have done anything */
  assert (!dictSet (&d0, stringNew ("eight", 5), intNew (8)));
  assert (dictLen (&d0) == 10);

  /* Found key `two' */
  assert (dictGet (&d0, stringNew ("two", 3), &output));
  assert (valEqual (output, intNew (2)));
  output = NULL;

  /* Found key `seven' */
  assert (dictGet (&d0, stringNew ("seven", 5), &output));
  assert (valEqual (output, intNew (7)));
  output = NULL;

  /* Can't find key `blah' */
  assert (!dictGet (&d0, stringNew ("blah", 4), &output));
  assert (!output);

  /* It can't delete keys that it can't find */
  assert (!dictDel (&d0, stringNew ("bleh", 4)));

  /* Delete a few keys */
  assert (dictDel (&d0, stringNew ("seven", 5)));
  assert (dictDel (&d0, stringNew ("six", 3)));
  assert (dictDel (&d0, stringNew ("three", 5)));
  assert (dictLen (&d0) == 7);

  printf ("The dict: ");
  valPrint (VAL (&d0));
  printf ("\n");
}

int main ()
{
  test_gen_args ();
  test_read_instr ();
  test_ch1 ();
  test_ch2 ();
  test_any1 ();
  test_any2 ();
  test_not1 ();
  test_not1_fail_twice ();
  test_not2 ();
  test_not2_fail_twice ();
  test_and1 ();
  test_and1_back_commit ();
  test_and2 ();
  test_and2_back_commit ();
  test_con1 ();
  test_con2 ();
  test_con3 ();
  test_ord1 ();
  test_ord2 ();
  test_ord3 ();
  test_rep1 ();
  test_rep1_partial_commit ();
  test_rep2 ();
  test_rep2_partial_commit ();
  test_var1 ();
  test_var2 ();
  test_span1 ();

  test_error0 ();
  test_throw ();
  test_label_not_3 ();

  test_cap1 ();

  test_lst_empty1 ();
  test_lst_any1 ();
  test_lst_any2 ();
  /* test_lst_var1 (); */
  test_lst_term1 ();
  test_lst_term2 ();
  test_lst_term3 ();
  test_lst_term4 ();

  /* test_lst_con1 (); */
  /* test_lst_con2 (); */
  /* test_lst_ord1 (); */
  /* test_lst_ord2 (); */
  /* test_lst_not1 (); */
  /* test_lst_not2 (); */
  /* test_lst_rep1 (); */
  /* test_lst_rep2 (); */

  test_lst_list1 ();
  test_lst_list2 ();
  test_lst_list3 ();
  test_lst_list4 ();

  test_val_equal_int ();
  test_val_equal_symbol ();
  test_val_equal_bool ();
  test_val_equal_cons ();
  test_val_equal_list ();
  test_val_equal_sublist ();

  test_dict0 ();
  return 0;
}
