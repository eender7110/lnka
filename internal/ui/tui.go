package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ANSI color codes for terminal styling
const (
	colorReset  = "\033[0m"
	colorPrompt = "\033[1;32m" // Bold Green
	colorDim    = "\033[2;90m" // Dim Gray
	colorHelp   = "\033[90m"   // Gray
	colorCursor = "\033[7m"    // Reverse video
	colorNormal = "\033[0m"    // Normal/White
)

// keyBindings defines all keyboard shortcuts for the multi-select UI
type keyBindings struct {
	quit         []string // Quit/abort shortcuts
	confirm      string   // Confirm selection
	filter       string   // Enter filter mode
	hideToggle   string   // Toggle hide unlinked items
	toggleSelect string   // Toggle item selection
	up           string   // Move cursor up
	down         string   // Move cursor down
	backspace    string   // Delete character in filter mode
}

// defaultKeyBindings contains the default keyboard shortcuts
var defaultKeyBindings = keyBindings{
	quit:         []string{"ctrl+c", "esc"},
	confirm:      "enter",
	filter:       "/",
	hideToggle:   "h",
	toggleSelect: " ",
	up:           "up",
	down:         "down",
	backspace:    "backspace",
}

// multiSelectModel is the Bubble Tea model for multi-select UI
// It manages the state for selecting multiple items from a list
type multiSelectModel struct {
	choices              []string        // Available choices
	choicesLower         []string        // Lowercase versions for efficient filtering
	selected             map[string]bool // Selected items
	selectedOrder        []string        // Order of selection for result
	selectedIndex        map[string]int  // Maps choice to position in selectedOrder (for O(1) removal)
	cursor               int             // Cursor position
	filter               string          // Filter text
	filtering            bool            // Filter mode active
	filtered             []string        // Filtered choices
	aborted              bool            // User pressed ESC
	title                string          // Optional title
	maxVisibleItems      int             // Maximum items to show before pagination
	hideUnlinked         bool            // Hide unlinked items when true
	cachedVisibleChoices []string        // Cached result of getVisibleChoices
	cacheValid           bool            // Whether the cache is valid
	keys                 keyBindings     // Keyboard shortcuts configuration
}

// Init initializes the model
func (m multiSelectModel) Init() tea.Cmd {
	return nil
}

// isKey checks if the pressed key matches any of the given keys
func isKey(pressed string, keys ...string) bool {
	for _, key := range keys {
		if pressed == key {
			return true
		}
	}
	return false
}

// Update handles messages
func (m multiSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Invalidate cache at the start of each update cycle
	m.cacheValid = false

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Handle quit keys
		if isKey(key, m.keys.quit...) {
			m.aborted = true
			return m, tea.Quit
		}

		// Handle confirm key
		if key == m.keys.confirm {
			if !m.filtering {
				return m, tea.Quit
			}
			// Exit filter mode on enter, keep the filter
			m.filtering = false
			m.clampCursor()
			return m, nil
		}

		// Handle filter key
		if key == m.keys.filter {
			if !m.filtering {
				m.filtering = true
				// Don't clear the existing filter, just allow editing
				return m, nil
			}
		}

		// Handle up key
		if key == m.keys.up {
			if !m.filtering && m.cursor > 0 {
				m.cursor--
			}
		}

		// Handle down key
		if key == m.keys.down {
			if !m.filtering {
				choices := m.getVisibleChoices()
				if m.cursor < len(choices)-1 {
					m.cursor++
				}
			}
		}

		// Handle toggle select key
		if key == m.keys.toggleSelect {
			if !m.filtering {
				m.handleToggleSelection()
			}
		}

		// Handle hide toggle key
		if key == m.keys.hideToggle {
			if !m.filtering {
				// Only allow toggle if there are selected items
				if len(m.selected) > 0 {
					// Remember current item before toggle
					choices := m.getVisibleChoices()
					var currentItem string
					if m.cursor >= 0 && m.cursor < len(choices) {
						currentItem = choices[m.cursor]
					}

					// Toggle mode
					m.hideUnlinked = !m.hideUnlinked

					// Try to keep cursor on the same item
					newChoices := m.getVisibleChoices()
					if currentItem != "" {
						for i, choice := range newChoices {
							if choice == currentItem {
								m.cursor = i
								return m, nil
							}
						}
					}
					// If item not found, clamp cursor
					m.clampCursor()
				}
			}
		}

		// Handle backspace key
		if key == m.keys.backspace {
			if m.filtering && len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.updateFiltered()
				m.clampCursor()
			}
		}

		// Add character to filter (if no other key matched)
		if m.filtering && len(key) == 1 {
			m.filter += key
			m.updateFiltered()
			m.clampCursor()
		}
	}
	return m, nil
}

