import LiveEditor from "../live/LiveEditor";
import { jsonStrippedGrammar, jsonSmallInput } from "../examples/presets";
import {
    HomeMain,
    HomeIntro,
    HomeDescription,
    HomeFeatures,
} from "./HomePage.styles";

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
            <LiveEditor grammar={jsonStrippedGrammar} input={jsonSmallInput} height="480px" />
        </HomeMain>
    );
}
