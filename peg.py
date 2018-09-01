# -*- coding: utf-8; -*-
#
# peg.py - Parsing Expression Grammar implementation
#
# Copyright (C) 2018  Lincoln Clarete
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.

from __future__ import print_function

import argparse
import functools
import io
import os
import pprint
import struct

# Python 2 & 3 support
try:
    import enum
except ImportError:
    import aenum as enum


class TokenTypes(enum.Enum):
    (IDENTIFIER,
     LITERAL,
     ARROW,
     PLUS,
     STAR,
     PRIORITY,
     QUESTION,
     AND,
     DOT,
     NOT,
     CLASS,
     OPEN,
     CLOSE,
     END,
    ) = range(14)

class Token:
    def __init__(self, _type, value=None, line=0, pos=0):
        self._type = _type
        self.value = value
        self.line = line
        self.pos = pos
    def __repr__(self):
        value = ", " + repr(self.value) if self.value else ""
        value += ", line=%d" % self.line
        value += ", pos=%d" % self.pos
        return "Token({}{})".format(self._type, value)
    def __eq__(self, other):
        return (isinstance(other, self.__class__) and
                other._type == self._type and
                other.value == self.value and
                other.line == self.line and
                other.pos == self.pos)

class Node:
    def __init__(self, value=None):
        self.value = value
    def __repr__(self):
        value = "{}".format(repr(self.value)) if self.value else ''
        return "{}({})".format(self.__class__.__name__, value)
    def __eq__(self, other):
        return (isinstance(other, self.__class__) and
                other.value == self.value)

class And(Node): pass

class Not(Node): pass

class Question(Node): pass

class Star(Node): pass

class Plus(Node): pass

class Expression(Node): pass

class Sequence(Node): pass

class Identifier(Node): pass

class Literal(Node): pass

class Class(Node): pass

class Dot(Node): pass

def fio(thing):
    "first if only"
    if isinstance(thing, list) and len(thing) == 1:
        return thing[0]
    return thing

class FormattedException(Exception): pass

class Parser:

    def __init__(self, code):
        self.code = code
        self.pos = 0
        self.line = 0
        self.token_start = 0
        self.token = None

    def peekc(self, n=0):
        if self.pos+n >= len(self.code): return None
        return self.code[self.pos+n]

    def nextc(self, n=1):
        self.pos += n
        return self.peekc()

    def testc(self, c, n=0):
        return self.peekc(n) == c

    def matchc(self, c):
        if not self.testc(c): return False
        self.nextc()
        return True

    def peekt(self):
        d = self.pos
        value = self.lex()
        self.pos = d
        return value

    def nextt(self):
        self.token = self.lex()
        return self.token

    def testt(self, t):
        return self.token._type == t

    def matcht(self, t):
        if not self.testt(t): return False
        value = self.token; self.nextt()
        return value

    def consumet(self, t):
        value = self.matcht(t)
        if value: return value
        raise SyntaxError("Expected %s but found %s" % (
            t.name, self.token._type.name))

    def t(self, _type, value=None):
        return Token(_type, value, line=self.line, pos=self.token_start)

    def lex(self):
        self.spacing()
        self.token_start = self.pos
        if self.peekc() is None: return self.t(TokenTypes.END)

        # Identifier <- IdentStart IdentCont* Spacing
        if self.peekc().isalpha():
            d = self.pos
            while self.peekc() and self.peekc().isalnum():
                self.pos += 1
            return self.t(TokenTypes.IDENTIFIER, self.code[d:self.pos])
        # Literal <- ['] (!['] Char)* ['] Spacing
        elif self.matchc("'"):
            return self.lexLiteral("'")
        # / ["] (!["] Char)* ["] Spacing
        elif self.matchc('"'):
            return self.lexLiteral('"')
        # Range <- ’[’ (!’]’ Range)* ’]’ Spacing
        elif self.matchc('['):
            return self.lexRange()
        # LEFTARROW ’<-’
        elif self.matchc('<'):
            if self.matchc('-'): return self.t(TokenTypes.ARROW)
            else: raise SyntaxError("Missing the dash in the arrow")
        elif self.matchc('('):
            return self.t(TokenTypes.OPEN)
        elif self.matchc(')'):
            return self.t(TokenTypes.CLOSE)
        elif self.matchc('/'):
            return self.t(TokenTypes.PRIORITY)
        elif self.matchc('.'):
            return self.t(TokenTypes.DOT)
        elif self.matchc('*'):
            return self.t(TokenTypes.STAR)
        elif self.matchc('+'):
            return self.t(TokenTypes.PLUS)
        elif self.matchc('!'):
            return self.t(TokenTypes.NOT)
        elif self.matchc('&'):
            return self.t(TokenTypes.AND)
        elif self.matchc('?'):
            return self.t(TokenTypes.QUESTION)
        else:
            raise SyntaxError("Unexpected char `{}'".format(self.peekc()))

    def lexRange(self):
        ranges = []
        chars = []
        # Range <- Char ’-’ Char / Char
        while not self.testc(']'):
            left = self.lexChar()
            if self.matchc('-'): ranges.append([left, self.lexChar()])
            else: chars.append(left)
        if not self.matchc(']'): raise SyntaxError("Expected end of class")
        return self.t(TokenTypes.CLASS, ranges + chars)

    def lexChar(self):
        # Char <- '\\' [nrt'"\[\]\\]
        if self.matchc('\\'):
            if self.matchc('n'): return '\n'
            elif self.matchc('r'): return '\r'
            elif self.matchc('t'): return '\t'
            elif self.matchc('\''): return '\''
            elif self.matchc('"'): return '"'
            elif self.matchc('['): return '['
            elif self.matchc(']'): return ']'
            elif self.matchc('\\'): return '\\'
            elif self.peekc() is None: SyntaxError('Unexpected end of input')
            else: raise SyntaxError('Unknown escape char `{}`'.format(self.peekc()))
        value = self.peekc()
        self.nextc()
        return value

    def lexLiteral(self, end):
        output = []
        while self.peekc() and not self.testc(end):
            output.append(self.lexChar())
        if not self.matchc(end): raise SyntaxError("Expected end of string")
        return self.t(TokenTypes.LITERAL, ''.join(output))

    def spacing(self):
        self.cleanspaces()
        # ’#’ (!EndOfLine .)* EndOfLine
        while self.matchc('#'):
            while not self.testc('\n') and self.peekc():
                self.nextc()
            self.cleanspaces()

    def cleanspaces(self):
        while True:
            if self.peekc() and self.peekc().isspace():
                if self.matchc('\n'): self.line += 1
                else: self.nextc()
            else: break

    def parseDefinitions(self):
        # Grammar <- Spacing Definition+ EndOfFile
        definitions = []
        while not self.testt(TokenTypes.END):
            definition = self.parseDefinition()
            if not definition: break
            definitions.append(definition)
        return definitions

    def parseDefinition(self):
        # Definition <- Identifier LEFTARROW Expression
        identifier = self.consumet(TokenTypes.IDENTIFIER)
        self.consumet(TokenTypes.ARROW)
        return {identifier.value: self.parseExpression()}

    def parseExpression(self):
        # Expression <- Sequence (SLASH Sequence)*
        output = [self.parseSequence()]
        while self.matcht(TokenTypes.PRIORITY):
            output.append(self.parseSequence())
        # We don't need to create a new expression when there's only
        # one element so we save a lil recursion here and there
        if len(output) > 1: return Expression(output)
        else: return fio(output)

    def parseSequence(self):
        # Sequence <- Prefix*
        output = []
        while True:
            # Prefix <- (AND / NOT)? Suffix
            prefix = lambda x: x
            if self.matcht(TokenTypes.AND): prefix = And
            elif self.matcht(TokenTypes.NOT): prefix = Not
            suffix = self.parseSuffix()
            if suffix is None: break
            output.append(prefix(suffix))
        if len(output) > 1: return Sequence(output)
        else: return fio(output)

    def parseSuffix(self):
        # Suffix <- Primary (QUESTION / STAR / PLUS)?
        output = [self.parsePrimary()]
        suffix = lambda x: x
        if self.matcht(TokenTypes.QUESTION): suffix = Question
        elif self.matcht(TokenTypes.STAR): suffix = Star
        elif self.matcht(TokenTypes.PLUS): suffix = Plus
        return suffix(fio(output))

    def parsePrimary(self):
        # Primary <- Identifier !LEFTARROW
        #          / OPEN Expression CLOSE
        #          / Literal / Class / DOT
        if self.testt(TokenTypes.IDENTIFIER) and self.peekt()._type != TokenTypes.ARROW:
            return Identifier(self.consumet(TokenTypes.IDENTIFIER).value)
        if self.testt(TokenTypes.LITERAL):
            return Literal(self.consumet(TokenTypes.LITERAL).value)
        elif self.testt(TokenTypes.CLASS):
            return Class(self.consumet(TokenTypes.CLASS).value)
        elif self.matcht(TokenTypes.DOT):
            return Dot()
        elif self.matcht(TokenTypes.OPEN):
            if self.matcht(TokenTypes.CLOSE): return []
            value = self.parseExpression()
            self.consumet(TokenTypes.CLOSE)
            return value
        return None

    def parse(self):
        self.nextt()
        return self.parseDefinitions()

    def run(self):
        try: return self.parse()
        except SyntaxError as exc:
            output = []
            code = self.code
            mark = self.token_start
            output.append(
                code[:mark] +
                '\033[41m' +
                code[mark] +
                '\033[0m' +
                '\033[91m <----- HERE!!\033[0m' +
                code[mark+1:]
            )
            lines = '\n'.join(output).split('\n')
            numbered = ['{:02d}: {}'.format(i+1, x) for i, x in enumerate(lines)]
            message = ['%s at line %d' % (str(exc), self.line+1), ''] + numbered
            raise FormattedException('\n'.join(message))


