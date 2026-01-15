package bookmark_create

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/bookmark_panel"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

type state int

const (
	stateSelectingRevision state = iota
	stateEnteringName
)

var _ operations.Operation = (*Operation)(nil)
var _ operations.TracksSelectedRevision = (*Operation)(nil)
var _ common.Focusable = (*Operation)(nil)
var _ common.Editable = (*Operation)(nil)

type Operation struct {
	context      *context.MainContext
	targetCommit *jj.Commit
	state        state
	name         textinput.Model
}

func (o *Operation) Init() tea.Cmd {
	return nil
}

func (o *Operation) IsFocused() bool {
	return o.state == stateSelectingRevision
}

func (o *Operation) IsEditing() bool {
	return o.state == stateEnteringName
}

func (o *Operation) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch o.state {
		case stateSelectingRevision:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				// Cancel and go back to bookmark panel
				return tea.Sequence(
					func() tea.Msg { return bookmark_panel.EndCreateModeMsg{} },
					common.Close,
				)
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if o.targetCommit != nil {
					// Move to name entry state
					o.state = stateEnteringName
					o.name.Focus()
					// Load bookmark suggestions (same as SetBookmarkOperation)
					if output, err := o.context.RunCommandImmediate(jj.BookmarkListMovable(o.targetCommit.GetChangeId())); err == nil {
						bookmarks := jj.ParseBookmarkListOutput(string(output))
						var suggestions []string
						for _, b := range bookmarks {
							if b.Name != "" && !b.Backwards {
								suggestions = append(suggestions, b.Name)
							}
						}
						o.name.SetSuggestions(suggestions)
					}
					return textinput.Blink
				}
				return nil
			}
		case stateEnteringName:
			switch msg.String() {
			case "esc":
				// Go back to selecting revision
				o.state = stateSelectingRevision
				o.name.SetValue("")
				o.name.Blur()
				return nil
			case "enter":
				name := strings.TrimSpace(o.name.Value())
				if name == "" {
					return nil
				}
				// Create the bookmark
				return tea.Sequence(
					o.context.RunCommand(
						jj.BookmarkCreate(o.targetCommit.GetChangeId(), name),
						common.Refresh,
					),
					func() tea.Msg { return bookmark_panel.EndCreateModeMsg{} },
					common.Close,
				)
			default:
				var cmd tea.Cmd
				o.name, cmd = o.name.Update(msg)
				o.name.SetValue(strings.ReplaceAll(o.name.Value(), " ", "-"))
				return cmd
			}
		}
	}
	return nil
}

func (o *Operation) ViewRect(_ *render.DisplayContext, _ layout.Box) {}

func (o *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if o.targetCommit == nil {
		return ""
	}

	changeId := commit.GetChangeId()

	switch pos {
	case operations.RenderBeforeChangeId:
		if o.targetCommit.GetChangeId() == changeId {
			if o.state == stateEnteringName {
				return o.name.View() + o.name.TextStyle.Render(" ")
			}
			return "<< on >> "
		}
	case operations.RenderPositionAfter:
		if o.targetCommit.GetChangeId() == changeId {
			palette := common.DefaultPalette
			marker := palette.Get("revisions markers").Render("<< on >>")
			var text string
			if o.state == stateEnteringName {
				text = palette.Get("revisions text").Render("enter bookmark name")
			} else {
				text = palette.Get("revisions text").Render("create bookmark here")
			}
			return fmt.Sprintf("%s %s", marker, text)
		}
	}

	return ""
}

func (o *Operation) RenderToDisplayContext(_ *render.DisplayContext, _ *jj.Commit, _ operations.RenderPosition, _ cellbuf.Rectangle, _ cellbuf.Position) int {
	return 0
}

func (o *Operation) DesiredHeight(_ *jj.Commit, _ operations.RenderPosition) int {
	return 0
}

func (o *Operation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	o.targetCommit = commit
	return nil
}

func (o *Operation) Name() string {
	if o.state == stateEnteringName {
		return "create bookmark (enter name)"
	}
	return "create bookmark"
}

func NewOperation(ctx *context.MainContext) *Operation {
	dimmedStyle := common.DefaultPalette.Get("revisions dimmed").Inline(true)
	textStyle := common.DefaultPalette.Get("revisions text").Inline(true)
	t := textinput.New()
	t.Width = 0
	t.ShowSuggestions = true
	t.CharLimit = 120
	t.Prompt = ""
	t.TextStyle = textStyle
	t.PromptStyle = t.TextStyle
	t.Cursor.TextStyle = t.TextStyle
	t.CompletionStyle = dimmedStyle
	t.PlaceholderStyle = t.CompletionStyle
	t.SetValue("")

	return &Operation{
		context: ctx,
		state:   stateSelectingRevision,
		name:    t,
	}
}
