# -*- coding: utf-8; -*-
#
# peg.py - Parsing Expression Grammar implementation
#
# Copyright (C) 2018-2019  Lincoln Clarete
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
     QUIET,
     CLASS,
     STRING,
     OPEN,
     CLOSE,
     OPCB,
     CLCB,
     OPLS,                      # Closed with CLCB
     OPCAP,
     LABEL,
     END,
    ) = range(21)

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

class Grammar(Node): pass

class Definition(Node): pass

class Expression(Node): pass

class CaptureBlock(Node): pass

class CaptureNode(Node): pass

class Sequence(Node): pass

class Identifier(Node): pass

class Literal(Node): pass

class String(Node): pass

class Class(Node): pass

class Dot(Node): pass

class Label(Node): pass

class Throw(Node): pass

class List(Node): pass

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
        if self.peekc().isalpha() or self.testc('_'):
            d = self.pos
            while self.peekc() and (self.peekc().isalnum() or self.testc('_')):
                self.pos += 1
            return self.t(TokenTypes.IDENTIFIER, self.code[d:self.pos])
        # Literal <- ['] (!['] Char)* ['] Spacing
        elif self.matchc("'"):
            return self.lexLiteral(TokenTypes.LITERAL, "'")
        # / ["] (!["] Char)* ["] Spacing
        elif self.matchc('"'):
            return self.lexLiteral(TokenTypes.STRING, '"')
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
        elif self.matchc(';'):
            return self.t(TokenTypes.QUIET)
        elif self.matchc('^'):
            return self.t(TokenTypes.LABEL)
        elif self.matchc('{'):
            return self.t(TokenTypes.OPLS)
        elif self.matchc('}'):
            return self.t(TokenTypes.CLCB)
        elif self.matchc('%'):
            if self.matchc('{'):
                return self.t(TokenTypes.OPCB)
            else:
                return self.t(TokenTypes.OPCAP)
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
            elif self.matchc('-'): return '-'
            elif self.matchc('\\'): return '\\'
            elif self.matchc('x'): return self.lexHex()
            elif self.peekc() is None: SyntaxError('Unexpected end of input')
            else: raise SyntaxError('Unknown escape char `{}`'.format(self.peekc()))
        value = self.peekc()
        self.nextc()
        return value

    def isHex(self):
        return (self.peekc() and
                (self.peekc().isdigit() or
                 ord(self.peekc().lower()) in range(ord('a'), ord('g'))))

    def lexHex(self):
        digits = []
        while self.isHex():
            digits.append(self.peekc())
            self.nextc()
        return chr(int(''.join(digits), 16))

    def lexLiteral(self, typ, end):
        output = []
        while self.peekc() and not self.testc(end):
            output.append(self.lexChar())
        if not self.matchc(end): raise SyntaxError("Expected end of string")
        return self.t(typ, ''.join(output))

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
        return Grammar(definitions)

    def parseDefinition(self):
        # Definition <- Identifier LEFTARROW Expression
        identifier = self.consumet(TokenTypes.IDENTIFIER)
        self.consumet(TokenTypes.ARROW)
        return Definition([identifier.value, self.parseExpression()])

    def parseExpression(self):
        # Expression <- Sequence (SLASH Sequence)*
        output = [self.parseSequence()]
        while self.matcht(TokenTypes.PRIORITY):
            output.append(self.parseSequence())
        return Expression(output)

    def parseSequence(self):
        # Sequence <- Prefix*
        output = []
        while True:
            # Prefix <- (AND / NOT)? Labeled
            prefix = lambda x: x
            if self.matcht(TokenTypes.AND): prefix = And
            elif self.matcht(TokenTypes.NOT): prefix = Not
            suffix = self.parseLabeled()
            if suffix is None: break
            output.append(prefix(suffix))
        if len(output) > 1: return Sequence(output)
        else: return fio(output)

    def parseLabeled(self):
        # Labeled <- Suffix Label?
        output = self.parseSuffix()
        if self.matcht(TokenTypes.LABEL):
            label = self.consumet(TokenTypes.IDENTIFIER)
            return Label([label.value, output])
        return output


    def parseSuffix(self):
        # Suffix <- Primary (QUESTION / STAR / PLUS)?
        output = self.parsePrimary()
        suffix = lambda x: x
        if self.matcht(TokenTypes.QUESTION): suffix = Question
        elif self.matcht(TokenTypes.STAR): suffix = Star
        elif self.matcht(TokenTypes.PLUS): suffix = Plus
        return suffix(output)

    def parsePrimary(self):
        # Primary <- Identifier !LEFTARROW
        #          / OPEN Expression CLOSE
        #          / Capture   # Extension
        #          / Literal / Class / DOT
        if self.testt(TokenTypes.OPCAP) and self.peekt()._type == TokenTypes.IDENTIFIER:
            self.consumet(TokenTypes.OPCAP)
            return CaptureNode(Identifier(self.consumet(TokenTypes.IDENTIFIER).value))
        elif self.testt(TokenTypes.IDENTIFIER) and self.peekt()._type != TokenTypes.ARROW:
            return Identifier(self.consumet(TokenTypes.IDENTIFIER).value)
        elif self.testt(TokenTypes.LITERAL):
            return Literal(self.consumet(TokenTypes.LITERAL).value)
        elif self.testt(TokenTypes.STRING):
            return String(self.consumet(TokenTypes.STRING).value)
        elif self.testt(TokenTypes.CLASS):
            return Class(self.consumet(TokenTypes.CLASS).value)
        elif self.matcht(TokenTypes.DOT):
            return Dot()
        elif self.matcht(TokenTypes.OPEN):
            if self.matcht(TokenTypes.CLOSE): return []
            value = self.parseExpression()
            self.consumet(TokenTypes.CLOSE)
            return value
        elif self.matcht(TokenTypes.OPCB):
            if self.matcht(TokenTypes.CLCB): return []
            value = CaptureBlock(self.parseExpression())
            self.consumet(TokenTypes.CLCB)
            return value
        elif self.matcht(TokenTypes.OPLS):
            if self.matcht(TokenTypes.CLCB): return List([])
            output = []
            while not self.matcht(TokenTypes.CLCB):
                output.append(self.parseExpression())
            return List(output)
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