def mergeDicts(dicts):
    return {x: y for di in dicts for x, y in di.items()}


class Match:

    def __init__(self, grammar, start, data):
        self.g = mergeDicts(grammar)
        self.start = start
        self.data = data
        self.pos = 0

    def current(self):
        if self.pos >= len(self.data): return None
        return self.data[self.pos]

    def advance(self, n=1):
        if self.pos+n >= len(self.data)+1: return None
        self.pos += n
        return self.current()

    def ret(self, mark):
        return self.data[mark:self.pos] or None

    def matchClass(self, atom):
        d = self.pos
        value = atom.value
        if not self.current(): return False, None

        for c in value:
            if isinstance(c, list):
                left, right = c
                if left <= self.current() <= right:
                    result = self.current()
                    self.advance()
                    return True, result
            else:
                if self.current() == c:
                    result = self.current()
                    self.advance();
                    return True, result
        return False, None

    def matchLiteral(self, atom):
        d = self.pos
        for c in atom.value:
            if self.current() == c:
                self.advance()
        output = self.ret(d)
        return output != None, output

    def matchDot(self, atom):
        d = self.pos
        self.advance()
        output = self.ret(d)
        return output != None, output

    def matchPlus(self, atom):
        match, value = self.matchAtom(atom.value)
        if match:
            out = [value] + self.matchStar(Star(atom.value))[1]
            return True, fio(out)
        return False, None

    def matchQuestion(self, atom):
        match, value = self.matchAtom(atom.value)
        return True, value

    def matchStar(self, atom):
        out = []
        while True:
            match, value = self.matchAtom(atom.value)
            if not match: break
            out.append(value)
        return True, out

    def matchNot(self, atom):
        d = self.pos
        match, value = self.matchAtom(atom.value)
        if match:
            # We got a match. To negate it, we'll reset the cursor
            # position to prior to the matchuation and return None.
            self.pos = d
            return False, None
        else:
            return True, None

    def matchSequence(self, atom):
        d = self.pos
        out = []
        for sa in atom.value:
            match, value = self.matchAtom(sa)
            if match and value: out.append(value)
            elif match: continue
            else:
                self.pos = d
                return False, None
        return True, fio(out)

    def matchExpression(self, atom):
        d = self.pos
        for sa in atom.value:
            match, value = self.matchAtom(sa)
            if match: return True, value
        return False, None

    def matchIdentifier(self, atom):
        return self.matchAtom(self.g[atom.value])

    def matchAtom(self, atom):
        if isinstance(atom, Class):
            return self.matchClass(atom)
        elif isinstance(atom, Literal):
            return self.matchLiteral(atom)
        elif isinstance(atom, Dot):
            return self.matchDot(atom)
        elif isinstance(atom, Identifier):
            return self.matchIdentifier(atom)
        elif isinstance(atom, Plus):
            return self.matchPlus(atom)
        elif isinstance(atom, Star):
            return self.matchStar(atom)
        elif isinstance(atom, Not):
            return self.matchNot(atom)
        elif isinstance(atom, Sequence):
            return self.matchSequence(atom)
        elif isinstance(atom, Expression):
            return self.matchExpression(atom)
        elif isinstance(atom, Question):
            return self.matchQuestion(atom)
        raise Exception('Unexpected atom')

    def run(self):
        return self.matchAtom(self.g[self.start])


