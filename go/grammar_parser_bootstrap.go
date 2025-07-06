package langlang

import (
	"strings"
)

var bytecodeForGrammarParserBootstrap = &Bytecode{
	code: []byte{
		11, 5, 0, 0, 0, 15, 0, 0, 11, 80, 4, 0, 6, 22, 0, 11, 55, 0, 0, 9, 15, 0, 11, 80, 4, 0, 6, 36, 0, 11, 181, 0, 0, 9, 29, 0, 11, 80, 4, 0, 6, 50, 0, 11, 199, 4, 0, 8, 53, 0, 14, 4, 0, 16, 12, 15, 2, 0, 11, 80, 4, 0, 15, 6, 0, 2, 64, 0, 2, 105, 0, 2, 109, 0, 2, 112, 0, 2, 111, 0, 2, 114, 0, 2, 116, 0, 16, 11, 80, 4, 0, 6, 101, 0, 11, 96, 2, 0, 8, 104, 0, 14, 7, 0, 11, 80, 4, 0, 6, 133, 0, 11, 80, 4, 0, 15, 6, 0, 2, 44, 0, 16, 11, 80, 4, 0, 11, 96, 2, 0, 9, 111, 0, 11, 80, 4, 0, 6, 159, 0, 15, 6, 0, 2, 102, 0, 2, 114, 0, 2, 111, 0, 2, 109, 0, 16, 8, 162, 0, 14, 9, 0, 11, 80, 4, 0, 6, 176, 0, 11, 168, 2, 0, 8, 179, 0, 14, 10, 0, 16, 12, 15, 3, 0, 11, 80, 4, 0, 11, 96, 2, 0, 11, 80, 4, 0, 11, 69, 4, 0, 11, 80, 4, 0, 11, 210, 0, 0, 16, 12, 15, 13, 0, 11, 80, 4, 0, 11, 252, 0, 0, 11, 80, 4, 0, 6, 250, 0, 11, 80, 4, 0, 15, 6, 0, 2, 47, 0, 16, 11, 80, 4, 0, 11, 252, 0, 0, 9, 228, 0, 16, 12, 15, 14, 0, 11, 80, 4, 0, 6, 13, 1, 11, 15, 1, 0, 9, 6, 1, 16, 12, 15, 15, 0, 11, 80, 4, 0, 6, 61, 1, 6, 38, 1, 15, 6, 0, 2, 35, 0, 16, 8, 58, 1, 6, 51, 1, 15, 6, 0, 2, 38, 0, 16, 8, 58, 1, 15, 6, 0, 2, 33, 0, 16, 8, 61, 1, 11, 80, 4, 0, 11, 71, 1, 0, 16, 12, 15, 16, 0, 11, 80, 4, 0, 11, 119, 1, 0, 6, 117, 1, 15, 6, 0, 6, 97, 1, 2, 94, 0, 8, 100, 1, 2, 209, 33, 16, 6, 111, 1, 11, 96, 2, 0, 8, 114, 1, 14, 18, 0, 8, 117, 1, 16, 12, 15, 17, 0, 11, 80, 4, 0, 11, 185, 1, 0, 11, 80, 4, 0, 6, 183, 1, 6, 150, 1, 15, 6, 0, 2, 63, 0, 16, 8, 180, 1, 6, 163, 1, 15, 6, 0, 2, 42, 0, 16, 8, 180, 1, 6, 176, 1, 15, 6, 0, 2, 43, 0, 16, 8, 180, 1, 11, 245, 3, 0, 8, 183, 1, 16, 12, 15, 19, 0, 6, 210, 1, 11, 96, 2, 0, 11, 80, 4, 0, 7, 207, 1, 11, 69, 4, 0, 5, 8, 29, 2, 6, 251, 1, 15, 6, 0, 2, 40, 0, 16, 11, 80, 4, 0, 11, 210, 0, 0, 11, 80, 4, 0, 6, 245, 1, 15, 6, 0, 2, 41, 0, 16, 8, 248, 1, 14, 21, 0, 8, 29, 2, 6, 5, 2, 11, 31, 2, 0, 8, 29, 2, 6, 15, 2, 11, 168, 2, 0, 8, 29, 2, 6, 25, 2, 11, 3, 3, 0, 8, 29, 2, 11, 237, 3, 0, 16, 12, 15, 22, 0, 11, 80, 4, 0, 15, 6, 0, 2, 123, 0, 16, 11, 80, 4, 0, 6, 74, 2, 11, 80, 4, 0, 7, 63, 2, 2, 125, 0, 5, 11, 80, 4, 0, 11, 210, 0, 0, 9, 52, 2, 11, 80, 4, 0, 6, 91, 2, 15, 6, 0, 2, 125, 0, 16, 8, 94, 2, 14, 25, 0, 16, 12, 15, 8, 0, 6, 110, 2, 3, 97, 0, 122, 0, 8, 124, 2, 6, 121, 2, 3, 65, 0, 90, 0, 8, 124, 2, 2, 95, 0, 6, 166, 2, 6, 138, 2, 3, 97, 0, 122, 0, 8, 163, 2, 6, 149, 2, 3, 65, 0, 90, 0, 8, 163, 2, 6, 160, 2, 3, 48, 0, 57, 0, 8, 163, 2, 2, 95, 0, 9, 127, 2, 16, 12, 15, 11, 0, 6, 217, 2, 15, 6, 0, 2, 39, 0, 16, 6, 198, 2, 7, 191, 2, 2, 39, 0, 5, 11, 91, 3, 0, 9, 184, 2, 6, 211, 2, 15, 6, 0, 2, 39, 0, 16, 8, 214, 2, 14, 27, 0, 8, 1, 3, 15, 6, 0, 2, 34, 0, 16, 6, 241, 2, 7, 234, 2, 2, 34, 0, 5, 11, 91, 3, 0, 9, 227, 2, 6, 254, 2, 15, 6, 0, 2, 34, 0, 16, 8, 1, 3, 14, 28, 0, 16, 12, 15, 23, 0, 11, 80, 4, 0, 15, 6, 0, 2, 91, 0, 16, 6, 34, 3, 7, 27, 3, 2, 93, 0, 5, 11, 52, 3, 0, 9, 20, 3, 6, 47, 3, 15, 6, 0, 2, 93, 0, 16, 8, 50, 3, 14, 30, 0, 16, 12, 15, 29, 0, 6, 85, 3, 11, 91, 3, 0, 15, 6, 0, 2, 45, 0, 16, 6, 79, 3, 11, 91, 3, 0, 8, 82, 3, 14, 31, 0, 8, 89, 3, 11, 91, 3, 0, 16, 12, 15, 26, 0, 6, 178, 3, 2, 92, 0, 6, 109, 3, 2, 110, 0, 8, 175, 3, 6, 118, 3, 2, 114, 0, 8, 175, 3, 6, 127, 3, 2, 116, 0, 8, 175, 3, 6, 136, 3, 2, 45, 0, 8, 175, 3, 6, 145, 3, 2, 39, 0, 8, 175, 3, 6, 154, 3, 2, 34, 0, 8, 175, 3, 6, 163, 3, 2, 91, 0, 8, 175, 3, 6, 172, 3, 2, 93, 0, 8, 175, 3, 2, 92, 0, 8, 235, 3, 6, 202, 3, 2, 92, 0, 3, 48, 0, 50, 0, 3, 48, 0, 55, 0, 3, 48, 0, 55, 0, 8, 235, 3, 6, 227, 3, 2, 92, 0, 3, 48, 0, 55, 0, 6, 224, 3, 3, 48, 0, 55, 0, 8, 224, 3, 8, 235, 3, 7, 234, 3, 2, 92, 0, 5, 1, 16, 12, 15, 24, 0, 2, 46, 0, 16, 12, 15, 20, 0, 6, 1, 4, 2, 185, 0, 8, 67, 4, 6, 10, 4, 2, 178, 0, 8, 67, 4, 6, 19, 4, 2, 179, 0, 8, 67, 4, 6, 28, 4, 2, 116, 32, 8, 67, 4, 6, 37, 4, 2, 117, 32, 8, 67, 4, 6, 46, 4, 2, 118, 32, 8, 67, 4, 6, 55, 4, 2, 119, 32, 8, 67, 4, 6, 64, 4, 2, 120, 32, 8, 67, 4, 2, 121, 32, 16, 12, 15, 12, 0, 2, 60, 0, 2, 45, 0, 16, 12, 15, 1, 0, 6, 113, 4, 6, 96, 4, 11, 115, 4, 0, 8, 110, 4, 6, 106, 4, 11, 153, 4, 0, 8, 110, 4, 11, 170, 4, 0, 9, 86, 4, 16, 12, 15, 32, 0, 15, 6, 0, 2, 47, 0, 2, 47, 0, 16, 15, 6, 0, 6, 146, 4, 7, 142, 4, 11, 170, 4, 0, 5, 1, 9, 134, 4, 16, 11, 170, 4, 0, 16, 12, 15, 33, 0, 6, 165, 4, 2, 32, 0, 8, 168, 4, 2, 9, 0, 16, 12, 15, 34, 0, 6, 182, 4, 2, 10, 0, 8, 197, 4, 6, 194, 4, 2, 13, 0, 2, 10, 0, 8, 197, 4, 2, 13, 0, 16, 12, 15, 5, 0, 7, 207, 4, 1, 5, 16, 12, 15, 21, 0, 16, 12, 15, 25, 0, 16, 12, 15, 30, 0, 16, 12, 15, 27, 0, 16, 12, 15, 28, 0, 16, 12, 15, 18, 0, 6, 252, 4, 7, 248, 4, 11, 185, 1, 0, 5, 1, 9, 240, 4, 16, 12, 15, 7, 0, 6, 34, 5, 7, 30, 5, 6, 25, 5, 2, 102, 0, 2, 114, 0, 2, 111, 0, 2, 109, 0, 8, 29, 5, 11, 168, 2, 0, 5, 1, 9, 4, 5, 16, 12, 15, 9, 0, 6, 54, 5, 7, 50, 5, 11, 168, 2, 0, 5, 1, 9, 42, 5, 16, 12, 15, 10, 0, 6, 73, 5, 7, 69, 5, 2, 10, 0, 5, 1, 9, 62, 5, 16, 12, 
	},
	strs: []string{
		"Grammar", "Spacing", "Import", "Definition", "eof", "EOF", "", "MissingImportName", "Identifier", "MissingImportFrom", "MissingImportSrc", "Literal", "LEFTARROW", "Expression", "Sequence", "Prefix", "Labeled", "Suffix", "MissingLabel", "Primary", "Superscript", "MissingClosingParen", "List", "Class", "Any", "MissingClosingCurly", "Char", "MissingClosingSQuote", "MissingClosingDQuote", "Range", "MissingClosingBracket", "MissingRightRange", "Comment", "Space", "EOL", 
	},
	rxps: map[int]int{
		7: 1278, 9: 1316, 10: 1336, 18: 1258, 21: 1233, 25: 1238, 27: 1248, 28: 1253, 30: 1243, 
	},
}
type GrammarParserBootstrap struct{
	input         string
	captureSpaces bool
	suppress      map[int]struct{}
	errLabels     map[string]string
}
func NewGrammarParserBootstrap() *GrammarParserBootstrap {
	s := bytecodeForGrammarParserBootstrap.findStrIDs([]string{"Spacing"})
	return &GrammarParserBootstrap{captureSpaces: true, suppress: s}
}
func (p *GrammarParserBootstrap) ParseGrammar() (Value, error) { return p.parseFn(5) }
func (p *GrammarParserBootstrap) ParseImport() (Value, error) { return p.parseFn(55) }
func (p *GrammarParserBootstrap) ParseDefinition() (Value, error) { return p.parseFn(181) }
func (p *GrammarParserBootstrap) ParseExpression() (Value, error) { return p.parseFn(210) }
func (p *GrammarParserBootstrap) ParseSequence() (Value, error) { return p.parseFn(252) }
func (p *GrammarParserBootstrap) ParsePrefix() (Value, error) { return p.parseFn(271) }
func (p *GrammarParserBootstrap) ParseLabeled() (Value, error) { return p.parseFn(327) }
func (p *GrammarParserBootstrap) ParseSuffix() (Value, error) { return p.parseFn(375) }
func (p *GrammarParserBootstrap) ParsePrimary() (Value, error) { return p.parseFn(441) }
func (p *GrammarParserBootstrap) ParseList() (Value, error) { return p.parseFn(543) }
func (p *GrammarParserBootstrap) ParseIdentifier() (Value, error) { return p.parseFn(608) }
func (p *GrammarParserBootstrap) ParseLiteral() (Value, error) { return p.parseFn(680) }
func (p *GrammarParserBootstrap) ParseClass() (Value, error) { return p.parseFn(771) }
func (p *GrammarParserBootstrap) ParseRange() (Value, error) { return p.parseFn(820) }
func (p *GrammarParserBootstrap) ParseChar() (Value, error) { return p.parseFn(859) }
func (p *GrammarParserBootstrap) ParseAny() (Value, error) { return p.parseFn(1005) }
func (p *GrammarParserBootstrap) ParseSuperscript() (Value, error) { return p.parseFn(1013) }
func (p *GrammarParserBootstrap) ParseLEFTARROW() (Value, error) { return p.parseFn(1093) }
func (p *GrammarParserBootstrap) ParseSpacing() (Value, error) { return p.parseFn(1104) }
func (p *GrammarParserBootstrap) ParseComment() (Value, error) { return p.parseFn(1139) }
func (p *GrammarParserBootstrap) ParseSpace() (Value, error) { return p.parseFn(1177) }
func (p *GrammarParserBootstrap) ParseEOL() (Value, error) { return p.parseFn(1194) }
func (p *GrammarParserBootstrap) ParseEOF() (Value, error) { return p.parseFn(1223) }
func (p *GrammarParserBootstrap) ParseMissingClosingParen() (Value, error) { return p.parseFn(1233) }
func (p *GrammarParserBootstrap) ParseMissingClosingCurly() (Value, error) { return p.parseFn(1238) }
func (p *GrammarParserBootstrap) ParseMissingClosingBracket() (Value, error) { return p.parseFn(1243) }
func (p *GrammarParserBootstrap) ParseMissingClosingSQuote() (Value, error) { return p.parseFn(1248) }
func (p *GrammarParserBootstrap) ParseMissingClosingDQuote() (Value, error) { return p.parseFn(1253) }
func (p *GrammarParserBootstrap) ParseMissingLabel() (Value, error) { return p.parseFn(1258) }
func (p *GrammarParserBootstrap) ParseMissingImportName() (Value, error) { return p.parseFn(1278) }
func (p *GrammarParserBootstrap) ParseMissingImportFrom() (Value, error) { return p.parseFn(1316) }
func (p *GrammarParserBootstrap) ParseMissingImportSrc() (Value, error) { return p.parseFn(1336) }
func (p *GrammarParserBootstrap) Parse() (Value, error) { return p.parseFn(5) }
func (p *GrammarParserBootstrap) SetInput(input string) { p.input = input }
func (p *GrammarParserBootstrap) SetLabelMessages(el map[string]string) { p.errLabels = el }
func (p *GrammarParserBootstrap) SetCaptureSpaces(v bool) { p.captureSpaces = v }
func (p *GrammarParserBootstrap) parseFn(addr uint16) (Value, error) {
	writeU16(bytecodeForGrammarParserBootstrap.code[1:], addr)
	suppress := map[int]struct{}{}
	if !p.captureSpaces {
		suppress = p.suppress
	}
	vm := newVirtualMachine(bytecodeForGrammarParserBootstrap, p.errLabels, suppress)
	val, _, err := vm.Match(strings.NewReader(p.input))
	return val, err
}