// View renders the UI
func (m multiSelectModel) View() string {
	if m.aborted {
		return ""
	}

	var b strings.Builder

	// Title (optional)
	if m.title != "" {
		b.WriteString(m.title)
		b.WriteString("\n\n")
	}

	// Filter prompt
	if m.filtering {
		b.WriteString(colorPrompt)
		b.WriteString("$ ")
		b.WriteString(colorReset)
		b.WriteString(m.filter)
		b.WriteString(colorCursor)
		b.WriteString(" ")
		b.WriteString(colorReset)
		b.WriteString("\n\n")
	}

	// Add separator line before list if no title or filter prompt is shown
	if m.title == "" && !m.filtering {
		b.WriteString("\n")
	}

	// Choices
	choices := m.getVisibleChoices()
	visibleStart := 0
	visibleEnd := len(choices)

	// Pagination
	if len(choices) > m.maxVisibleItems {
		if m.cursor >= m.maxVisibleItems {
			visibleStart = m.cursor - m.maxVisibleItems + 1
		}
		visibleEnd = visibleStart + m.maxVisibleItems
		if visibleEnd > len(choices) {
			visibleEnd = len(choices)
			visibleStart = max(0, visibleEnd-m.maxVisibleItems)
		}
	}

	for i := visibleStart; i < visibleEnd; i++ {
		choice := choices[i]
		cursor := " "
		if i == m.cursor && !m.filtering {
			cursor = "▶"
		}

		// Text color: dim gray if not selected, normal if selected
		textStyle := colorDim
		if m.selected[choice] {
			textStyle = colorNormal
		}

		b.WriteString(cursor)
		b.WriteString(" ")
		b.WriteString(textStyle)
		b.WriteString(choice)
		b.WriteString(colorReset)
		b.WriteString("\n")
	}

	// Help text with pagination info
	b.WriteString("\n")
	b.WriteString(colorHelp)

	// Build help text starting with pagination info if applicable
	helpText := ""
	if len(choices) > m.maxVisibleItems {
		helpText = fmt.Sprintf("%d-%d of %d | ", visibleStart+1, visibleEnd, len(choices))
	}

	if !m.filtering {
		helpText += "space: toggle | /: filter"
		// Add h: option only if there are selected items
		if len(m.selected) > 0 {
			if m.hideUnlinked {
				helpText += " | h: show all"
			} else {
				helpText += " | h: linked only"
			}
		}
		helpText += " | enter: confirm | esc: abort"
	} else {
		helpText += "type to filter | enter: exit filter | esc: abort"
	}

	b.WriteString(helpText)
	b.WriteString(colorReset)

	return b.String()
}

// getVisibleChoices returns filtered or all choices, respecting hideUnlinked mode
// Results are cached within a single update cycle for performance
func (m *multiSelectModel) getVisibleChoices() []string {
	// Return cached result if valid
	if m.cacheValid {
		return m.cachedVisibleChoices
	}

	var baseChoices []string

	// Start with filtered or all choices
	if m.filter != "" {
		baseChoices = m.filtered
	} else {
		baseChoices = m.choices
	}

	// Apply hideUnlinked filter if active
	var result []string
	if m.hideUnlinked {
		visible := make([]string, 0, len(m.selected))
		for _, choice := range baseChoices {
			if m.selected[choice] {
				visible = append(visible, choice)
			}
		}
		result = visible
	} else {
		result = baseChoices
	}

	// Cache the result
	m.cachedVisibleChoices = result
	m.cacheValid = true

	return result
}

// updateFiltered updates the filtered list
func (m *multiSelectModel) updateFiltered() {
	m.filtered = []string{}
	filterLower := strings.ToLower(m.filter)
	for i, choice := range m.choices {
		if strings.Contains(m.choicesLower[i], filterLower) {
			m.filtered = append(m.filtered, choice)
		}
	}
}