class Instructions(enum.Enum):
    (OP_HALT,
     OP_CHAR,
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
     OP_END,
    ) = range(19)

InstructionParams = {
    Instructions.OP_HALT: 0,
    Instructions.OP_CHAR: 1,
    Instructions.OP_ANY: 0,
    Instructions.OP_CHOICE: 1,
    Instructions.OP_COMMIT: 1,
    Instructions.OP_FAIL: 0,
    Instructions.OP_FAIL_TWICE: 1,
    Instructions.OP_PARTIAL_COMMIT: 1,
    Instructions.OP_BACK_COMMIT: 1,
    Instructions.OP_TEST_CHAR: 2,
    Instructions.OP_TEST_ANY: 2,
    Instructions.OP_JUMP: 1,
    Instructions.OP_CALL: 1,
    Instructions.OP_RETURN: 0,
    Instructions.OP_SPAN: 2,
    Instructions.OP_SET: 0,
    Instructions.OP_CAP_OPEN: 2,
    Instructions.OP_CAP_CLOSE: 2,
    Instructions.OP_END: 0,
}

INSTRUCTION_SIZE =  32
OPERATOR_SIZE    =  5
OPERATOR_OFFSET  =  (INSTRUCTION_SIZE - OPERATOR_SIZE)
SL_OPERAND_SIZE  =  OPERATOR_OFFSET
S1_OPERAND_SIZE  =  11
S2_OPERAND_SIZE  =  16


def gen(instruction_name, arg0=None, arg1=None):
    instruction = Instructions["OP_{}".format(instruction_name.upper())]
    if arg0 is not None and arg1 is not None: # Two args, 1) 10b 2) 16b
        return ((arg1 | (arg0 << S2_OPERAND_SIZE)) | (instruction.value << OPERATOR_OFFSET))
    elif arg0 is not None and arg1 is None: # Single 27bits arg
        return (arg0 & 0x07ffffff) | (instruction.value << OPERATOR_OFFSET)
    elif arg0 is None and arg1 is not None: # Not supported
        raise Exception("Plz use arg0 instead of arg1")
    else:                  # No arguments. Just padding with zeros
        return (instruction.value << OPERATOR_OFFSET)


class Capture:

    def __init__(self, compiler, capture, isTerminal, capId):
        self.capture = capture
        self.compiler = compiler
        self.isTerminal = isTerminal
        self.capId = capId

    def __enter__(self):
        if self.capture:
            self.compiler.emit(
                "cap_open", self.isTerminal, self.compiler._str(self.capId))
        return self

    def __exit__(self, _type, value, traceback):
        if self.capture:
            self.compiler.emit(
                "cap_close", self.isTerminal, self.compiler._str(self.capId))

class Wrap:
    "Utility to wrap a value around a thing that has a .value attr"
    def __init__(self, v):
        self.value = v


