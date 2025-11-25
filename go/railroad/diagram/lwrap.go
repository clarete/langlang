package diagram

func lineWrap(d diagram, maxWidth float64) (layout, error) {
	vi := &lineWrapVisitor{maxWidth: maxWidth}
	if err := d.Accept(vi); err != nil {
		return nil, err
	}
	output := vi.pop()
	return output, nil
}

type lineWrapVisitor struct {
	stack    []layout
	maxWidth float64
}

func (vi *lineWrapVisitor) AcceptTerm(v *term) error {
	vi.stack = append(vi.stack, newStation(ltr, v.label, true))
	return nil
}

func (vi *lineWrapVisitor) AcceptNonTerm(v *nonterm) error {
	vi.stack = append(vi.stack, newStation(ltr, v.label, false))
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

	// Check if wrapping is needed
	totalWidth := vi.calculateTotalWidth(items)
	if totalWidth <= vi.maxWidth || vi.maxWidth == 0 {
		// Fits on one line
		vi.push(newHConcat(ltr, items))
		return nil
	}

	// Need to wrap - use greedy line breaking
	lines := vi.greedyBreak(items)
	if len(lines) == 1 {
		// Only one line after all
		vi.push(newHConcat(ltr, lines[0]))
		return nil
	}

	// Multiple lines - wrap them vertically
	var wrappedLines []layout
	for _, line := range lines {
		wrappedLines = append(wrappedLines, newHConcat(ltr, line))
	}

	// Use vconcat-block to stack the lines
	vi.push(newVConcatBlock(ltr, &verticalTip{}, &verticalTip{}, pol_plus, wrappedLines[0], vi.mergeLines(wrappedLines[1:])))
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
		// Choice: use vconcat-inline
		vi.push(newVConcatInline(ltr, &verticalTip{}, &verticalTip{}, "choice", []layout{topLayout, bottomLayout}))
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

// calculateTotalWidth sums the widths of layout items, including spacing
func (vi *lineWrapVisitor) calculateTotalWidth(items []layout) float64 {
	if len(items) == 0 {
		return 0
	}
	total := 0.0
	for i, item := range items {
		total += item.getWidth()
		// Add railGap spacing between elements (but not after last)
		if i < len(items)-1 {
			total += railGap
		}
	}
	return total
}

// greedyBreak splits items into lines using greedy algorithm
// Fills each line as much as possible before breaking
func (vi *lineWrapVisitor) greedyBreak(items []layout) [][]layout {
	if len(items) == 0 {
		return nil
	}

	var lines [][]layout
	var currentLine []layout
	currentWidth := 0.0

	for _, item := range items {
		itemWidth := item.getWidth()

		// Calculate width if we add this item to current line
		widthWithItem := currentWidth
		if len(currentLine) > 0 {
			widthWithItem += railGap // spacing before item
		}
		widthWithItem += itemWidth

		// Check if item fits on current line
		if len(currentLine) > 0 && widthWithItem > vi.maxWidth {
			// Start new line
			lines = append(lines, currentLine)
			currentLine = []layout{item}
			currentWidth = itemWidth
		} else {
			// Add to current line
			currentLine = append(currentLine, item)
			currentWidth = widthWithItem
		}
	}

	// Add last line
	if len(currentLine) > 0 {
		lines = append(lines, currentLine)
	}

	return lines
}

// mergeLines combines multiple wrapped lines into a single layout tree
func (vi *lineWrapVisitor) mergeLines(lines []layout) layout {
	if len(lines) == 0 {
		return newRail(ltr, 0)
	}
	if len(lines) == 1 {
		return lines[0]
	}
	// Recursively build vconcat-block tree
	return newVConcatBlock(ltr, &verticalTip{}, &verticalTip{}, pol_plus, lines[0], vi.mergeLines(lines[1:]))
}
