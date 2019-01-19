/* -*- coding: utf-8; -*-
 *
 * test.c - Implementation of Parsing Machine for PEGs
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
  return ((arg1 & 0x7ffffff) | (arg0 << S2_OPERAND_SIZE) | opc << OPERATOR_OFFSET);
}

static inline uint32_t enc (uint32_t in)
{
  return
    ((in << 24) & 0xff000000) |
    ((in << 8 ) & 0x00ff0000) |
    ((in >> 8 ) & 0x0000ff00) |
    ((in >> 24) & 0x000000ff) ;
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
}

/*
  s[i] = ‘c’
  -------------------
  match ‘c’ s i = i+1 (ch.1)
*/
static void test_ch1 ()
{
  Machine m;
  uint32_t b[6];
  const char *i = "a";
  const char *o = NULL;
  DEBUGLN (" * t:ch.1");

  b[0] = enc (2*2);             /* Code Size */
  /* Start <- 'a' */
  b[1] = GEN1 (OP_CHAR, 'a');   /* 0x0: Char 'a' */
  b[2] = 0;                     /* 0x1: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  o = mMatch (&m, i, strlen (i));
  mFree (&m);

  assert (o);
  assert (o - i == 1);        /* Match */
}

/*
  s[i] != ‘c’
  -------------------
  match ‘c’ s i = nil (ch.2)
 */
static void test_ch2 ()
{
  Machine m;
  uint32_t b[3];
  DEBUGLN (" * t:ch.2");

  b[0] = enc (2*2);             /* Code Size */
  /* Start <- 'a' */
  b[1] = GEN1 (OP_CHAR, 'a');   /* 0x0: Char 'a' */
  b[2] = 0;                     /* 0x1: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  assert (!mMatch (&m, "x", 1)); /* Failed */
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
  uint32_t b[3];
  const char *i = "a";
  const char *o = NULL;
  DEBUGLN (" * t:any.1");

  b[0] = enc (2*2);             /* Code Size */
  /* . */
  b[1] = GEN0 (OP_ANY);         /* 0x0: Any */
  b[2] = 0;                     /* 0x1: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  o = mMatch (&m, i, strlen (i));
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - i == 1);          /* Match */
}

/*
  i > |s|
  -----------------
  match . s i = nil (any.2)
*/
static void test_any2 ()
{
  Machine m;
  uint32_t b[3];
  DEBUGLN (" * t:any.2");

  b[0] = enc (2*2);             /* Code Size */
  /* . */
  b[1] = GEN0 (OP_ANY);         /* 0x0: Any */
  b[2] = 0;                     /* 0x1: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  assert (!mMatch (&m, "", 0)); /* Failed */
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
  uint32_t b[6];
  const char *i = "b";
  const char *o = NULL;
  DEBUGLN (" * t:not.1");

  b[0] = enc (5*4);             /* Code Size */
  /* !'a' */
  b[1] = GEN1 (OP_CHOICE, 4);   /* 0x0: Choice 0x4 */
  b[2] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_COMMIT, 1);   /* 0x2: Commit 1 */
  b[4] = GEN0 (OP_FAIL);        /* 0x3: Fail */
  b[5] = 0;                     /* 0x4: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  o = mMatch (&m, i, strlen (i));
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - i == 0);        /* But didn't match anything */
}

void test_not1_fail_twice ()
{
  Machine m;
  uint32_t b[5];
  const char *i = "b";
  const char *o = NULL;
  DEBUGLN (" * t:not.1 (fail-twice)");

  b[0] = enc (4*4);             /* Code Size */
  /* !'a' */
  b[1] = GEN1 (OP_CHOICE, 4);   /* 0x0: Choice 0x4 */
  b[2] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[3] = GEN0 (OP_FAIL_TWICE);  /* 0x2: FailTwice */
  b[4] = GEN0 (OP_HALT);        /* 0x3: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  o = mMatch (&m, i, strlen (i));
  mFree (&m);

  assert (o);                   /* Did not fail */
  assert (o - i == 0);        /* But didn't match any char */
}

