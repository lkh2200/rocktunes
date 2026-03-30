package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
)

// Custom message types
type ioMsg string

// Application model
type model struct {
	choices    []string
	actions    map[string]func(string) tea.Cmd
	cursor     int
	downloads  map[int]struct{}
	loading    map[int]struct{}
	selected   map[int]struct{}
	spinners   map[string]spinner.Model
	textInputs map[string]textinput.Model
}

func initialModel() model {
	// Choices
	choices := []string{"Sync", "Youtube Download", "Archive.org Download"}

	actions := map[string]func(string) tea.Cmd{
		"Sync": func(_ string) tea.Cmd {
			return syncFiles()
		},
		"Youtube Download": func(input string) tea.Cmd {
			return startDownload("Youtube Download", input)
		},
		"Archive.org Download": func(input string) tea.Cmd {
			return startDownload("Archive.org Download", input)
		},
	}

	// Maps for spinners and text inputs
	spinners := make(map[string]spinner.Model)
	textInputs := make(map[string]textinput.Model)

	// Initialize a spinner for each choice
	for _, choice := range choices {
		s := spinner.New()
		s.Spinner = spinner.Dot
		s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
		spinners[choice] = s
	}

	// Initialize text input for Youtube
	yt_ti := textinput.New()
	yt_ti.Placeholder = "Enter URL here..."
	yt_ti.SetVirtualCursor(false)
	yt_ti.CharLimit = 156
	yt_ti.SetWidth(50)
	textInputs["Youtube Download"] = yt_ti

	// Initialize text input for Archive.org
	archive_ti := textinput.New()
	archive_ti.Placeholder = "Enter URL here..."
	archive_ti.SetVirtualCursor(false)
	archive_ti.CharLimit = 156
	archive_ti.SetWidth(50)
	textInputs["Archive.org Download"] = archive_ti

	// Return the fully initialized model
	return model{
		choices:    choices,
		actions:    actions,
		loading:    make(map[int]struct{}),
		selected:   make(map[int]struct{}),
		spinners:   spinners,
		textInputs: textInputs,
	}
}

// Initial commands
func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink}

	for _, sp := range m.spinners {
		cmds = append(cmds, sp.Tick)
	}

	return tea.Batch(cmds...)
}

// Initiate Download
func startDownload(choice string, input string) tea.Cmd {
	var command string

	switch choice {

	case "Archive.org Download":
		command = "python gdApp.py " + input

	case "Youtube Download":
		command = "./rockbox.sh " + input
	}

	return func() tea.Msg {
		c := exec.Command("sh", "-c", command)
		output, err := c.CombinedOutput()
		if err != nil {
			log.Printf("Command failed: %v", err)
		}
		log.Printf("Output: %v", string(output))
		return ioMsg(choice)
	}
}

func syncFiles() tea.Cmd {
	return func() tea.Msg {
		commands := []string{
			`rsync -av --size-only ~/Music/ /run/media/lachlanhenderson/IPOD/Music`,
			`rsync -av --size-only ~/Podcasts/ /run/media/lachlanhenderson/IPOD/Podcasts`,
			`rsync -av --size-only ~/Audiobooks/ /run/media/lachlanhenderson/IPOD/Audiobooks`,
		}

		for _, cmdStr := range commands {
			log.Printf("Running: %s", cmdStr)

			c := exec.Command("sh", "-c", cmdStr)
			output, err := c.CombinedOutput()

			if err != nil {
				log.Printf("Command failed: %v", err)
			}

			log.Printf("Output: %s", string(output))
		}

		return ioMsg("Sync")
	}
}

// Update application state
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		choice := m.choices[m.cursor]

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			_, isLoading := m.loading[m.cursor]
			if isLoading {
				delete(m.loading, m.cursor)
				delete(m.selected, m.cursor)
			} else {
				m.loading[m.cursor] = struct{}{}
				m.selected[m.cursor] = struct{}{}

				// Focus the text input for this choice
				if ti, ok := m.textInputs[choice]; ok {
					ti.Focus()
					clip, err := clipboard.ReadAll()
					if err != nil {
						log.Printf("Clipboard error: %v", err)
						clip = ""
					}

					ti.SetValue(clip)
					m.textInputs[choice] = ti
					log.Printf("Copied: %s", textinput.Paste())
				}

				input := ""
				if ti, ok := m.textInputs[choice]; ok {
					input = ti.Value()
					log.Printf("Input: %s", ti.Value())
				}

				if fn, ok := m.actions[choice]; ok {
					return m, fn(input)
				}
			}
		}

		// Update spinner & text input for current choice
		cmds := []tea.Cmd{}

		if sp, ok := m.spinners[choice]; ok {
			var cmd tea.Cmd
			m.spinners[choice], cmd = sp.Update(msg)
			cmds = append(cmds, cmd)
		}
		if ti, ok := m.textInputs[choice]; ok {
			var cmd tea.Cmd
			m.textInputs[choice], cmd = ti.Update(msg)
			cmds = append(cmds, cmd)
		}

		return m, tea.Batch(cmds...)

	case ioMsg:
		for i, choice := range m.choices {
			if choice == string(msg) {
				delete(m.loading, i)
				ti := m.textInputs[choice]
				ti.Blur()
				if ti, ok := m.textInputs[choice]; ok {
					ti.Blur()
					ti.SetValue("") // safe to reset
					m.textInputs[choice] = ti
				}
			}
		}
		return m, nil

	default:
		// Tick all spinners
		cmds := []tea.Cmd{}
		for choice, sp := range m.spinners {
			var cmd tea.Cmd
			m.spinners[choice], cmd = sp.Update(msg)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}
}

// Render UI
func (m model) View() tea.View {
	s := "Rocktunes !!!\n\n"

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checked := " "
		if _, ok := m.loading[i]; ok {
			if sp, exists := m.spinners[choice]; exists {
				checked = sp.View()
			}
		}

		field := ""
		if _, ok := m.selected[i]; ok {
			if ti, exists := m.textInputs[choice]; exists {
				field = ti.View()
			}
		}

		s += fmt.Sprintf("%s %2s %s\n%s\n", cursor, checked, choice, field)
	}

	s += "\nPress q to quit.\n"
	return tea.NewView(s)
}

func main() {
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
