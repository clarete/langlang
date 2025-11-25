package diagram

import (
	"fmt"
	"strings"
)

type layout interface {
	Accept(v LayoutVisitor) error
	String() string
	getWidth() float64
	setWidth(w float64)
}

type LayoutVisitor interface {
	AcceptRail(v *rail) error
	AcceptSpace(v *space) error
	AcceptStation(v *station) error
	AcceptHConcat(v *hconcat) error
	AcceptVConcatInline(v *vconcatInline) error
	AcceptVConcatBlock(v *vconcatBlock) error
}

type direction int

const (
	ltr direction = iota
	rtl
)

func (d direction) String() string {
	if d == ltr {
		return "ltr"
	}
	return "rtl"
}

type rail struct {
	dir   direction
	width float64
}

func newRail(dir direction, width float64) *rail { return &rail{dir: dir, width: width} }
func (v *rail) String() string                   { return fmt.Sprintf("(rail %s %.0f)", v.dir, v.width) }
func (v *rail) Accept(vi LayoutVisitor) error    { return vi.AcceptRail(v) }
func (v *rail) getWidth() float64                { return v.width }
func (v *rail) setWidth(w float64)               { v.width = w }

type space struct {
	dir   direction
	width float64
}

func newSpace(dir direction) *space            { return &space{dir: dir, width: railGap} }
func (v *space) String() string                { return fmt.Sprintf("(space %s)", v.dir) }
func (v *space) Accept(vi LayoutVisitor) error { return vi.AcceptSpace(v) }
func (v *space) getWidth() float64             { return v.width }
func (v *space) setWidth(w float64)            { v.width = w }

type station struct {
	dir        direction
	label      string
	isTerminal bool
	baseline   float64 // Baseline from Phase 1 metrics
	width      float64
	height     float64
}

func newStation(d direction, lb string, tm bool) *station {
	textWidth, textHeight := mkFonteSize(lb)
	return &station{
		dir:        d,
		label:      lb,
		isTerminal: tm,
		baseline:   0,
		width:      textWidth + 2*hpadding,
		height:     textHeight + 2*vpadding,
	}
}

func newStationWithBaseline(d direction, lb string, tm bool, baseline float64) *station {
	textWidth, textHeight := mkFonteSize(lb)
	return &station{
		dir:        d,
		label:      lb,
		isTerminal: tm,
		baseline:   baseline,
		width:      textWidth + 2*hpadding,
		height:     textHeight + 2*vpadding,
	}
}
func (v *station) Accept(vi LayoutVisitor) error { return vi.AcceptStation(v) }
func (v *station) getWidth() float64             { return v.width }
func (v *station) setWidth(w float64)            { v.width = w }
func (v *station) String() string {
	termFlag := "false"
	if v.isTerminal {
		termFlag = "true"
	}
	var labelStr string
	if v.isTerminal {
		labelStr = fmt.Sprintf(`"%s"`, v.label)
	} else {
		labelStr = fmt.Sprintf(`[%s]`, v.label)
	}
	return fmt.Sprintf("(station %s %s %s)", v.dir, labelStr, termFlag)
}

type hconcat struct {
	dir      direction
	children []layout
	width    float64
}

func newHConcat(dir direction, children []layout) *hconcat {
	// Calculate total width
	totalWidth := 0.0
	for i, child := range children {
		totalWidth += child.getWidth()
		if i < len(children)-1 {
			totalWidth += railGap
		}
	}
	return &hconcat{dir: dir, children: children, width: totalWidth}
}
func (v *hconcat) Accept(vi LayoutVisitor) error { return vi.AcceptHConcat(v) }
func (v *hconcat) getWidth() float64             { return v.width }
func (v *hconcat) setWidth(w float64)            { v.width = w }
func (v *hconcat) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("(hconcat %s", v.dir))
	for _, child := range v.children {
		sb.WriteString(" ")
		sb.WriteString(child.String())
	}
	sb.WriteString(")")
	return sb.String()
}

