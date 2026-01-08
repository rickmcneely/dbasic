package main

import (
	fmt "fmt"
	tea "github.com/charmbracelet/bubbletea"
	lipgloss "github.com/charmbracelet/lipgloss"
)

// Runtime helper functions

// Left returns the leftmost n characters
func Left(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if n >= len(s) {
		return s
	}
	return s[:n]
}

// Len returns the length of a string
func Len(s string) int {
	return len(s)
}

// Mid returns a substring starting at position start with length ln
func Mid(s string, start, ln int) string {
	if start < 1 {
		start = 1
	}
	startIdx := start - 1
	if startIdx >= len(s) {
		return ""
	}
	endIdx := startIdx + ln
	if endIdx > len(s) {
		endIdx = len(s)
	}
	return s[startIdx:endIdx]
}

// Chr returns the character for an ASCII code
func Chr(code int) string {
	return string(rune(code))
}

// Int converts to int
func Int(val interface{}) int {
	switch v := val.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

type EditorModel struct {
	CursorX int
	CursorY int
	Width int
	Height int
	Content string
	Message string
	ShowLineNumbers bool
	DialogMode int
	ScrollY int
	Modified bool
}


const DIALOG_NONE = 0
const DIALOG_HELP = 1
const DIALOG_ABOUT = 2
var (
	titleStyle lipgloss.Style
	statusStyle lipgloss.Style
	lineNumStyle lipgloss.Style
	textStyle lipgloss.Style
	cursorStyle lipgloss.Style
	dialogStyle lipgloss.Style
)


func PadRight(s string, width int) string {
	var result string = s
	for (len(result) < width) {
		result = (result + " ")
	}
	return result
}

func Spaces(count int) string {
	var result string = ""
	var i int
	for i = 1; i <= count; i += 1 {
		result = (result + " ")
	}
	return result
}

func InitStyles() {
	titleStyle = lipgloss.NewStyle().Background(lipgloss.Color("12")).Foreground(lipgloss.Color("15")).Bold(true)
	statusStyle = lipgloss.NewStyle().Background(lipgloss.Color("6")).Foreground(lipgloss.Color("0"))
	lineNumStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	textStyle = lipgloss.NewStyle().Background(lipgloss.Color("0")).Foreground(lipgloss.Color("15"))
	cursorStyle = lipgloss.NewStyle().Background(lipgloss.Color("15")).Foreground(lipgloss.Color("0"))
	dialogStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("12")).Padding(1, 2)
}

func CountLines(content string) int {
	if (len(content) == 0) {
		return 1
	}
	var count int = 1
	var i int
	for i = 1; i <= len(content); i += 1 {
		if (Mid(content, i, 1) == Chr(10)) {
			count = (count + 1)
		}
	}
	return count
}

func GetLine(content string, lineNum int) string {
	var currentLine int = 1
	var startPos int = 1
	var i int
	for i = 1; i <= len(content); i += 1 {
		if (currentLine == lineNum) {
			startPos = i
			break
		}
		if (Mid(content, i, 1) == Chr(10)) {
			currentLine = (currentLine + 1)
		}
	}
	if (currentLine < lineNum) {
		return ""
	}
	var result string = ""
	for i = startPos; i <= len(content); i += 1 {
		var ch string = Mid(content, i, 1)
		if (ch == Chr(10)) {
			break
		}
		result = (result + ch)
	}
	return result
}

func GetLineLength(content string, lineNum int) int {
	return len(GetLine(content, lineNum))
}

func (m EditorModel) Init() tea.Cmd {
	InitStyles()
	return nil
}