/*
  match p s i = i+j
  ------------------
  match !p s i = nil (not.2)
*/
void test_not2 ()
{
  Machine m;
  uint32_t b[6];
  DEBUGLN (" * t:not.2");

  b[0] = enc (5*4);             /* Code Size */
  /* !'a' */
  b[1] = GEN1 (OP_CHOICE, 4);   /* 0x0: Choice 0x4 */
  b[2] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_COMMIT, 1);   /* 0x2: Commit 1 */
  b[4] = GEN0 (OP_FAIL);        /* 0x3: Fail */
  b[5] = GEN0 (OP_HALT);        /* 0x4: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  assert (!mMatch (&m, "a", 1)); /* Failed */
  mFree (&m);
}

void test_not2_fail_twice ()
{
  Machine m;
  uint32_t b[5];
  DEBUGLN (" * t:not.2 (fail-twice)");

  b[0] = enc (4*4);             /* Code Size */
  /* !'a' */
  b[1] = GEN1 (OP_CHOICE, 4);   /* 0x0: Choice 0x4 */
  b[2] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[3] = GEN0 (OP_FAIL_TWICE);  /* 0x2: FailTwice */
  b[4] = GEN0 (OP_HALT);        /* 0x3: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  assert (!mMatch (&m, "a", 1)); /* Failed */
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
  uint32_t b[9];
  const char *i = "a";
  const char *o = NULL;
  DEBUGLN (" * t:and.1");

  b[0] = enc (8*4);             /* Code Size */
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
  mLoad (&m, (Bytecode *) b, sizeof (b));
  o = mMatch (&m, i, strlen (i));
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - i == 0);          /* But didn't match anything */
}

void test_and1_back_commit ()
{
  Machine m;
  uint32_t b[6];
  const char *i = "a";
  const char *o = NULL;
  DEBUGLN (" * t:and.1 (back-commit)");

  b[0] = enc (5*4);                /* Code Size */
  /* &'a' */
  b[1] = GEN1 (OP_CHOICE, 3);      /* 0x0: Choice 0x3 */
  b[2] = GEN1 (OP_CHAR, 'a');      /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_BACK_COMMIT, 2); /* 0x2: BackCommit 2 */
  b[4] = GEN0 (OP_FAIL);           /* 0x3: Fail */
  b[5] = GEN0 (OP_HALT);           /* 0x4: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  o = mMatch (&m, i, strlen (i));
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - i == 0);          /* But didn't match anything */
}

/*
match g p s i = nil
--------------------
match g &p s i = nil (and.2)
*/
void test_and2 ()
{
  Machine m;
  uint32_t b[9];
  DEBUGLN (" * t:and.2");

  b[0] = enc (8*4);             /* Code Size */
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
  mLoad (&m, (Bytecode *) b, sizeof (b));
  assert (!mMatch (&m, "b", 1)); /* Failed */
  mFree (&m);
}

void test_and2_back_commit ()
{
  Machine m;
  uint32_t b[6];
  DEBUGLN (" * t:and.2 (back-commit)");

  b[0] = enc (5*4);                /* Code Size */
  /* &'a' */
  b[1] = GEN1 (OP_CHOICE, 3);      /* 0x0: Choice 0x3 */
  b[2] = GEN1 (OP_CHAR, 'a');      /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_BACK_COMMIT, 2); /* 0x2: BackCommit 2 */
  b[4] = GEN0 (OP_FAIL);           /* 0x3: Fail */
  b[5] = GEN0 (OP_HALT);           /* 0x4: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  assert (!mMatch (&m, "b", 1)); /* Failed */
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
  uint32_t b[5];
  const char *i = "abc";
  const char *o = NULL;
  DEBUGLN (" * t:con.1");

  b[0] = enc (4*4);             /* Code Size */
  /* 'a' . 'c' */
  b[1] = GEN1 (OP_CHAR, 'a');   /* 0x0: Char 'a' */
  b[2] = GEN0 (OP_ANY);         /* 0x1: Any */
  b[3] = GEN1 (OP_CHAR, 'c');   /* 0x2: Char 'c' */
  b[4] = GEN0 (OP_HALT);        /* 0x3: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  o = mMatch (&m, i, strlen (i));
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - i == 3);          /* Matched all 3 chars */
}

