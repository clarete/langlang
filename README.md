
# Table of Contents

1.  [Introduction](#org06e6899)
    1.  [Project Status](#org5c23022)
    2.  [Supported targets](#orgdbf2362)
        1.  [Notes](#org3edd53f)
    3.  [Basic Usage](#orga3eef28)
        1.  [Command line arguments](#org778041e)
        2.  [Additional options](#org3998267)
2.  [Input Language](#org1058a33)
    1.  [Productions and Expressions](#org3cf121a)
    2.  [Terminals](#org0fb5f13)
    3.  [Non-Terminals](#org00a74f8)
    4.  [Expression Composition](#orgdeaec00)
        1.  [Ordered Choice](#orga1c2b48)
        2.  [Syntactic Predicates](#org32e20f5)
        3.  [Repetitions](#org27da150)
        4.  [Lexification](#org8700ee9)
        5.  [Error reporting with Labels](#org213bfd3)
        6.  [Import system](#orgcb2ae5e)
3.  [Roadmap](#org98a9561)
    1.  [0.0.9 (planned work):](#org191ba7f)
    2.  [0.0.10 and onwards:](#org26877fd)
4.  [Changelog](#orgc5060ed)
    1.  [go/v0.0.9 (unreleased)](#org1c49da1)
    2.  [go/v0.0.8](#org75b9750)
    3.  [go/v0.0.7](#org3bda141)
    4.  [go/v0.0.6](#org6155978)
    5.  [go/v0.0.5](#org8aa7232)


<a id="org06e6899"></a>

# Introduction

Bring your own grammar and get a feature rich parser generated for
different languages.  Some are reasons why you might want to use this:

-   Concise input grammar format and intuitive algorithm: generates
    recursive top-down parsers based on Parsing Expression Grammars
-   Automatic handling of whitespaces, making grammars less cluttered
-   Error reporting with custom messages via failure `labels`
-   Good error recovery support through recovery rules, which allow the
    parser to recover from known failures and produce an output tree
    even upon multiple parsing errors.


<a id="org5c23022"></a>

## Project Status

-   We're not 1.0 yet, so the API is not stable, which means that data
    structure shapes might change, and/or behavihor might change,
    drastically, and without much notice.
-   Don't submit pull requests without opening an issue and discussing
    your idea.  We will take the slow approach aiming at great design
    first, then being stable, then being featureful.


<a id="orgdbf2362"></a>

## Supported targets

-   [X] Go Lang¹
-   [ ] Python
-   [ ] Java Script
-   [ ] Rust²
-   [ ] Write your own code generator


<a id="org3edd53f"></a>

### Notes

1.  Go is the host and also a target language.  Although we're not 1.0
    yet, this implementation has served some real world use cases.
    The previous direction was to drop the Go implementation and
    invest on the Rust one, and this may eventually actually happen,
    but for now, the focus has changed back to using Go as the host
    language.

2.  Rust support will be re-introduced by making Rust a target
    language only, instead of having it to be both the host and one of
    the target languages.


<a id="orga3eef28"></a>

## Basic Usage

If you just want to test the waters, point the command line utility at
a grammar and pick a starting rule:

    cd go
    go run ./cmd/langlang/ -grammar ../grammars/json.peg

That will drop you into an initeractive shell that allows you to try
out different input expressions. e.g.:

    > [42]
    JSON (1..2:1)
    └── Value (1..5)
        └── Array (1..5)
            └── Sequence<3> (1..5)
                ├── "[" (1..2)
                ├── Value (2..4)
                │   └── Number (2..4)
                │       └── Int (2..4)
                │           └── "42" (2..4)
                └── "]" (4..5)

Take a look at other examples at the directory `grammars` in the root
of the repository.  It contains a grammar library for commonly used
input formats.


<a id="org778041e"></a>

### Command line arguments

-   `-grammar FILE`: is the only required parameter.  It takes the
    grammar FILE as input and if no output path is provided, an
    interactive shell presented.

-   `-grammar-ast`: Shows the AST of the input grammar.

-   `-grammar-asm`: Shows the ASM code generated from the input
    grammar.

-   `--disable-whitespace-handling`: If this flag is present, the
    whitespace handling productions won't be inserted into the AST.

-   `--output-path PATH`: this option replaces `stdout` as the output with
    the PATH value provided to this command.

-   `--output-language LANG`: this will cause the generator to output a
    parser in the target language LANG.  As of this writing, the only
    supported value is `go`, but there are plans to extend support to
    both Python, JavaScript/TypeScript, Rust and other languages.


<a id="org3998267"></a>

### Additional options

When generating Go code, the following additional knobs are provided
in the command line:

-   `--go-package`: allows customizing what goes in the `package`
    directive that starts each Go file.


<a id="org1058a33"></a>

# Input Language


<a id="org3cf121a"></a>

## Productions and Expressions

The input grammar is as simple as it can get.  It builds off of the
original PEG format, and other features are added conservatively.
Take the following input as an example:

    Production <- Expression

At the left side of the arrow there is an identifier and on the right
side, there is an expression.  These two together are called either
productions or (parsing) rules.  Let's go over how to compose them.
If you've ever seen or used regular expressions, you've got a head
start.


<a id="org0fb5f13"></a>

## Terminals

-   **Any**: matches any character, and only errors if it reaches
    the end of the input.  e.g.: `.`

-   **Literal**: anything around quotes (single and double quotes are the
    same).  e.g.: `'x'`

-   **Class and Range**: classes may contain either ranges or single
    characters.  e.g.: `[0-9]`, `[a-zA-Z]`, `[a-f0-9_]`.  This last
    example contains two ranges (`a-f` and `0-9`) and one single char
    (`_`).  It means **match either one of these**. e.g.: `[a-cA-C]` is
    translated to `'a' / 'b' / 'c' / 'A' / 'B' / 'C'`.


<a id="org00a74f8"></a>

## Non-Terminals

The biggest addition of this type of grammar on top of regular
expressions is the ability to define and recursively call productions.
Here's a grammar snippet for parsing numbers:

    Signed   <- ('-' / '+') Signed / Decimal
    Decimal  <- ([1-9][0-9]*) / '0'

The topmost production `Signed` calls itself or the production
`Decimal`.  It allows parsing signed and unsigned numbers
recursively. (e.g.: `+-+--1` and so forth would be accepted).


<a id="orgdeaec00"></a>

## Expression Composition

The following operators can be used on both Terminals and
Non-Terminals, on top of parenthesized expressions:

<table border="2" cellspacing="0" cellpadding="6" rules="groups" frame="hsides">


<colgroup>
<col  class="org-left" />

<col  class="org-left" />

<col  class="org-left" />
</colgroup>
<thead>
<tr>
<th scope="col" class="org-left">operator</th>
<th scope="col" class="org-left">example</th>
<th scope="col" class="org-left">comment</th>
</tr>
</thead>

<tbody>
<tr>
<td class="org-left"><b>ordered choice</b></td>
<td class="org-left"><code>e1 / e2</code></td>
<td class="org-left">&#xa0;</td>
</tr>


<tr>
<td class="org-left"><b>not predicate</b></td>
<td class="org-left"><code>!e</code></td>
<td class="org-left">&#xa0;</td>
</tr>


<tr>
<td class="org-left"><b>and predicate</b></td>
<td class="org-left"><code>&amp;e</code></td>
<td class="org-left">sugar for <code>!!e</code></td>
</tr>


<tr>
<td class="org-left"><b>zero or more</b></td>
<td class="org-left"><code>e*</code></td>
<td class="org-left">&#xa0;</td>
</tr>


<tr>
<td class="org-left"><b>one or more</b></td>
<td class="org-left"><code>e+</code></td>
<td class="org-left">sugar for <code>ee*</code></td>
</tr>


<tr>
<td class="org-left"><b>optional</b></td>
<td class="org-left"><code>e?</code></td>
<td class="org-left">sugar for <code>&amp;ee / !e</code></td>
</tr>


<tr>
<td class="org-left"><b>lexification</b></td>
<td class="org-left"><code>#e</code></td>
<td class="org-left">&#xa0;</td>
</tr>


<tr>
<td class="org-left"><b>label</b></td>
<td class="org-left"><code>e^label</code></td>
<td class="org-left">sugar for <code>e/throw(label)</code></td>
</tr>
</tbody>
</table>


<a id="orga1c2b48"></a>

### Ordered Choice

This operator tries expressions one at a time, from left to right, and
stops at the first one to succeed.  Or error if no alternatives work.
E.g.:

    SomeDigits <- '0' / '1' / '2' / '3' / '4'

Passing `6` to the above expression will generate an error.


<a id="org32e20f5"></a>

### Syntactic Predicates

Predicates are the mechanism that allows unlimited look ahead, as they
do not consume any input.  e.g.:

    BracketString <- "[" (!"]" .)* "]"

In the above example, the **any** expression isn't evaluated if the
parser finds the closing square bracket.

The **and** predicate (`&`) is just syntactical sugar for `!!`.


<a id="org27da150"></a>

### Repetitions

-   **Zero Or More** never fails because, as it can match its expression at
    least zero times.

-   **One Or More** the syntax sugar for calling the expression once,
    followed by applying zero or more to the same expression. It can
    fail at the first time it matches the expression.

-   **Optional** will match an expression zero or one time.


<a id="org8700ee9"></a>

### Lexification

By default, the generated parsers emit code to consume whitespaces
automatically before each item within a sequence of a production
that's considered not syntactic.  Productions are considered syntactic
if all their expressions are syntactic.  Expressions are considered
syntactic if their output tree is composed only of terminal matches.
If there's any path to a non-terminal match, the entire expression,
and production are considered non syntactic.  e.g.:

    NotSyntactic <- Syntactic "!"
    Syntactic    <- "a" "b" "c"

In the above example, there is no automatic space consumption injected
before the items of the sequence expression `"a" "b" "c"` as all of
them are terminals.  And the `NotSyntactic` production contains non
terminal calls, which makes it non-syntactic.  Therefore, automatic
space handling will be enabled for `NotSyntactic` and disabled for
`Syntactic`

For **disabling** automatic space handling of an expression, prefix it
with the lexification operator `#`. e.g.:

    Ordinal <- Decimal #('st' / 'nd' / 'rd' / 'th')^ord
    Decimal <- ([1-9][0-9]*) / '0'

In the above expression, `Decimal` is considered syntactic, which
disables automatic space handling.  `Ordinal` is not syntactic because
it calls out to another production with a non-terminal.  So, automatic
space handling is enabled for that production.  However, between the
non-terminal and the choice with terminals, space handling is
disabled.  This is what is expected

<table border="2" cellspacing="0" cellpadding="6" rules="groups" frame="hsides">


<colgroup>
<col  class="org-left" />

<col  class="org-left" />
</colgroup>
<thead>
<tr>
<th scope="col" class="org-left">Input</th>
<th scope="col" class="org-left">Result</th>
</tr>
</thead>

<tbody>
<tr>
<td class="org-left">" 3rd"</td>
<td class="org-left">succeeds</td>
</tr>


<tr>
<td class="org-left">"50th"</td>
<td class="org-left">succeeds</td>
</tr>


<tr>
<td class="org-left">"2 0th"</td>
<td class="org-left">fails</td>
</tr>


<tr>
<td class="org-left">"2 th"</td>
<td class="org-left">fails</td>
</tr>
</tbody>
</table>

The first input succeeds because space consumption is automatically
added to the left of the call to the non terminal `Decimal`, as
`Ordinal` is not syntactic.  But because the expression that follows
the non terminal is marked with the lexification operator, automatic
space handling won't be injected between the call to the non terminal
and the ordered choice with the syntactic suffixed `st`, `nd`, `rd`,
and `th`.

Here is maybe the most classic example of where lexification is
needed: Non-Syntactic String Literals.  Which uses eager look ahead
and spaces are significant.  e.g.:

    SyntacticStringLiteral     <- '"' (!'"' .) '"'
    NonSyntacticStringLiteral  <- DQ #((!DQ .)  DQ)

Without using the lexification operator on the rule
`NonSyntacticStringLiteral`, it would eat up the spaces after the
first quote, which can be undesired for string fields.

The rule `SyntacticStringLiteral` doesn't need the lexification
operator because all of its sub-expressions are terminals, therefore
the rule is syntactic and space consumption won't be generated by
default anyway.

There are definitely more use-cases of the lexification operator out
there, these are just the common ones.


<a id="org213bfd3"></a>

### Error reporting with Labels


<a id="orgcb2ae5e"></a>

### Import system

Productions of one grammar can be imported from another one.  That
allows reusing rules and delivering more consolidate grammar files and
more powerful parser generated at the end.

    // file player.peg
    @import AddrSpec from "./rfc5322.peg"
    
    Player <- "Name:" Name "," "Score:" Number "," "Email:" AddrSpec
    Name   <- [a-zA-Z ]+
    Number <- [0-9]+
    // ... elided for simplicity

    // file rfc5322.peg
    // https://datatracker.ietf.org/doc/html/rfc5322#section-3.4.1
    
    // ... elided for simplicity
    AddrSpec  <- LocalPart "@" Domain
    LocalPart <- DotAtom / QuotedString / ObsLocalPart
    Domain    <- DotAtom / DomainLiteral / ObsDomain
    // ... elided for simplicity

The above example illustrates that a rather complete email parser can
be used in other grammars using imports.  Behind the scenes, the
`AddrSpec` rule and all its dependencies have been merged into the
`player.peg` grammar.


<a id="org98a9561"></a>

# Roadmap


<a id="org191ba7f"></a>

## 0.0.9 (planned work):

-   SML: [compvm] known optimizations: set, span (charsets branch)
-   SML: [compvm] known optimizations: head-fail
-   SML: [compvm] known optimizations: inlining, tco
-   SML: [compvm] more profiling


<a id="org26877fd"></a>

## 0.0.10 and onwards:

-   SML: [compvm] allocate output nodes in an arena
-   MID: [genall] generator interface to be shared by all targets
-   SML: [gen-go] memoize results to guarantee O(1) parsing time
-   MID: [gen-py] Python Code Generator
-   MID: [gen-js] Java Script Code Generator
-   MID: [gen-rs] Rust Code Generator
-   MID: [devexp] Display Call Graph for debugging purposes


<a id="orgc5060ed"></a>

# Changelog


<a id="org1c49da1"></a>

## go/v0.0.9 (unreleased)

-   [FEAT/PERF: New Compiler and Virtual Machine based design](https://github.com/clarete/langlang/commit/e1276b6071ec41b747fdb5d0c1d38a6dc58e4798)
-   [FEAT: New Codegen based on the Compiler and VM](https://github.com/clarete/langlang/commit/ab3f63af92a052f0d5b7f4547c9f7e38f0d30171)
-   [PERF: VisitSequenceNode: shorten path with no or single item](https://github.com/clarete/langlang/commit/111206f683534608545830890033daa9d20cbe68)
-   [BUG FIX: escape dash so we can parse dashes within classes](https://github.com/clarete/langlang/commit/c918152380151bcbfcf0550bd73b404081c9fcd6)
-   [BUG FIX: 'file not found' errors swallowed by the ImportResolver](https://github.com/clarete/langlang/commit/0071f39de6f77eced59968cf2165fd8e1f4c5e52)
-   [FEAT: Bootstrap Parser off of Grammar Definition](https://github.com/clarete/langlang/commit/5bb30992bbedde7043dd2189a9a273b0f7e19687)
-   [BREAKING CHANGE: Revamp string representation of the AstNode API](https://github.com/clarete/langlang/commit/b6fd2ba806333b11dc8fb93fd5b66cebc62aeea4)
-   [BREAKING CHANGE: Revamp command line arguments](https://github.com/clarete/langlang/commit/afd1b9eedbc9fc9ad1cd57654418ab7f78199cb1)
-   [BREAKING CHANGE: New Error Reporting](https://github.com/clarete/langlang/commit/e4b716459bb9b39f0ced95ca99e2088f60892f84)
-   [BREAKING CHANGE: Move cmd to a directory with a better name](https://github.com/clarete/langlang/commit/b360504659703df19121965865e788bfe858e7f3)


<a id="org75b9750"></a>

## go/v0.0.8

-   [BUG FIX: Clear result cache when parser is reset](https://github.com/clarete/langlang/commit/5195eae565fea7c17ebad2d32f9b917908beec02)


<a id="org3bda141"></a>

## go/v0.0.7

-   [BUG FIX: Capturing error messages for CHOICE](https://github.com/clarete/langlang/commit/e2553fdaf69ab96ecc1a4184f21a0d61e27b069a)


<a id="org6155978"></a>

## go/v0.0.6

-   [PERF: Memoize production function results](https://github.com/clarete/langlang/commit/3b3e427ee91999aa30e56927b4b8994829f6105d)
-   [PERF: Remove fmt.Sprintf from core matching functions](https://github.com/clarete/langlang/commit/0fd67c472f60e5ce9b1e17c20bab7b443dbf62ad)


<a id="org8aa7232"></a>

## go/v0.0.5

-   [BREAKING CHANGE: Remove runtime dependencies from output parser](https://github.com/clarete/langlang/commit/fb6fdc9cf56dae3dcdd48c29ebc0ffae9c14ae9b)
-   [BREAKING CHANGE: Overhaul naming of all the node types](https://github.com/clarete/langlang/commit/3d276aeb7e89c31f0bca6acba1174f6889f7e45c)
-   [BUG FIX: Labels must be serched as well for recovery rules](https://github.com/clarete/langlang/commit/71c702ac3265bf80e6b5a3dd696b307a018ecc71)

