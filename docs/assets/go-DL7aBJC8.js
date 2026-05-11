const n=`// Go grammar based on https://go.dev/ref/spec (go1.26)
//
// Notes:
//
// - This grammar is written to work with --disable-spaces.
//
// - Tokens that trigger semicolons (identifiers, literals, ++, --, ), ], })
//   consume trailing Spacing (horizontal whitespace only).
//
// - Tokens that never trigger semicolons (operators, (, [, {, :, ,)
//   consume trailing Skip (whitespace including newlines/comments).
//
// - Semicolon insertion is *approximated* by accepting optional ';'
//   in most lists.

SourceFile       <- Skip PackageClause SemiOpt Skip (ImportDecl SemiOpt Skip)* TopLevelDeclList Skip EOF
TopLevelDeclList <- (TopLevelDecl (ListSep / &EOF))*
SemiOpt          <- SEMI?
ListSep          <- SEMI Skip / NL+
Skip             <- (HSpace / GeneralComment / EOL / LineComment)*

// Packages and imports

PackageClause    <- PACKAGE PackageName
PackageName      <- Identifier

ImportDecl       <- IMPORT (ImportSpec / LPAREN ImportSpecList? RPAREN)
ImportSpecList   <- (ImportSpec (ListSep / &RPAREN))*
ImportSpec       <- (DOT / Identifier)? ImportPath
ImportPath       <- StringLit

// Top-level declarations

TopLevelDecl     <- Declaration / FunctionDecl / MethodDecl
Declaration      <- ConstDecl / TypeDecl / VarDecl

ConstDecl        <- CONST (ConstSpec / LPAREN ConstSpecList? RPAREN)
ConstSpecList    <- (ConstSpec (ListSep / &RPAREN))*
ConstSpec        <- IdentifierList ((Type? EQU ExpressionList) / Type)?

VarDecl          <- VAR (VarSpec / LPAREN VarSpecList? RPAREN)
VarSpecList      <- (VarSpec (ListSep / &RPAREN))*
VarSpec          <- IdentifierList (Type (EQU ExpressionList)? / EQU ExpressionList)?

TypeDecl         <- TYPE (TypeSpec / LPAREN TypeSpecList? RPAREN)
TypeSpecList     <- (TypeSpec (ListSep / &RPAREN))*
TypeSpec         <- AliasDecl / TypeDef
AliasDecl        <- Identifier TypeParameters? EQU Type
TypeDef          <- Identifier TypeParameters? Type

TypeParameters   <- LBRACK TypeParamList RBRACK
TypeParamList    <- TypeParamDecl (COMMA TypeParamDecl)* COMMA?
TypeParamDecl    <- Identifier TypeConstraint?
TypeConstraint   <- TypeElem

FunctionDecl     <- FUNC Identifier TypeParameters? Signature FunctionBody?
MethodDecl       <- FUNC Receiver MethodName Signature FunctionBody?
MethodName       <- Identifier
Receiver         <- Parameters
FunctionBody     <- Block

// Types

Type <- TypeName TypeArgs?
      / TypeLit
      / LPAREN Type RPAREN

TypeName <- Identifier (DOT Identifier)?

TypeLit <- ArrayType
         / StructType
         / PointerType
         / FunctionType
         / InterfaceType
         / SliceType
         / MapType
         / ChannelType

ArrayType   <- LBRACK ArrayLength RBRACK ElementType
ArrayLength <- Expression
ElementType <- Type
PointerType <- STAR Type
SliceType   <- LBRACK RBRACK ElementType
MapType     <- MAP LBRACK Type RBRACK ElementType

ChannelType <- CHAN ElementType
             / CHAN SEND ElementType
             / SEND CHAN ElementType

StructType <- STRUCT LBRACE FieldDeclList? RBRACE
FieldDeclList <- (FieldDecl (ListSep / &RBRACE))*
FieldDecl <- (IdentifierList Type / EmbeddedField) Tag?
EmbeddedField <- STAR? TypeName TypeArgs?
Tag <- StringLit

FunctionType <- FUNC Signature
Signature <- Parameters Result?
Result <- Parameters / Type

Parameters <- LPAREN ParameterList? RPAREN
ParameterList <- ParameterDecl (COMMA ParameterDecl)* COMMA?
ParameterDecl <- IdentifierList ELLIPSIS? Type
             / ELLIPSIS? Type

InterfaceType <- INTERFACE LBRACE InterfaceElemList? RBRACE
InterfaceElemList <- (InterfaceElem (ListSep / &RBRACE))*
InterfaceElem <- MethodElem / TypeElem
MethodElem <- MethodName Signature
TypeElem <- TypeTerm (OR TypeTerm)*
TypeTerm <- Type / UnderlyingType
UnderlyingType <- TILDE Type

TypeArgs <- LBRACK TypeList RBRACK
TypeList <- Type (COMMA Type)* COMMA?

// Blocks and statements

Block <- LBRACE StatementList? RBRACE
StatementList <- (Statement (ListSep / &RBRACE / &CASE / &DEFAULT / &EOF))*

Statement <- Declaration
          / LabeledStmt
          / SimpleStmt
          / GotoStmt
          / GoStmt
          / ReturnStmt
          / BreakStmt
          / ContinueStmt
          / FallthroughStmt
          / Block
          / IfStmt
          / SwitchStmt
          / SelectStmt
          / ForStmt
          / DeferStmt
          / EmptyStmt

EmptyStmt <- SEMI
LabeledStmt <- Label !COLONEQU COLON Statement?
Label <- Identifier

SimpleStmt <- ExpressionList Spacing? SimpleStmtSuffix?
SimpleStmtSuffix <- AssignOp Spacing? ExpressionList
                  / COLONEQU Spacing? ExpressionList
                  / SEND Spacing? Expression
                  / INC / DEC
AssignOp <- EQU / PLUSEQU / MINUSEQU / STAREQU / DIVEQU / MODEQU
         / ANDEQU / OREQU / XOREQU / SHLEQU / SHREQU / ANDNOTEQU

GoStmt <- GO Expression
DeferStmt <- DEFER Expression
ReturnStmt <- RETURN ExpressionList?
BreakStmt <- BREAK Label?
ContinueStmt <- CONTINUE Label?
GotoStmt <- GOTO Label
FallthroughStmt <- FALLTHROUGH

IfStmt <- IF (SimpleStmt SEMI Skip)? ControlExpr Block (ELSE (IfStmt / Block))?

SwitchStmt <- ExprSwitchStmt / TypeSwitchStmt

ExprSwitchStmt <- SWITCH (SimpleStmt SEMI Skip)? ControlExpr? LBRACE ExprCaseClause* RBRACE
ExprCaseClause <- ExprSwitchCase COLON StatementList?
ExprSwitchCase <- CASE ExpressionList / DEFAULT

TypeSwitchStmt <- SWITCH (SimpleStmt SEMI Skip)? TypeSwitchGuard LBRACE TypeCaseClause* RBRACE
TypeSwitchGuard <- (Identifier COLONEQU)? PrimaryExpr DOT LPAREN TYPE RPAREN
TypeCaseClause <- TypeSwitchCase COLON StatementList?
TypeSwitchCase <- CASE TypeList / DEFAULT

SelectStmt <- SELECT LBRACE CommClause* RBRACE
CommClause <- CommCase COLON StatementList?
CommCase <- CASE SimpleStmt / DEFAULT

ForStmt <- FOR (RangeClause / ForClause / Condition)? Block
Condition <- ControlExpr
ForClause <- InitStmt? SEMI Skip Condition? SEMI Skip ForPostStmt?
InitStmt <- SimpleStmt
ForPostStmt <- ControlExprList Spacing? ForPostStmtSuffix?
ForPostStmtSuffix <- AssignOp Spacing? ControlExprList
                   / COLONEQU Spacing? ControlExprList
                   / SEND Spacing? ControlExpr
                   / INC / DEC
ControlExprList <- ControlExpr (COMMA ControlExpr)*
RangeClause <- (ExpressionList EQU / IdentifierList COLONEQU)? RANGE RangeExpr
RangeExpr <- ControlExpr

// Expressions

ExpressionList <- Expression (COMMA Expression)*
IdentifierList <- Identifier (COMMA Identifier)*

Expression <- LogicalOrExpr
LogicalOrExpr <- LogicalAndExpr (OROR LogicalAndExpr)*
LogicalAndExpr <- RelExpr (ANDAND RelExpr)*
RelExpr <- AddExpr (RelOp AddExpr)*
RelOp <- EQUEQU / NOTEQU / LT / LE / GT / GE
AddExpr <- MulExpr (AddOp MulExpr)*
AddOp <- PLUS / MINUS / OR / HAT
MulExpr <- UnaryExpr (MulOp UnaryExpr)*
MulOp <- STAR / DIV / MOD / SHL / SHR / AND / ANDNOT
UnaryExpr <- PrimaryExpr / UnaryOp UnaryExpr / RECEIVE UnaryExpr
UnaryOp <- PLUS / MINUS / BANG / HAT / STAR / AND

// ControlExpr is used in if/switch/for conditions to avoid parsing
// \`T { ... }\` as a composite literal when a statement block should follow.
ControlExpr <- ControlLogicalOrExpr
ControlLogicalOrExpr <- ControlLogicalAndExpr (OROR ControlLogicalAndExpr)*
ControlLogicalAndExpr <- ControlRelExpr (ANDAND ControlRelExpr)*
ControlRelExpr <- ControlAddExpr (RelOp ControlAddExpr)*
ControlAddExpr <- ControlMulExpr (AddOp ControlMulExpr)*
ControlMulExpr <- ControlUnaryExpr (MulOp ControlUnaryExpr)*
ControlUnaryExpr <- ControlPrimaryExpr / UnaryOp ControlUnaryExpr / RECEIVE ControlUnaryExpr
ControlPrimaryExpr <- ControlOperand (Selector / Index / Slice / GenericArgs / TypeAssertion / Arguments)*
ControlOperand <- BasicLit
               / FunctionLit
               / ControlCompositeLit
               / OperandName
               / Conversion
               / LPAREN Expression RPAREN
               / MethodExpr
ControlCompositeLit <- ControlLiteralType LiteralValue
ControlLiteralType <- ArrayType / LBRACK ELLIPSIS RBRACK ElementType / SliceType / MapType

PrimaryExpr <- Operand (Selector / Index / Slice / GenericArgs / TypeAssertion / Arguments)*
GenericArgs <- LBRACK TypeList RBRACK
Selector <- DOT Identifier
Index <- LBRACK Expression RBRACK
Slice <- LBRACK (Expression? COLON Expression COLON Expression / Expression? COLON Expression?) RBRACK
TypeAssertion <- DOT LPAREN Type RPAREN
Arguments <- LPAREN ArgumentList? RPAREN
ArgumentList <- (ExpressionList / (Type (COMMA ExpressionList)?)) ELLIPSIS? COMMA?

Operand <- Literal
        / OperandName
        / Conversion
        / LPAREN Expression RPAREN
        / MethodExpr
OperandName <- Identifier (DOT Identifier)?
MethodExpr <- ReceiverType DOT MethodName
ReceiverType <- Type
Conversion <- Type LPAREN Expression COMMA? RPAREN

Literal <- BasicLit / CompositeLit / FunctionLit
BasicLit <- ImaginaryLit / FloatLit / IntLit / RuneLit / StringLit
FunctionLit <- FUNC Signature FunctionBody

CompositeLit <- LiteralType LiteralValue
LiteralType <- StructType
            / ArrayType
            / LBRACK ELLIPSIS RBRACK ElementType
            / SliceType
            / MapType
            / TypeName TypeArgs?
LiteralValue <- LBRACE ElementList? COMMA? RBRACE
ElementList <- KeyedElement (COMMA KeyedElement)*
KeyedElement <- (Key COLON)? Element
Key <- LiteralValue / Expression
FieldName <- Identifier
Element <- Expression / LiteralValue

// Lexical rules

Identifier <- !Keyword Letter IdentCont* Spacing
Letter <- [a-zA-Z_] / [\\u{80}-\\u{FFFF}]
IdentCont <- Letter / Digit
Digit <- [0-9]

Keyword <- ('break'
          / 'case'
          / 'chan'
          / 'const'
          / 'continue'
          / 'default'
          / 'defer'
          / 'else'
          / 'fallthrough'
          / 'func'
          / 'for'
          / 'goto'
          / 'go'
          / 'if'
          / 'import'
          / 'interface'
          / 'map'
          / 'package'
          / 'range'
          / 'return'
          / 'select'
          / 'struct'
          / 'switch'
          / 'type'
          / 'var') !IdentCont

IntLit <- (BinaryLit / HexLit / OctalLit / DecimalLit) Spacing
DecimalLit <- '0' / [1-9] ([_]* Digit)*
BinaryLit <- ('0b' / '0B') [_]? [01] ([_]* [01])*
OctalLit <- ('0o' / '0O') [_]? [0-7] ([_]* [0-7])*
         / '0' ([_]* [0-7])+
HexLit <- ('0x' / '0X') [_]? HexDigits
HexDigits <- HexDigit ([_]* HexDigit)*
HexDigit <- [0-9a-fA-F]

FloatLit <- (DecFloatLit / HexFloatLit) Spacing
DecFloatLit <- DecimalDigits '.' DecimalDigits? Exponent?
             / '.' DecimalDigits Exponent?
             / DecimalDigits Exponent
DecimalDigits <- Digit ([_]* Digit)*
Exponent <- [eE] [+\\-]? DecimalDigits
HexFloatLit <- HexMantissa [pP] [+\\-]? DecimalDigits
HexMantissa <- ('0x' / '0X') HexDigits? '.' HexDigits / HexLit '.'?

ImaginaryLit <- (DecFloatLit / HexFloatLit / BinaryLit / HexLit / OctalLit / DecimalLit) 'i' Spacing

RuneLit <- ['] (EscapeSequence / ![\\\\'\\n\\r] .) ['] Spacing
StringLit <- RawStringLit / InterpretedStringLit
RawStringLit <- '\`' (!'\`' .)* '\`' Spacing
InterpretedStringLit <- '"' (EscapeSequence / ![\\\\"\\n\\r] .)* '"' Spacing

EscapeSequence <- '\\\\' ( [abfnrtv\\\\'"]
                       / OctalByteValue
                       / HexByteValue
                       / LittleUValue
                       / BigUValue
                       )
OctalByteValue <- [0-7][0-7][0-7]
HexByteValue <- 'x' HexDigit HexDigit
LittleUValue <- 'u' HexDigit HexDigit HexDigit HexDigit
BigUValue <- 'U' HexDigit HexDigit HexDigit HexDigit HexDigit HexDigit HexDigit HexDigit

// Tokens and spacing

PACKAGE <- 'package' !IdentCont Skip
IMPORT <- 'import' !IdentCont Skip
CONST <- 'const' !IdentCont Skip
TYPE <- 'type' !IdentCont Skip
VAR <- 'var' !IdentCont Skip
FUNC <- 'func' !IdentCont Skip
STRUCT <- 'struct' !IdentCont Skip
INTERFACE <- 'interface' !IdentCont Skip
MAP <- 'map' !IdentCont Skip
CHAN <- 'chan' !IdentCont Skip
RETURN <- 'return' !IdentCont Spacing
IF <- 'if' !IdentCont Skip
ELSE <- 'else' !IdentCont Skip
SWITCH <- 'switch' !IdentCont Skip
CASE <- 'case' !IdentCont Skip
DEFAULT <- 'default' !IdentCont Skip
SELECT <- 'select' !IdentCont Skip
FOR <- 'for' !IdentCont Skip
RANGE <- 'range' !IdentCont Skip
GO <- 'go' !IdentCont Skip
DEFER <- 'defer' !IdentCont Skip
BREAK <- 'break' !IdentCont Spacing
CONTINUE <- 'continue' !IdentCont Spacing
GOTO <- 'goto' !IdentCont Skip
FALLTHROUGH <- 'fallthrough' !IdentCont Spacing

SEMI   <- ';' Spacing
COMMA  <- ',' Skip
DOT    <- '.' Skip
COLON  <- ':' Skip
LPAREN <- '(' Skip
RPAREN <- ')' Spacing
LBRACK <- '[' Skip
RBRACK <- ']' Spacing
LBRACE <- '{' Skip
RBRACE <- '}' Spacing

ELLIPSIS <- '...' Skip
INC <- '++' Spacing
DEC <- '--' Spacing
COLONEQU <- ':=' Skip
SEND <- '<-' Skip
RECEIVE <- '<-' Skip

EQU <- '=' Skip
EQUEQU <- '==' Skip
NOTEQU <- '!=' Skip
LT <- '<' ![<=-] Skip
LE <- '<=' Skip
GT <- '>' ![>=] Skip
GE <- '>=' Skip

PLUS <- '+' ![=+] Skip
MINUS <- '-' ![=-] Skip
STAR <- '*' Skip
DIV <- '/' Skip
MOD <- '%' Skip
HAT <- '^' Skip
BANG <- '!' Skip
AND <- '&' ![&^=] Skip
OR <- '|' Skip
ANDNOT <- '&^' !'=' Skip
SHL <- '<<' !'=' Skip
SHR <- '>>' !'=' Skip
ANDAND <- '&&' Skip
OROR <- '||' Skip
TILDE <- '~' Skip

PLUSEQU <- '+=' Skip
MINUSEQU <- '-=' Skip
STAREQU <- '*=' Skip
DIVEQU <- '/=' Skip
MODEQU <- '%=' Skip
ANDEQU <- '&=' Skip
OREQU <- '|=' Skip
XOREQU <- '^=' Skip
SHLEQU <- '<<=' Skip
SHREQU <- '>>=' Skip
ANDNOTEQU <- '&^=' Skip

HSpace <- [ \\t\\u{C}]
EOL <- '\\r\\n' / '\\n' / '\\r'
Spacing <- (HSpace / GeneralComment)*
NL <- (EOL / (LineComment EOL?) / GeneralComment)+ HSpace*
LineComment <- '//' (![\\r\\n] .)*
GeneralComment <- '/*' (!'*/' .)* '*/'
`;export{n as default};