/*
  match p1 s i = i+j    match p2 s i + j = nil
  ----------------------------------------------
         match p1 p2 s i = nil (con.2)
 */
void test_con2 ()
{
  Machine m;
  uint32_t b[5];
  DEBUGLN (" * t:con.2");

  b[0] = enc (4*4);             /* Code Size */
  /* 'a' 'c' . */
  b[1] = GEN1 (OP_CHAR, 'a');   /* 0x0: Char 'a' */
  b[2] = GEN1 (OP_CHAR, 'c');   /* 0x1: Char 'c' */
  b[3] = GEN0 (OP_ANY);         /* 0x2: Any */
  b[4] = GEN0 (OP_HALT);        /* 0x3: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  assert (!mMatch (&m, "abc", 3)); /* Failed */
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
  uint32_t b[5];
  DEBUGLN (" * t:con.3");

  b[0] = enc (4*4);             /* Code Size */
  /* 'a' 'c' . */
  b[1] = GEN1 (OP_CHAR, 'a');   /* 0x0: Char 'a' */
  b[2] = GEN1 (OP_CHAR, 'c');   /* 0x1: Char 'c' */
  b[3] = GEN0 (OP_ANY);         /* 0x2: Any */
  b[4] = GEN0 (OP_HALT);        /* 0x3: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  assert (!mMatch (&m, "cba", 3)); /* Failed */
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
  uint32_t b[5];
  DEBUGLN (" * t:ord.1");

  /* 'a' / 'b' */
  b[0] = GEN1 (OP_CHOICE, 3);   /* 0x0: Choice 0x3 */
  b[1] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[2] = GEN1 (OP_COMMIT, 2);   /* 0x2: Commit 0x2 */
  b[3] = GEN1 (OP_CHAR, 'b');   /* 0x3: Char 'c' */
  b[4] = 0;                     /* 0x4: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  assert (!mMatch (&m, "c", 1)); /* Failed */
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
  uint32_t b[6];
  const char *i = "a";
  const char *o = NULL;
  DEBUGLN (" * t:ord.2");

  b[0] = enc (5*4);             /* Code Size */
  /* 'a' / 'b' */
  b[1] = GEN1 (OP_CHOICE, 3);   /* 0x0: Choice 0x3 */
  b[2] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_COMMIT, 2);   /* 0x2: Commit 0x2 */
  b[4] = GEN1 (OP_CHAR, 'b');   /* 0x3: Char 'c' */
  b[5] = GEN0 (OP_HALT);        /* 0x4: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  o = mMatch (&m, i, strlen (i));
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - i == 1);          /* Match the first char */
}

/*
  match p1 s i = nil    match p2 s i = i+k
  ----------------------------------------
      match p1 / p2 s i = i+k (ord.3)
 */
void test_ord3 ()
{
  Machine m;
  uint32_t b[6];
  const char *i = "b";
  const char *o = NULL;
  DEBUGLN (" * t:ord.3");

  b[0] = enc (5*4);             /* Code Size */
  /* 'a' / 'b' */
  b[1] = GEN1 (OP_CHOICE, 3);   /* 0x0: Choice 0x3 */
  b[2] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_COMMIT, 2);   /* 0x2: Commit 0x2 */
  b[4] = GEN1 (OP_CHAR, 'b');   /* 0x3: Char 'c' */
  b[5] = GEN0 (OP_HALT);        /* 0x4: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  o = mMatch (&m, i, strlen (i));
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - i == 1);          /* Match the first char */
}

