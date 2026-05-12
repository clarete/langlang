import jsonStrippedGrammar from "./json/json.stripped.peg?raw";
import jsonSmallInput from "./json/small.json?raw";
import csvGrammar from "./csv/csv.peg?raw";

const sexprGrammar = `// S-expressions
Expr   <- Atom / List
List   <- '(' Expr* ')'
Atom   <- Symbol / Number / String

Symbol <- [a-zA-Z-+*/=<>!?][a-zA-Z0-9-+*/=<>!?]*
Number <- '-'? [0-9]+ ('.' [0-9]+)?
String <- '"' (!'"' .)* '"'`

const sexprInput = `(define (fact n)
        (if (= n 0)
            1
          (* n (fact (- n 1)))))`;


export { jsonStrippedGrammar, jsonSmallInput, csvGrammar, sexprGrammar, sexprInput };
