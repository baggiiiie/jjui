package parser

import (
	"log"
	"strconv"
	"strings"

	"github.com/idursun/jjui/internal/screen"
)

type GraphRowLine struct {
	Segments []*screen.Segment
	Gutter   GraphGutter
	Flags    RowLineFlags
}

func NewGraphRowLine(segments []*screen.Segment) GraphRowLine {
	return GraphRowLine{
		Segments: segments,
		Gutter:   GraphGutter{Segments: make([]*screen.Segment, 0)},
	}
}

func (gr *GraphRowLine) findNextPrefix(startIdx int) int {
	for i := startIdx + 1; i < len(gr.Segments); i++ {
		if strings.TrimSpace(gr.Segments[i].Text) != "" &&
			strings.TrimSpace(gr.Segments[i].Text) != "," {
			return i
		}
	}
	return -1
}

func (gr *GraphRowLine) ParseRowPrefixes() (int, string, string, bool) {
	var w strings.Builder
	for _, segment := range gr.Segments {
		w.WriteString(segment.Text + "|")
	}
	log.Println(w.String())
	changeIDIdx := -1
	for i, segment := range gr.Segments {
		if isChangeIDLike(segment.Text) {
			log.Println(segment.Text, "is change id like")
			changeIDIdx = i
			break
		}
	}

	if changeIDIdx == -1 {
		return -1, "", "", false
	}
	changeIDText := strings.TrimSpace(gr.Segments[changeIDIdx].Text)

	// Check if changeID and commitID are in the same segment (comma-separated)
	var changeID, commitID string
	var commitIDIdx int

	if strings.Contains(changeIDText, ",") {
		// Same color case: "ky,38"
		parts := strings.SplitN(changeIDText, ",", 2)
		changeID = strings.TrimSpace(parts[0])
		commitID = strings.TrimSpace(parts[1])
		commitIDIdx = changeIDIdx // They're in the same segment
	} else {
		// Different color case: separate segments
		changeID = changeIDText
		commitIDIdx = gr.findNextPrefix(changeIDIdx)
		if commitIDIdx == -1 {
			return -1, "", "", false
		}
		commitID = strings.TrimSpace(gr.Segments[commitIDIdx].Text)
	}

	isDivergentIdx := gr.findNextPrefix(commitIDIdx)
	if isDivergentIdx == -1 {
		return -1, "", "", false
	}
	isDivergent, err := strconv.ParseBool(strings.TrimSpace(gr.Segments[isDivergentIdx].Text))
	if err != nil {
		isDivergent = false
	}

	// Remove changeID, commitID, and isDivergent prefixes
	gr.Segments = append(gr.Segments[:changeIDIdx], gr.Segments[isDivergentIdx+1:]...)

	return changeIDIdx, changeID, commitID, isDivergent
}

func (gr *GraphRowLine) chop(indent int) {
	if len(gr.Segments) == 0 {
		return
	}
	segments := gr.Segments
	gr.Segments = make([]*screen.Segment, 0)

	for i, s := range segments {
		extended := screen.Segment{
			Style: s.Style,
		}
		var textBuilder strings.Builder
		for _, p := range s.Text {
			if indent <= 0 {
				break
			}
			textBuilder.WriteRune(p)
			indent--
		}
		extended.Text = textBuilder.String()
		gr.Gutter.Segments = append(gr.Gutter.Segments, &extended)
		if len(extended.Text) < len(s.Text) {
			gr.Segments = append(gr.Segments, &screen.Segment{
				Text:  s.Text[len(extended.Text):],
				Style: s.Style,
			})
		}
		if indent <= 0 && len(segments)-i-1 > 0 {
			gr.Segments = segments[i+1:]
			break
		}
	}

	// break gutter into segments per rune
	segments = gr.Gutter.Segments
	gr.Gutter.Segments = make([]*screen.Segment, 0)
	for _, s := range segments {
		for _, p := range s.Text {
			extended := screen.Segment{
				Text:  string(p),
				Style: s.Style,
			}
			gr.Gutter.Segments = append(gr.Gutter.Segments, &extended)
		}
	}

	// Pad with spaces if indent is not fully consumed
	if indent > 0 && len(gr.Gutter.Segments) > 0 {
		lastSegment := gr.Gutter.Segments[len(gr.Gutter.Segments)-1]
		lastSegment.Text += strings.Repeat(" ", indent)
	}
}

func (gr *GraphRowLine) containsRune(r rune) bool {
	for _, segment := range gr.Gutter.Segments {
		if strings.ContainsRune(segment.Text, r) {
			return true
		}
	}
	return false
}