/*
  match p s i = i+j    match p∗ s i + j = i+j+k
  ----------------------------------------------
          match p∗ s i = i+j+k (rep.1)
*/
void test_rep1 ()
{
  Machine m;
  uint32_t b[5];
  const char *i = "aab";
  const char *o = NULL;
  DEBUGLN (" * t:rep.1");

  b[0] = enc (4*4);             /* Code Size */
  /* 'a*' */
  b[1] = GEN1 (OP_CHOICE, 3);   /* 0x0: Choice 3 */
  b[2] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_COMMIT, -2);  /* 0x2: Commit -2 */
  b[4] = GEN0 (OP_HALT);        /* 0x3: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  o = mMatch (&m, i, strlen (i));
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - i == 2);          /* Matched two chars */
}

void test_rep1_partial_commit ()
{
  Machine m;
  uint32_t b[5];
  const char *i = "aab";
  const char *o = NULL;
  DEBUGLN (" * t:rep.1 (partial-commit)");

  b[0] = enc (4*4);             /* Code Size */
  /* 'a*' */
  b[1] = GEN1 (OP_CHOICE, 3);          /* 0x0: Choice 3 */
  b[2] = GEN1 (OP_CHAR, 'a');          /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_PARTIAL_COMMIT, -1); /* 0x3: PartialCommit -1 */
  b[4] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  o = mMatch (&m, i, strlen (i));
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - i == 2);        /* Matched two chars */
}

/*
  match p s i = nil
  -----------------
  match p∗ s i = i (rep.2)
*/
void test_rep2 ()
{
  Machine m;
  uint32_t b[5];
  const char *i = "b";
  const char *o = NULL;
  DEBUGLN (" * t:rep.2");

  b[0] = enc (4*4);             /* Code Size */
  /* 'a*' */
  b[1] = GEN1 (OP_CHOICE, 3);   /* 0x0: Choice 3 */
  b[2] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_COMMIT, -2);  /* 0x2: Commit -2 */
  b[4] = GEN0 (OP_HALT);        /* 0x3: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  o = mMatch (&m, i, strlen (i));
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - i == 0);          /* But didn't match any char */
}

void test_rep2_partial_commit ()
{
  Machine m;
  uint32_t b[5];
  const char *i = "b";
  const char *o = NULL;
  DEBUGLN (" * t:rep.2 (partial-commit)");

  b[0] = enc (4*4);                    /* Code Size */
  /* 'a*' */
  b[1] = GEN1 (OP_CHOICE, 3);          /* 0x0: Choice 3 */
  b[2] = GEN1 (OP_CHAR, 'a');          /* 0x1: Char 'a' */
  b[3] = GEN1 (OP_PARTIAL_COMMIT, -1); /* 0x3: PartialCommit -1 */
  b[4] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  o = mMatch (&m, i, strlen (i));
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - i == 0);          /* But didn't match any char */
}

/*
  match g g(Ak) s i = i+j
  -----------------------
  match g Ak s i = i+j (var.1)
*/
void test_var1 ()
{
  Machine m;
  uint32_t b[13];
  const char *i = "1+1";
  const char *o = NULL;
  DEBUGLN (" * t:var.1");

  b[0x0] = enc (12*4);          /* Code Size */
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
  mLoad (&m, (Bytecode *) b, sizeof (b));
  o = mMatch (&m, i, strlen (i));
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - i == 3);          /* Matched the whole input */
}

/*
  match g g(Ak) s i = nil
  -----------------------
  match g Ak s i = nil (var.2)
*/
void test_var2 ()
{
  Machine m;
  uint32_t b[13];
  DEBUGLN (" * t:var.2");

  b[0x0] = enc (12*4);          /* Code Size */
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
  mLoad (&m, (Bytecode *) b, sizeof (b));
  assert (!mMatch (&m, "1+2", 3)); /* Failed */
  mFree (&m);
}

void test_span1 ()
{
  Machine m;
  uint32_t b[5];
  const char *i = "abcdefgh";
  const char *o = NULL;
  DEBUGLN (" * t:span.1");

  b[0x0] = enc (4*4);                /* Code Size */
  /* '[a-e]*' */
  b[0x1] = GEN1 (OP_CHOICE, 3);      /* 0x0: Choice 0x3 */
  b[0x2] = GEN2 (OP_SPAN, 'a', 'e'); /* 0x1: Span 'a'-'e' */
  b[0x3] = GEN1 (OP_COMMIT, -2);     /* 0x2: Commit -0x2 */
  b[0x4] = GEN0 (OP_HALT);           /* 0x3: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  o = mMatch (&m, i, strlen (i));
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - i == 5);          /* Matched chars */
}

