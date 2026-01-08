package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ListItem represents an item in a selectable list
type ListItem struct {
	ID          string
	Title       string
	Description string
	Icon        string
	Disabled    bool
	Selected    bool
}

// List renders a selectable list
type List struct {
	Items    []ListItem
	Cursor   int
	Width    int
	Height   int
	ShowDesc bool
}

// NewList creates a new list
func NewList(items []ListItem) *List {
	return &List{
		Items:    items,
		Cursor:   0,
		Width:    60,
		Height:   10,
		ShowDesc: true,
	}
}

// Up moves cursor up
func (l *List) Up() {
	if l.Cursor > 0 {
		l.Cursor--
		// Skip disabled items
		for l.Cursor > 0 && l.Items[l.Cursor].Disabled {
			l.Cursor--
		}
	}
}

// Down moves cursor down
func (l *List) Down() {
	if l.Cursor < len(l.Items)-1 {
		l.Cursor++
		// Skip disabled items
		for l.Cursor < len(l.Items)-1 && l.Items[l.Cursor].Disabled {
			l.Cursor++
		}
	}
}

// Toggle toggles selection of current item
func (l *List) Toggle() {
	if l.Cursor < len(l.Items) && !l.Items[l.Cursor].Disabled {
		l.Items[l.Cursor].Selected = !l.Items[l.Cursor].Selected
	}
}

// GetSelected returns selected items
func (l *List) GetSelected() []ListItem {
	var selected []ListItem
	for _, item := range l.Items {
		if item.Selected {
			selected = append(selected, item)
		}
	}
	return selected
}

// GetCurrent returns the current item
func (l *List) GetCurrent() *ListItem {
	if l.Cursor < len(l.Items) {
		return &l.Items[l.Cursor]
	}
	return nil
}

// Render renders the list
func (l *List) Render() string {
	var lines []string

	// Calculate visible range
	start := 0
	end := len(l.Items)
	if end > l.Height {
		// Scroll to keep cursor visible
		if l.Cursor >= l.Height/2 {
			start = l.Cursor - l.Height/2
		}
		if start+l.Height > len(l.Items) {
			start = len(l.Items) - l.Height
		}
		end = start + l.Height
	}

	for i := start; i < end && i < len(l.Items); i++ {
		item := l.Items[i]
		line := l.renderItem(item, i == l.Cursor)
		lines = append(lines, line)
	}

	// Add scroll indicators
	if start > 0 {
		lines = append([]string{lipgloss.NewStyle().Foreground(mutedColor).Render("  ↑ more")}, lines...)
	}
	if end < len(l.Items) {
		lines = append(lines, lipgloss.NewStyle().Foreground(mutedColor).Render("  ↓ more"))
	}

	return strings.Join(lines, "\n")
}

// renderItem renders a single list item
func (l *List) renderItem(item ListItem, isCursor bool) string {
	var style lipgloss.Style
	var prefix string

	if item.Disabled {
		style = lipgloss.NewStyle().Foreground(mutedColor)
		prefix = "  "
	} else if isCursor {
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primaryColor).
			Bold(true)
		prefix = "▸ "
	} else if item.Selected {
		style = lipgloss.NewStyle().Foreground(secondaryColor)
		prefix = "✓ "
	} else {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
		prefix = "  "
	}

	icon := ""
	if item.Icon != "" {
		icon = item.Icon + " "
	}

	title := prefix + icon + item.Title

	if l.ShowDesc && item.Description != "" && isCursor {
		desc := lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("    " + item.Description)
		return style.Width(l.Width).Render(title) + "\n" + desc
	}

	return style.Width(l.Width).Render(title)
}

// RadioList renders a radio button list (single selection)
type RadioList struct {
	*List
}

// NewRadioList creates a new radio list
func NewRadioList(items []ListItem) *RadioList {
	return &RadioList{List: NewList(items)}
}

// Select selects the current item (deselects others)
func (r *RadioList) Select() {
	for i := range r.Items {
		r.Items[i].Selected = (i == r.Cursor)
	}
}

// GetSelectedIndex returns the index of the selected item
func (r *RadioList) GetSelectedIndex() int {
	for i, item := range r.Items {
		if item.Selected {
			return i
		}
	}
	return -1
}

// CheckList renders a checkbox list (multiple selection)
type CheckList struct {
	*List
}

// NewCheckList creates a new checkbox list
func NewCheckList(items []ListItem) *CheckList {
	return &CheckList{List: NewList(items)}
}

// SelectAll selects all items
func (c *CheckList) SelectAll() {
	for i := range c.Items {
		if !c.Items[i].Disabled {
			c.Items[i].Selected = true
		}
	}
}

// DeselectAll deselects all items
func (c *CheckList) DeselectAll() {
	for i := range c.Items {
		c.Items[i].Selected = false
	}
}

// DiskList renders a list of disks
func DiskList(disks []DiskDisplay, cursor int, width int) string {
	var lines []string

	for i, disk := range disks {
		line := renderDiskItem(disk, i == cursor, width)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// DiskDisplay represents a disk for display
type DiskDisplay struct {
	Device    string
	Model     string
	Size      string
	Type      string
	Removable bool
}

func renderDiskItem(disk DiskDisplay, isCursor bool, width int) string {
	var style lipgloss.Style
	var prefix string

	if isCursor {
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primaryColor).
			Bold(true)
		prefix = "▸ "
	} else {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
		prefix = "  "
	}

	icon := "💾"
	if disk.Type == "nvme" {
		icon = "⚡"
	} else if disk.Removable {
		icon = "📱"
	}

	title := fmt.Sprintf("%s%s %s - %s (%s)", prefix, icon, disk.Device, disk.Model, disk.Size)

	return style.Width(width).Render(title)
}

// PartitionList renders a list of partitions
func PartitionList(partitions []PartitionDisplay, width int) string {
	var lines []string

	headerStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Bold(true)

	header := headerStyle.Render(fmt.Sprintf("%-12s %-10s %-10s %-8s %s",
		"Device", "Mount", "Size", "Type", "Label"))
	lines = append(lines, header)
	lines = append(lines, Divider(width))

	for _, part := range partitions {
		line := fmt.Sprintf("%-12s %-10s %-10s %-8s %s",
			part.Device, part.MountPoint, part.Size, part.FSType, part.Label)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// PartitionDisplay represents a partition for display
type PartitionDisplay struct {
	Device     string
	MountPoint string
	Size       string
	FSType     string
	Label      string
}
