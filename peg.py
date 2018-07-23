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
import enum
import functools
import io
import os
import pprint

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
        return self.t(TokenTypes.CLASS, ranges or ''.join(chars))

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
        if self.matchc('#'):
            while not self.testc('\n') and self.peekc():
                self.nextc()
            self.cleanspaces()

    def cleanspaces(self):
        while True:
            if self.matchc('\\'):
                if self.matchc('n'):
                    self.line += 1
            elif self.peekc() and self.peekc().isspace(): self.nextc()
            else: break

    def parseDefinitions(self):
        # Grammar <- Spacing Definition+ EndOfFile
        definitions = {}
        while not self.testt(TokenTypes.END):
            definition = self.parseDefinition()
            if not definition: break
            definitions.update(definition)
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
            output = ['%s at line %d' % (str(exc), self.line)]
            output.append('')
            code = self.code
            mark = self.token_start
            output.append(
                code[:mark] +
                '\u001b[41m' +
                code[mark] +
                '\u001b[0m' +
                code[mark+1:]
            )
            exc.msg = '\n'.join(output);
            raise exc


class Eval:

    def __init__(self, grammar, start, data):
        self.g = grammar
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

    def evalClass(self, atom):
        d = self.pos
        value = atom.value
        if not self.current(): return False, None

        if isinstance(value, list):
            for [left, right] in value:
                if left <= self.current() <= right:
                    value = self.current()
                    self.advance()
                    return True, value
        else:
            for char in value:
                if self.current() == char:
                    value = self.current()
                    self.advance();
                    return True, value
        return False, None

    def evalLiteral(self, atom):
        d = self.pos
        for c in atom.value:
            if self.current() == c:
                self.advance()
        output = self.ret(d)
        return output != None, output

    def evalDot(self, atom):
        d = self.pos
        self.advance()
        output = self.ret(d)
        return output != None, output

    def evalPlus(self, atom):
        match, value = self.evalAtom(atom.value)
        if match:
            out = [value] + self.evalStar(Star(atom.value))[1]
            return True, fio(out)
        return False, None

    def evalQuestion(self, atom):
        match, value = self.evalAtom(atom.value)
        return True, value

    def evalStar(self, atom):
        out = []
        while True:
            match, value = self.evalAtom(atom.value)
            if not match: break
            out.append(value)
        return True, out

    def evalNot(self, atom):
        d = self.pos
        match, value = self.evalAtom(atom.value)
        if match:
            # We got a match. To negate it, we'll reset the cursor
            # position to prior to the evaluation and return None.
            self.pos = d
            return False, None
        else:
            return True, None

    def evalSequence(self, atom):
        d = self.pos
        out = []
        for sa in atom.value:
            match, value = self.evalAtom(sa)
            if match and value: out.append(value)
            elif match: continue
            else:
                self.pos = d
                return False, None
        return True, fio(out)

    def evalExpression(self, atom):
        d = self.pos
        for sa in atom.value:
            match, value = self.evalAtom(sa)
            if match: return True, value
        return False, None

    def evalIdentifier(self, atom):
        return self.evalAtom(self.g[atom.value])

    def evalAtom(self, atom):
        if isinstance(atom, Class):
            return self.evalClass(atom)
        elif isinstance(atom, Literal):
            return self.evalLiteral(atom)
        elif isinstance(atom, Dot):
            return self.evalDot(atom)
        elif isinstance(atom, Identifier):
            return self.evalIdentifier(atom)
        elif isinstance(atom, Plus):
            return self.evalPlus(atom)
        elif isinstance(atom, Star):
            return self.evalStar(atom)
        elif isinstance(atom, Not):
            return self.evalNot(atom)
        elif isinstance(atom, Sequence):
            return self.evalSequence(atom)
        elif isinstance(atom, Expression):
            return self.evalExpression(atom)
        elif isinstance(atom, Question):
            return self.evalQuestion(atom)
        raise Exception('Unexpected atom')

    def run(self):
        return self.evalAtom(self.g[self.start])


def peg(g):
    p = Parser(g)
    return p.parse()

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


def test_runner_eval(g, start, data, expected):
    print('\033[92m{}\033[0m'.format(repr(g)), end=':\n    ')
    value = Eval(Parser(g).parse(), start, data).run()
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
        Token(TokenTypes.CLASS, ',\n',    line=0, pos=7),
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


