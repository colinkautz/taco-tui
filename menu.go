package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type orderPlacedMsg struct {
	cart *Cart
}

type fetchMenuMsg struct {
	categories []MenuCategory
}

var tabBorder = lipgloss.Border{
	Top:         "─",
	Bottom:      " ",
	Left:        "│",
	Right:       "│",
	TopLeft:     "╭",
	TopRight:    "╮",
	BottomLeft:  "┘",
	BottomRight: "└",
}

var (
	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(tbLightPurple).
			Border(tabBorder).
			BorderForeground(tbLightPurple).
			Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(tbLightPurple).
				Border(tabBorder).
				BorderForeground(tbDim).
				Padding(0, 1)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(tbLightPurple).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(tbOffWhite)

	dimItemStyle = lipgloss.NewStyle().
			Foreground(tbMuted)

	itemPriceStyle = lipgloss.NewStyle().
			Foreground(tbGreen)

	cartPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tbDim).
			Padding(0, 1)

	cartTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(tbLightPurple).
			MarginBottom(1)

	cartItemStyle = lipgloss.NewStyle().
			Foreground(tbOffWhite)

	cartTotalStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(tbGreen).
			MarginTop(1)

	cartEmptyStyle = lipgloss.NewStyle().
			Foreground(tbDim).
			Italic(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(tbMuted).
			MarginTop(1)

	searchBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tbLightPurple).
			Padding(0, 1)

	searchLabelStyle = lipgloss.NewStyle().
				Foreground(tbMuted)

	noResultsStyle = lipgloss.NewStyle().
			Foreground(tbMuted).
			MarginTop(1)
)

const (
	cartWidth      = 32
	itemPriceWidth = 6
	cartNameWidth  = 24
	cartQtyWidth   = 3
)

type menuState int

const (
	menuLoading menuState = iota
	menuReady
	menuError
)

type menuModel struct {
	categories  []MenuCategory
	activeTab   int
	tabOffset   int
	cursor      int
	cart        *Cart
	spinner     spinner.Model
	state       menuState
	storeNumber string
	width       int
	height      int
	filtering   bool
	filterQuery string
	topHeight   int
}

func newMenuModel(storeNumber string) *menuModel {
	s := spinner.New()
	s.Spinner = spinner.Moon
	s.Style = lipgloss.NewStyle().Foreground(tbPurple)

	return &menuModel{
		spinner:     s,
		state:       menuLoading,
		storeNumber: storeNumber,
		cart:        NewCart(),
	}
}

func (m *menuModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		fetchMenuCmd(m.storeNumber),
	)
}

func fetchMenuCmd(storeNumber string) tea.Cmd {
	return func() tea.Msg {
		return fetchMenuMsg{categories: fetchMenu(storeNumber)}
	}
}

func (m *menuModel) filterItems() []MenuItem {
	if m.filterQuery == "" {
		return nil
	}

	query := strings.ToLower(m.filterQuery)
	seen := make(map[string]struct{})
	var results []MenuItem

	for _, category := range m.categories {
		for _, item := range category.Items {
			if !strings.Contains(strings.ToLower(item.Name), query) {
				continue
			}
			if _, ok := seen[item.ProductID]; ok {
				continue
			}
			seen[item.ProductID] = struct{}{}
			results = append(results, item)
		}
	}

	return results
}