type vconcatInline struct {
	dir      direction
	entryTip tip
	exitTip  tip
	marker   string
	branches []layout
	baseline float64 // Baseline from Phase 1 metrics
	width    float64
}

func newVConcatInline(dir direction, entryTip tip, exitTip tip, marker string, branches []layout) *vconcatInline {
	// Width is the maximum of all branches
	maxWidth := 0.0
	for _, branch := range branches {
		if w := branch.getWidth(); w > maxWidth {
			maxWidth = w
		}
	}
	return &vconcatInline{dir: dir, entryTip: entryTip, exitTip: exitTip, marker: marker, branches: branches, width: maxWidth}
}

func newVConcatInlineWithBaseline(dir direction, entryTip tip, exitTip tip, marker string, branches []layout, baseline float64) *vconcatInline {
	// Width is the maximum of all branches
	maxWidth := 0.0
	for _, branch := range branches {
		if w := branch.getWidth(); w > maxWidth {
			maxWidth = w
		}
	}
	return &vconcatInline{dir: dir, entryTip: entryTip, exitTip: exitTip, marker: marker, branches: branches, baseline: baseline, width: maxWidth}
}

func (v *vconcatInline) Accept(vi LayoutVisitor) error { return vi.AcceptVConcatInline(v) }
func (v *vconcatInline) getWidth() float64             { return v.width }
func (v *vconcatInline) setWidth(w float64)            { v.width = w }
func (v *vconcatInline) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`(vconcat-inline %s %s %s "%s"`,
		v.dir,
		v.entryTip,
		v.exitTip,
		v.marker,
	))

	for _, branch := range v.branches {
		sb.WriteString(" ")
		sb.WriteString(branch.String())
	}

	sb.WriteString(")")

	return sb.String()

}

type vconcatBlock struct {
	dir      direction
	entryTip tip
	exitTip  tip
	polarity polarity
	mainPath layout
	loopPath layout
	width    float64
}

func newVConcatBlock(dir direction, entryTip tip, exitTip tip, polarity polarity, mainPath layout, loopPath layout) *vconcatBlock {
	// Width is mainPath + loopPath + spacing
	width := mainPath.getWidth() + railGap + loopPath.getWidth()
	return &vconcatBlock{dir: dir, entryTip: entryTip, exitTip: exitTip, polarity: polarity, mainPath: mainPath, loopPath: loopPath, width: width}
}
func (v *vconcatBlock) Accept(vi LayoutVisitor) error { return vi.AcceptVConcatBlock(v) }
func (v *vconcatBlock) getWidth() float64             { return v.width }
func (v *vconcatBlock) setWidth(w float64)            { v.width = w }
func (v *vconcatBlock) String() string {
	return fmt.Sprintf("(vconcat-block %s %s %s %s %s %s)",
		v.dir,
		v.entryTip,
		v.exitTip,
		v.polarity,
		v.mainPath,
		v.loopPath,
	)
}

type tip interface {
	Accept(tipVisitor) error
}

type tipVisitor interface {
	AcceptVerticalTip(v *verticalTip) error
	AcceptLogicalTip(v *logicalTip) error
	AcceptPhysicalTip(v *physicalTip) error
}

type verticalTip struct{}

func (v *verticalTip) Accept(vi tipVisitor) error { return vi.AcceptVerticalTip(v) }
func (v *verticalTip) String() string             { return "vertical" }

type logicalTip struct{ rowNumber int }

func (v *logicalTip) Accept(vi tipVisitor) error { return vi.AcceptLogicalTip(v) }
func (v *logicalTip) String() string             { return fmt.Sprintf("(logical %d)", v.rowNumber) }

type physicalTip struct{ proportion float64 }

func (v *physicalTip) Accept(vi tipVisitor) error { return vi.AcceptPhysicalTip(v) }
func (v *physicalTip) String() string             { return fmt.Sprintf("(physical %.0f)", v.proportion) }