def grammarAsDict(grammar):
    dicts = [{x.value[0]: x.value[1]} for x in grammar.value]
    return {x: y for di in dicts for x, y in di.items()}


class Match:

    def __init__(self, grammar, start, data):
        self.g = grammarAsDict(grammar)
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
     OP_THROW,
     OP_CAP_OPEN,
     OP_CAP_CLOSE,
     OP_ATOM,
     OP_OPEN,
     OP_CLOSE,
     OP_CAPCHAR,
     OP_END,
    ) = range(24)

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
    Instructions.OP_THROW: 1,
    Instructions.OP_CAP_OPEN: 2,
    Instructions.OP_CAP_CLOSE: 2,
    Instructions.OP_ATOM: 1,
    Instructions.OP_OPEN: 0,
    Instructions.OP_CLOSE: 0,
    Instructions.OP_CAPCHAR: 0,
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
        self.compiler = compiler
        self.capture = capture
        self.isTerminal = isTerminal
        self.capId = capId
        self.backup = compiler.capture

    def __enter__(self):
        self.compiler.capture = True
        self.compiler.emit(
            "cap_open",
            self.isTerminal,
            self.capId and self.compiler._str(self.capId))
        return self

    def __exit__(self, _type, value, traceback):
        self.compiler.capture = self.backup
        self.compiler.emit(
            "cap_close",
            self.isTerminal,
            self.capId and self.compiler._str(self.capId))


class DisableCapture:
    def __init__(self, c):
        self.c = c
        self.backup = c.capture

    def __enter__(self):
        self.c.capture = False

    def __exit__(self, _type, value, traceback):
        self.c.capture = self.backup


