package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Colors matching MS-DOS EDIT.COM
var (
	// Main colors
	blueBackground   = lipgloss.Color("17")  // Dark blue
	cyanText         = lipgloss.Color("14")  // Yellow/bright
	whiteText        = lipgloss.Color("15")  // Bright white
	blackText        = lipgloss.Color("0")   // Black
	grayBackground   = lipgloss.Color("7")   // Light gray
	highlightBg      = lipgloss.Color("6")   // Cyan

	// Styles
	menuBarStyle = lipgloss.NewStyle().
			Background(grayBackground).
			Foreground(blackText)

	menuItemStyle = lipgloss.NewStyle().
			Background(grayBackground).
			Foreground(blackText).
			Padding(0, 1)

	menuItemSelectedStyle = lipgloss.NewStyle().
				Background(highlightBg).
				Foreground(blackText).
				Padding(0, 1)

	editorStyle = lipgloss.NewStyle().
			Background(blueBackground).
			Foreground(cyanText)

	statusBarStyle = lipgloss.NewStyle().
			Background(grayBackground).
			Foreground(blackText)

	titleStyle = lipgloss.NewStyle().
			Background(blueBackground).
			Foreground(whiteText).
			Bold(true)
)

// Menu structure
type MenuItem struct {
	label    string
	shortcut string
}

var menus = []struct {
	name  string
	items []MenuItem
}{
	{"File", []MenuItem{
		{"New", "Ctrl+N"},
		{"Open...", "Ctrl+O"},
		{"Save", "Ctrl+S"},
		{"Save As...", "Ctrl+A"},
		{"-", ""},
		{"Exit", "Alt+X"},
	}},
	{"Edit", []MenuItem{
		{"Cut", "Ctrl+X"},
		{"Copy", "Ctrl+C"},
		{"Paste", "Ctrl+V"},
		{"Cut Line", "Ctrl+K"},
		{"Clear", "Del"},
	}},
	{"Search", []MenuItem{
		{"Find...", "Ctrl+F"},
		{"Repeat Last Find", "F3"},
		{"Replace...", "Ctrl+H"},
	}},
	{"Options", []MenuItem{
		{"Settings...", ""},
	}},
	{"Help", []MenuItem{
		{"About...", "F1"},
	}},
}

type dialogType int

const (
	dialogNone dialogType = iota
	dialogOpen
	dialogSaveAs
	dialogFind
	dialogReplace
	dialogAbout
	dialogConfirmNew
	dialogFileBrowser
)

// Position represents a cursor position in the document
type Position struct {
	X int // Column
	Y int // Line
}

type editorModel struct {
	// Text content
	lines     []string
	cursorX   int
	cursorY   int
	scrollY   int

	// Selection state
	selecting   bool     // true when actively selecting
	selectStart Position // Start of selection (anchor point)
	selectEnd   Position // End of selection (moving point)

	// Window dimensions
	width     int
	height    int

	// File state
	filename  string
	modified  bool

	// Menu state
	menuOpen      bool
	menuIndex     int
	submenuIndex  int

	// Dialog state
	dialog       dialogType
	dialogInput  string
	dialogCursor int

	// Search state
	findTerm    string
	replaceTerm string
	replaceMode bool // true = in replace input field

	// Clipboard
	clipboard string

	// Status message
	statusMsg string

	// File browser state
	browserDir       string   // Current directory
	browserFiles     []string // Files in current directory
	browserIndex     int      // Selected file index
	browserScroll    int      // Scroll offset
	browserForSave   bool     // true = Save As, false = Open
	browserPathInput bool     // true = editing path, false = browsing files

	// Mode
	quitting bool
}

func initialModel() editorModel {
	return editorModel{
		lines:    []string{""},
		filename: "Untitled",
		width:    80,
		height:   24,
	}
}

// Selection helper methods

// hasSelection returns true if there is an active selection
func (m *editorModel) hasSelection() bool {
	return m.selecting && (m.selectStart.X != m.selectEnd.X || m.selectStart.Y != m.selectEnd.Y)
}

// getSelectionBounds returns the start and end of selection in document order
func (m *editorModel) getSelectionBounds() (Position, Position) {
	start, end := m.selectStart, m.selectEnd
	// Ensure start comes before end
	if start.Y > end.Y || (start.Y == end.Y && start.X > end.X) {
		start, end = end, start
	}
	return start, end
}

// startSelection begins a new selection at the current cursor position
func (m *editorModel) startSelection() {
	m.selecting = true
	m.selectStart = Position{X: m.cursorX, Y: m.cursorY}
	m.selectEnd = Position{X: m.cursorX, Y: m.cursorY}
}

// updateSelection updates the selection end point to current cursor
func (m *editorModel) updateSelection() {
	m.selectEnd = Position{X: m.cursorX, Y: m.cursorY}
}

// clearSelection clears the current selection
func (m *editorModel) clearSelection() {
	m.selecting = false
}

// getSelectedText returns the currently selected text
func (m *editorModel) getSelectedText() string {
	if !m.hasSelection() {
		return ""
	}

	start, end := m.getSelectionBounds()

	if start.Y == end.Y {
		// Selection on single line
		line := m.lines[start.Y]
		startX := start.X
		endX := end.X
		if startX > len(line) {
			startX = len(line)
		}
		if endX > len(line) {
			endX = len(line)
		}
		return line[startX:endX]
	}

	// Multi-line selection
	var result strings.Builder

	// First line (from start to end of line)
	firstLine := m.lines[start.Y]
	startX := start.X
	if startX > len(firstLine) {
		startX = len(firstLine)
	}
	result.WriteString(firstLine[startX:])
	result.WriteString("\n")

	// Middle lines (complete lines)
	for y := start.Y + 1; y < end.Y; y++ {
		result.WriteString(m.lines[y])
		result.WriteString("\n")
	}

	// Last line (from start to end position)
	lastLine := m.lines[end.Y]
	endX := end.X
	if endX > len(lastLine) {
		endX = len(lastLine)
	}
	result.WriteString(lastLine[:endX])

	return result.String()
}

// deleteSelection deletes the selected text and returns it
func (m *editorModel) deleteSelection() string {
	if !m.hasSelection() {
		return ""
	}

	text := m.getSelectedText()
	start, end := m.getSelectionBounds()

	if start.Y == end.Y {
		// Single line deletion
		line := m.lines[start.Y]
		startX := start.X
		endX := end.X
		if startX > len(line) {
			startX = len(line)
		}
		if endX > len(line) {
			endX = len(line)
		}
		m.lines[start.Y] = line[:startX] + line[endX:]
	} else {
		// Multi-line deletion
		firstLine := m.lines[start.Y]
		lastLine := m.lines[end.Y]

		startX := start.X
		if startX > len(firstLine) {
			startX = len(firstLine)
		}
		endX := end.X
		if endX > len(lastLine) {
			endX = len(lastLine)
		}

		// Combine first part of first line with last part of last line
		m.lines[start.Y] = firstLine[:startX] + lastLine[endX:]

		// Remove lines in between
		m.lines = append(m.lines[:start.Y+1], m.lines[end.Y+1:]...)
	}

	// Move cursor to start of selection
	m.cursorX = start.X
	m.cursorY = start.Y
	m.clearSelection()
	m.modified = true

	return text
}

// isPositionInSelection checks if a position is within the selection
func (m *editorModel) isPositionInSelection(x, y int) bool {
	if !m.hasSelection() {
		return false
	}

	start, end := m.getSelectionBounds()

	if y < start.Y || y > end.Y {
		return false
	}

	if y == start.Y && y == end.Y {
		// Single line selection
		return x >= start.X && x < end.X
	}

	if y == start.Y {
		return x >= start.X
	}

	if y == end.Y {
		return x < end.X
	}

	// Middle line - fully selected
	return true
}

