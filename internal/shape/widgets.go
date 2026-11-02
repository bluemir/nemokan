package shape

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bluemir/monosketch/internal/bitmap"
	"github.com/bluemir/monosketch/internal/geo"
)

// The kinds of widgets, from Mockdown. Marks are ASCII, so they take one cell in any font.
// See ADR-0014.
//
// A size of zero in fit asks for the default size.

func init() {
	for _, newContent := range []func() WidgetContent{
		func() WidgetContent { return &ButtonContent{Label: "OK"} },
		func() WidgetContent { return &InputContent{} },
		func() WidgetContent { return &CardContent{Title: "Title"} },
		func() WidgetContent { return &TableContent{Columns: []string{"Name", "Email", "Role"}} },
		func() WidgetContent { return &ModalContent{Title: "Dialog"} },
		func() WidgetContent { return &CheckboxContent{Label: "Checkbox"} },
		func() WidgetContent { return &RadioContent{Label: "Option"} },
		func() WidgetContent { return &DropdownContent{Label: "Option"} },
		func() WidgetContent { return &ToggleContent{Label: "Toggle"} },
		func() WidgetContent { return &TabsContent{Tabs: []string{"Tab 1", "Tab 2", "Tab 3"}} },
		func() WidgetContent { return &SearchContent{Placeholder: "Search..."} },
		func() WidgetContent { return &ProgressContent{Value: 40} },
		func() WidgetContent { return &BreadcrumbContent{Items: []string{"Home", "Docs", "Page"}} },
		func() WidgetContent { return &PaginationContent{Current: 1, Total: 5} },
		func() WidgetContent {
			return &NavContent{Logo: "Logo", Links: []string{"Home", "About", "Contact"}, Action: "Sign in"}
		},
		func() WidgetContent { return &ListContent{Items: []string{"Item 1", "Item 2", "Item 3"}} },
		func() WidgetContent { return &PlaceholderContent{Label: "Content"} },
		func() WidgetContent { return &SplitContent{Split: 30} },
		func() WidgetContent { return &ImageContent{} },
	} {
		registerWidgetKind(newContent)
	}
}

func textWidth(text string) int {
	return bitmap.StringWidth(text)
}

// truncateText cuts text to fit in width cells.
func truncateText(text string, width int) string {
	var result strings.Builder
	used := 0
	for _, char := range text {
		charWidth := bitmap.CharWidth(char)
		if used+charWidth > width {
			break
		}
		result.WriteRune(char)
		used += charWidth
	}
	return result.String()
}

// fieldRun is a text the user edits. It takes at least one cell, so an empty text can still
// be double clicked.
func fieldRun(row, column int, text string, width int, name string, index int) WidgetRun {
	return WidgetRun{Row: row, Column: column, Text: truncateText(text, width), Field: &WidgetField{Name: name, Index: index}, Width: max(width, 1), Value: text}
}

func markRun(row, column int, text string) WidgetRun {
	return WidgetRun{Row: row, Column: column, Text: text}
}

// orDefault returns requested, or fallback if requested is zero, and not less than minimum.
func orDefault(requested, fallback, minimum int) int {
	if requested == 0 {
		requested = fallback
	}
	return max(requested, minimum)
}

// setItem returns items with the item at index changed to text.
func setItem(items []string, index int, text string) []string {
	if index < 0 || index >= len(items) {
		return items
	}
	changed := append([]string(nil), items...)
	changed[index] = text
	return changed
}

// boxedLine is a one line widget drawn as [prefix text____], such as an input. The text
// field takes the cells between the prefix and ].
func boxedLine(width int, prefix, text string, padding rune, name string) WidgetLayout {
	fieldColumn := 1 + textWidth(prefix)
	fieldWidth := max(width-fieldColumn-1, 0)
	shown := truncateText(text, fieldWidth)
	return WidgetLayout{Runs: []WidgetRun{
		markRun(0, 0, "["+prefix),
		markRun(0, fieldColumn+textWidth(shown), strings.Repeat(string(padding), max(fieldWidth-textWidth(shown), 0))),
		fieldRun(0, fieldColumn, text, fieldWidth, name, 0),
		markRun(0, width-1, "]"),
	}}
}

