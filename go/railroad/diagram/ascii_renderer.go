package diagram

// renderFrame stores the position and dimensions of a rendered element
type renderFrame struct {
	startX, startY int // Where this element started rendering
	width, height  int // Dimensions of the rendered element
}

type asciiRenderer struct {
	canvas *ASCII
	x, y   int           // Current cursor position
	stack  []renderFrame // Stack of rendered dimensions
}

// Stack helper methods
func (r *asciiRenderer) push(frame renderFrame) {
	r.stack = append(r.stack, frame)
}

func (r *asciiRenderer) pop() renderFrame {
	if len(r.stack) == 0 {
		return renderFrame{} // Return empty frame if stack is empty
	}
	idx := len(r.stack) - 1
	frame := r.stack[idx]
	r.stack = r.stack[:idx]
	return frame
}

func (r *asciiRenderer) peek() renderFrame {
	if len(r.stack) == 0 {
		return renderFrame{}
	}
	return r.stack[len(r.stack)-1]
}

func layoutToASCII(l layout, width, height int) string {
	canvas := NewASCII(width, height)

	// Calculate how much space we need above the main track
	// For choices, we need room for the top branch
	topPadding := calculateTopPadding(l)

	// Start with some horizontal offset for entry rail
	entryRailWidth := 0
	renderer := &asciiRenderer{canvas: canvas, x: entryRailWidth, y: topPadding}

	// // Draw entry rail
	// trackY := 1 // Assume tracks are at row 1 (middle of 3-row boxes)
	// canvas.DrawHLine(0, entryRailWidth-1, trackY, '━')

	// Render the layout
	l.Accept(renderer)
	// frame := renderer.peek()

	// // Draw exit rail
	// exitX := entryRailWidth + frame.width
	// exitRailWidth := 3
	// canvas.DrawHLine(exitX, exitX+exitRailWidth-1, trackY, '━')

	return canvas.String()
}

func (r *asciiRenderer) AcceptStation(s *station) error {
	startX, startY := r.x, r.y
	boxWidth := len(s.label) + 4 // label + padding
	boxHeight := 3

	r.canvas.DrawBox(r.x, r.y, boxWidth, boxHeight, s.label)

	// Draw rails entering/exiting
	r.canvas.Set(r.x-1, r.y+1, '━')        // Entry rail
	r.canvas.Set(r.x+boxWidth, r.y+1, '━') // Exit rail

	// Push dimensions onto stack
	r.push(renderFrame{
		startX: startX,
		startY: startY,
		width:  boxWidth,
		height: boxHeight,
	})

	// Move cursor to the right for next element
	r.x += boxWidth

	return nil
}

func (r *asciiRenderer) AcceptRail(rl *rail) error {
	startX, startY := r.x, r.y

	// Draw horizontal line
	railWidth := int(rl.width / 8) // Scale down from logical to characters
	if railWidth < 1 {
		railWidth = 1 // Minimum width of 1 character
	}

	r.canvas.DrawHLine(r.x, r.x+railWidth+10, r.y, '━')

	// Push dimensions
	r.push(renderFrame{
		startX: startX,
		startY: startY,
		width:  railWidth,
		height: 1,
	})

	// Move cursor
	r.x += railWidth

	return nil
}

func (r *asciiRenderer) AcceptHConcat(h *hconcat) error {
	startX, startY := r.x, r.y
	maxHeight := 0
	elementPadding := 1 // Space between elements in sequence (stations draw their own rails)

	for i, child := range h.children {
		// Add spacing between elements (but not before first)
		if i > 0 {
			r.x += elementPadding
		}

		// Render child at current cursor position and pop
		// child's dimensions
		child.Accept(r)
		childFrame := r.pop()

		// Track maximum height for vertical alignment
		if childFrame.height > maxHeight {
			maxHeight = childFrame.height
		}

		// Cursor already moved by child, no need to update
	}

	totalWidth := r.x - startX

	r.push(renderFrame{
		startX: startX,
		startY: startY,
		width:  totalWidth,
		height: maxHeight,
	})

	return nil
}

