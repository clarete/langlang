#include <stdint.h>
#include <stdio.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>
#include <assert.h>

#include "debug.h"

/*

Basic Functionality
===================

Matching
 * [x] Patterns
 * [ ] Expressions
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
 * [ ] &a (syntax suggar)
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
* [ ] BackCommit l
* [ ] TestChar l o
* [ ] TestAny n o
* [ ] Jump l
* [ ] Call l
* [ ] Return
* [ ] Set
* [ ] Span

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

Each instruction is 32 bit long. The first 4 bits are reserved for the
opcode and the other 28 bits store parameters for the instruction.  We
have instructions that take 0, 1 or 2 parameters. Since there are only
4 bits for instructions, we can have at most 31 of them.

The utility `OP_MASK()' can be used to read the OPCODE from a 32bit
instruction data. Each argument size introduces different functions.
They're Here are the types of arguments:

Instruction with 1 parameter (Eg.: Char x)
------------------------------------------
    opcode  | Parameter #1
    --------|--------------------------------------------------------
    |0|0|0|1|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|1|1|0|0|0|0|1|
    --------|--------------------------------------------------------
    [ 0 - 4 |                                                5 - 32 ]
    [     4 |                                                    28 ]

    * soperand() Read signed value
    * uoperand() Read unsigned value

Instruction with 2 parameters (Eg.: TestChar 4 97)
-------------------------------------------------
    opcode  | Parameter #1              | Parameter #2
    --------|---------------------------|----------------------------
    |0|0|0|1|0|0|0|0|0|0|0|0|0|0|0|1|0|0|0|0|0|0|0|0|0|1|1|0|0|0|0|1|
    --------|---------------------------|----------------------------
    [ 0 - 4 |                    5 - 18 |                   19 - 32 ]
    [     4 |                        14 |                        14 ]

    * s1operand() Read first operand as signed value
    * u1operand() Read first operand as unsigned values
    * s2operand() Read second operand as signed value
    * u2operand() Read second operand as unsigned value
*/

/* Instruction Offsets - All sizes are in bits */
#define INSTRUCTION_SIZE    32    /* Instruction size */
#define OPERATOR_SIZE       4     /* Operator size */
#define OPERATOR_OFFSET     (INSTRUCTION_SIZE - OPERATOR_SIZE)

/* [0..OPERATOR_OFFSET] */
#define S_OPERAND_SIZE   OPERATOR_OFFSET

/* [OPERATOR_OFFSET..S1_OPERAND_SIZE */
#define S1_OPERAND_SIZE    14
#define S1_OPERAND_OFFSET  OPERATOR_OFFSET - S1_OPERATOR_SIZE

#define S1_OPERAND_SIZE    14

#define S2_OPERAND_SIZE  14

/** Clear all 28bits from the right then shift to the right */
#define OP_MASK(c) (((c) & 0xff000000) >> OPERATOR_OFFSET)

/** Read unsigned single operand */
#define UOPERAND(op) (op->rand & 0x00ffffff)
/** Read signed values */
#define SIGNED(i,s) ((int32_t) ((i & (1 << (s - 1))) ? (i | ~((1 << s) - 1)) : i))
/** Read single operand from instruction */
#define SOPERAND0(op) SIGNED (op->rand, S_OPERAND_SIZE)

/* Arbitrary values */

#define STACK_SIZE 512

/* -- Error control & report utilities -- */

/** Retrieve name of opcode */
#define OP_NAME(o) opNames[o]