// toggleLine is a mark followed by a label, such as [x] Remember me. It takes the size of what
// it shows.
func toggleLine(mark, label string) WidgetLayout {
	return WidgetLayout{Runs: []WidgetRun{
		{Row: 0, Column: 0, Text: mark, Field: &widgetToggleField, Width: textWidth(mark)},
		fieldRun(0, textWidth(mark)+1, label, max(textWidth(label), 1), "label", 0),
	}}
}

func toggleLineSize(mark, label string) geo.Size {
	return geo.Size{Width: textWidth(mark) + 1 + max(textWidth(label), 1), Height: 1}
}

// joinedRuns lays out items in one row with separator between them. Each item is a field.
func joinedRuns(row int, items []string, separator string, name string) ([]WidgetRun, int) {
	var runs []WidgetRun
	column := 0
	for index, item := range items {
		if index > 0 {
			runs = append(runs, markRun(row, column, separator))
			column += textWidth(separator)
		}
		runs = append(runs, fieldRun(row, column, item, max(textWidth(item), 1), name, index))
		column += max(textWidth(item), 1)
	}
	return runs, column
}

// ButtonContent is [ OK ]. Only the width changes, and it is at least as wide as the label.
type ButtonContent struct {
	Label string `json:"label"`
}

func (*ButtonContent) Kind() WidgetKind { return "button" }

func (c *ButtonContent) fit(size geo.Size) geo.Size {
	return geo.Size{Width: max(size.Width, textWidth(c.Label)+4), Height: 1}
}

func (c *ButtonContent) layout(size geo.Size) WidgetLayout {
	inner := size.Width - 2
	label := truncateText(c.Label, max(inner-2, 0))
	column := 1 + max((inner-textWidth(label))/2, 0)
	return WidgetLayout{Runs: []WidgetRun{
		markRun(0, 0, "["),
		fieldRun(0, column, c.Label, max(textWidth(label), 1), "label", 0),
		markRun(0, size.Width-1, "]"),
	}}
}

func (c *ButtonContent) withText(_ WidgetField, text string) WidgetContent {
	return &ButtonContent{Label: text}
}

// InputContent is [placeholder____]. Only the width changes.
type InputContent struct {
	Placeholder string `json:"placeholder"`
}

func (*InputContent) Kind() WidgetKind { return "input" }

func (c *InputContent) fit(size geo.Size) geo.Size {
	return geo.Size{Width: orDefault(size.Width, 20, 3), Height: 1}
}

func (c *InputContent) layout(size geo.Size) WidgetLayout {
	return boxedLine(size.Width, "", c.Placeholder, '_', "placeholder")
}

func (c *InputContent) withText(_ WidgetField, text string) WidgetContent {
	return &InputContent{Placeholder: text}
}

// DropdownContent is [v Option    ]. Only the width changes.
type DropdownContent struct {
	Label string `json:"label"`
}

func (*DropdownContent) Kind() WidgetKind { return "dropdown" }

func (c *DropdownContent) fit(size geo.Size) geo.Size {
	return geo.Size{Width: orDefault(size.Width, max(textWidth(c.Label)+6, 16), 5), Height: 1}
}

func (c *DropdownContent) layout(size geo.Size) WidgetLayout {
	return boxedLine(size.Width, "v ", c.Label, ' ', "label")
}

func (c *DropdownContent) withText(_ WidgetField, text string) WidgetContent {
	return &DropdownContent{Label: text}
}

// SearchContent is [/ Search...    ]. Only the width changes.
type SearchContent struct {
	Placeholder string `json:"placeholder"`
}

func (*SearchContent) Kind() WidgetKind { return "search" }

func (c *SearchContent) fit(size geo.Size) geo.Size {
	return geo.Size{Width: orDefault(size.Width, 20, 5), Height: 1}
}

func (c *SearchContent) layout(size geo.Size) WidgetLayout {
	return boxedLine(size.Width, "/ ", c.Placeholder, ' ', "placeholder")
}

func (c *SearchContent) withText(_ WidgetField, text string) WidgetContent {
	return &SearchContent{Placeholder: text}
}

// ProgressContent is [####------] 40%. Only the width changes.
type ProgressContent struct {
	// Value is from 0 to 100.
	Value int `json:"value"`
}

func (*ProgressContent) Kind() WidgetKind { return "progress" }

func (c *ProgressContent) fit(size geo.Size) geo.Size {
	return geo.Size{Width: orDefault(size.Width, 24, 9), Height: 1}
}

