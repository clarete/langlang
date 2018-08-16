/* -*- coding: utf-8; -*-
 *
 * vm.h - Implementation of Parsing Machine for PEGs
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
#ifndef VM_GUARD
#define VM_GUARD

#include <stdint.h>
#include <stdlib.h>

/** Arbitrary value for the stack size */
#define STACK_SIZE 512

/** Report errors that stop the execution of the VM right away */
#define FATAL(f, ...)                                                  \
  do { fprintf (stderr, f "\n", ##__VA_ARGS__); exit (EXIT_FAILURE); } \
  while (0)


/* Instruction Offsets - All sizes are in bits */
#define INSTRUCTION_SIZE   32   /* Instruction size */
#define OPERATOR_SIZE      5    /* Operator size */
#define OPERATOR_OFFSET    (INSTRUCTION_SIZE - OPERATOR_SIZE)
#define SL_OPERAND_SIZE    OPERATOR_OFFSET /* 27b */
#define S1_OPERAND_SIZE    11
#define S2_OPERAND_SIZE    16

/** Clear all 28bits from the right then shift to the right */
#define OP_MASK(c) (((c) & 0xf8000000) >> OPERATOR_OFFSET)

/** Read unsigned single operand */
#define UOPERAND0(op) (op->rand & 0x7ffffff)
#define UOPERAND1(op) ((op->rand & 0x7ffffff) >> S2_OPERAND_SIZE)
#define UOPERAND2(op) (op->rand & ((1 << S2_OPERAND_SIZE) - 1))
/** Read signed values */
#define SIGNED(i,s) ((int32_t) ((i & (1 << (s - 1))) ? (i | ~((1 << s) - 1)) : (i & 0x7ffffff)))
/** Read single operand from instruction */
#define SOPERAND0(op) SIGNED (op->rand, SL_OPERAND_SIZE)
/* #define SOPERAND1(op) SIGNED (op->rand >> SS_OPERAND_SIZE, SS_OPERAND_SIZE) */
/* #define SOPERAND2(op) SIGNED (op->rand, SS_OPERAND_SIZE) */

/* Binary files are read into variables of this type */
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
  Instruction *code;
  BacktrackEntry stack[STACK_SIZE];
} Machine;

/* opcodes */
typedef enum {
  OP_CHAR = 0x1,
  OP_ANY,
  OP_CHOICE,
  OP_COMMIT,
  OP_FAIL,
  OP_FAIL_TWICE,
  OP_PARTIAL_COMMIT,
  OP_BACK_COMMIT,
  OP_TEST_CHAR,
  OP_TEST_ANY,
  OP_JUMP,
  OP_CALL,
  OP_RETURN,
  OP_SPAN,
  OP_HALT,
} OpCode;

/* Initialize the state of the machine. */
void mInit (Machine *m);
/* Free allocated resources. */
void mFree (Machine *m);
/* Load bytecode into the machine. */
void mLoad (Machine *m, Bytecode *code, size_t code_size);
/* Try to match input against the pattern loaded into the machine. */
const char *mMatch (Machine *m, const char *input, size_t input_size);

#endif  /* VM_GUARD */
