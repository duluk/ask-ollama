package linewrap

import (
	"bytes"
	// "fmt"
	"io"
	"strings"
)

type LineWrapper struct {
	maxWidth  int
	currWidth int
	tabWidth  int
	writer    io.Writer
}

func NewLineWrapper(maxWidth, tabWidth int, lwWriter io.Writer) *LineWrapper {
	return &LineWrapper{
		maxWidth: maxWidth,
		tabWidth: tabWidth,
		writer:   lwWriter,
	}
}

// Allow wrapping for input that comes in chunks, versus building the line and
// then splitting it and printing the whole line at once.
func (lw *LineWrapper) Write(data []byte) (n int, err error) {
	var buffer bytes.Buffer

	for i, b := range data {
		// Debug stuff I want to leave for future usage
		// if b < 32 || b > 126 {
		// 	fmt.Printf("Special character: %q (ASCII: %d)\n", b, b)
		// } else {
		// 	fmt.Printf("Character: %q (ASCII: %d)\n", b, b)
		// }

		switch b {
		case '\n':
			buffer.WriteByte('\n')
			lw.currWidth = 0
		case '\t':
			spaces := strings.Repeat(" ", lw.tabWidth)
			lw.currWidth += lw.tabWidth
			buffer.WriteString(spaces)
		case ' ':
			// Don't write a space if we're at the beginning of a line...
			if lw.currWidth != 0 {
				buffer.WriteByte(' ')
			}
			// ...unless the next character is a space (and is within bounds),
			// then we want to write it. This is for code indentation. However,
			// there is the possibility that we are reading a line that ends on
			// exactly maxWidth, followed by two spaces on the beginning of the
			// next line, both of which would be printed even though we
			// wouldn't want that. I'm not sure how to account for that.
			if lw.currWidth == 0 && i+1 < len(data) {
				if data[i+1] == ' ' {
					buffer.WriteByte(' ')
				}
			}

			// We still need to count the width; otherwise, all initial spaces
			// will likely be ignored
			lw.currWidth++

			if lw.currWidth >= lw.maxWidth {
				buffer.WriteByte('\n')
				lw.currWidth = 0
			}
		default:
			buffer.WriteByte(b)
			lw.currWidth++
		}
	}

	lw.writer.Write(buffer.Bytes())

	return len(data), nil
}
