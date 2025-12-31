const TINY_GRAMMAR = `Expr     <- Multi (( '+' / '-' ) Multi)*
Multi    <- Primary (( '*' / '/' ) Primary)*
Primary  <- Call / Id / Num / '(' Expr ')'

Call     <- Id '(' Params ')'
Params   <- (Expr (',' Expr)*)?

Num      <- [1-9][0-9]* / '0'
Id       <- [a-zA-Z_][a-zA-Z0-9_]*
`;

const TINY_INPUT = `
he+1
`;

const JSON_GRAMMAR = `// https://www.rfc-editor.org/rfc/rfc8259

JSON    <- Value^jsonValue EOF^eof
Value   <- Object / Array / String / Number / 'true' / 'false' / 'null'
Array   <- '[' (Value (',' Value^itemAfterComma)*)? ']'^arrayClose
Object  <- '{' (Member (',' Member^memberAfterComma)*)? '}'^objectClose
Member  <- String ':'^memberColon Value^memberValue

// Whitespaces are not allowed after the MINUS sign
Number  <- '-'? #(Int Frac? Exp?)
Int     <- '0' / ([1-9][0-9]*)
Frac    <- '.' [0-9]+^fracDigits
Exp     <- [eE][-+]?[0-9]+^expDigits

// Whitespaces are part of the string within quotes
String  <- '"' #(Char* ('"'^strClose))
Char    <- #(Escape / Unicode / (!'"' .))
Escape  <- '\\\\' ["\\\\/bfnrt]
Unicode <- #('\\\\' 'u' Hex^chrH1 Hex^chrH2 Hex^chrH3 Hex^chrH4)
Hex     <- [0-9A-Fa-f]

// Recovery Expressions
eof              <- .*
itemAfterComma   <- (![,\\]] .)*
memberAfterComma <- (![,}] .)*
memberValue      <- (![,}] .)*
memberColon      <-
arrayClose       <-
objectClose      <-
strClose         <-
jsonValue        <-
`;

const JSON_INPUT = `{ "name": "John", "age": 30, "city": "New York" }`;

const JSON_STRIPPED_GRAMMAR = `JSON    <- Value EOF
Value   <- Object / Array / String / Number / 'true' / 'false' / 'null'
Array   <- '[' (Value (',' Value)*)? ']'
Object  <- '{' (Member (',' Member)*)? '}'
Member  <- String ':' Value
Number  <- '-'? ('0' / ([1-9][0-9]*)) ('.' [0-9]+)? ([eE][-+]?[0-9]+)?
String  <- '"' (!["\\\\] .
              / ('\\\\' (["\\\\/bfnrt] / 'u' [0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f])))*
           '"'`;

const CSV_GRAMMAR = `File <- Line*
Line <- Val #((',' Val)* '\\n')
Val  <- (![,\\n] .)*`;

const CSV_INPUT = `name,age,city
John,30,New York
Jane,25,Los Angeles
Jim,35,Chicago
`;