void test_cap1 ()
{
  Machine m;
  uint32_t b[10];
  const char *i = "a";
  Object *out = NULL;
  DEBUGLN (" * t:cap.1");

  b[0x0] = enc (9*4);           /* Code Size */
  /* S <- 'a' */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x8);
  b[0x3] = GEN2 (OP_CAP_OPEN, 0x0, 0x0); /* CapOpen 0 (Main) */
  b[0x4] = GEN2 (OP_CAP_OPEN, 0x1, 0x1);
  b[0x5] = GEN1 (OP_CHAR, 0x61);
  b[0x6] = GEN2 (OP_CAP_CLOSE, 0x1, 0x1);
  b[0x7] = GEN2 (OP_CAP_CLOSE, 0x0, 0x0); /* CapClose 0 (Main) */
  b[0x8] = GEN0 (OP_RETURN);
  b[0x9] = GEN0 (OP_HALT);

  mInit (&m);
  mSymbol (&m, "Main", 4);
  mSymbol (&m, "Char", 4);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  assert (mMatch (&m, i, strlen (i)));
  out = mExtract (&m, i);

  printObj (out);
  printf ("\n");

  assert (out);                 /* Isn't empty */
  assert (CONSP (out));         /* Is a list */
  /* assert (CONSP (CAR (out))); */
  /* assert (SYMBOLP (CAR (CAR (out))));   /\* Has an symbol within it *\/ */
  /* assert (strcmp (SYMBOL (CAR (CAR (out)))->name, "a") == 0); /\* Has the right value *\/ */

  objFree (out);
  mFree (&m);
}

/*
         PEG
  G[ε] l ---> l (empty.1)
*/
static void test_lst_empty1 ()
{
  Machine m;
  uint32_t b[10];
  Object *input = NULL;
  DEBUGLN (" * t:empty.1");

  b[0x0] = enc (4*4);           /* Code Size */
  /* *Empty* */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x4);
  b[0x3] = GEN0 (OP_RETURN);
  b[0x4] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));

  input = makeCons (mSymbol (&m, "a", 1), OBJ (Nil));
  assert (mMatchList (&m, input) == input);

  objFree (input);
  mFree (&m);
}

/*
           PEG
  G[.] x:l ---> l (any.1)
*/
static void test_lst_any1 ()
{
  Machine m;
  uint32_t b[10];
  Object *input, *output;
  DEBUGLN (" * t:lst.any.1");

  b[0x0] = enc (5*4);           /* Code Size */
  /* S <- . */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x5);
  b[0x3] = GEN0 (OP_ANY);
  b[0x4] = GEN0 (OP_RETURN);
  b[0x5] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));

  input = makeCons (mSymbol (&m, "a", 1), (Object*) Nil);
  output = mMatchList (&m, input);
  assert (output == Nil);

  objFree (input);
  mFree (&m);
}

/*
         PEG
  G[.] ε ---> fail (any.2)
*/
static void test_lst_any2 ()
{
  Machine m;
  uint32_t b[10];
  DEBUGLN (" * t:lst.any.2");

  b[0x0] = enc (5*4);           /* Code Size */
  /* S <- . */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x5);
  b[0x3] = GEN0 (OP_ANY);
  b[0x4] = GEN0 (OP_RETURN);
  b[0x5] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));

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
  uint32_t b[10];
  Object *input, *output;
  DEBUGLN (" * t:lst.term.1");

  b[0x0] = enc (5*4);           /* Code Size */
  /* S <- "MyTerm" */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x5);
  b[0x3] = GEN1 (OP_ATOM, 0x0);
  b[0x4] = GEN0 (OP_RETURN);
  b[0x5] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));

  input = makeCons (mSymbol (&m, "MyTerm", 6), OBJ (Nil));
  output = mMatchList (&m, input);
  assert (output == Nil);

  objFree (input);
  mFree (&m);
}

