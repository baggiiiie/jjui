package bookmark_panel

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
)

var (
	_ common.Focusable = (*Model)(nil)
	_ help.KeyMap      = (*Model)(nil)
)

const handleSize = 3

type styles struct {
	title    lipgloss.Style
	border   lipgloss.Style
	empty    lipgloss.Style
	text     lipgloss.Style
	listText lipgloss.Style
	selected lipgloss.Style
	dimmed   lipgloss.Style
}

func createStyles() styles {
	return styles{
		title:    common.DefaultPalette.Get("bookmark panel title"),
		border:   common.DefaultPalette.GetBorder("bookmark panel border", lipgloss.NormalBorder()),
		empty:    common.DefaultPalette.Get("bookmark panel empty"),
		text:     common.DefaultPalette.Get("bookmark panel text"),
		listText: common.DefaultPalette.Get("menu text"),
		selected: common.DefaultPalette.Get("selected"),
		dimmed:   common.DefaultPalette.Get("menu dimmed"),
	}
}

type bookmarkItem struct {
	bookmark jj.Bookmark
}

func (i bookmarkItem) FilterValue() string {
	return i.bookmark.Name
}

func (i bookmarkItem) Title() string {
	title := i.bookmark.Name
	if i.bookmark.Conflict {
		title += " (conflict)"
	}
	return title
}

func (i bookmarkItem) Description() string {
	if i.bookmark.CommitId != "" {
		return fmt.Sprintf(" %s", i.bookmark.CommitId)
	}
	return ""
}

type bookmarkDelegate struct {
	styles styles
}

func (d bookmarkDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	bookmarkItem, ok := item.(bookmarkItem)
	if !ok {
		return
	}

	if m.Width() <= 0 {
		return
	}

	title := bookmarkItem.Title()
	desc := bookmarkItem.Description()

	titleStyle := d.styles.listText
	descStyle := d.styles.dimmed

	if index == m.Index() {
		titleStyle = d.styles.selected
		descStyle = d.styles.dimmed.Background(titleStyle.GetBackground())
	}

	// Render title on the left
	titleRendered := titleStyle.PaddingLeft(1).Render(title)
	titleWidth := lipgloss.Width(titleRendered)

	// Render description on the right (if it exists)
	descRendered := ""
	if desc != "" {
		availableWidth := m.Width() - titleWidth - 2 // Leave some padding
		if availableWidth > 0 && len(desc) > availableWidth {
			desc = desc[:availableWidth-1] + "…"
		}
		descRendered = descStyle.PaddingRight(1).Render(desc)
	}

	// Combine title and description on one line
	line := lipgloss.JoinHorizontal(lipgloss.Left, titleRendered, descRendered)
	line = lipgloss.PlaceHorizontal(m.Width()+2, lipgloss.Left, line, lipgloss.WithWhitespaceBackground(titleStyle.GetBackground()))

	fmt.Fprint(w, line)
}

func (d bookmarkDelegate) Height() int {
	return 1
}

func (d bookmarkDelegate) Spacing() int {
	return 1
}

func (d bookmarkDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

type loadBookmarksMsg struct {
	bookmarks []jj.Bookmark
}

type Model struct {
	context          *context.MainContext
	List             list.Model // Exported for sizing
	bookmarks        []jj.Bookmark
	visible          bool
	focused          bool
	keymap           config.KeyMappings[key.Binding]
	windowPercent    float64
	autoPosition     bool
	atBottom         bool
	moveMode         bool
	moveModeTarget   *jj.Commit
	createMode       bool
	createModeTarget *jj.Commit
	styles           styles
	// Dimensions for rendering (set by ui.go)
	width  int
	height int
}

func (m *Model) Init() tea.Cmd {
	return m.loadBookmarks
}

func (m *Model) loadBookmarks() tea.Msg {
	output, err := m.context.RunCommandImmediate(jj.BookmarkListSimple())
	if err != nil {
		return nil
	}
	bookmarks := jj.ParseSimpleBookmarkListOutput(string(output))
	return loadBookmarksMsg{bookmarks: bookmarks}
}

func (m *Model) IsFocused() bool {
	return m.focused
}

func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

func (m *Model) Visible() bool {
	return m.visible
}

func (m *Model) ToggleVisible() {
	m.visible = !m.visible
	if m.visible {
		m.focused = true
	} else {
		m.focused = false
	}
}

func (m *Model) WindowPercentage() float64 {
	return m.windowPercent
}

func (m *Model) SetWindowPercentage(percentage float64) {
	m.windowPercent = percentage
	if m.windowPercent < 10 {
		m.windowPercent = 10
	} else if m.windowPercent > 95 {
		m.windowPercent = 95
	}
}

func (m *Model) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}

