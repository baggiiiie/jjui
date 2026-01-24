package render

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/layout"
)

type RenderItemFunc func(dl *DisplayContext, index int, rect cellbuf.Rectangle)

type MeasureItemFunc func(index int) int

type ClickMessage = tea.Msg

type ClickMessageFunc func(index int) ClickMessage

type DragMessageFunc func(index int) tea.Msg

type ListRenderer struct {
	StartLine     int
	ScrollMsg     tea.Msg
	FirstRowIndex int
	LastRowIndex  int
	lastSpans     []Span // Store spans from last render for drag interactions
}

func NewListRenderer(scrollMsg tea.Msg) *ListRenderer {
	return &ListRenderer{
		StartLine:     0,
		ScrollMsg:     scrollMsg,
		FirstRowIndex: -1,
		LastRowIndex:  -1,
	}
}

// Render renders visible items to the DisplayContext.
// Note: Scroll interaction registration is the caller's responsibility.
func (r *ListRenderer) Render(
	dl *DisplayContext,
	viewRect layout.Box,
	itemCount int,
	cursor int,
	ensureCursorVisible bool,
	measure MeasureItemFunc,
	render RenderItemFunc,
	clickMsg ClickMessageFunc,
) {
	if itemCount <= 0 {
		return
	}

	viewHeight := viewRect.R.Dy()
	totalLines := 0
	for i := 0; i < itemCount; i++ {
		totalLines += measure(i)
	}
	maxStart := totalLines - viewHeight
	if maxStart < 0 {
		maxStart = 0
	}
	if r.StartLine < 0 {
		r.StartLine = 0
	}
	if r.StartLine > maxStart {
		r.StartLine = maxStart
	}

	viewport := Viewport{
		StartLine: r.StartLine,
		ViewRect: layout.Box{
			R: cellbuf.Rect(viewRect.R.Min.X, viewRect.R.Min.Y, viewRect.R.Dx(), viewRect.R.Dy()),
		},
	}

	if ensureCursorVisible && cursor >= 0 && cursor < itemCount {
		r.ensureCursorVisible(cursor, itemCount, viewRect.R.Dy(), measure)
		viewport.StartLine = r.StartLine
	}

	measureAdapter := func(req MeasureRequest) MeasureResult {
		if req.Index >= itemCount {
			return MeasureResult{DesiredLine: 0, MinLine: 0}
		}
		height := measure(req.Index)
		return MeasureResult{
			DesiredLine: height,
			MinLine:     height,
		}
	}

	spans, _ := LayoutAll(viewport, itemCount, measureAdapter)
	r.lastSpans = spans // Store spans for drag interactions
	r.FirstRowIndex = -1
	r.LastRowIndex = -1
	for _, span := range spans {
		if r.FirstRowIndex == -1 {
			r.FirstRowIndex = span.Index
		}
		r.LastRowIndex = span.Index
	}

	for _, span := range spans {
		if span.Index >= itemCount {
			continue
		}
		render(dl, span.Index, span.Rect)
	}

	for _, span := range spans {
		if span.Index >= itemCount {
			continue
		}

		dl.AddInteraction(
			span.Rect,
			clickMsg(span.Index),
			InteractionClick,
			0,
		)
	}

}

func (r *ListRenderer) ensureCursorVisible(
	cursor int,
	itemCount int,
	viewportHeight int,
	measure MeasureItemFunc,
) {
	if cursor < 0 || cursor >= itemCount || viewportHeight <= 0 {
		return
	}

	cursorStart := 0
	for i := 0; i < cursor && i < itemCount; i++ {
		cursorStart += measure(i)
	}

	cursorHeight := measure(cursor)
	cursorEnd := cursorStart + cursorHeight

	start := r.StartLine
	if start < 0 {
		start = 0
	}

	viewportEnd := start + viewportHeight

	if cursorStart < start {
		r.StartLine = cursorStart
	} else if cursorEnd > viewportEnd {
		r.StartLine = cursorEnd - viewportHeight
		if r.StartLine < 0 {
			r.StartLine = 0
		}
	}
}

func (r *ListRenderer) SetScrollOffset(offset int) {
	r.StartLine = offset
}

func (r *ListRenderer) GetScrollOffset() int {
	return r.StartLine
}

func (r *ListRenderer) GetFirstRowIndex() int {
	return r.FirstRowIndex
}

func (r *ListRenderer) GetLastRowIndex() int {
	return r.LastRowIndex
}

// RegisterScroll registers a scroll interaction for the given view rect.
// Call this after Render if you want to enable mouse wheel scrolling.
func (r *ListRenderer) RegisterScroll(dl *DisplayContext, viewRect layout.Box) {
	if r.ScrollMsg == nil {
		return
	}
	dl.AddInteraction(
		viewRect.R,
		r.ScrollMsg,
		InteractionScroll,
		0,
	)
}

// RegisterDrag registers drag interactions for each item.
// Call this after Render if you want to enable drag selection.
func (r *ListRenderer) RegisterDrag(dl *DisplayContext, dragMsg DragMessageFunc) {
	if dragMsg == nil {
		return
	}
	for _, span := range r.lastSpans {
		dl.AddInteraction(
			span.Rect,
			dragMsg(span.Index),
			InteractionDrag,
			0,
		)
	}
}

// ItemIndexAt returns the item index at the given screen position.
// Returns -1 if no item is at that position.
func (r *ListRenderer) ItemIndexAt(x, y int) int {
	for _, span := range r.lastSpans {
		if x >= span.Rect.Min.X && x < span.Rect.Max.X &&
			y >= span.Rect.Min.Y && y < span.Rect.Max.Y {
			return span.Index
		}
	}
	return -1
}
