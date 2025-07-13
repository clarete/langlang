package langlang

import (
	"strings"
)

var bytecodeForGrammarParserWirthBootstrap = &Bytecode{
	code: []byte{
		11, 5, 0, 0, 0, 15, 0, 0, 11, 209, 1, 0, 6, 22, 0, 11, 32, 0, 0, 9, 15, 0, 11, 209, 1, 0, 11, 76, 2, 0, 16, 12, 15, 2, 0, 11, 209, 1, 0, 11, 97, 1, 0, 11, 209, 1, 0, 15, 5, 0, 2, 61, 0, 16, 11, 209, 1, 0, 6, 68, 0, 11, 81, 0, 0, 8, 68, 0, 11, 209, 1, 0, 15, 5, 0, 2, 46, 0, 16, 16, 12, 15, 6, 0, 11, 209, 1, 0, 11, 123, 0, 0, 11, 209, 1, 0, 6, 121, 0, 11, 209, 1, 0, 15, 5, 0, 2, 124, 0, 16, 11, 209, 1, 0, 11, 123, 0, 0, 9, 99, 0, 16, 12, 15, 7, 0, 11, 209, 1, 0, 11, 146, 0, 0, 6, 144, 0, 11, 146, 0, 0, 9, 137, 0, 16, 12, 15, 8, 0, 6, 170, 0, 11, 97, 1, 0, 11, 209, 1, 0, 7, 167, 0, 2, 61, 0, 5, 8, 204, 0, 6, 180, 0, 11, 206, 0, 0, 8, 204, 0, 6, 190, 0, 11, 248, 0, 0, 8, 204, 0, 6, 200, 0, 11, 27, 1, 0, 8, 204, 0, 11, 62, 1, 0, 16, 12, 15, 9, 0, 11, 209, 1, 0, 11, 158, 1, 0, 11, 209, 1, 0, 6, 246, 0, 11, 209, 1, 0, 15, 5, 0, 2, 38, 32, 16, 11, 209, 1, 0, 11, 158, 1, 0, 8, 246, 0, 16, 12, 15, 10, 0, 11, 209, 1, 0, 15, 5, 0, 2, 40, 0, 16, 11, 209, 1, 0, 11, 81, 0, 0, 11, 209, 1, 0, 15, 5, 0, 2, 41, 0, 16, 16, 12, 15, 11, 0, 11, 209, 1, 0, 15, 5, 0, 2, 91, 0, 16, 11, 209, 1, 0, 11, 81, 0, 0, 11, 209, 1, 0, 15, 5, 0, 2, 93, 0, 16, 16, 12, 15, 12, 0, 11, 209, 1, 0, 15, 5, 0, 2, 123, 0, 16, 11, 209, 1, 0, 11, 81, 0, 0, 11, 209, 1, 0, 15, 5, 0, 2, 125, 0, 16, 16, 12, 15, 4, 0, 6, 111, 1, 3, 97, 0, 122, 0, 8, 125, 1, 6, 122, 1, 3, 65, 0, 90, 0, 8, 125, 1, 2, 95, 0, 6, 156, 1, 6, 139, 1, 3, 97, 0, 122, 0, 8, 153, 1, 6, 150, 1, 3, 65, 0, 90, 0, 8, 153, 1, 2, 95, 0, 9, 128, 1, 16, 12, 15, 13, 0, 6, 187, 1, 2, 34, 0, 6, 181, 1, 7, 177, 1, 2, 34, 0, 5, 1, 9, 170, 1, 2, 34, 0, 8, 207, 1, 2, 39, 0, 6, 204, 1, 7, 200, 1, 2, 39, 0, 5, 1, 9, 193, 1, 2, 39, 0, 16, 12, 15, 1, 0, 6, 232, 1, 6, 225, 1, 11, 12, 2, 0, 8, 229, 1, 11, 234, 1, 0, 9, 215, 1, 16, 12, 15, 15, 0, 2, 40, 0, 2, 42, 0, 6, 4, 2, 7, 0, 2, 2, 42, 0, 2, 41, 0, 5, 1, 9, 246, 1, 2, 42, 0, 2, 41, 0, 16, 12, 15, 14, 0, 6, 28, 2, 15, 5, 0, 2, 32, 0, 16, 8, 45, 2, 6, 41, 2, 15, 5, 0, 2, 9, 0, 16, 8, 45, 2, 11, 47, 2, 0, 16, 12, 15, 16, 0, 6, 62, 2, 2, 13, 0, 2, 10, 0, 8, 74, 2, 6, 71, 2, 2, 10, 0, 8, 74, 2, 2, 13, 0, 16, 12, 15, 3, 0, 7, 84, 2, 1, 5, 16, 12, 
	},
	strs: []string{
		"Syntax", "Spacing", "Production", "EOF", "Identifier", "", "Expression", "Term", "Factor", "Range", "Group", "Option", "Repetition", "Literal", "Space", "Comment", "EOL", 
	},
	rxps: map[int]int{
		
	},
	smap: map[string]int{
		"": 5, "Comment": 15, "EOF": 3, "EOL": 16, "Expression": 6, "Factor": 8, "Group": 10, "Identifier": 4, "Literal": 13, "Option": 11, "Production": 2, "Range": 9, "Repetition": 12, "Space": 14, "Spacing": 1, "Syntax": 0, "Term": 7, 
	},
}
type GrammarParserWirthBootstrap struct{
	input         string
	captureSpaces bool
	suppress      map[int]struct{}
	errLabels     map[string]string
}
func NewGrammarParserWirthBootstrap() *GrammarParserWirthBootstrap {
	suppress := map[int]struct{}{bytecodeForGrammarParserWirthBootstrap.smap["Spacing"]: struct{}{}}
	return &GrammarParserWirthBootstrap{captureSpaces: true, suppress: suppress}
}
func (p *GrammarParserWirthBootstrap) ParseSyntax() (Value, error) { return p.parseFn(5) }
func (p *GrammarParserWirthBootstrap) ParseProduction() (Value, error) { return p.parseFn(32) }
func (p *GrammarParserWirthBootstrap) ParseExpression() (Value, error) { return p.parseFn(81) }
func (p *GrammarParserWirthBootstrap) ParseTerm() (Value, error) { return p.parseFn(123) }
func (p *GrammarParserWirthBootstrap) ParseFactor() (Value, error) { return p.parseFn(146) }
func (p *GrammarParserWirthBootstrap) ParseRange() (Value, error) { return p.parseFn(206) }
func (p *GrammarParserWirthBootstrap) ParseGroup() (Value, error) { return p.parseFn(248) }
func (p *GrammarParserWirthBootstrap) ParseOption() (Value, error) { return p.parseFn(283) }
func (p *GrammarParserWirthBootstrap) ParseRepetition() (Value, error) { return p.parseFn(318) }
func (p *GrammarParserWirthBootstrap) ParseIdentifier() (Value, error) { return p.parseFn(353) }
func (p *GrammarParserWirthBootstrap) ParseLiteral() (Value, error) { return p.parseFn(414) }
func (p *GrammarParserWirthBootstrap) ParseSpacing() (Value, error) { return p.parseFn(465) }
func (p *GrammarParserWirthBootstrap) ParseComment() (Value, error) { return p.parseFn(490) }
func (p *GrammarParserWirthBootstrap) ParseSpace() (Value, error) { return p.parseFn(524) }
func (p *GrammarParserWirthBootstrap) ParseEOL() (Value, error) { return p.parseFn(559) }
func (p *GrammarParserWirthBootstrap) ParseEOF() (Value, error) { return p.parseFn(588) }
func (p *GrammarParserWirthBootstrap) Parse() (Value, error) { return p.parseFn(5) }
func (p *GrammarParserWirthBootstrap) SetInput(input string) { p.input = input }
func (p *GrammarParserWirthBootstrap) SetLabelMessages(el map[string]string) { p.errLabels = el }
func (p *GrammarParserWirthBootstrap) SetCaptureSpaces(v bool) { p.captureSpaces = v }
func (p *GrammarParserWirthBootstrap) parseFn(addr uint16) (Value, error) {
	writeU16(bytecodeForGrammarParserWirthBootstrap.code[1:], addr)
	var suppress map[int]struct{}
	if !p.captureSpaces {
		suppress = p.suppress
	}
	vm := newVirtualMachine(bytecodeForGrammarParserWirthBootstrap, p.errLabels, suppress)
	val, _, err := vm.Match(strings.NewReader(p.input))
	return val, err
}
