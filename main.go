package main

import (
	"fmt"
	"log"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type screen int

const (
	screenZip screen = iota
	screenStores
	screenMenu
	screenReceipt
)

var (
	receiptBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(tbPurple).
				Padding(1, 3).Width(50)

	receiptTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(tbLightPurple).
				MarginBottom(1)

	receiptItemStyle = lipgloss.NewStyle().
				Foreground(tbOffWhite)

	receiptTotalStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(tbGreen)

	receiptDimStyle = lipgloss.NewStyle().
			Foreground(tbMuted)

	// Zip input window styles
	zipWindowStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tbLightPurple).
			Padding(1, 3).
			Width(44)

	zipTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(tbLightPurple).
			MarginBottom(1)

	zipDescStyle = lipgloss.NewStyle().
			Foreground(tbMuted).
			MarginBottom(1)

	zipInputLineStyle = lipgloss.NewStyle().
				Foreground(tbOffWhite)

	zipErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			MarginTop(1)
)

type zipInputModel struct {
	value string
	err   string
	done  bool
}

func (z zipInputModel) validate() error {
	s := strings.TrimSpace(z.value)
	if len(s) != 5 {
		return fmt.Errorf("zip must be 5 digits")
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return fmt.Errorf("zip must be numeric")
		}
	}
	return nil
}

func (z zipInputModel) Update(msg tea.Msg) (zipInputModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return z, tea.Quit
		case "backspace":
			if len(z.value) > 0 {
				z.value = z.value[:len(z.value)-1]
				z.err = ""
			}
		case "enter":
			if err := z.validate(); err != nil {
				z.err = err.Error()
			} else {
				z.err = ""
				z.done = true
			}
		default:
			ch := key.String()
			if len(ch) == 1 && len(z.value) < 5 {
				z.value += ch
				z.err = ""
			}
		}
	}
	return z, nil
}

func (z zipInputModel) render(width, height int) string {
	cursor := "█"
	inputLine := zipInputLineStyle.Render(z.value + cursor)

	var lines []string
	lines = append(lines, zipTitleStyle.Render("🌮 Welcome to Taco TUI! 🌮"))
	lines = append(lines, zipDescStyle.Render("Enter your zip code to find nearby Taco Bell locations"))
	lines = append(lines, inputLine)
	if z.err != "" {
		lines = append(lines, zipErrorStyle.Render("✗ "+z.err))
	}

	window := zipWindowStyle.Render(strings.Join(lines, "\n"))

	outer := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center).
		AlignVertical(lipgloss.Center)

	return outer.Render(window)
}

type foundZipMsg struct {
	zip  string
	lat  float64
	long float64
}

type appModel struct {
	screen          screen
	width, height   int
	zipModel        zipInputModel
	zip             string
	storeModel      storeListModel
	storeModelReady bool
	menuModel       *menuModel
	menuModelReady  bool
	finalCart       *Cart
}

func newAppModel() *appModel {
	return &appModel{
		screen:   screenZip,
		zipModel: zipInputModel{},
	}
}

func (m *appModel) Init() tea.Cmd {
	return nil
}

func (m *appModel) setSize(width, height int) {
	m.width = width
	m.height = height

	if m.storeModelReady {
		m.storeModel.list.SetSize(width, height)
		m.storeModel.width = width
		m.storeModel.height = height
	}

	if m.menuModelReady {
		m.menuModel.width = width
		m.menuModel.height = height
	}
}

func (m *appModel) startStoreSearch(zip string) tea.Cmd {
	m.screen = screenStores
	m.storeModel = newStoreListModel(zip)
	m.storeModelReady = true
	m.storeModel.list.SetSize(m.width, m.height)
	m.storeModel.width = m.width
	m.storeModel.height = m.height

	return tea.Batch(
		m.storeModel.Init(),
		func() tea.Msg {
			lat, long := lookupZip(zip)
			return foundZipMsg{zip: zip, lat: lat, long: long}
		},
	)
}

func (m *appModel) openMenu(storeNumber string) tea.Cmd {
	m.screen = screenMenu
	m.menuModel = newMenuModel(storeNumber)
	m.menuModelReady = true
	m.menuModel.width = m.width
	m.menuModel.height = m.height
	return m.menuModel.Init()
}

