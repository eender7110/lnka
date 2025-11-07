# TODO - Code Improvements

## Code Review: internal/ui/tui.go

### ✅ Stärken

- **Saubere Architektur** - Gute Trennung zwischen Model, Update und View (Bubble Tea Pattern)
- **Gute Kommentare** - Code ist gut dokumentiert
- **Effiziente Filterung** - Pre-computed lowercase Strings für Case-Insensitive Search
- **Robuste Fehlerbehandlung** - Type assertions mit Checks

---

## 🚀 Verbesserungsvorschläge

### 1. Code-Duplizierung in Space-Handler (Zeilen 78-133) - HOCH

**Problem:** Die Space-Handler Logik ist sehr verschachtelt und schwer zu lesen.

**Vorschlag:** Extrahieren in separate Methoden:

```go
case " ":
    if !m.filtering {
        m.handleToggleSelection()
    }
```

Mit neuen Methoden:
```go
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

func (m *multiSelectModel) selectItem(choice string) {
    m.selected[choice] = true
    m.selectedOrder = append(m.selectedOrder, choice)
}

func (m *multiSelectModel) deselectItem(choice string, currentCursor int) {
    delete(m.selected, choice)
    m.removeFromOrder(choice)

    if m.hideUnlinked {
        m.handleHideUnlinkedAfterDeselect(choice, currentCursor)
    }
}
```

**Dateien:** `internal/ui/tui.go:78-133`

---

### 2. Ineffiziente Suche in selectedOrder (Zeilen 89-93) - HOCH

**Problem:** Linear search durch slice bei jedem Deselect - O(n) Komplexität.

**Vorschlag:** Verwenden Sie einen Index für O(1) Lookup:

```go
type multiSelectModel struct {
    // ... existing fields
    selectedIndex map[string]int // Maps choice to position in selectedOrder
}

func (m *multiSelectModel) removeFromOrder(choice string) {
    if idx, exists := m.selectedIndex[choice]; exists {
        m.selectedOrder = append(m.selectedOrder[:idx], m.selectedOrder[idx+1:]...)
        delete(m.selectedIndex, choice)

        // Update indices for remaining items
        for i := idx; i < len(m.selectedOrder); i++ {
            m.selectedIndex[m.selectedOrder[i]] = i
        }
    }
}

func (m *multiSelectModel) selectItem(choice string) {
    m.selected[choice] = true
    m.selectedIndex[choice] = len(m.selectedOrder)
    m.selectedOrder = append(m.selectedOrder, choice)
}
```

**Dateien:** `internal/ui/tui.go:22-35, 89-93`

---

### 3. Duplizierte getVisibleChoices() Aufrufe - MITTEL

**Problem:** `getVisibleChoices()` wird mehrfach in derselben Update-Iteration aufgerufen, was bei großen Listen Performance-Probleme verursachen kann.

**Vorschlag:** Caching innerhalb einer Update-Iteration:

```go
type multiSelectModel struct {
    // ... existing fields
    cachedVisibleChoices []string
    cacheValid           bool
}

func (m *multiSelectModel) getVisibleChoices() []string {
    if m.cacheValid {
        return m.cachedVisibleChoices
    }

    var baseChoices []string
    if m.filter != "" {
        baseChoices = m.filtered
    } else {
        baseChoices = m.choices
    }

    if m.hideUnlinked {
        visible := make([]string, 0, len(m.selected))
        for _, choice := range baseChoices {
            if m.selected[choice] {
                visible = append(visible, choice)
            }
        }
        m.cachedVisibleChoices = visible
        m.cacheValid = true
        return visible
    }

    m.cachedVisibleChoices = baseChoices
    m.cacheValid = true
    return baseChoices
}

func (m multiSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    m.cacheValid = false // Invalidate at start of update
    // ... rest of update logic
}
```

**Dateien:** `internal/ui/tui.go:267-290, 43`

---

### 4. Magic Strings für Key Bindings - MITTEL

**Problem:** Keys sind hardcoded als Strings, schwer zu ändern/konfigurieren.

**Vorschlag:** Konstanten oder Konfiguration:

```go
type keyBindings struct {
    quit         []string
    confirm      []string
    filter       string
    hideToggle   string
    toggleSelect string
    up           string
    down         string
}

var defaultKeyBindings = keyBindings{
    quit:         []string{"ctrl+c", "esc"},
    confirm:      []string{"enter"},
    filter:       "/",
    hideToggle:   "h",
    toggleSelect: " ",
    up:           "up",
    down:         "down",
}

type multiSelectModel struct {
    // ... existing fields
    keys keyBindings
}
```

**Dateien:** `internal/ui/tui.go:46-161`

---

### 5. Fehlende Edge Case Behandlung - NIEDRIG

**Problem:** Was passiert bei leerer `hideUnlinked` Liste? Kein visuelles Feedback.

**Vorschlag:** Visuelles Feedback hinzufügen:

