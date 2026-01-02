
# Table of Contents

1.  [Introduction](#introduction)
    1.  [Project Status](#project-status)
    2.  [Supported targets](#supported-targets)
        1.  [Notes](#notes)
    3.  [Basic Usage](#basic-usage)
        1.  [Interactive](#interactive)
        2.  [Point it at a file](#point-it-at-a-file)
        3.  [Command line arguments](#command-line-arguments)
        4.  [Go specific options](#go-specific-options)
        5.  [Other options](#other-options)
2.  [Input Language](#input-language)
    1.  [Productions and Expressions](#productions-and-expressions)
    2.  [Terminals](#terminals)
    3.  [Non-Terminals](#non-terminals)
    4.  [Expression Composition](#expression-composition)
        1.  [Ordered Choice](#ordered-choice)
        2.  [Syntactic Predicates](#syntactic-predicates)
        3.  [Repetitions](#repetitions)
        4.  [Lexification](#lexification)
        5.  [Error reporting with Labels](#error-reporting-with-labels)
        6.  [Import system](#import-system)
3.  [Roadmap](#roadmap)
4.  [Changelog](#changelog)
    1.  [go/v0.0.12 (unreleased)](#gov0012-unreleased)
    2.  [go/v0.0.11](#gov0011)
    3.  [go/v0.0.10](#gov0010)
    4.  [go/v0.0.9](#gov009)
    5.  [go/v0.0.8](#gov008)
    6.  [go/v0.0.7](#gov007)
    7.  [go/v0.0.6](#gov006)
    8.  [go/v0.0.5](#gov005)


<a id="introduction"></a>

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


<a id="project-status"></a>

## Project Status

-   We're not 1.0 yet, so the API is not stable, which means that data
    structure shapes might change, and/or behavihor might change,
    drastically, and without much notice.
-   Don't submit pull requests without opening an issue and discussing
    your idea.  We will take the slow approach aiming at great design
    first, then being stable, then being featureful.


<a id="supported-targets"></a>

## Supported targets

-   [X] Go Lang¹
-   [ ] Python
-   [ ] Java Script
-   [ ] Rust²
-   [ ] Write your own code generator


<a id="notes"></a>

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


<a id="basic-usage"></a>

## Basic Usage


<a id="interactive"></a>

### Interactive

If you just want to test the waters, point the command line utility at
a grammar and pick a starting rule:

    cd go
    go run ./cmd/langlang -grammar ../grammars/json.peg

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


<a id="point-it-at-a-file"></a>

### Point it at a file

    go run ./cmd/langlang -grammar ../grammars/json.peg -input /tmp/foo.json

Take a look at other examples at the directory `grammars` in the root
of the repository.  It contains a grammar library for commonly used
input formats.


<a id="command-line-arguments"></a>

### Command line arguments

-   `-grammar FILE`: is the only required parameter.  It takes the
    grammar FILE as input and if no output path is provided, an
    interactive shell presented.

-   `-output-path PATH`: Path in which the generated parser will be
    saved.  (Not required for interactive mode.)

-   `-output-language LANG`: this will cause the generator to output a
    parser in the target language LANG.  As of this writing, the only
    supported values are `goeval` and `go`, but there are plans to
    extend support to both Python, JavaScript/TypeScript, Rust and
    other languages.  Notice this option is required if `-output-path`
    is set.  (Not required for interactive mode.)

-   `-disable-spaces`: If this flag is set the space handling
    productions won't be inserted into the AST.  This is useful for
    grammars that already have spacing rules placed throughout their
    definitions.

-   `-disable-captures`: If this flag is set the generated parser won't
    capture any values (functioning as matching only.)

-   `-disable-capture-spaces`: If this flag is set the generated parser
    won't capture space chars in the output tree.  This is useful if
    you only care for generating an AST of the parse tree, for example.

-   `-suppress-spaces`: Even when captures are enabled when generating
    a parser, one might want the parser to suppress capturing space
    chars anyway.  This flag does just that on runtime (in contrast to
    `-disable-space-captures`, that disables all captures in
    ****compile**** time, thus it is mostly useful for the interactive
    use.)

-   `-show-fails`: If this flag is set, the runtime will collect a set
    of characters that were attempted to match but failed just so the
    default error message can be more informative.  Since this might
    incur some performance costs while not being a critical feature, it
    can be safely disabled.


<a id="go-specific-options"></a>

### Go specific options

When generating Go code, the following additional knobs are provided
in the command line:

-   `-go-package`: allows customizing what goes in the `package`
    directive that starts each Go file.
-   `-go-parser`: overrides the default name (`Parser`) of the
    generated parser struct.  This is useful to put more than one
    parser in the same package.
-   `-go-remove-lib`: add just the parsing program (`Parser` struct) to
    the generated parser, not the evaluator.  This is also useful to
    put more than one parser in the same package.


<a id="other-options"></a>

### Other options

-   `-grammar-ast`: Shows the AST of the input grammar.

-   `-grammar-asm`: Shows the ASM code generated from the input
    grammar.

-   `-disable-builtins`: Do not add builtin rules into the grammar
    (mostly space handling at this point.)

-   `-disable-charsets`: Disable rewriting classes as `charsets`.
    Mostly useful for debugging.

-   `-disable-inline`: Disable inlining productions.  Mostly useful for
    debugging.

-   `-disable-inline-defs=false`: When inlining is enabled, the parser
    generated at the end won't include methods for the productions that
    have been inlined by default.  If you do want to generate the code
    for all the productions, even the inlined ones, disable this
    option.


<a id="input-language"></a>

# Input Language


<a id="productions-and-expressions"></a>

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


<a id="terminals"></a>

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


<a id="non-terminals"></a>

## Non-Terminals

The biggest addition of this type of grammar on top of regular
expressions is the ability to define and recursively call productions.
Here's a grammar snippet for parsing numbers:

    Signed   <- ('-' / '+') Signed / Decimal
    Decimal  <- ([1-9][0-9]*) / '0'

The topmost production `Signed` calls itself or the production
`Decimal`.  It allows parsing signed and unsigned numbers
recursively. (e.g.: `+-+--1` and so forth would be accepted).


<a id="expression-composition"></a>

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


<a id="ordered-choice"></a>

### Ordered Choice

This operator tries expressions one at a time, from left to right, and
stops at the first one to succeed.  Or error if no alternatives work.
E.g.:

    SomeDigits <- '0' / '1' / '2' / '3' / '4'

Passing `6` to the above expression will generate an error.


<a id="syntactic-predicates"></a>

### Syntactic Predicates

Predicates are the mechanism that allows unlimited look ahead, as they
do not consume any input.  e.g.:

    BracketString <- "[" (!"]" .)* "]"

In the above example, the **any** expression isn't evaluated if the
parser finds the closing square bracket.

The **and** predicate (`&`) is just syntactical sugar for `!!`.


<a id="repetitions"></a>

### Repetitions

-   **Zero Or More** never fails because, as it can match its expression at
    least zero times.

-   **One Or More** the syntax sugar for calling the expression once,
    followed by applying zero or more to the same expression. It can
    fail at the first time it matches the expression.

-   **Optional** will match an expression zero or one time.


<a id="lexification"></a>

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


<a id="error-reporting-with-labels"></a>

### Error reporting with Labels


<a id="import-system"></a>

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


<a id="roadmap"></a>

# Roadmap

-   Known optimization Head-Fail (compiler, vm)
-   Incremental Parsing (compiler, vm)
-   Left Recursion (compiler, vm)
-   Parse Data Structures (compiler, vm)
-   Semantic Actions (compiler, vm)
-   Display Call Graph for debugging purposes (compiler, tools)
-   Built-in Indent rule for Python-like grammars (compiler, vm)
-   New targets: C, JavaScript, WASM, Python, Java, Rust, Zig
-   Integration with serde of target language (like go and rust)


<a id="changelog"></a>

# Changelog


<a id="gov0012-unreleased"></a>

## go/v0.0.12 (unreleased)

-   [FEAT: Add 32-bit char/range opcodes for non-BMP Unicode](https://github.com/clarete/langlang/commit/6a455fe9d0771ae876cacf62aa5231907ea47deb)
-   [FEAT[api]: Add Tree.CursorU16 and unify position indexing](https://github.com/clarete/langlang/commit/c613467c46accd3ee2dd8b2a215caa0506ccc47d)
-   [FEAT[api]: Reintroduce API for retrieving line/column position](https://github.com/clarete/langlang/commit/57660e47906b3c00fc12540eb8e274d58a29adc5)
-   [FEAT[api]: Add `Tree.Copy` method and document memory ownership](https://github.com/clarete/langlang/commit/40fc6105e21a805d82ee3cdab376893f3b642913)


<a id="gov0011"></a>

## go/v0.0.11

-   [PERF: Use replace map with slice for expectedInfo](https://github.com/clarete/langlang/commit/e14c67f6312168595126d8363ce960a167d6b90f)
-   [PERF[BREAKING]: Pre-compile error labels to make parser creation cheaper](https://github.com/clarete/langlang/commit/0386254902c1d838bf69ac7e59b173755abf1810)
-   [PERF[BREAKING]: Change Capture Tree design to a Struct of Arrays](https://github.com/clarete/langlang/commit/e2dff9dd8fdab1c380be5c575da95c5b98c9e674)
-   [FEAT/PERF[BREAKING]: Remove runtime suppression of rule captures](https://github.com/clarete/langlang/commit/16351891a52cc2b7be3265e9e835ff08151d3adb)
-   [PERF: Replace map lookup with bitset for recovery expr check](https://github.com/clarete/langlang/commit/a660bae4f4f476b2513a56e8818b37f7399e637d)
-   [PERF: Reduce allocations by using an arena for frame captures](https://github.com/clarete/langlang/commit/6d54884a83c4bc1b34140737fc97c17d8687e15d)


<a id="gov0010"></a>

## go/v0.0.10

-   [BUG FIX: Fix space injection around repetitions](https://github.com/clarete/langlang/commit/8ae815837f6a432eceff8bf03fc59eb7b39f5b35)
-   [BUG FIX: Fix disabling capturing spaces](https://github.com/clarete/langlang/commit/e7cf6f1a25f924053d4d863dad53a58c5e2875c4)
-   [FEAT: Use pointer receivers for all Value.Accept() methods](https://github.com/clarete/langlang/commit/1de1e5286ee9ca8b7d8a14d2e0a1ed5424daf7de)
-   [FEAT: Format generated parser code](https://github.com/clarete/langlang/commit/78a72123bac3ca8161b26b2d77a573f0e7efe6b9)
-   [PERF[BREAKING]: A naive implementation of inlining](https://github.com/clarete/langlang/commit/e56339fdabe20f9a2a46ea785567e17b3cf3b4e2)
-   [PERF: Avoid bounds check within tight loop](https://github.com/clarete/langlang/commit/f382b55ff67e8f5b47b6c1576a6c2a8825f644ac)
-   [PERF: Specialize cap{,-partial,-back}-commit](https://github.com/clarete/langlang/commit/af2a9b40c060ae8173af3095d46da5895b280a11)
-   [PERF: hoist cursor and pc in hot path, clean up FFP update](https://github.com/clarete/langlang/commit/8b8a56bc8a82b956f258610b3adfbe3a1825c4d4)
-   [PERF: remove rune/int conversions using charsets](https://github.com/clarete/langlang/commit/253d59b1d6d2062a3a1ca13450da773a7d26fbf5)
-   [PERF: Drop Input interface and use concrete MemInput all over](https://github.com/clarete/langlang/commit/c20f169246c5efcbe9ab016f4514c8404655082b)
-   [PERF: Rework how we decode uint16 from our bytecode](https://github.com/clarete/langlang/commit/5f0f3794ecb22778e993322420c5122bd0e1b240) ([PR2](https://github.com/clarete/langlang/commit/5f0f3794ecb22778e993322420c5122bd0e1b240))
-   [PERF[BREAKING]: Keep only Range within String & use []byte vs string](https://github.com/clarete/langlang/commit/0d662030a8434b71d7defbe3b007e73563562864)
-   [PERF: Replace Span with Range and remove line/column from vm](https://github.com/clarete/langlang/commit/adfb7d12534efa864b574748065714ddbbd2c11f)
-   [PERF: Rework initializating vm when creating parser](https://github.com/clarete/langlang/commit/68d7425da659b898c33780e36ab4fb3d95943b46)


<a id="gov009"></a>

## go/v0.0.9

-   [PERF: known optimizations: charsets (set, span)](https://github.com/clarete/langlang/commit/d08d9ca46d610901beba3b7f7e63ca4f59cd54ff)
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


<a id="gov008"></a>

## go/v0.0.8

-   [BUG FIX: Clear result cache when parser is reset](https://github.com/clarete/langlang/commit/5195eae565fea7c17ebad2d32f9b917908beec02)


<a id="gov007"></a>

## go/v0.0.7

-   [BUG FIX: Capturing error messages for CHOICE](https://github.com/clarete/langlang/commit/e2553fdaf69ab96ecc1a4184f21a0d61e27b069a)


<a id="gov006"></a>

## go/v0.0.6

-   [PERF: Memoize production function results](https://github.com/clarete/langlang/commit/3b3e427ee91999aa30e56927b4b8994829f6105d)
-   [PERF: Remove fmt.Sprintf from core matching functions](https://github.com/clarete/langlang/commit/0fd67c472f60e5ce9b1e17c20bab7b443dbf62ad)


<a id="gov005"></a>

## go/v0.0.5

-   [BREAKING CHANGE: Remove runtime dependencies from output parser](https://github.com/clarete/langlang/commit/fb6fdc9cf56dae3dcdd48c29ebc0ffae9c14ae9b)
-   [BREAKING CHANGE: Overhaul naming of all the node types](https://github.com/clarete/langlang/commit/3d276aeb7e89c31f0bca6acba1174f6889f7e45c)
-   [BUG FIX: Labels must be serched as well for recovery rules](https://github.com/clarete/langlang/commit/71c702ac3265bf80e6b5a3dd696b307a018ecc71)