// moveToNextWord moves cursor to the start of next word
func (m *editorModel) moveToNextWord() {
	line := m.lines[m.cursorY]

	// If at end of line, go to next line
	if m.cursorX >= len(line) {
		if m.cursorY < len(m.lines)-1 {
			m.cursorY++
			m.cursorX = 0
		}
		return
	}

	// Skip current word (non-space characters)
	for m.cursorX < len(line) && !isWordSeparator(line[m.cursorX]) {
		m.cursorX++
	}

	// Skip spaces
	for m.cursorX < len(line) && isWordSeparator(line[m.cursorX]) {
		m.cursorX++
	}
}

// moveToPrevWord moves cursor to the start of previous word
func (m *editorModel) moveToPrevWord() {
	line := m.lines[m.cursorY]

	// If at start of line, go to end of previous line
	if m.cursorX <= 0 {
		if m.cursorY > 0 {
			m.cursorY--
			m.cursorX = len(m.lines[m.cursorY])
		}
		return
	}

	// Move back one to get off current position
	m.cursorX--

	// Skip spaces going backwards
	for m.cursorX > 0 && isWordSeparator(line[m.cursorX-1]) {
		m.cursorX--
	}

	// Skip word characters going backwards
	for m.cursorX > 0 && !isWordSeparator(line[m.cursorX-1]) {
		m.cursorX--
	}
}

// isWordSeparator returns true if the character is a word separator
func isWordSeparator(c byte) bool {
	return c == ' ' || c == '\t' || c == '.' || c == ',' || c == ';' ||
		c == ':' || c == '!' || c == '?' || c == '"' || c == '\'' ||
		c == '(' || c == ')' || c == '[' || c == ']' || c == '{' || c == '}'
}

func (m editorModel) Init() tea.Cmd {
	return nil // WithAltScreen handles initial setup
}

func (m editorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Handle dialogs first
		if m.dialog != dialogNone {
			return m.handleDialogKeys(msg)
		}

		// Handle menu navigation
		if m.menuOpen {
			return m.handleMenuKeys(msg)
		}

		// Clear status message on any key
		m.statusMsg = ""

		key := msg.String()

		switch key {
		case "ctrl+q", "alt+x":
			m.quitting = true
			return m, tea.Quit

		case "alt+f":
			m.menuOpen = true
			m.menuIndex = 0
			m.submenuIndex = 0
			return m, nil

		case "alt+e":
			m.menuOpen = true
			m.menuIndex = 1
			m.submenuIndex = 0
			return m, nil

		case "alt+s":
			m.menuOpen = true
			m.menuIndex = 2
			m.submenuIndex = 0
			return m, nil

		case "alt+o":
			m.menuOpen = true
			m.menuIndex = 3
			m.submenuIndex = 0
			return m, nil

		case "alt+h":
			m.menuOpen = true
			m.menuIndex = 4
			m.submenuIndex = 0
			return m, nil

		case "f10", "alt":
			m.menuOpen = true
			m.submenuIndex = 0
			return m, nil

		// File shortcuts
		case "ctrl+n":
			return m.executeMenuAction(0, "New")
		case "ctrl+o":
			return m.executeMenuAction(0, "Open...")
		case "ctrl+s":
			return m.executeMenuAction(0, "Save")

		// Edit shortcuts - Ctrl+A is now Select All
		case "ctrl+a":
			// Select all
			m.selecting = true
			m.selectStart = Position{X: 0, Y: 0}
			lastLine := len(m.lines) - 1
			m.selectEnd = Position{X: len(m.lines[lastLine]), Y: lastLine}
			m.cursorY = lastLine
			m.cursorX = len(m.lines[lastLine])
			m.statusMsg = "All text selected"

		case "ctrl+x":
			// Cut - only cuts selected text (use Ctrl+K for line)
			if m.hasSelection() {
				m.clipboard = m.deleteSelection()
				m.statusMsg = "Selection cut to clipboard"
			} else {
				// No selection - do nothing (Ctrl+K cuts line)
				m.statusMsg = "No selection (Ctrl+K to cut line)"
			}

		case "ctrl+k":
			// Cut entire line
			if m.cursorY < len(m.lines) {
				m.clipboard = m.lines[m.cursorY]
				m.lines = append(m.lines[:m.cursorY], m.lines[m.cursorY+1:]...)
				if len(m.lines) == 0 {
					m.lines = []string{""}
				}
				if m.cursorY >= len(m.lines) {
					m.cursorY = len(m.lines) - 1
				}
				m.cursorX = 0
				m.modified = true
				m.clearSelection()
				m.statusMsg = "Line cut to clipboard"
			}

		case "ctrl+c":
			// Copy - if selection, copy selected text; otherwise copy line
			if m.hasSelection() {
				m.clipboard = m.getSelectedText()
				m.statusMsg = "Selection copied to clipboard"
			} else {
				m.clipboard = m.lines[m.cursorY]
				m.statusMsg = "Line copied to clipboard"
			}

		case "ctrl+v":
			// Paste - delete selection first if any
			if m.hasSelection() {
				m.deleteSelection()
			}
			if m.clipboard != "" {
				// Check if clipboard has newlines
				if strings.Contains(m.clipboard, "\n") {
					// Multi-line paste
					clipLines := strings.Split(m.clipboard, "\n")
					line := m.lines[m.cursorY]
					before := line[:m.cursorX]
					after := line[m.cursorX:]

					// First line
					m.lines[m.cursorY] = before + clipLines[0]

					// Middle lines
					newLines := make([]string, 0, len(m.lines)+len(clipLines)-1)
					newLines = append(newLines, m.lines[:m.cursorY+1]...)
					for i := 1; i < len(clipLines)-1; i++ {
						newLines = append(newLines, clipLines[i])
					}
					// Last line with remainder
					if len(clipLines) > 1 {
						newLines = append(newLines, clipLines[len(clipLines)-1]+after)
					}
					newLines = append(newLines, m.lines[m.cursorY+1:]...)
					m.lines = newLines

					// Move cursor to end of pasted text
					m.cursorY += len(clipLines) - 1
					m.cursorX = len(clipLines[len(clipLines)-1])
				} else {
					// Single line paste
					line := m.lines[m.cursorY]
					m.lines[m.cursorY] = line[:m.cursorX] + m.clipboard + line[m.cursorX:]
					m.cursorX += len(m.clipboard)
				}
				m.modified = true
				m.statusMsg = "Pasted from clipboard"
			}

		// Search shortcuts
		case "ctrl+f":
			return m.executeMenuAction(2, "Find...")
		case "f3":
			return m.executeMenuAction(2, "Repeat Last Find")
		case "ctrl+h":
			return m.executeMenuAction(2, "Replace...")

		// Help shortcuts
		case "f1":
			return m.executeMenuAction(4, "About...")

		// Selection with Shift+Arrow keys
		case "shift+left":
			if !m.selecting {
				m.startSelection()
			}
			if m.cursorX > 0 {
				m.cursorX--
			} else if m.cursorY > 0 {
				m.cursorY--
				m.cursorX = len(m.lines[m.cursorY])
			}
			m.updateSelection()

		case "shift+right":
			if !m.selecting {
				m.startSelection()
			}
			if m.cursorX < len(m.lines[m.cursorY]) {
				m.cursorX++
			} else if m.cursorY < len(m.lines)-1 {
				m.cursorY++
				m.cursorX = 0
			}
			m.updateSelection()

		case "shift+up":
			if !m.selecting {
				m.startSelection()
			}
			if m.cursorY > 0 {
				m.cursorY--
				if m.cursorX > len(m.lines[m.cursorY]) {
					m.cursorX = len(m.lines[m.cursorY])
				}
			}
			m.updateSelection()

		case "shift+down":
			if !m.selecting {
				m.startSelection()
			}
			if m.cursorY < len(m.lines)-1 {
				m.cursorY++
				if m.cursorX > len(m.lines[m.cursorY]) {
					m.cursorX = len(m.lines[m.cursorY])
				}
			}
			m.updateSelection()

		case "shift+home":
			if !m.selecting {
				m.startSelection()
			}
			m.cursorX = 0
			m.updateSelection()

		case "shift+end":
			if !m.selecting {
				m.startSelection()
			}
			m.cursorX = len(m.lines[m.cursorY])
			m.updateSelection()

		// Word selection with Ctrl+Shift+Arrow
		case "ctrl+shift+left":
			if !m.selecting {
				m.startSelection()
			}
			m.moveToPrevWord()
			m.updateSelection()

		case "ctrl+shift+right":
			if !m.selecting {
				m.startSelection()
			}
			m.moveToNextWord()
			m.updateSelection()

		// Word movement without selection
		case "ctrl+left":
			m.clearSelection()
			m.moveToPrevWord()

		case "ctrl+right":
			m.clearSelection()
			m.moveToNextWord()

		// Regular movement clears selection
		case "left":
			m.clearSelection()
			if m.cursorX > 0 {
				m.cursorX--
			} else if m.cursorY > 0 {
				m.cursorY--
				m.cursorX = len(m.lines[m.cursorY])
			}

		case "right":
			m.clearSelection()
			if m.cursorX < len(m.lines[m.cursorY]) {
				m.cursorX++
			} else if m.cursorY < len(m.lines)-1 {
				m.cursorY++
				m.cursorX = 0
			}

		case "up":
			m.clearSelection()
			if m.cursorY > 0 {
				m.cursorY--
				if m.cursorX > len(m.lines[m.cursorY]) {
					m.cursorX = len(m.lines[m.cursorY])
				}
			}

		case "down":
			m.clearSelection()
			if m.cursorY < len(m.lines)-1 {
				m.cursorY++
				if m.cursorX > len(m.lines[m.cursorY]) {
					m.cursorX = len(m.lines[m.cursorY])
				}
			}

		case "home":
			m.clearSelection()
			m.cursorX = 0

		case "end":
			m.clearSelection()
			m.cursorX = len(m.lines[m.cursorY])

		case "pgup":
			m.clearSelection()
			m.cursorY -= m.height - 4
			if m.cursorY < 0 {
				m.cursorY = 0
			}

		case "pgdown":
			m.clearSelection()
			m.cursorY += m.height - 4
			if m.cursorY >= len(m.lines) {
				m.cursorY = len(m.lines) - 1
			}

		case "enter":
			// Delete selection if any
			if m.hasSelection() {
				m.deleteSelection()
			}
			// Split line at cursor
			line := m.lines[m.cursorY]
			before := line[:m.cursorX]
			after := line[m.cursorX:]
			m.lines[m.cursorY] = before
			// Insert new line
			newLines := make([]string, len(m.lines)+1)
			copy(newLines, m.lines[:m.cursorY+1])
			newLines[m.cursorY+1] = after
			copy(newLines[m.cursorY+2:], m.lines[m.cursorY+1:])
			m.lines = newLines
			m.cursorY++
			m.cursorX = 0
			m.modified = true

		case "backspace":
			if m.hasSelection() {
				m.deleteSelection()
			} else if m.cursorX > 0 {
				line := m.lines[m.cursorY]
				m.lines[m.cursorY] = line[:m.cursorX-1] + line[m.cursorX:]
				m.cursorX--
				m.modified = true
			} else if m.cursorY > 0 {
				// Join with previous line
				prevLine := m.lines[m.cursorY-1]
				m.cursorX = len(prevLine)
				m.lines[m.cursorY-1] = prevLine + m.lines[m.cursorY]
				m.lines = append(m.lines[:m.cursorY], m.lines[m.cursorY+1:]...)
				m.cursorY--
				m.modified = true
			}

		case "delete":
			if m.hasSelection() {
				m.deleteSelection()
			} else {
				line := m.lines[m.cursorY]
				if m.cursorX < len(line) {
					m.lines[m.cursorY] = line[:m.cursorX] + line[m.cursorX+1:]
					m.modified = true
				} else if m.cursorY < len(m.lines)-1 {
					// Join with next line
					m.lines[m.cursorY] = line + m.lines[m.cursorY+1]
					m.lines = append(m.lines[:m.cursorY+1], m.lines[m.cursorY+2:]...)
					m.modified = true
				}
			}

		case "tab":
			// Delete selection first if any
			if m.hasSelection() {
				m.deleteSelection()
			}
			// Insert spaces for tab
			line := m.lines[m.cursorY]
			m.lines[m.cursorY] = line[:m.cursorX] + "    " + line[m.cursorX:]
			m.cursorX += 4
			m.modified = true

		case "esc":
			// Clear selection
			m.clearSelection()

		default:
			// Insert character (replaces selection if any)
			if len(key) == 1 && key[0] >= 32 {
				if m.hasSelection() {
					m.deleteSelection()
				}
				line := m.lines[m.cursorY]
				m.lines[m.cursorY] = line[:m.cursorX] + key + line[m.cursorX:]
				m.cursorX++
				m.modified = true
			}
		}

		// Adjust scroll
		m.adjustScroll()
	}

	return m, nil
}