```go
func (m multiSelectModel) View() string {
    // ... existing code

    choices := m.getVisibleChoices()

    // Show message when hideUnlinked is active but list is empty
    if m.hideUnlinked && len(choices) == 0 {
        b.WriteString(colorDim)
        b.WriteString("No linked items to display\n")
        b.WriteString("Press 'h' to show all items\n")
        b.WriteString(colorReset)
        b.WriteString("\n")
    }

    // ... rest of view logic
}
```

**Dateien:** `internal/ui/tui.go:206-222`

---

### 6. Pagination Indikator fehlt - HOCH

**Problem:** User sieht nicht, dass es mehr Items gibt (wichtig für UX).

**Vorschlag:** Pagination Hinweis hinzufügen:

```go
// After the item list (after line 242)
if len(choices) > m.maxVisibleItems {
    b.WriteString("\n")
    b.WriteString(colorDim)
    b.WriteString(fmt.Sprintf("Showing %d-%d of %d items",
        visibleStart+1, visibleEnd, len(choices)))
    if visibleStart > 0 {
        b.WriteString(" (↑ for more)")
    }
    if visibleEnd < len(choices) {
        b.WriteString(" (↓ for more)")
    }
    b.WriteString(colorReset)
}
```

**Dateien:** `internal/ui/tui.go:242-243`

---

### 7. Performance bei großen Listen - NIEDRIG

**Problem:** `hideUnlinked` Filter iteriert bei jedem Aufruf über alle Choices, auch wenn alle Selected Items schon gefunden wurden.

**Vorschlag:** Früher Abbruch bei bekannter Größe:

```go
// In getVisibleChoices() method
if m.hideUnlinked {
    visible := make([]string, 0, len(m.selected))
    selectedCount := len(m.selected)

    for _, choice := range baseChoices {
        if m.selected[choice] {
            visible = append(visible, choice)
            if len(visible) == selectedCount {
                break // All selected items found, no need to continue
            }
        }
    }
    return visible
}
```

**Dateien:** `internal/ui/tui.go:279-287`

---

### 8. Test Coverage - MITTEL

**Problem:** Keine Unit Tests vorhanden für kritische Funktionen.

**Vorschlag:** Tests für kritische Funktionen hinzufügen:

```go
// File: internal/ui/tui_test.go

package ui

import "testing"

func TestGetVisibleChoices_HideUnlinked(t *testing.T) {
    m := &multiSelectModel{
        choices:      []string{"a", "b", "c", "d"},
        selected:     map[string]bool{"b": true, "d": true},
        hideUnlinked: true,
    }

    visible := m.getVisibleChoices()

    if len(visible) != 2 {
        t.Errorf("expected 2 visible choices, got %d", len(visible))
    }

    expected := map[string]bool{"b": true, "d": true}
    for _, v := range visible {
        if !expected[v] {
            t.Errorf("unexpected item in visible: %s", v)
        }
    }
}

func TestGetVisibleChoices_WithFilter(t *testing.T) {
    m := &multiSelectModel{
        choices:      []string{"apple", "banana", "apricot", "berry"},
        choicesLower: []string{"apple", "banana", "apricot", "berry"},
        filter:       "ap",
        filtered:     []string{"apple", "apricot"},
        selected:     map[string]bool{"apple": true},
        hideUnlinked: false,
    }

    visible := m.getVisibleChoices()

    if len(visible) != 2 {
        t.Errorf("expected 2 visible choices, got %d", len(visible))
    }
}

func TestGetVisibleChoices_FilterAndHide(t *testing.T) {
    m := &multiSelectModel{
        choices:      []string{"apple", "banana", "apricot", "berry"},
        choicesLower: []string{"apple", "banana", "apricot", "berry"},
        filter:       "ap",
        filtered:     []string{"apple", "apricot"},
        selected:     map[string]bool{"apple": true},
        hideUnlinked: true,
    }

    visible := m.getVisibleChoices()

    if len(visible) != 1 {
        t.Errorf("expected 1 visible choice, got %d", len(visible))
    }
    if len(visible) > 0 && visible[0] != "apple" {
        t.Errorf("expected 'apple', got '%s'", visible[0])
    }
}

func TestClampCursor_EmptyList(t *testing.T) {
    m := &multiSelectModel{
        choices: []string{},
        cursor:  5,
    }

    m.clampCursor()

    if m.cursor != 0 {
        t.Errorf("expected cursor 0 for empty list, got %d", m.cursor)
    }
}

func TestClampCursor_OutOfBounds(t *testing.T) {
    m := &multiSelectModel{
        choices: []string{"a", "b", "c"},
        cursor:  10,
    }

    m.clampCursor()

    if m.cursor != 2 {
        t.Errorf("expected cursor 2 (last item), got %d", m.cursor)
    }
}
```

**Neue Datei:** `internal/ui/tui_test.go`

---

### 9. Accessibility Verbesserungen - NIEDRIG

