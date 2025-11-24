package diagram

import "strings"

type ASCII struct {
	grid   [][]rune
	width  int
	height int
}

func NewASCII(width, height int) *ASCII {
	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = make([]rune, width)
		for j := range grid[i] {
			grid[i][j] = ' ' // Initialize with spaces
		}
	}
	return &ASCII{grid: grid, width: width, height: height}
}

func (c *ASCII) Set(x, y int, ch rune) {
	if x >= 0 && x < c.width && y >= 0 && y < c.height {
		c.grid[y][x] = ch
	}
}

func (c *ASCII) DrawHLine(x1, x2, y int, ch rune) {
	for x := x1; x <= x2; x++ {
		c.Set(x, y, ch)
	}
}

func (c *ASCII) DrawVLine(x, y1, y2 int, ch rune) {
	for y := y1; y <= y2; y++ {
		c.Set(x, y, ch)
	}
}

func (c *ASCII) DrawBox(x, y, width, height int, label string) {
	// Draw corners and edges
	c.Set(x, y, '┌')
	c.Set(x+width-1, y, '┐')
	c.Set(x, y+height-1, '└')
	c.Set(x+width-1, y+height-1, '┘')

	// Top and bottom
	c.DrawHLine(x+1, x+width-2, y, '─')
	c.DrawHLine(x+1, x+width-2, y+height-1, '─')

	// Left and right
	c.DrawVLine(x, y+1, y+height-2, '│')
	c.DrawVLine(x+width-1, y+1, y+height-2, '│')

	// Label (centered)
	labelX := x + (width-len(label))/2
	labelY := y + height/2
	for i, ch := range label {
		c.Set(labelX+i, labelY, ch)
	}
}

func (c *ASCII) String() string {
	var sb strings.Builder
	for _, row := range c.grid {
		sb.WriteString(string(row))
		sb.WriteRune('\n')
	}
	return sb.String()
}