/*
           PEG
  G[a] b:l ---> fail (term.2)
*/
static void test_lst_term2 ()
{
  Machine m;
  uint32_t b[10];
  Object *input;
  DEBUGLN (" * t:lst.term.2");

  b[0x0] = enc (5*4);           /* Code Size */
  /* S <- "aTerm" */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x5);
  b[0x3] = GEN1 (OP_ATOM, 0x0);
  b[0x4] = GEN0 (OP_RETURN);
  b[0x5] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));

  /* Create the 0th term in the symbol table */
  mSymbol (&m, "aTerm", 5);

  /* Create the input with another symbol */
  input = makeCons (mSymbol (&m, "myTerm", 6), OBJ (Nil));
  assert (!mMatchList (&m, input));

  objFree (input);
  mFree (&m);
}

/*
         PEG
  G[a] ε ---> fail (term.3)
*/
static void test_lst_term3 ()
{
  Machine m;
  uint32_t b[10];
  DEBUGLN (" * t:lst.term.3");

  b[0x0] = enc (5*4);           /* Code Size */
  /* S <- "aTerm" */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x5);
  b[0x3] = GEN1 (OP_ATOM, 0x0);
  b[0x4] = GEN0 (OP_RETURN);
  b[0x5] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));

  /* Create the 0th term referenced in the bytecode grammar within the
   * symbol table */
  mSymbol (&m, "aTerm", 5);

  /* Run the above grammar on an empty input */
  assert (!mMatchList (&m, OBJ (Nil)));

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
  uint32_t b[10];
  Object *input;
  DEBUGLN (" * t:lst.term.4");

  b[0x0] = enc (5*4);           /* Code Size */
  /* S <- "aTerm" */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x5);
  b[0x3] = GEN1 (OP_ATOM, 0x0);
  b[0x4] = GEN0 (OP_RETURN);
  b[0x5] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));

  /* Create the 0th term referenced in the bytecode grammar within the
   * symbol table */
  mSymbol (&m, "aTerm", 5);

  /* Run the above grammar on an input with a list within a list as
   * the first element */
  input = makeCons (makeCons (mSymbol (&m, "test", 4), OBJ (Nil)), OBJ (Nil));
  assert (!mMatchList (&m, input));

  /* Cleanup */
  objFree (input);
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
  uint32_t b[10];
  Object *input, *output;
  DEBUGLN (" * t:lst.list.1");

  b[0x0] = enc (7*4);           /* Code Size */
  /* S <- { "test" } */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x7);
  b[0x3] = GEN0 (OP_OPEN);
  b[0x4] = GEN1 (OP_ATOM, 0x0);
  b[0x5] = GEN0 (OP_CLOSE);
  b[0x6] = GEN0 (OP_RETURN);
  b[0x7] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));

  /* Create the 0th term referenced in the bytecode grammar within the
   * symbol table */
  mSymbol (&m, "test", 4);

  /* Run the above grammar on an input with "test" as the first (and
   * only) element of a list */
  input = makeCons (makeCons (mSymbol (&m, "test", 4), OBJ (Nil)), OBJ (Nil));
  output = mMatchList (&m, input);

  /* It does match! */
  assert (output);

  /* Cleanup */
  objFree (input);
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
  uint32_t b[10];
  Object *input, *output;
  DEBUGLN (" * t:lst.list.2");

  b[0x0] = enc (7*4);           /* Code Size */
  /* S <- { "foo" } */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x7);
  b[0x3] = GEN0 (OP_OPEN);
  b[0x4] = GEN1 (OP_ATOM, 0x0);
  b[0x5] = GEN0 (OP_CLOSE);
  b[0x6] = GEN0 (OP_RETURN);
  b[0x7] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));

  /* Create the 0th term referenced in the bytecode grammar within the
   * symbol table */
  mSymbol (&m, "foo", 3);

  /* Run the above grammar on an input with "test" as the first (and
   * only) element of a list */
  input = makeCons (makeCons (mSymbol (&m, "test", 4), OBJ (Nil)), OBJ (Nil));
  output = mMatchList (&m, input);

  /* It DOES NOT match! */
  assert (!output);

  /* Cleanup */
  objFree (input);
  mFree (&m);
}