class Compiler:

    def __init__(self, grammar, capture=False):
        self.ga = grammar
        # The grammar as a dictionary. Since we lose the order, that's
        # why the same data as above but with a different format.
        self.g = {x: y for di in grammar for x, y in di.items()}
        # The start rule is just the first one for now
        self.start = list(grammar[0].keys())[0]
        # Write cursor for `self.code'
        self.pos = 0
        # Store the binary code that will be packed in the end of the
        # process. Each entry is a uint32_t generated with `gen()'
        self.code = []
        # Save all the call instructions that need to be updated with
        # the final address of where they want to call to.
        self.callsites = {}
        # If we're going to generate capture opcodes or not.
        self.capture = capture
        # Capture id
        self.captureId = 0
        # All strings that couldn't fit within an instruction
        # parameter.
        self.strings = []

    def emit(self, *args):
        self.code.append(gen(*args))
        self.pos += 1
        return self.pos

    def cc(self, atom):
        currentPos = self.pos
        self.compileAtom(atom)
        programSize = self.pos - currentPos
        return programSize

    def _str(self, value):
        if value not in self.strings:
            self.strings.append(value)
        return self.strings.index(value)

    def _capture(self, isTerminal, capid):
        return Capture(self, self.capture, isTerminal, capid)

    def _disableCapture(self):
        class DisableCapture:
            def __init__(self, c):
                self.c = c
                self.backup = c.capture
            def __enter__(self):
                self.c.capture = False
            def __exit__(self, _type, value, traceback):
                self.c.capture = self.backup
        return DisableCapture(self)

    def compileNot(self, atom):
        pos = self.emit("choice") -1
        with self._disableCapture():
            size = self.cc(atom.value)
        self.code[pos] = gen("choice", size + 3)
        self.emit("commit", 1)
        self.emit("fail")

    def compileAnd(self, atom):
        pos = self.emit("choice")
        self.compileNot(atom)
        size = self.pos - pos
        self.code[pos-1] = gen("choice", size + 3)
        self.emit("commit", 1)
        self.emit("fail")

    def compileLiteral(self, literal):
        currentPos = self.pos
        with self._capture(1, literal.value):
            for i in literal.value:
                self.emit("char", ord(i))
        return self.pos - currentPos

    def compileAny(self, atom):
        with self._capture(1, 'Any'):
            self.emit("any")

    def compileRange(self, theRange):
        currentPos = self.pos
        left, right = theRange
        with self._capture(1, '{}-{}'.format(left, right)):
            self.emit("span", ord(left), ord(right))
        return self.pos - currentPos

    def compileSequence(self, sequence):
        for atom in sequence.value:
            self.compileAtom(atom)

    def _compileChoices(self, choices, compFunc):
        commits = []     # List of positions that need to be rewritten
        for i, c in enumerate(choices):
            if i + 1 == len(choices):
                compFunc(c)
                break
            pos = self.emit("choice") -1
            size = compFunc(c)
            self.code[pos] = gen("choice", size + 2)
            commits.append(self.emit("commit") - 1)
        for i in commits:       # Rewrite the locations for commits
            self.code[i] = gen("commit", self.pos - i)

    def compileExpression(self, expr):
        self._compileChoices(expr.value, self.cc)

    def compileIdentifier(self, atom):
        # Emit an OP_CALL as a placeholder to be patched at the end of
        # the code generation
        pos = self.emit("call") -1
        if atom.value not in self.callsites:
            self.callsites[atom.value] = []
        self.callsites[atom.value].append(pos)

    def compileStar(self, atom):
        pos = self.emit("choice") -1
        size = self.cc(atom.value)
        self.code[pos] = gen("choice", size + 2)
        self.emit("commit", -(size + 1))

    def compilePlus(self, atom):
        self.cc(atom.value)
        self.compileStar(atom)

    def compileQuestion(self, atom):
        pos = self.emit('choice')-1
        size = self.cc(atom.value)
        self.code[pos] = gen('choice', size+2)
        self.emit('commit', 1)

    def compileRangeOrLiteral(self, thing):
        if isinstance(thing, list):
            return self.compileRange(thing)
        return self.compileLiteral(Wrap(thing))

    def compileClass(self, atom):
        """Generate code for matching classes of characters

        Classes of characteres can be either one interval, multiple
        intervals or sets of characters.

        If `atom' describes only one interval, a single `Span'
        instruction is generated.
        """
        if len(atom.value) == 1:
            return self.compileRangeOrLiteral(atom.value[0])
        else:
            return self._compileChoices(atom.value, self.compileRangeOrLiteral)

    def compileAtom(self, atom):
        if isinstance(atom, Literal): self.compileLiteral(atom)
        elif isinstance(atom, Dot): self.compileAny(atom)
        elif isinstance(atom, Not): self.compileNot(atom)
        elif isinstance(atom, And): self.compileAnd(atom)
        elif isinstance(atom, Star): self.compileStar(atom)
        elif isinstance(atom, Plus): self.compilePlus(atom)
        elif isinstance(atom, Question): self.compileQuestion(atom)
        elif isinstance(atom, Class): self.compileClass(atom)
        elif isinstance(atom, Identifier): self.compileIdentifier(atom)
        elif isinstance(atom, Sequence): self.compileSequence(atom)
        elif isinstance(atom, Expression): self.compileExpression(atom)
        else: raise Exception("Unknown atom %s" % atom)

    def genCode(self):
        # It's always 2 because invariant contains two instructions
        self.emit("call", 2)
        pos = self.emit("jump")
        size = 0
        addresses = {}
        for nt in self.ga:
            [[name, rule]] = nt.items()
            addresses[name] = self.pos
            self._str(name)
            with self._capture(0, name): self.cc(rule)
            self.emit("return")
            size += self.pos - addresses[name]
        self.code[pos-1] = gen("jump", size + 2)
        self.emit("halt")

        # Patch all OP_CALL instructions generated with final
        # addresses
        for identifier, callsites in self.callsites.items():
            for cs in callsites:
                self.code[cs] = gen("call", addresses[identifier] - cs)
        # Pack the integers as binary data
        return struct.pack('>' + ('I' * len(self.code)), *self.code)

    def assemble(self):
        code = self.genCode()   # has to run before the rest
        # Write string table size & string table entries
        assembled = uint8(len(self.strings))
        for i in self.strings:
            assembled += uint8(len(i))
            s = i.encode('ascii')
            assembled += struct.pack('>' + ('s'*len(s)), *s)
        # Write code size & code
        assembled += uint16(len(code))
        assembled += code
        return assembled


def uint8(v): return struct.pack('>B', v)
def uint16(v): return struct.pack('>H', v)

## --- Utilities ---

OP_MASK   = lambda c: (((c) & 0xf8000000) >> OPERATOR_OFFSET)
UOPERAND0 = lambda c: ((c) & 0x7ffffff)
UOPERAND1 = lambda c: (UOPERAND0(c) >> S2_OPERAND_SIZE)
UOPERAND2 = lambda c: (UOPERAND0(c) & ((1 << S2_OPERAND_SIZE) - 1))


def dbgcc(c, bc):
    if c[-1] == '\n': c = c[:-1]
    print('\033[92m{}\033[0m'.format(c.encode('utf-8')), end=':\n')
    # Reading facilities
    cursor = 0
    readuint8 = lambda: struct.unpack('>B', bc[cursor])[0]
    readuint16 = lambda: struct.unpack('>H', bc[cursor:cursor+2])[0]
    readstring = lambda n: ''.join(struct.unpack('>' + 's'*n, bc[cursor:cursor+n]))
    # Parse header
    headerSize = readuint8(); cursor += 1
    print('Header(%s)' % headerSize)
    for i in range(headerSize):
        ssize = readuint8(); cursor += 1
        content = readstring(ssize);
        print("   0x%02x: String(%2ld) %s" % (i, ssize, repr(content)))
        cursor += ssize
    # Parse body
    codeSize = readuint16(); cursor += 2
    codeStart = len(bc)-codeSize
    print('Code(%d)' % codeSize)
    unpacked = struct.unpack('>' + ('I' * (codeSize/4)), bc[codeStart:])
    for i, instr in enumerate(unpacked):
        obj = Instructions(OP_MASK(instr))
        name = obj.name
        argc = InstructionParams[obj]
        if argc == 1:
            val = '      0x{:03x}'.format(UOPERAND0(instr))
        elif argc == 2:
            val = ' 0x{:02x} 0x{:03x}'.format(
                UOPERAND1(instr), UOPERAND2(instr))
        else:
            val = '           '
        print('   0x{:03x} 0x{:08x} [{:>17}{}]'.format(i, instr, obj.name, val))
    return bc

## --- tests ---

csv = r'''
File <- CSV*
CSV  <- Val (',' Val)* '\n'
Val  <- (![,\n] .)*
'''

