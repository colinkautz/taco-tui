package main

import tea "charm.land/bubbletea/v2"

func (m *menuModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filtering {
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m.handleFilteringKey(msg)
	}

	if isQuitKey(msg) {
		return m, tea.Quit
	}

	if m.state != menuReady {
		return m, nil
	}

	return m.handleNormalKey(msg)
}

func isQuitKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "q", "ctrl+c":
		return true
	default:
		return false
	}
}

func (m *menuModel) handleFilteringKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filterQuery = ""
		m.cursor = 0

	case "backspace":
		if len(m.filterQuery) > 0 {
			m.filterQuery = m.filterQuery[:len(m.filterQuery)-1]
			m.cursor = 0
		}

	case "up":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down":
		results := m.filterItems()
		if m.cursor < len(results)-1 {
			m.cursor++
		}

	case "right":
		results := m.filterItems()
		if len(results) > 0 && m.cursor < len(results) {
			m.cart.AddItem(results[m.cursor])
		}

	case "left":
		results := m.filterItems()
		if len(results) > 0 && m.cursor < len(results) {
			m.cart.RemoveItem(results[m.cursor].ProductID)
		}

	case " ", "space":
		m.filterQuery += " "
		m.cursor = 0

	default:
		if len(msg.String()) == 1 {
			m.filterQuery += msg.String()
			m.cursor = 0
		}
	}

	return m, nil
}

func (m *menuModel) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "/":
		m.filtering = true
		m.filterQuery = ""
		m.cursor = 0

	case ",":
		if m.activeTab > 0 {
			m.activeTab--
			m.cursor = 0
			if m.activeTab < m.tabOffset {
				m.tabOffset = m.activeTab
			}
		}

	case ".":
		if m.activeTab < len(m.categories)-1 {
			m.activeTab++
			m.cursor = 0
		}

	case "up":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down":
		items := m.currentItems()
		if m.cursor < len(items)-1 {
			m.cursor++
		}

	case "right":
		items := m.currentItems()
		if len(items) > 0 {
			m.cart.AddItem(items[m.cursor])
		}

	case "left":
		items := m.currentItems()
		if len(items) > 0 {
			m.cart.RemoveItem(items[m.cursor].ProductID)
		}

	case "c":
		if !m.cart.IsEmpty() {
			return m, func() tea.Msg {
				return orderPlacedMsg{cart: m.cart}
			}
		}
	}

	return m, nil
}
