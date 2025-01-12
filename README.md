# Process Party 🎉

Process Party is a powerful CLI tool that allows you to run and manage multiple processes simultaneously with unified standard output, interactive input capabilities, and file system triggers.

## Features

- Run multiple processes concurrently
- Unified standard output
- Interactive process management
- File system watching and triggers
- Process-to-process triggers
- Color-coded output
- Process status tracking
- Input piping to specific or all processes

## Installation

```bash
go install github.com/mpmcintyre/process-party
```

## Usage

### Configuration File

Process Party supports configuration files in three formats:

- YAML (`.yaml` or `.yml`)
- JSON (`.json`)
- TOML (`.toml`)

```bash
# Basic usage
process-party ./path/to/config.yaml

# Or with JSON or TOML
process-party ./path/to/config.json
process-party ./path/to/config.toml
```

### Inline Commands

```bash
# Multiple execute flags are supported
process-party ./path/to/config.yaml -e "npm run start" --execute "cmd echo hello"
```

### Global Configuration Options

| Option                | Type   | Description                   | Default |
| --------------------- | ------ | ----------------------------- | ------- |
| `indicate_every_line` | `bool` | Separate output for each line | `false` |
| `show_timestamp`      | `bool` | Display timestamps for output | `false` |

### Process Configuration Options

| Option               | Type       | Description                               | Possible Values                                              |
| -------------------- | ---------- | ----------------------------------------- | ------------------------------------------------------------ |
| `name`               | `string`   | Unique name for the process               | Any string                                                   |
| `command`            | `string`   | Command to execute                        | Any valid shell command                                      |
| `args`               | `[]string` | Arguments for the command                 | List of strings                                              |
| `prefix`             | `string`   | Prefix for output lines                   | Any string                                                   |
| `color`              | `string`   | Output color for the process              | `yellow`, `blue`, `green`, `red`, `cyan`, `white`, `magenta` |
| `on_failure`         | `string`   | Action on process failure                 | `buzzkill`, `wait`, `restart`                                |
| `on_complete`        | `string`   | Action on process completion              | `buzzkill`, `wait`, `restart`                                |
| `seperate_new_lines` | `bool`     | Separate output for each line             | `true`/`false`                                               |
| `show_pid`           | `bool`     | Display process ID                        | `true`/`false`                                               |
| `silent`             | `bool`     | Mute output from command                  | `true`/`false`                                               |
| `delay`              | `int`      | Initial delay before starting             | Milliseconds                                                 |
| `timeout_on_exit`    | `int`      | Timeout when exiting                      | Milliseconds                                                 |
| `restart_delay`      | `int`      | Delay before restarting                   | Milliseconds                                                 |
| `restart_attempts`   | `int`      | Number of restart attempts before exiting | Integer (negative implies always restart)                    |

### File System Trigger Options

| Option          | Type       | Description                      | Possible Values  |
| --------------- | ---------- | -------------------------------- | ---------------- |
| `non_recursive` | `bool`     | Do not watch subdirectories      | `true`/`false`   |
| `watch`         | `[]string` | Directories/files to watch       | List of paths    |
| `ignore`        | `[]string` | Directories/files to ignore      | List of paths    |
| `filter_for`    | `[]string` | File patterns to include/exclude | List of patterns |

### Process Trigger Options

| Option        | Type       | Description                           | Possible Values       |
| ------------- | ---------- | ------------------------------------- | --------------------- |
| `on_start`    | `[]string` | Trigger when these processes start    | List of process names |
| `on_complete` | `[]string` | Trigger when these processes complete | List of process names |
| `on_error`    | `[]string` | Trigger when these processes error    | List of process names |

## Example Configuration

```yaml
# Global settings
indicate_every_line: true # Show prefix on every line
show_timestamp: true # Show timestamps in output

processes:
  - name: "web-server" # Process name
    command: "npm" # Command to run
    args: ["start"] # Command arguments
    prefix: "web" # Output prefix
    color: "green" # Prefix color
    show_pid: true # Show process ID in output

    # Process behavior
    delay: 0 # Startup delay in milliseconds
    restart_attempts: 0 # Number of restart attempts (-1 for infinite)
    restart_delay: 0 # Delay before restart

    # File system triggers
    trigger:
      filesystem:
        watch: ["./src"] # Directories to watch
        ignore: ["node_modules"] # Directories to ignore
        filter_for: [".js", ".jsx"] # File filters
        non_recursive: false # Watch subdirectories

      # Process triggers
      process:
        on_start: ["db"] # Run when these processes start
        on_complete: [] # Run when these processes complete
        on_error: [] # Run when these processes error

  - name: "database"
    command: "mongod"
    prefix: "db"
    color: "blue"
    on_failure: "buzzkill" # Exit behavior on failure
```

## Interactive CLI Usage

When running Process Party, you can interact with processes using these commands:

- `all:<input>`: Send input to all running processes
- `<process-name>:<input>` or `<process-prefix>:<input>`: Send input to a specific process
- `status` or `s`: Display status of all processes
- `exit`: Terminate all processes
- `help`: Show available commands

### Example

```bash
# Start Process Party
process-party ./config.yaml

# Send input to all processes
> all:start

# Send input to a specific process
> web-server:reload

# Check process status
> status

# Exit
> exit
```

## Exit Statuses

| Status       | Description                  |
| ------------ | ---------------------------- |
| `running`    | Process is active            |
| `exited`     | Process completed normally   |
| `failed`     | Process encountered an error |
| `restarting` | Process is being restarted   |

## License

MIT

## Author

Michael