arith = r'''
Add <- Mul '+' Add / Mul
Mul <- Pri '*' Mul / Pri
Pri <- '(' Add ')' / Num
Num <- [0-9]+
'''

def test_runner(f, g, expected, *args):
    print('\033[92m{}\033[0m'.format(repr(g)), end=':\n    ')
    value = f(g)(*args)
    pprint.pprint(value)
    try:
        assert(value == expected)
    except Exception as exc:
        import pdb; pdb.set_trace()
        raise exc
    print()


def test_runner_match(g, start, data, expected):
    print('\033[92m{}\033[0m'.format(repr(g)), end=':\n    ')
    value = Match(Parser(g).parse(), start, data).run()
    pprint.pprint(value)
    assert(value == expected)
    print()

def test_runner_compile(g, expected):
    print('\033[92m{}\033[0m'.format(repr(g)), end=':\n    ')
    value = Compile(Parser(g).parse()).run()
    pprint.pprint(value)
    assert(value == expected)
    print()


def expand_tokenizer(x):
    pa = Parser(x)
    ts = []
    while True:
        t = pa.nextt()
        ts.append(t)
        if t._type == TokenTypes.END: break
    return lambda: ts


def test_tokenizer():
    test = functools.partial(test_runner, expand_tokenizer)

    test('Rule1 <- "tx"', [
        Token(TokenTypes.IDENTIFIER, 'Rule1', line=0, pos=0),
        Token(TokenTypes.ARROW,               line=0, pos=6),
        Token(TokenTypes.LITERAL,    'tx',    line=0, pos=9),
        Token(TokenTypes.END,                 line=0, pos=13),
    ])

    test('V <- (![,\\n] .)*', [
        Token(TokenTypes.IDENTIFIER, 'V', line=0, pos=0),
        Token(TokenTypes.ARROW,           line=0, pos=2),
        Token(TokenTypes.OPEN,            line=0, pos=5),
        Token(TokenTypes.NOT,             line=0, pos=6),
        Token(TokenTypes.CLASS, [',', '\n'], line=0, pos=7),
        Token(TokenTypes.DOT,             line=0, pos=13),
        Token(TokenTypes.CLOSE,           line=0, pos=14),
        Token(TokenTypes.STAR,            line=0, pos=15),
        Token(TokenTypes.END,             line=0, pos=16),
    ])

    test('Hex <- [a-fA-F0-9]', [
        Token(TokenTypes.IDENTIFIER, 'Hex', line=0, pos=0),
        Token(TokenTypes.ARROW,             line=0, pos=4),
        Token(TokenTypes.CLASS,
              [['a', 'f'], ['A', 'F'], ['0', '9']],
              line=0, pos=7),
        Token(TokenTypes.END, line=0, pos=18),
    ])

    test('Hex <- [a-fA-F_]', [
        Token(TokenTypes.IDENTIFIER, 'Hex', line=0, pos=0),
        Token(TokenTypes.ARROW,             line=0, pos=4),
        Token(TokenTypes.CLASS,
              [['a', 'f'], ['A', 'F'], '_'],
              line=0, pos=7),
        Token(TokenTypes.END, line=0, pos=16),
    ])

    test("# multiple\n#lines\n#with\n#comments\nS <- 'a'", [
        Token(TokenTypes.IDENTIFIER, 'S', line=4, pos=34),
        Token(TokenTypes.ARROW, line=4, pos=36),
        Token(TokenTypes.LITERAL, 'a', line=4, pos=39),
        Token(TokenTypes.END, line=4, pos=42),
    ])


def test_parser():
    test = functools.partial(test_runner, lambda x: Parser(x).parse)

    test('# ', [])

    test('#foo\r\nR0 <- "a"', [{'R0': Literal('a')}])

    test(r"R0 <- '\\' [nrt'\"\[\]]", [
        {'R0': Sequence([Literal('\\'), Class(['n', 'r', 't', "'", '"', '[', ']'])])}])

    test("R <- '\\r\\n' / '\\n' / '\\r'", [
        {'R': Expression([Literal('\r\n'), Literal('\n'), Literal('\r')])}])

    test('# foo\n R1 <- "a"\nR2 <- \'b\'', [
        {'R1': Literal('a')},
        {'R2': Literal('b')}])

    test('Definition1 <- "tx"', [{'Definition1': Literal('tx')}])

    test('Int <- [0-9]+', [{'Int': Plus(Class([['0', '9']]))}])

    test('Foo <- [a-z_]', [{'Foo': Class([['a', 'z'], '_'])}])

    test('EndOfFile <- !.', [{'EndOfFile': Not(Dot())}])

    test('R0 <- "oi" "tenta"?', [{'R0': Sequence([Literal('oi'), Question(Literal("tenta"))])}])

    test('Foo <- ("a" / "b")+', [{'Foo': Plus(Expression([Literal('a'), Literal('b')]))}])

    test('R0 <- "a"\n      / "b"\nR1 <- "c"', [
        {'R0': Expression([Literal('a'), Literal('b')])},
        {'R1': Literal('c')}])

    test('R0 <- R1 ("," R1)*\nR1 <- [0-9]+', [
        {'R0': Sequence([
            Identifier('R1'),
            Star(Sequence([Literal(','), Identifier('R1')]))])},
        {'R1': Plus(Class([['0', '9']]))}])

    test(r"""# first line with comment
Spacing    <- (Space / Comment)*
Comment    <- '#' (!EndOfLine .)* EndOfLine
Space      <- ' ' / '\t' / EndOfLine
EndOfLine  <- '\r\n' / '\n' / '\r'
EndOfFile  <- !.
    """, [
        {'Spacing': Star(Expression([
            Identifier('Space'),
            Identifier('Comment')]))},
        {'Comment': Sequence([
            Literal('#'),
            Star(Sequence([Not(Identifier('EndOfLine')), Dot()])),
            Identifier('EndOfLine')])},
        {'Space': Expression([
            Literal(' '),
            Literal('\t'),
            Identifier('EndOfLine')])},
        {'EndOfLine': Expression([
            Literal('\r\n'),
            Literal('\n'),
            Literal('\r')])},
        {'EndOfFile': Not(Dot())}])

    # Some real stuff

    test(csv, [
        {'File': Star(Identifier('CSV'))},
        {'CSV': Sequence([
            Identifier('Val'),
            Star(Sequence([Literal(','), Identifier('Val')])),
            Literal('\n')])},
        {'Val': Star(Sequence([Not(Class([',', '\n'])), Dot()]))}])

    test(arith, [
        {'Add': Expression([Sequence([Identifier('Mul'),
                                      Literal('+'),
                                      Identifier('Add')]),
                            Identifier('Mul')])},
        {'Mul': Expression([Sequence([Identifier('Pri'),
                                      Literal('*'),
                                      Identifier('Mul')]),
                            Identifier('Pri')])},
        {'Pri': Expression([Sequence([Literal('('),
                                      Identifier('Add'),
                                      Literal(')')]),
                            Identifier('Num')])},
        {'Num': Plus(Class([['0', '9']]))}])


