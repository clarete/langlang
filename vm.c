#include <stdint.h>
#include <stdio.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>

/*

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

/** Advance m->pc one step and return last byte read */
#define READ8C(m) (*m->pc++)

/** Move m->pc two bytes ahead and read them into an uint16_t */
#define READ16C(m) (m->pc += 2, (uint16_t) ((m->pc[-2] << 8 | m->pc[-1])))

/** Retrieve name of opcode */
#define OP_NAME(o) opNames[o]

/** Report errors that stop the execution of the VM right away */
#define FATAL(f, ...)                                                  \
  do { fprintf (stderr, f "\n", ##__VA_ARGS__); exit (EXIT_FAILURE); } \
  while (0)

/* opcodes */
typedef enum {
  OP_CHAR = 0x1,
  OP_END,
} Instructions;

/* The code is represented as a list of instructions */
typedef uint8_t Bytecode;

/* Virtual Machine */
typedef struct {
  const char *s;
  long i;
  Bytecode *pc;
} Machine;

static const char *opNames[OP_END] = {
  [OP_CHAR] = "OP_CHAR",
};

/* Set initial values for the machine */
void mInit (Machine *m, Bytecode *code, const char *input)
{
  m->pc = code;
  m->s = input;
  m->i = 0;
}

/* Run the matching machine */
void mEval (Machine *m)
{
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
    case OP_CHAR:
      printf ("OP_CHAR: %c (%x)\n", operand, operand);
      if (m->s[m->i] == operand) m->i++;
      else m->i = -1;
      break;
    default: FATAL ("Unknown Instruction 0x%04x", opcode);
    }
    break;
  }
}

/* Reads the entire content of the file under `path' into `buffer' */
void read_file (const char *path, void **buffer, size_t *size)
{
  FILE *fp = fopen (path, "rb");
  if (!fp) FATAL ("Can't open file %s", path);
  /* Read file size */
  fseek (fp, 0, SEEK_END);
  *size = ftell (fp);
  rewind (fp);
  /* Allocate buffer and read the file into it */
  if ((*buffer = malloc (*size)) == NULL) {
    fclose (fp);
    FATAL ("Can't read file into memory %s", path);
  }
  if ((fread (*buffer, 1, *size, fp) != *size)) {
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
  uint8_t *input = NULL;

  read_file (grammar_file, (void *) &grammar, &grammar_size);
  read_file (input_file, (void *) &input, &input_size);

  mInit (&m, grammar, (const char *) input);
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