**Problem:** Nur grundlegende Tastenkombinationen verfügbar.

**Vorschlag:** Zusätzliche Tastenkombinationen für Power-User:

```go
// In Update() method, add these cases:

case "j":
    // Vim-style down
    if !m.filtering && m.cursor < len(m.getVisibleChoices())-1 {
        m.cursor++
    }

case "k":
    // Vim-style up
    if !m.filtering && m.cursor > 0 {
        m.cursor--
    }

case "g":
    // Go to top
    if !m.filtering {
        m.cursor = 0
    }

case "G":
    // Go to bottom
    if !m.filtering {
        choices := m.getVisibleChoices()
        m.cursor = len(choices) - 1
    }

case "ctrl+a":
    // Select all visible
    if !m.filtering {
        choices := m.getVisibleChoices()
        for _, choice := range choices {
            if !m.selected[choice] {
                m.selected[choice] = true
                m.selectedOrder = append(m.selectedOrder, choice)
            }
        }
    }

case "ctrl+d":
    // Deselect all
    if !m.filtering {
        m.selected = make(map[string]bool)
        m.selectedOrder = []string{}
        if m.hideUnlinked {
            m.hideUnlinked = false
            m.clampCursor()
        }
    }

case "pgdown", "ctrl+f":
    // Page down
    if !m.filtering {
        choices := m.getVisibleChoices()
        m.cursor = min(m.cursor + m.maxVisibleItems, len(choices)-1)
    }

case "pgup", "ctrl+b":
    // Page up
    if !m.filtering {
        m.cursor = max(m.cursor - m.maxVisibleItems, 0)
    }
```

**Dateien:** `internal/ui/tui.go:46-161`

**Help Text Update:**
```go
helpText := "space: toggle | j/k/↑/↓: move | /: filter | g/G: top/bottom"
```

---

### 10. Dokumentation verbessern - NIEDRIG

**Problem:** Keine Paket-Level Dokumentation.

**Vorschlag:** Paket-Level Dokumentation hinzufügen:

```go
// Package ui provides terminal user interface components using Bubble Tea.
//
// The package includes:
//   - Multi-select list with filtering and hiding capabilities
//   - Confirmation dialogs for yes/no prompts
//   - Keyboard navigation and visual feedback
//
// Key features:
//   - Filter mode: Press '/' to search through items
//   - Hide mode: Press 'h' to toggle between all/linked items
//   - Pagination: Automatically handles large lists
//   - Smart cursor positioning: Maintains cursor position across mode switches
//
// Multi-Select UI
//
// The multi-select interface allows users to select multiple items from a list
// with keyboard navigation. Features include:
//
//   - Space: Toggle item selection
//   - Up/Down or j/k: Navigate items
//   - /: Enter filter mode to search
//   - h: Toggle between showing all items or only linked items
//   - Enter: Confirm selection
//   - Esc: Abort
//
// Example usage:
//
//   files := []string{"file1.txt", "file2.txt", "file3.txt"}
//   enabled := []string{"file1.txt"}
//   selected, err := ui.ShowMultiSelect(files, enabled, "Select files", 10)
//   if err != nil {
//       // Handle error
//   }
//   // Use selected files
//
// Confirmation Dialog
//
// The confirmation dialog shows a simple yes/no prompt:
//
//   confirmed, err := ui.ShowConfirmation("Delete all files?")
//   if err != nil {
//       // Handle error
//   }
//   if confirmed {
//       // Perform action
//   }
package ui
```

**Dateien:** `internal/ui/tui.go:1` (vor package declaration)

---

## 📊 Prioritäten

### Hoch (sofort angehen)
- [x] #1 - Code-Duplizierung in Space-Handler ✅ (implementiert 2025-11-07)
- [x] #2 - Ineffiziente Suche in selectedOrder ✅ (implementiert 2025-11-07)
- [x] #6 - Pagination Indikator fehlt ✅ (implementiert 2025-11-07)

### Mittel (bald implementieren)
- [x] #3 - Duplizierte getVisibleChoices() Aufrufe ✅ (implementiert 2025-11-07)
- [x] #4 - Magic Strings für Key Bindings ✅ (implementiert 2025-11-07)
- [x] #8 - Test Coverage ✅ (implementiert 2025-11-07, 19.6% coverage)

### Niedrig (nice-to-have)
- [ ] #5 - Fehlende Edge Case Behandlung
- [ ] #7 - Performance bei großen Listen
- [ ] #9 - Accessibility Verbesserungen
- [ ] #10 - Dokumentation verbessern

---

## 📝 Notizen

- Alle Vorschläge sind rückwärtskompatibel
- Tests sollten vor Refactoring geschrieben werden
- Performance-Verbesserungen sind besonders wichtig bei >1000 Items
- Accessibility-Features können schrittweise hinzugefügt werden

---

*Erstellt am: 2025-11-07*
*Review von: Claude Code*
