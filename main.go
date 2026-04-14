package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
	"github.com/joho/godotenv"
)

// Custom message types
type ioMsg string

// Application model
type model struct {
	choices    []string
	actions    map[string]func(string) tea.Cmd
	cursor     int
	charger    chargerModel
	downloads  map[int]struct{}
	loading    map[int]struct{}
	selected   map[int]struct{}
	spinners   map[string]spinner.Model
	textInputs map[string]textinput.Model
}

// Charger model
type chargerModel struct {
	cable  string
	head   string
	colour string
}

func initialModel() model {
	var colour, head string

	charger := chargerModel{
		cable: `                %s
                |
                |____
                     |
                     |`,
		head:   head,
		colour: colour,
	}

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
		charger:    charger,
		actions:    actions,
		loading:    make(map[int]struct{}),
		selected:   make(map[int]struct{}),
		spinners:   spinners,
		textInputs: textInputs,
	}
}

// Render the charger element of the UI
func renderCharger(charger chargerModel) string {
	s := fmt.Sprintf(charger.cable, charger.head)

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(charger.colour))

	return style.Render(s)
}

// Initial commands
func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink}

	for _, sp := range m.spinners {
		cmds = append(cmds, sp.Tick)
	}
	cmds = append(cmds, connectionMonitor())

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

func connectionMonitor() tea.Cmd {
	connection := isConnected()
	log.Print("Monitoring connection :)")
	time.Sleep(750 * time.Millisecond)
	return func() tea.Msg {
		s := fmt.Sprintf("connection: %t", connection)
		log.Printf("Msg: %s", s)
		return ioMsg(s)
	}
}

// returns a boolean for the connection status of the ipod
func isConnected() bool {
	user := os.Getenv("USER")
	path := "/run/media/" + user + "/IPOD"

	_, err := os.Stat(path)
	return err == nil
}

func syncFiles() tea.Cmd {
	return func() tea.Msg {
		commands := []string{
			fmt.Sprintf("rsync -av --size-only %s %s",
				os.Getenv("SOURCE_MUSIC"),
				os.Getenv("DEST_MUSIC"),
			),
			fmt.Sprintf("rsync -av --size-only %s %s",
				os.Getenv("SOURCE_PODCASTS"),
				os.Getenv("DEST_PODCASTS"),
			),
			fmt.Sprintf("rsync -av --size-only %s %s",
				os.Getenv("SOURCE_AUDIOBOOKS"),
				os.Getenv("DEST_AUDIOBOOKS"),
			),
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

		switch string(msg) {
		case "connection: true":
			m.charger.head = "-"
			m.charger.colour = "82" // green
			return m, connectionMonitor()

		case "connection: false":
			m.charger.head = "^"
			m.charger.colour = "205" // pink
			return m, connectionMonitor()
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
	s := `
__________               __
\______   \ ____   ____ |  | _
 |       _//  _ \_/ ___\|  |/ /___________
 |    |   (  ♪_♪ )  \___|    ♪ \__    ___/_ __  ____   ____   ______
 |____|_  /\____/ \___  ♪__|_ \  |    | |  |  \/    \_/ __ \ /  ___/
        \/            \/     \/  |    | |  |  /   |  \  ___/ \___ \
                ＿  ♪            |____| |____/|___|  /\___  ♪____  ♪
               |■|♪                                \/     \/     \/
               |◎|
	`
	s += renderCharger(m.charger) + "\n"

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
	godotenv.Load()
	// Clear the log file at startup
	err := os.WriteFile("debug.log", []byte{}, 0644)
	if err != nil {
		fmt.Println("failed to clear log:", err)
		os.Exit(1)
	}
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