const LANGLANG_GRAMMAR = `// The package \`langlang_syntax\` aims to implement the following
// syntax that is based on Parsing Expression Grammars with additional
// conservative extensions, such as an import system and automatic
// white space handling.

// Hierarchical syntax
Grammar     <- Import* Definition* EOF^eof
Import      <- "@import" Identifier^MissingImportName ("," Identifier)* "from"^MissingImportFrom Literal^MissingImportSrc
Definition  <- Identifier LEFTARROW Expression // this always succeeds, there's no point in labeling

Expression  <- Sequence (SLASH Sequence)*
Sequence    <- Prefix*
Prefix      <- ("#" / "&" / "!")? Labeled
Labeled     <- Suffix #(([^⇑] Identifier^MissingLabel)?)
Suffix      <- Primary ("?" / "*" / "+" / Superscript)?
Primary     <- Identifier !LEFTARROW
             / "(" Expression ")"^MissingClosingParen
             / List / Literal / Class / Any
List        <- "{" (!"}" Expression)* "}"^MissingClosingCurly

// Lexical syntax
Identifier  <- [a-zA-Z_][a-zA-Z0-9_]*
Literal     <- ['] #((!['] Char)* [']^MissingClosingSQuote)
             / ["] #((!["] Char)* ["]^MissingClosingDQuote)
Class       <- '[' #((!']' Range)* ']'^MissingClosingBracket)
Range       <- #(Char '-' !']' Char^MissingRightRange) / Char
Char        <- #(Escape / Unicode / .)
Escape      <- '\\\\' [nrt'"\\-\\[\\]\\\\]
Unicode     <- #('\\\\' 'u' Hex^chrH1 Hex^chrH2 Hex^chrH3 Hex^chrH4)
Hex         <- [0-9A-Fa-f]
Any         <- "."
Superscript <- [¹²³⁴⁵⁶⁷⁸⁹]
LEFTARROW   <- '<-'
SLASH       <- '/' !'/'   // negation prevents collision with comments

// Override the builtin spacing rule. Notice, it *must* be a lexical
// rule, otherwise it becomes an unbounded recursion
Spacing <- #(Comment / Space / EOL)*
Comment <- #('//' (!EOL .)* EOL?)
Space   <- [\\t\\u000C ]
EOL     <- [\\n\\r]
EOF     <- !.

// Recovery expressions

MissingClosingParen    <-
MissingClosingCurly    <-
MissingClosingBracket  <-
MissingClosingSQuote   <-
MissingClosingDQuote   <-
MissingLabel           <- (!Primary .)*

MissingImportName <- (!("from" / Literal) .)*
MissingImportFrom <- (!Literal .)*
MissingImportSrc  <- (!"\\n" .)*`;

const XML_UNSTABLE_GRAMMAR = `TopLevel  <- PROLOG? _* DTD? _* Element _*
PROLOG    <- '<?xml' (!'?>' .)* '?>'
DTD       <- '<' (!'>' .)* '>'
Element   <- '<' Name (_+ Attribute)* ('/>' / '>' Content '</' Name '>') _*
Name      <- [A-Za-z:] ('-' / [A-Za-z0-9:._])*
Attribute <- Name _* '=' _* String
String    <- '"' (!'"' .)* '"'
Content   <- (Element / CDataSec / CharData)*
CDataSec  <- '<![CDATA[' (!']]>' .)* ']]>' _*
COMMENT   <- '<!--' (!'-->' .)* '-->' _*
CharData  <- (!'<' .)+
_         <- [ \\t\\r\\n]`;

const XML_UNSTABLE_INPUT = `<?xml version="1.0" encoding="UTF-8"?>
<note>
  <to>Tove</to>
  <from>Jani</from>
  <heading>Reminder</heading>
  <body>Don't forget me this weekend!</body>
</note>`;

const PROTO_CIRC_GRAMMAR = `Program  <- Text* !.
Text     <- Circuit / Expr

Expr     <- Multi (( '+' / '-' ) Multi)*
Multi    <- Primary (( '*' / '/' ) Primary)*
Primary  <- Call / Attr / Port / Id / Num / '(' Expr ')'

Attr     <- Id '.'  (Call / Id / Attr)
Circuit  <- Id '=' Expr
Call     <- Id '(' Params ')'
Params   <- (Expr (',' Expr)*)?
Port     <- Id '[' Num ']'

Num      <- [1-9][0-9]* / '0'
Id       <- [a-zA-Z_][a-zA-Z0-9_]*`;

const PROTO_CIRC_GRAMMAR_INPUT_1 = `pin1 = std.pin
not1 = std.not
led1 = std.led

connect(pin1[0], not1[0])
connect(not1[1], led1[0])
`;

export default {
    demo: {
        grammar: TINY_GRAMMAR,
        input: TINY_INPUT,
    },
    json: {
        grammar: JSON_GRAMMAR,
        input: JSON_INPUT,
    },
    jsonStripped: {
        grammar: JSON_STRIPPED_GRAMMAR,
        input: JSON_INPUT,
    },
    csv: {
        grammar: CSV_GRAMMAR,
        input: CSV_INPUT,
    },
    xmlUnstable: {
        grammar: XML_UNSTABLE_GRAMMAR,
        input: XML_UNSTABLE_INPUT,
    },
    langlang: {
        grammar: LANGLANG_GRAMMAR,
        input: LANGLANG_GRAMMAR,
    },
    protoCirc: {
        grammar: PROTO_CIRC_GRAMMAR,
        input: PROTO_CIRC_GRAMMAR_INPUT_1,
    },
} as const;