func (c *ProgressContent) layout(size geo.Size) WidgetLayout {
	value := min(max(c.Value, 0), 100)
	label := fmt.Sprintf("%d%%", value)
	barWidth := max(size.Width-3-len(label), 0)
	filled := (value*barWidth + 50) / 100
	return WidgetLayout{Runs: []WidgetRun{
		markRun(0, 0, "["+strings.Repeat("#", filled)+strings.Repeat("-", barWidth-filled)+"]"),
		fieldRun(0, barWidth+3, label, len(label), "value", 0),
	}}
}

// withText takes a number. Other texts keep the value.
func (c *ProgressContent) withText(_ WidgetField, text string) WidgetContent {
	value, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(text), "%"))
	if err != nil {
		return c
	}
	return &ProgressContent{Value: min(max(value, 0), 100)}
}

// CheckboxContent is [x] Label. It takes the size of what it shows.
type CheckboxContent struct {
	Label     string `json:"label"`
	IsChecked bool   `json:"isChecked"`
}

func (*CheckboxContent) Kind() WidgetKind { return "checkbox" }

func (c *CheckboxContent) mark() string {
	if c.IsChecked {
		return "[x]"
	}
	return "[ ]"
}

func (c *CheckboxContent) fit(geo.Size) geo.Size        { return toggleLineSize(c.mark(), c.Label) }
func (c *CheckboxContent) layout(geo.Size) WidgetLayout { return toggleLine(c.mark(), c.Label) }
func (c *CheckboxContent) toggled() WidgetContent {
	return &CheckboxContent{Label: c.Label, IsChecked: !c.IsChecked}
}
func (c *CheckboxContent) withText(_ WidgetField, text string) WidgetContent {
	return &CheckboxContent{Label: text, IsChecked: c.IsChecked}
}

// RadioContent is (o) Label. It takes the size of what it shows.
type RadioContent struct {
	Label      string `json:"label"`
	IsSelected bool   `json:"isSelected"`
}

func (*RadioContent) Kind() WidgetKind { return "radio" }

func (c *RadioContent) mark() string {
	if c.IsSelected {
		return "(o)"
	}
	return "( )"
}

func (c *RadioContent) fit(geo.Size) geo.Size        { return toggleLineSize(c.mark(), c.Label) }
func (c *RadioContent) layout(geo.Size) WidgetLayout { return toggleLine(c.mark(), c.Label) }
func (c *RadioContent) toggled() WidgetContent {
	return &RadioContent{Label: c.Label, IsSelected: !c.IsSelected}
}
func (c *RadioContent) withText(_ WidgetField, text string) WidgetContent {
	return &RadioContent{Label: text, IsSelected: c.IsSelected}
}

// ToggleContent is [==o] Label when on and [o==] Label when off. It takes the size of what it
// shows.
type ToggleContent struct {
	Label string `json:"label"`
	IsOn  bool   `json:"isOn"`
}

func (*ToggleContent) Kind() WidgetKind { return "toggle" }

func (c *ToggleContent) mark() string {
	if c.IsOn {
		return "[==o]"
	}
	return "[o==]"
}

func (c *ToggleContent) fit(geo.Size) geo.Size        { return toggleLineSize(c.mark(), c.Label) }
func (c *ToggleContent) layout(geo.Size) WidgetLayout { return toggleLine(c.mark(), c.Label) }
func (c *ToggleContent) toggled() WidgetContent       { return &ToggleContent{Label: c.Label, IsOn: !c.IsOn} }
func (c *ToggleContent) withText(_ WidgetField, text string) WidgetContent {
	return &ToggleContent{Label: text, IsOn: c.IsOn}
}

// BreadcrumbContent is Home > Docs > Page. It takes the size of what it shows.
type BreadcrumbContent struct {
	Items []string `json:"items"`
}

func (*BreadcrumbContent) Kind() WidgetKind { return "breadcrumb" }

const breadcrumbSeparator = " > "

func (c *BreadcrumbContent) fit(geo.Size) geo.Size {
	_, width := joinedRuns(0, c.Items, breadcrumbSeparator, "items")
	return geo.Size{Width: max(width, 1), Height: 1}
}

func (c *BreadcrumbContent) layout(geo.Size) WidgetLayout {
	runs, _ := joinedRuns(0, c.Items, breadcrumbSeparator, "items")
	return WidgetLayout{Runs: runs}
}

