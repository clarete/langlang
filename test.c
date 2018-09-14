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
  uint32_t b[2];
  const char *i = "a";
  const char *o = NULL;
  DEBUGLN (" * t:ch.1");

  /* Start <- 'a' */
  b[0] = GEN1 (OP_CHAR, 'a'); /* 0x0: Char 'a' */
  b[1] = 0;                   /* 0x1: Halt */

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
  uint32_t b[2];
  DEBUGLN (" * t:ch.2");

  /* Start <- 'a' */
  b[0] = GEN1 (OP_CHAR, 'a');   /* 0x0: Char 'a' */
  b[1] = 0;                     /* 0x1: Halt */

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
  uint32_t b[2];
  const char *i = "a";
  const char *o = NULL;
  DEBUGLN (" * t:any.1");

  /* . */
  b[0] = GEN0 (OP_ANY);         /* 0x0: Any */
  b[1] = 0;                     /* 0x1: Halt */

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
  uint32_t b[2];
  DEBUGLN (" * t:any.2");

  /* . */
  b[0] = GEN0 (OP_ANY);         /* 0x0: Any */
  b[1] = 0;                     /* 0x1: Halt */

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
  uint32_t b[5];
  const char *i = "b";
  const char *o = NULL;
  DEBUGLN (" * t:not.1");

  /* !'a' */
  b[0] = GEN1 (OP_CHOICE, 4);   /* 0x0: Choice 0x4 */
  b[1] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[2] = GEN1 (OP_COMMIT, 1);   /* 0x1: Commit 1 */
  b[3] = GEN0 (OP_FAIL);        /* 0x2: Fail */
  b[4] = 0;                     /* 0x3: Halt */

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
  uint32_t b[4];
  const char *i = "b";
  const char *o = NULL;
  DEBUGLN (" * t:not.1 (fail-twice)");

  /* !'a' */
  b[0] = GEN1 (OP_CHOICE, 4);   /* 0x0: Choice 0x4 */
  b[1] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[2] = GEN0 (OP_FAIL_TWICE);  /* 0x2: FailTwice */
  b[3] = 0;                     /* 0x3: Halt */

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
  uint32_t b[5];
  DEBUGLN (" * t:not.2");

  /* !'a' */
  b[0] = GEN1 (OP_CHOICE, 4);   /* 0x0: Choice 0x4 */
  b[1] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[2] = GEN1 (OP_COMMIT, 1);   /* 0x2: Commit 1 */
  b[3] = GEN0 (OP_FAIL);        /* 0x3: Fail */
  b[4] = 0;                     /* 0x4: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  assert (!mMatch (&m, "a", 1)); /* Failed */
  mFree (&m);
}

void test_not2_fail_twice ()
{
  Machine m;
  uint32_t b[4];
  DEBUGLN (" * t:not.2 (fail-twice)");

  /* !'a' */
  b[0] = GEN1 (OP_CHOICE, 4);   /* 0x0: Choice 0x4 */
  b[1] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[2] = GEN0 (OP_FAIL_TWICE);  /* 0x2: FailTwice */
  b[3] = 0;                     /* 0x3: Halt */

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
  uint32_t b[8];
  const char *i = "a";
  const char *o = NULL;
  DEBUGLN (" * t:and.1");

  /* &'a' */
  b[0] = GEN1 (OP_CHOICE, 7);   /* 0x0: Choice 0x7 */
  b[1] = GEN1 (OP_CHOICE, 4);   /* 0x1: Choice 0x4 */
  b[2] = GEN1 (OP_CHAR, 'a');   /* 0x2: Char 'a' */
  b[3] = GEN1 (OP_COMMIT, 1);   /* 0x3: Commit 1 */
  b[4] = GEN0 (OP_FAIL);        /* 0x4: Fail */
  b[5] = GEN1 (OP_COMMIT, 1);   /* 0x5: Commit 1 */
  b[6] = GEN0 (OP_FAIL);        /* 0x6: Fail */
  b[7] = 0;                     /* 0x7: Halt */

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
  uint32_t b[5];
  const char *i = "a";
  const char *o = NULL;
  DEBUGLN (" * t:and.1 (back-commit)");

  /* &'a' */
  b[0] = GEN1 (OP_CHOICE, 3);      /* 0x0: Choice 0x3 */
  b[1] = GEN1 (OP_CHAR, 'a');      /* 0x1: Char 'a' */
  b[2] = GEN1 (OP_BACK_COMMIT, 2); /* 0x2: BackCommit 2 */
  b[3] = GEN0 (OP_FAIL);           /* 0x3: Fail */
  b[4] = 0;                        /* 0x4: Halt */

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
  uint32_t b[8];
  DEBUGLN (" * t:and.2");

  /* &'a' */
  b[0] = GEN1 (OP_CHOICE, 7);   /* 0x0: Choice 0x7 */
  b[1] = GEN1 (OP_CHOICE, 4);   /* 0x1: Choice 0x4 */
  b[2] = GEN1 (OP_CHAR, 'a');   /* 0x2: Char 'a' */
  b[3] = GEN1 (OP_COMMIT, 1);   /* 0x3: Commit 1 */
  b[4] = GEN0 (OP_FAIL);        /* 0x4: Fail */
  b[5] = GEN1 (OP_COMMIT, 1);   /* 0x5: Commit 1 */
  b[6] = GEN0 (OP_FAIL);        /* 0x6: Fail */
  b[7] = 0;                     /* 0x7: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  assert (!mMatch (&m, "b", 1)); /* Failed */
  mFree (&m);
}

