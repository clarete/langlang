/* -*- coding: utf-8; -*-
 *
 * peg.h - Implementation of Parsing Machine for PEGs
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

/** Clear all 27bits from the right then shift to the right */
#define OP_MASK(c) (((c) & 0xf8000000) >> OPERATOR_OFFSET)
/** Clear the operator from the instruction */
#define RN_MASK(c) ((c) & 0x7ffffff)

/** Read instruction operands */
#define UOPERAND0(r) ((r)->u32)
#define UOPERAND1(r) ((r)->u.r1)
#define UOPERAND2(r) ((r)->u.r2)
#define SOPERAND0(r) ((r)->s32)
#define SOPERAND1(r) ((r)->s.r1)
#define SOPERAND2(r) ((r)->s.r2)

/* Default error handling. Labels above 1 are user defined  */
#define PEG_SUCCESS 0
#define PEG_FAILURE 1

/* Binary files are read into variables of this type */
typedef uint8_t Bytecode;

/* Instruction following the format of 4b operator and 12b operand */
typedef struct {
  unsigned short rator: 5;
  union {
    uint32_t u32: 27;
    int32_t s32: 27;
    struct {
      uint32_t r2: 16;
      uint32_t r1: 11;
    } u;
    struct {
      int32_t r2: 16;
      int32_t r1: 11;
    } s;
  };
} Instruction;

/* Entry that's stored in the Machine's stack for supporting backtrack
   on the ordered choice operator */
typedef struct {
  const char *i;
  Value *l;
  Instruction *pc;
  uint32_t btCount;
  uint32_t ltCount;
} BacktrackEntry;

/* Virtual Machine */
typedef struct {
  Instruction *code;
  BacktrackEntry *stack;
  List symbols;               /* Store unique symbols within the VM */
  const char *i;              /* Last `i' seen on success */
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
  OP_THROW,
  OP_CAP_OPEN,
  OP_CAP_CLOSE,
  OP_ATOM,
  OP_OPEN,
  OP_CLOSE,
  OP_CAPCHAR,
  OP_END,
} OpCode;

/* Initialize the state of the machine. */
void mInit (Machine *m);
/* Free allocated resources. */
void mFree (Machine *m);
/* Load bytecode into the machine. */
void mLoad (Machine *m, Bytecode *code);
/* Create a new symbol & store it within the machine's symbol table */
Value *mSymbol (Machine *m, const char *sym, size_t len);
/* Try to match input against the pattern loaded into the machine. */
uint32_t mMatch (Machine *m, const char *input, size_t input_size, Value **out);
/* Try to match input list against pattern loaded into the VM */
Value *mMatchList (Machine *m, Value *input);
/* Run grammar file on input file and extract output */
Value *mRunFile (Machine *m, const char *grammar_file, const char *input_file);

#endif  /* VM_GUARD */
