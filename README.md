
# Table of Contents

1.  [Introduction](#org6c3d217)
    1.  [Project Status](#org6c5d5ef)
    2.  [Currently supported output languages](#orgf688c54)
        1.  [Notes](#org8cb0b98)
    3.  [Basic Usage](#org9d8ed89)
2.  [Input Language](#orgc8eb565)
    1.  [Productions and Expressions](#org31bbc90)
    2.  [Terminals](#org9ba338b)
    3.  [Non-Terminals](#orgb5b47dc)
    4.  [Expression Composition](#orgf59985e)
        1.  [Ordered Choice](#org06520a7)
        2.  [Predicates (Not/And)](#org41c997f)
        3.  [Repetition ({Zero,One} Or More)](#org71bd02e)
        4.  [Lexification](#orgd670ab2)
        5.  [Error reporting with Labels](#org1fcee3d)
        6.  [Import system](#orgf182d74)
3.  [Generator Options](#orgcd645aa)
    1.  [Go](#orgfd49fb8)
4.  [Roadmap](#orgd15f717)


<a id="org6c3d217"></a>

# Introduction

Bring your own grammar and get a feature rich parser generated for
different languages.  The are reasons why you might want to use this:

-   Concise input grammar format and intuitive algorithm: generates
    recursive top-down parsers based on Parsing Expression Grammars
-   Automatic handling of white spaces, making grammars less cluttered
-   Error reporting with custom messages via failure `labels`
-   Partial support for declaring error recovery rules, which allow
    incremental parsing that returns an output tree even upon multiple
    parsing errors.


<a id="org6c5d5ef"></a>

## Project Status

-   We're not 1.0 yet, so the API is not stable, which means that data
    structure shapes might change, and/or behavihor might change,
    drastically, and without much notice.
-   Don't submit pull requests without opening an issue and discussing
    your idea.  We will take the slow approach aiming at great design
    first, then being stable, then being featureful.


<a id="orgf688c54"></a>

## Currently supported output languages

-   [X] Rust¹
-   [X] Go Lang²
-   [ ] Python
-   [ ] Java Script
-   [ ] Write your own code generator


<a id="org8cb0b98"></a>

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


<a id="org9d8ed89"></a>

## Basic Usage

If you just want to test the waters, point the command line utility at
a grammar and pick a starting rule:

    cargo run --bin langlang run --grammar-file grammars/json.peg --start-rule JSON

That will drop you into an initeractive shell that allows you to try
out different input expressions.

Take a look at other examples at the directory `grammars` in the root
of the repository.  It contains a grammar library for commonly used
input formats.


<a id="orgc8eb565"></a>

# Input Language


<a id="org31bbc90"></a>

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


<a id="org9ba338b"></a>

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


<a id="orgb5b47dc"></a>

## Non-Terminals

The biggest addition of this type of grammar on top of regular
expressions is the ability to define and recursively call productions.
Here's a grammar snippet for parsing numbers:

    Signed   <- ('-' / '+') Signed / Decimal
    Decimal  <- ([1-9][0-9]*) / '0'

The topmost production `Signed` calls itself or the production
`Decimal`.  It allows parsing signed and unsigned numbers
recursively. (e.g.: `+-+--1` and so forth would be accepted).


<a id="orgf59985e"></a>

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


<a id="org06520a7"></a>

### Ordered Choice

This operator tries expressions one at a time, from left to right, and
stops at the first one to succeed.  Or error if no alternatives work.
E.g.:

    SomeDigits <- '0' / '1' / '2' / '3' / '4'

Passing `6` to the above expression will generate an error.


<a id="org41c997f"></a>

### Predicates (Not/And)

Predicates are the mechanism that allows unlimited look ahead, as they
do not consume any input.  e.g.:

    BracketString <- "[" (!"]" .)* "]"

In the above example, the **any** expression isn't evaluated if the
parser finds the closing square bracket.

The **and** predicate (`&`) is just syntactical sugar for `!!`.


<a id="org71bd02e"></a>

### Repetition ({Zero,One} Or More)

-   **Zero Or More**: it never fails, as it can match its expression at
    least zero times.

-   **One Or More** is the syntax sugar for calling the expression once,
    followed by applying zero or more to the same expression.  It can
    fail at the first time it matches the expression.

-   **Optional** it will match an expression zero or one time.


<a id="orgd670ab2"></a>

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
`Ordinal` is not syntactic.  But, because the expression that follows
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


<a id="org1fcee3d"></a>

### Error reporting with Labels


<a id="orgf182d74"></a>

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


<a id="orgcd645aa"></a>

# Generator Options


<a id="orgfd49fb8"></a>

## Go

The Go code generator provides the following additional knobs to the
command line:

-   `--go-package`: allows customizing what goes in the `package`
    directive that starts each Go file.

-   `--go-prefix`: allows customizing structs generated prefixing what
    is passed to this option.  This is especially useful if there are
    two grammars to be parsed in the same package.  At least one will
    need a prefix, so the generic `Parser` name doesn't collide. e.g.:
    `-go-prefix Tiny` would generate a `TinyParser` struct, a
    `NewTinyParser` constructor, etc.


<a id="orgd15f717"></a>

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