func (m editorModel) handleMenuKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		m.menuOpen = false
		return m, nil

	case "left":
		m.menuIndex--
		if m.menuIndex < 0 {
			m.menuIndex = len(menus) - 1
		}
		m.submenuIndex = 0

	case "right":
		m.menuIndex++
		if m.menuIndex >= len(menus) {
			m.menuIndex = 0
		}
		m.submenuIndex = 0

	case "up":
		m.submenuIndex--
		for m.submenuIndex >= 0 && menus[m.menuIndex].items[m.submenuIndex].label == "-" {
			m.submenuIndex--
		}
		if m.submenuIndex < 0 {
			m.submenuIndex = len(menus[m.menuIndex].items) - 1
		}

	case "down":
		m.submenuIndex++
		for m.submenuIndex < len(menus[m.menuIndex].items) && menus[m.menuIndex].items[m.submenuIndex].label == "-" {
			m.submenuIndex++
		}
		if m.submenuIndex >= len(menus[m.menuIndex].items) {
			m.submenuIndex = 0
		}

	case "enter":
		// Execute menu action
		item := menus[m.menuIndex].items[m.submenuIndex]
		m.menuOpen = false
		return m.executeMenuAction(m.menuIndex, item.label)

	default:
		// Check for hotkey - single letter that matches first char of menu item
		if len(key) == 1 {
			keyRune := rune(strings.ToLower(key)[0])
			for _, item := range menus[m.menuIndex].items {
				if item.label != "-" && len(item.label) > 0 {
					firstChar := rune(strings.ToLower(string(item.label[0]))[0])
					if keyRune == firstChar {
						m.menuOpen = false
						return m.executeMenuAction(m.menuIndex, item.label)
					}
				}
			}
		}
	}

	return m, nil
}

