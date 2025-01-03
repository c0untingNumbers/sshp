package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	path string
)

type model struct {
	choices  []string         // items on the to-do list
	cursor   int              // which to-do list item our cursor is pointing at
	selected map[int]struct{} // which to-do items are selected
}

func initialModel() model {
	return model{
		choices:  []string{},
		selected: make(map[int]struct{}),
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
	case string:
		fmt.Println(msg) // Print error message if any
	case []string:
		m.choices = msg
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.UpdateSSHKeysConfig()
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else if m.cursor == 0 {
				m.cursor = len(m.choices) - 1
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			} else if m.cursor == len(m.choices)-1 {
				m.cursor = 0
			}
		case "enter", " ":
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
	s := "What should we buy at the market?\n\n"
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
	s += "\nPress q to quit.\n"
	return s
}

func main() {
	if runtime.GOOS == "windows" {
		path = fmt.Sprintf("C:\\Users\\%s\\AppData\\Local\\1Password\\config\\ssh\\agent.toml", os.Getenv("USER"))
	} else if runtime.GOOS == "linux" {
		path = fmt.Sprintf("/home/%s/.config/1Password/ssh/agent.toml", os.Getenv("USER"))
	}
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