func (m *Model) Expand() {
	m.SetWindowPercentage(m.windowPercent + 5)
}

func (m *Model) Shrink() {
	m.SetWindowPercentage(m.windowPercent - 5)
}

func (m *Model) AutoPosition() bool {
	return m.autoPosition
}

func (m *Model) AtBottom() bool {
	return m.atBottom
}

func (m *Model) SetPosition(autoPos bool, atBottom bool) {
	m.autoPosition = autoPos
	m.atBottom = atBottom
}

func (m *Model) IsMoveMode() bool {
	return m.moveMode
}

func (m *Model) SetMoveMode(commit *jj.Commit) {
	m.moveMode = true
	m.moveModeTarget = commit
	m.focused = false
}

func (m *Model) ExitMoveMode() {
	m.moveMode = false
	m.moveModeTarget = nil
}

func (m *Model) GetMoveModeTarget() *jj.Commit {
	return m.moveModeTarget
}

func (m *Model) IsCreateMode() bool {
	return m.createMode
}

func (m *Model) SetCreateMode(commit *jj.Commit) {
	m.createMode = true
	m.createModeTarget = commit
	m.focused = false
}

func (m *Model) ExitCreateMode() {
	m.createMode = false
	m.createModeTarget = nil
}

func (m *Model) GetCreateModeTarget() *jj.Commit {
	return m.createModeTarget
}

func (m *Model) SelectedBookmark() *jj.Bookmark {
	if item, ok := m.List.SelectedItem().(bookmarkItem); ok {
		return &item.bookmark
	}
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if !m.visible {
		return nil
	}

	switch msg := msg.(type) {
	case loadBookmarksMsg:
		m.bookmarks = msg.bookmarks
		items := make([]list.Item, len(m.bookmarks))
		for i, b := range m.bookmarks {
			items[i] = bookmarkItem{bookmark: b}
		}
		return m.List.SetItems(items)

	case EndMoveModeMsg:
		// Move operation ended, restore focus to bookmark panel
		m.moveMode = false
		m.moveModeTarget = nil
		m.focused = true
		return m.loadBookmarks

	case EndCreateModeMsg:
		// Create operation ended, restore focus to bookmark panel
		m.createMode = false
		m.createModeTarget = nil
		m.focused = true
		return m.loadBookmarks

	case tea.KeyMsg:
		if !m.focused {
			return nil
		}

		// If user is typing in the filter, let the list handle it
		if m.List.SettingFilter() {
			var cmd tea.Cmd
			m.List, cmd = m.List.Update(msg)
			return cmd
		}

		switch {
		case key.Matches(msg, m.keymap.Cancel):
			// If in move mode, just cancel move mode
			if m.moveMode {
				m.moveMode = false
				m.moveModeTarget = nil
				return nil
			}
			// If filter is active, clear it; otherwise close the panel
			if m.List.FilterState() != list.Unfiltered {
				m.List.ResetFilter()
				return nil
			}
			m.ToggleVisible()
			return nil
		case key.Matches(msg, m.keymap.Apply):
			// Enter key - update revset to trunk()::bookmark
			if bookmark := m.SelectedBookmark(); bookmark != nil {
				newRevset := fmt.Sprintf("trunk()::%s", bookmark.Name)
				return common.UpdateRevSet(newRevset)
			}
			return nil
		case msg.String() == "m":
			// Start move mode - keep panel open but unfocus it
			if bookmark := m.SelectedBookmark(); bookmark != nil {
				m.moveMode = true
				m.focused = false
				// Return a message to signal move mode with the bookmark
				return func() tea.Msg {
					return StartMoveModeMsg{Bookmark: bookmark}
				}
			}
			return nil
		case msg.String() == "c":
			// Start create mode - keep panel open but unfocus it
			m.createMode = true
			m.focused = false
			return func() tea.Msg {
				return StartCreateModeMsg{}
			}
		case msg.String() == "n":
			// Create a new revision on the current bookmark
			if bookmark := m.SelectedBookmark(); bookmark != nil {
				selected := jj.NewSelectedRevisions(&jj.Commit{ChangeId: bookmark.Name})
				return m.context.RunCommand(jj.New(selected), common.RefreshAndSelect("@"))
			}
			return nil
		case msg.String() == "d":
			// Delete bookmark
			if bookmark := m.SelectedBookmark(); bookmark != nil && bookmark.IsDeletable() {
				return m.context.RunCommand(jj.BookmarkDelete(bookmark.Name), common.Refresh, m.loadBookmarks)
			}
			return nil
		case msg.String() == "f":
			// Forget bookmark
			if bookmark := m.SelectedBookmark(); bookmark != nil {
				return m.context.RunCommand(jj.BookmarkForget(bookmark.Name), common.Refresh, m.loadBookmarks)
			}
			return nil
		case msg.String() == "t":
			// Track bookmark
			if bookmark := m.SelectedBookmark(); bookmark != nil && bookmark.IsTrackable() {
				remote := config.GetGitDefaultRemote(config.Current)
				return m.context.RunCommand(jj.BookmarkTrack(bookmark.Name, remote), common.Refresh, m.loadBookmarks)
			}
			return nil
		case msg.String() == "u":
			// Untrack bookmark (if has remotes)
			if bookmark := m.SelectedBookmark(); bookmark != nil && len(bookmark.Remotes) > 0 {
				// For now, untrack the first remote (origin if available)
				remote := bookmark.Remotes[0].Remote
				return m.context.RunCommand(jj.BookmarkUntrack(bookmark.Name, remote), common.Refresh, m.loadBookmarks)
			}
			return nil
		}
	}

	// Update the list
	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return cmd
}

