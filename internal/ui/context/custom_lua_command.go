package context

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/common"
)

// CustomLuaCommand executes a Lua script via the capability bridge.
type CustomLuaCommand struct {
	CustomCommandBase
	Script  string `toml:"lua"`
	LuaFile string `toml:"lua_file"`
}

func (c *CustomLuaCommand) LoadFromFile(configDir string) error {
	if c.LuaFile == "" {
		return nil
	}

	var filePath string
	if filepath.IsAbs(c.LuaFile) {
		filePath = c.LuaFile
	} else {
		filePath = filepath.Join(configDir, c.LuaFile)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	c.Script = string(content)
	return nil
}

func (c CustomLuaCommand) IsApplicableTo(item SelectedItem) bool {
	return true
}

func (c CustomLuaCommand) Description(ctx *MainContext) string {
	return fmt.Sprintf("lua: %s", c.Script)
}

func (c CustomLuaCommand) Prepare(ctx *MainContext) tea.Cmd {
	return func() tea.Msg {
		return common.RunLuaScriptMsg{Script: c.Script}
	}
}