void test_and2_back_commit ()
{
  Machine m;
  uint32_t b[5];
  DEBUGLN (" * t:and.2 (back-commit)");

  /* &'a' */
  b[0] = GEN1 (OP_CHOICE, 3);      /* 0x0: Choice 0x3 */
  b[1] = GEN1 (OP_CHAR, 'a');      /* 0x1: Char 'a' */
  b[2] = GEN1 (OP_BACK_COMMIT, 2); /* 0x2: BackCommit 2 */
  b[3] = GEN0 (OP_FAIL);           /* 0x3: Fail */
  b[4] = 0;                        /* 0x4: Halt */

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
  uint32_t b[4];
  const char *i = "abc";
  const char *o = NULL;
  DEBUGLN (" * t:con.1");

  /* 'a' . 'c' */
  b[0] = GEN1 (OP_CHAR, 'a');   /* 0x0: Char 'a' */
  b[1] = GEN0 (OP_ANY);         /* 0x1: Any */
  b[2] = GEN1 (OP_CHAR, 'c');   /* 0x2: Char 'c' */
  b[3] = 0;                     /* 0x3: Halt */

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
  uint32_t b[4];
  DEBUGLN (" * t:con.2");

  /* 'a' 'c' . */
  b[0] = GEN1 (OP_CHAR, 'a');   /* 0x0: Char 'a' */
  b[1] = GEN1 (OP_CHAR, 'c');   /* 0x1: Char 'c' */
  b[2] = GEN0 (OP_ANY);         /* 0x2: Any */
  b[3] = 0;                     /* 0x3: Halt */

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
  uint32_t b[4];
  DEBUGLN (" * t:con.3");

  /* 'a' 'c' . */
  b[0] = GEN1 (OP_CHAR, 'a');   /* 0x0: Char 'a' */
  b[1] = GEN1 (OP_CHAR, 'c');   /* 0x1: Char 'c' */
  b[2] = GEN0 (OP_ANY);         /* 0x2: Any */
  b[3] = 0;                     /* 0x3: Halt */

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
  uint32_t b[5];
  const char *i = "a";
  const char *o = NULL;
  DEBUGLN (" * t:ord.2");

  /* 'a' / 'b' */
  b[0] = GEN1 (OP_CHOICE, 3);   /* 0x0: Choice 0x3 */
  b[1] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[2] = GEN1 (OP_COMMIT, 2);   /* 0x2: Commit 0x2 */
  b[3] = GEN1 (OP_CHAR, 'b');   /* 0x3: Char 'c' */
  b[4] = 0;                     /* 0x4: Halt */

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
  uint32_t b[5];
  const char *i = "b";
  const char *o = NULL;
  DEBUGLN (" * t:ord.3");

  /* 'a' / 'b' */
  b[0] = GEN1 (OP_CHOICE, 3);   /* 0x0: Choice 0x3 */
  b[1] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[2] = GEN1 (OP_COMMIT, 2);   /* 0x2: Commit 0x2 */
  b[3] = GEN1 (OP_CHAR, 'b');   /* 0x3: Char 'c' */
  b[4] = 0;                     /* 0x4: Halt */

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
  uint32_t b[4];
  const char *i = "aab";
  const char *o = NULL;
  DEBUGLN (" * t:rep.1");

  /* 'a*' */
  b[0] = GEN1 (OP_CHOICE, 3);   /* 0x0: Choice 3 */
  b[1] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[2] = GEN1 (OP_COMMIT, -2);  /* 0x2: Commit -2 */
  b[3] = 0;                     /* 0x3: Halt */

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
  uint32_t b[4];
  const char *i = "aab";
  const char *o = NULL;
  DEBUGLN (" * t:rep.1 (partial-commit)");

  /* 'a*' */
  b[0] = GEN1 (OP_CHOICE, 3);          /* 0x0: Choice 3 */
  b[1] = GEN1 (OP_CHAR, 'a');          /* 0x1: Char 'a' */
  b[2] = GEN1 (OP_PARTIAL_COMMIT, -1); /* 0x3: PartialCommit -1 */
  b[3] = 0;

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
  uint32_t b[4];
  const char *i = "b";
  const char *o = NULL;
  DEBUGLN (" * t:rep.2");

  /* 'a*' */
  b[0] = GEN1 (OP_CHOICE, 3);   /* 0x0: Choice 3 */
  b[1] = GEN1 (OP_CHAR, 'a');   /* 0x1: Char 'a' */
  b[2] = GEN1 (OP_COMMIT, -2);  /* 0x2: Commit -2 */
  b[3] = 0;                     /* 0x3: Halt */

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
  uint32_t b[4];
  const char *i = "b";
  const char *o = NULL;
  DEBUGLN (" * t:rep.2 (partial-commit)");

  /* 'a*' */
  b[0] = GEN1 (OP_CHOICE, 3);          /* 0x0: Choice 3 */
  b[1] = GEN1 (OP_CHAR, 'a');          /* 0x1: Char 'a' */
  b[2] = GEN1 (OP_PARTIAL_COMMIT, -1); /* 0x3: PartialCommit -1 */
  b[3] = 0;

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
  uint32_t b[12];
  const char *i = "1+1";
  const char *o = NULL;
  DEBUGLN (" * t:var.1");

  /* 'a*' */

  /* Start */
  b[0x0] = GEN1 (OP_CALL, 0x2); /* 0x0: Call 0x2 */
  b[0x1] = GEN1 (OP_JUMP, 0xb); /* 0x1: Jump 0xb */
  /* S <- D '+' D */
  b[0x2] = GEN1 (OP_CALL, 0x4); /* 0x2: Call 0x4 */
  b[0x3] = GEN1 (OP_CHAR, '+'); /* 0x3: Char '+' */
  b[0x4] = GEN1 (OP_CALL, 0x2); /* 0x4: Call 0x2 */
  b[0x5] = GEN0 (OP_RETURN);    /* 0x5: Return */
  /* D <- '0' / '1' */
  b[0x6] = GEN1 (OP_CHOICE, 3); /* 0x6: Choice 0x3 */
  b[0x7] = GEN1 (OP_CHAR, '0'); /* 0x7: Char '0' */
  b[0x8] = GEN1 (OP_COMMIT, 2); /* 0x8: Commit 0x2 */
  b[0x9] = GEN1 (OP_CHAR, '1'); /* 0x9: Char '1' */
  b[0xa] = GEN0 (OP_RETURN);    /* 0xa: Return */
  /* End */
  b[0xb] = 0;                   /* 0xb: Halt */

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
  uint32_t b[12];
  DEBUGLN (" * t:var.2");

  /* 'a*' */

  /* Start */
  b[0x0] = GEN1 (OP_CALL, 0x2); /* 0x0: Call 0x2 */
  b[0x1] = GEN1 (OP_JUMP, 0xb); /* 0x1: Jump 0xb */
  /* S <- D '+' D */
  b[0x2] = GEN1 (OP_CALL, 0x4); /* 0x2: Call 0x4 */
  b[0x3] = GEN1 (OP_CHAR, '+'); /* 0x3: Char '+' */
  b[0x4] = GEN1 (OP_CALL, 0x2); /* 0x4: Call 0x2 */
  b[0x5] = GEN0 (OP_RETURN);    /* 0x5: Return */
  /* D <- '0' / '1' */
  b[0x6] = GEN1 (OP_CHOICE, 3); /* 0x6: Choice 0x3 */
  b[0x7] = GEN1 (OP_CHAR, '0'); /* 0x7: Char '0' */
  b[0x8] = GEN1 (OP_COMMIT, 2); /* 0x8: Commit 0x2 */
  b[0x9] = GEN1 (OP_CHAR, '1'); /* 0x9: Char '1' */
  b[0xa] = GEN0 (OP_RETURN);    /* 0xa: Return */
  /* End */
  b[0xb] = 0;                   /* 0xb: Halt */

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  assert (!mMatch (&m, "1+2", 3)); /* Failed */
  mFree (&m);
}

