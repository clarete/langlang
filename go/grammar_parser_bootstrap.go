package langlang

import (
	"strings"
)

var bytecodeForGrammarParserBootstrap = &Bytecode{
	code: []byte{
		11, 5, 0, 0, 0, 15, 0, 0, 11, 205, 4, 0, 6, 22, 0, 11, 55, 0, 0, 9, 15, 0, 11, 205, 4, 0, 6, 36, 0, 11, 181, 0, 0, 9, 29, 0, 11, 205, 4, 0, 6, 50, 0, 11, 71, 5, 0, 8, 53, 0, 14, 4, 0, 16, 12, 15, 2, 0, 11, 205, 4, 0, 15, 6, 0, 2, 64, 0, 2, 105, 0, 2, 109, 0, 2, 112, 0, 2, 111, 0, 2, 114, 0, 2, 116, 0, 16, 11, 205, 4, 0, 6, 101, 0, 11, 93, 2, 0, 8, 104, 0, 14, 7, 0, 11, 205, 4, 0, 6, 133, 0, 11, 205, 4, 0, 15, 6, 0, 2, 44, 0, 16, 11, 205, 4, 0, 11, 93, 2, 0, 9, 111, 0, 11, 205, 4, 0, 6, 159, 0, 15, 6, 0, 2, 102, 0, 2, 114, 0, 2, 111, 0, 2, 109, 0, 16, 8, 162, 0, 14, 9, 0, 11, 205, 4, 0, 6, 176, 0, 11, 165, 2, 0, 8, 179, 0, 14, 10, 0, 16, 12, 15, 3, 0, 11, 205, 4, 0, 11, 93, 2, 0, 11, 205, 4, 0, 11, 179, 4, 0, 11, 205, 4, 0, 11, 210, 0, 0, 16, 12, 15, 13, 0, 11, 205, 4, 0, 11, 249, 0, 0, 11, 205, 4, 0, 6, 247, 0, 11, 205, 4, 0, 11, 190, 4, 0, 11, 205, 4, 0, 11, 249, 0, 0, 9, 228, 0, 16, 12, 15, 14, 0, 11, 205, 4, 0, 6, 10, 1, 11, 12, 1, 0, 9, 3, 1, 16, 12, 15, 16, 0, 11, 205, 4, 0, 6, 58, 1, 6, 35, 1, 15, 6, 0, 2, 35, 0, 16, 8, 55, 1, 6, 48, 1, 15, 6, 0, 2, 38, 0, 16, 8, 55, 1, 15, 6, 0, 2, 33, 0, 16, 8, 58, 1, 11, 205, 4, 0, 11, 68, 1, 0, 16, 12, 15, 17, 0, 11, 205, 4, 0, 11, 116, 1, 0, 6, 114, 1, 15, 6, 0, 6, 94, 1, 2, 94, 0, 8, 97, 1, 2, 209, 33, 16, 6, 108, 1, 11, 93, 2, 0, 8, 111, 1, 14, 19, 0, 8, 114, 1, 16, 12, 15, 18, 0, 11, 205, 4, 0, 11, 182, 1, 0, 11, 205, 4, 0, 6, 180, 1, 6, 147, 1, 15, 6, 0, 2, 63, 0, 16, 8, 177, 1, 6, 160, 1, 15, 6, 0, 2, 42, 0, 16, 8, 177, 1, 6, 173, 1, 15, 6, 0, 2, 43, 0, 16, 8, 177, 1, 11, 99, 4, 0, 8, 180, 1, 16, 12, 15, 20, 0, 6, 207, 1, 11, 93, 2, 0, 11, 205, 4, 0, 7, 204, 1, 11, 179, 4, 0, 5, 8, 26, 2, 6, 248, 1, 15, 6, 0, 2, 40, 0, 16, 11, 205, 4, 0, 11, 210, 0, 0, 11, 205, 4, 0, 6, 242, 1, 15, 6, 0, 2, 41, 0, 16, 8, 245, 1, 14, 22, 0, 8, 26, 2, 6, 2, 2, 11, 28, 2, 0, 8, 26, 2, 6, 12, 2, 11, 165, 2, 0, 8, 26, 2, 6, 22, 2, 11, 0, 3, 0, 8, 26, 2, 11, 91, 4, 0, 16, 12, 15, 23, 0, 11, 205, 4, 0, 15, 6, 0, 2, 123, 0, 16, 11, 205, 4, 0, 6, 71, 2, 11, 205, 4, 0, 7, 60, 2, 2, 125, 0, 5, 11, 205, 4, 0, 11, 210, 0, 0, 9, 49, 2, 11, 205, 4, 0, 6, 88, 2, 15, 6, 0, 2, 125, 0, 16, 8, 91, 2, 14, 26, 0, 16, 12, 15, 8, 0, 6, 107, 2, 3, 97, 0, 122, 0, 8, 121, 2, 6, 118, 2, 3, 65, 0, 90, 0, 8, 121, 2, 2, 95, 0, 6, 163, 2, 6, 135, 2, 3, 97, 0, 122, 0, 8, 160, 2, 6, 146, 2, 3, 65, 0, 90, 0, 8, 160, 2, 6, 157, 2, 3, 48, 0, 57, 0, 8, 160, 2, 2, 95, 0, 9, 124, 2, 16, 12, 15, 11, 0, 6, 214, 2, 15, 6, 0, 2, 39, 0, 16, 6, 195, 2, 7, 188, 2, 2, 39, 0, 5, 11, 100, 3, 0, 9, 181, 2, 6, 208, 2, 15, 6, 0, 2, 39, 0, 16, 8, 211, 2, 14, 28, 0, 8, 254, 2, 15, 6, 0, 2, 34, 0, 16, 6, 238, 2, 7, 231, 2, 2, 34, 0, 5, 11, 100, 3, 0, 9, 224, 2, 6, 251, 2, 15, 6, 0, 2, 34, 0, 16, 8, 254, 2, 14, 29, 0, 16, 12, 15, 24, 0, 11, 205, 4, 0, 15, 6, 0, 2, 91, 0, 16, 6, 31, 3, 7, 24, 3, 2, 93, 0, 5, 11, 49, 3, 0, 9, 17, 3, 6, 44, 3, 15, 6, 0, 2, 93, 0, 16, 8, 47, 3, 14, 31, 0, 16, 12, 15, 30, 0, 6, 94, 3, 11, 205, 4, 0, 11, 100, 3, 0, 11, 205, 4, 0, 15, 6, 0, 2, 45, 0, 16, 11, 205, 4, 0, 6, 88, 3, 11, 100, 3, 0, 8, 91, 3, 14, 32, 0, 8, 98, 3, 11, 100, 3, 0, 16, 12, 15, 27, 0, 6, 113, 3, 11, 137, 3, 0, 8, 135, 3, 6, 123, 3, 11, 220, 3, 0, 8, 135, 3, 7, 130, 3, 2, 92, 0, 5, 15, 6, 0, 1, 16, 16, 12, 15, 33, 0, 2, 92, 0, 6, 152, 3, 2, 110, 0, 8, 218, 3, 6, 161, 3, 2, 114, 0, 8, 218, 3, 6, 170, 3, 2, 116, 0, 8, 218, 3, 6, 179, 3, 2, 39, 0, 8, 218, 3, 6, 188, 3, 2, 34, 0, 8, 218, 3, 6, 197, 3, 2, 45, 0, 8, 218, 3, 6, 206, 3, 2, 91, 0, 8, 218, 3, 6, 215, 3, 2, 93, 0, 8, 218, 3, 2, 92, 0, 16, 12, 15, 34, 0, 11, 205, 4, 0, 15, 6, 0, 2, 92, 0, 16, 11, 205, 4, 0, 15, 6, 0, 2, 117, 0, 16, 11, 205, 4, 0, 6, 3, 4, 11, 59, 4, 0, 8, 6, 4, 14, 35, 0, 11, 205, 4, 0, 6, 20, 4, 11, 59, 4, 0, 8, 23, 4, 14, 37, 0, 11, 205, 4, 0, 6, 37, 4, 11, 59, 4, 0, 8, 40, 4, 14, 38, 0, 11, 205, 4, 0, 6, 54, 4, 11, 59, 4, 0, 8, 57, 4, 14, 39, 0, 16, 12, 15, 36, 0, 6, 73, 4, 3, 48, 0, 57, 0, 8, 89, 4, 6, 84, 4, 3, 65, 0, 70, 0, 8, 89, 4, 3, 97, 0, 102, 0, 16, 12, 15, 25, 0, 2, 46, 0, 16, 12, 15, 21, 0, 6, 111, 4, 2, 185, 0, 8, 177, 4, 6, 120, 4, 2, 178, 0, 8, 177, 4, 6, 129, 4, 2, 179, 0, 8, 177, 4, 6, 138, 4, 2, 116, 32, 8, 177, 4, 6, 147, 4, 2, 117, 32, 8, 177, 4, 6, 156, 4, 2, 118, 32, 8, 177, 4, 6, 165, 4, 2, 119, 32, 8, 177, 4, 6, 174, 4, 2, 120, 32, 8, 177, 4, 2, 121, 32, 16, 12, 15, 12, 0, 2, 60, 0, 2, 45, 0, 16, 12, 15, 15, 0, 2, 47, 0, 7, 203, 4, 2, 47, 0, 5, 16, 12, 15, 1, 0, 6, 238, 4, 6, 221, 4, 11, 240, 4, 0, 8, 235, 4, 6, 231, 4, 11, 28, 5, 0, 8, 235, 4, 11, 54, 5, 0, 9, 211, 4, 16, 12, 15, 40, 0, 15, 6, 0, 2, 47, 0, 2, 47, 0, 16, 15, 6, 0, 6, 15, 5, 7, 11, 5, 11, 54, 5, 0, 5, 1, 9, 3, 5, 16, 6, 26, 5, 11, 54, 5, 0, 8, 26, 5, 16, 12, 15, 41, 0, 6, 40, 5, 2, 9, 0, 8, 52, 5, 6, 49, 5, 2, 12, 0, 8, 52, 5, 2, 32, 0, 16, 12, 15, 42, 0, 6, 66, 5, 2, 10, 0, 8, 69, 5, 2, 13, 0, 16, 12, 15, 5, 0, 7, 79, 5, 1, 5, 16, 12, 15, 22, 0, 16, 12, 15, 26, 0, 16, 12, 15, 31, 0, 16, 12, 15, 28, 0, 16, 12, 15, 29, 0, 16, 12, 15, 19, 0, 6, 124, 5, 7, 120, 5, 11, 182, 1, 0, 5, 1, 9, 112, 5, 16, 12, 15, 7, 0, 6, 162, 5, 7, 158, 5, 6, 153, 5, 2, 102, 0, 2, 114, 0, 2, 111, 0, 2, 109, 0, 8, 157, 5, 11, 165, 2, 0, 5, 1, 9, 132, 5, 16, 12, 15, 9, 0, 6, 182, 5, 7, 178, 5, 11, 165, 2, 0, 5, 1, 9, 170, 5, 16, 12, 15, 10, 0, 6, 201, 5, 7, 197, 5, 2, 10, 0, 5, 1, 9, 190, 5, 16, 12, 
	},
	strs: []string{
		"Grammar", "Spacing", "Import", "Definition", "eof", "EOF", "", "MissingImportName", "Identifier", "MissingImportFrom", "MissingImportSrc", "Literal", "LEFTARROW", "Expression", "Sequence", "SLASH", "Prefix", "Labeled", "Suffix", "MissingLabel", "Primary", "Superscript", "MissingClosingParen", "List", "Class", "Any", "MissingClosingCurly", "Char", "MissingClosingSQuote", "MissingClosingDQuote", "Range", "MissingClosingBracket", "MissingRightRange", "Escape", "Unicode", "chrH1", "Hex", "chrH2", "chrH3", "chrH4", "Comment", "Space", "EOL", 
	},
	rxps: map[int]int{
		7: 1406, 9: 1444, 10: 1464, 19: 1386, 22: 1361, 26: 1366, 28: 1376, 29: 1381, 31: 1371, 
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
func (p *GrammarParserBootstrap) ParseSequence() (Value, error) { return p.parseFn(249) }
func (p *GrammarParserBootstrap) ParsePrefix() (Value, error) { return p.parseFn(268) }
func (p *GrammarParserBootstrap) ParseLabeled() (Value, error) { return p.parseFn(324) }
func (p *GrammarParserBootstrap) ParseSuffix() (Value, error) { return p.parseFn(372) }
func (p *GrammarParserBootstrap) ParsePrimary() (Value, error) { return p.parseFn(438) }
func (p *GrammarParserBootstrap) ParseList() (Value, error) { return p.parseFn(540) }
func (p *GrammarParserBootstrap) ParseIdentifier() (Value, error) { return p.parseFn(605) }
func (p *GrammarParserBootstrap) ParseLiteral() (Value, error) { return p.parseFn(677) }
func (p *GrammarParserBootstrap) ParseClass() (Value, error) { return p.parseFn(768) }
func (p *GrammarParserBootstrap) ParseRange() (Value, error) { return p.parseFn(817) }
func (p *GrammarParserBootstrap) ParseChar() (Value, error) { return p.parseFn(868) }
func (p *GrammarParserBootstrap) ParseEscape() (Value, error) { return p.parseFn(905) }
func (p *GrammarParserBootstrap) ParseUnicode() (Value, error) { return p.parseFn(988) }
func (p *GrammarParserBootstrap) ParseHex() (Value, error) { return p.parseFn(1083) }
func (p *GrammarParserBootstrap) ParseAny() (Value, error) { return p.parseFn(1115) }
func (p *GrammarParserBootstrap) ParseSuperscript() (Value, error) { return p.parseFn(1123) }
func (p *GrammarParserBootstrap) ParseLEFTARROW() (Value, error) { return p.parseFn(1203) }
func (p *GrammarParserBootstrap) ParseSLASH() (Value, error) { return p.parseFn(1214) }
func (p *GrammarParserBootstrap) ParseSpacing() (Value, error) { return p.parseFn(1229) }
func (p *GrammarParserBootstrap) ParseComment() (Value, error) { return p.parseFn(1264) }
func (p *GrammarParserBootstrap) ParseSpace() (Value, error) { return p.parseFn(1308) }
func (p *GrammarParserBootstrap) ParseEOL() (Value, error) { return p.parseFn(1334) }
func (p *GrammarParserBootstrap) ParseEOF() (Value, error) { return p.parseFn(1351) }
func (p *GrammarParserBootstrap) ParseMissingClosingParen() (Value, error) { return p.parseFn(1361) }
func (p *GrammarParserBootstrap) ParseMissingClosingCurly() (Value, error) { return p.parseFn(1366) }
func (p *GrammarParserBootstrap) ParseMissingClosingBracket() (Value, error) { return p.parseFn(1371) }
func (p *GrammarParserBootstrap) ParseMissingClosingSQuote() (Value, error) { return p.parseFn(1376) }
func (p *GrammarParserBootstrap) ParseMissingClosingDQuote() (Value, error) { return p.parseFn(1381) }
func (p *GrammarParserBootstrap) ParseMissingLabel() (Value, error) { return p.parseFn(1386) }
func (p *GrammarParserBootstrap) ParseMissingImportName() (Value, error) { return p.parseFn(1406) }
func (p *GrammarParserBootstrap) ParseMissingImportFrom() (Value, error) { return p.parseFn(1444) }
func (p *GrammarParserBootstrap) ParseMissingImportSrc() (Value, error) { return p.parseFn(1464) }
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
