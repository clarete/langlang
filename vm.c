#include <stdint.h>
#include <stdio.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>

/*

Machine Instructions
====================

* [x] Char C
* [x] Any
* [x] Choice L
* [x] Commit L
* [x] Fail
* [ ] Jump L
* [ ] Call L
* [ ] Return

Instruction Format
==================

Each instruction is 16 bit long. The first 4 bits are reserved for the
opcode and the other 12 bits can be used to store parameters for the
instruction:

  The instruction "Char 'a'" will be represented in the following
  format:

    opcode  | parameter
    4bits   | 12bits
    --------|------------------------
    |0|0|0|1|0|0|0|0|0|1|1|0|0|0|0|1|
    ---------------------------------

Bytecode Format
===============

The Bytecode object is a sequence of instructions.

      16bits  16bits  16bits
    ----|-------|-------|----
    | Inst1 | Inst2 | InstN |
    -------------------------
 */

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
  OP_END,
} Instructions;

/* The code is represented as a list of instructions */
typedef uint8_t Bytecode;

/* Entry that's stored in the Machine's stack for supporting backtrack
   on the ordered choice operator */
typedef struct {
  unsigned long i;
  Bytecode *pc;
} BacktrackEntry;

/* Virtual Machine */
typedef struct {
  const char *s;
  size_t s_size;
  unsigned long i;
  Bytecode *pc;
  BacktrackEntry stack[STACK_SIZE];
  bool fail;
} Machine;

/* Helps debugging */
static const char *opNames[OP_END] = {
  [OP_CHAR] = "OP_CHAR",
  [OP_ANY] = "OP_ANY",
  [OP_CHOICE] = "OP_CHOICE",
  [OP_COMMIT] = "OP_COMMIT",
  [OP_FAIL] = "OP_FAIL",
};

/* Create a new backtrack entry */
BacktrackEntry newBacktrackEntry (unsigned long i, Bytecode *pc)
{
  BacktrackEntry b = { i, pc };
  return b;
}

/* Set initial values for the machine */
void mInit (Machine *m, Bytecode *code, const char *input, size_t input_size)
{
  memset (m->stack, 0, STACK_SIZE * sizeof (void *));
  m->pc = code;
  m->s = input;
  m->s_size = input_size;
  m->i = 0;
  m->fail = false;
}

/* Run the matching machine */
void mEval (Machine *m)
{
  /** Advance m->pc one step and return last byte read */
#define READ8C(m) (*m->pc++)
  /** Move m->pc two bytes ahead and read them into an uint16_t */
#define READ16C(m) (m->pc += 2, (uint16_t) ((m->pc[-2] << 8 | m->pc[-1])))
  /** Push data onto the machine's stack  */
#define PUSH(d) ((*sp++) = d)
  /** Pop data from the machine's stack  */
#define POP() (--sp)

  BacktrackEntry bt, *sp = m->stack;
  uint16_t instruction, operand;
  short opcode;

  while (true) {
    /* Fetch instruction */
    instruction = READ16C (m);

    /* Decode opcode & operand */
    opcode = (instruction & 0xF000) >> 12;
    operand = instruction & 0x0FFF;

    /* Execute instruction */
    switch (opcode) {
    case 0: return;
    case OP_FAIL:
      m->fail = true;
      break;
    case OP_CHAR:
      if (m->s[m->i] == operand) m->i++;
      else m->fail = true;
      break;
    case OP_ANY:
      if (m->i < m->s_size) m->i++;
      else m->fail = true;
      break;
    case OP_CHOICE:
      PUSH (newBacktrackEntry (m->i, m->pc + operand));
      break;
    case OP_COMMIT:
      if (!m->fail) {
        POP ();            /* Discard backtrack entry */
        m->pc += operand;  /* Jump to the given position */
      }
      break;
    default:
      FATAL ("Unknown Instruction 0x%04x", opcode);
    }

    if (m->fail) {
      if (sp > m->stack) {
        /* Fail〈(pc,i1):e〉 ----> 〈pc,i1,e〉 */
        bt = *POP ();
        m->i = bt.i;
        m->pc = bt.pc;
        m->fail = false;        /* We're backtracking */
      } else {
        /* 〈pc,i,e〉 ----> Fail〈e〉 */
        break;
      }
    }
  }
}

/* Reads the entire content of the file under `path' into `buffer' */
void read_file (const char *path, uint8_t **buffer, size_t *size)
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

  read_file (grammar_file, &grammar, &grammar_size);
  read_file (input_file, (uint8_t **) &input, &input_size);

  mInit (&m, grammar, input, input_size);
  mEval (&m);

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
  match ‘c’ s i = i+1 (ch.1)
*/
static void test_ch1 ()
{
  Machine m;
  /* Start <- 'a' */
  Bytecode b[4] = { 0x0010, 0x0061, 0, 0 }; /* Char 'a' */
  printf (" * t:ch.1\n");
  mInit (&m, b, "a", 1);
  mEval (&m);
  assert (!m.fail);
  assert (m.i == 1);
}

