package main

import (
	"os"
	"strconv"
	fmt "fmt"
	tea "github.com/charmbracelet/bubbletea"
	lipgloss "github.com/charmbracelet/lipgloss"
	"strings"
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

// ReadFile reads entire file contents
func ReadFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// Instr finds the position of substring in string (1-based)
func Instr(s, substr string) int {
	idx := strings.Index(s, substr)
	if idx == -1 {
		return 0
	}
	return idx + 1
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

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// WriteFile writes string to file
func WriteFile(path, content string) {
	os.WriteFile(path, []byte(content), 0644)
}

// Val converts a string to float64
func Val(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

type EditorModel struct {
	CursorX int
	CursorY int
	Width int
	Height int
	Content string
	LineCount int
	ScrollX int
	ScrollY int
	SelectStartX int
	SelectStartY int
	SelectEndX int
	SelectEndY int
	Selecting bool
	Clipboard string
	Filename string
	Modified bool
	MenuOpen int
	MenuIndex int
	DialogMode int
	DialogInput string
	DialogCursor int
	SearchText string
	ReplaceText string
	SearchWrap bool
	SearchCase bool
	Message string
	ShowLineNumbers bool
	TabSize int
	InsertMode bool
}


const MENU_NONE = 0
const MENU_FILE = 1
const MENU_EDIT = 2
const MENU_SEARCH = 3
const MENU_OPTIONS = 4
const MENU_HELP = 5
const DIALOG_NONE = 0
const DIALOG_HELP = 1
const DIALOG_ABOUT = 2
const DIALOG_OPEN = 3
const DIALOG_SAVE = 4
const DIALOG_SAVEAS = 5
const DIALOG_FIND = 6
const DIALOG_REPLACE = 7
const DIALOG_GOTO = 8
const DIALOG_CONFIRM_NEW = 9
const DIALOG_CONFIRM_EXIT = 10
var (
	menuBarStyle lipgloss.Style
	menuItemStyle lipgloss.Style
	menuItemSelectedStyle lipgloss.Style
	menuHotkeyStyle lipgloss.Style
	menuHotkeySelectedStyle lipgloss.Style
	menuDropdownStyle lipgloss.Style
	textAreaStyle lipgloss.Style
	statusBarStyle lipgloss.Style
	dialogStyle lipgloss.Style
	dialogTitleStyle lipgloss.Style
	lineNumStyle lipgloss.Style
	cursorStyle lipgloss.Style
	selectedStyle lipgloss.Style
)


func InitStyles() {
	menuBarStyle = lipgloss.NewStyle().Background(lipgloss.Color("7")).Foreground(lipgloss.Color("0"))
	menuItemStyle = lipgloss.NewStyle().Background(lipgloss.Color("7")).Foreground(lipgloss.Color("0"))
	menuItemSelectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("0")).Foreground(lipgloss.Color("7"))
	menuHotkeyStyle = lipgloss.NewStyle().Background(lipgloss.Color("7")).Foreground(lipgloss.Color("1"))
	menuHotkeySelectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("0")).Foreground(lipgloss.Color("9"))
	menuDropdownStyle = lipgloss.NewStyle().Background(lipgloss.Color("4")).Foreground(lipgloss.Color("15")).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("15"))
	textAreaStyle = lipgloss.NewStyle().Background(lipgloss.Color("4")).Foreground(lipgloss.Color("15"))
	statusBarStyle = lipgloss.NewStyle().Background(lipgloss.Color("6")).Foreground(lipgloss.Color("0"))
	dialogStyle = lipgloss.NewStyle().Background(lipgloss.Color("7")).Foreground(lipgloss.Color("0")).Border(lipgloss.DoubleBorder()).BorderForeground(lipgloss.Color("0")).Padding(1, 2)
	dialogTitleStyle = lipgloss.NewStyle().Background(lipgloss.Color("4")).Foreground(lipgloss.Color("15")).Padding(0, 1)
	lineNumStyle = lipgloss.NewStyle().Background(lipgloss.Color("4")).Foreground(lipgloss.Color("8"))
	cursorStyle = lipgloss.NewStyle().Background(lipgloss.Color("15")).Foreground(lipgloss.Color("4"))
	selectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("3")).Foreground(lipgloss.Color("0"))
}

func RepeatChar(ch string, count int) string {
	var result string = ""
	var i int
	for i = 1; i <= count; i += 1 {
		result = (result + ch)
	}
	return result
}

func PadRight(s string, width int) string {
	if (len(s) >= width) {
		return Left(s, width)
	}
	return (s + RepeatChar(" ", (width - len(s))))
}