/** Report errors that stop the execution of the VM right away */
#define FATAL(f, ...)                                                  \
  do { fprintf (stderr, f "\n", ##__VA_ARGS__); exit (EXIT_FAILURE); } \
  while (0)

/* opcodes */
typedef enum {
  OP_CHAR = 0x1,
  OP_ANY,
  OP_CHOICE,
  OP_COMMIT,
  OP_FAIL,
  OP_FAIL_TWICE,
  OP_PARTIAL_COMMIT,
  OP_END,
} Instructions;

/* The code is represented as a list of instructions */
typedef uint8_t Bytecode;

/* Instruction following the format of 4b operator and 12b operand */
typedef struct {
  unsigned short rator: 4;
  uint32_t rand: 28;
} Instruction;

/* Entry that's stored in the Machine's stack for supporting backtrack
   on the ordered choice operator */
typedef struct {
  const char *i;
  Instruction *pc;
} BacktrackEntry;

/* Virtual Machine */
typedef struct {
  const char *s;
  size_t s_size;
  Instruction *code;
  BacktrackEntry stack[STACK_SIZE];
} Machine;

/* Helps debugging */
static const char *opNames[OP_END] = {
  [OP_CHAR] = "OP_CHAR",
  [OP_ANY] = "OP_ANY",
  [OP_CHOICE] = "OP_CHOICE",
  [OP_COMMIT] = "OP_COMMIT",
  [OP_FAIL] = "OP_FAIL",
  [OP_FAIL_TWICE] = "OP_FAIL_TWICE",
  [OP_PARTIAL_COMMIT] = "OP_PARTIAL_COMMIT",
};

/* Set initial values for the machine */
void mInit (Machine *m, const char *input, size_t input_size)
{
  memset (m->stack, 0, STACK_SIZE * sizeof (void *));
  m->code = NULL;               /* Will be set by mRead() */
  m->s = input;
  m->s_size = input_size;
}

void mFree (Machine *m)
{
  free (m->code);
  m->code = NULL;
}

void mRead (Machine *m, Bytecode *code, size_t code_size)
{
  Instruction *tmp;
  uint32_t instr;

  /* Code size is in uint8_t and each instruction is 16bits */
  if ((tmp = m->code = calloc (sizeof (Instruction), code_size / 4 + 2)) == NULL)
    FATAL ("Can't allocate %s", "memory");
  while (*code) {
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
const char *mEval (Machine *m)
{
  BacktrackEntry *sp = m->stack;
  Instruction *pc = m->code;
  const char *i = m->s;

  /** Push data onto the machine's stack  */
#define PUSH(ii,pp) do { sp->i = ii; sp->pc = pp; sp++; } while (0)
  /** Pop data from the machine's stack. Notice it doesn't dereference
      the pointer, callers are supposed to do that when needed. */
#define POP() (--sp)
  /** The end of the input is the offset from the cursor to the end of
      the input string. */
#define THE_END (m->s + m->s_size)

  while (true) {

    /* No-op if TEST isn't defined */
    DEBUG_INSTRUCTION_NEXT ();

    switch (pc->rator) {
    case 0: return i;
    case OP_CHAR:
      DEBUG ("       OP_CHAR: `%c' == `%c' ? %d", *i,
             UOPERAND (pc), *i == UOPERAND (pc));
      if (*i == UOPERAND (pc)) { i++; pc++; }
      else goto fail;
      continue;
    case OP_ANY:
      DEBUG ("       OP_ANY: `%c' < |s| ? %d", *i, i < THE_END);
      if (i < THE_END) { i++; pc++; }
      else goto fail;
      continue;
    case OP_CHOICE:
      DEBUG ("       OP_CHOICE: `%p'", i);
      PUSH (i, pc + UOPERAND (pc));
      DEBUG_STACK ();
      pc++;
      continue;
    case OP_COMMIT:
      assert (sp > m->stack);
      POP ();                   /* Discard backtrack entry */
      pc += SOPERAND0 (pc);     /* Jump to the given position */
      DEBUG_STACK ();
      continue;
    case OP_PARTIAL_COMMIT:
      DEBUG ("       OP_PARTIAL_COMMIT: %s", i);
      pc += SOPERAND0 (pc);
      (sp - 1)->i = i;
      DEBUG_STACK ();
      continue;
    case OP_FAIL_TWICE:
      POP ();                   /* Drop top of stack & Fall through */
    case OP_FAIL:
    fail:
      /* No-op if TEST isn't defined */
      DEBUG_FAILSTATE ();
      DEBUG_STACK ();

      if (sp > m->stack) {
        /* Fail〈(pc,i1):e〉 ----> 〈pc,i1,e〉 */
        do i = (*POP ()).i;
        while (i == NULL);
        pc = sp->pc;            /* Restore the program counter */
      } else {
        /* 〈pc,i,e〉 ----> Fail〈e〉 */
        return NULL;
      }
      continue;
    default:
      FATAL ("Unknown Instruction 0x%04x", pc->rator);
    }
  }

#undef PUSH
#undef POP
#undef THE_END
}

/* Reads the entire content of the file under `path' into `buffer' */
void readFile (const char *path, uint8_t **buffer, size_t *size)
{
  FILE *fp = fopen (path, "rb");
  if (!fp) FATAL ("Can't open file %s", path);

  /* Read file size. */
  fseek (fp, 0, SEEK_END);
  *size = ftell (fp);
  rewind (fp);
  /* Allocate buffer and read the file into it.  The +1 is reserved
     for the NULL char. */
  if ((*buffer = calloc (0, *size + 1)) == NULL) {
    fclose (fp);
    FATAL ("Can't read file into memory %s", path);
  }
  if ((fread (*buffer, sizeof (uint8_t), *size, fp) != *size)) {
    fclose (fp);
    FATAL ("Can't read file %s", path);
  }
  fclose (fp);
}

/* Read input files and kick things off */
int run (const char *grammar_file, const char *input_file)
{
  Machine m;
  size_t grammar_size = 0, input_size = 0;
  Bytecode *grammar = NULL;
  char *input = NULL;

  readFile (grammar_file, &grammar, &grammar_size);
  readFile (input_file, (uint8_t **) &input, &input_size);

  mInit (&m, input, input_size);
  mRead (&m, grammar, grammar_size);
  mEval (&m);
  mFree (&m);

  free (grammar);
  free (input);
  return EXIT_SUCCESS;
}

/* Print out instructions on to how to use the program */
void usage (const char *program, const char *msg)
{
  if (msg) fprintf (stderr, "%s\n", msg);
  fprintf (stderr, "Usage: %s --grammar <GRAMMAR-FILE> --input <INPUT-FILE>\n", program);
  exit (0);
}

/* Read next command line argument. */
#define NEXT_OPT() (--argc, ++args)

/* Test if current command line argument matches short or long
   description. */
#define MATCH_OPT(short_desc,long_desc) \
  (argc > 0) && (strcmp (*args, short_desc) == 0 || strcmp (*args, long_desc) == 0)

#ifndef TEST

/* Temporary main function */
int main (int argc, char **argv)
{
  char **args = argv;

  /* Variables to keep command line provided values */
  char *grammar = NULL, *input = NULL;
  bool help = false;

  /* Read the command line options */
  while (argc > 0) {
    if (MATCH_OPT ("-g", "--grammar"))
      grammar = *NEXT_OPT ();
    if (MATCH_OPT ("-i", "--input"))
      input = *NEXT_OPT ();
    if (MATCH_OPT ("-h", "--help"))
      help = true;
    NEXT_OPT ();
  }

  /* User asked for help */
  if (help) {
    usage (argv[0], NULL);
    return EXIT_SUCCESS;
  }

  /* Validate values received from the command line */
  if (!grammar || !input) {
    usage (argv[0], "Both Grammar and Input file are required.");
    return EXIT_FAILURE;
  }

  /* Welcome to the machine */
  return run (grammar, input);
}

#else  /* TEST */

#include <assert.h>

/*
  s[i] = ‘c’
  -------------------
  match ‘c’ s i = i+1 (ch.1)
*/
static void test_ch1 ()
{
  Machine m;
  /* Start <- 'a' */
  Bytecode b[8] = {
    0x10, 0x0, 0x0, 0x61,     /* Char 'a' */
    0x00, 0x0, 0x0, 0x00,     /* Halt */
  };
  const char *o;
  DEBUG (" * t:ch.1 %s", "");

  mInit (&m, "a", 1);
  mRead (&m, b, 8);
  o = mEval (&m);
  mFree (&m);

  assert (o);
  assert (o - m.s == 1);        /* Match */
}

/*
  s[i] != ‘c’
  -------------------
  match ‘c’ s i = nil (ch.2)
 */
static void test_ch2 ()
{
  Machine m;
  /* Start <- 'a' */
  Bytecode b[8] = {
    0x10, 0x0, 0x0, 0x61,     /* Char 'a' */
    0x00, 0x0, 0x0, 0x00,     /* Halt */
  };
  DEBUG (" * t:ch.2 %s", "");

  mInit (&m, "x", 1);
  mRead (&m, b, 8);
  assert (!mEval (&m));         /* Failed */
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
  /* Start <- '.' */
  Bytecode b[8] = {
    0x20, 0x0, 0x0, 0x0,      /* Any */
    0x00, 0x0, 0x0, 0x0,      /* Halt */
  };
  const char *o;
  DEBUG (" * t:any.1 %s", "");

  mInit (&m, "a", 1);
  mRead (&m, b, 8);
  o = mEval (&m);
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - m.s == 1);        /* Match */
}

/*
  i > |s|
  -----------------
  match . s i = nil (any.2)
*/
void test_any2 ()
{
  Machine m;
  /* '.' */
  Bytecode b[8] = {
    0x20, 0x0, 0x0, 0x0,      /* Any */
    0x00, 0x0, 0x0, 0x0,      /* Halt */
  };
  DEBUG (" * t:any.2 %s", "");

  mInit (&m, "", 0);
  mRead (&m, b, 8);
  assert (!mEval (&m));         /* Failed */
  mFree (&m);
}

/*
  match p s i = nil
  -----------------
  match !p s i = i (not.1)
*/
void test_not1 ()
{
  Machine m;
  /* !'a' */
  Bytecode b[20] = {
    0x30, 0x0, 0x0, 0x04, /* Choice 0x04 */
    0x10, 0x0, 0x0, 0x61, /* Char 'a' */
    0x40, 0x0, 0x0, 0x01, /* Commit 1 */
    0x50, 0x0, 0x0, 0x00, /* Fail */
    0x00, 0x0, 0x0, 0x00, /* Halt */
  };
  const char *o;
  DEBUG (" * t:not.1 %s", "");

  mInit (&m, "b", 0);
  mRead (&m, b, 20);
  o = mEval (&m);
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - m.s == 0);        /* But didn't match anything */
}

void test_not1_fail_twice ()
{
  Machine m;
  /* !'a' */
  Bytecode b[16] = {
    0x30, 0x0, 0x0, 0x05, /* Choice 0x05 */
    0x10, 0x0, 0x0, 0x61, /* Char 'a' */
    0x60, 0x0, 0x0, 0x00, /* FailTwice */
    0x00, 0x0, 0x0, 0x00, /* Halt */
  };
  const char *o;
  printf (" * t:not.1 fail-twice\n");

  mInit (&m, "b", 0);
  mRead (&m, b, 16);
  o = mEval (&m);
  mFree (&m);

  assert (o);                   /* Did not fail */
  assert (o - m.s == 0);        /* But didn't match any char */
}

/*
  match p s i = i+j
  ------------------
  match !p s i = nil (not.2)
*/
void test_not2 ()
{
  Machine m;
  /* !'a' */
  Bytecode b[20] = {
    0x30, 0x0, 0x0, 0x04, /* Choice 0x04 */
    0x10, 0x0, 0x0, 0x61, /* Char 'a' */
    0x40, 0x0, 0x0, 0x01, /* Commit 1 */
    0x50, 0x0, 0x0, 0x00, /* Fail */
    0x00, 0x0, 0x0, 0x00, /* Halt */
  };
  DEBUG (" * t:not.2 %s", "");

  mInit (&m, "a", 0);
  mRead (&m, b, 20);
  assert (!mEval (&m));         /* Failed */
  mFree (&m);
}

void test_not2_fail_twice ()
{
  Machine m;
  /* !'a' */
  Bytecode b[16] = {
    0x30, 0x0, 0x0, 0x05, /* Choice 0x05 */
    0x10, 0x0, 0x0, 0x61, /* Char 'a' */
    0x60, 0x0, 0x0, 0x00, /* FailTwice */
    0x00, 0x0, 0x0, 0x00, /* Halt */
  };
  DEBUG (" * t:not.2 fail-twice %s", "");

  mInit (&m, "a", 0);
  mRead (&m, b, 16);
  assert (!mEval (&m));         /* Failed */
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
  /* &'a' */
  Bytecode b[32] = {
    0x30, 0x0, 0x0, 0x07, /* Choice 0x07 */
    0x30, 0x0, 0x0, 0x04, /* Choice 0x04 */
    0x10, 0x0, 0x0, 0x61, /* Char 'a' */
    0x40, 0x0, 0x0, 0x01, /* Commit 1 */
    0x50, 0x0, 0x0, 0x00, /* Fail */
    0x40, 0x0, 0x0, 0x01, /* Commit 1 */
    0x50, 0x0, 0x0, 0x00, /* Fail */
    0x00, 0x0, 0x0, 0x00, /* Halt */
  };
  const char *o;
  DEBUG (" * t:and.%s", "1");

  mInit (&m, "a", 1);
  mRead (&m, b, 32);
  o = mEval (&m);
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - m.s == 0);        /* But didn't match anything */
}

/*
match g p s i = nil
--------------------
match g &p s i = nil (and.2)
*/
void test_and2 ()
{
  Machine m;
  /* &'a' */
  Bytecode b[32] = {
    0x30, 0x0, 0x0, 0x07, /* Choice 0x07 */
    0x30, 0x0, 0x0, 0x04, /* Choice 0x04 */
    0x10, 0x0, 0x0, 0x61, /* Char 'a' */
    0x40, 0x0, 0x0, 0x01, /* Commit 1 */
    0x50, 0x0, 0x0, 0x00, /* Fail */
    0x40, 0x0, 0x0, 0x01, /* Commit 1 */
    0x50, 0x0, 0x0, 0x00, /* Fail */
    0x00, 0x0, 0x0, 0x00, /* Halt */
  };
  DEBUG (" * t:and.%s", "2");

  mInit (&m, "b", 0);
  mRead (&m, b, 32);
  assert (!mEval (&m));         /* Failed */
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
  /* 'a' '.' 'c' */
  Bytecode b[16] = {
    0x10, 0x0, 0x0, 0x61, /* Char 'a' */
    0x20, 0x0, 0x0, 0x00, /* Any */
    0x10, 0x0, 0x0, 0x63, /* Char 'c' */
    0x00, 0x0, 0x0, 0x00, /* Halt */
  };
  const char *o;

  DEBUG (" * t:con.1 %s", "");

  mInit (&m, "abc", 3);
  mRead (&m, b, 16);
  o = mEval (&m);
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - m.s == 3);        /* Matched all 3 chars */
}