func (r *asciiRenderer) AcceptVConcatInline(v *vconcatInline) error {
	startX, startY := r.x, r.y

	// Main track runs horizontally through the middle
	// Calculate the middle track position based on branch count and heights
	numBranches := len(v.branches)
	if numBranches == 0 {
		return nil
	}

	// Render all branches first to get their dimensions
	type branchInfo struct {
		layout   layout
		baseline float64
		width    int
		height   int
	}
	branches := make([]branchInfo, numBranches)
	maxBranchWidth := 0

	for i, branch := range v.branches {
		branchBaseline := 1.0
		if station, ok := branch.(*station); ok {
			branchBaseline = station.baseline / 8.0
			if branchBaseline < 1 {
				branchBaseline = 1
			}
		}

		// Render branch to get dimensions
		tempX, tempY := r.x, r.y
		r.x = startX + 100 // Temp position off-screen
		r.y = startY + 100
		branch.Accept(r)
		frame := r.pop()
		r.x = tempX
		r.y = tempY

		branches[i] = branchInfo{
			layout:   branch,
			baseline: branchBaseline,
			width:    frame.width,
			height:   frame.height,
		}
		if frame.width > maxBranchWidth {
			maxBranchWidth = frame.width
		}
	}

	// Main track runs horizontally at the choice's baseline (from Phase 1)
	// This should align with entry/exit rails from layoutToASCII (typically at startY + 1)
	mainTrackY := startY + int(v.baseline/8.0) // Scale baseline to char units
	if mainTrackY < startY+1 {
		mainTrackY = startY + 1
	}

	// Position branches symmetrically around the main track
	// For two branches: place first above, second below
	branchPositions := make([]int, numBranches) // Y position for each branch
	branchTrackYs := make([]int, numBranches)   // Track Y for each branch

	if numBranches == 2 {
		// Position branches so main track is in the middle
		// First branch goes above the main track
		// Gap includes: 1 row from branch track to branch bottom, 1 row vertical line, 1 row to main track
		verticalGap := 2

		// First branch: position so its track is verticalGap above main track
		branchTrackYs[0] = mainTrackY - verticalGap
		branchPositions[0] = branchTrackYs[0] - int(branches[0].baseline)

		// Second branch: position so its track is verticalGap below main track
		branchTrackYs[1] = mainTrackY + verticalGap
		branchPositions[1] = branchTrackYs[1] - int(branches[1].baseline)
	} else {
		// For multiple branches, distribute them sequentially
		// (This is a fallback; proper multi-branch layout needs more work)
		currentY := startY
		for i, b := range branches {
			branchPositions[i] = currentY
			branchTrackYs[i] = currentY + int(b.baseline)
			currentY += b.height + 1
		}
	}

	// Split point on the left
	splitX := startX + 4
	// Merge point on the right
	mergeX := splitX + maxBranchWidth + 4

	// Draw the main horizontal track through
	r.canvas.Set(splitX, mainTrackY, '┫')
	r.canvas.Set(mergeX, mainTrackY, '┣')

	// Draw each branch
	for i, b := range branches {
		branchY := branchPositions[i]
		branchTrackY := branchTrackYs[i]

		// Draw the branch
		r.x = splitX + 2
		r.y = branchY
		b.layout.Accept(r)
		r.pop()

		// Draw connectors from split to branch
		if branchTrackY < mainTrackY {
			// Branch above main track
			r.canvas.Set(splitX, branchTrackY, '┏')
			r.canvas.DrawHLine(splitX+1, splitX+1, branchTrackY, '━')
			for y := branchTrackY + 1; y < mainTrackY; y++ {
				r.canvas.Set(splitX, y, '┃')
			}
		} else if branchTrackY > mainTrackY {
			// Branch below main track
			r.canvas.Set(splitX, branchTrackY, '┗')
			r.canvas.DrawHLine(splitX+1, splitX+1, branchTrackY, '━')
			for y := mainTrackY + 1; y < branchTrackY; y++ {
				r.canvas.Set(splitX, y, '┃')
			}
		}

		// Draw connectors from branch to merge
		branchEndX := splitX + 2 + b.width
		if branchTrackY < mainTrackY {
			// Branch above main track
			r.canvas.Set(mergeX, branchTrackY, '┓')
			r.canvas.DrawHLine(branchEndX, mergeX-1, branchTrackY, '━')
			for y := branchTrackY + 1; y < mainTrackY; y++ {
				r.canvas.Set(mergeX, y, '┃')
			}
		} else if branchTrackY > mainTrackY {
			// Branch below main track
			r.canvas.Set(mergeX, branchTrackY, '┛')
			r.canvas.DrawHLine(branchEndX, mergeX-1, branchTrackY, '━')
			for y := mainTrackY + 1; y < branchTrackY; y++ {
				r.canvas.Set(mergeX, y, '┃')
			}
		}
	}

	// Calculate total dimensions
	// Find the topmost and bottommost points
	minY := branchPositions[0]
	maxY := branchPositions[0] + branches[0].height
	for i := 1; i < numBranches; i++ {
		if branchPositions[i] < minY {
			minY = branchPositions[i]
		}
		bottomY := branchPositions[i] + branches[i].height
		if bottomY > maxY {
			maxY = bottomY
		}
	}

	// The frame starts at the topmost branch position
	actualStartY := minY
	totalHeight := maxY - minY
	totalWidth := mergeX - startX + 1

	r.push(renderFrame{
		startX: startX,
		startY: actualStartY,
		width:  totalWidth,
		height: totalHeight,
	})

	r.x = mergeX + 1
	// Set cursor Y so that the next element's baseline aligns with main track
	// Since elements typically have track at startY + 1, we set r.y = mainTrackY - 1
	r.y = mainTrackY - 1

	return nil
}

