package langlang

var bytecodeForGrammarParserBootstrap = &Bytecode{
	code: []byte{
		11, 5, 0, 0, 0, 15, 0, 0, 11, 211, 3, 0, 6, 22, 0, 11, 55, 0, 0, 9, 15, 0, 11, 211, 3, 0, 6, 36, 0, 11, 178, 0, 0, 9, 29, 0, 11, 211, 3, 0, 6, 50, 0, 11, 49, 4, 0, 8, 53, 0, 14, 4, 0, 16, 12, 15, 2, 0, 11, 211, 3, 0, 2, 64, 0, 2, 105, 0, 2, 109, 0, 2, 112, 0, 2, 111, 0, 2, 114, 0, 2, 116, 0, 19, 7, 0, 11, 211, 3, 0, 6, 100, 0, 11, 52, 2, 0, 8, 103, 0, 14, 6, 0, 11, 211, 3, 0, 6, 131, 0, 11, 211, 3, 0, 17, 0, 0, 19, 1, 0, 11, 211, 3, 0, 11, 52, 2, 0, 9, 110, 0, 11, 211, 3, 0, 6, 156, 0, 2, 102, 0, 2, 114, 0, 2, 111, 0, 2, 109, 0, 19, 4, 0, 8, 159, 0, 14, 8, 0, 11, 211, 3, 0, 6, 173, 0, 11, 63, 2, 0, 8, 176, 0, 14, 9, 0, 16, 12, 15, 3, 0, 11, 211, 3, 0, 11, 52, 2, 0, 11, 211, 3, 0, 11, 185, 3, 0, 11, 211, 3, 0, 11, 207, 0, 0, 16, 12, 15, 12, 0, 11, 211, 3, 0, 11, 246, 0, 0, 11, 211, 3, 0, 6, 244, 0, 11, 211, 3, 0, 11, 196, 3, 0, 11, 211, 3, 0, 11, 246, 0, 0, 9, 225, 0, 16, 12, 15, 13, 0, 6, 3, 1, 11, 5, 1, 0, 9, 252, 0, 16, 12, 15, 15, 0, 11, 211, 3, 0, 6, 24, 1, 17, 1, 0, 19, 1, 0, 8, 24, 1, 11, 211, 3, 0, 11, 34, 1, 0, 16, 12, 15, 16, 0, 11, 211, 3, 0, 11, 82, 1, 0, 6, 80, 1, 15, 18, 0, 6, 60, 1, 2, 94, 0, 8, 63, 1, 2, 209, 33, 16, 6, 74, 1, 11, 52, 2, 0, 8, 77, 1, 14, 19, 0, 8, 80, 1, 16, 12, 15, 17, 0, 11, 211, 3, 0, 11, 145, 1, 0, 11, 211, 3, 0, 6, 143, 1, 6, 112, 1, 17, 2, 0, 19, 1, 0, 8, 140, 1, 6, 124, 1, 17, 3, 0, 19, 1, 0, 8, 140, 1, 6, 136, 1, 17, 4, 0, 19, 1, 0, 8, 140, 1, 11, 105, 3, 0, 8, 143, 1, 16, 12, 15, 20, 0, 6, 170, 1, 11, 52, 2, 0, 11, 211, 3, 0, 7, 167, 1, 11, 185, 3, 0, 5, 8, 243, 1, 6, 209, 1, 17, 5, 0, 19, 1, 0, 11, 211, 3, 0, 11, 207, 0, 0, 11, 211, 3, 0, 6, 203, 1, 17, 6, 0, 19, 1, 0, 8, 206, 1, 14, 22, 0, 8, 243, 1, 6, 219, 1, 11, 245, 1, 0, 8, 243, 1, 6, 229, 1, 11, 63, 2, 0, 8, 243, 1, 6, 239, 1, 11, 150, 2, 0, 8, 243, 1, 11, 97, 3, 0, 16, 12, 15, 23, 0, 11, 211, 3, 0, 17, 7, 0, 19, 1, 0, 11, 211, 3, 0, 6, 31, 2, 11, 211, 3, 0, 7, 20, 2, 17, 8, 0, 5, 11, 211, 3, 0, 11, 207, 0, 0, 9, 9, 2, 11, 211, 3, 0, 6, 47, 2, 17, 8, 0, 19, 1, 0, 8, 50, 2, 14, 26, 0, 16, 12, 15, 7, 0, 17, 9, 0, 18, 10, 0, 16, 12, 15, 10, 0, 6, 110, 2, 17, 11, 0, 19, 1, 0, 6, 92, 2, 7, 85, 2, 17, 11, 0, 5, 11, 235, 2, 0, 9, 78, 2, 6, 104, 2, 17, 11, 0, 19, 1, 0, 8, 107, 2, 14, 28, 0, 8, 148, 2, 17, 12, 0, 19, 1, 0, 6, 133, 2, 7, 126, 2, 17, 12, 0, 5, 11, 235, 2, 0, 9, 119, 2, 6, 145, 2, 17, 12, 0, 19, 1, 0, 8, 148, 2, 14, 29, 0, 16, 12, 15, 24, 0, 11, 211, 3, 0, 17, 13, 0, 19, 1, 0, 6, 180, 2, 7, 173, 2, 17, 14, 0, 5, 11, 197, 2, 0, 9, 166, 2, 6, 192, 2, 17, 14, 0, 19, 1, 0, 8, 195, 2, 14, 31, 0, 16, 12, 15, 30, 0, 6, 229, 2, 11, 235, 2, 0, 17, 15, 0, 19, 1, 0, 6, 223, 2, 11, 235, 2, 0, 8, 226, 2, 14, 32, 0, 8, 233, 2, 11, 235, 2, 0, 16, 12, 15, 27, 0, 6, 248, 2, 11, 9, 3, 0, 8, 7, 3, 6, 2, 3, 11, 20, 3, 0, 8, 7, 3, 15, 18, 0, 1, 16, 16, 12, 15, 33, 0, 17, 16, 0, 17, 17, 0, 16, 12, 15, 34, 0, 17, 16, 0, 19, 1, 0, 17, 18, 0, 19, 1, 0, 6, 45, 3, 11, 89, 3, 0, 8, 48, 3, 14, 35, 0, 6, 58, 3, 11, 89, 3, 0, 8, 61, 3, 14, 37, 0, 6, 71, 3, 11, 89, 3, 0, 8, 74, 3, 14, 38, 0, 6, 84, 3, 11, 89, 3, 0, 8, 87, 3, 14, 39, 0, 16, 12, 15, 36, 0, 17, 19, 0, 16, 12, 15, 25, 0, 17, 20, 0, 16, 12, 15, 21, 0, 6, 117, 3, 2, 185, 0, 8, 183, 3, 6, 126, 3, 2, 178, 0, 8, 183, 3, 6, 135, 3, 2, 179, 0, 8, 183, 3, 6, 144, 3, 2, 116, 32, 8, 183, 3, 6, 153, 3, 2, 117, 32, 8, 183, 3, 6, 162, 3, 2, 118, 32, 8, 183, 3, 6, 171, 3, 2, 119, 32, 8, 183, 3, 6, 180, 3, 2, 120, 32, 8, 183, 3, 2, 121, 32, 16, 12, 15, 11, 0, 2, 60, 0, 2, 45, 0, 16, 12, 15, 14, 0, 17, 21, 0, 7, 209, 3, 17, 21, 0, 5, 16, 12, 15, 1, 0, 6, 244, 3, 6, 227, 3, 11, 246, 3, 0, 8, 241, 3, 6, 237, 3, 11, 33, 4, 0, 8, 241, 3, 11, 41, 4, 0, 9, 217, 3, 16, 12, 15, 40, 0, 2, 47, 0, 2, 47, 0, 19, 2, 0, 15, 18, 0, 6, 20, 4, 7, 16, 4, 11, 41, 4, 0, 5, 1, 9, 8, 4, 16, 6, 31, 4, 11, 41, 4, 0, 8, 31, 4, 16, 12, 15, 41, 0, 17, 22, 0, 16, 12, 15, 42, 0, 17, 23, 0, 16, 12, 15, 5, 0, 7, 57, 4, 1, 5, 16, 12, 15, 22, 0, 16, 12, 15, 26, 0, 16, 12, 15, 31, 0, 16, 12, 15, 28, 0, 16, 12, 15, 29, 0, 16, 12, 15, 19, 0, 6, 102, 4, 7, 98, 4, 11, 145, 1, 0, 5, 1, 9, 90, 4, 16, 12, 15, 6, 0, 6, 140, 4, 7, 136, 4, 6, 131, 4, 2, 102, 0, 2, 114, 0, 2, 111, 0, 2, 109, 0, 8, 135, 4, 11, 63, 2, 0, 5, 1, 9, 110, 4, 16, 12, 15, 8, 0, 6, 160, 4, 7, 156, 4, 11, 63, 2, 0, 5, 1, 9, 148, 4, 16, 12, 15, 9, 0, 18, 24, 0, 16, 12, 
	},
	strs: []string{
		"Grammar", "Spacing", "Import", "Definition", "eof", "EOF", "MissingImportName", "Identifier", "MissingImportFrom", "MissingImportSrc", "Literal", "LEFTARROW", "Expression", "Sequence", "SLASH", "Prefix", "Labeled", "Suffix", "", "MissingLabel", "Primary", "Superscript", "MissingClosingParen", "List", "Class", "Any", "MissingClosingCurly", "Char", "MissingClosingSQuote", "MissingClosingDQuote", "Range", "MissingClosingBracket", "MissingRightRange", "Escape", "Unicode", "chrH1", "Hex", "chrH2", "chrH3", "chrH4", "Comment", "Space", "EOL", 
	},
	rxps: map[int]int{
		6: 1128, 8: 1166, 9: 1186, 19: 1108, 22: 1083, 26: 1088, 28: 1098, 29: 1103, 31: 1093, 
	},
	smap: map[string]int{
		"": 18, "Any": 25, "Char": 27, "Class": 24, "Comment": 40, "Definition": 3, "EOF": 5, "EOL": 42, "Escape": 33, "Expression": 12, "Grammar": 0, "Hex": 36, "Identifier": 7, "Import": 2, "LEFTARROW": 11, "Labeled": 16, "List": 23, "Literal": 10, "MissingClosingBracket": 31, "MissingClosingCurly": 26, "MissingClosingDQuote": 29, "MissingClosingParen": 22, "MissingClosingSQuote": 28, "MissingImportFrom": 8, "MissingImportName": 6, "MissingImportSrc": 9, "MissingLabel": 19, "MissingRightRange": 32, "Prefix": 15, "Primary": 20, "Range": 30, "SLASH": 14, "Sequence": 13, "Space": 41, "Spacing": 1, "Suffix": 17, "Superscript": 21, "Unicode": 34, "chrH1": 35, "chrH2": 37, "chrH3": 38, "chrH4": 39, "eof": 4, 
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
func (p *GrammarParserBootstrap) ParseUnicode() (Value, error) { return p.parseFn(788) }
func (p *GrammarParserBootstrap) ParseHex() (Value, error) { return p.parseFn(857) }
func (p *GrammarParserBootstrap) ParseAny() (Value, error) { return p.parseFn(865) }
func (p *GrammarParserBootstrap) ParseSuperscript() (Value, error) { return p.parseFn(873) }
func (p *GrammarParserBootstrap) ParseLEFTARROW() (Value, error) { return p.parseFn(953) }
func (p *GrammarParserBootstrap) ParseSLASH() (Value, error) { return p.parseFn(964) }
func (p *GrammarParserBootstrap) ParseSpacing() (Value, error) { return p.parseFn(979) }
func (p *GrammarParserBootstrap) ParseComment() (Value, error) { return p.parseFn(1014) }
func (p *GrammarParserBootstrap) ParseSpace() (Value, error) { return p.parseFn(1057) }
func (p *GrammarParserBootstrap) ParseEOL() (Value, error) { return p.parseFn(1065) }
func (p *GrammarParserBootstrap) ParseEOF() (Value, error) { return p.parseFn(1073) }
func (p *GrammarParserBootstrap) ParseMissingClosingParen() (Value, error) { return p.parseFn(1083) }
func (p *GrammarParserBootstrap) ParseMissingClosingCurly() (Value, error) { return p.parseFn(1088) }
func (p *GrammarParserBootstrap) ParseMissingClosingBracket() (Value, error) { return p.parseFn(1093) }
func (p *GrammarParserBootstrap) ParseMissingClosingSQuote() (Value, error) { return p.parseFn(1098) }
func (p *GrammarParserBootstrap) ParseMissingClosingDQuote() (Value, error) { return p.parseFn(1103) }
func (p *GrammarParserBootstrap) ParseMissingLabel() (Value, error) { return p.parseFn(1108) }
func (p *GrammarParserBootstrap) ParseMissingImportName() (Value, error) { return p.parseFn(1128) }
func (p *GrammarParserBootstrap) ParseMissingImportFrom() (Value, error) { return p.parseFn(1166) }
func (p *GrammarParserBootstrap) ParseMissingImportSrc() (Value, error) { return p.parseFn(1186) }
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