/*
  match p1 s i = i+j    match p2 s i + j = nil
  ----------------------------------------------
         match p1 p2 s i = nil (con.2)
 */
void test_con2 ()
{
  Machine m;
  /* 'a' 'c' '.' */
  Bytecode b[16] = {
    0x10, 0x0, 0x0, 0x61, /* Char 'a' */
    0x10, 0x0, 0x0, 0x63, /* Char 'c' */
    0x20, 0x0, 0x0, 0x00, /* Any */
    0x00, 0x0, 0x0, 0x00, /* Halt */
  };
  DEBUG (" * t:con.2 %s", "");

  mInit (&m, "abc", 3);
  mRead (&m, b, 16);
  assert (!mEval (&m));         /* Failed */
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
  /* 'a' 'c' '.' */
  Bytecode b[16] = {
    0x10, 0x0, 0x0, 0x61, /* Char 'a' */
    0x10, 0x0, 0x0, 0x63, /* Char 'c' */
    0x20, 0x0, 0x0, 0x00, /* Any */
    0x00, 0x0, 0x0, 0x00, /* Halt */
  };
  DEBUG (" * t:con.3 %s", "");

  mInit (&m, "cba", 3);
  mRead (&m, b, 16);
  assert (!mEval (&m));         /* Failed */
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
  /* 'a' / 'b' */
  Bytecode b[20] = {
    0x30, 0x0, 0x0, 0x03, /* Choice 0x03 */
    0x10, 0x0, 0x0, 0x61, /* Char 'a' */
    0x40, 0x0, 0x0, 0x03, /* Commit 0x03 */
    0x10, 0x0, 0x0, 0x62, /* Char 'b' */
    0x00, 0x0, 0x0, 0x00, /* Halt */
  };
  DEBUG (" * t:ord.1 %s", "");

  mInit (&m, "c", 1);
  mRead (&m, b, 20);
  assert (!mEval (&m));         /* Failed */
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
  /* 'a' / 'b' */
  Bytecode b[20] = {
    0x30, 0x0, 0x0, 0x03, /* Choice 0x03 */
    0x10, 0x0, 0x0, 0x61, /* Char 'a' */
    0x40, 0x0, 0x0, 0x03, /* Commit 0x03 */
    0x10, 0x0, 0x0, 0x62, /* Char 'b' */
    0x00, 0x0, 0x0, 0x00, /* Halt */
  };
  const char *o;
  DEBUG (" * t:ord.2 %s", "");

  mInit (&m, "a", 1);
  mRead (&m, b, 20);
  o = mEval (&m);
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - m.s == 1);        /* Match the first char */
}

