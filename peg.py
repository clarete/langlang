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

import enum
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
    def __init__(self, _type, value=None):
        self._type = _type
        self.value = value
    def __repr__(self):
        value = " " + repr(self.value) if self.value else ""
        return "Token({}{})".format(self._type, value)
    def __eq__(self, other):
        return (isinstance(other, self.__class__) and
                other._type == self._type and
                other.value == self.value)

class Node:
    def __repr__(self):
        return "{}()".format(self.__class__.__name__)
    def __eq__(self, other):
        return isinstance(other, self.__class__)

class And(Node): pass

class Not(Node): pass

class Question(Node): pass

class Star(Node): pass

class Plus(Node): pass

class Primary(Node):
    def __init__(self, name=None):
        self.name = name
    def __repr__(self):
        name = "{}".format(repr(self.name)) if self.name else ''
        return "{}({})".format(self.__class__.__name__, name)
    def __eq__(self, other):
        return (isinstance(other, self.__class__) and
                other.name == self.name)

class Identifier(Primary): pass

class Literal(Primary): pass

class Class(Primary): pass

class Dot(Primary): pass

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
        self.token = None

    def peekc(self, n=0):
        if self.pos+n >= len(self.code): return None
        return self.code[self.pos+n]

    def nextc(self):
        self.pos += 1
        return self.peekc()

    def testc(self, c):
        return self.peekc() == c

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
        raise SyntaxError("Expected token %s, received %s" % (
            repr(t), repr(self.token._type)))

    def lex(self):
        self.spacing()
        if self.peekc() is None: return Token(TokenTypes.END)

        # Identifier <- IdentStart IdentCont* Spacing
        if self.peekc().isalpha():
            d = self.pos
            while self.peekc() and self.peekc().isalnum(): self.pos += 1
            return Token(TokenTypes.IDENTIFIER, self.code[d:self.pos])
        elif self.matchc('"'):
            d = self.pos
            while self.peekc() and not self.testc('"'): self.nextc()
            if not self.matchc('"'): raise SyntaxError("Expected end of string")
            return Token(TokenTypes.LITERAL, self.code[d:self.pos-1])
        elif self.matchc('['):
            d = self.pos
            while not self.testc(']'): self.nextc()
            if not self.matchc(']'): raise SyntaxError("Expected end of class")
            return Token(TokenTypes.CLASS, self.code[d:self.pos-1])
        elif self.testc('<') and self.peekc(1) == '-':
            self.nextc(); self.nextc()
            return Token(TokenTypes.ARROW)
        elif self.matchc('('):
            return Token(TokenTypes.OPEN)
        elif self.matchc(')'):
            return Token(TokenTypes.CLOSE)
        elif self.matchc('/'):
            return Token(TokenTypes.PRIORITY)
        elif self.matchc('.'):
            return Token(TokenTypes.DOT)
        elif self.matchc('*'):
            return Token(TokenTypes.STAR)
        elif self.matchc('+'):
            return Token(TokenTypes.PLUS)
        elif self.matchc('!'):
            return Token(TokenTypes.NOT)
        else:
            raise SyntaxError("Unexpected char `{}'".format(self.peekc()))

    def spacing(self):
        self.cleanspaces()
        # ’#’ (!EndOfLine .)* EndOfLine
        if self.matchc('#'):
            while not self.matchc('\n'):
                if not self.peekc(): break
                self.nextc()
                self.line += 1
            self.cleanspaces()

    def cleanspaces(self):
        while True:
            if self.matchc('\n'): self.line += 1
            elif self.peekc() and self.peekc().isspace(): self.nextc()
            else: break

    def parse(self):
        try:
            self.nextt()
            return self.parseDefinitions()
        except SyntaxError as exc:
            print(self.code)

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
        return fio(output)

    def parseSequence(self):
        # Sequence <- Prefix*
        output = []
        while True:
            # Prefix <- (AND / NOT)? Suffix
            if self.matcht(TokenTypes.AND):
                output.append(And())
            elif self.matcht(TokenTypes.NOT):
                output.append(Not())
            suffix = self.parseSuffix()
            if suffix is None: break
            output.append(suffix)
        return fio(output)

    def parseSuffix(self):
        # Suffix <- Primary (QUESTION / STAR / PLUS)?
        output = [self.parsePrimary()]
        if self.matcht(TokenTypes.QUESTION):
            output.append(Question())
        elif self.matcht(TokenTypes.STAR):
            output.append(Star())
        elif self.matcht(TokenTypes.PLUS):
            output.append(Plus())
        return fio(output)

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


def peg(g):
    p = Parser(g)
    return p.parse()


def test():

    def t(x):
        pa = Parser(x)
        ts = []
        while True:
            t = pa.nextt()
            ts.append(t)
            if t._type == TokenTypes.END: break
        return ts

    p = lambda x: Parser(x).parse()
    pp = pprint.pprint

    # # # # Tokenizer # # # #

    # pp(t('Rule1 <- "tx"'))
    assert(t('Rule1 <- "tx"') == [
        Token(TokenTypes.IDENTIFIER, 'Rule1'),
        Token(TokenTypes.ARROW),
        Token(TokenTypes.LITERAL, 'tx'),
        Token(TokenTypes.END),
    ])

    # pp(t('Value <- (![,\n] .)*'))
    assert(t('Value <- (![,\n] .)*') == [
        Token(TokenTypes.IDENTIFIER, 'Value'),
        Token(TokenTypes.ARROW),
        Token(TokenTypes.OPEN),
        Token(TokenTypes.NOT),
        Token(TokenTypes.CLASS, ',\n'),
        Token(TokenTypes.DOT),
        Token(TokenTypes.CLOSE),
        Token(TokenTypes.STAR),
        Token(TokenTypes.END),
    ])

    # # # # Parser # # # #

    def ppp(g, expected):
        print(repr(g), end=':\n    ')
        pp(p(g))
        assert(p(g) == expected)

    ppp('# ', {})

    ppp('# foo\n R1 <- "a"', {'R1': Literal('a')})

    ppp('Rule1 <- "tx"', {'Rule1': Literal('tx')})

    ppp('Int <- [0-9]+', {'Int': [Class('0-9'), Plus()]})

    ppp('EndOfFile <- !.', {'EndOfFile': [Not(), Dot()]})

    ppp('R0 <- "a"\nR1 <- "b"', {'R0': Literal('a'), 'R1': Literal('b')})

    ppp('Foo <- ("a" / "b")+', {'Foo': [[Literal('a'), Literal('b')], Plus()]})

    ppp('R0 <- "a"\n      / "b"\nR1 <- "c"', {
        'R0': [Literal('a'), Literal('b')],
        'R1': Literal('c'),
    })

    ppp('R0 <- R1 ("," R1)*\nR1 <- [0-9]+', {
        'R0': [Identifier('R1'), [[Literal(','), Identifier('R1')], Star()]],
        'R1': [Class('0-9'), Plus()],
    })

    # ppp('x <- "a', {})

    # Some real stuff
    csv = '''
File <- CSV*
CSV <- Value ( "," Value )* "\n"
Value <- (![,\n] .)*
'''

    # pp(t(csv))
    pp(p(csv))



def main():
    pass


if __name__ == '__main__':
    test()
    main()