def _safe_from_error(p):
    raised = False
    value = None
    try:
        value = p.parse()
    except Exception as exc:
        raised = True
        value = exc
    return raised, value


def test_parse_errors():
    p = Parser("X 'a'")
    raised, value = _safe_from_error(p)
    assert(raised == True)
    assert(isinstance(value, SyntaxError))
    assert(str(value) == "Expected ARROW but found LITERAL")
    assert(p.token == Token(TokenTypes.LITERAL, 'a', line=0, pos=2))

    p = Parser("X < 'a'")
    raised, value = _safe_from_error(p)
    assert(raised == True)
    assert(isinstance(value, SyntaxError))
    assert(str(value) == "Missing the dash in the arrow")
    assert(p.token == Token(TokenTypes.IDENTIFIER, 'X', line=0, pos=0))


    value = Parser('AtoC <- [a-c]\nNoAtoC <- !AtoC .\nEOF < !.\n').run()
    expected = r'''Missing the dash in the arrow at line 2

AtoC <- [a-c]
NoAtoC <- !AtoC .
EOF \x1b[31m\x1b[0m !.
'''
    #assert(value == expected)


def test_match():
    e = Match({}, '', "affbcdea&2")
    assert(e.matchAtom(Class(['a', 'f']))   == (True, 'a'));   assert(e.pos == 1)
    assert(e.matchAtom(Class('gd'))         == (False, None)); assert(e.pos == 1)
    assert(e.matchAtom(Class('xyf'))        == (True, 'f'));   assert(e.pos == 2)
    assert(e.matchAtom(Class([['a', 'f']])) == (True, 'f'));   assert(e.pos == 3)
    assert(e.matchAtom(Class([['a', 'f']])) == (True, 'b'));   assert(e.pos == 4)
    assert(e.matchAtom(Class([['a', 'f']])) == (True, 'c'));   assert(e.pos == 5)
    assert(e.current() == 'd');                               assert(e.pos == 5)
    assert(e.matchAtom(Literal('a'))        == (False, None)); assert(e.pos == 5)
    assert(e.matchAtom(Literal('d'))        == (True, 'd'));   assert(e.pos == 6)
    assert(e.current() == 'e');                               assert(e.pos == 6)
    assert(e.matchAtom(Dot())               == (True, 'e'));   assert(e.pos == 7)
    assert(e.matchAtom(Dot())               == (True, 'a'));   assert(e.pos == 8)
    assert(e.matchAtom(Dot())               == (True, '&'));   assert(e.pos == 9)
    assert(e.matchAtom(Dot())               == (True, '2'));   assert(e.pos == 10)
    assert(e.matchAtom(Dot())               == (False, None)); assert(e.pos == 10)

    e = Match(Parser('Lit <- "test"').parse(), 'Lit', "testando")
    assert(e.matchAtom(Identifier('Lit')) == (True, "test"))

    e = Match(Parser('Digit <- [0-9]').parse(), 'Digit', "42f")
    assert(e.matchAtom(Identifier('Digit')) == (True, "4"))
    assert(e.matchAtom(Identifier('Digit')) == (True, "2"))
    assert(e.matchAtom(Identifier('Digit')) == (False, None))
    assert(e.current() == 'f')

    e = Match(Parser('Digit <- [0-9]+').parse(), 'Digit', "42f")
    assert(e.matchAtom(Identifier('Digit')) == (True, ['4', '2']))
    e = Match(Parser('Digit <- [0-9]*').parse(), 'Digit', "2048f")
    assert(e.matchAtom(Identifier('Digit')) == (True, ['2', '0', '4', '8']))
    e = Match(Parser('Digit <- [0-9]*').parse(), 'Digit', '2048')
    assert(e.matchAtom(Identifier('Digit')) == (True, ['2', '0', '4', '8']))

    e = Match(Parser('AtoC <- [a-c]\nNoAtoC <- !AtoC .\nEOF <- !.\n').parse(), '', 'abcdef')
    assert(e.matchAtom(Identifier('NoAtoC')) == (False, None))  ; assert(e.pos == 0)
    assert(e.matchAtom(Identifier('AtoC'))   == (True, 'a'))    ; assert(e.pos == 1)
    assert(e.matchAtom(Identifier('NoAtoC')) == (False, None))  ; assert(e.pos == 1)
    assert(e.matchAtom(Identifier('AtoC'))   == (True, 'b'))    ; assert(e.pos == 2)
    assert(e.matchAtom(Identifier('AtoC'))   == (True, 'c'))    ; assert(e.pos == 3)
    assert(e.matchAtom(Identifier('AtoC'))   == (False, None))  ; assert(e.pos == 3)
    assert(e.matchAtom(Identifier('AtoC'))   == (False, None))  ; assert(e.pos == 3)
    assert(e.matchAtom(Identifier('NoAtoC')) == (True, 'd'))    ; assert(e.pos == 4)
    assert(e.matchAtom(Identifier('EOF'))    == (False, None))  ; assert(e.pos == 4)
    assert(e.matchAtom(Identifier('AtoC'))   == (False, None))  ; assert(e.pos == 4)
    assert(e.matchAtom(Identifier('NoAtoC')) == (True, 'e'))    ; assert(e.pos == 5)
    assert(e.matchAtom(Identifier('NoAtoC')) == (True, 'f'))    ; assert(e.pos == 6)
    assert(e.matchAtom(Identifier('NoAtoC')) == (False, None))  ; assert(e.pos == 6)
    assert(e.matchAtom(Identifier('AtoC'))   == (False, None))  ; assert(e.pos == 6)
    assert(e.matchAtom(Identifier('EOF'))    == (True, None))   ; assert(e.pos == 6)

    e = Match(Parser('EOF <- !.\nALL <- [a-f]').parse(), 'EOF', 'f')
    assert(e.matchAtom(Identifier('EOF')) == (False, None))
    assert(e.matchAtom(Identifier('ALL')) == (True, 'f'))
    e = Match(Parser(arith).parse(), 'Add', "12+34*56")
    assert(e.matchAtom(Identifier('Add')) == (True, [
        ['1', '2'],
        '+', [['3', '4'],
              '*',
              ['5', '6']]]))

    e = Match(Parser(csv).parse(), 'File', "Name,Num,Lang\nLink,3,pt-br\n")
    assert(e.matchAtom(Identifier('File')) == (True, [[['N', 'a', 'm', 'e'],
      [[',', ['N', 'u', 'm']],
       [',', ['L', 'a', 'n', 'g']]],
      '\n'],
     [['L', 'i', 'n', 'k'],
      [[',', ['3']],
       [',', ['p', 't', '-', 'b', 'r']]], '\n']]))

    e = Match(Parser("EndOfLine <- '\r\n' / '\n' / '\r'").parse(), 'EndOfLine', "\n\r\n\r")
    assert(e.matchAtom(Identifier('EndOfLine')) == (True, '\n')); assert(e.pos == 1);

    e = Match(Parser("""

NL <- '\r\n'
    / '\n'
    / '\r'
    / '\t'

C <- [a-z]+
   / NL

""").parse(), 'EndOfLine', "tst\n\r\t")

    assert(e.matchAtom(Identifier('C')) == (True, ['t', 's', 't']))  ; assert(e.pos == 3)
    assert(e.matchAtom(Identifier('C')) == (True, '\n'))             ; assert(e.pos == 4)
    assert(e.matchAtom(Identifier('C')) == (True, '\r'))
    assert(e.matchAtom(Identifier('C')) == (True, '\t'))