func (m EditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var sizeMsg tea.WindowSizeMsg
	var sizeOk bool
	sizeMsg, sizeOk = msg.(tea.WindowSizeMsg)
	if sizeOk {
		m.Width = Int(sizeMsg.Width)
		m.Height = Int(sizeMsg.Height)
		return m, nil
	}
	var keyMsg tea.KeyMsg
	var ok bool
	keyMsg, ok = msg.(tea.KeyMsg)
	if !(ok) {
		return m, nil
	}
	var keyStr string
	keyStr = keyMsg.String()
	if (m.DialogMode != DIALOG_NONE) {
		if (((keyStr == "esc") || (keyStr == "enter")) || (keyStr == "q")) {
			m.DialogMode = DIALOG_NONE
		}
		return m, nil
	}
	var totalLines int = CountLines(m.Content)
	var currentLineLen int = GetLineLength(m.Content, (m.CursorY + 1))
	if ((keyStr == "ctrl+c") || (keyStr == "ctrl+q")) {
		return m, tea.Quit
	} else if (keyStr == "f1") {
		m.DialogMode = DIALOG_HELP
		m.Message = "Help"
	} else if (keyStr == "f10") {
		m.DialogMode = DIALOG_ABOUT
		m.Message = "About"
	} else if (keyStr == "ctrl+l") {
		m.ShowLineNumbers = !(m.ShowLineNumbers)
		if m.ShowLineNumbers {
			m.Message = "Line numbers ON"
		} else {
			m.Message = "Line numbers OFF"
		}
	} else if (keyStr == "left") {
		if (m.CursorX > 0) {
			m.CursorX = (m.CursorX - 1)
		} else if (m.CursorY > 0) {
			m.CursorY = (m.CursorY - 1)
			m.CursorX = GetLineLength(m.Content, (m.CursorY + 1))
		}
		m.Message = ""
	} else if (keyStr == "right") {
		if (m.CursorX < currentLineLen) {
			m.CursorX = (m.CursorX + 1)
		} else if (m.CursorY < (totalLines - 1)) {
			m.CursorY = (m.CursorY + 1)
			m.CursorX = 0
		}
		m.Message = ""
	} else if (keyStr == "up") {
		if (m.CursorY > 0) {
			m.CursorY = (m.CursorY - 1)
			var upLineLen int = GetLineLength(m.Content, (m.CursorY + 1))
			if (m.CursorX > upLineLen) {
				m.CursorX = upLineLen
			}
		}
		m.Message = ""
	} else if (keyStr == "down") {
		if (m.CursorY < (totalLines - 1)) {
			m.CursorY = (m.CursorY + 1)
			var downLineLen int = GetLineLength(m.Content, (m.CursorY + 1))
			if (m.CursorX > downLineLen) {
				m.CursorX = downLineLen
			}
		}
		m.Message = ""
	} else if (keyStr == "home") {
		m.CursorX = 0
		m.Message = ""
	} else if (keyStr == "end") {
		m.CursorX = currentLineLen
		m.Message = ""
	} else if (keyStr == "ctrl+home") {
		m.CursorX = 0
		m.CursorY = 0
		m.ScrollY = 0
		m.Message = "Top of file"
	} else if (keyStr == "ctrl+end") {
		m.CursorY = (totalLines - 1)
		m.CursorX = GetLineLength(m.Content, totalLines)
		m.Message = "End of file"
	} else if (keyStr == "pgup") {
		var pgUpSize int = (m.Height - 4)
		if (pgUpSize < 1) {
			pgUpSize = 1
		}
		m.CursorY = (m.CursorY - pgUpSize)
		if (m.CursorY < 0) {
			m.CursorY = 0
		}
		m.Message = ""
	} else if (keyStr == "pgdown") {
		var pgDownSize int = (m.Height - 4)
		if (pgDownSize < 1) {
			pgDownSize = 1
		}
		m.CursorY = (m.CursorY + pgDownSize)
		if (m.CursorY >= totalLines) {
			m.CursorY = (totalLines - 1)
		}
		m.Message = ""
	} else {
		m.Message = ("Key: " + keyStr)
	}
	var visibleLines int = (m.Height - 4)
	if (visibleLines < 1) {
		visibleLines = 1
	}
	if (m.CursorY < m.ScrollY) {
		m.ScrollY = m.CursorY
	} else if (m.CursorY >= (m.ScrollY + visibleLines)) {
		m.ScrollY = ((m.CursorY - visibleLines) + 1)
	}
	return m, nil
}