/*
             PEG
  G[{p}] a:l ---> fail (list.3)
*/
static void test_lst_list3 ()
{
  Machine m;
  uint32_t b[10];
  Object *input, *output;
  DEBUGLN (" * t:lst.list.3");

  b[0x0] = enc (7*4);           /* Code Size */
  /* S <- { "foo" } */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x7);
  b[0x3] = GEN0 (OP_OPEN);
  b[0x4] = GEN1 (OP_ATOM, 0x0);
  b[0x5] = GEN0 (OP_CLOSE);
  b[0x6] = GEN0 (OP_RETURN);
  b[0x7] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));

  /* Create the 0th term referenced in the bytecode grammar within the
   * symbol table */
  mSymbol (&m, "foo", 3);

  /* Run the above grammar on an input with "test" as the first (and
   * only) element of a list */
  input = makeCons (mSymbol (&m, "test", 4), OBJ (Nil));
  output = mMatchList (&m, input);

  /* It DOES NOT match! */
  assert (!output);

  /* Cleanup */
  objFree (input);
  mFree (&m);
}

/*
           PEG
  G[{p}] ε ---> fail
*/
static void test_lst_list4 ()
{
  Machine m;
  uint32_t b[10];
  Object *output;
  DEBUGLN (" * t:lst.list.4");

  b[0x0] = enc (7*4);           /* Code Size */
  /* S <- { "foo" } */
  b[0x1] = GEN1 (OP_CALL, 0x2);
  b[0x2] = GEN1 (OP_JUMP, 0x7);
  b[0x3] = GEN0 (OP_OPEN);
  b[0x4] = GEN1 (OP_ATOM, 0x0);
  b[0x5] = GEN0 (OP_CLOSE);
  b[0x6] = GEN0 (OP_RETURN);
  b[0x7] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));

  /* Create the 0th term referenced in the bytecode grammar within the
   * symbol table */
  mSymbol (&m, "foo", 3);

  /* Run the above grammar on a empty input */
  output = mMatchList (&m, OBJ (Nil));

  /* It DOES NOT match! */
  assert (!output);

  /* Cleanup */
  mFree (&m);
}

void test_obj_equal_int ()
{
  Object *o1 = makeInt (2);
  Object *o2 = makeInt (3);
  Object *o3 = makeInt (2);

  assert (!objEqual (o1, o2));
  assert (!objEqual (o2, o3));
  assert (objEqual (o1, o3));

  objFree (o1);
  objFree (o2);
  objFree (o3);
}

void test_obj_equal_symbol ()
{
  Machine m;
  Object *o1, *o2, *o3;

  mInit (&m);
  o1 = mSymbol (&m, "Hi!", 3);
  o2 = mSymbol (&m, "Oi!", 3);
  o3 = mSymbol (&m, "Hi!", 3);

  assert (!objEqual (o1, o2));
  assert (!objEqual (o2, o3));
  assert (objEqual (o1, o3));

  mFree (&m);
}

void test_obj_equal_cons ()
{
  Machine m;
  Object *o1, *o2, *o3;

  mInit (&m);
  o1 = makeCons (mSymbol (&m, "Hi!", 3), makeCons (makeInt (5), NULL));
  o2 = makeCons (mSymbol (&m, "Hi!", 3), makeInt (5));
  o3 = makeCons (mSymbol (&m, "Hi!", 3), makeCons (makeInt (5), NULL));

  assert (!objEqual (o1, o2));
  assert (!objEqual (o2, o3));
  assert (objEqual (o1, o3));

  objFree (o1);
  objFree (o2);
  objFree (o3);
  mFree (&m);
}

int main ()
{
  test_gen_args ();
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

  test_obj_equal_int ();
  test_obj_equal_symbol ();
  test_obj_equal_cons ();
  return 0;
}