/*
  match p1 s i = nil    match p2 s i = i+k
  ----------------------------------------
      match p1 / p2 s i = i+k (ord.3)
 */
void test_ord3 ()
{
  Machine m;
  /* 'a' / 'b' */
  Bytecode b[20] = {
    0x30, 0x0, 0x0, 0x03, /* Choice 0x03 */
    0x10, 0x0, 0x0, 0x61, /* Char 'a' */
    0x40, 0x0, 0x0, 0x03, /* Commit 0x03 */
    0x10, 0x0, 0x0, 0x62, /* Char 'b' */
    0x00, 0x0, 0x0, 0x00, /* Halt */
  };
  const char *o;
  DEBUG (" * t:ord.3 %s", "");

  mInit (&m, "b", 1);
  mRead (&m, b, 20);
  o = mEval (&m);
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - m.s == 1);        /* Match the first char */
}

/*
  match p s i = i+j    match p∗ s i + j = i+j+k
  ----------------------------------------------
          match p∗ s i = i+j+k (rep.1)
*/
void test_rep1 ()
{
  Machine m;
  /* 'a*' */
  Bytecode b[16] = {
    0x30, 0x00, 0x00, 0x03, /* Choice 0x03 */
    0x10, 0x00, 0x00, 0x61, /* Char 'a' */
    0x4f, 0xff, 0xff, 0xfe, /* Commit 0xffe (-2) */
    0x00, 0x00, 0x00, 0x00, /* Halt */
  };
  const char *o;
  DEBUG (" * t:rep.1 %s", "");

  mInit (&m, "aab", 1);
  mRead (&m, b, 16);
  o = mEval (&m);
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - m.s == 2);        /* Matched two chars */
}