void test_span1 ()
{
  Machine m;
  uint32_t b[4];
  const char *i = "abcdefgh";
  const char *o = NULL;
  DEBUGLN (" * t:span.1");

  /* '[a-e]*' */
  b[0x0] = GEN1 (OP_CHOICE, 3);      /* 0x0: Choice 0x3 */
  b[0x1] = GEN2 (OP_SPAN, 'a', 'e'); /* 0x1: Span 'a'-'e' */
  b[0x2] = GEN1 (OP_COMMIT, -2);     /* 0x2: Commit -0x2 */
  b[0x3] = 0;

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
  uint32_t b[9];
  const char *i = "a";
  Object *out = NULL;
  DEBUGLN (" * t:cap.1");

  /* S <- 'a' */
  b[0x0] = GEN1 (OP_CALL, 0x2);
  b[0x1] = GEN1 (OP_JUMP, 0x8);
  b[0x2] = GEN2 (OP_CAP_OPEN, 0x0, 0x0); /* CapOpen 0 (Main) */
  b[0x3] = GEN2 (OP_CAP_OPEN, 0x1, 0x1);
  b[0x4] = GEN1 (OP_CHAR, 0x61);
  b[0x5] = GEN2 (OP_CAP_CLOSE, 0x1, 0x1);
  b[0x6] = GEN2 (OP_CAP_CLOSE, 0x0, 0x0); /* CapClose 0 (Main) */
  b[0x7] = GEN0 (OP_RETURN);
  b[0x8] = GEN0 (OP_HALT);

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  assert (mMatch (&m, i, strlen (i)));
  out = mExtract (&m, i);

  printObj (out);
  printf ("\n");

  assert (out);                 /* Isn't empty */
  assert (CONSP (out));         /* Is a list */
  /* assert (CONSP (CAR (out))); */
  /* assert (ATOMP (CAR (CAR (out))));   /\* Has an atom within it *\/ */
  /* assert (strcmp (ATOM (CAR (CAR (out)))->name, "a") == 0); /\* Has the right value *\/ */

  mFree (&m);
}