func (r *asciiRenderer) AcceptVConcatBlock(v *vconcatBlock) error {
	startX, startY := r.x, r.y

	// Special case: pol_plus means this is a wrapped sequence, not a loop
	// Render lines stacked vertically without loop arrows
	if v.polarity == pol_plus {
		// Render main path (first line)
		v.mainPath.Accept(r)
		mainFrame := r.pop()

		// Render loop path (remaining lines, might be nested vconcat-block)
		r.x = startX
		r.y = startY + mainFrame.height + 1 // Next line
		v.loopPath.Accept(r)
		loopFrame := r.pop()

		// Push combined dimensions
		totalHeight := mainFrame.height + 1 + loopFrame.height
		totalWidth := max(mainFrame.width, loopFrame.width)

		r.push(renderFrame{
			startX: startX,
			startY: startY,
			width:  totalWidth,
			height: totalHeight,
		})

		// Move cursor past the wrapped content
		r.x = startX + totalWidth
		r.y = startY

		return nil
	}

	// Normal case: this is a loop (pol_minus)
	// Check if main path is empty (zero-or-more case: (- () X))
	if rail, isRail := v.mainPath.(*rail); isRail && rail.width == 0 {
		// Render zero-or-more pattern: straight track with
		// loop below
		//
		// ━┳━━━━━━━┳━
		//  ┃ ┌───┐ ┃
		//  ┗━│ X │━┛
		//    └───┘
		trackY := startY + 1 // Main track at row 1

		// Render loop element to get its dimensions
		tempX, tempY := r.x, r.y
		r.x = startX + 100
		r.y = startY + 100
		v.loopPath.Accept(r)
		loopFrame := r.pop()
		r.x = tempX
		r.y = tempY

		// Entry junction
		entryX := startX
		r.canvas.Set(entryX, trackY, '┳')

		loopStartX := entryX + 2
		loopY := trackY + 2
		exitX := loopStartX + loopFrame.width + 1

		// straight-through track
		r.canvas.DrawHLine(entryX+1, exitX-1, trackY, '━')

		r.canvas.Set(exitX, trackY, '┳')
		r.canvas.Set(entryX, trackY+1, '┃')
		r.canvas.Set(exitX, trackY+1, '┃')
		r.canvas.Set(entryX, loopY, '┗')
		r.canvas.Set(exitX, loopY, '┛')

		// Render loop element
		loopBaseline := 1.0
		if station, ok := v.loopPath.(*station); ok {
			loopBaseline = station.baseline / 8.0
			if loopBaseline < 1 {
				loopBaseline = 1
			}
		}
		r.x = loopStartX
		r.y = loopY - int(loopBaseline)
		v.loopPath.Accept(r)
		r.pop()

		// Draw connecting rails
		r.canvas.DrawHLine(entryX+1, loopStartX-1, loopY, '━')
		r.canvas.DrawHLine(loopStartX+loopFrame.width, exitX-1, loopY, '━')

		// Push dimensions
		r.push(renderFrame{
			startX: startX,
			startY: startY,
			width:  exitX - startX + 1,
			height: loopY - startY + 1,
		})

		r.x = exitX + 1
		r.y = startY
		return nil
	}

	// Normal case: main path is not empty
	mainBaseline := 1.0
	if station, ok := v.mainPath.(*station); ok {
		mainBaseline = station.baseline / 8.0
		if mainBaseline < 1 {
			mainBaseline = 1
		}
	}

	loopBaseline := 1.0
	if station, ok := v.loopPath.(*station); ok {
		loopBaseline = station.baseline / 8.0
		if loopBaseline < 1 {
			loopBaseline = 1
		}
	}

	// Draw entry junction before the loop
	entryX := startX
	trackY := startY + int(mainBaseline)
	r.canvas.Set(entryX, trackY, '┳')

	// Render main path horizontally (after entry junction)
	r.x = startX + 2 // Small gap after entry junction
	r.y = startY
	v.mainPath.Accept(r)
	mainFrame := r.pop()

	// Draw entry rail from junction to main path
	r.canvas.DrawHLine(entryX+1, startX+1, trackY, '━')

	// Check if loop path is empty (one-or-more case)
	if _, isEmpty := v.loopPath.(*rail); isEmpty {
		exitX := startX + 3 + mainFrame.width
		r.canvas.Set(exitX, trackY, '┳')
		r.canvas.Set(entryX, trackY+1, '┃')
		r.canvas.Set(exitX, trackY+1, '┃')
		// return track going back
		returnY := trackY + 2
		r.canvas.DrawHLine(entryX, exitX, returnY, '━')
		r.canvas.Set(exitX, returnY, '┛')
		r.canvas.Set(entryX, returnY, '┗')

		r.push(renderFrame{
			startX: startX,
			startY: startY,
			width:  exitX - startX + 1,
			height: 3, // main + drop + return
		})

		r.x = exitX + 1
		r.y = startY
		return nil
	}

	// For loops with separator: render horizontally with return
	// underneath Draw connecting rail from main path to loop path
	r.canvas.DrawHLine(startX+2+mainFrame.width, startX+2+mainFrame.width+1, trackY, '━')

	// Render loop path continuing horizontally
	r.x = startX + 2 + mainFrame.width + 2
	r.y = startY
	v.loopPath.Accept(r)
	loopFrame := r.pop()

	// Draw exit junction after loop path
	exitX := startX + 2 + mainFrame.width + 2 + loopFrame.width
	r.canvas.Set(exitX, trackY, '┳')

	// Draw exit rail after merge point
	exitPadding := 2
	r.canvas.DrawHLine(exitX+1, exitX+exitPadding, trackY, '━')

	dropY := trackY + 1
	r.canvas.Set(entryX, dropY, '┃')
	r.canvas.Set(exitX, dropY, '┃')

	// return track underneath connecting back
	returnY := dropY + 1
	r.canvas.DrawHLine(entryX, exitX, returnY, '━')
	r.canvas.Set(entryX, returnY, '┗')
	r.canvas.Set(exitX, returnY, '┛')

	totalWidth := exitX - startX + exitPadding + 1
	maxHeight := max(mainFrame.height, loopFrame.height)
	totalHeight := maxHeight + 2 // +2 for drop and return track

	// Push dimensions
	r.push(renderFrame{
		startX: startX,
		startY: startY,
		width:  totalWidth,
		height: totalHeight,
	})

	r.x = exitX + exitPadding + 1
	r.y = startY

	return nil
}

