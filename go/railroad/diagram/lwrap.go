package diagram

func lineWrap(d diagram) (layout, error) {
	vi := &lineWrapVisitor{}
	if err := d.Accept(vi); err != nil {
		return nil, err
	}
	output := vi.pop()
	return output, nil
}

type lineWrapVisitor struct {
	stack []layout
}

func (vi *lineWrapVisitor) AcceptTerm(v *term) error {
	vi.stack = append(vi.stack, newStationWithBaseline(ltr, v.label, true, v.baseline))
	return nil
}

func (vi *lineWrapVisitor) AcceptNonTerm(v *nonterm) error {
	vi.stack = append(vi.stack, newStationWithBaseline(ltr, v.label, false, v.baseline))
	return nil
}

func (vi *lineWrapVisitor) AcceptSeq(v *seq) error {
	var items []layout
	for _, item := range v.items {
		if err := item.Accept(vi); err != nil {
			return err
		}
		items = append(items, vi.pop())
	}
	vi.push(newHConcat(ltr, items))
	return nil
}

func (vi *lineWrapVisitor) AcceptStack(v *stack) error {
	if err := v.top.Accept(vi); err != nil {
		return err
	}
	topLayout := vi.pop()

	if err := v.bottom.Accept(vi); err != nil {
		return err
	}
	bottomLayout := vi.pop()

	if v.pol == pol_plus {
		// Choice: use vconcat-inline with baseline from Phase 1
		vi.push(newVConcatInlineWithBaseline(ltr, &verticalTip{}, &verticalTip{}, "choice", []layout{topLayout, bottomLayout}, v.baseline))
	} else {
		// Loop: use vconcat-block
		vi.push(newVConcatBlock(ltr, &verticalTip{}, &verticalTip{}, v.pol, topLayout, bottomLayout))
	}

	return nil
}

func (vi *lineWrapVisitor) AcceptEmpty(v *empty) error {
	// Empty elements become zero-width rails
	vi.push(newRail(ltr, 0))
	return nil
}

func (vi *lineWrapVisitor) push(l layout) {
	vi.stack = append(vi.stack, l)
}

func (vi *lineWrapVisitor) pop() layout {
	idx := len(vi.stack) - 1
	top := vi.stack[idx]
	vi.stack = vi.stack[:idx]
	return top
}
