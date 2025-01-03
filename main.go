package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	path string
	keys = keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter/space", "toggle selection"),
		),
	}
)

// keyMap defines a set of keybindings. To work for help it must satisfy
// key.Map. It could also very easily be a map[string]key.Binding.
type keyMap struct {
	Up    key.Binding
	Down  key.Binding
	Help  key.Binding
	Quit  key.Binding
	Enter key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},   // first column
		{k.Help, k.Quit}, // second column
	}
}

type model struct {
	choices    []string         // items on the to-do list
	cursor     int              // which to-do list item our cursor is pointing at
	selected   map[int]struct{} // which to-do items are selected
	keys       keyMap
	help       help.Model
	inputStyle lipgloss.Style
	quitting   bool
}

func initialModel() model {
	return model{
		choices:    []string{},
		selected:   make(map[int]struct{}),
		keys:       keys,
		help:       help.New(),
		inputStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#FF75B7")),
	}
}

// LoadSSHKeysCmd is a custom command that reads the SSH keys and returns an update function
func (m model) LoadSSHKeysCmd() tea.Cmd {
	return func() tea.Msg {
		file, err := os.Open(path)
		if err != nil {
			return fmt.Sprintf("Error opening file: %v", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		var isSSHKeys bool
		var currentLines, choices []string
		var lineCount int

		for scanner.Scan() {
			line := scanner.Text()
			trimmedLine := strings.TrimSpace(line)

			if trimmedLine == "" {
				continue
			}
			if strings.Contains(trimmedLine, "[[ssh-keys]]") {
				if trimmedLine[0] != '#' {
					m.selected[len(choices)] = struct{}{}
				}
				isSSHKeys = true
				lineCount = 1
			} else if isSSHKeys && lineCount > 0 {
				lineCount++
				currentLines = append(currentLines, line)

				if lineCount == 2 && strings.Contains(trimmedLine, "vault") {
					// If we have 2 lines, we have a vault
					vaultName := strings.Split(currentLines[0], " = ")[1]
					choices = append(choices, fmt.Sprintf("Vault %s", vaultName))
					currentLines = nil
					isSSHKeys = false
				} else if lineCount == 3 {
					// If we have 3 lines, we have an item in a vault
					itemName := strings.Split(currentLines[0], " = ")[1]
					vaultName := strings.Split(currentLines[1], " = ")[1]
					choices = append(choices, fmt.Sprintf("Item %s in Vault %s", itemName, vaultName))
					currentLines = nil
					isSSHKeys = false
				}
			}
		}
		if len(currentLines) == 2 {
			vaultName := strings.Split(currentLines[1], " = ")[1]
			choices = append(choices, fmt.Sprintf("Vault: %s\n", vaultName))
		}

		if err := scanner.Err(); err != nil {
			return fmt.Sprintf("Error reading file: %v", err)
		}

		return choices
	}
}

func (m model) UpdateSSHKeysConfig() {
	file, err := os.Create(path)
	if err != nil {
		fmt.Printf("Error creating file: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	for i, choice := range m.choices {
		// Check if the item is selected
		if _, ok := m.selected[i]; ok {
			// Write active [[ssh-keys]] blocks
			if strings.Contains(choice, "Item") && strings.Contains(choice, "Vault") {
				item := strings.Split(choice, "\"")[1]
				vault := strings.Split(choice, "\"")[3]
				sshkey := fmt.Sprintf("[[ssh-keys]]\nitem = \"%s\"\nvault = \"%s\"\n\n", item, vault)
				_, err = writer.WriteString(sshkey)
			} else if strings.Contains(choice, "Vault") {
				vault := strings.Split(choice, "\"")[1]
				sshkey := fmt.Sprintf("[[ssh-keys]]\nvault = \"%s\"\n\n", vault)
				_, err = writer.WriteString(sshkey)
			}
		} else {
			// Write inactive (commented-out) [[ssh-keys]] blocks
			if strings.Contains(choice, "Item") && strings.Contains(choice, "Vault") {
				item := strings.Split(choice, "\"")[1]
				vault := strings.Split(choice, "\"")[3]
				sshkey := fmt.Sprintf("#[[ssh-keys]]\n#item = \"%s\"\n#vault = \"%s\"\n\n", item, vault)
				_, err = writer.WriteString(sshkey)
			} else if strings.Contains(choice, "Vault") {
				vault := strings.Split(choice, "\"")[1]
				sshkey := fmt.Sprintf("#[[ssh-keys]]\n#vault = \"%s\"\n\n", vault)
				_, err = writer.WriteString(sshkey)
			}
		}

		// Check for write errors
		if err != nil {
			fmt.Printf("Error writing to file: %v", err)
		}
	}

	if err := writer.Flush(); err != nil {
		fmt.Printf("Error flushing writer: %v", err)
	}
}

func (m model) Init() tea.Cmd {
	return m.LoadSSHKeysCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		// If we set a width on the help menu it can gracefully truncate
		// its view as needed.
		m.help.Width = msg.Width

	case string:
		fmt.Println(msg) // Print error message if any

	case []string:
		m.choices = msg

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):

			// Update the SSH keys config file before quitting
			m.UpdateSSHKeysConfig()

			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			} else if m.cursor == 0 {
				m.cursor = len(m.choices) - 1
			}
		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			} else if m.cursor == len(m.choices)-1 {
				m.cursor = 0
			}
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		case key.Matches(msg, m.keys.Enter):
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return "Bye!\n"
	}

	s := "Toggle which SSH keys you would like\n\n"
	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		checked := " "
		if _, ok := m.selected[i]; ok {
			checked = "x"
		}
		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	helpView := m.help.View(m.keys)
	height := len(m.choices) - strings.Count(s, "\n") - strings.Count(helpView, "\n")
	if height < 1 {
		height = 1
	}

	return s + strings.Repeat("\n", height) + helpView
}

func main() {

	if runtime.GOOS == "windows" {
		path = fmt.Sprintf("C:\\Users\\%s\\AppData\\Local\\1Password\\config\\ssh\\agent.toml", os.Getenv("USERNAME"))
		cmd := exec.Command("cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else if runtime.GOOS == "linux" {
		path = fmt.Sprintf("/home/%s/.config/1Password/ssh/agent.toml", os.Getenv("USER"))
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()

	}

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