func (m *menuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case fetchMenuMsg:
		if len(msg.categories) == 0 {
			m.state = menuError
			return m, nil
		}
		m.categories = msg.categories
		m.state = menuReady
		return m, nil

	case spinner.TickMsg:
		if m.state == menuLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case tea.MouseClickMsg:
		if m.state != menuReady {
			return m, nil
		}

		menuWidth := m.width - cartWidth - 2
		if msg.X >= menuWidth {
			return m, nil
		}

		itemY := msg.Y - m.topHeight
		if itemY < 0 {
			return m, nil
		}

		var items []MenuItem
		if m.filtering {
			items = m.filterItems()
		} else {
			items = m.currentItems()
		}

		helpHeight := lipgloss.Height(m.renderHelp())
		contentHeight := m.height - m.topHeight - helpHeight - 1
		start, _ := m.visibleRange(len(items), contentHeight)
		clickedIndex := start + itemY
		if clickedIndex < 0 || clickedIndex >= len(items) {
			return m, nil
		}

		m.cursor = clickedIndex
		if msg.Button == tea.MouseLeft {
			m.cart.AddItem(items[clickedIndex])
		} else if msg.Button == tea.MouseRight {
			m.cart.RemoveItem(items[clickedIndex].ProductID)
		}
		return m, nil

	case tea.MouseWheelMsg:
		if m.state != menuReady {
			return m, nil
		}

		var items []MenuItem
		if m.filtering {
			items = m.filterItems()
		} else {
			items = m.currentItems()
		}

		if msg.Button == tea.MouseWheelDown {
			if m.cursor < len(items)-1 {
				m.cursor++
			}
		} else if msg.Button == tea.MouseWheelUp {
			if m.cursor > 0 {
				m.cursor--
			}
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *menuModel) currentItems() []MenuItem {
	if len(m.categories) == 0 {
		return nil
	}
	return m.categories[m.activeTab].Items
}

func (m *menuModel) View() tea.View {
	if m.width == 0 {
		return tea.NewView("")
	}

	switch m.state {
	case menuLoading:
		return tea.NewView(fmt.Sprintf("\n\n  %s loading menu...\n\n", m.spinner.View()))
	case menuError:
		return tea.NewView("\n\n  ❌ failed to load menu\n\n  press q to quit\n")
	}

	menuWidth := m.width - cartWidth - 2
	help := m.renderHelp()
	helpHeight := lipgloss.Height(help)

	var top, items string
	if m.filtering {
		top = m.renderSearchBox(menuWidth)
		m.topHeight = lipgloss.Height(top)
		contentHeight := m.height - m.topHeight - helpHeight - 1
		items = m.renderFilterResults(menuWidth, contentHeight)
	} else {
		top = m.renderTabs(menuWidth)
		m.topHeight = lipgloss.Height(top)
		contentHeight := m.height - m.topHeight - helpHeight - 1
		items = m.renderItems(menuWidth, contentHeight)
	}

	cart := m.renderCart(m.height - helpHeight - 1)

	left := lipgloss.JoinVertical(lipgloss.Left, top, items)
	leftHeight := lipgloss.Height(left)
	remaining := m.height - leftHeight - helpHeight
	if remaining > 0 {
		left = left + strings.Repeat("\n", remaining)
	}

	left = lipgloss.NewStyle().
		Width(menuWidth).
		Render(left)

	content := lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", cart)
	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, content, help))
}

func (m *menuModel) renderSearchBox(width int) string {
	cursor := "█"
	query := m.filterQuery + cursor
	inner := searchLabelStyle.Render("search: ") + lipgloss.NewStyle().Foreground(tbOffWhite).Render(query)
	return searchBoxStyle.Width(width - 2).Render(inner)
}

func (m *menuModel) visibleRange(total, height int) (int, int) {
	if total <= 0 {
		return 0, 0
	}

	maxVisible := height
	if maxVisible < 1 {
		maxVisible = 1
	}
	if maxVisible > total {
		maxVisible = total
	}

	start := 0
	if m.cursor >= maxVisible {
		start = m.cursor - maxVisible + 1
	}

	end := start + maxVisible
	if end > total {
		end = total
	}

	return start, end
}

func (m *menuModel) renderItemLine(name string, price float64, active bool, width int) string {
	if width <= 0 {
		return ""
	}

	selectorWidth := 2
	gapWidth := 2
	priceWidth := itemPriceWidth

	nameWidth := width - selectorWidth - gapWidth - priceWidth
	if nameWidth < 8 {
		nameWidth = 8
	}

	selector := "  "
	if active {
		selector = "> "
	}

	nameCell := fmt.Sprintf("%-*s", nameWidth, truncate(name, nameWidth))
	priceCell := fmt.Sprintf("%*s", priceWidth, fmt.Sprintf("$%.2f", price))

	row := selector + nameCell + strings.Repeat(" ", gapWidth) + priceCell

	if active {
		row = selectedItemStyle.Render(row)
	} else {
		row = normalItemStyle.Render(row)
	}

	return lipgloss.NewStyle().Width(width).Render(row)
}

func (m *menuModel) renderRows(items []MenuItem, width, height int, emptyMessage string) string {
	if len(items) == 0 {
		return dimItemStyle.Render(emptyMessage)
	}

	start, end := m.visibleRange(len(items), height)
	var rows []string

	for i := start; i < end; i++ {
		rows = append(rows, m.renderItemLine(items[i].Name, items[i].Price, i == m.cursor, width))
	}

	if len(items) > end-start {
		rows = append(rows, dimItemStyle.Render(fmt.Sprintf("  %d/%d", m.cursor+1, len(items))))
	}

	return strings.Join(rows, "\n")
}

func (m *menuModel) renderFilterResults(width, height int) string {
	results := m.filterItems()

	if m.filterQuery == "" {
		return noResultsStyle.Render("  start typing to search...")
	}
	if len(results) == 0 {
		return noResultsStyle.Render("  no results for \"" + m.filterQuery + "\"")
	}

	return m.renderRows(results, width, height, "")
}

func renderTabLabel(name string, active bool) string {
	caser := cases.Title(language.English)
	formattedName := caser.String(name)
	if active {
		return activeTabStyle.Render(formattedName)
	}
	return inactiveTabStyle.Render(formattedName)
}

