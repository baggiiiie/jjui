package status

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/idursun/jjui/internal/config"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/exec_process"
	"github.com/idursun/jjui/internal/ui/fuzzy_files"
	"github.com/idursun/jjui/internal/ui/fuzzy_input"
	"github.com/idursun/jjui/internal/ui/fuzzy_search"
)

var accept = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "accept"))

type commandStatus int

const (
	none commandStatus = iota
	commandRunning
	commandCompleted
	commandFailed
)

var _ common.Model = (*Model)(nil)

type Model struct {
	*common.ViewNode
	context    *context.MainContext
	spinner    spinner.Model
	input      textinput.Model
	keyMap     help.KeyMap
	command    string
	status     commandStatus
	running    bool
	mode       string
	editStatus editStatus
	history    map[string][]string
	fuzzy      fuzzy_search.Model
	styles     styles
}

type styles struct {
	shortcut lipgloss.Style
	dimmed   lipgloss.Style
	text     lipgloss.Style
	title    lipgloss.Style
	success  lipgloss.Style
	error    lipgloss.Style
}

// a function that will be used to show
// dynamic help when editing is focused.
type editStatus = func() (help.KeyMap, string)

func emptyEditStatus() (help.KeyMap, string) {
	return nil, ""
}

func (m *Model) IsFocused() bool {
	return m.editStatus != nil
}

func (m *Model) FuzzyView() string {
	if m.fuzzy == nil {
		return ""
	}
	return m.fuzzy.View()
}

const CommandClearDuration = 3 * time.Second

type clearMsg string

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	km := config.Current.GetKeyMap()
	switch msg := msg.(type) {
	case clearMsg:
		if m.command == string(msg) {
			m.command = ""
			m.status = none
		}
		return nil
	case common.CommandRunningMsg:
		m.command = string(msg)
		m.status = commandRunning
		return m.spinner.Tick
	case common.CommandCompletedMsg:
		if msg.Err != nil {
			m.status = commandFailed
		} else {
			m.status = commandCompleted
		}
		commandToBeCleared := m.command
		return tea.Tick(CommandClearDuration, func(time.Time) tea.Msg {
			return clearMsg(commandToBeCleared)
		})
	case common.FileSearchMsg:
		m.mode = "rev file"
		m.input.Prompt = "> "
		m.loadEditingSuggestions()
		m.fuzzy, m.editStatus = fuzzy_files.NewModel(msg)
		return tea.Batch(m.fuzzy.Init(), m.input.Focus())
	case common.ExecProcessCompletedMsg:
		if msg.Err != nil {
			m.mode = "exec " + msg.Msg.Mode.Mode
			m.input.Prompt = msg.Msg.Mode.Prompt
			m.loadEditingSuggestions()
			m.fuzzy, m.editStatus = fuzzy_input.NewModel(&m.input, m.input.AvailableSuggestions())

			// Avoid to change the current behavior when coming back from exec process
			focusCmd := m.input.Focus()
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(msg.Msg.Line)}
			updateCmd := m.Update(keyMsg)
			return tea.Batch(m.fuzzy.Init(), focusCmd, updateCmd)
		}
		return nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, km.Cancel) && m.IsFocused():
			var cmd tea.Cmd
			if m.fuzzy != nil {
				_, cmd = m.fuzzy.Update(msg)
			}

			m.fuzzy = nil
			m.editStatus = nil
			m.input.Reset()
			return cmd
		case key.Matches(msg, accept) && m.IsFocused():
			editMode := m.mode
			input := m.input.Value()
			prompt := m.input.Prompt
			fuzzy := m.fuzzy
			m.saveEditingSuggestions()

			m.fuzzy = nil
			m.command = ""
			m.editStatus = nil
			m.mode = ""
			m.input.Reset()

			switch {
			case strings.HasSuffix(editMode, "file"):
				_, cmd := fuzzy.Update(msg)
				return cmd
			case strings.HasPrefix(editMode, "exec"):
				return func() tea.Msg { return exec_process.ExecMsgFromLine(prompt, input) }
			}
			return func() tea.Msg { return common.QuickSearchMsg(input) }
		case key.Matches(msg, km.ExecJJ, km.ExecShell) && !m.IsFocused():
			mode := common.ExecJJ
			if key.Matches(msg, km.ExecShell) {
				mode = common.ExecShell
			}
			m.mode = "exec " + mode.Mode
			m.input.Prompt = mode.Prompt
			m.loadEditingSuggestions()

			m.fuzzy, m.editStatus = fuzzy_input.NewModel(&m.input, m.input.AvailableSuggestions())
			return tea.Batch(m.fuzzy.Init(), m.input.Focus())
		case key.Matches(msg, km.QuickSearch) && !m.IsFocused():
			m.editStatus = emptyEditStatus
			m.mode = "search"
			m.input.Prompt = "> "
			m.loadEditingSuggestions()
			return m.input.Focus()
		default:
			if m.IsFocused() {
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				if m.fuzzy != nil {
					cmd = tea.Batch(cmd, fuzzy_search.Search(m.input.Value(), msg))
				}
				return cmd
			}
		}
		return nil
	default:
		var cmd tea.Cmd
		if m.status == commandRunning {
			m.spinner, cmd = m.spinner.Update(msg)
		}
		if m.fuzzy != nil {
			m.fuzzy, cmd = fuzzy_search.Update(m.fuzzy, msg)
		}
		return cmd
	}
}

