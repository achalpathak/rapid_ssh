package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

var selected_item []string

type item struct {
	title, desc string
	cmd         []string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type model struct {
	list list.Model
}

func (m model) Init() tea.Cmd {
	return nil
}

type Server struct {
	Name            string `json:"name"`
	JumpHost        string `json:"gateway_host,omitempty"`
	GatewayUserName string `json:"gateway_user_name,omitempty"`
	TargetUserName  string `json:"target_user_name,omitempty"`
	TargetHost      string `json:"target_host,omitempty"`
	KeyPath         string `json:"key_path,omitempty"`
	ExecCommand     string `json:"exec_command,omitempty"`
}

type Conf struct {
	Servers         []Server
	DefaultUserName string `json:"default_user_name"`
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
		if msg.String() == "enter" {
			selected := m.list.SelectedItem()

			// If an item is selected, print its details
			if selected != nil {
				selectedItem := selected.(item)
				selected_item = selectedItem.cmd
				return m, tea.Quit
			}
			// return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	// fmt.Printf('cmd', cmd)
	return m, cmd
}

func (m model) View() string {
	return docStyle.Render(m.list.View())
}

func initCheck() (string, error) {
	// check is config.json exists otherwise create
	// Get the home directory
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting home directory: %v", err)
	}

	// Path to the config file
	configPath := filepath.Join(home, ".config", "rapid_ssh", "config.json")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create the directory if it doesn't exist
		err := os.MkdirAll(filepath.Dir(configPath), 0755)
		if err != nil {
			log.Fatalf("Error creating directory: %v", err)
		}

		// Create the config file
		file, err := os.Create(configPath)
		if err != nil {
			log.Fatalf("Error creating config file: %v", err)
		}
		defer file.Close()

		fmt.Println("Config file created at:", configPath)
	} else if err != nil {
		log.Fatalf("Error checking config file existence: %v", err)
	}

	return configPath, nil
}

func readConfig(filename string) (Conf, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return Conf{}, err
	}

	var conf Conf
	err = json.Unmarshal(file, &conf)
	if err != nil {
		return Conf{}, err
	}

	return conf, nil
}

func main() {
	configFile, err := initCheck()
	if err != nil {
		log.Fatalf("Error initializing program: %s", err)
	}

	conf, err := readConfig(configFile)
	if err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	var items []list.Item

	for _, server := range conf.Servers {
		var sshArgs []string
		gateway_user_name := conf.DefaultUserName
		target_user_name := conf.DefaultUserName

		if server.GatewayUserName != "" {
			gateway_user_name = server.GatewayUserName
		}
		if server.TargetUserName != "" {
			target_user_name = server.TargetUserName
		}

		if server.JumpHost != "" {
			sshArgs = []string{
				"ssh",
				"-J",
				fmt.Sprintf("%s@%s", gateway_user_name, server.JumpHost),
			}
		} else {
			sshArgs = []string{"ssh"}
		}

		sshArgs = append(sshArgs, fmt.Sprintf("%s@%s", target_user_name, server.TargetHost))
		// Set default key if not provided
		if server.KeyPath != "" {
			sshArgs = append(sshArgs, fmt.Sprintf("-i %s", server.KeyPath))
		}

		// Set exec command if provided
		if server.ExecCommand != "" {
			sshArgs = append(sshArgs, "-t")
			sshArgs = append(sshArgs, server.ExecCommand)
		}
		// Construct item and add it to items slice
		items = append(items, item{
			title: fmt.Sprintf("%s", server.Name),
			desc:  fmt.Sprintf("SSH to %s", server.TargetHost),
			cmd:   sshArgs,
		})
	}

	m := model{list: list.New(items, list.NewDefaultDelegate(), 0, 0)}
	m.list.Title = "Select a server to SSH into"

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {

		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	// Create the SSH command
	sshCmd := exec.Command(selected_item[0], selected_item[1:]...)
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr
	sshCmd.Stdin = os.Stdin
	fmt.Println("Connecting to server ...")
	err = sshCmd.Run()
	if err != nil {
		log.Fatalf("Error executing SSH command: %v", err)
	}
}