void test_cap2 ()
{
  Machine m;
  uint32_t b[14];
  const char *i = "ab";
  Object *out = NULL;
  DEBUGLN (" * t:cap.2");

  /* S <- 'a' 'b' / 'ab' */
  b[0x0] = GEN1 (OP_CALL, 0x2);
  b[0x1] = GEN1 (OP_JUMP, 0x0d);
  b[0x2] = GEN2 (OP_CAP_OPEN, 0x0, 0x0); /* CapOpen NT 0 (Main) */
  b[0x3] = GEN2 (OP_CAP_OPEN, 0x0, 0x1); /* CapOpen NT 1 (Seq) */
  b[0x4] = GEN2 (OP_CAP_OPEN, 0x1, 0x2); /* CapOpen YT 2 */
  b[0x5] = GEN1 (OP_CHAR, 0x61);
  b[0x6] = GEN2 (OP_CAP_CLOSE, 0x1, 0x2); /* CapClose YT 2 */
  b[0x7] = GEN2 (OP_CAP_OPEN, 0x1, 0x3);
  b[0x8] = GEN1 (OP_CHAR, 0x62);
  b[0x9] = GEN2 (OP_CAP_CLOSE, 0x1, 0x3);
  b[0xa] = GEN2 (OP_CAP_CLOSE, 0x0, 0x1); /* CapClose 0 (Seq) */
  b[0xb] = GEN2 (OP_CAP_CLOSE, 0x0, 0x0); /* CapClose 0 (Main) */
  b[0xc] = GEN0 (OP_RETURN);
  b[0xd] = OP_HALT;

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  assert (mMatch (&m, i, strlen (i)));
  out = mExtract (&m, i);

  printObj (out);
  printf ("\n");

  assert (out);                 /* Isn't empty */
  assert (CONSP (out));         /* Is a list */
  /* assert (CONSP (CAR (out))); */
  /* assert (CONSP (CAR (CAR (out)))); */
  /* assert (ATOMP (CAR (CAR (CAR (out))))); */
  /* assert (strcmp (ATOM (CAR (CAR (CAR (out))))->name, "a") == 0); */
  /* assert (ATOMP (CAR (CDR (CAR (CAR (out)))))); */
  /* assert (strcmp (ATOM (CAR (CDR (CAR (CAR (out)))))->name, "b") == 0); */

  mFree (&m);
}

