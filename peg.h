/* -*- coding: utf-8; -*-
 *
 * peg.h - Implementation of Parsing Machine for PEGs
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

#include "error.h"
#include "value.h"

/** Arbitrary value for the stack size */
#define STACK_SIZE 512

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
  unsigned short rator: 5;
  uint32_t rand: 27;
} Instruction;

typedef enum {
  CapOpen,
  CapClose,
} CaptureType;

typedef struct {
  CaptureType type;
  const char *pos;
  uint16_t idx;
  uint16_t term;
} CaptureEntry;

/* Entry that's stored in the Machine's stack for supporting backtrack
   on the ordered choice operator */
typedef struct {
  const char *i;
  Object *l;
  Instruction *pc;
  uint32_t cap;
} BacktrackEntry;

/* Virtual Machine */
typedef struct {
  Instruction *code;
  BacktrackEntry *stack;
  CaptureEntry *captures;
  uint32_t cap;                 /* Top of the capture stack */
  ObjectTable symbols;          /* Store unique symbols within the VM */
  const char *li;               /* Last `i' seen when backtraking */
} Machine;

/* opcodes */
typedef enum {
  OP_HALT = 0x0,
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
  OP_SET,
  OP_CAP_OPEN,
  OP_CAP_CLOSE,
  OP_ATOM,
  OP_OPEN,
  OP_CLOSE,
  OP_END,
} OpCode;

/* Initialize the state of the machine. */
void mInit (Machine *m);
/* Free allocated resources. */
void mFree (Machine *m);
/* Load bytecode into the machine. */
void mLoad (Machine *m, Bytecode *code, size_t code_size);
/* Create a new symbol & store it within the machine's symbol table */
Object *mSymbol (Machine *m, const char *sym, size_t len);
/* Try to match input against the pattern loaded into the machine. */
const char *mMatch (Machine *m, const char *input, size_t input_size);
/* Try to match input list against pattern loaded into the VM */
Object *mMatchList (Machine *m, Object *input);
/* Extract matches from the machine's capture stack */
Object *mExtract (Machine *m, const char *input);
/* Run grammar file on input file and extract output */
Object *mRunFile (Machine *m, const char *grammar_file, const char *input_file);

#endif  /* VM_GUARD */
