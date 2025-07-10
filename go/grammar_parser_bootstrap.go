package langlang

import (
	"strings"
)

var bytecodeForGrammarParserBootstrap = &Bytecode{
	code: []byte{
		11, 5, 0, 0, 0, 15, 0, 0, 11, 193, 4, 0, 6, 22, 0, 11, 55, 0, 0, 9, 15, 0, 11, 193, 4, 0, 6, 36, 0, 11, 181, 0, 0, 9, 29, 0, 11, 193, 4, 0, 6, 50, 0, 11, 59, 5, 0, 8, 53, 0, 14, 4, 0, 16, 12, 15, 2, 0, 11, 193, 4, 0, 15, 6, 0, 2, 64, 0, 2, 105, 0, 2, 109, 0, 2, 112, 0, 2, 111, 0, 2, 114, 0, 2, 116, 0, 16, 11, 193, 4, 0, 6, 101, 0, 11, 93, 2, 0, 8, 104, 0, 14, 7, 0, 11, 193, 4, 0, 6, 133, 0, 11, 193, 4, 0, 15, 6, 0, 2, 44, 0, 16, 11, 193, 4, 0, 11, 93, 2, 0, 9, 111, 0, 11, 193, 4, 0, 6, 159, 0, 15, 6, 0, 2, 102, 0, 2, 114, 0, 2, 111, 0, 2, 109, 0, 16, 8, 162, 0, 14, 9, 0, 11, 193, 4, 0, 6, 176, 0, 11, 165, 2, 0, 8, 179, 0, 14, 10, 0, 16, 12, 15, 3, 0, 11, 193, 4, 0, 11, 93, 2, 0, 11, 193, 4, 0, 11, 167, 4, 0, 11, 193, 4, 0, 11, 210, 0, 0, 16, 12, 15, 13, 0, 11, 193, 4, 0, 11, 249, 0, 0, 11, 193, 4, 0, 6, 247, 0, 11, 193, 4, 0, 11, 178, 4, 0, 11, 193, 4, 0, 11, 249, 0, 0, 9, 228, 0, 16, 12, 15, 14, 0, 11, 193, 4, 0, 6, 10, 1, 11, 12, 1, 0, 9, 3, 1, 16, 12, 15, 16, 0, 11, 193, 4, 0, 6, 58, 1, 6, 35, 1, 15, 6, 0, 2, 35, 0, 16, 8, 55, 1, 6, 48, 1, 15, 6, 0, 2, 38, 0, 16, 8, 55, 1, 15, 6, 0, 2, 33, 0, 16, 8, 58, 1, 11, 193, 4, 0, 11, 68, 1, 0, 16, 12, 15, 17, 0, 11, 193, 4, 0, 11, 116, 1, 0, 6, 114, 1, 15, 6, 0, 6, 94, 1, 2, 94, 0, 8, 97, 1, 2, 209, 33, 16, 6, 108, 1, 11, 93, 2, 0, 8, 111, 1, 14, 19, 0, 8, 114, 1, 16, 12, 15, 18, 0, 11, 193, 4, 0, 11, 182, 1, 0, 11, 193, 4, 0, 6, 180, 1, 6, 147, 1, 15, 6, 0, 2, 63, 0, 16, 8, 177, 1, 6, 160, 1, 15, 6, 0, 2, 42, 0, 16, 8, 177, 1, 6, 173, 1, 15, 6, 0, 2, 43, 0, 16, 8, 177, 1, 11, 87, 4, 0, 8, 180, 1, 16, 12, 15, 20, 0, 6, 207, 1, 11, 93, 2, 0, 11, 193, 4, 0, 7, 204, 1, 11, 167, 4, 0, 5, 8, 26, 2, 6, 248, 1, 15, 6, 0, 2, 40, 0, 16, 11, 193, 4, 0, 11, 210, 0, 0, 11, 193, 4, 0, 6, 242, 1, 15, 6, 0, 2, 41, 0, 16, 8, 245, 1, 14, 22, 0, 8, 26, 2, 6, 2, 2, 11, 28, 2, 0, 8, 26, 2, 6, 12, 2, 11, 165, 2, 0, 8, 26, 2, 6, 22, 2, 11, 0, 3, 0, 8, 26, 2, 11, 79, 4, 0, 16, 12, 15, 23, 0, 11, 193, 4, 0, 15, 6, 0, 2, 123, 0, 16, 11, 193, 4, 0, 6, 71, 2, 11, 193, 4, 0, 7, 60, 2, 2, 125, 0, 5, 11, 193, 4, 0, 11, 210, 0, 0, 9, 49, 2, 11, 193, 4, 0, 6, 88, 2, 15, 6, 0, 2, 125, 0, 16, 8, 91, 2, 14, 26, 0, 16, 12, 15, 8, 0, 6, 107, 2, 3, 97, 0, 122, 0, 8, 121, 2, 6, 118, 2, 3, 65, 0, 90, 0, 8, 121, 2, 2, 95, 0, 6, 163, 2, 6, 135, 2, 3, 97, 0, 122, 0, 8, 160, 2, 6, 146, 2, 3, 65, 0, 90, 0, 8, 160, 2, 6, 157, 2, 3, 48, 0, 57, 0, 8, 160, 2, 2, 95, 0, 9, 124, 2, 16, 12, 15, 11, 0, 6, 214, 2, 15, 6, 0, 2, 39, 0, 16, 6, 195, 2, 7, 188, 2, 2, 39, 0, 5, 11, 88, 3, 0, 9, 181, 2, 6, 208, 2, 15, 6, 0, 2, 39, 0, 16, 8, 211, 2, 14, 28, 0, 8, 254, 2, 15, 6, 0, 2, 34, 0, 16, 6, 238, 2, 7, 231, 2, 2, 34, 0, 5, 11, 88, 3, 0, 9, 224, 2, 6, 251, 2, 15, 6, 0, 2, 34, 0, 16, 8, 254, 2, 14, 29, 0, 16, 12, 15, 24, 0, 11, 193, 4, 0, 15, 6, 0, 2, 91, 0, 16, 6, 31, 3, 7, 24, 3, 2, 93, 0, 5, 11, 49, 3, 0, 9, 17, 3, 6, 44, 3, 15, 6, 0, 2, 93, 0, 16, 8, 47, 3, 14, 31, 0, 16, 12, 15, 30, 0, 6, 82, 3, 11, 88, 3, 0, 15, 6, 0, 2, 45, 0, 16, 6, 76, 3, 11, 88, 3, 0, 8, 79, 3, 14, 32, 0, 8, 86, 3, 11, 88, 3, 0, 16, 12, 15, 27, 0, 6, 101, 3, 11, 125, 3, 0, 8, 123, 3, 6, 111, 3, 11, 208, 3, 0, 8, 123, 3, 7, 118, 3, 2, 92, 0, 5, 15, 6, 0, 1, 16, 16, 12, 15, 33, 0, 2, 92, 0, 6, 140, 3, 2, 110, 0, 8, 206, 3, 6, 149, 3, 2, 114, 0, 8, 206, 3, 6, 158, 3, 2, 116, 0, 8, 206, 3, 6, 167, 3, 2, 39, 0, 8, 206, 3, 6, 176, 3, 2, 34, 0, 8, 206, 3, 6, 185, 3, 2, 45, 0, 8, 206, 3, 6, 194, 3, 2, 91, 0, 8, 206, 3, 6, 203, 3, 2, 93, 0, 8, 206, 3, 2, 92, 0, 16, 12, 15, 34, 0, 11, 193, 4, 0, 15, 6, 0, 2, 92, 0, 16, 11, 193, 4, 0, 15, 6, 0, 2, 117, 0, 16, 11, 193, 4, 0, 6, 247, 3, 11, 47, 4, 0, 8, 250, 3, 14, 35, 0, 11, 193, 4, 0, 6, 8, 4, 11, 47, 4, 0, 8, 11, 4, 14, 37, 0, 11, 193, 4, 0, 6, 25, 4, 11, 47, 4, 0, 8, 28, 4, 14, 38, 0, 11, 193, 4, 0, 6, 42, 4, 11, 47, 4, 0, 8, 45, 4, 14, 39, 0, 16, 12, 15, 36, 0, 6, 61, 4, 3, 48, 0, 57, 0, 8, 77, 4, 6, 72, 4, 3, 65, 0, 70, 0, 8, 77, 4, 3, 97, 0, 102, 0, 16, 12, 15, 25, 0, 2, 46, 0, 16, 12, 15, 21, 0, 6, 99, 4, 2, 185, 0, 8, 165, 4, 6, 108, 4, 2, 178, 0, 8, 165, 4, 6, 117, 4, 2, 179, 0, 8, 165, 4, 6, 126, 4, 2, 116, 32, 8, 165, 4, 6, 135, 4, 2, 117, 32, 8, 165, 4, 6, 144, 4, 2, 118, 32, 8, 165, 4, 6, 153, 4, 2, 119, 32, 8, 165, 4, 6, 162, 4, 2, 120, 32, 8, 165, 4, 2, 121, 32, 16, 12, 15, 12, 0, 2, 60, 0, 2, 45, 0, 16, 12, 15, 15, 0, 2, 47, 0, 7, 191, 4, 2, 47, 0, 5, 16, 12, 15, 1, 0, 6, 226, 4, 6, 209, 4, 11, 228, 4, 0, 8, 223, 4, 6, 219, 4, 11, 16, 5, 0, 8, 223, 4, 11, 42, 5, 0, 9, 199, 4, 16, 12, 15, 40, 0, 15, 6, 0, 2, 47, 0, 2, 47, 0, 16, 15, 6, 0, 6, 3, 5, 7, 255, 4, 11, 42, 5, 0, 5, 1, 9, 247, 4, 16, 6, 14, 5, 11, 42, 5, 0, 8, 14, 5, 16, 12, 15, 41, 0, 6, 28, 5, 2, 9, 0, 8, 40, 5, 6, 37, 5, 2, 12, 0, 8, 40, 5, 2, 32, 0, 16, 12, 15, 42, 0, 6, 54, 5, 2, 10, 0, 8, 57, 5, 2, 13, 0, 16, 12, 15, 5, 0, 7, 67, 5, 1, 5, 16, 12, 15, 22, 0, 16, 12, 15, 26, 0, 16, 12, 15, 31, 0, 16, 12, 15, 28, 0, 16, 12, 15, 29, 0, 16, 12, 15, 19, 0, 6, 112, 5, 7, 108, 5, 11, 182, 1, 0, 5, 1, 9, 100, 5, 16, 12, 15, 7, 0, 6, 150, 5, 7, 146, 5, 6, 141, 5, 2, 102, 0, 2, 114, 0, 2, 111, 0, 2, 109, 0, 8, 145, 5, 11, 165, 2, 0, 5, 1, 9, 120, 5, 16, 12, 15, 9, 0, 6, 170, 5, 7, 166, 5, 11, 165, 2, 0, 5, 1, 9, 158, 5, 16, 12, 15, 10, 0, 6, 189, 5, 7, 185, 5, 2, 10, 0, 5, 1, 9, 178, 5, 16, 12, 
	},
	strs: []string{
		"Grammar", "Spacing", "Import", "Definition", "eof", "EOF", "", "MissingImportName", "Identifier", "MissingImportFrom", "MissingImportSrc", "Literal", "LEFTARROW", "Expression", "Sequence", "SLASH", "Prefix", "Labeled", "Suffix", "MissingLabel", "Primary", "Superscript", "MissingClosingParen", "List", "Class", "Any", "MissingClosingCurly", "Char", "MissingClosingSQuote", "MissingClosingDQuote", "Range", "MissingClosingBracket", "MissingRightRange", "Escape", "Unicode", "chrH1", "Hex", "chrH2", "chrH3", "chrH4", "Comment", "Space", "EOL", 
	},
	rxps: map[int]int{
		7: 1394, 9: 1432, 10: 1452, 19: 1374, 22: 1349, 26: 1354, 28: 1364, 29: 1369, 31: 1359, 
	},
	smap: map[string]int{
		"": 6, "Any": 25, "Char": 27, "Class": 24, "Comment": 40, "Definition": 3, "EOF": 5, "EOL": 42, "Escape": 33, "Expression": 13, "Grammar": 0, "Hex": 36, "Identifier": 8, "Import": 2, "LEFTARROW": 12, "Labeled": 17, "List": 23, "Literal": 11, "MissingClosingBracket": 31, "MissingClosingCurly": 26, "MissingClosingDQuote": 29, "MissingClosingParen": 22, "MissingClosingSQuote": 28, "MissingImportFrom": 9, "MissingImportName": 7, "MissingImportSrc": 10, "MissingLabel": 19, "MissingRightRange": 32, "Prefix": 16, "Primary": 20, "Range": 30, "SLASH": 15, "Sequence": 14, "Space": 41, "Spacing": 1, "Suffix": 18, "Superscript": 21, "Unicode": 34, "chrH1": 35, "chrH2": 37, "chrH3": 38, "chrH4": 39, "eof": 4, 
	},
}
type GrammarParserBootstrap struct{
	input         string
	captureSpaces bool
	suppress      map[int]struct{}
	errLabels     map[string]string
}
func NewGrammarParserBootstrap() *GrammarParserBootstrap {
	suppress := map[int]struct{}{bytecodeForGrammarParserBootstrap.smap["Spacing"]: struct{}{}}
	return &GrammarParserBootstrap{captureSpaces: true, suppress: suppress}
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
func (p *GrammarParserBootstrap) ParseChar() (Value, error) { return p.parseFn(856) }
func (p *GrammarParserBootstrap) ParseEscape() (Value, error) { return p.parseFn(893) }
func (p *GrammarParserBootstrap) ParseUnicode() (Value, error) { return p.parseFn(976) }
func (p *GrammarParserBootstrap) ParseHex() (Value, error) { return p.parseFn(1071) }
func (p *GrammarParserBootstrap) ParseAny() (Value, error) { return p.parseFn(1103) }
func (p *GrammarParserBootstrap) ParseSuperscript() (Value, error) { return p.parseFn(1111) }
func (p *GrammarParserBootstrap) ParseLEFTARROW() (Value, error) { return p.parseFn(1191) }
func (p *GrammarParserBootstrap) ParseSLASH() (Value, error) { return p.parseFn(1202) }
func (p *GrammarParserBootstrap) ParseSpacing() (Value, error) { return p.parseFn(1217) }
func (p *GrammarParserBootstrap) ParseComment() (Value, error) { return p.parseFn(1252) }
func (p *GrammarParserBootstrap) ParseSpace() (Value, error) { return p.parseFn(1296) }
func (p *GrammarParserBootstrap) ParseEOL() (Value, error) { return p.parseFn(1322) }
func (p *GrammarParserBootstrap) ParseEOF() (Value, error) { return p.parseFn(1339) }
func (p *GrammarParserBootstrap) ParseMissingClosingParen() (Value, error) { return p.parseFn(1349) }
func (p *GrammarParserBootstrap) ParseMissingClosingCurly() (Value, error) { return p.parseFn(1354) }
func (p *GrammarParserBootstrap) ParseMissingClosingBracket() (Value, error) { return p.parseFn(1359) }
func (p *GrammarParserBootstrap) ParseMissingClosingSQuote() (Value, error) { return p.parseFn(1364) }
func (p *GrammarParserBootstrap) ParseMissingClosingDQuote() (Value, error) { return p.parseFn(1369) }
func (p *GrammarParserBootstrap) ParseMissingLabel() (Value, error) { return p.parseFn(1374) }
func (p *GrammarParserBootstrap) ParseMissingImportName() (Value, error) { return p.parseFn(1394) }
func (p *GrammarParserBootstrap) ParseMissingImportFrom() (Value, error) { return p.parseFn(1432) }
func (p *GrammarParserBootstrap) ParseMissingImportSrc() (Value, error) { return p.parseFn(1452) }
func (p *GrammarParserBootstrap) Parse() (Value, error) { return p.parseFn(5) }
func (p *GrammarParserBootstrap) SetInput(input string) { p.input = input }
func (p *GrammarParserBootstrap) SetLabelMessages(el map[string]string) { p.errLabels = el }
func (p *GrammarParserBootstrap) SetCaptureSpaces(v bool) { p.captureSpaces = v }
func (p *GrammarParserBootstrap) parseFn(addr uint16) (Value, error) {
	writeU16(bytecodeForGrammarParserBootstrap.code[1:], addr)
	var suppress map[int]struct{}
	if !p.captureSpaces {
		suppress = p.suppress
	}
	vm := newVirtualMachine(bytecodeForGrammarParserBootstrap, p.errLabels, suppress)
	val, _, err := vm.Match(strings.NewReader(p.input))
	return val, err
}
