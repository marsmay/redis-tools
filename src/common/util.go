package common

import (
	"fmt"
	"strings"

	"github.com/marsmay/golib/math2"
)

func ProgressBar(width int, done, total int64, description string) {
	percent := math2.Percent(done, total, 2)
	doneWidth := math2.Min(100, int(percent)) * width / 100
	fmt.Printf("%6.2f%%|%s%s| %s\r", percent, strings.Repeat("█", doneWidth), strings.Repeat(" ", width-doneWidth), description)
}
