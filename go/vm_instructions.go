package langlang

type Instruction interface {
	// Name returns the name of the instruction
	Name() string

	// SizeInBytes returns the size of the instruction in bytes
	SizeInBytes() int
}

// opAnySizeInBytes defines how many bytes the `Any` operator needs
var opAnySizeInBytes = 1

type IAny struct{}

func (IAny) Name() string {
	return "any"
}

func (IAny) SizeInBytes() int {
	return opAnySizeInBytes
}

var opCharSizeInBytes = 3

type IChar struct{ Char rune }

func (IChar) Name() string {
	return "char"
}

func (IChar) SizeInBytes() int {
	return opCharSizeInBytes
}

var opSpanSizeInBytes = 5

type ISpan struct{ Hi, Lo rune }

func (ISpan) Name() string {
	return "span"
}

func (ISpan) SizeInBytes() int {
	return opSpanSizeInBytes
}

type ILabel struct{ ID int }

func (ILabel) Name() string {
	return "label"
}

// SizeInBytes returns zero for the ILabel instruction because it
// doesn't really get written into the output bytecode.
func (ILabel) SizeInBytes() int {
	return 0
}

// globalUniqueID is a global counter used for generating unique label
// IDs.  See the function NewILabel().
var globalUniqueID int

// NewILabel creates a new `ILabel` instruction with a unique ID
func NewILabel() ILabel {
	globalUniqueID++
	return ILabel{ID: globalUniqueID}
}

var opChoiceSizeInBytes = 3

type IChoice struct{ Label ILabel }

func (IChoice) Name() string {
	return "choice"
}

func (IChoice) SizeInBytes() int {
	return opChoiceSizeInBytes
}

type IChoicePred struct{ Label ILabel }

func (IChoicePred) Name() string {
	return "choice_pred"
}

func (IChoicePred) SizeInBytes() int {
	return opChoiceSizeInBytes
}

var opCommitSizeInBytes = 3

type ICommit struct{ Label ILabel }

func (ICommit) Name() string {
	return "commit"
}

func (ICommit) SizeInBytes() int {
	return opCommitSizeInBytes
}

type IPartialCommit struct{ Label ILabel }

func (IPartialCommit) Name() string {
	return "partial_commit"
}

func (IPartialCommit) SizeInBytes() int {
	return opCommitSizeInBytes
}

type IBackCommit struct{ Label ILabel }

func (IBackCommit) Name() string {
	return "back_commit"
}

func (IBackCommit) SizeInBytes() int {
	return opCommitSizeInBytes
}

var opFailSizeInBytes = 1

type IFail struct{}

func (IFail) Name() string {
	return "fail"
}

func (IFail) SizeInBytes() int {
	return opFailSizeInBytes
}

type IFailTwice struct{}

func (IFailTwice) Name() string {
	return "failt_wice"
}

func (IFailTwice) SizeInBytes() int {
	return 1
}

type IJump struct{ Label ILabel }

func (IJump) Name() string {
	return "jump"
}

func (IJump) SizeInBytes() int {
	return 3
}

// opCallSizeInBytes contains the following bytes
//  1. operator
//  2. low nib of 16bit uint label address
//  3. high nib of 16bit uint label address
//  4. uint8 precedence level
var opCallSizeInBytes = 4

type ICall struct {
	Label      ILabel
	Precedence int
}

func (ICall) Name() string {
	return "call"
}

func (ICall) SizeInBytes() int {
	return opCallSizeInBytes
}

// opReturnSizeInBytes contains just one byte for the operator
var opReturnSizeInBytes = 1

type IReturn struct{}

func (IReturn) Name() string {
	return "return"
}

func (IReturn) SizeInBytes() int {
	return opReturnSizeInBytes
}

var opThrowSizeInBytes = 2

type IThrow struct{ ErrorLabel int }

func (IThrow) Name() string {
	return "throw"
}

func (IThrow) SizeInBytes() int {
	return opThrowSizeInBytes
}

var opHaltSizeInBytes = 1

type IHalt struct{}

func (IHalt) Name() string {
	return "halt"
}

func (IHalt) SizeInBytes() int {
	return opHaltSizeInBytes
}

var opCapBeginSizeInBytes = 3

type ICapBegin struct{ ID int }

func (ICapBegin) Name() string {
	return "cap_begin"
}

func (ICapBegin) SizeInBytes() int {
	return opCapBeginSizeInBytes
}

var opCapEndSizeInBytes = 1

type ICapEnd struct{}

func (ICapEnd) Name() string {
	return "cap_end"
}

func (ICapEnd) SizeInBytes() int {
	return opCapEndSizeInBytes
}