func (m editorModel) handleDialogKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle file browser separately
	if m.dialog == dialogFileBrowser {
		return m.handleFileBrowserKeys(msg)
	}

	switch msg.String() {
	case "esc":
		m.dialog = dialogNone
		return m, nil

	case "enter":
		switch m.dialog {
		case dialogOpen:
			if m.dialogInput != "" {
				m.loadFile(m.dialogInput)
			}
			m.dialog = dialogNone
		case dialogSaveAs:
			if m.dialogInput != "" {
				m.saveFile(m.dialogInput)
			}
			m.dialog = dialogNone
		case dialogFind:
			m.findTerm = m.dialogInput
			m.dialog = dialogNone
			if m.findTerm != "" {
				m.findNext()
			}
		case dialogReplace:
			if !m.replaceMode {
				m.findTerm = m.dialogInput
				m.replaceMode = true
				m.dialogInput = m.replaceTerm
				m.dialogCursor = len(m.dialogInput)
			} else {
				m.replaceTerm = m.dialogInput
				m.dialog = dialogNone
				// Do replace
				if m.findTerm != "" {
					m.replaceAll()
				}
			}
		case dialogConfirmNew:
			// User confirmed, create new file
			m.lines = []string{""}
			m.cursorX, m.cursorY, m.scrollY = 0, 0, 0
			m.filename = "Untitled"
			m.modified = false
			m.dialog = dialogNone
			m.statusMsg = "New file"
		case dialogAbout:
			m.dialog = dialogNone
		}
		return m, nil

	case "tab":
		if m.dialog == dialogReplace {
			// Toggle between find and replace fields
			if !m.replaceMode {
				m.findTerm = m.dialogInput
				m.replaceMode = true
				m.dialogInput = m.replaceTerm
			} else {
				m.replaceTerm = m.dialogInput
				m.replaceMode = false
				m.dialogInput = m.findTerm
			}
			m.dialogCursor = len(m.dialogInput)
		}
		return m, nil

	case "backspace":
		if m.dialogCursor > 0 && len(m.dialogInput) > 0 {
			m.dialogInput = m.dialogInput[:m.dialogCursor-1] + m.dialogInput[m.dialogCursor:]
			m.dialogCursor--
		}

	case "delete":
		if m.dialogCursor < len(m.dialogInput) {
			m.dialogInput = m.dialogInput[:m.dialogCursor] + m.dialogInput[m.dialogCursor+1:]
		}

	case "left":
		if m.dialogCursor > 0 {
			m.dialogCursor--
		}

	case "right":
		if m.dialogCursor < len(m.dialogInput) {
			m.dialogCursor++
		}

	case "home":
		m.dialogCursor = 0

	case "end":
		m.dialogCursor = len(m.dialogInput)

	default:
		// Insert character
		if len(msg.String()) == 1 && msg.String()[0] >= 32 {
			m.dialogInput = m.dialogInput[:m.dialogCursor] + msg.String() + m.dialogInput[m.dialogCursor:]
			m.dialogCursor++
		}
	}

	return m, nil
}

func (m *editorModel) replaceAll() {
	count := 0
	for i, line := range m.lines {
		if strings.Contains(line, m.findTerm) {
			m.lines[i] = strings.ReplaceAll(line, m.findTerm, m.replaceTerm)
			count++
			m.modified = true
		}
	}
	if count > 0 {
		m.statusMsg = fmt.Sprintf("Replaced in %d line(s)", count)
	} else {
		m.statusMsg = "Not found: " + m.findTerm
	}
}

func (m editorModel) executeMenuAction(menuIdx int, action string) (tea.Model, tea.Cmd) {
	switch menuIdx {
	case 0: // File menu
		switch action {
		case "New":
			if m.modified {
				m.dialog = dialogConfirmNew
				m.dialogInput = ""
			} else {
				m.lines = []string{""}
				m.cursorX, m.cursorY, m.scrollY = 0, 0, 0
				m.filename = "Untitled"
				m.modified = false
				m.statusMsg = "New file"
			}
		case "Open...":
			m.openFileBrowser(false)
		case "Save":
			if m.filename == "Untitled" {
				m.openFileBrowser(true)
			} else {
				m.saveFile(m.filename)
			}
		case "Save As...":
			m.openFileBrowser(true)
		case "Exit":
			return m, tea.Quit
		}
	case 1: // Edit menu
		switch action {
		case "Cut":
			// Cut selected text only
			if m.hasSelection() {
				m.clipboard = m.deleteSelection()
				m.statusMsg = "Selection cut to clipboard"
			} else {
				m.statusMsg = "No selection (use Cut Line for whole line)"
			}
		case "Copy":
			if m.hasSelection() {
				m.clipboard = m.getSelectedText()
				m.statusMsg = "Selection copied to clipboard"
			} else {
				m.clipboard = m.lines[m.cursorY]
				m.statusMsg = "Line copied to clipboard"
			}
		case "Paste":
			if m.clipboard != "" {
				if m.hasSelection() {
					m.deleteSelection()
				}
				// Check if clipboard has newlines
				if strings.Contains(m.clipboard, "\n") {
					// Multi-line paste
					clipLines := strings.Split(m.clipboard, "\n")
					line := m.lines[m.cursorY]
					before := line[:m.cursorX]
					after := line[m.cursorX:]
					m.lines[m.cursorY] = before + clipLines[0]
					newLines := make([]string, 0, len(m.lines)+len(clipLines)-1)
					newLines = append(newLines, m.lines[:m.cursorY+1]...)
					for i := 1; i < len(clipLines)-1; i++ {
						newLines = append(newLines, clipLines[i])
					}
					if len(clipLines) > 1 {
						newLines = append(newLines, clipLines[len(clipLines)-1]+after)
					}
					newLines = append(newLines, m.lines[m.cursorY+1:]...)
					m.lines = newLines
					m.cursorY += len(clipLines) - 1
					m.cursorX = len(clipLines[len(clipLines)-1])
				} else {
					line := m.lines[m.cursorY]
					m.lines[m.cursorY] = line[:m.cursorX] + m.clipboard + line[m.cursorX:]
					m.cursorX += len(m.clipboard)
				}
				m.modified = true
				m.statusMsg = "Pasted from clipboard"
			}
		case "Cut Line":
			// Cut entire line
			if m.cursorY < len(m.lines) {
				m.clipboard = m.lines[m.cursorY]
				m.lines = append(m.lines[:m.cursorY], m.lines[m.cursorY+1:]...)
				if len(m.lines) == 0 {
					m.lines = []string{""}
				}
				if m.cursorY >= len(m.lines) {
					m.cursorY = len(m.lines) - 1
				}
				m.cursorX = 0
				m.modified = true
				m.clearSelection()
				m.statusMsg = "Line cut to clipboard"
			}
		case "Clear":
			if m.hasSelection() {
				m.deleteSelection()
			} else {
				m.lines[m.cursorY] = ""
				m.cursorX = 0
				m.modified = true
			}
		}
	case 2: // Search menu
		switch action {
		case "Find...":
			m.dialog = dialogFind
			m.dialogInput = m.findTerm
			m.dialogCursor = len(m.dialogInput)
		case "Repeat Last Find":
			if m.findTerm != "" {
				m.findNext()
			}
		case "Replace...":
			m.dialog = dialogReplace
			m.dialogInput = m.findTerm
			m.dialogCursor = len(m.dialogInput)
			m.replaceMode = false
		}
	case 3: // Options menu
		switch action {
		case "Settings...":
			m.statusMsg = "Settings not implemented"
		}
	case 4: // Help menu
		switch action {
		case "About...":
			m.dialog = dialogAbout
		}
	}
	return m, nil
}

func (m *editorModel) saveFile(filename string) {
	content := strings.Join(m.lines, "\n")
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		m.statusMsg = "Error: " + err.Error()
	} else {
		m.filename = filename
		m.modified = false
		m.statusMsg = "Saved: " + filename
	}
}

func (m *editorModel) loadFile(filename string) {
	content, err := os.ReadFile(filename)
	if err != nil {
		m.statusMsg = "Error: " + err.Error()
		return
	}
	m.lines = strings.Split(string(content), "\n")
	if len(m.lines) == 0 {
		m.lines = []string{""}
	}
	m.filename = filename
	m.cursorX, m.cursorY, m.scrollY = 0, 0, 0
	m.modified = false
	m.statusMsg = "Loaded: " + filename
}