func (c *BreadcrumbContent) withText(field WidgetField, text string) WidgetContent {
	return &BreadcrumbContent{Items: setItem(c.Items, field.Index, text)}
}

// PaginationContent is < 1 [2] 3 4 5 ... >, showing up to 5 pages around the current one. It
// takes the size of what it shows, and has no text to edit.
type PaginationContent struct {
	Current int `json:"current"`
	Total   int `json:"total"`
}

func (*PaginationContent) Kind() WidgetKind { return "pagination" }

func (c *PaginationContent) text() string {
	total := max(c.Total, 1)
	current := min(max(c.Current, 1), total)
	visible := min(total, 5)
	start := max(current-2, 1)
	if start+visible-1 > total {
		start = max(total-visible+1, 1)
	}
	parts := []string{"<"}
	for page := start; page < start+visible; page++ {
		if page == current {
			parts = append(parts, fmt.Sprintf("[%d]", page))
		} else {
			parts = append(parts, strconv.Itoa(page))
		}
	}
	if start+visible-1 < total {
		parts = append(parts, "...")
	}
	parts = append(parts, ">")
	return strings.Join(parts, " ")
}

func (c *PaginationContent) fit(geo.Size) geo.Size {
	return geo.Size{Width: len(c.text()), Height: 1}
}

func (c *PaginationContent) layout(geo.Size) WidgetLayout {
	return WidgetLayout{Runs: []WidgetRun{markRun(0, 0, c.text())}}
}

func (c *PaginationContent) withText(WidgetField, string) WidgetContent { return c }

// TabsContent is the tabs with the active one in [ ], over a line. Only the width changes,
// and it is at least as wide as the tabs.
type TabsContent struct {
	Tabs   []string `json:"tabs"`
	Active int      `json:"active"`
}

func (*TabsContent) Kind() WidgetKind { return "tabs" }

// runs puts each tab in [ ] or in spaces of the same width, so the tabs do not move when
// another one becomes active.
func (c *TabsContent) runs() ([]WidgetRun, int) {
	var runs []WidgetRun
	column := 0
	for index, tab := range c.Tabs {
		if index > 0 {
			column++
		}
		tabWidth := max(textWidth(tab), 1)
		if index == c.Active {
			runs = append(runs, markRun(0, column, "["), markRun(0, column+tabWidth+3, "]"))
		}
		runs = append(runs, fieldRun(0, column+2, tab, tabWidth, "tabs", index))
		column += tabWidth + 4
	}
	return runs, column
}

func (c *TabsContent) fit(size geo.Size) geo.Size {
	_, width := c.runs()
	return geo.Size{Width: max(size.Width, width, 1), Height: 2}
}

func (c *TabsContent) layout(size geo.Size) WidgetLayout {
	runs, _ := c.runs()
	return WidgetLayout{Runs: append(runs, markRun(1, 0, strings.Repeat("─", size.Width)))}
}

func (c *TabsContent) withText(field WidgetField, text string) WidgetContent {
	return &TabsContent{Tabs: setItem(c.Tabs, field.Index, text), Active: c.Active}
}

// NavContent is the logo and the links on the left and the action button on the right, over a
// line. Only the width changes, and it is at least as wide as what it shows.
type NavContent struct {
	Logo   string   `json:"logo"`
	Links  []string `json:"links"`
	Action string   `json:"action"`
}

func (*NavContent) Kind() WidgetKind { return "nav" }

const navGap = "   "

// leftRuns are the logo and the links.
func (c *NavContent) leftRuns() ([]WidgetRun, int) {
	runs := []WidgetRun{fieldRun(0, 0, c.Logo, max(textWidth(c.Logo), 1), "logo", 0)}
	column := max(textWidth(c.Logo), 1)
	if len(c.Links) == 0 {
		return runs, column
	}
	links, linksWidth := joinedRuns(0, c.Links, navGap, "links")
	column += len(navGap)
	for _, link := range links {
		link.Column += column
		runs = append(runs, link)
	}
	return runs, column + linksWidth
}

func (c *NavContent) actionWidth() int {
	return max(textWidth(c.Action), 1) + 4
}