func (r *asciiRenderer) AcceptSpace(v *space) error {
	startX, startY := r.x, r.y

	// Space is flexible - in a full implementation this would
	// expand during justification.  For now, use railGap scaled
	// to character width (same scaling as rails use)
	spaceWidth := int(railGap) / 8 // railGap is 10.0, so 10/8 = 1 char
	if spaceWidth < 1 {
		spaceWidth = 1 // Minimum one character
	}

	// r.canvas.DrawHLine(r.x, r.x+spaceWidth-1, r.y, '·')

	r.push(renderFrame{
		startX: startX,
		startY: startY,
		width:  spaceWidth,
		height: 1,
	})

	r.x += spaceWidth

	return nil
}

// calculateTopPadding determines how much vertical space we need above y=0
// to prevent clipping of elements that extend above their baseline
func calculateTopPadding(l layout) int {
	visitor := &topPaddingCalculator{}
	l.Accept(visitor)
	return visitor.maxTopExtent
}

type topPaddingCalculator struct {
	maxTopExtent int
}

func (c *topPaddingCalculator) AcceptRail(v *rail) error {
	return nil
}

func (c *topPaddingCalculator) AcceptSpace(v *space) error {
	return nil
}

func (c *topPaddingCalculator) AcceptStation(v *station) error {
	// Stations don't extend above their start position
	return nil
}

func (c *topPaddingCalculator) AcceptHConcat(v *hconcat) error {
	// Check all children
	for _, child := range v.children {
		child.Accept(c)
	}
	return nil
}

func (c *topPaddingCalculator) AcceptVConcatInline(v *vconcatInline) error {
	// This is a choice - need to account for top branch
	if len(v.branches) == 0 {
		return nil
	}

	// For a 2-branch choice, the top branch needs space above the main track
	// Main track is at baseline, top branch track is 2 rows above
	// Top branch position is: topBranchTrack - topBranchBaseline

	// Simplified calculation: assume we need at least 3 rows above baseline
	// (1 for box top, 1 for track, 1 for connection)
	topExtent := int(v.baseline/8.0) + 3

	if topExtent > c.maxTopExtent {
		c.maxTopExtent = topExtent
	}

	// Recurse into branches
	for _, branch := range v.branches {
		branch.Accept(c)
	}

	return nil
}

func (c *topPaddingCalculator) AcceptVConcatBlock(v *vconcatBlock) error {
	// Check both paths
	v.mainPath.Accept(c)
	v.loopPath.Accept(c)
	return nil
}