// clampCursor ensures cursor is within valid range
func (m *multiSelectModel) clampCursor() {
	choices := m.getVisibleChoices()
	if len(choices) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor >= len(choices) {
		m.cursor = len(choices) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// handleToggleSelection handles space key press to toggle item selection
func (m *multiSelectModel) handleToggleSelection() {
	choices := m.getVisibleChoices()
	if m.cursor < 0 || m.cursor >= len(choices) {
		return
	}

	choice := choices[m.cursor]
	if m.selected[choice] {
		m.deselectItem(choice, m.cursor)
	} else {
		m.selectItem(choice)
	}
}

// selectItem marks an item as selected
func (m *multiSelectModel) selectItem(choice string) {
	m.selected[choice] = true
	m.selectedIndex[choice] = len(m.selectedOrder)
	m.selectedOrder = append(m.selectedOrder, choice)
}

// deselectItem removes an item from selection
func (m *multiSelectModel) deselectItem(choice string, currentCursor int) {
	delete(m.selected, choice)
	m.removeFromOrder(choice)

	if m.hideUnlinked {
		m.handleHideUnlinkedAfterDeselect(choice, currentCursor)
	}
}

// removeFromOrder removes a choice from the selectedOrder slice using O(1) index lookup
func (m *multiSelectModel) removeFromOrder(choice string) {
	idx, exists := m.selectedIndex[choice]
	if !exists {
		return
	}

	// Remove from order slice
	m.selectedOrder = append(m.selectedOrder[:idx], m.selectedOrder[idx+1:]...)
	delete(m.selectedIndex, choice)

	// Update indices for all items after the removed one
	for i := idx; i < len(m.selectedOrder); i++ {
		m.selectedIndex[m.selectedOrder[i]] = i
	}
}

// handleHideUnlinkedAfterDeselect handles cursor positioning after deselecting in hideUnlinked mode
func (m *multiSelectModel) handleHideUnlinkedAfterDeselect(deselectedChoice string, currentCursor int) {
	choices := m.getVisibleChoices()

	// Check if there are any linked items left
	hasLinkedItems := false
	for _, c := range choices {
		if m.selected[c] {
			hasLinkedItems = true
			break
		}
	}

	if !hasLinkedItems {
		// Switch to "show all" and position cursor on the deselected item
		m.switchToShowAllMode(deselectedChoice)
	} else {
		// Adjust cursor after item disappears from list
		m.adjustCursorAfterItemRemoved(currentCursor)
	}
}

// switchToShowAllMode switches from "linked only" to "show all" mode
func (m *multiSelectModel) switchToShowAllMode(cursorItem string) {
	m.hideUnlinked = false
	newChoices := m.getVisibleChoices()

	// Find the item in the new list and position cursor on it
	for i, c := range newChoices {
		if c == cursorItem {
			m.cursor = i
			return
		}
	}

	// Fallback: clamp cursor if item not found
	m.clampCursor()
}

// adjustCursorAfterItemRemoved adjusts cursor position after an item is removed from visible list
func (m *multiSelectModel) adjustCursorAfterItemRemoved(previousCursor int) {
	newChoices := m.getVisibleChoices()
	if len(newChoices) == 0 {
		m.cursor = 0
		return
	}

	// Try to stay at same index, or move to previous if at end
	if previousCursor >= len(newChoices) {
		m.cursor = len(newChoices) - 1
	} else {
		m.cursor = previousCursor
	}
}

// ShowMultiSelect displays a multi-select UI for choosing files to enable
func ShowMultiSelect(availableFiles []string, currentlyEnabled []string, title string, maxVisibleItems int) ([]string, error) {
	if len(availableFiles) == 0 {
		return nil, fmt.Errorf("no files available to enable")
	}

	// Create initial selection map, order, and index
	selected := make(map[string]bool)
	selectedOrder := []string{}
	selectedIndex := make(map[string]int)
	for i, file := range currentlyEnabled {
		selected[file] = true
		selectedOrder = append(selectedOrder, file)
		selectedIndex[file] = i
	}

	// Pre-compute lowercase versions for efficient filtering
	choicesLower := make([]string, len(availableFiles))
	for i, choice := range availableFiles {
		choicesLower[i] = strings.ToLower(choice)
	}

	// Find initial cursor position (first selected item if any)
	initialCursor := 0
	if len(currentlyEnabled) > 0 {
		// Find the position of the first selected item in availableFiles
		firstSelected := currentlyEnabled[0]
		for i, file := range availableFiles {
			if file == firstSelected {
				initialCursor = i
				break
			}
		}
	}

	// Create model
	m := multiSelectModel{
		choices:         availableFiles,
		choicesLower:    choicesLower,
		selected:        selected,
		selectedOrder:   selectedOrder,
		selectedIndex:   selectedIndex,
		cursor:          initialCursor,
		title:           title,
		maxVisibleItems: maxVisibleItems,
		keys:            defaultKeyBindings,
	}

	// Run the program
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("program error: %w", err)
	}

	// Type assert with check
	model, ok := finalModel.(multiSelectModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}

	// Check if aborted
	if model.aborted {
		return nil, fmt.Errorf("user aborted")
	}

	// Return selected items in order
	return model.selectedOrder, nil
}

// confirmModel is the Bubble Tea model for confirmation dialog
// It manages the state for a yes/no confirmation prompt
type confirmModel struct {
	message  string
	selected bool // true = yes, false = no
	aborted  bool
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.aborted = true
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case "left":
			m.selected = true
		case "right":
			m.selected = false
		case "y", "Y":
			m.selected = true
			return m, tea.Quit
		case "n", "N":
			m.selected = false
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	if m.aborted {
		return ""
	}

	var b strings.Builder
	b.WriteString(m.message)
	b.WriteString("\n\n")

	yesStyle := "[ Yes ]"
	noStyle := "[ No ]"

	if m.selected {
		yesStyle = colorPrompt + "[ Yes ]" + colorReset
	} else {
		noStyle = colorPrompt + "[ No ]" + colorReset
	}

	b.WriteString(yesStyle)
	b.WriteString("  ")
	b.WriteString(noStyle)
	b.WriteString("\n\n")
	b.WriteString(colorHelp)
	b.WriteString("arrows: move | enter/y/n: select | esc: abort")
	b.WriteString(colorReset)

	return b.String()
}

// ShowConfirmation displays a confirmation dialog
func ShowConfirmation(message string) (bool, error) {
	m := confirmModel{
		message:  message,
		selected: true, // Default to Yes
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return false, fmt.Errorf("program error: %w", err)
	}

	// Type assert with check
	model, ok := finalModel.(confirmModel)
	if !ok {
		return false, fmt.Errorf("unexpected model type")
	}

	if model.aborted {
		return false, fmt.Errorf("user aborted")
	}

	return model.selected, nil
}