def walker(tree, exclude=(), gather=()):
    stk = [tree]
    found = []
    while stk:
        current = stk.pop()
        if isinstance(current, exclude):
            pass
        elif isinstance(current, gather):
            if current not in found:
                found.append(current)
        elif isinstance(current, Node):
            stk.append(current.value)
        elif isinstance(current, list):
            for c in current:
                stk.append(c)
    return found


def markTerminals(node):
    for terminal in walker(node, Not, (String, Literal, Dot, Class)):
        terminal.capture = True


def followIdentifiers(d, skip, node):
    for identifier in walker(node, Not, Identifier):
        markTerminals(d[identifier.value])
        if identifier not in skip:
            followIdentifiers(d, skip, d[identifier.value])
        else:
            print("Skipping captures for rule `{}'".format(
                identifier.value))


def markCaptures(g):
    d = grammarAsDict(g)
    # Get all block capture operators
    blocks = walker(g, Not, CaptureBlock)
    # Get list of all identifiers that are not directly within block
    # capture operators
    skip = walker(g, (Not, CaptureBlock), Identifier)
    # Remove the identifiers that are indirectly used by the block
    # capture operators from the skip list
    for block in blocks:
        for ident in walker(block, Not, Identifier):
            if ident in skip:
                skip.remove(ident)
    # Kick off the marking of terminals
    for block in blocks:
        markTerminals(block)
        followIdentifiers(d, skip, block)
    return g


class Compiler:

    def __init__(self, grammar, capture=False):
        self.ga = markCaptures(grammar)
        # The start rule is just the first one for now
        self.start = grammar.value[0].value[0]
        # Write cursor for `self.code'
        self.pos = 0
        # Store the binary code that will be packed in the end of the
        # process. Each entry is a uint32_t generated with `gen()'
        self.code = []
        # Save all the call instructions that need to be updated with
        # the final address of where they want to call to.
        self.callsites = {}
        # Flag that signals if the grammar contains any capture
        # operators.
        self.hasCaptureOps = bool(
            walker(grammar, Not, (CaptureBlock, CaptureNode)))
        # Flag to signal if cap_{open,close} instructions should be
        # emitted or not. Changes throughout the execution of the
        # compiler.
        self.capture = False
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

    def _capture(self, isTerminal, capid=False):
        return Capture(self, self.capture, isTerminal, capid)

    def _disableCapture(self):
        return DisableCapture(self)

    def compileCaptureBlock(self, atom):
        with self._capture(1):
            self.compileAtom(atom.value)

    def compileCaptureNode(self, atom):
        assert(isinstance(atom.value, Identifier))
        self._str(atom.value.value)
        with self._capture(0, atom.value.value):
            self.compileAtom(atom.value)

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

    def compileString(self, string):
        currentPos = self.pos
        idx = self._str(string.value)
        self.emit("atom", idx)
        return self.pos - currentPos

    def compileLiteral(self, literal):
        currentPos = self.pos
        for i in literal.value:
            self.emit("char", ord(i))
            if self.capture or getattr(literal, 'capture', None):
                self.emit("capchar")
        return self.pos - currentPos

    def compileAny(self, atom):
        currentPos = self.pos
        self.emit("any")
        if self.capture or getattr(atom, 'capture', None):
            self.emit("capchar")
        return self.pos - currentPos

    def compileRange(self, theRange, capture):
        currentPos = self.pos
        left, right = theRange
        self.emit("span", ord(left), ord(right))
        if self.capture or capture:
            self.emit("capchar")
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

    def compileRangeOrLiteral(self, capture, thing):
        if isinstance(thing, list):
            return self.compileRange(thing, capture)
        lit = Literal(thing)
        if capture: lit.capture = capture
        return self.compileLiteral(lit)

    def compileClass(self, atom):
        """Generate code for matching classes of characters

        Classes of characteres can be either one interval, multiple
        intervals or sets of characters.

        If `atom' describes only one interval, a single `Span'
        instruction is generated.
        """
        compileRangeOrLiteral = functools.partial(
            self.compileRangeOrLiteral,
            getattr(atom, 'capture', None))
        if len(atom.value) == 1:
            return compileRangeOrLiteral(atom.value[0])
        else:
            return self._compileChoices(atom.value, compileRangeOrLiteral)

    def compileLabel(self, atom):
        label, sub = atom.value
        return self._compileChoices([sub, Throw(label)], self.cc)

    def compileThrow(self, atom):
        # The +2 ensures that labels can't be lower than 2.
        self.emit('throw', self._str(atom.value) + 2)

    def compileList(self, lst):
        self.emit('open')
        for i in lst.value: self.compileAtom(i)
        self.emit('close')

    def compileAtom(self, atom):
        if isinstance(atom, Literal): self.compileLiteral(atom)
        elif isinstance(atom, String): self.compileString(atom)
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
        elif isinstance(atom, Label): self.compileLabel(atom)
        elif isinstance(atom, Throw): self.compileThrow(atom)
        elif isinstance(atom, CaptureNode): self.compileCaptureNode(atom)
        elif isinstance(atom, CaptureBlock): self.compileCaptureBlock(atom)
        elif isinstance(atom, List): self.compileList(atom)
        else: raise Exception("Unknown atom %s" % atom)

    def genCode(self):
        # It's always 2 because invariant contains two instructions
        self.emit("call", 2)
        pos = self.emit("jump")
        # Above invariant + the cap_open below that appears if
        # hasCaptureOps is True.
        distance = self.hasCaptureOps and 3 or 2
        size = 0
        addresses = {}

        if self.hasCaptureOps:
            self.emit('cap_open', 0, self._str(self.start))
        for definition in self.ga.value:
            identifier, expression = definition.value
            addresses[identifier] = self.pos
            self.cc(expression)
            self.emit("return")
            size += self.pos - addresses[identifier]
        if self.hasCaptureOps:
            self.emit('cap_close', 0, self._str(self.start))
        self.code[pos-1] = gen("jump", size + distance)
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
        assembled = uint16(len(self.strings))
        for i in self.strings:
            assembled += uint8(len(i))
            s = i.encode('ascii')
            assembled += struct.pack('>' + ('s'*len(s)), *s)
        # Write code size & code
        assembled += uint16(len(self.code))
        assembled += code
        return assembled


