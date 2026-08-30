package render

import (
	"io"
	"text/tabwriter"
)

// NewTabWriter returns a tabwriter configured the same way for every human table hasp prints —
// one place to tune column spacing so every noun's output stays visually consistent (P7).
func NewTabWriter(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
}
