import LiveEditor from "../live/LiveEditor";
import {
    HomeMain,
    HomeIntro,
    HomeDescription,
    HomeFeatures,
} from "./HomePage.styles";

const DEMO_GRAMMAR = `JSON    <- Value
Value   <- Object / Array / String / Number / Bool / 'null'
Object  <- '{' (Member (',' Member)*)? '}'
Member  <- String ':' Value
Array   <- '[' (Value (',' Value)*)? ']'
Number  <- [0-9]+
Bool    <- 'true' / 'false'
String  <- '"' (!'"' .)* '"'`;

const DEMO_INPUT = `{"name": "langlang", "cool": true}`;

export default function HomePage() {
    return (
        <HomeMain>
            <HomeIntro>
                <HomeDescription>
                    Bring a grammar, get a parser. langlang turns Parsing Expression Grammars
                    into fast, self-contained parsers with error recovery,
                    automatic whitespace handling.
                </HomeDescription>
                <HomeFeatures>
                    <li>simple syntax, readable, unambiguous, no shift/reduce conflicts</li>
                    <li>Error labels and recovery rules produce useful trees from invalid input</li>
                    <li>Automatic whitespace handling keeps grammars concise</li>
                    <li>Generate parsers ahead-of-time or load grammars dynamically at runtime</li>
                    <li>Ahead-of-time parsers are output with no external dependencies</li>
                </HomeFeatures>
            </HomeIntro>
            <LiveEditor grammar={DEMO_GRAMMAR} input={DEMO_INPUT} height="480px" />
        </HomeMain>
    );
}