void test_rep1_partial_commit ()
{
  Machine m;
  /* 'a*' */
  Bytecode b[16] = {
    0x30, 0x00, 0x00, 0x03, /* Choice 0x03 */
    0x10, 0x00, 0x00, 0x61, /* Char 'a' */
    0x7f, 0xff, 0xff, 0xff, /* PartialCommit 0xffff (-1) */
    0x00, 0x00, 0x00, 0x00, /* Halt */
  };
  const char *o;
  DEBUG (" * t:rep.1 %s", "(partial-commit)");

  mInit (&m, "aab", 1);
  mRead (&m, b, 16);
  o = mEval (&m);
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - m.s == 2);        /* Matched two chars */
}

/*
  match p s i = nil
  -----------------
  match p∗ s i = i (rep.2)
*/
void test_rep2 ()
{
  Machine m;
  /* 'a*' */
  Bytecode b[16] = {
    0x30, 0x00, 0x00, 0x03, /* Choice 0x03 */
    0x10, 0x00, 0x00, 0x61, /* Char 'a' */
    0x4f, 0xff, 0xff, 0xfe, /* Commit 0xffe (-2) */
    0x00, 0x00, 0x00, 0x00, /* Halt */
  };
  const char *o;
  DEBUG (" * t:rep.2 %s", "");

  mInit (&m, "b", 1);
  mRead (&m, b, 16);
  o = mEval (&m);
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - m.s == 0);        /* But didn't match any char */
}

void test_rep2_partial_commit ()
{
  Machine m;
  /* 'a*' */
  Bytecode b[16] = {
    0x30, 0x00, 0x00, 0x03, /* Choice 0x03 */
    0x10, 0x00, 0x00, 0x61, /* Char 'a' */
    0x7f, 0xff, 0xff, 0xff, /* PartialCommit 0xffff (-1) */
    0x00, 0x00, 0x00, 0x00, /* Halt */
  };
  const char *o;
  DEBUG (" * t:rep.2 %s", "(partial-commit)");

  mInit (&m, "b", 1);
  mRead (&m, b, 16);
  o = mEval (&m);
  mFree (&m);

  assert (o);                   /* Didn't fail */
  assert (o - m.s == 0);        /* But didn't match any char */
}

int main ()
{
  test_ch1 ();
  test_ch2 ();
  test_any1 ();
  test_any2 ();
  test_not1 ();
  test_not1_fail_twice ();
  test_not2 ();
  test_not2_fail_twice ();
  test_and1 ();
  test_and2 ();
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
  return 0;
}

#endif  /* TEST */
