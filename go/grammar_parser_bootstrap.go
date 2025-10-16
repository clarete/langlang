package langlang

var bytecodeForGrammarParserBootstrap = &Bytecode{
	code: []byte{
		11, 5, 0, 0, 0, 15, 0, 0, 11, 215, 3, 0, 6, 22, 0, 11, 55, 0, 0, 9, 15, 0, 11, 215, 3, 0, 6, 36, 0, 11, 178, 0, 0, 9, 29, 0, 11, 215, 3, 0, 6, 50, 0, 11, 55, 4, 0, 8, 53, 0, 14, 4, 0, 16, 12, 15, 2, 0, 11, 215, 3, 0, 2, 64, 0, 2, 105, 0, 2, 109, 0, 2, 112, 0, 2, 111, 0, 2, 114, 0, 2, 116, 0, 19, 7, 0, 11, 215, 3, 0, 6, 100, 0, 11, 52, 2, 0, 8, 103, 0, 14, 7, 0, 11, 215, 3, 0, 6, 131, 0, 11, 215, 3, 0, 17, 0, 0, 19, 1, 0, 11, 215, 3, 0, 11, 52, 2, 0, 9, 110, 0, 11, 215, 3, 0, 6, 156, 0, 2, 102, 0, 2, 114, 0, 2, 111, 0, 2, 109, 0, 19, 4, 0, 8, 159, 0, 14, 9, 0, 11, 215, 3, 0, 6, 173, 0, 11, 63, 2, 0, 8, 176, 0, 14, 10, 0, 16, 12, 15, 3, 0, 11, 215, 3, 0, 11, 52, 2, 0, 11, 215, 3, 0, 11, 188, 3, 0, 11, 215, 3, 0, 11, 207, 0, 0, 16, 12, 15, 13, 0, 11, 215, 3, 0, 11, 246, 0, 0, 11, 215, 3, 0, 6, 244, 0, 11, 215, 3, 0, 11, 200, 3, 0, 11, 215, 3, 0, 11, 246, 0, 0, 9, 225, 0, 16, 12, 15, 14, 0, 6, 3, 1, 11, 5, 1, 0, 9, 252, 0, 16, 12, 15, 16, 0, 11, 215, 3, 0, 6, 24, 1, 17, 1, 0, 19, 1, 0, 8, 24, 1, 11, 215, 3, 0, 11, 34, 1, 0, 16, 12, 15, 17, 0, 11, 215, 3, 0, 11, 82, 1, 0, 6, 80, 1, 15, 6, 0, 6, 60, 1, 2, 94, 0, 8, 63, 1, 2, 209, 33, 16, 6, 74, 1, 11, 52, 2, 0, 8, 77, 1, 14, 19, 0, 8, 80, 1, 16, 12, 15, 18, 0, 11, 215, 3, 0, 11, 145, 1, 0, 11, 215, 3, 0, 6, 143, 1, 6, 112, 1, 17, 2, 0, 19, 1, 0, 8, 140, 1, 6, 124, 1, 17, 3, 0, 19, 1, 0, 8, 140, 1, 6, 136, 1, 17, 4, 0, 19, 1, 0, 8, 140, 1, 11, 108, 3, 0, 8, 143, 1, 16, 12, 15, 20, 0, 6, 170, 1, 11, 52, 2, 0, 11, 215, 3, 0, 7, 167, 1, 11, 188, 3, 0, 5, 8, 243, 1, 6, 209, 1, 17, 5, 0, 19, 1, 0, 11, 215, 3, 0, 11, 207, 0, 0, 11, 215, 3, 0, 6, 203, 1, 17, 6, 0, 19, 1, 0, 8, 206, 1, 14, 22, 0, 8, 243, 1, 6, 219, 1, 11, 245, 1, 0, 8, 243, 1, 6, 229, 1, 11, 63, 2, 0, 8, 243, 1, 6, 239, 1, 11, 150, 2, 0, 8, 243, 1, 11, 99, 3, 0, 16, 12, 15, 23, 0, 11, 215, 3, 0, 17, 7, 0, 19, 1, 0, 11, 215, 3, 0, 6, 31, 2, 11, 215, 3, 0, 7, 20, 2, 17, 8, 0, 5, 11, 215, 3, 0, 11, 207, 0, 0, 9, 9, 2, 11, 215, 3, 0, 6, 47, 2, 17, 8, 0, 19, 1, 0, 8, 50, 2, 14, 26, 0, 16, 12, 15, 8, 0, 17, 9, 0, 18, 10, 0, 16, 12, 15, 11, 0, 6, 110, 2, 17, 11, 0, 19, 1, 0, 6, 92, 2, 7, 85, 2, 17, 11, 0, 5, 11, 235, 2, 0, 9, 78, 2, 6, 104, 2, 17, 11, 0, 19, 1, 0, 8, 107, 2, 14, 28, 0, 8, 148, 2, 17, 12, 0, 19, 1, 0, 6, 133, 2, 7, 126, 2, 17, 12, 0, 5, 11, 235, 2, 0, 9, 119, 2, 6, 145, 2, 17, 12, 0, 19, 1, 0, 8, 148, 2, 14, 29, 0, 16, 12, 15, 24, 0, 11, 215, 3, 0, 17, 13, 0, 19, 1, 0, 6, 180, 2, 7, 173, 2, 17, 14, 0, 5, 11, 197, 2, 0, 9, 166, 2, 6, 192, 2, 17, 14, 0, 19, 1, 0, 8, 195, 2, 14, 31, 0, 16, 12, 15, 30, 0, 6, 229, 2, 11, 235, 2, 0, 17, 15, 0, 19, 1, 0, 6, 223, 2, 11, 235, 2, 0, 8, 226, 2, 14, 32, 0, 8, 233, 2, 11, 235, 2, 0, 16, 12, 15, 27, 0, 6, 248, 2, 11, 9, 3, 0, 8, 7, 3, 6, 2, 3, 11, 21, 3, 0, 8, 7, 3, 15, 6, 0, 1, 16, 16, 12, 17, 16, 0, 17, 17, 0, 20, 33, 0, 2, 0, 12, 15, 34, 0, 17, 16, 0, 19, 1, 0, 17, 18, 0, 19, 1, 0, 6, 46, 3, 11, 90, 3, 0, 8, 49, 3, 14, 35, 0, 6, 59, 3, 11, 90, 3, 0, 8, 62, 3, 14, 37, 0, 6, 72, 3, 11, 90, 3, 0, 8, 75, 3, 14, 38, 0, 6, 85, 3, 11, 90, 3, 0, 8, 88, 3, 14, 39, 0, 16, 12, 17, 19, 0, 20, 36, 0, 1, 0, 12, 17, 20, 0, 20, 25, 0, 1, 0, 12, 15, 21, 0, 6, 120, 3, 2, 185, 0, 8, 186, 3, 6, 129, 3, 2, 178, 0, 8, 186, 3, 6, 138, 3, 2, 179, 0, 8, 186, 3, 6, 147, 3, 2, 116, 32, 8, 186, 3, 6, 156, 3, 2, 117, 32, 8, 186, 3, 6, 165, 3, 2, 118, 32, 8, 186, 3, 6, 174, 3, 2, 119, 32, 8, 186, 3, 6, 183, 3, 2, 120, 32, 8, 186, 3, 2, 121, 32, 16, 12, 2, 60, 0, 2, 45, 0, 20, 12, 0, 2, 0, 12, 15, 15, 0, 17, 21, 0, 7, 213, 3, 17, 21, 0, 5, 16, 12, 15, 1, 0, 6, 248, 3, 6, 231, 3, 11, 250, 3, 0, 8, 245, 3, 6, 241, 3, 11, 37, 4, 0, 8, 245, 3, 11, 46, 4, 0, 9, 221, 3, 16, 12, 15, 40, 0, 2, 47, 0, 2, 47, 0, 19, 2, 0, 15, 6, 0, 6, 24, 4, 7, 20, 4, 11, 46, 4, 0, 5, 1, 9, 12, 4, 16, 6, 35, 4, 11, 46, 4, 0, 8, 35, 4, 16, 12, 17, 22, 0, 20, 41, 0, 1, 0, 12, 17, 23, 0, 20, 42, 0, 1, 0, 12, 15, 5, 0, 7, 63, 4, 1, 5, 16, 12, 15, 22, 0, 16, 12, 15, 26, 0, 16, 12, 15, 31, 0, 16, 12, 15, 28, 0, 16, 12, 15, 29, 0, 16, 12, 15, 19, 0, 6, 108, 4, 7, 104, 4, 11, 145, 1, 0, 5, 1, 9, 96, 4, 16, 12, 15, 7, 0, 6, 146, 4, 7, 142, 4, 6, 137, 4, 2, 102, 0, 2, 114, 0, 2, 111, 0, 2, 109, 0, 8, 141, 4, 11, 63, 2, 0, 5, 1, 9, 116, 4, 16, 12, 15, 9, 0, 6, 166, 4, 7, 162, 4, 11, 63, 2, 0, 5, 1, 9, 154, 4, 16, 12, 15, 10, 0, 18, 24, 0, 16, 12, 
	},
	strs: []string{
		"Grammar", "Spacing", "Import", "Definition", "eof", "EOF", "", "MissingImportName", "Identifier", "MissingImportFrom", "MissingImportSrc", "Literal", "LEFTARROW", "Expression", "Sequence", "SLASH", "Prefix", "Labeled", "Suffix", "MissingLabel", "Primary", "Superscript", "MissingClosingParen", "List", "Class", "Any", "MissingClosingCurly", "Char", "MissingClosingSQuote", "MissingClosingDQuote", "Range", "MissingClosingBracket", "MissingRightRange", "Escape", "Unicode", "chrH1", "Hex", "chrH2", "chrH3", "chrH4", "Comment", "Space", "EOL", 
	},
	rxps: map[int]int{
		7: 1134, 9: 1172, 10: 1192, 19: 1114, 22: 1089, 26: 1094, 28: 1104, 29: 1109, 31: 1099, 
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
	vm            *virtualMachine
	captureSpaces bool
}
func NewGrammarParserBootstrap() *GrammarParserBootstrap {
	spcAddr := bytecodeForGrammarParserBootstrap.smap["Spacing"]
	supprset := make(map[int]struct{})
	supprset[spcAddr] = struct{}{}
	vm := NewVirtualMachine(bytecodeForGrammarParserBootstrap, map[string]string{}, supprset, true)
	return &GrammarParserBootstrap{vm: vm}
}
func (p *GrammarParserBootstrap) ParseGrammar() (Value, error) { return p.parseFn(5) }
func (p *GrammarParserBootstrap) ParseImport() (Value, error) { return p.parseFn(55) }
func (p *GrammarParserBootstrap) ParseDefinition() (Value, error) { return p.parseFn(178) }
func (p *GrammarParserBootstrap) ParseExpression() (Value, error) { return p.parseFn(207) }
func (p *GrammarParserBootstrap) ParseSequence() (Value, error) { return p.parseFn(246) }
func (p *GrammarParserBootstrap) ParsePrefix() (Value, error) { return p.parseFn(261) }
func (p *GrammarParserBootstrap) ParseLabeled() (Value, error) { return p.parseFn(290) }
func (p *GrammarParserBootstrap) ParseSuffix() (Value, error) { return p.parseFn(338) }
func (p *GrammarParserBootstrap) ParsePrimary() (Value, error) { return p.parseFn(401) }
func (p *GrammarParserBootstrap) ParseList() (Value, error) { return p.parseFn(501) }
func (p *GrammarParserBootstrap) ParseIdentifier() (Value, error) { return p.parseFn(564) }
func (p *GrammarParserBootstrap) ParseLiteral() (Value, error) { return p.parseFn(575) }
func (p *GrammarParserBootstrap) ParseClass() (Value, error) { return p.parseFn(662) }
func (p *GrammarParserBootstrap) ParseRange() (Value, error) { return p.parseFn(709) }
func (p *GrammarParserBootstrap) ParseChar() (Value, error) { return p.parseFn(747) }
func (p *GrammarParserBootstrap) ParseEscape() (Value, error) { return p.parseFn(777) }
func (p *GrammarParserBootstrap) ParseUnicode() (Value, error) { return p.parseFn(789) }
func (p *GrammarParserBootstrap) ParseHex() (Value, error) { return p.parseFn(858) }
func (p *GrammarParserBootstrap) ParseAny() (Value, error) { return p.parseFn(867) }
func (p *GrammarParserBootstrap) ParseSuperscript() (Value, error) { return p.parseFn(876) }
func (p *GrammarParserBootstrap) ParseLEFTARROW() (Value, error) { return p.parseFn(956) }
func (p *GrammarParserBootstrap) ParseSLASH() (Value, error) { return p.parseFn(968) }
func (p *GrammarParserBootstrap) ParseSpacing() (Value, error) { return p.parseFn(983) }
func (p *GrammarParserBootstrap) ParseComment() (Value, error) { return p.parseFn(1018) }
func (p *GrammarParserBootstrap) ParseSpace() (Value, error) { return p.parseFn(1061) }
func (p *GrammarParserBootstrap) ParseEOL() (Value, error) { return p.parseFn(1070) }
func (p *GrammarParserBootstrap) ParseEOF() (Value, error) { return p.parseFn(1079) }
func (p *GrammarParserBootstrap) ParseMissingClosingParen() (Value, error) { return p.parseFn(1089) }
func (p *GrammarParserBootstrap) ParseMissingClosingCurly() (Value, error) { return p.parseFn(1094) }
func (p *GrammarParserBootstrap) ParseMissingClosingBracket() (Value, error) { return p.parseFn(1099) }
func (p *GrammarParserBootstrap) ParseMissingClosingSQuote() (Value, error) { return p.parseFn(1104) }
func (p *GrammarParserBootstrap) ParseMissingClosingDQuote() (Value, error) { return p.parseFn(1109) }
func (p *GrammarParserBootstrap) ParseMissingLabel() (Value, error) { return p.parseFn(1114) }
func (p *GrammarParserBootstrap) ParseMissingImportName() (Value, error) { return p.parseFn(1134) }
func (p *GrammarParserBootstrap) ParseMissingImportFrom() (Value, error) { return p.parseFn(1172) }
func (p *GrammarParserBootstrap) ParseMissingImportSrc() (Value, error) { return p.parseFn(1192) }
func (p *GrammarParserBootstrap) Parse() (Value, error)                 { return p.parseFn(5) }
func (p *GrammarParserBootstrap) SetInput(input string)                 { p.input = input }
func (p *GrammarParserBootstrap) SetLabelMessages(el map[string]string) { p.vm.errLabels = el }
func (p *GrammarParserBootstrap) SetShowFails(v bool)                   { p.vm.showFails = v }
func (p *GrammarParserBootstrap) SetCaptureSpaces(v bool)               { p.captureSpaces = v }
func (p *GrammarParserBootstrap) parseFn(addr int) (Value, error) {
	input := NewMemInput(p.input)
	val, _, err := p.vm.MatchRule(&input, addr)
	return val, err
}
