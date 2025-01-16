package pp

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	color "github.com/fatih/color"
	"gopkg.in/yaml.v3"
)

type (
	ExitCommand string
	ProcessType int
	ColourCode  string
	Verbosity   int

	FileSystemTrigger struct {
		NonRecursive   bool     `toml:"non_recursive" json:"non_recursive" yaml:"non_recursive"` // Do not recursively watch new directories
		DebounceTime   uint16   `toml:"debounce_ms" json:"debounce_ms" yaml:"debounce_ms"`       // Time between fs triggers
		Watch          []string `toml:"watch" json:"watch" yaml:"watch"`                         // List of directories/folders to watch
		Ignore         []string `toml:"ignore" json:"ignore" yaml:"ignore"`                      // List of directories/folders to ignore
		ContainFilters []string `toml:"filter_for" json:"filter_for" yaml:"filter_for"`          // Include or exclude files
	}

	ProcessTrigger struct {
		OnStart    []string `toml:"on_start" json:"on_start" yaml:"on_start"`          // Trigger run when listed process started successfully
		OnComplete []string `toml:"on_complete" json:"on_complete" yaml:"on_complete"` // Trigger run when listed process exits successfully
		OnError    []string `toml:"on_error" json:"on_error" yaml:"on_error"`          // Trigger run when listed process errors
	}

	Trigger struct {
		RunOnStart bool              `toml:"run_on_start" json:"run_on_start" yaml:"run_on_start"`          // Runs the process on starting process party
		EndOnNew   bool              `toml:"restart_process" json:"restart_process" yaml:"restart_process"` // End old process on new trigger
		FileSystem FileSystemTrigger `toml:"filesystem" json:"filesystem" yaml:"filesystem"`                // Filesystem triggers
		Process    ProcessTrigger    `toml:"process" json:"process" yaml:"process"`                         // Process triggers
	}

	Process struct {
		// Info
		Name             string     `toml:"name" json:"name" yaml:"name"`                                           // Name of the process
		Command          string     `toml:"command" json:"command" yaml:"command"`                                  // Command to run
		Args             []string   `toml:"args" json:"args" yaml:"args"`                                           // Arguments for the command
		Prefix           string     `toml:"prefix" json:"prefix" yaml:"prefix"`                                     // Prefix used for printing (empty for none)
		Color            ColourCode `toml:"color" json:"color" yaml:"color"`                                        // Customize prefix colour
		SeperateNewLines bool       `toml:"seperate_new_lines" json:"seperate_new_lines" yaml:"seperate_new_lines"` // Display prefix on every line or only once per output sequence
		DisplayPid       bool       `toml:"show_pid" json:"show_pid" yaml:"show_pid"`                               // Show the PID of the process
		StartStream      string     `toml:"stdin_on_start" json:"stdin_on_start" yaml:"stdin_on_start"`             // Stream sequence to the command on startup
		Silent           bool       `toml:"silent" json:"silent" yaml:"silent"`                                     // Mute output from the command
		// Behaviour
		Trigger         Trigger     `toml:"trigger" json:"trigger" yaml:"trigger"`                                           // Any triggers that can start the process
		Delay           int         `toml:"delay" json:"delay" yaml:"delay"`                                                 // Delay on starting the process
		RestartDelay    int         `toml:"restart_delay" json:"restart_delay" yaml:"restart_delay"`                         // Delay before restarting the process
		OnFailure       ExitCommand `toml:"on_failure" json:"on_failure" yaml:"on_failure"`                                  // Exit behaviour on process failure
		OnComplete      ExitCommand `toml:"on_complete,omitempty" json:"on_complete,omitempty" yaml:"on_complete,omitempty"` // Exit behaviour on successful exit
		RestartAttempts int         `toml:"restart_attempts" json:"restart_attempts" yaml:"restart_attempts"`                // Restart attempts for the process (<0 to always restart)
		TimeoutOnExit   int         `toml:"timeout_on_exit" json:"timeout_on_exit" yaml:"timeout_on_exit"`                   // Time allowed for the process to exit when externally killed before killing the process
		// Runtime
		ShowTimestamp bool   // Show timestamp private setting obtained from config
		Pid           string // Private PID value assigned on process successful start
	}

	Config struct {
		Processes        []Process `toml:"processes" json:"processes" yaml:"processes"`
		SeperateNewLines bool      `toml:"indicate_every_line" json:"indicate_every_line" yaml:"indicate_every_line"`
		ShowTimestamp    bool      `toml:"show_timestamp" json:"show_timestamp" yaml:"show_timestamp"`
		filePresent      bool
	}
)