func (m *Model) View() string {
	if !m.visible {
		return ""
	}

	// Apply border on the edge facing the main content
	// If panel is at bottom, border is on top; if on the right, border is on the left
	borderStyle := m.styles.border.
		Border(lipgloss.NormalBorder(), m.atBottom, false, false, !m.atBottom)

	title := fmt.Sprintf("Bookmarks (%d)", len(m.bookmarks))
	if m.moveMode && m.moveModeTarget != nil {
		title = fmt.Sprintf("Move bookmark to: %s", m.moveModeTarget.GetChangeId())
	} else if m.createMode {
		title = "Create bookmark: select revision"
	}

	header := m.styles.title.Render(title)

	var listView string
	// Show a message if no bookmarks
	if len(m.bookmarks) == 0 {
		listView = m.styles.empty.Render("No bookmarks found")
	} else {
		listView = m.List.View()
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		listView,
	)

	return borderStyle.Render(content)
}

func (m *Model) ShortHelp() []key.Binding {
	if m.moveMode {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "move bookmark")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel move")),
		}
	}
	if m.createMode {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select revision")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel create")),
		}
	}
	if m.List.SettingFilter() {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "apply filter")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		}
	}
	bindings := []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "view revset")),
		key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "create")),
		key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new revision")),
		key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "move")),
		key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "forget")),
		key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "track")),
		key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "untrack")),
	}
	if m.List.IsFiltered() {
		bindings = append(bindings, key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear filter")))
	} else {
		bindings = append(bindings, m.List.KeyMap.Filter)
		bindings = append(bindings, key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")))
	}
	return bindings
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

type StartMoveModeMsg struct {
	Bookmark *jj.Bookmark
}

type EndMoveModeMsg struct{}

type StartCreateModeMsg struct{}

type EndCreateModeMsg struct{}

func NewModel(c *context.MainContext) *Model {
	keymap := config.Current.GetKeyMap()
	styles := createStyles()

	delegate := bookmarkDelegate{
		styles: styles,
	}

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowFilter(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()
	// Hide filter input cursor when not actively filtering
	l.FilterInput.Cursor.SetMode(0) // CursorHide mode

	return &Model{
		context:       c,
		List:          l,
		visible:       false,
		focused:       false,
		keymap:        keymap,
		windowPercent: 50.0,
		autoPosition:  true,
		atBottom:      false, // Default to right side
		styles:        styles,
	}
}
