
# Table of Contents

1.  [Introduction](#orga48aafa)
    1.  [Project Status](#org36b6f24)
    2.  [Currently supported output languages](#org5c951ba)
        1.  [Notes](#org1d674fb)
    3.  [Basic Usage](#orgf6fc674)
2.  [Input Language](#orgeb5706c)
    1.  [Productions and Expressions](#org1b14d36)
    2.  [Terminals](#orgc4baa34)
    3.  [Non-Terminals](#orgb708024)
    4.  [Expression Composition](#org0856e34)
        1.  [Ordered Choice](#org22ea646)
        2.  [Syntactic Predicates](#org4296efe)
        3.  [Repetitions](#orgad9e099)
        4.  [Lexification](#orgbaa2345)
        5.  [Error reporting with Labels](#orgebaef5d)
        6.  [Import system](#orgc1dd541)
3.  [Generator Options](#orge75e1ed)
    1.  [Go](#orgcd88847)
        1.  [Basic command line arguments](#org8f0b682)
        2.  [Additional options](#org63bd5cb)
4.  [Roadmap](#orgf4a66e8)
5.  [Changelog](#org18b62b5)
    1.  [go/v0.0.5](#org8c62d77)


<a id="orga48aafa"></a>

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


<a id="org36b6f24"></a>

## Project Status

-   We're not 1.0 yet, so the API is not stable, which means that data
    structure shapes might change, and/or behavihor might change,
    drastically, and without much notice.
-   Don't submit pull requests without opening an issue and discussing
    your idea.  We will take the slow approach aiming at great design
    first, then being stable, then being featureful.


<a id="org5c951ba"></a>

## Currently supported output languages

-   [X] Rust¹
-   [X] Go Lang²
-   [ ] Python
-   [ ] Java Script
-   [ ] Write your own code generator


<a id="org1d674fb"></a>

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


<a id="orgf6fc674"></a>

## Basic Usage

If you just want to test the waters, point the command line utility at
a grammar and pick a starting rule:

    cargo run --bin langlang run --grammar-file grammars/json.peg --start-rule JSON

That will drop you into an initeractive shell that allows you to try
out different input expressions.

Take a look at other examples at the directory `grammars` in the root
of the repository.  It contains a grammar library for commonly used
input formats.


<a id="orgeb5706c"></a>

# Input Language


<a id="org1b14d36"></a>

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


<a id="orgc4baa34"></a>

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


<a id="orgb708024"></a>

## Non-Terminals

The biggest addition of this type of grammar on top of regular
expressions is the ability to define and recursively call productions.
Here's a grammar snippet for parsing numbers:

    Signed   <- ('-' / '+') Signed / Decimal
    Decimal  <- ([1-9][0-9]*) / '0'

The topmost production `Signed` calls itself or the production
`Decimal`.  It allows parsing signed and unsigned numbers
recursively. (e.g.: `+-+--1` and so forth would be accepted).


<a id="org0856e34"></a>

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


<a id="org22ea646"></a>

### Ordered Choice

This operator tries expressions one at a time, from left to right, and
stops at the first one to succeed.  Or error if no alternatives work.
E.g.:

    SomeDigits <- '0' / '1' / '2' / '3' / '4'

Passing `6` to the above expression will generate an error.


<a id="org4296efe"></a>

### Syntactic Predicates

Predicates are the mechanism that allows unlimited look ahead, as they
do not consume any input.  e.g.:

    BracketString <- "[" (!"]" .)* "]"

In the above example, the **any** expression isn't evaluated if the
parser finds the closing square bracket.

The **and** predicate (`&`) is just syntactical sugar for `!!`.


<a id="orgad9e099"></a>

### Repetitions

-   **Zero Or More** never fails because, as it can match its expression at
    least zero times.

-   **One Or More** the syntax sugar for calling the expression once,
    followed by applying zero or more to the same expression. It can
    fail at the first time it matches the expression.

-   **Optional** will match an expression zero or one time.


<a id="orgbaa2345"></a>

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


<a id="orgebaef5d"></a>

### Error reporting with Labels


<a id="orgc1dd541"></a>

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


<a id="orge75e1ed"></a>

# Generator Options


<a id="orgcd88847"></a>

## Go

For using the [deprecated] go generator, you can run the following
command:

    go run ./go/cmd -grammar grammars.json


<a id="org8f0b682"></a>

### Basic command line arguments

-   `--grammar FILE`: is the only required parameter.  It takes the
    grammar FILE as input and if no other command line arguments are
    provided, the output is printed to `stdout`.

-   `--output PATH`: this option replaces `stdout` as the output with
    the PATH value provided to this command.

-   `--ast-only`: this will prevent the generator from outputing the
    final parser, and instead, it output the grammar's AST.

-   `--language LANG`: this will cause the generator to output a parser
    in the target language LANG.  As of this writing, the only
    supported value is `go`, but there are plans to extend support to
    both Python and JavaScript/TypeScript.


<a id="org63bd5cb"></a>

### Additional options

The Go code generator provides the following additional knobs to the
command line:

-   `--go-package`: allows customizing what goes in the `package`
    directive that starts each Go file.


<a id="orgf4a66e8"></a>

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


<a id="org18b62b5"></a>

# Changelog


<a id="org8c62d77"></a>

## go/v0.0.5

-   [BREAKING CHANGE: Remove runtime dependencies from output parser](https://github.com/clarete/langlang/commit/fb6fdc9cf56dae3dcdd48c29ebc0ffae9c14ae9b)
-   [BREAKING CHANGE: Overhaul naming of all the node types](https://github.com/clarete/langlang/commit/3d276aeb7e89c31f0bca6acba1174f6889f7e45c)
-   [BUG FIX: Labels must be serched as well for recovery rules](https://github.com/clarete/langlang/commit/71c702ac3265bf80e6b5a3dd696b307a018ecc71)

