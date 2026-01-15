package bookmark_move

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/bookmark_panel"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

var _ operations.Operation = (*Operation)(nil)
var _ common.Focusable = (*Operation)(nil)

type Operation struct {
	context      *context.MainContext
	bookmarkName string
	targetCommit *jj.Commit
}

func (o *Operation) Init() tea.Cmd {
	return nil
}

func (o *Operation) IsFocused() bool {
	// Focused to receive key events (esc/enter) for move mode
	return true
}

func (o *Operation) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			// Use Sequence to ensure EndMoveModeMsg is processed before closing
			return tea.Sequence(
				func() tea.Msg { return bookmark_panel.EndMoveModeMsg{} },
				common.Close,
			)
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if o.targetCommit != nil {
				// Move bookmark to target revision
				return tea.Sequence(
					o.context.RunCommand(
						jj.BookmarkMove(o.targetCommit.GetChangeId(), o.bookmarkName),
						common.Refresh,
					),
					func() tea.Msg { return bookmark_panel.EndMoveModeMsg{} },
					common.Close,
				)
			}
			return tea.Sequence(
				func() tea.Msg { return bookmark_panel.EndMoveModeMsg{} },
				common.Close,
			)
		}
	}
	return nil
}

func (o *Operation) View() string {
	return ""
}

func (o *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if o.targetCommit == nil {
		return ""
	}

	changeId := commit.GetChangeId()

	switch pos {
	case operations.RenderBeforeChangeId:
		if o.targetCommit.GetChangeId() == changeId {
			return "<< onto >> "
		}
	case operations.RenderPositionAfter:
		if o.targetCommit.GetChangeId() == changeId {
			palette := common.DefaultPalette
			marker := palette.Get("revisions markers").Render("<< onto >>")
			text := palette.Get("revisions text").Render(
				fmt.Sprintf("move bookmark '%s' to %s", o.bookmarkName, changeId))
			return fmt.Sprintf("%s %s", marker, text)
		}
	}

	return ""
}

func (o *Operation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	o.targetCommit = commit
	return nil
}

func (o *Operation) Name() string {
	return fmt.Sprintf("move bookmark '%s'", o.bookmarkName)
}

func NewOperation(ctx *context.MainContext, bookmarkName string) *Operation {
	return &Operation{
		context:      ctx,
		bookmarkName: bookmarkName,
	}
}