func (m *editorModel) findNext() {
	if m.findTerm == "" {
		return
	}
	// Start searching from current position
	startLine := m.cursorY
	startCol := m.cursorX + 1

	for i := 0; i < len(m.lines); i++ {
		lineNum := (startLine + i) % len(m.lines)
		line := m.lines[lineNum]
		searchStart := 0
		if i == 0 {
			searchStart = startCol
		}
		if searchStart < len(line) {
			idx := strings.Index(line[searchStart:], m.findTerm)
			if idx >= 0 {
				m.cursorY = lineNum
				m.cursorX = searchStart + idx
				m.adjustScroll()
				m.statusMsg = "Found"
				return
			}
		}
	}
	m.statusMsg = "Not found: " + m.findTerm
}

func (m *editorModel) openFileBrowser(forSave bool) {
	m.browserForSave = forSave
	m.browserIndex = 0
	m.browserScroll = 0
	m.browserPathInput = false // Start in file list mode

	// Get current directory
	dir, err := os.Getwd()
	if err != nil {
		m.statusMsg = "Error: " + err.Error()
		return
	}

	// If we have a current file, use its directory
	if m.filename != "Untitled" && m.filename != "" {
		dir = filepath.Dir(m.filename)
		if dir == "." {
			dir, _ = os.Getwd()
		}
	}

	m.browserDir = dir
	m.loadBrowserDir()

	// For save, set the input to current filename
	if forSave && m.filename != "Untitled" {
		m.dialogInput = filepath.Base(m.filename)
	} else {
		m.dialogInput = ""
	}
	m.dialogCursor = len(m.dialogInput)
	m.dialog = dialogFileBrowser
}

func (m *editorModel) loadBrowserDir() {
	entries, err := os.ReadDir(m.browserDir)
	if err != nil {
		m.statusMsg = "Error: " + err.Error()
		return
	}

	m.browserFiles = []string{}

	// Add parent directory if not at root
	if m.browserDir != "/" {
		m.browserFiles = append(m.browserFiles, "..")
	}

	// Separate directories and files
	var dirs, files []string
	for _, entry := range entries {
		name := entry.Name()
		// Skip hidden files
		if strings.HasPrefix(name, ".") {
			continue
		}
		if entry.IsDir() {
			dirs = append(dirs, name+"/")
		} else {
			files = append(files, name)
		}
	}

	// Sort and append
	sort.Strings(dirs)
	sort.Strings(files)
	m.browserFiles = append(m.browserFiles, dirs...)
	m.browserFiles = append(m.browserFiles, files...)

	m.browserIndex = 0
	m.browserScroll = 0
}

func (m editorModel) handleFileBrowserKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	browserHeight := 15 // Number of visible items
	key := msg.String()

	// Escape always closes
	if key == "esc" {
		m.dialog = dialogNone
		return m, nil
	}

	// Tab toggles between file list and input field
	if key == "tab" {
		m.browserPathInput = !m.browserPathInput
		if m.browserPathInput {
			// Entering path input mode - set input to current dir for easy editing
			if m.dialogInput == "" {
				m.dialogInput = m.browserDir
				m.dialogCursor = len(m.dialogInput)
			}
		}
		return m, nil
	}

	// Handle input mode (typing path or filename)
	if m.browserPathInput {
		switch key {
		case "enter":
			// Try to navigate to the typed path or open/save file
			input := m.dialogInput

			// Expand ~ to home directory
			if strings.HasPrefix(input, "~/") {
				home, err := os.UserHomeDir()
				if err == nil {
					input = filepath.Join(home, input[2:])
				}
			} else if input == "~" {
				home, err := os.UserHomeDir()
				if err == nil {
					input = home
				}
			}

			// Make relative paths absolute
			if !filepath.IsAbs(input) {
				input = filepath.Join(m.browserDir, input)
			}

			// Clean the path
			input = filepath.Clean(input)

			// Check what the path is
			info, err := os.Stat(input)
			if err == nil {
				if info.IsDir() {
					// It's a directory - navigate to it
					m.browserDir = input
					m.loadBrowserDir()
					m.dialogInput = ""
					m.dialogCursor = 0
					m.browserPathInput = false
				} else {
					// It's a file - open or save
					if m.browserForSave {
						m.saveFile(input)
					} else {
						m.loadFile(input)
					}
					m.dialog = dialogNone
				}
			} else if m.browserForSave {
				// File doesn't exist but we're saving - check if parent dir exists
				parentDir := filepath.Dir(input)
				if info, err := os.Stat(parentDir); err == nil && info.IsDir() {
					m.saveFile(input)
					m.dialog = dialogNone
				} else {
					m.statusMsg = "Invalid path: " + input
				}
			} else {
				m.statusMsg = "Not found: " + input
			}
			return m, nil

		case "backspace":
			if m.dialogCursor > 0 && len(m.dialogInput) > 0 {
				m.dialogInput = m.dialogInput[:m.dialogCursor-1] + m.dialogInput[m.dialogCursor:]
				m.dialogCursor--
			}

		case "delete":
			if m.dialogCursor < len(m.dialogInput) {
				m.dialogInput = m.dialogInput[:m.dialogCursor] + m.dialogInput[m.dialogCursor+1:]
			}

		case "left":
			if m.dialogCursor > 0 {
				m.dialogCursor--
			}

		case "right":
			if m.dialogCursor < len(m.dialogInput) {
				m.dialogCursor++
			}

		case "home":
			m.dialogCursor = 0

		case "end":
			m.dialogCursor = len(m.dialogInput)

		case "ctrl+u":
			// Clear input
			m.dialogInput = ""
			m.dialogCursor = 0

		default:
			// Type character
			if len(key) == 1 && key[0] >= 32 {
				m.dialogInput = m.dialogInput[:m.dialogCursor] + key + m.dialogInput[m.dialogCursor:]
				m.dialogCursor++
			}
		}
		return m, nil
	}

	// Handle file list mode
	switch key {
	case "up":
		if m.browserIndex > 0 {
			m.browserIndex--
			if m.browserIndex < m.browserScroll {
				m.browserScroll = m.browserIndex
			}
		}

	case "down":
		if m.browserIndex < len(m.browserFiles)-1 {
			m.browserIndex++
			if m.browserIndex >= m.browserScroll+browserHeight {
				m.browserScroll = m.browserIndex - browserHeight + 1
			}
		}

	case "pgup":
		m.browserIndex -= browserHeight
		if m.browserIndex < 0 {
			m.browserIndex = 0
		}
		m.browserScroll = m.browserIndex

	case "pgdown":
		m.browserIndex += browserHeight
		if m.browserIndex >= len(m.browserFiles) {
			m.browserIndex = len(m.browserFiles) - 1
		}
		if m.browserIndex >= m.browserScroll+browserHeight {
			m.browserScroll = m.browserIndex - browserHeight + 1
		}

	case "home":
		m.browserIndex = 0
		m.browserScroll = 0

	case "end":
		m.browserIndex = len(m.browserFiles) - 1
		if m.browserIndex >= browserHeight {
			m.browserScroll = m.browserIndex - browserHeight + 1
		}

	case "enter":
		if len(m.browserFiles) > 0 {
			selected := m.browserFiles[m.browserIndex]

			if selected == ".." {
				// Go up one directory
				m.browserDir = filepath.Dir(m.browserDir)
				m.loadBrowserDir()
			} else if strings.HasSuffix(selected, "/") {
				// Enter directory
				m.browserDir = filepath.Join(m.browserDir, strings.TrimSuffix(selected, "/"))
				m.loadBrowserDir()
			} else {
				// File selected
				fullPath := filepath.Join(m.browserDir, selected)
				if m.browserForSave {
					m.saveFile(fullPath)
				} else {
					m.loadFile(fullPath)
				}
				m.dialog = dialogNone
			}
		}
		return m, nil

	default:
		// Quick search - type first letter to jump to matching file
		if len(key) == 1 && key[0] >= 32 {
			keyLower := strings.ToLower(key)
			for i, file := range m.browserFiles {
				if strings.HasPrefix(strings.ToLower(file), keyLower) {
					m.browserIndex = i
					if m.browserIndex < m.browserScroll {
						m.browserScroll = m.browserIndex
					} else if m.browserIndex >= m.browserScroll+browserHeight {
						m.browserScroll = m.browserIndex - browserHeight + 1
					}
					break
				}
			}
		}
	}

	return m, nil
}