func CenterStr(s string, width int) string {
	if (len(s) >= width) {
		return Left(s, width)
	}
	var padding int = ((width - len(s)) / 2)
	return ((RepeatChar(" ", padding) + s) + RepeatChar(" ", ((width - len(s)) - padding)))
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
	if (len(content) == 0) {
		return ""
	}
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

func SetLine(content string, lineNum int, newLine string) string {
	var lines string = ""
	var currentLine int = 1
	var i int
	var lineStart int = 1
	for i = 1; i <= (len(content) + 1); i += 1 {
		var ch string = ""
		if (i <= len(content)) {
			ch = Mid(content, i, 1)
		}
		if ((ch == Chr(10)) || (i > len(content))) {
			if (currentLine == lineNum) {
				lines = (lines + newLine)
			} else {
				lines = (lines + Mid(content, lineStart, (i - lineStart)))
			}
			if (ch == Chr(10)) {
				lines = (lines + Chr(10))
			}
			currentLine = (currentLine + 1)
			lineStart = (i + 1)
		}
	}
	return lines
}

func InsertCharAt(content string, lineNum int, col int, ch string) string {
	var line string = GetLine(content, lineNum)
	var newLine string
	if (col <= 0) {
		newLine = (ch + line)
	} else if (col >= len(line)) {
		newLine = (line + ch)
	} else {
		newLine = ((Left(line, col) + ch) + Mid(line, (col + 1), (len(line) - col)))
	}
	return SetLine(content, lineNum, newLine)
}

func DeleteCharAt(content string, lineNum int, col int) string {
	var line string = GetLine(content, lineNum)
	if ((col < 0) || (col >= len(line))) {
		return content
	}
	var newLine string
	if (col == 0) {
		newLine = Mid(line, 2, (len(line) - 1))
	} else {
		newLine = (Left(line, col) + Mid(line, (col + 2), ((len(line) - col) - 1)))
	}
	return SetLine(content, lineNum, newLine)
}

func InsertNewLine(content string, lineNum int, col int) string {
	var line string = GetLine(content, lineNum)
	var beforeCursor string = Left(line, col)
	var afterCursor string = Mid(line, (col + 1), (len(line) - col))
	var result string = ""
	var currentLine int = 1
	var i int
	var lineStart int = 1
	for i = 1; i <= (len(content) + 1); i += 1 {
		var ch string = ""
		if (i <= len(content)) {
			ch = Mid(content, i, 1)
		}
		if ((ch == Chr(10)) || (i > len(content))) {
			if (currentLine == lineNum) {
				result = (((result + beforeCursor) + Chr(10)) + afterCursor)
			} else {
				result = (result + Mid(content, lineStart, (i - lineStart)))
			}
			if ((ch == Chr(10)) && (currentLine != lineNum)) {
				result = (result + Chr(10))
			}
			currentLine = (currentLine + 1)
			lineStart = (i + 1)
		}
	}
	return result
}

func JoinWithPrevLine(content string, lineNum int) string {
	if (lineNum <= 1) {
		return content
	}
	var prevLine string = GetLine(content, (lineNum - 1))
	var currLine string = GetLine(content, lineNum)
	var joinedLine string = (prevLine + currLine)
	var result string = ""
	var currentLine int = 1
	var i int
	var lineStart int = 1
	for i = 1; i <= (len(content) + 1); i += 1 {
		var ch string = ""
		if (i <= len(content)) {
			ch = Mid(content, i, 1)
		}
		if ((ch == Chr(10)) || (i > len(content))) {
			if (currentLine == (lineNum - 1)) {
				result = (result + joinedLine)
			} else if (currentLine != lineNum) {
				result = (result + Mid(content, lineStart, (i - lineStart)))
			}
			if (((ch == Chr(10)) && (currentLine != (lineNum - 1))) && (currentLine != lineNum)) {
				result = (result + Chr(10))
			} else if ((ch == Chr(10)) && (currentLine == (lineNum - 1))) {
			} else if ((ch == Chr(10)) && (currentLine < (lineNum - 1))) {
				result = (result + Chr(10))
			}
			currentLine = (currentLine + 1)
			lineStart = (i + 1)
		}
	}
	return result
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
	var key string = keyMsg.String()
	if (m.DialogMode != DIALOG_NONE) {
		return HandleDialogInput(m, key)
	}
	if (m.MenuOpen != MENU_NONE) {
		return HandleMenuInput(m, key)
	}
	if ((key == "alt+f") || (key == "f10")) {
		m.MenuOpen = MENU_FILE
		m.MenuIndex = 0
		return m, nil
	} else if (key == "alt+e") {
		m.MenuOpen = MENU_EDIT
		m.MenuIndex = 0
		return m, nil
	} else if (key == "alt+s") {
		m.MenuOpen = MENU_SEARCH
		m.MenuIndex = 0
		return m, nil
	} else if (key == "alt+o") {
		m.MenuOpen = MENU_OPTIONS
		m.MenuIndex = 0
		return m, nil
	} else if ((key == "alt+h") || (key == "f1")) {
		m.DialogMode = DIALOG_HELP
		return m, nil
	} else if (key == "ctrl+n") {
		return DoNewFile(m)
	} else if (key == "ctrl+o") {
		m.DialogMode = DIALOG_OPEN
		m.DialogInput = m.Filename
		m.DialogCursor = len(m.DialogInput)
		return m, nil
	} else if (key == "ctrl+s") {
		return DoSaveFile(m)
	} else if ((key == "ctrl+q") || (key == "alt+x")) {
		if m.Modified {
			m.DialogMode = DIALOG_CONFIRM_EXIT
			return m, nil
		}
		return m, tea.Quit
	} else if (key == "ctrl+f") {
		m.DialogMode = DIALOG_FIND
		m.DialogInput = m.SearchText
		m.DialogCursor = len(m.DialogInput)
		return m, nil
	} else if (key == "ctrl+h") {
		m.DialogMode = DIALOG_REPLACE
		m.DialogInput = m.SearchText
		m.DialogCursor = len(m.DialogInput)
		return m, nil
	} else if (key == "f3") {
		return DoFindNext(m)
	} else if (key == "ctrl+g") {
		m.DialogMode = DIALOG_GOTO
		m.DialogInput = ""
		m.DialogCursor = 0
		return m, nil
	} else if (key == "ctrl+c") {
		return DoCopy(m)
	} else if (key == "ctrl+x") {
		return DoCut(m)
	} else if (key == "ctrl+v") {
		return DoPaste(m)
	} else if (key == "ctrl+a") {
		return DoSelectAll(m)
	} else if (key == "insert") {
		m.InsertMode = !(m.InsertMode)
		if m.InsertMode {
			m.Message = "Insert mode"
		} else {
			m.Message = "Overwrite mode"
		}
		return m, nil
	}
	return HandleEditorInput(m, key)
}

func StartSelection(m *EditorModel) {
	if !((*m).Selecting) {
		(*m).Selecting = true
		(*m).SelectStartX = (*m).CursorX
		(*m).SelectStartY = (*m).CursorY
	}
}

func ClearSelection(m *EditorModel) {
	(*m).Selecting = false
}

func HandleEditorInput(m EditorModel, key string) (tea.Model, tea.Cmd) {
	var totalLines int = CountLines(m.Content)
	var currentLineLen int = GetLineLength(m.Content, (m.CursorY + 1))
	if (key == "shift+left") {
		StartSelection(&m)
		if (m.CursorX > 0) {
			m.CursorX = (m.CursorX - 1)
		} else if (m.CursorY > 0) {
			m.CursorY = (m.CursorY - 1)
			m.CursorX = GetLineLength(m.Content, (m.CursorY + 1))
		}
		m.SelectEndX = m.CursorX
		m.SelectEndY = m.CursorY
		return m, nil
	} else if (key == "shift+right") {
		StartSelection(&m)
		if (m.CursorX < currentLineLen) {
			m.CursorX = (m.CursorX + 1)
		} else if (m.CursorY < (totalLines - 1)) {
			m.CursorY = (m.CursorY + 1)
			m.CursorX = 0
		}
		m.SelectEndX = m.CursorX
		m.SelectEndY = m.CursorY
		return m, nil
	} else if (key == "shift+up") {
		StartSelection(&m)
		if (m.CursorY > 0) {
			m.CursorY = (m.CursorY - 1)
			var upLen int = GetLineLength(m.Content, (m.CursorY + 1))
			if (m.CursorX > upLen) {
				m.CursorX = upLen
			}
		}
		m.SelectEndX = m.CursorX
		m.SelectEndY = m.CursorY
		return m, nil
	} else if (key == "shift+down") {
		StartSelection(&m)
		if (m.CursorY < (totalLines - 1)) {
			m.CursorY = (m.CursorY + 1)
			var downLen int = GetLineLength(m.Content, (m.CursorY + 1))
			if (m.CursorX > downLen) {
				m.CursorX = downLen
			}
		}
		m.SelectEndX = m.CursorX
		m.SelectEndY = m.CursorY
		return m, nil
	} else if (key == "shift+home") {
		StartSelection(&m)
		m.CursorX = 0
		m.SelectEndX = m.CursorX
		m.SelectEndY = m.CursorY
		return m, nil
	} else if (key == "shift+end") {
		StartSelection(&m)
		m.CursorX = currentLineLen
		m.SelectEndX = m.CursorX
		m.SelectEndY = m.CursorY
		return m, nil
	}
	if (key == "left") {
		ClearSelection(&m)
		if (m.CursorX > 0) {
			m.CursorX = (m.CursorX - 1)
		} else if (m.CursorY > 0) {
			m.CursorY = (m.CursorY - 1)
			m.CursorX = GetLineLength(m.Content, (m.CursorY + 1))
		}
		m.Message = ""
	} else if (key == "right") {
		ClearSelection(&m)
		if (m.CursorX < currentLineLen) {
			m.CursorX = (m.CursorX + 1)
		} else if (m.CursorY < (totalLines - 1)) {
			m.CursorY = (m.CursorY + 1)
			m.CursorX = 0
		}
		m.Message = ""
	} else if (key == "up") {
		ClearSelection(&m)
		if (m.CursorY > 0) {
			m.CursorY = (m.CursorY - 1)
			var upLineLen int = GetLineLength(m.Content, (m.CursorY + 1))
			if (m.CursorX > upLineLen) {
				m.CursorX = upLineLen
			}
		}
		m.Message = ""
	} else if (key == "down") {
		ClearSelection(&m)
		if (m.CursorY < (totalLines - 1)) {
			m.CursorY = (m.CursorY + 1)
			var downLineLen int = GetLineLength(m.Content, (m.CursorY + 1))
			if (m.CursorX > downLineLen) {
				m.CursorX = downLineLen
			}
		}
		m.Message = ""
	} else if (key == "home") {
		ClearSelection(&m)
		m.CursorX = 0
		m.Message = ""
	} else if (key == "end") {
		ClearSelection(&m)
		m.CursorX = currentLineLen
		m.Message = ""
	} else if (key == "ctrl+home") {
		m.CursorX = 0
		m.CursorY = 0
		m.ScrollY = 0
		m.Message = ""
	} else if (key == "ctrl+end") {
		m.CursorY = (totalLines - 1)
		m.CursorX = GetLineLength(m.Content, totalLines)
		m.Message = ""
	} else if (key == "pgup") {
		var pgSize int = (m.Height - 4)
		if (pgSize < 1) {
			pgSize = 1
		}
		m.CursorY = (m.CursorY - pgSize)
		if (m.CursorY < 0) {
			m.CursorY = 0
		}
		m.Message = ""
	} else if (key == "pgdown") {
		var pgDownSize int = (m.Height - 4)
		if (pgDownSize < 1) {
			pgDownSize = 1
		}
		m.CursorY = (m.CursorY + pgDownSize)
		if (m.CursorY >= totalLines) {
			m.CursorY = (totalLines - 1)
		}
		m.Message = ""
	} else if (key == "enter") {
		m.Content = InsertNewLine(m.Content, (m.CursorY + 1), m.CursorX)
		m.CursorY = (m.CursorY + 1)
		m.CursorX = 0
		m.Modified = true
		m.Message = ""
	} else if (key == "backspace") {
		if (m.CursorX > 0) {
			m.Content = DeleteCharAt(m.Content, (m.CursorY + 1), (m.CursorX - 1))
			m.CursorX = (m.CursorX - 1)
			m.Modified = true
		} else if (m.CursorY > 0) {
			var prevLen int = GetLineLength(m.Content, m.CursorY)
			m.Content = JoinWithPrevLine(m.Content, (m.CursorY + 1))
			m.CursorY = (m.CursorY - 1)
			m.CursorX = prevLen
			m.Modified = true
		}
		m.Message = ""
	} else if (key == "delete") {
		if (m.CursorX < currentLineLen) {
			m.Content = DeleteCharAt(m.Content, (m.CursorY + 1), m.CursorX)
			m.Modified = true
		} else if (m.CursorY < (totalLines - 1)) {
			var nextLine string = GetLine(m.Content, (m.CursorY + 2))
			var currLine string = GetLine(m.Content, (m.CursorY + 1))
			m.Content = SetLine(m.Content, (m.CursorY + 1), (currLine + nextLine))
			var newContent string = ""
			var lineNum int = 1
			var total int = CountLines(m.Content)
			for lineNum = 1; lineNum <= total; lineNum += 1 {
				if (lineNum != (m.CursorY + 2)) {
					if (len(newContent) > 0) {
						newContent = (newContent + Chr(10))
					}
					newContent = (newContent + GetLine(m.Content, lineNum))
				}
			}
			m.Content = newContent
			m.Modified = true
		}
		m.Message = ""
	} else if (key == "tab") {
		var ti int
		for ti = 1; ti <= m.TabSize; ti += 1 {
			m.Content = InsertCharAt(m.Content, (m.CursorY + 1), m.CursorX, " ")
			m.CursorX = (m.CursorX + 1)
		}
		m.Modified = true
		m.Message = ""
	} else if (key == "esc") {
		m.Selecting = false
		m.Message = ""
	} else if (len(key) == 1) {
		if (m.InsertMode || (m.CursorX >= currentLineLen)) {
			m.Content = InsertCharAt(m.Content, (m.CursorY + 1), m.CursorX, key)
		} else {
			m.Content = DeleteCharAt(m.Content, (m.CursorY + 1), m.CursorX)
			m.Content = InsertCharAt(m.Content, (m.CursorY + 1), m.CursorX, key)
		}
		m.CursorX = (m.CursorX + 1)
		m.Modified = true
		m.Message = ""
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
	var visibleCols int = (m.Width - 8)
	if m.ShowLineNumbers {
		visibleCols = (visibleCols - 6)
	}
	if (visibleCols < 10) {
		visibleCols = 10
	}
	if (m.CursorX < m.ScrollX) {
		m.ScrollX = m.CursorX
	} else if (m.CursorX >= (m.ScrollX + visibleCols)) {
		m.ScrollX = ((m.CursorX - visibleCols) + 1)
	}
	return m, nil
}

func HandleMenuInput(m EditorModel, key string) (tea.Model, tea.Cmd) {
	if (key == "esc") {
		m.MenuOpen = MENU_NONE
		return m, nil
	} else if (key == "left") {
		m.MenuOpen = (m.MenuOpen - 1)
		if (m.MenuOpen < MENU_FILE) {
			m.MenuOpen = MENU_HELP
		}
		m.MenuIndex = 0
		return m, nil
	} else if (key == "right") {
		m.MenuOpen = (m.MenuOpen + 1)
		if (m.MenuOpen > MENU_HELP) {
			m.MenuOpen = MENU_FILE
		}
		m.MenuIndex = 0
		return m, nil
	} else if (key == "up") {
		m.MenuIndex = (m.MenuIndex - 1)
		if (m.MenuIndex < 0) {
			m.MenuIndex = (GetMenuItemCount(m.MenuOpen) - 1)
		}
		return m, nil
	} else if (key == "down") {
		m.MenuIndex = (m.MenuIndex + 1)
		if (m.MenuIndex >= GetMenuItemCount(m.MenuOpen)) {
			m.MenuIndex = 0
		}
		return m, nil
	} else if (key == "enter") {
		return ExecuteMenuItem(m)
	}
	return m, nil
}

func GetMenuItemCount(menu int) int {
	if (menu == MENU_FILE) {
		return 6
	} else if (menu == MENU_EDIT) {
		return 6
	} else if (menu == MENU_SEARCH) {
		return 4
	} else if (menu == MENU_OPTIONS) {
		return 2
	} else if (menu == MENU_HELP) {
		return 2
	}
	return 0
}

func ExecuteMenuItem(m EditorModel) (tea.Model, tea.Cmd) {
	var currentMenu int = m.MenuOpen
	m.MenuOpen = MENU_NONE
	if (currentMenu == MENU_FILE) {
		if (m.MenuIndex == 0) {
			return DoNewFile(m)
		} else if (m.MenuIndex == 1) {
			m.DialogMode = DIALOG_OPEN
			m.DialogInput = m.Filename
			m.DialogCursor = len(m.DialogInput)
		} else if (m.MenuIndex == 2) {
			return DoSaveFile(m)
		} else if (m.MenuIndex == 3) {
			m.DialogMode = DIALOG_SAVEAS
			m.DialogInput = m.Filename
			m.DialogCursor = len(m.DialogInput)
		} else if (m.MenuIndex == 5) {
			if m.Modified {
				m.DialogMode = DIALOG_CONFIRM_EXIT
			} else {
				return m, tea.Quit
			}
		}
	} else if (currentMenu == MENU_EDIT) {
		if (m.MenuIndex == 0) {
			return DoCut(m)
		} else if (m.MenuIndex == 1) {
			return DoCopy(m)
		} else if (m.MenuIndex == 2) {
			return DoPaste(m)
		} else if (m.MenuIndex == 4) {
			return DoSelectAll(m)
		}
	} else if (currentMenu == MENU_SEARCH) {
		if (m.MenuIndex == 0) {
			m.DialogMode = DIALOG_FIND
			m.DialogInput = m.SearchText
			m.DialogCursor = len(m.DialogInput)
		} else if (m.MenuIndex == 1) {
			return DoFindNext(m)
		} else if (m.MenuIndex == 2) {
			m.DialogMode = DIALOG_REPLACE
			m.DialogInput = m.SearchText
			m.DialogCursor = len(m.DialogInput)
		} else if (m.MenuIndex == 3) {
			m.DialogMode = DIALOG_GOTO
			m.DialogInput = ""
			m.DialogCursor = 0
		}
	} else if (currentMenu == MENU_OPTIONS) {
		if (m.MenuIndex == 0) {
			m.ShowLineNumbers = !(m.ShowLineNumbers)
		} else if (m.MenuIndex == 1) {
			m.InsertMode = !(m.InsertMode)
		}
	} else if (currentMenu == MENU_HELP) {
		if (m.MenuIndex == 0) {
			m.DialogMode = DIALOG_HELP
		} else if (m.MenuIndex == 1) {
			m.DialogMode = DIALOG_ABOUT
		}
	}
	return m, nil
}

func HandleDialogInput(m EditorModel, key string) (tea.Model, tea.Cmd) {
	if (key == "esc") {
		m.DialogMode = DIALOG_NONE
		return m, nil
	}
	if ((m.DialogMode == DIALOG_HELP) || (m.DialogMode == DIALOG_ABOUT)) {
		if (((key == "enter") || (key == "esc")) || (key == " ")) {
			m.DialogMode = DIALOG_NONE
		}
		return m, nil
	}
	if ((m.DialogMode == DIALOG_CONFIRM_NEW) || (m.DialogMode == DIALOG_CONFIRM_EXIT)) {
		if ((key == "y") || (key == "Y")) {
			if (m.DialogMode == DIALOG_CONFIRM_EXIT) {
				return m, tea.Quit
			} else {
				m.Content = ""
				m.Filename = ""
				m.Modified = false
				m.CursorX = 0
				m.CursorY = 0
				m.ScrollX = 0
				m.ScrollY = 0
				m.DialogMode = DIALOG_NONE
				m.Message = "New file"
			}
		} else if (((key == "n") || (key == "N")) || (key == "esc")) {
			m.DialogMode = DIALOG_NONE
		}
		return m, nil
	}
	if (key == "enter") {
		if (m.DialogMode == DIALOG_OPEN) {
			return DoOpenFile(m, m.DialogInput)
		} else if ((m.DialogMode == DIALOG_SAVE) || (m.DialogMode == DIALOG_SAVEAS)) {
			m.Filename = m.DialogInput
			return DoSaveFile(m)
		} else if (m.DialogMode == DIALOG_FIND) {
			m.SearchText = m.DialogInput
			m.DialogMode = DIALOG_NONE
			return DoFindNext(m)
		} else if (m.DialogMode == DIALOG_REPLACE) {
			m.SearchText = m.DialogInput
			m.DialogMode = DIALOG_NONE
			return DoReplace(m)
		} else if (m.DialogMode == DIALOG_GOTO) {
			return DoGotoLine(m)
		}
	} else if (key == "backspace") {
		if (m.DialogCursor > 0) {
			m.DialogInput = (Left(m.DialogInput, (m.DialogCursor - 1)) + Mid(m.DialogInput, (m.DialogCursor + 1), (len(m.DialogInput) - m.DialogCursor)))
			m.DialogCursor = (m.DialogCursor - 1)
		}
	} else if (key == "delete") {
		if (m.DialogCursor < len(m.DialogInput)) {
			m.DialogInput = (Left(m.DialogInput, m.DialogCursor) + Mid(m.DialogInput, (m.DialogCursor + 2), ((len(m.DialogInput) - m.DialogCursor) - 1)))
		}
	} else if (key == "left") {
		if (m.DialogCursor > 0) {
			m.DialogCursor = (m.DialogCursor - 1)
		}
	} else if (key == "right") {
		if (m.DialogCursor < len(m.DialogInput)) {
			m.DialogCursor = (m.DialogCursor + 1)
		}
	} else if (key == "home") {
		m.DialogCursor = 0
	} else if (key == "end") {
		m.DialogCursor = len(m.DialogInput)
	} else if (len(key) == 1) {
		m.DialogInput = ((Left(m.DialogInput, m.DialogCursor) + key) + Mid(m.DialogInput, (m.DialogCursor + 1), (len(m.DialogInput) - m.DialogCursor)))
		m.DialogCursor = (m.DialogCursor + 1)
	}
	return m, nil
}

func DoNewFile(m EditorModel) (tea.Model, tea.Cmd) {
	if m.Modified {
		m.DialogMode = DIALOG_CONFIRM_NEW
		return m, nil
	}
	m.Content = ""
	m.Filename = ""
	m.Modified = false
	m.CursorX = 0
	m.CursorY = 0
	m.ScrollX = 0
	m.ScrollY = 0
	m.Message = "New file"
	return m, nil
}

func DoOpenFile(m EditorModel, filename string) (tea.Model, tea.Cmd) {
	m.DialogMode = DIALOG_NONE
	if (len(filename) == 0) {
		m.Message = "No filename specified"
		return m, nil
	}
	var content string = ReadFile(filename)
	if ((len(content) == 0) && !(FileExists(filename))) {
		m.Message = ("File not found: " + filename)
		return m, nil
	}
	m.Content = content
	m.Filename = filename
	m.Modified = false
	m.CursorX = 0
	m.CursorY = 0
	m.ScrollX = 0
	m.ScrollY = 0
	m.Message = ("Opened: " + filename)
	return m, nil
}

func DoSaveFile(m EditorModel) (tea.Model, tea.Cmd) {
	if (len(m.Filename) == 0) {
		m.DialogMode = DIALOG_SAVEAS
		m.DialogInput = ""
		m.DialogCursor = 0
		return m, nil
	}
	m.DialogMode = DIALOG_NONE
	WriteFile(m.Filename, m.Content)
	m.Modified = false
	m.Message = ("Saved: " + m.Filename)
	return m, nil
}

func IsInSelection(m EditorModel, row int, col int) bool {
	if !(m.Selecting) {
		return false
	}
	var startY int
	var startX int
	var endY int
	var endX int
	startY, startX, endY, endX = GetSelectionBounds(m)
	if ((row < startY) || (row > endY)) {
		return false
	}
	if ((row == startY) && (row == endY)) {
		return ((col >= startX) && (col < endX))
	} else if (row == startY) {
		return (col >= startX)
	} else if (row == endY) {
		return (col < endX)
	} else {
		return true
	}
}

func GetSelectionBounds(m EditorModel) (int, int, int, int) {
	var startY int = m.SelectStartY
	var startX int = m.SelectStartX
	var endY int = m.SelectEndY
	var endX int = m.SelectEndX
	if ((endY < startY) || ((endY == startY) && (endX < startX))) {
		var tmpY int = startY
		var tmpX int = startX
		startY = endY
		startX = endX
		endY = tmpY
		endX = tmpX
	}
	return startY, startX, endY, endX
}

func GetSelectedText(m EditorModel) string {
	if !(m.Selecting) {
		return ""
	}
	var startY int
	var startX int
	var endY int
	var endX int
	startY, startX, endY, endX = GetSelectionBounds(m)
	var result string = ""
	var y int
	for y = startY; y <= endY; y += 1 {
		var line string = GetLine(m.Content, (y + 1))
		var lineStart int = 0
		var lineEnd int = len(line)
		if (y == startY) {
			lineStart = startX
		}
		if (y == endY) {
			lineEnd = endX
		}
		if (lineEnd > lineStart) {
			result = (result + Mid(line, (lineStart + 1), (lineEnd - lineStart)))
		}
		if (y < endY) {
			result = (result + Chr(10))
		}
	}
	return result
}

func DeleteSelection(m *EditorModel) (int, int) {
	if !((*m).Selecting) {
		return (*m).CursorX, (*m).CursorY
	}
	var startY int
	var startX int
	var endY int
	var endX int
	startY, startX, endY, endX = GetSelectionBounds((*m))
	var totalLines int = CountLines((*m).Content)
	var newContent string = ""
	var y int
	for y = 0; y <= (totalLines - 1); y += 1 {
		var line string = GetLine((*m).Content, (y + 1))
		if ((y < startY) || (y > endY)) {
			if (len(newContent) > 0) {
				newContent = (newContent + Chr(10))
			}
			newContent = (newContent + line)
		} else if ((y == startY) && (y == endY)) {
			if (len(newContent) > 0) {
				newContent = (newContent + Chr(10))
			}
			newContent = ((newContent + Left(line, startX)) + Mid(line, (endX + 1), (len(line) - endX)))
		} else if (y == startY) {
			if (len(newContent) > 0) {
				newContent = (newContent + Chr(10))
			}
			newContent = (newContent + Left(line, startX))
		} else if (y == endY) {
			newContent = (newContent + Mid(line, (endX + 1), (len(line) - endX)))
		}
	}
	(*m).Content = newContent
	(*m).Selecting = false
	(*m).Modified = true
	return startX, startY
}

func DoCopy(m EditorModel) (tea.Model, tea.Cmd) {
	if m.Selecting {
		m.Clipboard = GetSelectedText(m)
		m.Message = "Copied"
	} else {
		m.Clipboard = GetLine(m.Content, (m.CursorY + 1))
		m.Message = "Line copied"
	}
	return m, nil
}

func DoCut(m EditorModel) (tea.Model, tea.Cmd) {
	if m.Selecting {
		m.Clipboard = GetSelectedText(m)
		var newX int
		var newY int
		newX, newY = DeleteSelection(&m)
		m.CursorX = newX
		m.CursorY = newY
		m.Message = "Cut"
	} else {
		m.Clipboard = GetLine(m.Content, (m.CursorY + 1))
		var totalLines int = CountLines(m.Content)
		if (totalLines == 1) {
			m.Content = ""
		} else {
			var newContent string = ""
			var i int
			for i = 1; i <= totalLines; i += 1 {
				if (i != (m.CursorY + 1)) {
					if (len(newContent) > 0) {
						newContent = (newContent + Chr(10))
					}
					newContent = (newContent + GetLine(m.Content, i))
				}
			}
			m.Content = newContent
		}
		if (m.CursorY >= CountLines(m.Content)) {
			m.CursorY = (CountLines(m.Content) - 1)
			if (m.CursorY < 0) {
				m.CursorY = 0
			}
		}
		var newLineLen int = GetLineLength(m.Content, (m.CursorY + 1))
		if (m.CursorX > newLineLen) {
			m.CursorX = newLineLen
		}
		m.Modified = true
		m.Message = "Line cut"
	}
	return m, nil
}

func DoPaste(m EditorModel) (tea.Model, tea.Cmd) {
	if (len(m.Clipboard) == 0) {
		m.Message = "Clipboard empty"
		return m, nil
	}
	if m.Selecting {
		var delX int
		var delY int
		delX, delY = DeleteSelection(&m)
		m.CursorX = delX
		m.CursorY = delY
	}
	var clipLines int = CountLines(m.Clipboard)
	if (clipLines == 1) {
		var line string = GetLine(m.Content, (m.CursorY + 1))
		var newLine string = ((Left(line, m.CursorX) + m.Clipboard) + Mid(line, (m.CursorX + 1), (len(line) - m.CursorX)))
		m.Content = SetLine(m.Content, (m.CursorY + 1), newLine)
		m.CursorX = (m.CursorX + len(m.Clipboard))
	} else {
		var currentLine string = GetLine(m.Content, (m.CursorY + 1))
		var beforeCursor string = Left(currentLine, m.CursorX)
		var afterCursor string = Mid(currentLine, (m.CursorX + 1), (len(currentLine) - m.CursorX))
		var newContent string = ""
		var totalLines int = CountLines(m.Content)
		var y int
		for y = 1; y <= totalLines; y += 1 {
			if (y == (m.CursorY + 1)) {
				var clipY int
				for clipY = 1; clipY <= clipLines; clipY += 1 {
					if (len(newContent) > 0) {
						newContent = (newContent + Chr(10))
					}
					if (clipY == 1) {
						newContent = ((newContent + beforeCursor) + GetLine(m.Clipboard, clipY))
					} else if (clipY == clipLines) {
						newContent = ((newContent + GetLine(m.Clipboard, clipY)) + afterCursor)
					} else {
						newContent = (newContent + GetLine(m.Clipboard, clipY))
					}
				}
			} else {
				if (len(newContent) > 0) {
					newContent = (newContent + Chr(10))
				}
				newContent = (newContent + GetLine(m.Content, y))
			}
		}
		m.Content = newContent
		m.CursorY = ((m.CursorY + clipLines) - 1)
		m.CursorX = len(GetLine(m.Clipboard, clipLines))
	}
	m.Modified = true
	m.Message = "Pasted"
	return m, nil
}

func DoSelectAll(m EditorModel) (tea.Model, tea.Cmd) {
	var totalLines int = CountLines(m.Content)
	m.Selecting = true
	m.SelectStartX = 0
	m.SelectStartY = 0
	m.SelectEndY = (totalLines - 1)
	m.SelectEndX = GetLineLength(m.Content, totalLines)
	m.CursorY = m.SelectEndY
	m.CursorX = m.SelectEndX
	m.Message = "All selected"
	return m, nil
}

func DoFindNext(m EditorModel) (tea.Model, tea.Cmd) {
	if (len(m.SearchText) == 0) {
		m.Message = "No search text"
		return m, nil
	}
	var totalLines int = CountLines(m.Content)
	var startLine int = (m.CursorY + 1)
	var startCol int = (m.CursorX + 1)
	var i int
	for i = startLine; i <= totalLines; i += 1 {
		var line string = GetLine(m.Content, i)
		var searchStart int = 1
		if (i == startLine) {
			searchStart = (startCol + 1)
		}
		var pos int = Instr(Mid(line, searchStart, ((len(line) - searchStart) + 1)), m.SearchText)
		if (pos > 0) {
			m.CursorY = (i - 1)
			m.CursorX = (((searchStart - 1) + pos) - 1)
			m.Message = "Found"
			return m, nil
		}
	}
	if m.SearchWrap {
		for i = 1; i <= startLine; i += 1 {
			var wrapLine string = GetLine(m.Content, i)
			var wrapPos int = Instr(wrapLine, m.SearchText)
			if (wrapPos > 0) {
				m.CursorY = (i - 1)
				m.CursorX = (wrapPos - 1)
				m.Message = "Found (wrapped)"
				return m, nil
			}
		}
	}
	m.Message = ("Not found: " + m.SearchText)
	return m, nil
}

func DoReplace(m EditorModel) (tea.Model, tea.Cmd) {
	m.Message = "Replace not yet implemented"
	return m, nil
}

func DoGotoLine(m EditorModel) (tea.Model, tea.Cmd) {
	m.DialogMode = DIALOG_NONE
	var lineNum int = Int(Val(m.DialogInput))
	var totalLines int = CountLines(m.Content)
	if (lineNum < 1) {
		lineNum = 1
	} else if (lineNum > totalLines) {
		lineNum = totalLines
	}
	m.CursorY = (lineNum - 1)
	m.CursorX = 0
	m.Message = fmt.Sprintf("Line %d", lineNum)
	return m, nil
}

func GetDropdownHeight(menu int) int {
	if (menu == MENU_FILE) {
		return 8
	} else if (menu == MENU_EDIT) {
		return 8
	} else if (menu == MENU_SEARCH) {
		return 6
	} else if (menu == MENU_OPTIONS) {
		return 4
	} else if (menu == MENU_HELP) {
		return 4
	}
	return 0
}

func (m EditorModel) View() string {
	var view string = ""
	var totalLines int = CountLines(m.Content)
	view = (RenderMenuBar(m) + Chr(10))
	var dropdownHeight int = 0
	if (m.MenuOpen != MENU_NONE) {
		view = (view + RenderDropdown(m))
		dropdownHeight = GetDropdownHeight(m.MenuOpen)
	}
	var contentHeight int = ((m.Height - 3) - dropdownHeight)
	if (contentHeight < 1) {
		contentHeight = 1
	}
	var lineNumWidth int = 0
	if m.ShowLineNumbers {
		lineNumWidth = 6
	}
	var contentWidth int = (m.Width - lineNumWidth)
	if (contentWidth < 10) {
		contentWidth = 10
	}
	var i int
	for i = 0; i <= (contentHeight - 1); i += 1 {
		var lineIdx int = (m.ScrollY + i)
		var lineText string = ""
		if (lineIdx < totalLines) {
			lineText = GetLine(m.Content, (lineIdx + 1))
		}
		if ((m.ScrollX > 0) && (len(lineText) > m.ScrollX)) {
			lineText = Mid(lineText, (m.ScrollX + 1), (len(lineText) - m.ScrollX))
		} else if (m.ScrollX > 0) {
			lineText = ""
		}
		if m.ShowLineNumbers {
			if (lineIdx < totalLines) {
				view = (view + lineNumStyle.Render(fmt.Sprintf("%5d ", (lineIdx + 1))))
			} else {
				view = (view + lineNumStyle.Render("      "))
			}
		}
		var lineContent string = ""
		var col int
		var cursorCol int = (m.CursorX - m.ScrollX)
		var renderedLen int = 0
		for col = 0; col <= (contentWidth - 1); col += 1 {
			var actualCol int = (m.ScrollX + col)
			var ch string = " "
			if (col < len(lineText)) {
				ch = Mid(lineText, (col + 1), 1)
			}
			if ((lineIdx == m.CursorY) && (col == cursorCol)) {
				lineContent = (lineContent + cursorStyle.Render(ch))
			} else if IsInSelection(m, lineIdx, actualCol) {
				lineContent = (lineContent + selectedStyle.Render(ch))
			} else {
				lineContent = (lineContent + textAreaStyle.Render(ch))
			}
			renderedLen = (renderedLen + 1)
		}
		view = (view + lineContent)
		view = (view + Chr(10))
	}
	var status string = ""
	if (len(m.Filename) > 0) {
		status = (" " + m.Filename)
	} else {
		status = " [Untitled]"
	}
	if m.Modified {
		status = (status + " *")
	}
	status = (status + fmt.Sprintf("  Ln %d, Col %d", (m.CursorY + 1), (m.CursorX + 1)))
	if !(m.InsertMode) {
		status = (status + "  OVR")
	}
	if (len(m.Message) > 0) {
		status = ((status + "  ") + m.Message)
	}
	view = ((view + statusBarStyle.Render(PadRight(status, m.Width))) + Chr(10))
	view = (view + menuBarStyle.Render(PadRight(" F1=Help  F10=Menu  Ctrl+S=Save  Ctrl+Q=Quit", m.Width)))
	if (m.DialogMode != DIALOG_NONE) {
		view = ((view + Chr(10)) + RenderDialog(m))
	}
	return view
}

func RenderMenuItemWithHotkey(name string, hotkeyPos int, selected bool) string {
	var before string = Left(name, (hotkeyPos - 1))
	var hotkey string = Mid(name, hotkeyPos, 1)
	var after string = Mid(name, (hotkeyPos + 1), (len(name) - hotkeyPos))
	if selected {
		return ((menuItemSelectedStyle.Render((" " + before)) + menuHotkeySelectedStyle.Render(hotkey)) + menuItemSelectedStyle.Render((after + " ")))
	} else {
		return ((menuItemStyle.Render((" " + before)) + menuHotkeyStyle.Render(hotkey)) + menuItemStyle.Render((after + " ")))
	}
}

func RenderMenuBar(m EditorModel) string {
	var bar string = ""
	bar = (bar + RenderMenuItemWithHotkey("File", 1, (m.MenuOpen == MENU_FILE)))
	bar = (bar + RenderMenuItemWithHotkey("Edit", 1, (m.MenuOpen == MENU_EDIT)))
	bar = (bar + RenderMenuItemWithHotkey("Search", 1, (m.MenuOpen == MENU_SEARCH)))
	bar = (bar + RenderMenuItemWithHotkey("Options", 1, (m.MenuOpen == MENU_OPTIONS)))
	bar = (bar + RenderMenuItemWithHotkey("Help", 1, (m.MenuOpen == MENU_HELP)))
	var currentLen int = 38
	if (m.Width > currentLen) {
		bar = (bar + menuBarStyle.Render(RepeatChar(" ", (m.Width - currentLen))))
	}
	return bar
}

func RenderDropdown(m EditorModel) string {
	var items string = ""
	if (m.MenuOpen == MENU_FILE) {
		items = (RenderMenuItem("New         Ctrl+N", (m.MenuIndex == 0)) + Chr(10))
		items = ((items + RenderMenuItem("Open        Ctrl+O", (m.MenuIndex == 1))) + Chr(10))
		items = ((items + RenderMenuItem("Save        Ctrl+S", (m.MenuIndex == 2))) + Chr(10))
		items = ((items + RenderMenuItem("Save As...       ", (m.MenuIndex == 3))) + Chr(10))
		items = ((items + RenderMenuItem("-----------------", false)) + Chr(10))
		items = (items + RenderMenuItem("Exit        Alt+X ", (m.MenuIndex == 5)))
	} else if (m.MenuOpen == MENU_EDIT) {
		items = (RenderMenuItem("Cut         Ctrl+X", (m.MenuIndex == 0)) + Chr(10))
		items = ((items + RenderMenuItem("Copy        Ctrl+C", (m.MenuIndex == 1))) + Chr(10))
		items = ((items + RenderMenuItem("Paste       Ctrl+V", (m.MenuIndex == 2))) + Chr(10))
		items = ((items + RenderMenuItem("-----------------", false)) + Chr(10))
		items = ((items + RenderMenuItem("Select All  Ctrl+A", (m.MenuIndex == 4))) + Chr(10))
		items = (items + RenderMenuItem("Clear            ", (m.MenuIndex == 5)))
	} else if (m.MenuOpen == MENU_SEARCH) {
		items = (RenderMenuItem("Find        Ctrl+F", (m.MenuIndex == 0)) + Chr(10))
		items = ((items + RenderMenuItem("Find Next   F3    ", (m.MenuIndex == 1))) + Chr(10))
		items = ((items + RenderMenuItem("Replace     Ctrl+H", (m.MenuIndex == 2))) + Chr(10))
		items = (items + RenderMenuItem("Go to Line  Ctrl+G", (m.MenuIndex == 3)))
	} else if (m.MenuOpen == MENU_OPTIONS) {
		var lineNumCheck string = "[ ]"
		if m.ShowLineNumbers {
			lineNumCheck = "[X]"
		}
		var insertCheck string = "[ ]"
		if m.InsertMode {
			insertCheck = "[X]"
		}
		items = (RenderMenuItem((lineNumCheck + " Line Numbers "), (m.MenuIndex == 0)) + Chr(10))
		items = (items + RenderMenuItem((insertCheck + " Insert Mode  "), (m.MenuIndex == 1)))
	} else if (m.MenuOpen == MENU_HELP) {
		items = (RenderMenuItem("Help        F1    ", (m.MenuIndex == 0)) + Chr(10))
		items = (items + RenderMenuItem("About            ", (m.MenuIndex == 1)))
	}
	return (menuDropdownStyle.Render(items) + Chr(10))
}

func RenderMenuItem(text string, selected bool) string {
	if selected {
		return menuItemSelectedStyle.Render(text)
	}
	return text
}

func RenderDialog(m EditorModel) string {
	var content string = ""
	var title string = ""
	if (m.DialogMode == DIALOG_HELP) {
		title = " Help "
		content = (("MS-DOS Editor Clone - Keyboard Shortcuts" + Chr(10)) + Chr(10))
		content = ((content + "File Operations:") + Chr(10))
		content = ((content + "  Ctrl+N    New file") + Chr(10))
		content = ((content + "  Ctrl+O    Open file") + Chr(10))
		content = ((content + "  Ctrl+S    Save file") + Chr(10))
		content = (((content + "  Ctrl+Q    Exit") + Chr(10)) + Chr(10))
		content = ((content + "Edit Operations:") + Chr(10))
		content = ((content + "  Ctrl+C    Copy line") + Chr(10))
		content = ((content + "  Ctrl+X    Cut line") + Chr(10))
		content = (((content + "  Ctrl+V    Paste") + Chr(10)) + Chr(10))
		content = ((content + "Navigation:") + Chr(10))
		content = ((content + "  Arrow keys, Home, End, PgUp, PgDn") + Chr(10))
		content = (((content + "  Ctrl+Home/End  Start/end of file") + Chr(10)) + Chr(10))
		content = ((content + "Search:") + Chr(10))
		content = ((content + "  Ctrl+F    Find") + Chr(10))
		content = ((content + "  F3        Find next") + Chr(10))
		content = (((content + "  Ctrl+G    Go to line") + Chr(10)) + Chr(10))
		content = (content + "Press Enter or Esc to close")
	} else if (m.DialogMode == DIALOG_ABOUT) {
		title = " About "
		content = ("DBasic EDIT" + Chr(10))
		content = (((content + "Version 1.0") + Chr(10)) + Chr(10))
		content = ((content + "A clone of MS-DOS 5.0 EDIT.COM") + Chr(10))
		content = (((content + "Written entirely in DBasic") + Chr(10)) + Chr(10))
		content = (content + "Press Enter or Esc to close")
	} else if (m.DialogMode == DIALOG_OPEN) {
		title = " Open File "
		content = (("Filename:" + Chr(10)) + Chr(10))
		content = (((((content + "[") + m.DialogInput) + "]") + Chr(10)) + Chr(10))
		content = (content + "Enter=Open  Esc=Cancel")
	} else if (m.DialogMode == DIALOG_SAVEAS) {
		title = " Save As "
		content = (("Filename:" + Chr(10)) + Chr(10))
		content = (((((content + "[") + m.DialogInput) + "]") + Chr(10)) + Chr(10))
		content = (content + "Enter=Save  Esc=Cancel")
	} else if (m.DialogMode == DIALOG_FIND) {
		title = " Find "
		content = (("Search for:" + Chr(10)) + Chr(10))
		content = (((((content + "[") + m.DialogInput) + "]") + Chr(10)) + Chr(10))
		content = (content + "Enter=Find  Esc=Cancel")
	} else if (m.DialogMode == DIALOG_GOTO) {
		title = " Go to Line "
		content = (("Line number:" + Chr(10)) + Chr(10))
		content = (((((content + "[") + m.DialogInput) + "]") + Chr(10)) + Chr(10))
		content = (content + "Enter=Go  Esc=Cancel")
	} else if (m.DialogMode == DIALOG_CONFIRM_NEW) {
		title = " New File "
		content = (("File has been modified." + Chr(10)) + Chr(10))
		content = (content + "Discard changes? (Y/N)")
	} else if (m.DialogMode == DIALOG_CONFIRM_EXIT) {
		title = " Exit "
		content = (("File has been modified." + Chr(10)) + Chr(10))
		content = (content + "Exit without saving? (Y/N)")
	}
	return ((dialogTitleStyle.Render(title) + Chr(10)) + dialogStyle.Render(content))
}

func Main() {
	var model EditorModel
	model.CursorX = 0
	model.CursorY = 0
	model.Width = 80
	model.Height = 25
	model.ScrollX = 0
	model.ScrollY = 0
	model.SelectStartX = 0
	model.SelectStartY = 0
	model.SelectEndX = 0
	model.SelectEndY = 0
	model.Selecting = false
	model.Clipboard = ""
	model.Filename = ""
	model.Modified = false
	model.MenuOpen = MENU_NONE
	model.MenuIndex = 0
	model.DialogMode = DIALOG_NONE
	model.DialogInput = ""
	model.DialogCursor = 0
	model.SearchText = ""
	model.ReplaceText = ""
	model.SearchWrap = true
	model.SearchCase = false
	model.Message = "Press F1 for Help, F10 for Menu"
	model.ShowLineNumbers = false
	model.TabSize = 4
	model.InsertMode = true
	var args []string = os.Args
	if (len(args) > 1) {
		var filename string = args[1]
		if FileExists(filename) {
			model.Content = ReadFile(filename)
			model.Filename = filename
			model.Message = ("Opened: " + filename)
		} else {
			model.Filename = filename
			model.Message = ("New file: " + filename)
		}
	} else {
		model.Content = ("' Welcome to DBasic EDIT" + Chr(10))
		model.Content = ((model.Content + "' A clone of MS-DOS 5.0 EDIT.COM") + Chr(10))
		model.Content = ((model.Content + "'") + Chr(10))
		model.Content = ((model.Content + "' Press F10 or Alt+F to open the menu") + Chr(10))
		model.Content = ((model.Content + "' Press F1 for help") + Chr(10))
		model.Content = ((model.Content + "'") + Chr(10))
		model.Content = ((model.Content + "' Start typing to edit...") + Chr(10))
	}
	tea.NewProgram(model, tea.WithAltScreen()).Run()
}

func main() {
	Main()
}