def test_parser():
    test = functools.partial(test_runner, lambda x: Parser(x).parse)

    test('# ', {})

    test('#foo\r\nR0 <- "a"', {'R0': Literal('a')})

    test(r"R0 <- '\\' [nrt'\"\[\]]", {
        'R0': Sequence([Literal('\\'), Class('nrt\'"[]')]),
    })

    test("R <- '\r\n' / '\n' / '\r'\\n", {
        'R': Expression([Literal('\r\n'), Literal('\n'), Literal('\r')])
    })

    test('# foo\n R1 <- "a"\nR2 <- \'b\'', {
        'R1': Literal('a'),
        'R2': Literal('b')
    })

    test('Definition1 <- "tx"', {'Definition1': Literal('tx')})

    test('Int <- [0-9]+', {'Int': Plus(Class([['0', '9']]))})

    test('EndOfFile <- !.', {'EndOfFile': Not(Dot())})

    test('R0 <- "oi" "tenta"?', {'R0': Sequence([Literal('oi'), Question(Literal("tenta"))])})

    test('Foo <- ("a" / "b")+', {'Foo': Plus(Expression([Literal('a'), Literal('b')]))})

    test('R0 <- "a"\\n      / "b"\\nR1 <- "c"', {
        'R0': Expression([Literal('a'), Literal('b')]),
        'R1': Literal('c'),
    })

    test('R0 <- R1 ("," R1)*\\nR1 <- [0-9]+', {
        'R0': Sequence([
            Identifier('R1'),
            Star(Sequence([Literal(','), Identifier('R1')]))]),
        'R1': Plus(Class([['0', '9']])),
    })

    test(r"""# first line with comment
Spacing    <- (Space / Comment)*
Comment    <- '#' (!EndOfLine .)* EndOfLine
Space      <- ' ' / '\t' / EndOfLine
EndOfLine  <- '\r\n' / '\n' / '\r'
EndOfFile  <- !.
          """, {
              'Spacing': Star(Expression([Identifier('Space'), Identifier('Comment')])),
              'Comment': Sequence([
                  Literal('#'),
                  Star(Sequence([Not(Identifier('EndOfLine')), Dot()])),
                  Identifier('EndOfLine'),
              ]),
              'Space': Expression([Literal(' '), Literal('\t'), Identifier('EndOfLine')]),
              'EndOfLine': Expression([Literal('\r\n'), Literal('\n'), Literal('\r')]),
              'EndOfFile': Not(Dot()),
          })

    # Some real stuff

    test(csv, {
        'File': Star(Identifier('CSV')),
        'Val': Star(Sequence([Not(Class(',\n')), Dot()])),
        'CSV': Sequence([
            Identifier('Val'),
            Star(Sequence([Literal(','), Identifier('Val')])),
            Literal('\n')]),
    })

    test(arith, {
        'Add': Expression([Sequence([Identifier('Mul'), Literal('+'), Identifier('Add')]),
                           Identifier('Mul')]),
        'Mul': Expression([Sequence([Identifier('Pri'), Literal('*'), Identifier('Mul')]),
                           Identifier('Pri')]),
        'Num': Plus(Class([['0', '9']])),
        'Pri': Expression([Sequence([Literal('('), Identifier('Add'), Literal(')')]),
                Identifier('Num')]),
    })


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


    value = Parser('AtoC <- [a-c]\\nNoAtoC <- !AtoC .\\nEOF < !.\\n').run()
    expected = r'''Missing the dash in the arrow at line 2

AtoC <- [a-c]
NoAtoC <- !AtoC .
EOF \x1b[31m\x1b[0m !.
'''
    #assert(value == expected)