const (
	ExitCommandBuzzkill ExitCommand = "buzzkill"
	ExitCommandWait     ExitCommand = "wait"
	ExitCommandRestart  ExitCommand = "restart"
)

const (
	ColourCmdYellow  ColourCode = "yellow"
	ColourCmdBlue    ColourCode = "blue"
	ColourCmdGreen   ColourCode = "green"
	ColourCmdRed     ColourCode = "red"
	ColourCmdCyan    ColourCode = "cyan"
	ColourCmdWhite   ColourCode = "white"
	ColourCmdMagenta ColourCode = "magenta"
)

// Gets the coloured print function for the writer
func (p *Process) GetFgColour() func(format string, a ...interface{}) string {
	switch p.Color {
	case ColourCmdYellow:
		return color.YellowString
	case ColourCmdBlue:
		return color.BlueString
	case ColourCmdGreen:
		return color.GreenString
	case ColourCmdRed:
		return color.RedString
	case ColourCmdCyan:
		return color.CyanString
	case ColourCmdWhite:
		return color.WhiteString
	case ColourCmdMagenta:
		return color.MagentaString
	default:
		return color.WhiteString
	}
}

// Returns if the process has an fs trigger
func (p *Process) HasFsTrigger() bool {
	return len(p.Trigger.FileSystem.Watch) > 0
}

// Returns if the process has a process trigger
func (p *Process) HasProcessTrigger() bool {
	return len(p.Trigger.Process.OnComplete) > 0 ||
		len(p.Trigger.Process.OnStart) > 0 ||
		len(p.Trigger.Process.OnError) > 0
}

// Returns if the process has an fs trigger or process trigger
func (t *Process) HasTrigger() bool {
	return t.HasFsTrigger() || t.HasProcessTrigger()
}

func (c *Config) GenerateExampleConfig(path string) error {

	fmt.Printf("Generating config - %s\n", path)

	exampleProcess := Process{
		Name:             "my process",
		Prefix:           "EXAMPLE",
		Command:          "ls",
		OnFailure:        "wait",
		OnComplete:       "wait",
		Args:             []string{},
		Color:            ColourCmdGreen,
		DisplayPid:       false,
		Silent:           false,
		SeperateNewLines: false,
		Delay:            0,
		RestartDelay:     0,
		RestartAttempts:  0,
		TimeoutOnExit:    1,
		StartStream:      "",
		Trigger: Trigger{
			FileSystem: FileSystemTrigger{
				Watch:          []string{},
				Ignore:         []string{},
				ContainFilters: []string{},
			},
			Process: ProcessTrigger{
				OnStart:    []string{},
				OnComplete: []string{},
				OnError:    []string{},
			},
		},
	}

	c.SeperateNewLines = true
	c.ShowTimestamp = true
	c.Processes = append(c.Processes, exampleProcess)

	dotSplit := strings.Split(path, ".")
	filetype := dotSplit[len(dotSplit)-1]
	var data []byte
	var err error
	switch strings.ToLower(filetype) {
	case "toml":
		data, err = toml.Marshal(c)
	case "yaml":
		data, err = yaml.Marshal(c)
	case "yml":
		data, err = yaml.Marshal(c)
	case "json":
		data, err = json.Marshal(c)
	default:
		return errors.New("unknown filetype provided for config generation -- .toml, .yaml/.yml, or .json supported")

	}

	if err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	if err != nil {
		return err
	}

	fmt.Println("Config successfully generated")

	return nil
}

// Creates an empty default config that needs to be populated
func CreateConfig() *Config {
	return &Config{
		Processes:        []Process{},
		SeperateNewLines: true,
		ShowTimestamp:    true,
		filePresent:      false,
	}
}

func (c *Config) ScanDir(path string) (string, error) {
	dirs, err := os.ReadDir(path)

	if err != nil {
		return "", err
	}
	for _, directory := range dirs {
		if !directory.IsDir() {
			if strings.Contains(directory.Name(), "process-party") {
				return directory.Name(), nil
			}
		}

	}

	return "", nil
}

