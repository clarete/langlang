package langlang

type Instruction interface {
	// Name returns the name of the instruction
	Name() string

	// SizeInBytes returns the size of the instruction in bytes
	SizeInBytes() int

	// SourceLocation returns the source location of the instruction
	SourceLocation() SourceLocation
}

type IAny struct {
	sl SourceLocation
}

func (IAny) Name() string                     { return "any" }
func (IAny) SizeInBytes() int                 { return opAnySizeInBytes }
func (i IAny) SourceLocation() SourceLocation { return i.sl }

type IChar struct {
	Char rune
	sl   SourceLocation
}

func (i IChar) Name() string {
	if fitsU16Rune(i.Char) {
		return "char"
	}
	return "char32"
}
func (i IChar) SizeInBytes() int {
	if fitsU16Rune(i.Char) {
		return opCharSizeInBytes
	}
	return opChar32SizeInBytes
}
func (i IChar) SourceLocation() SourceLocation { return i.sl }

type IRange struct {
	Hi, Lo rune
	sl     SourceLocation
}

func (i IRange) Name() string {
	if fitsU16Rune(i.Lo) && fitsU16Rune(i.Hi) {
		return "range"
	}
	return "range32"
}
func (i IRange) SizeInBytes() int {
	if fitsU16Rune(i.Lo) && fitsU16Rune(i.Hi) {
		return opRangeSizeInBytes
	}
	return opRange32SizeInBytes
}
func (i IRange) SourceLocation() SourceLocation { return i.sl }

type ISet struct {
	cs *charset
	sl SourceLocation
}

func (ISet) Name() string                     { return "set" }
func (ISet) SizeInBytes() int                 { return opSetSizeInBytes }
func (i ISet) SourceLocation() SourceLocation { return i.sl }

type ISpan struct {
	cs *charset
	sl SourceLocation
}

func (ISpan) Name() string                     { return "span" }
func (ISpan) SizeInBytes() int                 { return opSpanSizeInBytes }
func (i ISpan) SourceLocation() SourceLocation { return i.sl }

type ILabel struct {
	ID int
	sl SourceLocation
}

// SizeInBytes returns zero for the ILabel instruction because it
// doesn't really get written into the output bytecode.
func (ILabel) SizeInBytes() int                 { return 0 }
func (ILabel) Name() string                     { return "label" }
func (i ILabel) SourceLocation() SourceLocation { return i.sl }

// globalUniqueID is a global counter used for generating unique label
// IDs.  See the function NewILabel().
var globalUniqueID int

// NewILabel creates a new `ILabel` instruction with a unique ID
func NewILabel() ILabel {
	globalUniqueID++
	return ILabel{ID: globalUniqueID}
}

func NewILabelWithSourceLocation(sl SourceLocation) ILabel {
	lb := NewILabel()
	lb.sl = sl
	return lb
}

type IChoice struct {
	Label ILabel
	sl    SourceLocation
}

func (IChoice) Name() string                     { return "choice" }
func (IChoice) SizeInBytes() int                 { return opChoiceSizeInBytes }
func (i IChoice) SourceLocation() SourceLocation { return i.sl }

type IChoicePred struct {
	Label ILabel
	sl    SourceLocation
}

func (IChoicePred) Name() string                     { return "choice_pred" }
func (IChoicePred) SizeInBytes() int                 { return opChoiceSizeInBytes }
func (i IChoicePred) SourceLocation() SourceLocation { return i.sl }

type ICommit struct {
	Label ILabel
	sl    SourceLocation
}

func (ICommit) Name() string                     { return "commit" }
func (ICommit) SizeInBytes() int                 { return opCommitSizeInBytes }
func (i ICommit) SourceLocation() SourceLocation { return i.sl }

type IBackCommit struct {
	Label ILabel
	sl    SourceLocation
}

func (IBackCommit) Name() string                     { return "back_commit" }
func (IBackCommit) SizeInBytes() int                 { return opCommitSizeInBytes }
func (i IBackCommit) SourceLocation() SourceLocation { return i.sl }

type IPartialCommit struct {
	Label ILabel
	sl    SourceLocation
}

func (IPartialCommit) Name() string                     { return "partial_commit" }
func (IPartialCommit) SizeInBytes() int                 { return opCommitSizeInBytes }
func (i IPartialCommit) SourceLocation() SourceLocation { return i.sl }

type ICapPartialCommit struct {
	Label ILabel
	sl    SourceLocation
}

func (ICapPartialCommit) Name() string                     { return "cap_partial_commit" }
func (ICapPartialCommit) SizeInBytes() int                 { return opCommitSizeInBytes }
func (i ICapPartialCommit) SourceLocation() SourceLocation { return i.sl }

type ICapCommit struct {
	Label ILabel
	sl    SourceLocation
}

func (ICapCommit) Name() string                     { return "cap_commit" }
func (ICapCommit) SizeInBytes() int                 { return opCommitSizeInBytes }
func (i ICapCommit) SourceLocation() SourceLocation { return i.sl }

type ICapBackCommit struct {
	Label ILabel
	sl    SourceLocation
}

func (ICapBackCommit) Name() string                     { return "cap_back_commit" }
func (ICapBackCommit) SizeInBytes() int                 { return opCommitSizeInBytes }
func (i ICapBackCommit) SourceLocation() SourceLocation { return i.sl }

type IFail struct{ sl SourceLocation }

func (IFail) Name() string                     { return "fail" }
func (IFail) SizeInBytes() int                 { return opFailSizeInBytes }
func (i IFail) SourceLocation() SourceLocation { return i.sl }