def test_eval():
    e = Eval({}, '', "affbcdea&2")
    assert(e.evalAtom(Class('af'))         == (True, 'a'));   assert(e.pos == 1)
    assert(e.evalAtom(Class('gd'))         == (False, None)); assert(e.pos == 1)
    assert(e.evalAtom(Class('xyf'))        == (True, 'f'));   assert(e.pos == 2)
    assert(e.evalAtom(Class([['a', 'f']])) == (True, 'f'));   assert(e.pos == 3)
    assert(e.evalAtom(Class([['a', 'f']])) == (True, 'b'));   assert(e.pos == 4)
    assert(e.evalAtom(Class([['a', 'f']])) == (True, 'c'));   assert(e.pos == 5)
    assert(e.current() == 'd');                               assert(e.pos == 5)
    assert(e.evalAtom(Literal('a'))        == (False, None)); assert(e.pos == 5)
    assert(e.evalAtom(Literal('d'))        == (True, 'd'));   assert(e.pos == 6)
    assert(e.current() == 'e');                               assert(e.pos == 6)
    assert(e.evalAtom(Dot())               == (True, 'e'));   assert(e.pos == 7)
    assert(e.evalAtom(Dot())               == (True, 'a'));   assert(e.pos == 8)
    assert(e.evalAtom(Dot())               == (True, '&'));   assert(e.pos == 9)
    assert(e.evalAtom(Dot())               == (True, '2'));   assert(e.pos == 10)
    assert(e.evalAtom(Dot())               == (False, None)); assert(e.pos == 10)

    e = Eval(Parser('Lit <- "test"').parse(), 'Lit', "testando")
    assert(e.evalAtom(Identifier('Lit')) == (True, "test"))

    e = Eval(Parser('Digit <- [0-9]').parse(), 'Digit', "42f")
    assert(e.evalAtom(Identifier('Digit')) == (True, "4"))
    assert(e.evalAtom(Identifier('Digit')) == (True, "2"))
    assert(e.evalAtom(Identifier('Digit')) == (False, None))
    assert(e.current() == 'f')

    e = Eval(Parser('Digit <- [0-9]+').parse(), 'Digit', "42f")
    assert(e.evalAtom(Identifier('Digit')) == (True, ['4', '2']))
    e = Eval(Parser('Digit <- [0-9]*').parse(), 'Digit', "2048f")
    assert(e.evalAtom(Identifier('Digit')) == (True, ['2', '0', '4', '8']))
    e = Eval(Parser('Digit <- [0-9]*').parse(), 'Digit', '2048')
    assert(e.evalAtom(Identifier('Digit')) == (True, ['2', '0', '4', '8']))

    e = Eval(Parser('AtoC <- [a-c]\\nNoAtoC <- !AtoC .\\nEOF <- !.\\n').parse(), '', 'abcdef')
    assert(e.evalAtom(Identifier('NoAtoC')) == (False, None))  ; assert(e.pos == 0)
    assert(e.evalAtom(Identifier('AtoC'))   == (True, 'a'))    ; assert(e.pos == 1)
    assert(e.evalAtom(Identifier('NoAtoC')) == (False, None))  ; assert(e.pos == 1)
    assert(e.evalAtom(Identifier('AtoC'))   == (True, 'b'))    ; assert(e.pos == 2)
    assert(e.evalAtom(Identifier('AtoC'))   == (True, 'c'))    ; assert(e.pos == 3)
    assert(e.evalAtom(Identifier('AtoC'))   == (False, None))  ; assert(e.pos == 3)
    assert(e.evalAtom(Identifier('AtoC'))   == (False, None))  ; assert(e.pos == 3)
    assert(e.evalAtom(Identifier('NoAtoC')) == (True, 'd'))    ; assert(e.pos == 4)
    assert(e.evalAtom(Identifier('EOF'))    == (False, None))  ; assert(e.pos == 4)
    assert(e.evalAtom(Identifier('AtoC'))   == (False, None))  ; assert(e.pos == 4)
    assert(e.evalAtom(Identifier('NoAtoC')) == (True, 'e'))    ; assert(e.pos == 5)
    assert(e.evalAtom(Identifier('NoAtoC')) == (True, 'f'))    ; assert(e.pos == 6)
    assert(e.evalAtom(Identifier('NoAtoC')) == (False, None))  ; assert(e.pos == 6)
    assert(e.evalAtom(Identifier('AtoC'))   == (False, None))  ; assert(e.pos == 6)
    assert(e.evalAtom(Identifier('EOF'))    == (True, None))   ; assert(e.pos == 6)

    e = Eval(Parser('EOF <- !.\\nALL <- [a-f]').parse(), 'EOF', 'f')
    assert(e.evalAtom(Identifier('EOF')) == (False, None))
    assert(e.evalAtom(Identifier('ALL')) == (True, 'f'))
    e = Eval(Parser(arith).parse(), 'Add', "12+34*56")
    assert(e.evalAtom(Identifier('Add')) == (True, [
        ['1', '2'],
        '+', [['3', '4'],
              '*',
              ['5', '6']]]))

    e = Eval(Parser(csv).parse(), 'File', "Name,Num,Lang\nLink,3,pt-br\n")
    assert(e.evalAtom(Identifier('File')) == (True, [[['N', 'a', 'm', 'e'],
      [[',', ['N', 'u', 'm']],
       [',', ['L', 'a', 'n', 'g']]],
      '\n'],
     [['L', 'i', 'n', 'k'],
      [[',', ['3']],
       [',', ['p', 't', '-', 'b', 'r']]], '\n']]))

    e = Eval(Parser("EndOfLine <- '\r\n' / '\n' / '\r'").parse(), 'EndOfLine', "\n\r\n\r")
    assert(e.evalAtom(Identifier('EndOfLine')) == (True, '\n')); assert(e.pos == 1);

    e = Eval(Parser("""

NL <- '\r\n'
    / '\n'
    / '\r'
    / '\t'

C <- [a-z]+
   / NL

""").parse(), 'EndOfLine', "tst\n\r\t")

    assert(e.evalAtom(Identifier('C')) == (True, ['t', 's', 't']))  ; assert(e.pos == 3)
    assert(e.evalAtom(Identifier('C')) == (True, '\n'))             ; assert(e.pos == 4)
    assert(e.evalAtom(Identifier('C')) == (True, '\r'))
    assert(e.evalAtom(Identifier('C')) == (True, '\t'))


def test():
    test_tokenizer()
    test_parser()
    # test_parse_errors()
    test_eval()


def main():
    parser = argparse.ArgumentParser(
        description='Parse structured data with Parsing Expression Grammars')
    parser.add_argument(
        '-g', '--grammar', dest='grammar', action='store',
        help='Grammar File')
    parser.add_argument(
        '-d', '--data', dest='data', action='store',
        help='Data File')
    parser.add_argument(
        '-s', '--start', dest='start', action='store',
        help='Start rule. Which rule the parser should start at.')
    args = parser.parse_args()

    with io.open(os.path.abspath(args.grammar), 'r') as grammarFile:
        grammar = Parser(grammarFile.read()).run()
    with io.open(os.path.abspath(args.data), 'r') as dataFile:
        output = Eval(grammar, args.start, dataFile.read()).run()
        pprint.pprint(output)

if __name__ == '__main__':
    test()
    main()
