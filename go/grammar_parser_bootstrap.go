package langlang

var bytecodeForGrammarParserBootstrap = &Bytecode{
	code: []byte{
		11, 5, 0, 0, 0, 15, 0, 0, 11, 231, 3, 0, 6, 22, 0, 11, 55, 0, 0, 9, 15, 0, 11, 231, 3, 0, 6, 36, 0, 11, 181, 0, 0, 9, 29, 0, 11, 231, 3, 0, 6, 50, 0, 11, 70, 4, 0, 8, 53, 0, 14, 4, 0, 16, 12, 15, 2, 0, 11, 231, 3, 0, 15, 6, 0, 2, 64, 0, 2, 105, 0, 2, 109, 0, 2, 112, 0, 2, 111, 0, 2, 114, 0, 2, 116, 0, 16, 11, 231, 3, 0, 6, 101, 0, 11, 63, 2, 0, 8, 104, 0, 14, 7, 0, 11, 231, 3, 0, 6, 133, 0, 11, 231, 3, 0, 15, 6, 0, 17, 0, 0, 16, 11, 231, 3, 0, 11, 63, 2, 0, 9, 111, 0, 11, 231, 3, 0, 6, 159, 0, 15, 6, 0, 2, 102, 0, 2, 114, 0, 2, 111, 0, 2, 109, 0, 16, 8, 162, 0, 14, 9, 0, 11, 231, 3, 0, 6, 176, 0, 11, 74, 2, 0, 8, 179, 0, 14, 10, 0, 16, 12, 15, 3, 0, 11, 231, 3, 0, 11, 63, 2, 0, 11, 231, 3, 0, 11, 205, 3, 0, 11, 231, 3, 0, 11, 210, 0, 0, 16, 12, 15, 13, 0, 11, 231, 3, 0, 11, 249, 0, 0, 11, 231, 3, 0, 6, 247, 0, 11, 231, 3, 0, 11, 216, 3, 0, 11, 231, 3, 0, 11, 249, 0, 0, 9, 228, 0, 16, 12, 15, 14, 0, 6, 6, 1, 11, 8, 1, 0, 9, 255, 0, 16, 12, 15, 16, 0, 11, 231, 3, 0, 6, 28, 1, 15, 6, 0, 17, 1, 0, 16, 8, 28, 1, 11, 231, 3, 0, 11, 38, 1, 0, 16, 12, 15, 17, 0, 11, 231, 3, 0, 11, 86, 1, 0, 6, 84, 1, 15, 6, 0, 6, 64, 1, 2, 94, 0, 8, 67, 1, 2, 209, 33, 16, 6, 78, 1, 11, 63, 2, 0, 8, 81, 1, 14, 19, 0, 8, 84, 1, 16, 12, 15, 18, 0, 11, 231, 3, 0, 11, 152, 1, 0, 11, 231, 3, 0, 6, 150, 1, 6, 117, 1, 15, 6, 0, 17, 2, 0, 16, 8, 147, 1, 6, 130, 1, 15, 6, 0, 17, 3, 0, 16, 8, 147, 1, 6, 143, 1, 15, 6, 0, 17, 4, 0, 16, 8, 147, 1, 11, 125, 3, 0, 8, 150, 1, 16, 12, 15, 20, 0, 6, 177, 1, 11, 63, 2, 0, 11, 231, 3, 0, 7, 174, 1, 11, 205, 3, 0, 5, 8, 252, 1, 6, 218, 1, 15, 6, 0, 17, 5, 0, 16, 11, 231, 3, 0, 11, 210, 0, 0, 11, 231, 3, 0, 6, 212, 1, 15, 6, 0, 17, 6, 0, 16, 8, 215, 1, 14, 22, 0, 8, 252, 1, 6, 228, 1, 11, 254, 1, 0, 8, 252, 1, 6, 238, 1, 11, 74, 2, 0, 8, 252, 1, 6, 248, 1, 11, 165, 2, 0, 8, 252, 1, 11, 117, 3, 0, 16, 12, 15, 23, 0, 11, 231, 3, 0, 15, 6, 0, 17, 7, 0, 16, 11, 231, 3, 0, 6, 41, 2, 11, 231, 3, 0, 7, 30, 2, 17, 8, 0, 5, 11, 231, 3, 0, 11, 210, 0, 0, 9, 19, 2, 11, 231, 3, 0, 6, 58, 2, 15, 6, 0, 17, 8, 0, 16, 8, 61, 2, 14, 26, 0, 16, 12, 15, 8, 0, 17, 9, 0, 18, 10, 0, 16, 12, 15, 11, 0, 6, 123, 2, 15, 6, 0, 17, 11, 0, 16, 6, 104, 2, 7, 97, 2, 17, 11, 0, 5, 11, 253, 2, 0, 9, 90, 2, 6, 117, 2, 15, 6, 0, 17, 11, 0, 16, 8, 120, 2, 14, 28, 0, 8, 163, 2, 15, 6, 0, 17, 12, 0, 16, 6, 147, 2, 7, 140, 2, 17, 12, 0, 5, 11, 253, 2, 0, 9, 133, 2, 6, 160, 2, 15, 6, 0, 17, 12, 0, 16, 8, 163, 2, 14, 29, 0, 16, 12, 15, 24, 0, 11, 231, 3, 0, 15, 6, 0, 17, 13, 0, 16, 6, 196, 2, 7, 189, 2, 17, 14, 0, 5, 11, 214, 2, 0, 9, 182, 2, 6, 209, 2, 15, 6, 0, 17, 14, 0, 16, 8, 212, 2, 14, 31, 0, 16, 12, 15, 30, 0, 6, 247, 2, 11, 253, 2, 0, 15, 6, 0, 17, 15, 0, 16, 6, 241, 2, 11, 253, 2, 0, 8, 244, 2, 14, 32, 0, 8, 251, 2, 11, 253, 2, 0, 16, 12, 15, 27, 0, 6, 10, 3, 11, 27, 3, 0, 8, 25, 3, 6, 20, 3, 11, 38, 3, 0, 8, 25, 3, 15, 6, 0, 1, 16, 16, 12, 15, 33, 0, 17, 16, 0, 17, 17, 0, 16, 12, 15, 34, 0, 15, 6, 0, 17, 16, 0, 16, 15, 6, 0, 17, 18, 0, 16, 6, 65, 3, 11, 109, 3, 0, 8, 68, 3, 14, 35, 0, 6, 78, 3, 11, 109, 3, 0, 8, 81, 3, 14, 37, 0, 6, 91, 3, 11, 109, 3, 0, 8, 94, 3, 14, 38, 0, 6, 104, 3, 11, 109, 3, 0, 8, 107, 3, 14, 39, 0, 16, 12, 15, 36, 0, 17, 19, 0, 16, 12, 15, 25, 0, 17, 20, 0, 16, 12, 15, 21, 0, 6, 137, 3, 2, 185, 0, 8, 203, 3, 6, 146, 3, 2, 178, 0, 8, 203, 3, 6, 155, 3, 2, 179, 0, 8, 203, 3, 6, 164, 3, 2, 116, 32, 8, 203, 3, 6, 173, 3, 2, 117, 32, 8, 203, 3, 6, 182, 3, 2, 118, 32, 8, 203, 3, 6, 191, 3, 2, 119, 32, 8, 203, 3, 6, 200, 3, 2, 120, 32, 8, 203, 3, 2, 121, 32, 16, 12, 15, 12, 0, 2, 60, 0, 2, 45, 0, 16, 12, 15, 15, 0, 17, 21, 0, 7, 229, 3, 17, 21, 0, 5, 16, 12, 15, 1, 0, 6, 8, 4, 6, 247, 3, 11, 10, 4, 0, 8, 5, 4, 6, 1, 4, 11, 54, 4, 0, 8, 5, 4, 11, 62, 4, 0, 9, 237, 3, 16, 12, 15, 40, 0, 15, 6, 0, 2, 47, 0, 2, 47, 0, 16, 15, 6, 0, 6, 41, 4, 7, 37, 4, 11, 62, 4, 0, 5, 1, 9, 29, 4, 16, 6, 52, 4, 11, 62, 4, 0, 8, 52, 4, 16, 12, 15, 41, 0, 17, 22, 0, 16, 12, 15, 42, 0, 17, 23, 0, 16, 12, 15, 5, 0, 7, 78, 4, 1, 5, 16, 12, 15, 22, 0, 16, 12, 15, 26, 0, 16, 12, 15, 31, 0, 16, 12, 15, 28, 0, 16, 12, 15, 29, 0, 16, 12, 15, 19, 0, 6, 123, 4, 7, 119, 4, 11, 152, 1, 0, 5, 1, 9, 111, 4, 16, 12, 15, 7, 0, 6, 161, 4, 7, 157, 4, 6, 152, 4, 2, 102, 0, 2, 114, 0, 2, 111, 0, 2, 109, 0, 8, 156, 4, 11, 74, 2, 0, 5, 1, 9, 131, 4, 16, 12, 15, 9, 0, 6, 181, 4, 7, 177, 4, 11, 74, 2, 0, 5, 1, 9, 169, 4, 16, 12, 15, 10, 0, 18, 24, 0, 16, 12, 
	},
	strs: []string{
		"Grammar", "Spacing", "Import", "Definition", "eof", "EOF", "", "MissingImportName", "Identifier", "MissingImportFrom", "MissingImportSrc", "Literal", "LEFTARROW", "Expression", "Sequence", "SLASH", "Prefix", "Labeled", "Suffix", "MissingLabel", "Primary", "Superscript", "MissingClosingParen", "List", "Class", "Any", "MissingClosingCurly", "Char", "MissingClosingSQuote", "MissingClosingDQuote", "Range", "MissingClosingBracket", "MissingRightRange", "Escape", "Unicode", "chrH1", "Hex", "chrH2", "chrH3", "chrH4", "Comment", "Space", "EOL", 
	},
	rxps: map[int]int{
		7: 1149, 9: 1187, 10: 1207, 19: 1129, 22: 1104, 26: 1109, 28: 1119, 29: 1124, 31: 1114, 
	},
	smap: map[string]int{
		"": 6, "Any": 25, "Char": 27, "Class": 24, "Comment": 40, "Definition": 3, "EOF": 5, "EOL": 42, "Escape": 33, "Expression": 13, "Grammar": 0, "Hex": 36, "Identifier": 8, "Import": 2, "LEFTARROW": 12, "Labeled": 17, "List": 23, "Literal": 11, "MissingClosingBracket": 31, "MissingClosingCurly": 26, "MissingClosingDQuote": 29, "MissingClosingParen": 22, "MissingClosingSQuote": 28, "MissingImportFrom": 9, "MissingImportName": 7, "MissingImportSrc": 10, "MissingLabel": 19, "MissingRightRange": 32, "Prefix": 16, "Primary": 20, "Range": 30, "SLASH": 15, "Sequence": 14, "Space": 41, "Spacing": 1, "Suffix": 18, "Superscript": 21, "Unicode": 34, "chrH1": 35, "chrH2": 37, "chrH3": 38, "chrH4": 39, "eof": 4, 
	},
	sets: []charset{
		{bits: [32]byte{0,0,0,0,0,16,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,74,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,0,0,0,128,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,0,4,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,0,8,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,0,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,0,2,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,8,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,32,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,0,0,0,0,254,255,255,135,254,255,255,7,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,0,0,255,3,254,255,255,135,254,255,255,7,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,128,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,4,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,0,0,0,0,0,0,0,8,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,0,0,0,0,0,0,0,32,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,0,32,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,0,0,0,0,0,0,0,16,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,132,32,0,0,0,0,0,56,0,64,20,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,0,0,0,0,0,0,0,0,0,0,32,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,0,0,255,3,126,0,0,0,126,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,0,64,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,0,0,0,0,128,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,18,0,0,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{0,36,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,}},
		{bits: [32]byte{255,251,255,255,255,255,255,255,255,255,255,255,255,255,255,255,255,255,255,255,255,255,255,255,255,255,255,255,255,255,255,255,}},
	},
	sexp: [][]expected{
		{expected{a: ','},},
		{expected{a: '!'},expected{a: '#'},expected{a: '&'},},
		{expected{a: '?'},},
		{expected{a: '*'},},
		{expected{a: '+'},},
		{expected{a: '('},},
		{expected{a: ')'},},
		{expected{a: '{'},},
		{expected{a: '}'},},
		{expected{a: 'A', b: 'Z'},expected{a: '_'},expected{a: 'a', b: 'z'},},
		{expected{a: '0', b: '9'},expected{a: 'A', b: 'Z'},expected{a: '_'},expected{a: 'a', b: 'z'},},
		{expected{a: '\''},},
		{expected{a: '"'},},
		{expected{a: '['},},
		{expected{a: ']'},},
		{expected{a: '-'},},
		{expected{a: '\\'},},
		{expected{a: '"'},expected{a: '\''},expected{a: '-'},expected{a: '[', b: ']'},expected{a: 'n'},expected{a: 'r'},expected{a: 't'},},
		{expected{a: 'u'},},
		{expected{a: '0', b: '9'},expected{a: 'A', b: 'F'},expected{a: 'a', b: 'f'},},
		{expected{a: '.'},},
		{expected{a: '/'},},
		{expected{a: '\t'},expected{a: '\f'},expected{a: ' '},},
		{expected{a: '\n'},expected{a: '\r'},},
		{},
	},
}
type GrammarParserBootstrap struct{
	input         string
	captureSpaces bool
	showFails     bool
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
func (p *GrammarParserBootstrap) ParsePrefix() (Value, error) { return p.parseFn(264) }
func (p *GrammarParserBootstrap) ParseLabeled() (Value, error) { return p.parseFn(294) }
func (p *GrammarParserBootstrap) ParseSuffix() (Value, error) { return p.parseFn(342) }
func (p *GrammarParserBootstrap) ParsePrimary() (Value, error) { return p.parseFn(408) }
func (p *GrammarParserBootstrap) ParseList() (Value, error) { return p.parseFn(510) }
func (p *GrammarParserBootstrap) ParseIdentifier() (Value, error) { return p.parseFn(575) }
func (p *GrammarParserBootstrap) ParseLiteral() (Value, error) { return p.parseFn(586) }
func (p *GrammarParserBootstrap) ParseClass() (Value, error) { return p.parseFn(677) }
func (p *GrammarParserBootstrap) ParseRange() (Value, error) { return p.parseFn(726) }
func (p *GrammarParserBootstrap) ParseChar() (Value, error) { return p.parseFn(765) }
func (p *GrammarParserBootstrap) ParseEscape() (Value, error) { return p.parseFn(795) }
func (p *GrammarParserBootstrap) ParseUnicode() (Value, error) { return p.parseFn(806) }
func (p *GrammarParserBootstrap) ParseHex() (Value, error) { return p.parseFn(877) }
func (p *GrammarParserBootstrap) ParseAny() (Value, error) { return p.parseFn(885) }
func (p *GrammarParserBootstrap) ParseSuperscript() (Value, error) { return p.parseFn(893) }
func (p *GrammarParserBootstrap) ParseLEFTARROW() (Value, error) { return p.parseFn(973) }
func (p *GrammarParserBootstrap) ParseSLASH() (Value, error) { return p.parseFn(984) }
func (p *GrammarParserBootstrap) ParseSpacing() (Value, error) { return p.parseFn(999) }
func (p *GrammarParserBootstrap) ParseComment() (Value, error) { return p.parseFn(1034) }
func (p *GrammarParserBootstrap) ParseSpace() (Value, error) { return p.parseFn(1078) }
func (p *GrammarParserBootstrap) ParseEOL() (Value, error) { return p.parseFn(1086) }
func (p *GrammarParserBootstrap) ParseEOF() (Value, error) { return p.parseFn(1094) }
func (p *GrammarParserBootstrap) ParseMissingClosingParen() (Value, error) { return p.parseFn(1104) }
func (p *GrammarParserBootstrap) ParseMissingClosingCurly() (Value, error) { return p.parseFn(1109) }
func (p *GrammarParserBootstrap) ParseMissingClosingBracket() (Value, error) { return p.parseFn(1114) }
func (p *GrammarParserBootstrap) ParseMissingClosingSQuote() (Value, error) { return p.parseFn(1119) }
func (p *GrammarParserBootstrap) ParseMissingClosingDQuote() (Value, error) { return p.parseFn(1124) }
func (p *GrammarParserBootstrap) ParseMissingLabel() (Value, error) { return p.parseFn(1129) }
func (p *GrammarParserBootstrap) ParseMissingImportName() (Value, error) { return p.parseFn(1149) }
func (p *GrammarParserBootstrap) ParseMissingImportFrom() (Value, error) { return p.parseFn(1187) }
func (p *GrammarParserBootstrap) ParseMissingImportSrc() (Value, error) { return p.parseFn(1207) }
func (p *GrammarParserBootstrap) Parse() (Value, error)                 { return p.parseFn(5) }
func (p *GrammarParserBootstrap) SetInput(input string)                 { p.input = input }
func (p *GrammarParserBootstrap) SetLabelMessages(el map[string]string) { p.errLabels = el }
func (p *GrammarParserBootstrap) SetCaptureSpaces(v bool)               { p.captureSpaces = v }
func (p *GrammarParserBootstrap) SetShowFails(v bool)                   { p.showFails = v }
func (p *GrammarParserBootstrap) parseFn(addr uint16) (Value, error)    {
	writeU16(bytecodeForGrammarParserBootstrap.code[1:], addr)
	var suppress map[int]struct{}
	if !p.captureSpaces {
		suppress = p.suppress
	}
	vm := newVirtualMachine(bytecodeForGrammarParserBootstrap, p.errLabels, suppress, p.showFails)
	input := NewMemInput(p.input)
	val, _, err := vm.Match(&input)
	return val, err
}