func (m *appModel) updateActiveScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenZip:
		updated, cmd := m.zipModel.Update(msg)
		m.zipModel = updated

		if m.zipModel.done {
			zip := strings.TrimSpace(m.zipModel.value)
			m.zipModel.done = false
			m.zip = zip
			return m, m.startStoreSearch(zip)
		}
		return m, cmd

	case screenStores:
		updated, cmd := m.storeModel.Update(msg)
		m.storeModel = updated.(storeListModel)
		return m, cmd

	case screenMenu:
		updated, cmd := m.menuModel.Update(msg)
		m.menuModel = updated.(*menuModel)
		return m, cmd

	case screenReceipt:
		return m, nil
	}

	return m, nil
}

func (m *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		return m.updateActiveScreen(msg)

	case foundZipMsg:
		return m, fetchStoresCmd(msg.lat, msg.long)

	case storeSelectedMsg:
		return m, m.openMenu(msg.store.StoreNumber)

	case orderPlacedMsg:
		m.screen = screenReceipt
		m.finalCart = msg.cart
		return m, nil

	case tea.KeyMsg:
		if m.screen == screenReceipt {
			switch msg.String() {
			case "q", "ctrl+c", "enter":
				return m, tea.Quit
			case "n":
				m.screen = screenZip
				m.zipModel = zipInputModel{}
				m.zip = ""
				m.storeModelReady = false
				m.menuModelReady = false
				m.menuModel = nil
				m.finalCart = nil
				return m, nil
			}
			return m, nil
		}

		if m.screen == screenMenu && m.menuModelReady {
			switch msg.String() {
			case "esc":
				m.screen = screenStores
				m.menuModel = nil
				m.menuModelReady = false
				return m, nil
			}
		}
	}

	return m.updateActiveScreen(msg)
}

func (m *appModel) View() tea.View {
	var v tea.View
	switch m.screen {
	case screenZip:
		v = tea.NewView(m.zipModel.render(m.width, m.height))
	case screenStores:
		v = m.storeModel.View()
	case screenMenu:
		v = m.menuModel.View()
	case screenReceipt:
		return m.receiptView()
	default:
		return tea.NewView("")
	}
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m *appModel) receiptView() tea.View {
	if m.finalCart == nil {
		return tea.NewView("")
	}

	const (
		receiptNameWidth  = 28
		receiptQtyWidth   = 4
		receiptPriceWidth = 10
		receiptLineWidth  = receiptNameWidth + receiptQtyWidth + receiptPriceWidth
	)

	var lines []string
	lines = append(lines, receiptTitleStyle.Render("🌮 Your Order"))
	lines = append(lines, "")

	for _, entry := range m.finalCart.Items() {
		nameLines := wrapText(entry.Item.Name, receiptNameWidth)
		qty := fmt.Sprintf("x%d", entry.Quantity)
		price := fmt.Sprintf("$%.2f", entry.Item.Price*float64(entry.Quantity))

		qtyCell := lipgloss.NewStyle().
			Width(receiptQtyWidth).
			Align(lipgloss.Right).
			Render(receiptDimStyle.Render(qty))

		priceCell := lipgloss.NewStyle().
			Width(receiptPriceWidth).
			Align(lipgloss.Right).
			Render(price)

		for i, nameLine := range nameLines {
			if i == 0 {
				line := fmt.Sprintf("%-*s %s %s",
					receiptNameWidth,
					nameLine,
					qtyCell,
					priceCell,
				)
				lines = append(lines, receiptItemStyle.Render(line))
				continue
			}

			lines = append(lines, receiptItemStyle.Render(nameLine))
		}
	}

	lines = append(lines, strings.Repeat("─", receiptLineWidth))
	lines = append(lines, receiptTotalStyle.Render(
		fmt.Sprintf("%-*s %s",
			receiptNameWidth+receiptQtyWidth+1,
			"Total",
			m.finalCart.FormattedTotal(),
		),
	))
	lines = append(lines, "")
	lines = append(lines, receiptDimStyle.Render("press enter or q to exit"))
	lines = append(lines, receiptDimStyle.Render("press n for a new order"))

	content := strings.Join(lines, "\n")

	outer := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center).
		AlignVertical(lipgloss.Center)

	v := tea.NewView(outer.Render(receiptBorderStyle.Render(content)))
	v.AltScreen = true
	return v
}

func main() {
	p := tea.NewProgram(newAppModel())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