type IFailTwice struct{ sl SourceLocation }

func (IFailTwice) Name() string                     { return "fail_twice" }
func (IFailTwice) SizeInBytes() int                 { return opFailSizeInBytes }
func (i IFailTwice) SourceLocation() SourceLocation { return i.sl }

type IJump struct {
	Label ILabel
	sl    SourceLocation
}

func (IJump) Name() string                     { return "jump" }
func (IJump) SizeInBytes() int                 { return opJumpSizeInBytes }
func (i IJump) SourceLocation() SourceLocation { return i.sl }

// ICall is a regular (non-left-recursive) function call
type ICall struct {
	Label ILabel
	sl    SourceLocation
}

func (ICall) Name() string                     { return "call" }
func (ICall) SizeInBytes() int                 { return opCallSizeInBytes }
func (i ICall) SourceLocation() SourceLocation { return i.sl }

// ICallLR is a left-recursive call with precedence handling
type ICallLR struct {
	Label      ILabel
	Precedence int
	sl         SourceLocation
}

func (ICallLR) Name() string                     { return "call_lr" }
func (ICallLR) SizeInBytes() int                 { return opCallLRSizeInBytes }
func (i ICallLR) SourceLocation() SourceLocation { return i.sl }

// IReturn is a regular (non-left-recursive) return
type IReturn struct{ sl SourceLocation }

func (IReturn) Name() string                     { return "return" }
func (IReturn) SizeInBytes() int                 { return opReturnSizeInBytes }
func (i IReturn) SourceLocation() SourceLocation { return i.sl }

// IReturnLR is a left-recursive return with memo table handling
type IReturnLR struct{ sl SourceLocation }

func (IReturnLR) Name() string                     { return "return_lr" }
func (IReturnLR) SizeInBytes() int                 { return opReturnLRSizeInBytes }
func (i IReturnLR) SourceLocation() SourceLocation { return i.sl }

// ICapReturn is a capture-aware return (non-left-recursive)
type ICapReturn struct{ sl SourceLocation }

func (ICapReturn) Name() string                     { return "cap_return" }
func (ICapReturn) SizeInBytes() int                 { return opReturnSizeInBytes }
func (i ICapReturn) SourceLocation() SourceLocation { return i.sl }

// ICapReturnLR is a capture-aware left-recursive return
type ICapReturnLR struct{ sl SourceLocation }

func (ICapReturnLR) Name() string                     { return "cap_return_lr" }
func (ICapReturnLR) SizeInBytes() int                 { return opCapReturnLRSizeInBytes }
func (i ICapReturnLR) SourceLocation() SourceLocation { return i.sl }

type IThrow struct {
	ErrorLabel int
	sl         SourceLocation
}

func (IThrow) Name() string                     { return "throw" }
func (IThrow) SizeInBytes() int                 { return opThrowSizeInBytes }
func (i IThrow) SourceLocation() SourceLocation { return i.sl }

type IHalt struct{ sl SourceLocation }

func (IHalt) Name() string                     { return "halt" }
func (IHalt) SizeInBytes() int                 { return opHaltSizeInBytes }
func (i IHalt) SourceLocation() SourceLocation { return i.sl }

type ICapBegin struct {
	ID int
	sl SourceLocation
}

func (ICapBegin) Name() string                     { return "cap_begin" }
func (ICapBegin) SizeInBytes() int                 { return opCapBeginSizeInBytes }
func (i ICapBegin) SourceLocation() SourceLocation { return i.sl }

type ICapEnd struct{ sl SourceLocation }

func (ICapEnd) Name() string                     { return "cap_end" }
func (ICapEnd) SizeInBytes() int                 { return opCapEndSizeInBytes }
func (i ICapEnd) SourceLocation() SourceLocation { return i.sl }

type ICapTerm struct {
	Offset int
	sl     SourceLocation
}

func (ICapTerm) Name() string                     { return "cap_term" }
func (ICapTerm) SizeInBytes() int                 { return opCapTermSizeInBytes }
func (i ICapTerm) SourceLocation() SourceLocation { return i.sl }

type ICapNonTerm struct {
	ID     int
	Offset int
	sl     SourceLocation
}

func (ICapNonTerm) Name() string                     { return "cap_non_term" }
func (ICapNonTerm) SizeInBytes() int                 { return opCapNonTermSizeInBytes }
func (i ICapNonTerm) SourceLocation() SourceLocation { return i.sl }

type ICapTermBeginOffset struct{ sl SourceLocation }

func (ICapTermBeginOffset) Name() string                     { return "cap_term_begin_offset" }
func (ICapTermBeginOffset) SizeInBytes() int                 { return opCapTermBeginOffsetSizeInBytes }
func (i ICapTermBeginOffset) SourceLocation() SourceLocation { return i.sl }

type ICapNonTermBeginOffset struct {
	ID int
	sl SourceLocation
}

func (ICapNonTermBeginOffset) Name() string                     { return "cap_non_term_begin_offset" }
func (ICapNonTermBeginOffset) SizeInBytes() int                 { return opCapNonTermBeginOffsetSizeInBytes }
func (i ICapNonTermBeginOffset) SourceLocation() SourceLocation { return i.sl }

type ICapEndOffset struct{ sl SourceLocation }

func (ICapEndOffset) Name() string                     { return "cap_end_offset" }
func (ICapEndOffset) SizeInBytes() int                 { return opCapEndOffsetSizeInBytes }
func (i ICapEndOffset) SourceLocation() SourceLocation { return i.sl }