// Parses an inline command (not config related) to be added to the config
func (c *Config) ParseInlineCmd(cmd string) error {
	s := strings.Split(cmd, " ")
	if len(s) == 0 {
		return errors.New("empty command provided")
	}
	command := s[0]
	args := []string{}
	prefix := command
	if len(s) > 1 {
		args = s[:1]
	}

	// Count how many processes have the same name and increment it for the prefix
	count := 0
	for _, process := range c.Processes {
		if strings.Contains(process.Command, command) {
			count++
		}
	}

	if count != 0 {
		prefix = fmt.Sprintf("cmd%d", count)
	}

	p := Process{
		OnFailure:  ExitCommandBuzzkill,
		OnComplete: ExitCommandWait,
		Command:    command,
		Args:       args,
		Prefix:     prefix,
	}
	c.Processes = append(c.Processes, p)

	return nil
}

// Reads a config file and attempts to parse the configuration
func (c *Config) ParseFile(path string, silent bool) error {
	origin, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	_, err = os.ReadDir(origin)

	if err == nil {
		newpath, err := c.ScanDir(origin)
		if err != nil {
			return err
		}
		if newpath == "" {
			return errors.New("Unusable file format or directory not containing \"process-party.[yml|toml|json]\" not provided - " + origin)
		}
		path = filepath.Join(origin, newpath)
		color.HiBlack("\nFound process-party config file: %s \n\n", path)
	}

	extensions := strings.Split(path, ".")
	buffer, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(buffer)

	switch extensions[len(extensions)-1] {
	case "toml":
		_, err := toml.Decode(text, &c)
		if err != nil {
			return err
		}

	case "json":
		err = json.Unmarshal([]byte(text), &c)
		if err != nil {
			return err
		}

	case "yaml":
		err = yaml.Unmarshal([]byte(text), &c)
		if err != nil {
			return err
		}
	case "yml":
		err = yaml.Unmarshal([]byte(text), &c)
		if err != nil {
			return err
		}
	default:
		return errors.New("unsupported filetype provided")
	}

	uniqueChecks := map[string]bool{}

	if !silent {
		color.HiGreen("Found %d processes in %s", len(c.Processes), path)
		color.HiBlack("Process tasks:")
	}
	outputString := "["
	waitingString := "["
	waitCounter := 0
	for i := range c.Processes {
		output := fmt.Sprintf("%#v", c.Processes[i].Name)
		if c.Processes[i].Silent {
			output = fmt.Sprintf("%#v (silent)", c.Processes[i].Name)
		}
		if i == len(c.Processes)-1 {
			outputString += output
			// Triggers
			if c.Processes[i].HasTrigger() {
				waitingString += output
				waitCounter++
			}
		} else {
			outputString += fmt.Sprintf("%s, ", output)
			if c.Processes[i].HasTrigger() {
				waitingString += fmt.Sprintf("%s, ", output)
				waitCounter++
			}
		}

		// Fix broken commands (command is 1 value, append the rest to args)
		cmdSplit := strings.Split(c.Processes[i].Command, " ")
		if len(cmdSplit) > 1 {
			c.Processes[i].Args = append(strings.Split(c.Processes[i].Command, " ")[:1], c.Processes[i].Args...)
			c.Processes[i].Command = strings.Split(c.Processes[i].Command, " ")[0]
		}
		// Set general values
		c.Processes[i].SeperateNewLines = c.SeperateNewLines
		c.Processes[i].ShowTimestamp = c.ShowTimestamp
		if c.Processes[i].Trigger.FileSystem.DebounceTime == 0 {
			c.Processes[i].Trigger.FileSystem.DebounceTime = 1
		}
		// Check for duplicate uniques
		if uniqueChecks[c.Processes[i].Name] {
			return errors.New("Config contains duplicate unique fields. Offending item: Name - " + c.Processes[i].Name)
		} else {
			uniqueChecks[c.Processes[i].Name] = true
		}

	}
	if !silent {
		color.HiBlack("%s]\n\n", outputString)
		if waitCounter == 1 {
			color.HiGreen("%d process waiting for triggers", waitCounter)
		} else {
			color.HiGreen("%d processes waiting for triggers", waitCounter)
		}
		if waitCounter > 0 {
			color.HiBlack("%s]\n\n", waitingString)
		}
	}

	c.filePresent = true

	return nil
}

// Utility function to check if a directory exists
func DirectoryExists(dirname string, pathToDir string) (bool, error) {
	dirs, err := os.ReadDir(pathToDir)
	if err != nil {
		return false, err
	}

	for _, dir := range dirs {
		if dir.Name() == dirname && dir.Type().IsDir() {
			return true, nil
		}
	}

	return false, nil
}

// Utility function to check if a file exists
func FileExists(filename string, path string) (bool, error) {
	dirs, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}

	for _, dir := range dirs {
		if dir.Name() == filename && dir.Type().IsRegular() {
			return true, nil
		}
	}

	return false, nil
}