func (c *NavContent) fit(size geo.Size) geo.Size {
	_, leftWidth := c.leftRuns()
	minWidth := leftWidth + len(navGap) + c.actionWidth()
	return geo.Size{Width: max(orDefault(size.Width, minWidth+10, 1), minWidth), Height: 2}
}

func (c *NavContent) layout(size geo.Size) WidgetLayout {
	runs, _ := c.leftRuns()
	actionColumn := size.Width - c.actionWidth()
	runs = append(runs,
		markRun(0, actionColumn, "["),
		fieldRun(0, actionColumn+2, c.Action, max(textWidth(c.Action), 1), "action", 0),
		markRun(0, size.Width-1, "]"),
		markRun(1, 0, strings.Repeat("─", size.Width)),
	)
	return WidgetLayout{Runs: runs}
}

func (c *NavContent) withText(field WidgetField, text string) WidgetContent {
	changed := &NavContent{Logo: c.Logo, Links: c.Links, Action: c.Action}
	switch field.Name {
	case "logo":
		changed.Logo = text
	case "links":
		changed.Links = setItem(c.Links, field.Index, text)
	case "action":
		changed.Action = text
	}
	return changed
}

// ListContent is an item in each row after "- ". Only the width changes, and the height is
// the number of items.
type ListContent struct {
	Items []string `json:"items"`
}

func (*ListContent) Kind() WidgetKind { return "list" }

func (c *ListContent) fit(size geo.Size) geo.Size {
	widest := 0
	for _, item := range c.Items {
		widest = max(widest, textWidth(item))
	}
	return geo.Size{Width: orDefault(size.Width, widest+2, 3), Height: max(len(c.Items), 1)}
}

func (c *ListContent) layout(size geo.Size) WidgetLayout {
	var runs []WidgetRun
	for index, item := range c.Items {
		runs = append(runs, markRun(index, 0, "-"), fieldRun(index, 2, item, size.Width-2, "items", index))
	}
	return WidgetLayout{Runs: runs}
}

func (c *ListContent) withText(field WidgetField, text string) WidgetContent {
	return &ListContent{Items: setItem(c.Items, field.Index, text)}
}

// boxSize is the size of a widget drawn in a box, which the user can change.
func boxSize(size, defaultSize, minSize geo.Size) geo.Size {
	return geo.Size{
		Width:  orDefault(size.Width, defaultSize.Width, minSize.Width),
		Height: orDefault(size.Height, defaultSize.Height, minSize.Height),
	}
}

// CardContent is a box with a title over a divider.
type CardContent struct {
	Title string `json:"title"`
}

func (*CardContent) Kind() WidgetKind { return "card" }

func (c *CardContent) fit(size geo.Size) geo.Size {
	return boxSize(size, geo.Size{Width: 30, Height: 8}, geo.Size{Width: 6, Height: 3})
}

func (c *CardContent) layout(size geo.Size) WidgetLayout {
	layout := WidgetLayout{HasBox: true, Runs: []WidgetRun{fieldRun(1, 2, c.Title, size.Width-4, "title", 0)}}
	if size.Height > 3 {
		layout.DividerRows = []int{2}
	}
	return layout
}

func (c *CardContent) withText(_ WidgetField, text string) WidgetContent {
	return &CardContent{Title: text}
}

// TableContent is a box with the column names over a divider, and a line between columns.
// The columns have the same width.
type TableContent struct {
	Columns []string `json:"columns"`
}

func (*TableContent) Kind() WidgetKind { return "table" }

func (c *TableContent) fit(size geo.Size) geo.Size {
	return boxSize(size, geo.Size{Width: 40, Height: 7}, geo.Size{Width: max(len(c.Columns)*3+1, 4), Height: 3})
}

// columnPositions are the columns of the left border, the lines between columns and the
// right border.
func (c *TableContent) columnPositions(width int) []int {
	count := max(len(c.Columns), 1)
	positions := make([]int, count+1)
	for i := range positions {
		positions[i] = (i*(width-1)*2 + count) / (count * 2)
	}
	return positions
}

func (c *TableContent) layout(size geo.Size) WidgetLayout {
	positions := c.columnPositions(size.Width)
	layout := WidgetLayout{HasBox: true, DividerColumns: positions[1 : len(positions)-1]}
	if size.Height > 3 {
		layout.DividerRows = []int{2}
	}
	for index, column := range c.Columns {
		layout.Runs = append(layout.Runs, fieldRun(1, positions[index]+2, column, positions[index+1]-positions[index]-3, "columns", index))
	}
	return layout
}