func (m *menuModel) tabLayout(width int) ([]string, int) {
	prevIndicator := dimItemStyle.Render("‹ ")
	nextIndicator := dimItemStyle.Render(" ›")
	prevWidth := lipgloss.Width(prevIndicator)
	nextWidth := lipgloss.Width(nextIndicator)

	adjustedOffset := m.tabOffset
	for {
		used := 0
		if adjustedOffset > 0 {
			used += prevWidth
		}

		lastVisible := adjustedOffset - 1
		for i := adjustedOffset; i < len(m.categories); i++ {
			tab := renderTabLabel(m.categories[i].Name, i == m.activeTab)
			tabWidth := lipgloss.Width(tab)
			trailingWidth := 0
			if i < len(m.categories)-1 {
				trailingWidth = nextWidth
			}

			if (used+tabWidth+trailingWidth > width) && i > adjustedOffset {
				break
			}

			used += tabWidth
			lastVisible = i
		}

		if m.activeTab <= lastVisible {
			break
		}

		adjustedOffset++
	}

	var tabs []string
	if adjustedOffset > 0 {
		tabs = append(tabs, prevIndicator)
	}

	used := 0
	if adjustedOffset > 0 {
		used = prevWidth
	}

	lastVisible := adjustedOffset
	for i := adjustedOffset; i < len(m.categories); i++ {
		tab := renderTabLabel(m.categories[i].Name, i == m.activeTab)
		tabWidth := lipgloss.Width(tab)
		trailingWidth := 0
		if i < len(m.categories)-1 {
			trailingWidth = nextWidth
		}

		if used+tabWidth+trailingWidth > width && i > adjustedOffset {
			break
		}

		used += tabWidth
		lastVisible = i
		tabs = append(tabs, tab)
	}

	if lastVisible < len(m.categories)-1 {
		tabs = append(tabs, nextIndicator)
	}

	return tabs, adjustedOffset
}

func (m *menuModel) renderTabs(width int) string {
	if len(m.categories) == 0 {
		return ""
	}

	tabs, offset := m.tabLayout(width)
	m.tabOffset = offset
	return lipgloss.JoinHorizontal(lipgloss.Center, tabs...)
}

func (m *menuModel) renderItems(width, height int) string {
	return m.renderRows(m.currentItems(), width, height, "  no items in this category")
}

func wrapText(s string, width int) []string {
	if width <= 0 {
		return []string{""}
	}

	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	line := words[0]

	for _, word := range words[1:] {
		next := line + " " + word
		if lipgloss.Width(next) <= width {
			line = next
			continue
		}
		lines = append(lines, line)
		line = word
	}

	lines = append(lines, line)
	return lines
}

func (m *menuModel) renderCartItem(entry *CartItem) string {
	nameLines := wrapText(entry.Item.Name, cartNameWidth)
	qty := fmt.Sprintf("x%d", entry.Quantity)
	price := fmt.Sprintf("$%.2f", entry.Item.Price*float64(entry.Quantity))

	qtyCell := lipgloss.NewStyle().
		Width(cartQtyWidth).
		Align(lipgloss.Right).
		Render(dimItemStyle.Render(qty))

	var lines []string
	for i, nameLine := range nameLines {
		if i == 0 {
			lines = append(lines, fmt.Sprintf("%-*s %s",
				cartNameWidth,
				nameLine,
				qtyCell,
			))
			continue
		}
		lines = append(lines, nameLine)
	}

	lines = append(lines, itemPriceStyle.Render(price))
	return strings.Join(lines, "\n")
}

func (m *menuModel) renderCart(height int) string {
	var lines []string

	lines = append(lines, cartTitleStyle.Render("🛒 Cart"))

	if m.cart.IsEmpty() {
		lines = append(lines, cartEmptyStyle.Render("empty"))
	} else {
		for _, entry := range m.cart.Items() {
			lines = append(lines, cartItemStyle.Render(m.renderCartItem(entry)))
		}
		lines = append(lines, strings.Repeat("─", cartWidth-4))
		lines = append(lines, cartTotalStyle.Render("Total: "+m.cart.FormattedTotal()))
	}

	content := strings.Join(lines, "\n")
	return cartPanelStyle.Width(cartWidth).Height(height).Render(content)
}

func (m *menuModel) renderHelp() string {
	if m.filtering {
		return helpStyle.Render(strings.Join([]string{
			"↑/↓ navigate",
			"←/→ qty",
			"esc exit search",
		}, "  ·  "))
	}
	return helpStyle.Render(strings.Join([]string{
		",/. category",
		"↑/↓ item",
		"←/→ qty",
		"/ search",
		"c checkout",
		"esc back",
		"q quit",
	}, "  ·  "))
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}
