package langlang

type Instruction interface {
	// Name returns the name of the instruction
	Name() string

	// SizeInBytes returns the size of the instruction in bytes
	SizeInBytes() int
}

type IAny struct{}

func (IAny) Name() string     { return "any" }
func (IAny) SizeInBytes() int { return opAnySizeInBytes }

type IChar struct{ Char rune }

func (IChar) Name() string     { return "char" }
func (IChar) SizeInBytes() int { return opCharSizeInBytes }

type IRange struct{ Hi, Lo rune }

func (IRange) Name() string     { return "range" }
func (IRange) SizeInBytes() int { return opRangeSizeInBytes }

type ISet struct{ cs *charset }

func (ISet) Name() string     { return "set" }
func (ISet) SizeInBytes() int { return opSetSizeInBytes }

type ISpan struct{ cs *charset }

func (ISpan) Name() string     { return "span" }
func (ISpan) SizeInBytes() int { return opSpanSizeInBytes }

type ILabel struct{ ID int }

// SizeInBytes returns zero for the ILabel instruction because it
// doesn't really get written into the output bytecode.
func (ILabel) SizeInBytes() int { return 0 }
func (ILabel) Name() string     { return "label" }

// globalUniqueID is a global counter used for generating unique label
// IDs.  See the function NewILabel().
var globalUniqueID int

// NewILabel creates a new `ILabel` instruction with a unique ID
func NewILabel() ILabel {
	globalUniqueID++
	return ILabel{ID: globalUniqueID}
}

type IChoice struct{ Label ILabel }

func (IChoice) Name() string     { return "choice" }
func (IChoice) SizeInBytes() int { return opChoiceSizeInBytes }

type IChoicePred struct{ Label ILabel }

func (IChoicePred) Name() string     { return "choice_pred" }
func (IChoicePred) SizeInBytes() int { return opChoiceSizeInBytes }

type ICommit struct{ Label ILabel }

func (ICommit) Name() string     { return "commit" }
func (ICommit) SizeInBytes() int { return opCommitSizeInBytes }

type IPartialCommit struct{ Label ILabel }

func (IPartialCommit) Name() string     { return "partial_commit" }
func (IPartialCommit) SizeInBytes() int { return opCommitSizeInBytes }

type IBackCommit struct{ Label ILabel }

func (IBackCommit) Name() string     { return "back_commit" }
func (IBackCommit) SizeInBytes() int { return opCommitSizeInBytes }

type IFail struct{}

func (IFail) Name() string     { return "fail" }
func (IFail) SizeInBytes() int { return opFailSizeInBytes }

type IFailTwice struct{}

func (IFailTwice) Name() string     { return "fail_twice" }
func (IFailTwice) SizeInBytes() int { return opFailSizeInBytes }

type IJump struct{ Label ILabel }

func (IJump) Name() string     { return "jump" }
func (IJump) SizeInBytes() int { return opJumpSizeInBytes }

type ICall struct {
	Label      ILabel
	Precedence int
}

func (ICall) Name() string     { return "call" }
func (ICall) SizeInBytes() int { return opCallSizeInBytes }

type IReturn struct{}

func (IReturn) Name() string     { return "return" }
func (IReturn) SizeInBytes() int { return opReturnSizeInBytes }

type IThrow struct{ ErrorLabel int }

func (IThrow) Name() string     { return "throw" }
func (IThrow) SizeInBytes() int { return opThrowSizeInBytes }

type IHalt struct{}

func (IHalt) Name() string     { return "halt" }
func (IHalt) SizeInBytes() int { return opHaltSizeInBytes }

type ICapBegin struct{ ID int }

func (ICapBegin) Name() string     { return "cap_begin" }
func (ICapBegin) SizeInBytes() int { return opCapBeginSizeInBytes }

type ICapEnd struct{}

func (ICapEnd) Name() string     { return "cap_end" }
func (ICapEnd) SizeInBytes() int { return opCapEndSizeInBytes }

type ICapTerm struct{ Offset int }

func (ICapTerm) Name() string     { return "cap_term" }
func (ICapTerm) SizeInBytes() int { return opCapTermSizeInBytes }

type ICapNonTerm struct {
	ID     int
	Offset int
}

func (ICapNonTerm) Name() string     { return "cap_non_term" }
func (ICapNonTerm) SizeInBytes() int { return opCapNonTermSizeInBytes }