func (m *Model) saveEditingSuggestions() {
	input := m.input.Value()
	if len(strings.TrimSpace(input)) == 0 {
		return
	}
	h := m.context.Histories.GetHistory(config.HistoryKey(m.mode), true)
	h.Append(input)
}

func (m *Model) loadEditingSuggestions() {
	h := m.context.Histories.GetHistory(config.HistoryKey(m.mode), true)
	history := h.Entries()
	m.input.ShowSuggestions = true
	m.input.SetSuggestions([]string(history))
}

func (m *Model) View() string {
	modeWidth := max(10, len(m.mode)+2)
	availableWidth := m.Width - modeWidth - 1 // -1 for separator space

	var commandStatusMark string
	switch m.status {
	case commandRunning:
		commandStatusMark = m.styles.text.Render(m.spinner.View())
	case commandFailed:
		commandStatusMark = m.styles.error.Render("✗ ")
	case commandCompleted:
		commandStatusMark = m.styles.success.Render("✓ ")
	default:
		commandStatusMark = m.helpView(m.keyMap, availableWidth)
		commandStatusMark = lipgloss.PlaceHorizontal(availableWidth, 0, commandStatusMark, lipgloss.WithWhitespaceBackground(m.styles.text.GetBackground()))
	}

	ret := m.styles.text.Render(strings.ReplaceAll(m.command, "\n", "⏎"))
	if m.IsFocused() {
		commandStatusMark = ""
		editKeys, editHelp := m.editStatus()
		if editKeys != nil {
			editHelp = lipgloss.JoinHorizontal(0, m.helpView(editKeys, availableWidth), editHelp)
		}
		promptWidth := len(m.input.Prompt) + 2
		m.input.Width = m.Width - modeWidth - promptWidth - lipgloss.Width(editHelp)
		ret = lipgloss.JoinHorizontal(0, m.input.View(), editHelp)
	}

	// Calculate height of the help/status section
	helpHeight := lipgloss.Height(commandStatusMark)
	if helpHeight == 0 {
		helpHeight = 1
	}

	// Create mode indicator that spans the same height as help
	if helpHeight%2 == 0 {
		m.mode += "\nmode"
	}
	mode := m.styles.title.
		Width(modeWidth).Height(helpHeight).
		AlignVertical(lipgloss.Center).
		AlignHorizontal(lipgloss.Center).
		Render(m.mode)

	// Join mode with the rest of the content
	ret = lipgloss.JoinHorizontal(lipgloss.Left, mode, m.styles.text.Render(" "), commandStatusMark, ret)
	height := lipgloss.Height(ret)
	return lipgloss.Place(m.Width, height, 0, 0, ret, lipgloss.WithWhitespaceBackground(m.styles.text.GetBackground()))
}

func (m *Model) SetHelp(keyMap help.KeyMap) {
	m.keyMap = keyMap
}

func (m *Model) SetMode(mode string) {
	if !m.IsFocused() {
		m.mode = mode
	}
}

func (m *Model) helpView(keyMap help.KeyMap, availableWidth int) string {
	shortHelp := keyMap.ShortHelp()
	var entries []string
	for _, binding := range shortHelp {
		if !binding.Enabled() {
			continue
		}
		h := binding.Help()
		entries = append(entries, m.styles.shortcut.Render(h.Key)+m.styles.dimmed.PaddingLeft(1).Render(h.Desc))
	}

	separator := m.styles.dimmed.Render(" • ")

	// Build help text and check if it fits in one row
	var rows []string
	var currentRow []string
	currentWidth := 0

	for i, entry := range entries {
		entryWidth := lipgloss.Width(entry)
		sepWidth := 0
		if i > 0 {
			sepWidth = lipgloss.Width(separator)
		}

		// Check if adding this entry would exceed available width
		if currentWidth+sepWidth+entryWidth > availableWidth && len(currentRow) > 0 {
			// Start a new row
			rows = append(rows, strings.Join(currentRow, separator))
			currentRow = []string{entry}
			currentWidth = entryWidth
		} else {
			// Add to current row
			if len(currentRow) > 0 {
				currentWidth += sepWidth
			}
			currentRow = append(currentRow, entry)
			currentWidth += entryWidth
		}
	}

	// Add the last row
	if len(currentRow) > 0 {
		rows = append(rows, strings.Join(currentRow, separator))
	}

	// Join rows with newlines
	return strings.Join(rows, "\n")
}

func New(context *context.MainContext) *Model {
	styles := styles{
		shortcut: common.DefaultPalette.Get("status shortcut"),
		dimmed:   common.DefaultPalette.Get("status dimmed"),
		text:     common.DefaultPalette.Get("status text"),
		title:    common.DefaultPalette.Get("status title"),
		success:  common.DefaultPalette.Get("status success"),
		error:    common.DefaultPalette.Get("status error"),
	}
	s := spinner.New()
	s.Spinner = spinner.Dot

	t := textinput.New()
	t.Width = 50
	t.TextStyle = styles.text
	t.CompletionStyle = styles.dimmed
	t.PlaceholderStyle = styles.dimmed

	return &Model{
		ViewNode: common.NewViewNode(0, 0),
		context:  context,
		spinner:  s,
		command:  "",
		status:   none,
		input:    t,
		keyMap:   nil,
		styles:   styles,
	}
}