func (m *editorModel) adjustScroll() {
	editorHeight := m.height - 3 // Minus menu bar, title, status bar
	if editorHeight < 1 {
		editorHeight = 1
	}

	if m.cursorY < m.scrollY {
		m.scrollY = m.cursorY
	}
	if m.cursorY >= m.scrollY+editorHeight {
		m.scrollY = m.cursorY - editorHeight + 1
	}
}

func (m editorModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Menu bar
	b.WriteString(m.renderMenuBar())
	b.WriteString("\n")

	// Title bar
	title := fmt.Sprintf(" %s ", m.filename)
	if m.modified {
		title += "*"
	}
	titlePadding := (m.width - len(title)) / 2
	titleBar := strings.Repeat("─", titlePadding) + title + strings.Repeat("─", m.width-titlePadding-len(title))
	b.WriteString(titleStyle.Width(m.width).Render(titleBar))
	b.WriteString("\n")

	// Editor area
	editorHeight := m.height - 4 // Menu, title, status, bottom border
	if editorHeight < 1 {
		editorHeight = 1
	}

	cursorStyle := lipgloss.NewStyle().Background(cyanText).Foreground(blueBackground)
	selectionStyle := lipgloss.NewStyle().Background(lipgloss.Color("7")).Foreground(blueBackground) // Gray bg for selection

	for i := 0; i < editorHeight; i++ {
		lineNum := m.scrollY + i
		var line string
		if lineNum < len(m.lines) {
			line = m.lines[lineNum]
		}

		// Truncate or pad line to fit width
		visualWidth := m.width
		if len(line) > visualWidth {
			line = line[:visualWidth]
		}

		// Build the styled line with selection and cursor support
		var styledLine strings.Builder
		lineLen := len(line)

		// Extend line for rendering purposes (for cursor past end)
		displayLine := line
		if lineNum == m.cursorY && m.cursorX >= lineLen {
			displayLine = line + strings.Repeat(" ", m.cursorX-lineLen+1)
		}

		for x := 0; x < visualWidth; x++ {
			var char string
			if x < len(displayLine) {
				char = string(displayLine[x])
			} else {
				char = " "
			}

			// Determine styling for this character
			isCursor := lineNum == m.cursorY && x == m.cursorX
			isSelected := m.isPositionInSelection(x, lineNum)

			if isCursor {
				styledLine.WriteString(cursorStyle.Render(char))
			} else if isSelected {
				styledLine.WriteString(selectionStyle.Render(char))
			} else {
				styledLine.WriteString(editorStyle.Render(char))
			}
		}

		b.WriteString(styledLine.String())
		b.WriteString("\n")
	}

	// Status bar
	status := fmt.Sprintf(" Line: %d  Col: %d ", m.cursorY+1, m.cursorX+1)
	helpText := "F10=Menu  Alt+X=Exit"
	// Show status message if present
	if m.statusMsg != "" {
		helpText = m.statusMsg
	}
	padding := m.width - len(status) - len(helpText)
	if padding < 0 {
		padding = 0
	}
	statusLine := status + strings.Repeat(" ", padding) + helpText
	b.WriteString(statusBarStyle.Width(m.width).Render(statusLine))

	result := b.String()

	// Render dropdown menu if open
	if m.menuOpen {
		result = m.renderWithMenu(result)
	}

	// Render dialog if open
	if m.dialog != dialogNone {
		result = m.renderWithDialog(result)
	}

	return result
}

func (m editorModel) renderWithDialog(base string) string {
	lines := strings.Split(base, "\n")

	dialogStyle := lipgloss.NewStyle().
		Background(grayBackground).
		Foreground(blackText).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(blackText).
		BorderBackground(grayBackground)

	inputStyle := lipgloss.NewStyle().
		Background(whiteText).
		Foreground(blackText)

	var title, label string
	var showInput bool = true
	var content string

	switch m.dialog {
	case dialogFileBrowser:
		return m.renderFileBrowser(base)
	case dialogOpen:
		title = " Open File "
		label = "Filename: "
	case dialogSaveAs:
		title = " Save As "
		label = "Filename: "
	case dialogFind:
		title = " Find "
		label = "Search for: "
	case dialogReplace:
		title = " Replace "
		if !m.replaceMode {
			label = "Search for: "
		} else {
			label = "Replace with: "
		}
	case dialogConfirmNew:
		title = " New File "
		content = "Discard changes? (Enter=Yes, Esc=No)"
		showInput = false
	case dialogAbout:
		title = " About "
		content = "EDIT.COM Clone\n\nWritten in Go with Bubble Tea\nInspired by MS-DOS EDIT.COM\n\nDBasic Project 2024\n\nPress Esc to close"
		showInput = false
	}

	// Build dialog content
	dialogWidth := 50
	var dialogLines []string
	dialogLines = append(dialogLines, title)

	if showInput {
		// Input field
		inputWidth := dialogWidth - len(label) - 4
		inputText := m.dialogInput
		if len(inputText) > inputWidth {
			inputText = inputText[len(inputText)-inputWidth:]
		}
		// Pad input to full width
		inputText += strings.Repeat(" ", inputWidth-len(inputText))
		dialogLines = append(dialogLines, " "+label+inputStyle.Render(inputText)+" ")
		dialogLines = append(dialogLines, "")
		dialogLines = append(dialogLines, " Enter=OK  Esc=Cancel ")
	} else {
		for _, line := range strings.Split(content, "\n") {
			dialogLines = append(dialogLines, " "+line+" ")
		}
	}

	// Calculate dialog position (center)
	dialogHeight := len(dialogLines) + 2 // +2 for borders
	startY := (m.height - dialogHeight) / 2
	startX := (m.width - dialogWidth - 4) / 2

	// Render dialog box
	boxContent := strings.Join(dialogLines, "\n")
	dialogBox := dialogStyle.Width(dialogWidth).Render(boxContent)
	dialogBoxLines := strings.Split(dialogBox, "\n")
	dialogActualWidth := lipgloss.Width(dialogBoxLines[0])

	// Overlay dialog on base - rebuild each line completely
	for i, dLine := range dialogBoxLines {
		lineNum := startY + i
		if lineNum >= 0 && lineNum < len(lines) {
			// Get the original text content for this line
			origLine := m.getPlainLineContent(lineNum)

			// Convert to runes for proper Unicode handling
			runes := []rune(origLine)
			for len(runes) < m.width {
				runes = append(runes, ' ')
			}

			// Build left part using rune slicing
			var leftPart string
			if startX > 0 {
				if startX <= len(runes) {
					leftPart = string(runes[:startX])
				} else {
					leftPart = string(runes)
				}
			}
			leftPadding := startX - len([]rune(leftPart))
			if leftPadding < 0 {
				leftPadding = 0
			}

			// Build right part using rune slicing
			rightStart := startX + dialogActualWidth
			var rightPart string
			if rightStart < len(runes) {
				rightPart = string(runes[rightStart:])
			}
			rightPadding := m.width - startX - dialogActualWidth - len([]rune(rightPart))
			if rightPadding < 0 {
				rightPadding = 0
			}

			// Rebuild the complete line
			newLine := editorStyle.Render(leftPart + strings.Repeat(" ", leftPadding))
			newLine += dLine
			newLine += editorStyle.Render(rightPart + strings.Repeat(" ", rightPadding))
			lines[lineNum] = newLine
		}
	}

	return strings.Join(lines, "\n")
}