func (m EditorModel) View() string {
	var view string = ""
	var totalLines int = CountLines(m.Content)
	var title string = " DBASIC EDIT "
	if m.Modified {
		title = (title + "[Modified] ")
	}
	var padding string = Spaces((m.Width - len(title)))
	view = (titleStyle.Render((title + padding)) + Chr(10))
	var contentHeight int = (m.Height - 3)
	if (contentHeight < 1) {
		contentHeight = 1
	}
	var lineNumWidth int = 0
	if m.ShowLineNumbers {
		lineNumWidth = 5
	}
	var i int
	for i = 0; i <= (contentHeight - 1); i += 1 {
		var lineIdx int = (m.ScrollY + i)
		var lineText string = ""
		if (lineIdx < totalLines) {
			lineText = GetLine(m.Content, (lineIdx + 1))
		}
		if m.ShowLineNumbers {
			if (lineIdx < totalLines) {
				view = (view + lineNumStyle.Render(fmt.Sprintf("%4d ", (lineIdx + 1))))
			} else {
				view = (view + lineNumStyle.Render("     "))
			}
		}
		if (lineIdx == m.CursorY) {
			var beforeCursor string = ""
			var cursorChar string = " "
			var afterCursor string = ""
			if ((m.CursorX > 0) && (len(lineText) > 0)) {
				if (m.CursorX <= len(lineText)) {
					beforeCursor = Left(lineText, m.CursorX)
				} else {
					beforeCursor = lineText
				}
			}
			if (m.CursorX < len(lineText)) {
				cursorChar = Mid(lineText, (m.CursorX + 1), 1)
				if ((m.CursorX + 1) < len(lineText)) {
					afterCursor = Mid(lineText, (m.CursorX + 2), ((len(lineText) - m.CursorX) - 1))
				}
			}
			view = (view + textStyle.Render(beforeCursor))
			view = (view + cursorStyle.Render(cursorChar))
			view = (view + textStyle.Render(afterCursor))
		} else {
			view = (view + textStyle.Render(lineText))
		}
		var currentLen int = (len(lineText) + lineNumWidth)
		if ((lineIdx == m.CursorY) && (m.CursorX >= len(lineText))) {
			currentLen = (currentLen + 1)
		}
		if (currentLen < m.Width) {
			view = (view + Spaces((m.Width - currentLen)))
		}
		view = (view + Chr(10))
	}
	var status string = fmt.Sprintf(" Ln %d, Col %d | Lines: %d", (m.CursorY + 1), (m.CursorX + 1), totalLines)
	if (len(m.Message) > 0) {
		status = ((status + " | ") + m.Message)
	}
	var statusPad string = Spaces((m.Width - len(status)))
	view = ((view + statusStyle.Render((status + statusPad))) + Chr(10))
	view = (view + " F1=Help  Ctrl+L=Line#  Ctrl+Q=Quit")
	if (m.DialogMode == DIALOG_HELP) {
		view = ((view + Chr(10)) + Chr(10))
		var helpText string = (("DBASIC EDIT - Keyboard Shortcuts" + Chr(10)) + Chr(10))
		helpText = ((helpText + "Navigation:") + Chr(10))
		helpText = ((helpText + "  Arrow keys    Move cursor") + Chr(10))
		helpText = ((helpText + "  Home/End      Start/end of line") + Chr(10))
		helpText = ((helpText + "  PgUp/PgDn     Page up/down") + Chr(10))
		helpText = ((helpText + "  Ctrl+Home     Start of file") + Chr(10))
		helpText = (((helpText + "  Ctrl+End      End of file") + Chr(10)) + Chr(10))
		helpText = ((helpText + "Commands:") + Chr(10))
		helpText = ((helpText + "  F1            This help") + Chr(10))
		helpText = ((helpText + "  F10           About") + Chr(10))
		helpText = ((helpText + "  Ctrl+L        Toggle line numbers") + Chr(10))
		helpText = (((helpText + "  Ctrl+Q/C      Quit") + Chr(10)) + Chr(10))
		helpText = (helpText + "Press any key to close")
		view = dialogStyle.Render(helpText)
	} else if (m.DialogMode == DIALOG_ABOUT) {
		view = ((view + Chr(10)) + Chr(10))
		var aboutText string = (("DBASIC EDIT" + Chr(10)) + Chr(10))
		aboutText = ((aboutText + "A text editor demonstration") + Chr(10))
		aboutText = ((aboutText + "written in DBasic using") + Chr(10))
		aboutText = (((aboutText + "the Bubble Tea TUI framework.") + Chr(10)) + Chr(10))
		aboutText = ((aboutText + "This showcases DBasic's ability") + Chr(10))
		aboutText = ((aboutText + "to implement Go interfaces and") + Chr(10))
		aboutText = (((aboutText + "use external packages.") + Chr(10)) + Chr(10))
		aboutText = (aboutText + "Press any key to close")
		view = dialogStyle.Render(aboutText)
	}
	return view
}

func Main() {
	var model EditorModel
	model.CursorX = 0
	model.CursorY = 0
	model.Width = 80
	model.Height = 24
	model.ScrollY = 0
	model.ShowLineNumbers = true
	model.DialogMode = DIALOG_NONE
	model.Modified = false
	model.Message = "Welcome to DBasic Edit!"
	model.Content = ("' Welcome to DBASIC EDIT" + Chr(10))
	model.Content = ((model.Content + "' A simple text editor written in DBasic") + Chr(10))
	model.Content = (model.Content + Chr(10))
	model.Content = ((model.Content + "SUB Main()") + Chr(10))
	model.Content = ((model.Content + "    PRINT \"Hello, World!\"") + Chr(10))
	model.Content = ((model.Content + "END SUB") + Chr(10))
	model.Content = (model.Content + Chr(10))
	model.Content = ((model.Content + "' Use arrow keys to navigate") + Chr(10))
	model.Content = ((model.Content + "' Press F1 for help") + Chr(10))
	model.Content = (model.Content + "' Press Ctrl+Q to quit")
	tea.NewProgram(model, tea.WithAltScreen()).Run()
}

func main() {
	Main()
}