void test_cap3 ()
{
  Machine m;
  uint32_t b[16];
  const char *i = "b";
  Object *out = NULL;
  DEBUGLN (" * t:cap.3");

  /* S <- !'a' . */
  b[0x00] = GEN1 (     OP_CALL,       0x02);
  b[0x01] = GEN1 (     OP_JUMP,       0x0e);
  b[0x02] = GEN2 ( OP_CAP_OPEN, 0x00, 0x00);
  b[0x03] = GEN2 ( OP_CAP_OPEN, 0x00, 0x01);
  b[0x04] = GEN1 (   OP_CHOICE,       0x04);
  b[0x05] = GEN1 (     OP_CHAR,       0x61);
  b[0x06] = GEN1 (   OP_COMMIT,       0x01);
  b[0x07] = GEN0 (     OP_FAIL            );
  b[0x08] = GEN2 ( OP_CAP_OPEN, 0x01, 0x02);
  b[0x09] = GEN0 (      OP_ANY            );
  b[0x0a] = GEN2 (OP_CAP_CLOSE, 0x01, 0x02);
  b[0x0b] = GEN2 (OP_CAP_CLOSE, 0x00, 0x01);
  b[0x0c] = GEN2 (OP_CAP_CLOSE, 0x00, 0x00);
  b[0x0d] = GEN0 (   OP_RETURN            );
  b[0x0e] = GEN0 (     OP_HALT            );

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  assert (mMatch (&m, i, strlen (i)));
  out = mExtract (&m, i);

  printObj (out);
  printf ("\n");
}

void test_cap4 ()
{
  Machine m;
  uint32_t b[17];
  const char *i = "bcde";
  Object *out = NULL;
  DEBUGLN (" * t:cap.4");

  /* S <- (!'a' .)* */
  b[0x00] = GEN1 (     OP_CALL,       0x02);
  b[0x01] = GEN1 (     OP_JUMP,       0x10);
  b[0x02] = GEN2 ( OP_CAP_OPEN, 0x00, 0x00);
  b[0x03] = GEN1 (   OP_CHOICE,       0x0b);
  b[0x04] = GEN2 ( OP_CAP_OPEN, 0x00, 0x01);
  b[0x05] = GEN1 (   OP_CHOICE,       0x04);
  b[0x06] = GEN1 (     OP_CHAR,       0x61);
  b[0x07] = GEN1 (   OP_COMMIT,       0x01);
  b[0x08] = GEN0 (     OP_FAIL            );
  b[0x09] = GEN2 ( OP_CAP_OPEN, 0x01, 0x02);
  b[0x0a] = GEN0 (      OP_ANY            );
  b[0x0b] = GEN2 (OP_CAP_CLOSE, 0x01, 0x02);
  b[0x0c] = GEN2 (OP_CAP_CLOSE, 0x00, 0x01);
  b[0x0d] = GEN1 (   OP_COMMIT,  0x7fffff6);
  b[0x0e] = GEN2 (OP_CAP_CLOSE, 0x00, 0x00);
  b[0x0f] = GEN0 (   OP_RETURN            );
  b[0x10] = GEN0 (     OP_HALT            );

  mInit (&m);
  mLoad (&m, (Bytecode *) b, sizeof (b));
  assert (mMatch (&m, i, strlen (i)));
  out = mExtract (&m, i);

  printObj (out);
  printf ("\n");
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
  test_cap2 ();
  test_cap3 ();
  test_cap4 ();

  return 0;
}