def test_instruction():
    # No arguments
    assert(
        gen("any") == 0b00010000000000000000000000000000
    )
    # One Argument
    assert(
        gen("char", ord('a')) == 0b00001000000000000000000001100001
    )
    # Two Arguments
    assert(
        gen("span", ord('a'), ord('e')) == 0b01110000011000010000000001100101
    )


def test_compile():
    bn = lambda *bc: struct.pack('>' + ('I' * len(bc)), *bc)
    ccc = lambda co, **f: Compiler(Parser(co).run(), **f)
    def cc(code, **f):
        dbgcc(code, ccc(code, **f).assemble())
        return ccc(code, **f).genCode()

    # Char 'c'
    assert(cc("S <- 'a'") == bn(
        gen("call", 2),
        gen("jump", 4),
        gen("char", ord('a')),
        gen("return"),
        gen("halt"),
    ))

    # Any
    assert(cc("S <- .") == bn(
        gen("call", 2),
        gen("jump", 4),
        gen("any"),
        gen("return"),
        gen("halt"),
    ))

    # Not
    assert(cc("S <- !'a'") == bn(
        gen("call", 2),
        gen("jump", 7),
        gen("choice", 4),
        gen("char", ord('a')),
        gen("commit", 1),
        gen("fail"),
        gen("return"),
        gen("halt"),
    ))

    # And
    assert(cc("S <- &'a'") == bn(
        gen("call", 2),
        gen("jump", 10),
        gen("choice", 7),
        gen("choice", 4),
        gen("char", ord('a')),
        gen("commit", 1),
        gen("fail"),
        gen("commit", 1),
        gen("fail"),
        gen("return"),
        gen("halt"),
    ))

    # Concatenation
    assert(cc("S <- 'a' . 'c'") == bn(
        gen("call", 2),
        gen("jump", 6),
        gen("char", ord('a')),
        gen("any"),
        gen("char", ord('c')),
        gen("return"),
        gen("halt"),
    ))

    # Ordered Choice
    assert(cc("S <- 'a' / 'b'") == bn(
        gen("call", 2),
        gen("jump", 7),
        gen("choice", 3),
        gen("char", ord('a')),
        gen("commit", 2),
        gen("char", ord('b')),
        gen("return"),
        gen("halt"),
    ))

    assert(cc("S <- 'a' / 'b' / 'c' / 'd'") == bn(
        gen("call", 2),
        gen("jump", 13),
        gen("choice", 3),
        gen("char", ord('a')),
        gen("commit", 8),
        gen("choice", 3),
        gen("char", ord('b')),
        gen("commit", 5),
        gen("choice", 3),
        gen("char", ord('c')),
        gen("commit", 2),
        gen("char", ord('d')),
        gen("return"),
        gen("halt"),
    ))

    # Repetition (Star)
    assert(cc("S <- 'a'*") == bn(
        gen("call", 2),
        gen("jump", 6),
        gen("choice", 3),
        gen("char", ord('a')),
        gen("commit", -2),
        gen("return"),
        gen("halt"),
    ))

    # Plus
    assert(cc("S <- 'a'+") == bn(
        gen("call", 2),
        gen("jump", 7),
        gen("char", ord('a')),
        gen("choice", 3),
        gen("char", ord('a')),
        gen("commit", -2),
        gen("return"),
        gen("halt"),
    ))

    # Question
    assert(cc("S <- 'a' 'b'?") == bn(
        gen("call", 2),
        gen("jump", 7),
        gen("char", ord('a')),
        gen("choice", 3),
        gen("char", ord('b')),
        gen("commit", 1),
        gen("return"),
        gen("halt"),
    ))

    # Class -> Char
    assert(cc("S <- [a]") == bn(
        gen("call", 2),
        gen("jump", 4),
        gen("char", ord('a')),
        gen("return"),
        gen("halt"),
    ))

    # O0: Class -> Char / Char
    assert(cc("S <- [ab]") == bn(
        gen("call", 2),
        gen("jump", 7),
        gen("choice", 3),
        gen("char", ord('a')),
        gen("commit", 2),
        gen("char", ord('b')),
        gen("return"),
        gen("halt"),
    ))

    # O0: Class -> Span / Span
    assert(cc("S <- [a-zA-Z]") == bn(
        gen("call", 2),
        gen("jump", 7),
        gen("choice", 3),
        gen("span", ord('a'), ord('z')),
        gen("commit", 2),
        gen("span", ord('A'), ord('Z')),
        gen("return"),
        gen("halt"),
    ))

    # Class -> Span / Char
    assert(cc("S <- [a-z_]") == bn(
        gen("call", 2),
        gen("jump", 7),
        gen("choice", 3),
        gen("span", ord('a'), ord('z')),
        gen("commit", 2),
        gen("char", ord('_')),
        gen("return"),
        gen("halt"),
    ))

    # Class -> Span / Span / Char
    assert(cc("S <- [a-zA-Z_]") == bn(
        gen("call", 2),
        gen("jump", 10),
        gen("choice", 3),
        gen("span", ord('a'), ord('z')),
        gen("commit", 5),
        gen("choice", 3),
        gen("span", ord('A'), ord('Z')),
        gen("commit", 2),
        gen("char", ord('_')),
        gen("return"),
        gen("halt"),
    ))

    # O1: Class -> Set
    # assert(cc("S <- [']") == bn(
    # ))

    # Grammar/Variables (Call/Return)
    assert(cc("S <- D '+'D\nD <- '0' / '1'") == bn(
        gen("call", 2),
        gen("jump", 11),
        gen("call", 4),
        gen("char", ord('+')),
        gen("call", 2),
        gen("return"),
        gen("choice", 3),
        gen("char", ord('0')),
        gen("commit", 2),
        gen("char", ord('1')),
        gen("return"),
        gen("halt"),
    ))

    ## Captures

    assert(cc("S <- 'a'", capture=True) == bn(
        gen("call", 0x02),      # 0x00: Call 0x02
        gen("jump", 0x08),      # 0x01: Jump 0x08
        gen("cap_open", 0, 0),  # 0x02: CapOpen 0 (Main)
        gen("cap_open", 1, 1),  # 0x03: CapOpen 1
        gen("char", ord('a')),  # 0x04: Char 'a'
        gen("cap_close", 1, 1), # 0x05: CapOpen 1
        gen("cap_close", 0, 0), # 0x06: CapClose (Main)
        gen("return"),          # 0x07: Return
        gen("halt"),            # 0x08: Halt
    ))

    assert(cc("S <- 'a' 'b'", capture=True) == bn(
        gen("call", 0x02),      # 0x01: Call 0x02
        gen("jump", 0x0b),      # 0x02: Jump 0x0b

        gen("cap_open", 0, 0),  # 0x03: CapOpen 0 (Main)

        gen("cap_open", 1, 1),  # 0x04: CapOpen 1 (Seq#0)
        gen("char", ord('a')),  # 0x05: Char 'a'
        gen("cap_close", 1, 1), # 0x06: CapClose 1 (Seq#0)

        gen("cap_open", 1, 2),  # 0x07: CapOpen 2 (Seq#1)
        gen("char", ord('b')),  # 0x08: Char 'a'
        gen("cap_close", 1, 2), # 0x09: CapClose 2 (Seq#1)

        gen("cap_close", 0, 0), # 0x0a: CapClose 0 (Main)

        gen("return"),          # 0x0b: Return
        gen("halt"),            # 0x0c: Halt
    ))