/*
  s[i] 6 = ‘c’
  -------------------
  match ‘c’ s i = nil (ch.2)
 */
static void test_ch2 ()
{
  Machine m;
  /* Start <- 'a' */
  Bytecode b[4] = { 0x0010, 0x0061, 0, 0 }; /* Char 'a' */
  printf (" * t:ch.2\n");
  mInit (&m, b, "x", 1);
  mEval (&m);
  assert (m.fail);
  assert (m.i == 0);
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
  Bytecode b[4] = { 0x0020, 0x0000, 0, 0 }; /* Any */
  printf (" * t:any.1\n");
  mInit (&m, b, "a", 1);
  mEval (&m);
  assert (!m.fail);
  assert (m.i == 1);
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
  Bytecode b[4] = { 0x0020, 0x0000, 0, 0 }; /* Any */
  printf (" * t:any.2\n");
  mInit (&m, b, "", 0);
  mEval (&m);
  assert (m.fail);
  assert (m.i == 0);
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
  Bytecode b[10] = {
    0x0030, 0x0005, /* Choice 0x0005 */
    0x0010, 0x0061, /* Char 'a' */
    0x0040, 0x0001, /* Commit 1 */
    0x0050, 0x0000, /* Fail */
    0, 0
  };
  printf (" * t:not.1\n");
  mInit (&m, b, "b", 0);
  mEval (&m);
  assert (!m.fail);
  assert (m.i == 0);
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
  Bytecode b[10] = {
    0x0030, 0x0005, /* Choice 0x0005 */
    0x0010, 0x0061, /* Char 'a' */
    0x0040, 0x0000, /* Commit 1 */
    0x0050, 0x0000, /* Fail */
    0, 0
  };
  printf (" * t:not.2\n");
  mInit (&m, b, "a", 0);
  mEval (&m);
  printf (" * FOOO: %ld\n", m.i);
  assert (m.fail);
  /* assert (m.i == 0); */
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
  Bytecode b[8] = {
    0x0010, 0x0061, /* Char 'a' */
    0x0020, 0x0000, /* Any */
    0x0010, 0x0063, /* Char 'c' */
    0, 0,
  };
  printf (" * t:con.1\n");
  mInit (&m, b, "abc", 3);
  mEval (&m);
  assert (!m.fail);
  assert (m.i == 3);
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
  Bytecode b[8] = {
    0x0010, 0x0061, /* Char 'a' */
    0x0010, 0x0063, /* Char 'c' */
    0x0020, 0x0000, /* Any */
    0, 0,
  };
  printf (" * t:con.2\n");
  mInit (&m, b, "abc", 3);
  mEval (&m);
  assert (m.fail);
  assert (m.i == 1);
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
  Bytecode b[8] = {
    0x0010, 0x0061, /* Char 'a' */
    0x0010, 0x0063, /* Char 'c' */
    0x0020, 0x0000, /* Any */
    0, 0,
  };
  printf (" * t:con.3\n");
  mInit (&m, b, "cba", 3);
  mEval (&m);
  assert (m.fail);
  assert (m.i == 0);
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
  Bytecode b[10] = {
    0x0030, 0x0004, /* Choice 0x0004 */
    0x0010, 0x0061, /* Char 'a' */
    0x0040, 0x0002, /* Commit 0x0003 */
    0x0010, 0x0062, /* Char 'b' */
    0, 0,
  };
  printf (" * t:ord.1\n");
  mInit (&m, b, "c", 1);
  mEval (&m);
  assert (m.fail);
  assert (m.i == 0);
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
  Bytecode b[10] = {
    0x0030, 0x0004, /* Choice 0x0004 */
    0x0010, 0x0061, /* Char 'a' */
    0x0040, 0x0002, /* Commit 0x0003 */
    0x0010, 0x0062, /* Char 'b' */
    0, 0,
  };
  printf (" * t:ord.2\n");
  mInit (&m, b, "a", 1);
  mEval (&m);
  assert (m.i == 1);
  /* TODO: Ensure that stack is empty */
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
  Bytecode b[10] = {
    0x0030, 0x0004, /* Choice 0x0004 */
    0x0010, 0x0061, /* Char 'a' */
    0x0040, 0x0002, /* Commit 0x0003 */
    0x0010, 0x0062, /* Char 'b' */
    0, 0,
  };
  printf (" * t:ord.3\n");
  mInit (&m, b, "b", 1);
  mEval (&m);
  assert (m.i == 1);
}

int main ()
{
  test_ch1 ();
  test_ch2 ();
  test_any1 ();
  test_any2 ();
  test_not1 ();
  test_not2 ();
  test_con1 ();
  test_con2 ();
  test_con3 ();
  test_ord1 ();
  test_ord2 ();
  test_ord3 ();
  return 0;
}

#endif  /* TEST */
