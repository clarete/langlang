package diagram

const (
	hpadding    = 10.0 // horizontal padding in boxes
	vpadding    = 5.0  // vertical padding in boxes
	railGap     = 10.0 // space between elements in sequence
	branchGap   = 10.0 // vertical space between choice branches
	loopGap     = 15.0 // vertical space for loop-back
	curveHeight = 10.0 // height of connection curves
)

func computeVerticalMetrics(d diagram) error {
	vi := &vmetrics{}
	return d.Accept(vi)
}

type vmetrics struct{}

func (vi *vmetrics) AcceptTerm(v *term) error {
	textWidth, textHeight := mkFonteSize(v.label)
	v.width = textWidth + 2*hpadding
	v.height = textHeight + 2*vpadding
	v.baseline = v.height / 2 // center
	return nil
}

func (vi *vmetrics) AcceptNonTerm(v *nonterm) error {
	textWidth, textHeight := mkFonteSize(v.label)
	v.width = textWidth + 2*hpadding
	v.height = textHeight + 2*vpadding
	v.baseline = v.height / 2 // center
	return nil
}

func (vi *vmetrics) AcceptSeq(v *seq) error {
	v.height = 0
	v.width = 0
	v.baseline = 0

	if len(v.items) == 0 {
		return nil
	}

	var (
		maxBaseline float64
		maxDepth    float64
	)

	for _, item := range v.items {
		if err := item.Accept(vi); err != nil {
			return err
		}
		bl := item.Baseline()
		depth := item.Height() - bl

		if bl > maxBaseline {
			maxBaseline = bl
		}
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	v.height = maxBaseline + maxDepth
	v.baseline = maxBaseline

	for i, item := range v.items {
		v.width += item.Width()
		if i < len(v.items)-1 {
			v.width += railGap
		}
	}
	return nil
}

func (vi *vmetrics) AcceptStack(v *stack) error {
	if err := v.top.Accept(vi); err != nil {
		return err
	}
	if err := v.bottom.Accept(vi); err != nil {
		return err
	}
	if v.pol == pol_plus {
		// compute choice:
		v.height = 2 * curveHeight
		v.height += v.top.Height()
		v.height += v.bottom.Height()
		v.width = max(v.top.Width(), v.bottom.Width())
	} else {
		// compute loop:
		loopBackHeight := v.bottom.Height()
		v.width = max(v.top.Width(), v.bottom.Width())
		v.height = v.top.Height()
		if loopBackHeight > 0 {
			v.height += loopGap
			v.height += 2 * curveHeight
			v.height += loopBackHeight
		}
	}
	// both types above use the same centered baseline
	v.baseline = v.Height() / 2 // center
	return nil
}

func (vi *vmetrics) AcceptEmpty(*empty) error {
	return nil
}

func mkFonteSize(input string) (float64, float64) {
	var (
		// 8px-14px per char (assuming a monospaced font)
		textWidth  = float64(len(input)) * 8.0
		textHeight = 14.0
	)
	return textWidth, textHeight
}