func (m editorModel) renderFileBrowser(base string) string {
	lines := strings.Split(base, "\n")

	// ANSI codes
	editorBg := "\x1b[48;5;17m\x1b[38;5;14m"  // Blue bg, yellow text
	dialogBg := "\x1b[47;30m"                  // Gray bg, black text
	dialogSelBg := "\x1b[46;30m"               // Cyan bg, black text (selected)
	inputBg := "\x1b[47;34m"                   // Gray bg, blue text (directory)
	fileBg := "\x1b[47;30m"                    // Gray bg, black text (file)
	dirBg := "\x1b[47;33m"                     // Gray bg, yellow text (directory)
	reset := "\x1b[0m"

	// Dialog dimensions
	dialogWidth := 60
	browserHeight := 15
	dialogHeight := browserHeight + 6 // Files + header + input + help

	// Calculate position
	startX := (m.width - dialogWidth) / 2
	startY := (m.height - dialogHeight) / 2
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	// Build dialog lines
	title := " Open "
	if m.browserForSave {
		title = " Save As "
	}

	// Helper to build a dialog line with proper Unicode handling
	buildDialogLine := func(lineNum int, content string, contentWidth int) string {
		origLine := m.getPlainLineContent(lineNum)

		// Convert to runes for proper Unicode handling
		runes := []rune(origLine)
		for len(runes) < m.width {
			runes = append(runes, ' ')
		}

		// Left part using rune slicing
		var leftPart string
		if startX > 0 {
			if startX <= len(runes) {
				leftPart = string(runes[:startX])
			} else {
				leftPart = string(runes)
			}
		}
		leftPad := startX - len([]rune(leftPart))
		if leftPad < 0 {
			leftPad = 0
		}

		// Right part using rune slicing
		rightStart := startX + contentWidth
		var rightPart string
		if rightStart < len(runes) {
			rightPart = string(runes[rightStart:])
		}
		rightPad := m.width - startX - contentWidth - len([]rune(rightPart))
		if rightPad < 0 {
			rightPad = 0
		}

		return editorBg + leftPart + strings.Repeat(" ", leftPad) + reset +
			content +
			editorBg + rightPart + strings.Repeat(" ", rightPad) + reset
	}

	lineNum := startY

	// Top border
	topBorder := dialogBg + "╔" + strings.Repeat("═", dialogWidth-2) + "╗" + reset
	if lineNum < len(lines) {
		lines[lineNum] = buildDialogLine(lineNum, topBorder, dialogWidth)
	}
	lineNum++

	// Title line
	titlePad := (dialogWidth - 2 - len(title)) / 2
	titleLine := dialogBg + "║" + strings.Repeat(" ", titlePad) + title + strings.Repeat(" ", dialogWidth-2-titlePad-len(title)) + "║" + reset
	if lineNum < len(lines) {
		lines[lineNum] = buildDialogLine(lineNum, titleLine, dialogWidth)
	}
	lineNum++

	// Directory/Path input line
	var dirLine string
	if m.browserPathInput {
		// Show editable path input field
		pathLabel := "Path: "
		inputWidth := dialogWidth - 4 - len(pathLabel)
		inputText := m.dialogInput
		// Handle cursor position
		visibleStart := 0
		if m.dialogCursor > inputWidth-1 {
			visibleStart = m.dialogCursor - inputWidth + 1
		}
		if visibleStart > len(inputText) {
			visibleStart = 0
		}
		visibleEnd := visibleStart + inputWidth
		if visibleEnd > len(inputText) {
			visibleEnd = len(inputText)
		}
		displayText := ""
		if visibleStart < len(inputText) {
			displayText = inputText[visibleStart:visibleEnd]
		}
		displayText += strings.Repeat(" ", inputWidth-len(displayText))
		// Highlight with cursor indicator
		dirLine = dialogBg + "║ " + pathLabel + "\x1b[44;37m" + displayText + dialogBg + " ║" + reset // Blue bg when editing
	} else {
		// Show current directory (read-only display)
		dirDisplay := m.browserDir
		if len(dirDisplay) > dialogWidth-6 {
			dirDisplay = "..." + dirDisplay[len(dirDisplay)-(dialogWidth-9):]
		}
		dirLine = dialogBg + "║ " + inputBg + dirDisplay + strings.Repeat(" ", dialogWidth-4-len(dirDisplay)) + dialogBg + " ║" + reset
	}
	if lineNum < len(lines) {
		lines[lineNum] = buildDialogLine(lineNum, dirLine, dialogWidth)
	}
	lineNum++

	// Separator
	sepLine := dialogBg + "╟" + strings.Repeat("─", dialogWidth-2) + "╢" + reset
	if lineNum < len(lines) {
		lines[lineNum] = buildDialogLine(lineNum, sepLine, dialogWidth)
	}
	lineNum++

	// File list
	for i := 0; i < browserHeight; i++ {
		fileIdx := m.browserScroll + i
		var fileName string
		isSelected := fileIdx == m.browserIndex

		if fileIdx < len(m.browserFiles) {
			fileName = m.browserFiles[fileIdx]
		}

		// Determine style
		bg := fileBg
		if strings.HasSuffix(fileName, "/") || fileName == ".." {
			bg = dirBg
		}
		if isSelected {
			bg = dialogSelBg
		}

		// Truncate if needed
		displayName := fileName
		maxNameLen := dialogWidth - 4
		if len(displayName) > maxNameLen {
			displayName = displayName[:maxNameLen-3] + "..."
		}

		// Build line content
		fileContent := dialogBg + "║ " + bg + displayName + strings.Repeat(" ", dialogWidth-4-len(displayName)) + dialogBg + " ║" + reset
		if lineNum < len(lines) {
			lines[lineNum] = buildDialogLine(lineNum, fileContent, dialogWidth)
		}
		lineNum++
	}

	// Separator
	if lineNum < len(lines) {
		lines[lineNum] = buildDialogLine(lineNum, sepLine, dialogWidth)
	}
	lineNum++

	// Input line (for save) - show filename field when not in path input mode
	if m.browserForSave && !m.browserPathInput {
		inputLabel := "Filename: "
		inputWidth := dialogWidth - 4 - len(inputLabel)
		inputText := m.dialogInput
		if len(inputText) > inputWidth {
			inputText = inputText[len(inputText)-inputWidth:]
		}
		inputText += strings.Repeat(" ", inputWidth-len(inputText))
		inputLine := dialogBg + "║ " + inputLabel + "\x1b[47;30m" + inputText + dialogBg + " ║" + reset
		if lineNum < len(lines) {
			lines[lineNum] = buildDialogLine(lineNum, inputLine, dialogWidth)
		}
		lineNum++
	}

	// Help line - show different help based on mode
	var helpText string
	if m.browserPathInput {
		helpText = "Enter=Go  Tab=Files  Ctrl+U=Clear  Esc=Cancel"
	} else if m.browserForSave {
		helpText = "Enter=Select  Tab=Path  Esc=Cancel"
	} else {
		helpText = "Enter=Open  Tab=Path  Esc=Cancel"
	}
	if len(helpText) > dialogWidth-4 {
		helpText = helpText[:dialogWidth-4]
	}
	helpLine := dialogBg + "║ " + helpText + strings.Repeat(" ", dialogWidth-4-len(helpText)) + " ║" + reset
	if lineNum < len(lines) {
		lines[lineNum] = buildDialogLine(lineNum, helpLine, dialogWidth)
	}
	lineNum++

	// Bottom border
	bottomBorder := dialogBg + "╚" + strings.Repeat("═", dialogWidth-2) + "╝" + reset
	if lineNum < len(lines) {
		lines[lineNum] = buildDialogLine(lineNum, bottomBorder, dialogWidth)
	}

	return strings.Join(lines, "\n")
}