func (c *TableContent) withText(field WidgetField, text string) WidgetContent {
	return &TableContent{Columns: setItem(c.Columns, field.Index, text)}
}

// ModalContent is a box with a title and x over a divider, and Cancel and OK buttons at the
// bottom right.
type ModalContent struct {
	Title string `json:"title"`
}

func (*ModalContent) Kind() WidgetKind { return "modal" }

const modalButtons = "[ Cancel ] [ OK ]"

func (c *ModalContent) fit(size geo.Size) geo.Size {
	return boxSize(size, geo.Size{Width: 36, Height: 9}, geo.Size{Width: 8, Height: 4})
}

func (c *ModalContent) layout(size geo.Size) WidgetLayout {
	layout := WidgetLayout{HasBox: true, DividerRows: []int{2}, Runs: []WidgetRun{
		fieldRun(1, 2, c.Title, size.Width-6, "title", 0),
		markRun(1, size.Width-3, "x"),
	}}
	buttonsColumn := size.Width - 2 - len(modalButtons)
	if size.Height-2 > 2 && buttonsColumn > 0 {
		layout.Runs = append(layout.Runs, markRun(size.Height-2, buttonsColumn, modalButtons))
	}
	return layout
}

func (c *ModalContent) withText(_ WidgetField, text string) WidgetContent {
	return &ModalContent{Title: text}
}

// PlaceholderContent is a box with a label in the middle.
type PlaceholderContent struct {
	Label string `json:"label"`
}

func (*PlaceholderContent) Kind() WidgetKind { return "placeholder" }

func (c *PlaceholderContent) fit(size geo.Size) geo.Size {
	return boxSize(size, geo.Size{Width: 24, Height: 6}, geo.Size{Width: 5, Height: 3})
}

func (c *PlaceholderContent) layout(size geo.Size) WidgetLayout {
	inner := size.Width - 2
	label := truncateText(c.Label, inner)
	column := 1 + (inner-textWidth(label))/2
	return WidgetLayout{HasBox: true, Runs: []WidgetRun{
		fieldRun((size.Height-1)/2, column, c.Label, max(textWidth(label), 1), "label", 0),
	}}
}

func (c *PlaceholderContent) withText(_ WidgetField, text string) WidgetContent {
	return &PlaceholderContent{Label: text}
}

// SplitContent is a box split into left and right by a line. It has no text to edit.
type SplitContent struct {
	// Split is where the line is, in percent of the width from the left.
	Split int `json:"split"`
}

func (*SplitContent) Kind() WidgetKind { return "split" }

func (c *SplitContent) fit(size geo.Size) geo.Size {
	return boxSize(size, geo.Size{Width: 48, Height: 12}, geo.Size{Width: 5, Height: 3})
}

func (c *SplitContent) layout(size geo.Size) WidgetLayout {
	column := ((size.Width-1)*min(max(c.Split, 0), 100) + 50) / 100
	return WidgetLayout{HasBox: true, DividerColumns: []int{min(max(column, 1), size.Width-2)}}
}

func (c *SplitContent) withText(WidgetField, string) WidgetContent { return c }

// ImageContent is a box crossed by two diagonals. It has no text to edit.
type ImageContent struct{}

func (*ImageContent) Kind() WidgetKind { return "image" }

func (c *ImageContent) fit(size geo.Size) geo.Size {
	return boxSize(size, geo.Size{Width: 20, Height: 8}, geo.Size{Width: 5, Height: 3})
}

func (c *ImageContent) layout(size geo.Size) WidgetLayout {
	layout := WidgetLayout{HasBox: true}
	innerWidth := size.Width - 2
	innerHeight := size.Height - 2
	for row := range innerHeight {
		offset := 0
		if innerHeight > 1 {
			offset = (row*(innerWidth-1)*2 + innerHeight - 1) / ((innerHeight - 1) * 2)
		}
		left := 1 + offset
		right := size.Width - 2 - offset
		if left == right {
			layout.Runs = append(layout.Runs, markRun(row+1, left, "X"))
			continue
		}
		layout.Runs = append(layout.Runs, markRun(row+1, left, `\`), markRun(row+1, right, "/"))
	}
	return layout
}

func (c *ImageContent) withText(WidgetField, string) WidgetContent { return c }