def test_compile_header():
    def cc(src, **f):
        c = Compiler(Parser(src).run(), capture=True, **f)
        c.genCode()
        return c.strings

    # Name of the rule
    assert(cc("S <- 'a'") == ['S', 'a'])
    # Name of the rules and set of chars
    assert(cc("S <- D [%?!~#$&+]\nD <- [0-9]+") == [
        'S',
        '%', '?', '!', '~', '#', '$', '&', '+',
        'D',
        '0-9',
    ])
    # Name of the rules and set of chars without dups
    assert(cc("S <- [ai sim hein]") == [
        'S',
        'a', 'i', ' ', 's', 'm', 'h', 'e', 'n',
    ])


def test():
    test_tokenizer()
    test_parser()
    # test_parse_errors()
    test_match()
    test_instruction()
    test_compile()
    test_compile_header()


# ---- Run the thing ----


def parseG(args):
    with io.open(os.path.abspath(args.grammar), 'r', encoding='utf-8') as grammarFile:
        grammarSrc = grammarFile.read()
        grammar = Parser(grammarSrc).run()
    return grammarSrc, grammar


def compileG(args, grammarSrc, grammar):
    name, _ = os.path.splitext(args.grammar)
    with io.open('%s.bin' % name, 'wb') as out:
        compiled = Compiler(grammar, capture=True).assemble()
        out.write(compiled)
        if not args.quiet: dbgcc(grammarSrc, compiled)


def matchG(args, grammarSrc, grammar):
    with io.open(os.path.abspath(args.data), 'r') as dataFile:
        output = Match(grammar, args.start, dataFile.read()).run()
        pprint.pprint(output)


def run(args):
    try:
        grammarSrc, grammar = parseG(args)
        if args.compile: compileG(args, grammarSrc, grammar)
        else: matchG(args, grammarSrc, grammar)
    except FormattedException as exc:
        print(exc.message)
        exit(1)


def main():
    parser = argparse.ArgumentParser(
        description='Parse structured data with Parsing Expression Grammars')
    parser.add_argument(
        '-g', '--grammar', dest='grammar', action='store',
        help='Grammar File')
    parser.add_argument(
        '-c', '--compile', dest='compile', action='store_true', default=False,
        help='Compile grammar')
    parser.add_argument(
        '-q', '--quiet', dest='quiet', action='store_true', default=False,
        help='Quiet')
    parser.add_argument(
        '-d', '--data', dest='data', action='store',
        help='Data File')
    parser.add_argument(
        '-s', '--start', dest='start', action='store',
        help='Start rule. Which rule the parser should start at.')
    parser.add_argument(
        '-t', '--tests', dest='test', action='store_true', default=False,
        help='Run tests.')
    args = parser.parse_args()

    if args.test: exit(test())

    if not (args.grammar and (args.data or args.compile)):
        parser.print_help()
        exit(0)

    run(args)


if __name__ == '__main__':
    main()
