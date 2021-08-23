package applier

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/fatih/color"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
)

type IAMZeroDiff struct {
	unified *gotextdiff.Unified
}

func IAMZeroDiffFromUnified(u gotextdiff.Unified) IAMZeroDiff {
	return IAMZeroDiff{unified: &u}
}

func (d IAMZeroDiff) Format(f fmt.State, r rune) {
	u := *d.unified

	red := color.New(color.FgRed)
	green := color.New(color.FgGreen)

	if len(u.Hunks) == 0 {
		return
	}
	fmt.Fprintf(f, "%s\n", u.From)
	// fmt.Fprintf(f, "+++ %s\n", u.To)
	for _, hunk := range u.Hunks {
		fromCount, toCount := 0, 0
		for _, l := range hunk.Lines {
			switch l.Kind {
			case gotextdiff.Delete:
				fromCount++
			case gotextdiff.Insert:
				toCount++
			default:
				fromCount++
				toCount++
			}
		}
		fmt.Fprint(f, "@@")
		if fromCount > 1 {
			red.Fprintf(f, " -%d,%d", hunk.FromLine, fromCount)
		} else {
			red.Fprintf(f, " -%d", hunk.FromLine)
		}
		if toCount > 1 {
			green.Fprintf(f, " +%d,%d", hunk.ToLine, toCount)
		} else {
			green.Fprintf(f, " +%d", hunk.ToLine)
		}
		fmt.Fprint(f, " @@\n")
		for _, l := range hunk.Lines {
			switch l.Kind {
			case gotextdiff.Delete:
				red.Fprintf(f, "-%s", l.Content)
			case gotextdiff.Insert:
				green.Fprintf(f, "+%s", l.Content)
			default:
				fmt.Fprintf(f, " %s", l.Content)
			}
			if !strings.HasSuffix(l.Content, "\n") {
				fmt.Fprintf(f, "\n\\ No newline at end of file\n")
			}
		}
	}
}

// GetDiff reads a file and gets the diff between the file and a provided
// `modified` string
func GetDiff(file, modified string) (string, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}
	edits := myers.ComputeEdits(span.URIFromPath(file), string(data), modified)
	u := gotextdiff.ToUnified(file, file, string(data), edits)
	f := IAMZeroDiffFromUnified(u)
	diff := fmt.Sprint(f)
	return diff, nil
}