func (m editorModel) renderMenuBar() string {
	var result strings.Builder

	for i, menu := range menus {
		isSelected := m.menuOpen && i == m.menuIndex

		// Build the menu item text with ANSI codes for hotkey
		var itemText string
		if len(menu.name) > 0 {
			// Hotkey (first letter) in red
			if isSelected {
				itemText = "\x1b[46;31;1m" + string(menu.name[0]) + "\x1b[46;30m" // Cyan bg, red bold -> cyan bg, black
			} else {
				itemText = "\x1b[47;31;1m" + string(menu.name[0]) + "\x1b[47;30m" // Gray bg, red bold -> gray bg, black
			}
			if len(menu.name) > 1 {
				itemText += menu.name[1:]
			}
		}

		// Add spacing and background
		if isSelected {
			result.WriteString("\x1b[46;30m " + itemText + " \x1b[0m")
		} else {
			result.WriteString("\x1b[47;30m " + itemText + " \x1b[0m")
		}
	}

	// Pad to full width
	menuContent := result.String()
	contentWidth := 0
	for _, menu := range menus {
		contentWidth += len(menu.name) + 2 // name + 2 spaces
	}
	padding := m.width - contentWidth
	if padding < 0 {
		padding = 0
	}

	return menuContent + "\x1b[47;30m" + strings.Repeat(" ", padding) + "\x1b[0m"
}

// getMenuXPosition returns the X position where a menu dropdown should appear
func (m editorModel) getMenuXPosition(menuIdx int) int {
	x := 0
	for i := 0; i < menuIdx; i++ {
		x += len(menus[i].name) + 2 // name + 2 spaces padding
	}
	return x
}

func (m editorModel) renderWithMenu(base string) string {
	if !m.menuOpen {
		return base
	}

	lines := strings.Split(base, "\n")

	// Calculate menu X position based on which menu is selected
	menuX := m.getMenuXPosition(m.menuIndex)

	// Build dropdown
	menu := menus[m.menuIndex]
	maxWidth := 0
	for _, item := range menu.items {
		w := len(item.label) + len(item.shortcut) + 4
		if w > maxWidth {
			maxWidth = w
		}
	}
	// Minimum width for aesthetics
	if maxWidth < 20 {
		maxWidth = 20
	}

	// ANSI codes for colors
	editorBg := "\x1b[48;5;17m\x1b[38;5;14m" // Blue bg, yellow text
	titleBg := "\x1b[48;5;17m\x1b[38;5;15m"  // Blue bg, white text (for title bar)
	menuBg := "\x1b[47;30m"                   // Gray bg, black text
	menuSelBg := "\x1b[46;30m"                // Cyan bg, black text
	hotkey := "\x1b[31;1m"                    // Red bold for hotkey
	reset := "\x1b[0m"

	// Helper to build a complete line with menu overlay - ensures full width coverage
	buildLine := func(lineNum int, menuContent string, menuWidth int) string {
		origLine := m.getPlainLineContent(lineNum)

		// Convert to runes for proper Unicode handling
		runes := []rune(origLine)

		// Ensure line is padded to screen width (in runes/characters)
		for len(runes) < m.width {
			runes = append(runes, ' ')
		}

		// Left part (before menu) - use rune slicing
		var leftPart string
		if menuX > 0 {
			if menuX <= len(runes) {
				leftPart = string(runes[:menuX])
			} else {
				leftPart = string(runes)
			}
		}

		// Right part (after menu) - use rune slicing
		rightStart := menuX + menuWidth
		var rightPart string
		if rightStart < len(runes) {
			rightPart = string(runes[rightStart:])
		}

		// Ensure right part fills to screen edge (in runes)
		rightRunes := []rune(rightPart)
		totalLen := menuX + menuWidth + len(rightRunes)
		if totalLen < m.width {
			rightPart += strings.Repeat(" ", m.width-totalLen)
		}

		// Use title style for line 1, editor style for others
		bg := editorBg
		if lineNum == 1 {
			bg = titleBg
		}

		return bg + leftPart + reset + menuContent + bg + rightPart + reset
	}

	// Title bar line (line 1) - top border of dropdown
	topBorder := menuBg + "┌" + strings.Repeat("─", maxWidth-2) + "┐" + reset
	if len(lines) > 1 {
		lines[1] = buildLine(1, topBorder, maxWidth)
	}

	// Menu items with hotkey highlighting
	for i, item := range menu.items {
		lineNum := i + 2
		if lineNum >= len(lines) {
			break
		}

		var itemContent string
		if item.label == "-" {
			itemContent = menuBg + "├" + strings.Repeat("─", maxWidth-2) + "┤" + reset
		} else {
			bg := menuBg
			if i == m.submenuIndex {
				bg = menuSelBg
			}

			// Build label with hotkey highlighted (first letter in red)
			labelWithHotkey := ""
			if len(item.label) > 0 {
				labelWithHotkey = hotkey + string(item.label[0]) + reset + bg
				if len(item.label) > 1 {
					labelWithHotkey += item.label[1:]
				}
			}

			// Calculate padding (based on actual label length, not ANSI codes)
			labelPad := maxWidth - len(item.label) - len(item.shortcut) - 2
			if labelPad < 0 {
				labelPad = 0
			}

			itemContent = bg + "│" + labelWithHotkey + strings.Repeat(" ", labelPad) + item.shortcut + "│" + reset
		}

		lines[lineNum] = buildLine(lineNum, itemContent, maxWidth)
	}

	// Bottom border
	bottomLineNum := len(menu.items) + 2
	if bottomLineNum < len(lines) {
		bottomBorder := menuBg + "└" + strings.Repeat("─", maxWidth-2) + "┘" + reset
		lines[bottomLineNum] = buildLine(bottomLineNum, bottomBorder, maxWidth)
	}

	return strings.Join(lines, "\n")
}

// getPlainLineContent returns the text content of a line (editor content)
func (m editorModel) getPlainLineContent(screenLine int) string {
	// screenLine 0 = menu bar
	// screenLine 1 = title bar
	// screenLine 2+ = editor content

	if screenLine == 0 {
		return "" // Menu bar - handled separately
	}

	if screenLine == 1 {
		// Title bar - return the title content
		title := fmt.Sprintf(" %s ", m.filename)
		if m.modified {
			title += "*"
		}
		titlePadding := (m.width - len(title)) / 2
		return strings.Repeat("─", titlePadding) + title + strings.Repeat("─", m.width-titlePadding-len(title))
	}

	editorLine := screenLine - 2 + m.scrollY
	if editorLine < 0 || editorLine >= len(m.lines) {
		return ""
	}

	return m.lines[editorLine]
}

func main() {
	m := initialModel()

	// Load file if specified
	if len(os.Args) > 1 {
		filename := os.Args[1]
		content, err := os.ReadFile(filename)
		if err == nil {
			m.filename = filename
			m.lines = strings.Split(string(content), "\n")
			if len(m.lines) == 0 {
				m.lines = []string{""}
			}
		}
	}

	// Use WithAltScreen to restore terminal on exit
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
