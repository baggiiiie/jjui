package batch

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

// Operation represents batch selection mode
type Operation struct {
	context *context.MainContext
	start   *jj.Commit // The revision where batch mode was initiated (RevisionA)
	current *jj.Commit // The currently highlighted revision
	focused bool
	keyMap  config.KeyMappings[key.Binding]
	styles  styles
}

type styles struct {
	marker lipgloss.Style
}

var (
	_ operations.Operation              = (*Operation)(nil)
	_ common.Focusable                  = (*Operation)(nil)
	_ operations.TracksSelectedRevision = (*Operation)(nil)
)

// NewOperation creates a new batch selection operation
func NewOperation(ctx *context.MainContext, start *jj.Commit) *Operation {
	markerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12"))

	return &Operation{
		context: ctx,
		start:   start,
		current: start,
		focused: true,
		keyMap:  config.Current.GetKeyMap(),
		styles: styles{
			marker: markerStyle,
		},
	}
}

func (o *Operation) Init() tea.Cmd {
	return nil
}

func (o *Operation) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		return o.handleIntent(msg)
	case tea.KeyMsg:
		return o.HandleKey(msg)
	}
	return nil
}

func (o *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, o.keyMap.Apply):
		return o.handleIntent(intents.BatchConfirm{})
	case key.Matches(msg, o.keyMap.Cancel):
		return o.handleIntent(intents.BatchCancel{})
	case key.Matches(msg, o.keyMap.ToggleSelect):
		return o.handleIntent(intents.RevisionsToggleSelect{})
	}
	return nil
}

func (o *Operation) handleIntent(intent intents.Intent) tea.Cmd {
	switch intent.(type) {
	case intents.BatchConfirm:
		return o.confirmSelection()
	case intents.BatchCancel:
		return o.cancel()
	case intents.RevisionsToggleSelect:
		// Allow individual toggle in batch mode
		if o.current != nil {
			o.context.ToggleCheckedItem(context.SelectedRevision{
				CommitId: o.current.CommitId,
				ChangeId: o.current.ChangeId,
			})
		}
		return nil
	default:
		return nil
	}
}

// BatchToggleRangeMsg is sent when the user confirms a batch selection
type BatchToggleRangeMsg struct {
	StartChangeId string
	EndChangeId   string
}

// confirmSelection sends a message to toggle all revisions between start and current
func (o *Operation) confirmSelection() tea.Cmd {
	if o.start == nil || o.current == nil {
		return o.cancel()
	}

	return tea.Batch(
		func() tea.Msg {
			return BatchToggleRangeMsg{
				StartChangeId: o.start.ChangeId,
				EndChangeId:   o.current.ChangeId,
			}
		},
		o.cancel(),
	)
}

func (o *Operation) cancel() tea.Cmd {
	return func() tea.Msg {
		return intents.Cancel{}
	}
}

func (o *Operation) ViewRect(dl *render.DisplayContext, box layout.Box) {
	if o.start == nil {
		return
	}

	startText := fmt.Sprintf("Batch select from: %s", o.start.GetChangeId())
	if o.current != nil && o.current.ChangeId != o.start.ChangeId {
		startText += fmt.Sprintf(" to: %s", o.current.GetChangeId())
	}
	startText += " (Enter to confirm, Esc to cancel)"

	style := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	dl.AddDraw(box.R, style.Render(startText), 0)
}

func (o *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		o.keyMap.Apply,
		o.keyMap.Cancel,
	}
}

func (o *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{o.ShortHelp()}
}

func (o *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderBeforeChangeId {
		return ""
	}
	if commit == nil || o.start == nil {
		return ""
	}

	if commit.ChangeId == o.start.ChangeId {
		return o.styles.marker.Render("<< start >>")
	}

	if o.current != nil && commit.ChangeId == o.current.ChangeId {
		return o.styles.marker.Render("<< end >>")
	}

	return ""
}

func (o *Operation) RenderToDisplayContext(_ *render.DisplayContext, _ *jj.Commit, _ operations.RenderPosition, _ cellbuf.Rectangle, _ cellbuf.Position) int {
	return 0
}

func (o *Operation) DesiredHeight(_ *jj.Commit, _ operations.RenderPosition) int {
	return 0
}

func (o *Operation) Name() string {
	return "batch"
}

func (o *Operation) IsFocused() bool {
	return o.focused
}

func (o *Operation) SetFocused(focused bool) {
	o.focused = focused
}

func (o *Operation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	o.current = commit
	return nil
}
