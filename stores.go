package main

import (
	"fmt"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type storeSelectedMsg struct {
	store Store
}

type storeErrorMsg struct {
	err error
}

var (
	storeListTileStyle = lipgloss.NewStyle().
				MarginLeft(2).Foreground(tbLightPurple).Bold(true)

	storeStatusStyle = lipgloss.NewStyle().
				MarginLeft(2).MarginBottom(1).Foreground(tbMuted)

	storeHelpStyle = lipgloss.NewStyle().
			MarginLeft(2).Foreground(tbMuted)
)

type storeListState int

const (
	storeListLoading storeListState = iota
	storeListReady
	storeListError
)

type storeListModel struct {
	list    list.Model
	spinner spinner.Model
	state   storeListState
	zip     string
	err     string
	width   int
	height  int
}

type fetchStoresMsg struct {
	stores []Store
}

func newStoreListModel(zip string) *storeListModel {
	s := spinner.New()
	s.Spinner = spinner.Moon
	s.Style = lipgloss.NewStyle().Foreground(tbPurple)

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(tbLightPurple).
		BorderLeftForeground(tbLightPurple)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(tbLightPurple).
		BorderLeftForeground(tbLightPurple)
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.
		Foreground(tbOffWhite)
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.
		Foreground(tbOffWhite)

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = fmt.Sprintf("🌮 Taco Bell Stores near %s", zip)
	l.Styles.Title = storeListTileStyle
	l.Styles.StatusBar = storeStatusStyle
	l.Styles.HelpStyle = storeHelpStyle
	l.Help.Styles.ShortKey = lipgloss.NewStyle().Foreground(tbMuted)
	l.Help.Styles.ShortDesc = lipgloss.NewStyle().Foreground(tbMuted)
	l.Help.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(tbDim)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)

	return &storeListModel{
		list:    l,
		spinner: s,
		state:   storeListLoading,
		zip:     zip,
	}
}

func (m *storeListModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func fetchStoresCmd(lat, long float64) tea.Cmd {
	return func() tea.Msg {
		stores, fetchErr := fetchStores(lat, long)
		if fetchErr != nil {
			return storeErrorMsg{err: fetchErr}
		}
		return fetchStoresMsg{stores: stores}
	}
}

func (m *storeListModel) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *storeListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height)
		return m, nil

	case fetchStoresMsg:
		items := make([]list.Item, len(msg.stores))
		for i, s := range msg.stores {
			items[i] = s
		}

		m.list.SetItems(items)
		m.list.Title = fmt.Sprintf("🌮 %d Taco Bell store(s) near %s 🌮", len(items), m.zip)
		m.state = storeListReady
		return m, nil

	case storeErrorMsg:
		m.state = storeListError
		fetchErr, ok := msg.err.(*FetchError)
		if ok && fetchErr.Kind == FetchErrEmpty {
			m.err = fmt.Sprintf("no Taco Bell locations found near %s", m.zip)
		} else {
			m.err = msg.err.Error()
		}
		return m, nil

	case spinner.TickMsg:
		if m.state == storeListLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.MouseClickMsg:
		if m.state != storeListReady || m.list.FilterState() == list.Filtering {
			return m, nil
		}

		if msg.Button == tea.MouseLeft {
			const headerLines = 2
			const itemHeight = 3

			clickedIndex := (msg.Y - headerLines) / itemHeight
			itemsOnPage := m.list.Paginator.ItemsOnPage(len(m.list.VisibleItems()))
			if clickedIndex < 0 || clickedIndex >= itemsOnPage {
				return m, nil
			}

			delta := clickedIndex - m.list.Cursor()
			for i := 0; i < delta; i++ {
				m.list.CursorDown()
			}
			for i := 0; i > delta; i-- {
				m.list.CursorUp()
			}
			if delta == 0 {
				if selected, ok := m.list.SelectedItem().(Store); ok {
					return m, func() tea.Msg { return storeSelectedMsg{store: selected} }
				}
			}
		}
		return m, nil

	case tea.MouseWheelMsg:
		if m.state != storeListReady || m.list.FilterState() == list.Filtering {
			return m, nil
		}

		if msg.Button == tea.MouseWheelDown {
			m.list.CursorDown()
		} else if msg.Button == tea.MouseWheelUp {
			m.list.CursorUp()
		}
		return m, nil

	case tea.KeyMsg:
		if m.state != storeListReady {
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}

		if m.list.FilterState() == list.Filtering {
			return m.updateList(msg)
		}

		switch msg.String() {
		case "enter":
			if selected, ok := m.list.SelectedItem().(Store); ok {
				return m, func() tea.Msg {
					return storeSelectedMsg{store: selected}
				}
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	if m.state == storeListReady {
		return m.updateList(msg)
	}

	return m, nil
}

func (m *storeListModel) View() tea.View {
	switch m.state {
	case storeListLoading:
		return tea.NewView(fmt.Sprintf("\n\n  %s finding stores near %s...\n\n", m.spinner.View(), m.zip))

	case storeListError:
		return tea.NewView(fmt.Sprintf("\n\n  ❌ %s\n\n%s",
			m.err,
			storeStatusStyle.Render("press q to quit"),
		))

	default:
		return tea.NewView(m.list.View())
	}
}
