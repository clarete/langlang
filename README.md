
# Table of Contents

1.  [Introduction](#org4c066ff)
    1.  [Project Status](#orgb0d4960)
    2.  [Currently supported output languages](#org292e3f8)
        1.  [Notes](#org1e315a6)
    3.  [Basic Usage](#org30fb095)
2.  [Input Language](#orgebbec8d)
    1.  [Productions and Expressions](#org525b058)
    2.  [Terminals](#org1a5f3c0)
    3.  [Non-Terminals](#orgaf74545)
    4.  [Expression Composition](#orgf8be86c)
        1.  [Ordered Choice](#org70b11a2)
        2.  [Syntactic Predicates](#org4e87702)
        3.  [Repetitions](#orgd4c6b6d)
        4.  [Lexification](#org36e0505)
        5.  [Error reporting with Labels](#org8a0fd2f)
        6.  [Import system](#org49d51fb)
3.  [Generator Options](#org9e2f248)
    1.  [Go](#org1b526e5)
        1.  [Basic command line arguments](#org82a6635)
        2.  [Additional options](#org49533b9)
4.  [Roadmap](#orgf34bc21)
5.  [Changelog](#orgc94d287)
    1.  [go/v0.0.8](#org0889635)
    2.  [go/v0.0.7](#org7a87f51)
    3.  [go/v0.0.6](#org6b37504)
    4.  [go/v0.0.5](#org82c9545)


<a id="org4c066ff"></a>

# Introduction

Bring your own grammar and get a feature rich parser generated for
different languages.  The are reasons why you might want to use this:

-   Concise input grammar format and intuitive algorithm: generates
    recursive top-down parsers based on Parsing Expression Grammars
-   Automatic handling of whitespaces, making grammars less cluttered
-   Error reporting with custom messages via failure `labels`
-   Partial support for declaring error recovery rules, which allow
    incremental parsing that returns an output tree even upon multiple
    parsing errors.


<a id="orgb0d4960"></a>

## Project Status

-   We're not 1.0 yet, so the API is not stable, which means that data
    structure shapes might change, and/or behavihor might change,
    drastically, and without much notice.
-   Don't submit pull requests without opening an issue and discussing
    your idea.  We will take the slow approach aiming at great design
    first, then being stable, then being featureful.


<a id="org292e3f8"></a>

## Currently supported output languages

-   [X] Rust¹
-   [X] Go Lang²
-   [ ] Python
-   [ ] Java Script
-   [ ] Write your own code generator


<a id="org1e315a6"></a>

### Notes

1.  Rust support is based on a virtual machine as its runtime, and not
    on a generated parser.  That is unlikely to change because our
    tooling will be built in Rust, so we need more flexibility on the
    "host implementation".  We may give an option to also generate a
    parser in Rust code that doesn't depend on any libraries, if it
    provides any value.  But such work isn't planned as of right now.

2.  We're in the middle of dropping the Go implementation in favor of
    a generating a Go parser from the code written in Rust.
    Prototyping a few features, like the import system and automatic
    space handling, was very useful, but once the refactoring of the
    Rust implementation is in a good place, the Rust version will be
    better as it will make it easier to generate parsers for other
    languages than Rust and Go.


<a id="org30fb095"></a>

## Basic Usage

If you just want to test the waters, point the command line utility at
a grammar and pick a starting rule:

    cargo run --bin langlang run --grammar-file grammars/json.peg --start-rule JSON

That will drop you into an initeractive shell that allows you to try
out different input expressions.

Take a look at other examples at the directory `grammars` in the root
of the repository.  It contains a grammar library for commonly used
input formats.


<a id="orgebbec8d"></a>

# Input Language


<a id="org525b058"></a>

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


<a id="org1a5f3c0"></a>

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


<a id="orgaf74545"></a>

## Non-Terminals

The biggest addition of this type of grammar on top of regular
expressions is the ability to define and recursively call productions.
Here's a grammar snippet for parsing numbers:

    Signed   <- ('-' / '+') Signed / Decimal
    Decimal  <- ([1-9][0-9]*) / '0'

The topmost production `Signed` calls itself or the production
`Decimal`.  It allows parsing signed and unsigned numbers
recursively. (e.g.: `+-+--1` and so forth would be accepted).


<a id="orgf8be86c"></a>

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


<a id="org70b11a2"></a>

### Ordered Choice

This operator tries expressions one at a time, from left to right, and
stops at the first one to succeed.  Or error if no alternatives work.
E.g.:

    SomeDigits <- '0' / '1' / '2' / '3' / '4'

Passing `6` to the above expression will generate an error.


<a id="org4e87702"></a>

### Syntactic Predicates

Predicates are the mechanism that allows unlimited look ahead, as they
do not consume any input.  e.g.:

    BracketString <- "[" (!"]" .)* "]"

In the above example, the **any** expression isn't evaluated if the
parser finds the closing square bracket.

The **and** predicate (`&`) is just syntactical sugar for `!!`.


<a id="orgd4c6b6d"></a>

### Repetitions

-   **Zero Or More** never fails because, as it can match its expression at
    least zero times.

-   **One Or More** the syntax sugar for calling the expression once,
    followed by applying zero or more to the same expression. It can
    fail at the first time it matches the expression.

-   **Optional** will match an expression zero or one time.


<a id="org36e0505"></a>

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


<a id="org8a0fd2f"></a>

### Error reporting with Labels


<a id="org49d51fb"></a>

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


<a id="org9e2f248"></a>

# Generator Options


<a id="org1b526e5"></a>

## Go

For using the go generator, you can run the following command:

    go run ./go/cmd/langlang -grammar grammars/json.peg


<a id="org82a6635"></a>

### Basic command line arguments

-   `--grammar FILE`: is the only required parameter.  It takes the
    grammar FILE as input and if no other command line arguments are
    provided, the output is printed to `stdout`.

-   `--ast-only`: this will prevent the generator from outputing the
    final parser, and instead, it output the grammar's AST.

-   `--asm-only`: this will prevent the generator from outputing the
    final parser, and instead, it output the translation of the grammar
    into Bytecode.

-   `--disable-whitespace-handling`: If this flag is present, the
    whitespace handling productions won't be inserted into the AST.

-   `--output-path PATH`: this option replaces `stdout` as the output with
    the PATH value provided to this command.

-   `--output-type TYPE`: decides what type of parser will be
    generated.  The available options are `code` for generating code in
    the target language, or `bytecode` for translating the input
    grammar into bytecode (notice that this option will also generate
    an evaluator.)

-   `--output-language LANG`: this will cause the generator to output a parser
    in the target language LANG.  As of this writing, the only
    supported value is `go`, but there are plans to extend support to
    both Python and JavaScript/TypeScript.


<a id="org49533b9"></a>

### Additional options

The Go code generator provides the following additional knobs to the
command line:

-   `--go-package`: allows customizing what goes in the `package`
    directive that starts each Go file.


<a id="orgf34bc21"></a>

# Roadmap

-   [ ] MID: [gen<sub>go</sub>] rewrite Go generator in Rust
-   [ ] MID: [genall] generator interface to be shared by all targets
-   [ ] SML: [gen<sub>go</sub>] memoize results to guarantee O(1) parsing time
-   [ ] SML: [gen<sub>go</sub>] allocate output nodes in an arena
-   [ ] MID: [gen<sub>py</sub>] Python Code Generator: Start from scratch
-   [ ] MID: [gen<sub>js</sub>] Java Script Code Generator
-   [ ] MID: [gen<sub>go</sub>] explore generating Go ASM code instead of text
-   [ ] MID: Display Call Graph for debugging purposes
-   [ ] BIG: Bootstrap off hand written parser, so grammar writters can
    take advantage of the features baked into the parser generator


<a id="orgc94d287"></a>

# Changelog


<a id="org0889635"></a>

## go/v0.0.8

-   [BUG FIX: Clear result cache when parser is reset](https://github.com/clarete/langlang/commit/5195eae565fea7c17ebad2d32f9b917908beec02)


<a id="org7a87f51"></a>

## go/v0.0.7

-   [BUG FIX: Capturing error messages for CHOICE](https://github.com/clarete/langlang/commit/e2553fdaf69ab96ecc1a4184f21a0d61e27b069a)


<a id="org6b37504"></a>

## go/v0.0.6

-   [PERF: Memoize production function results](https://github.com/clarete/langlang/commit/3b3e427ee91999aa30e56927b4b8994829f6105d)
-   [PERF: Remove fmt.Sprintf from core matching functions](https://github.com/clarete/langlang/commit/0fd67c472f60e5ce9b1e17c20bab7b443dbf62ad)


<a id="org82c9545"></a>

## go/v0.0.5

-   [BREAKING CHANGE: Remove runtime dependencies from output parser](https://github.com/clarete/langlang/commit/fb6fdc9cf56dae3dcdd48c29ebc0ffae9c14ae9b)
-   [BREAKING CHANGE: Overhaul naming of all the node types](https://github.com/clarete/langlang/commit/3d276aeb7e89c31f0bca6acba1174f6889f7e45c)
-   [BUG FIX: Labels must be serched as well for recovery rules](https://github.com/clarete/langlang/commit/71c702ac3265bf80e6b5a3dd696b307a018ecc71)