uint8 = lambda v: struct.pack('>B', v)
uint16 = lambda v: struct.pack('>H', v)
readuint8 = lambda chunk: struct.unpack('>B', chunk)[0]
readuint16 = lambda chunk: struct.unpack('>H', chunk)[0]
readstring = lambda chunk: ''.join(struct.unpack('>' + 's'*len(chunk), chunk))


## --- Utilities ---

OP_MASK   = lambda c: (((c) & 0xf8000000) >> OPERATOR_OFFSET)
UOPERAND0 = lambda c: ((c) & 0x7ffffff)
UOPERAND1 = lambda c: (UOPERAND0(c) >> S2_OPERAND_SIZE)
UOPERAND2 = lambda c: (UOPERAND0(c) & ((1 << S2_OPERAND_SIZE) - 1))


def dbgcc(c, bc, header):
    if c[-1] == '\n': c = c[:-1]
    print('\033[92m{}\033[0m'.format(c.encode('utf-8')), end=':\n')
    cursor = 0
    # Parse header
    if header:
        headerSize = readuint16(bc[cursor:cursor+2]); cursor += 2
        print('Header(%s)' % headerSize)
        for i in range(headerSize):
            ssize = readuint8(bc[cursor]); cursor += 1
            content = readstring(bc[cursor:cursor+ssize]);
            print("   0x%02x: String(%2ld) %s" % (i, ssize, repr(content)))
            cursor += ssize
    # Parse body
        codeSize = readuint16(bc[cursor:cursor+2]); cursor += 2
        print('Code(%d)' % codeSize)
    else:
        codeSize = len(bc) / 4
    unpacked = struct.unpack('>' + ('I' * codeSize), bc[cursor:])
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

    test("Rule1 <- 'tx'", [
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

    test("_ <- 'a'", [
        Token(TokenTypes.IDENTIFIER, '_', line=0, pos=0),
        Token(TokenTypes.ARROW, line=0, pos=2),
        Token(TokenTypes.LITERAL, 'a', line=0, pos=5),
        Token(TokenTypes.END, line=0, pos=8),
    ])

    test("S_ <- 'a'", [
        Token(TokenTypes.IDENTIFIER, 'S_', line=0, pos=0),
        Token(TokenTypes.ARROW, line=0, pos=3),
        Token(TokenTypes.LITERAL, 'a', line=0, pos=6),
        Token(TokenTypes.END, line=0, pos=9),
    ])

    # Hex
    test("S <- '\\x30'", [
        Token(TokenTypes.IDENTIFIER, 'S', line=0, pos=0),
        Token(TokenTypes.ARROW, line=0, pos=2),
        Token(TokenTypes.LITERAL, '0', line=0, pos=5),
        Token(TokenTypes.END, line=0, pos=11),
    ])

    # Capture Node
    test("S <- %A", [
        Token(TokenTypes.IDENTIFIER, 'S', line=0, pos=0),
        Token(TokenTypes.ARROW, line=0, pos=2),
        Token(TokenTypes.OPCAP, line=0, pos=5),
        Token(TokenTypes.IDENTIFIER, 'A', line=0, pos=6),
        Token(TokenTypes.END, line=0, pos=7),
    ])

    # Capture Block Operator
    test("S <- %{ [a-z]* }", [
        Token(TokenTypes.IDENTIFIER, 'S', line=0, pos=0),
        Token(TokenTypes.ARROW, line=0, pos=2),
        Token(TokenTypes.OPCB, line=0, pos=5),
        Token(TokenTypes.CLASS, [['a', 'z']], line=0, pos=8),
        Token(TokenTypes.STAR, line=0, pos=13),
        Token(TokenTypes.CLCB, line=0, pos=15),
        Token(TokenTypes.END, line=0, pos=16),
    ])

    # Lists
    test("A <- !{ .* } .", [
        Token(TokenTypes.IDENTIFIER, 'A', line=0, pos=0),
        Token(TokenTypes.ARROW,           line=0, pos=2),
        Token(TokenTypes.NOT,             line=0, pos=5),
        Token(TokenTypes.OPLS,            line=0, pos=6),
        Token(TokenTypes.DOT,             line=0, pos=8),
        Token(TokenTypes.STAR,            line=0, pos=9),
        Token(TokenTypes.CLCB,            line=0, pos=11),
        Token(TokenTypes.DOT,             line=0, pos=13),
        Token(TokenTypes.END,             line=0, pos=14),
    ])

    # Atom Literals
    test('A <- { "atom" }', [
        Token(TokenTypes.IDENTIFIER, 'A', line=0, pos=0),
        Token(TokenTypes.ARROW,           line=0, pos=2),
        Token(TokenTypes.OPLS,            line=0, pos=5),
        Token(TokenTypes.STRING, 'atom',  line=0, pos=7),
        Token(TokenTypes.CLCB,            line=0, pos=14),
        Token(TokenTypes.END,             line=0, pos=15),
    ])

    # Error Labels
    test("A <- 'a'^lab", [
        Token(TokenTypes.IDENTIFIER, 'A',   line=0, pos=0),
        Token(TokenTypes.ARROW,             line=0, pos=2),
        Token(TokenTypes.LITERAL, 'a',      line=0, pos=5),
        Token(TokenTypes.LABEL,             line=0, pos=8),
        Token(TokenTypes.IDENTIFIER, 'lab', line=0, pos=9),
        Token(TokenTypes.END,               line=0, pos=12),
    ])
    test("S <- A^lab\nA <- 'a'", [
        Token(TokenTypes.IDENTIFIER, 'S',   line=0, pos=0),
        Token(TokenTypes.ARROW,             line=0, pos=2),
        Token(TokenTypes.IDENTIFIER, 'A',   line=0, pos=5),
        Token(TokenTypes.LABEL,             line=0, pos=6),
        Token(TokenTypes.IDENTIFIER, 'lab', line=0, pos=7),
        Token(TokenTypes.IDENTIFIER, 'A',   line=1, pos=11),
        Token(TokenTypes.ARROW,             line=1, pos=13),
        Token(TokenTypes.LITERAL, 'a',      line=1, pos=16),
        Token(TokenTypes.END,               line=1, pos=19),
    ])


def test_parser():
    test = functools.partial(test_runner, lambda x: Parser(x).parse)

    test('# ', Grammar([]))

    test("#foo\r\nR0 <- 'a'", Grammar([
        Definition([
            'R0', Expression([Literal('a')])])]))

    test(r"R0 <- '\\' [nrt'\"\[\]]", Grammar([
        Definition([
            'R0', Expression([
                Sequence([
                    Literal('\\'),
                    Class(['n', 'r', 't', "'", '"', '[', ']'])])])])]))

    test("R <- '\\r\\n' / '\\n' / '\\r'", Grammar([
        Definition([
            'R', Expression([
                Literal('\r\n'),
                Literal('\n'),
                Literal('\r')])])]))

    test('# foo\n R1 <- "a"\nR2 <- \'b\'', Grammar([
        Definition(['R1', Expression([String('a')])]),
        Definition(['R2', Expression([Literal('b')])])]))

    test("Definition1 <- 'tx'",
         Grammar([Definition(['Definition1', Expression([Literal('tx')])])]))

    test('Int <- [0-9]+', Grammar([
        Definition(['Int', Expression([Plus(Class([['0', '9']]))])])]))

    test('Foo <- [a-z_]', Grammar([
        Definition(['Foo', Expression([Class([['a', 'z'], '_'])])])]))

    test('EndOfFile <- !.', Grammar([
        Definition(['EndOfFile', Expression([Not(Dot())])])]))

    test("R0 <- 'oi' 'tenta'?", Grammar([
        Definition(['R0', Expression([
            Sequence([Literal('oi'), Question(Literal("tenta"))])])])]))

    test("Foo <- ('a' / 'b')+", Grammar([
        Definition(['Foo', Expression([
            Plus(Expression([Literal('a'), Literal('b')]))])])]))

    test("R0 <- 'a'\n      / 'b'\nR1 <- 'c'", Grammar([
        Definition(['R0', Expression([Literal('a'), Literal('b')])]),
        Definition(['R1', Expression([Literal('c')])])]))

    test("R0 <- R1 (',' R1)*\nR1 <- [0-9]+", Grammar([
        Definition(['R0', Expression([Sequence([
            Identifier('R1'),
            Star(Expression([Sequence([Literal(','), Identifier('R1')])]))])])]),
        Definition(['R1', Expression([Plus(Class([['0', '9']]))])])]))

    test('A <- !{ .* } .', Grammar([
        Definition(['A', Expression([
            Sequence([
                Not(List([Expression([Star(Dot())])])),
                Dot(),
            ])
        ])])
    ]))

    test(r"""# first line with comment
Spacing    <- (Space / Comment)*
Comment    <- '#' (!EndOfLine .)* EndOfLine
Space      <- ' ' / '\t' / EndOfLine
EndOfLine  <- '\r\n' / '\n' / '\r'
EndOfFile  <- !.
    """, Grammar([
        Definition(['Spacing', Expression([
            Star(Expression([
                Identifier('Space'),
                Identifier('Comment')]))])]),
        Definition(['Comment', Expression([
            Sequence([
                Literal('#'),
                Star(Expression([Sequence([Not(Identifier('EndOfLine')), Dot()])])),
                Identifier('EndOfLine')])])]),
        Definition(['Space', Expression([
            Literal(' '),
            Literal('\t'),
            Identifier('EndOfLine')])]),
        Definition(['EndOfLine', Expression([
            Literal('\r\n'),
            Literal('\n'),
            Literal('\r')])]),
        Definition(['EndOfFile', Expression([Not(Dot())])])]))

    # Some real stuff

    test(csv, Grammar([
        Definition(['File', Expression([Star(Identifier('CSV'))])]),
        Definition(['CSV', Expression([Sequence([
            Identifier('Val'),
            Star(Expression([Sequence([Literal(','), Identifier('Val')])])),
            Literal('\n')])])]),
        Definition(['Val', Expression([
            Star(Expression([Sequence([Not(Class([',', '\n'])), Dot()])]))])])]))

    test(arith, Grammar([
        Definition(['Add', Expression([Sequence([Identifier('Mul'),
                                      Literal('+'),
                                      Identifier('Add')]),
                            Identifier('Mul')])]),
        Definition(['Mul', Expression([Sequence([Identifier('Pri'),
                                      Literal('*'),
                                      Identifier('Mul')]),
                            Identifier('Pri')])]),
        Definition(['Pri', Expression([Sequence([Literal('('),
                                      Identifier('Add'),
                                      Literal(')')]),
                            Identifier('Num')])]),
        Definition(['Num', Expression([Plus(Class([['0', '9']]))])])]))

    # Captures

    test("S <- %{ A }\nA <- 'a'\n", Grammar([
        Definition(['S', Expression([
            CaptureBlock(Expression([Identifier('A')]))])]),
        Definition(['A', Expression([Literal('a')])])
    ]))

    test("S <- %A\nA <- %{ 'a' }", Grammar([
        Definition(['S', Expression([CaptureNode(Identifier('A'))])]),
        Definition(['A', Expression([CaptureBlock(Expression([Literal('a')]))])])
    ]))

    # Error Labels

    test("A <- 'a'^label", Grammar([
        Definition(['A', Expression([Label([
            'label', Literal('a')
        ])])])
    ]))

    test("S <- %A^label\nA <- %{ 'a' }", Grammar([
        Definition(['S', Expression([
            Label(['label', CaptureNode(Identifier('A'))]),
        ])]),
        Definition(['A', Expression([
            CaptureBlock(Expression([Literal('a')]))
        ])]),
    ]))


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
    e = Match(Grammar([]), '', "affbcdea&2")
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

    e = Match(Parser("Lit <- 'test'").parse(), 'Lit', "testando")
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


def test_rewrite():
    p = lambda code: Parser(code).run()


def test_compile():
    bn = lambda *bc: struct.pack('>' + ('I' * len(bc)), *bc)
    def cc(code, **f):
        parsed = Parser(code).run()
        genCode = lambda: Compiler(parsed, **f).genCode()
        dbgcc(code, genCode(), header=False)
        return genCode()

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

    # Error labels

    assert(cc("S <- 'a'^f") == bn(
        gen("call", 2),
        gen("jump", 7),
        gen("choice", 3),
        gen("char", ord('a')),
        gen("commit", 2),
        gen("throw", 2),
        gen("return"),
        gen("halt"),
    ))

    # Lists
    assert(cc("A <- !{ .* } .") == bn(
        gen("call",   0x02),    # 0x00: Call 0x02
        gen("jump",   0x0c),    # 0x01: Jump 0x0c
        gen("choice", 0x08),    # 0x02: Choice 0x08

        gen("open"),            # 0x03: Open
        gen("choice", 0x03),    # 0x04: Choice 0x03
        gen("any"),             # 0x05: Any
        gen("commit", -2),      # 0x06: Commit 0x7fffffe
        gen("close"),           # 0x07: Close

        gen("commit", 1),       # 0x08: Commit 0x01
        gen("fail"),            # 0x09: Fail

        gen("any"),             # 0x0a: Any

        gen("return"),          # 0x0b: Return
        gen("halt"),            # 0x0c: Halt
    ))

    # Atom
    assert(cc('A <- { "test" }\n') == bn(
        gen("call",   0x02),    # 0x00: Call 0x02
        gen("jump",   0x06),    # 0x01: Jump 0x06
        gen("open"),            # 0x02: Open
        gen("atom",   0x00),    # 0x03: Atom 0x00
        gen("close"),           # 0x04: Close
        gen("return"),          # 0x05: Return
        gen("halt"),            # 0x06: Halt
    ))

    # Captures
    assert(cc("S <- %{ 'a' }") == bn(
        gen("call", 0x02),      # 0x00: Call 0x02
        gen("jump", 0x08),      # 0x01: Jump 0x07
        gen("cap_open", 0, 0),  # 0x02: CapOpen 0 0
        gen("cap_open", 1, 0),  # 0x03: CapOpen 1 0
        gen("char", ord('a')),  # 0x04: Char 'a'
        gen("capchar"),         # 0x05: CapChar
        gen("cap_close", 1, 0), # 0x06: CapClose 1 0
        gen("return"),          # 0x07: Return
        gen("cap_close", 0, 0), # 0x08: CapClose 0 0
        gen("halt"),            # 0x07: Halt
    ))

    assert(cc("S <- %{ 'a' 'b' }") == bn(
        gen("call", 0x02),      # 0x00: Call 0x02
        gen("jump", 0x0a),      # 0x01: Jump 0x09
        gen("cap_open", 0, 0),  # 0x02: CapOpen 0 0
        gen("cap_open", 1, 0),  # 0x03: CapOpen 1 0
        gen("char", ord('a')),  # 0x04: Char 'a'
        gen("capchar"),         # 0x05: CapChar
        gen("char", ord('b')),  # 0x06: Char 'a'
        gen("capchar"),         # 0x07: CapChar
        gen("cap_close", 1, 0), # 0x08: CapClose 1 0
        gen("return"),          # 0x09: Return
        gen("cap_close", 0, 0), # 0x0a: CapOpen 0 0
        gen("halt"),            # 0x0b: Halt
    ))

    assert(cc("S <- %A\nA <- %{ 'a' }") == bn(
        gen("call", 0x02),      # 0x00: Call 0x02
        gen("jump", 0x0c),      # 0x01: Jump 0x0b
        gen("cap_open", 0, 0),  # 0x02: CapOpen 0 0
        gen("cap_open", 0, 1),  # 0x03: CapOpen 0 1
        gen("call", 0x03),      # 0x04: Call 0x03
        gen("cap_close", 0, 1), # 0x05: CapClose 0 0
        gen("return"),          # 0x06: Return
        gen("cap_open", 1, 0),  # 0x07: CapOpen 1 0
        gen("char", ord('a')),  # 0x08: Char 0x97
        gen("capchar"),         # 0x09: CapChar
        gen("cap_close", 1, 0), # 0x0a: CapClose 1 0
        gen("return"),          # 0x0b: Return
        gen("cap_close", 0, 0), # 0x0c: CapClose 1 0
        gen("halt"),            # 0x0d: Halt
    ))


def test_compile_header():
    def cc(src, **f):
        c = Compiler(Parser(src).run(), capture=True, **f)
        c.genCode()
        return c.strings

    # Name of the rule and char
    assert(cc("S <- 'a'") == [])
    # Name of the rules and set of chars
    assert(cc("S <- %D\nD <- [0-9]+") == ['S', 'D'])
    # Name of the rules and set of chars without dups
    assert(cc("S <- %A %B\nA<-'a'\nB<-'b'") == ['S', 'A', 'B'])


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


def outputName(args):
    name, ext = os.path.splitext(args.grammar)
    return args.output or name + ext.replace('peg', 'bin')


def compileG(args, grammarSrc, grammar):
    with io.open(outputName(args), 'wb') as out:
        compiled = Compiler(grammar, capture=args.capture).assemble()
        out.write(compiled)
        if not args.quiet: dbgcc(grammarSrc, compiled, header=True)


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
    except:
        os.unlink(outputName(args))
        raise


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
        '-o', '--output', dest='output', action='store',
        help='Output of the compiled Grammar')
    parser.add_argument(
        '-q', '--quiet', dest='quiet', action='store_true', default=False,
        help='Quiet')
    parser.add_argument(
        '-p', '--capture', dest='capture', action='store_true', default=False,
        help='Add CAPTURE instructions')
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
