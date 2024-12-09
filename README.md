# Process Party 🎉

Process Party is a powerful CLI tool that allows you to run and manage multiple processes simultaneously with unified standard output and interactive input capabilities.

## Features

- Run multiple processes concurrently
- Unified standard output
- Interactive process management
- Configurable process behaviors
- Color-coded output
- Process status tracking
- Input piping to specific or all processes

## Installation

```bash
# Installation instructions (replace with actual installation method)
go install github.com/mpmcintyre/process-party
```

## Usage

### Configuration File

Process Party supports configuration files in three formats:

- YAML (`.yaml` or `.yml`)
- JSON (`.json`)
- TOML (`.toml`)

Run Process Party by specifying the path to your configuration file:

```bash
# Basic usage
process-party ./path/to/config.yaml

# Or with JSON or TOML
process-party ./path/to/config.json
process-party ./path/to/config.toml
```

### Inline Commands

You can also specify inline commands using the `--execute` (or `-e`) flag:

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

| Option               | Type       | Description                   | Possible Values                                              |
| -------------------- | ---------- | ----------------------------- | ------------------------------------------------------------ |
| `name`               | `string`   | Unique name for the process   | Any string                                                   |
| `command`            | `string`   | Command to execute            | Any valid shell command                                      |
| `args`               | `[]string` | Arguments for the command     | List of strings                                              |
| `prefix`             | `string`   | Prefix for output lines       | Any string                                                   |
| `color`              | `string`   | Output color for the process  | `yellow`, `blue`, `green`, `red`, `cyan`, `white`, `magenta` |
| `on_failure`         | `string`   | Action on process failure     | `buzzkill`, `wait`, `restart`                                |
| `on_complete`        | `string`   | Action on process completion  | `buzzkill`, `wait`, `restart`                                |
| `seperate_new_lines` | `bool`     | Separate output for each line | `true`/`false`                                               |
| `show_pid`           | `bool`     | Display process ID            | `true`/`false`                                               |
| `delay`              | `int`      | Initial delay before starting | Seconds                                                      |
| `timeout_on_exit`    | `int`      | Timeout when exiting          | Seconds                                                      |
| `restart_delay`      | `int`      | Delay before restarting       | Seconds                                                      |

## Example Configurations

### YAML Example

```yaml
processes:
  - name: web-server
    command: python
    args: ["-m", "http.server"]
    color: green
    on_failure: restart
    restart_delay: 5

  - name: database
    command: postgres
    color: blue
    show_pid: true
```

### JSON Example

```json
{
  "processes": [
    {
      "name": "web-server",
      "command": "python",
      "args": ["-m", "http.server"],
      "color": "green",
      "on_failure": "restart",
      "restart_delay": 5
    },
    {
      "name": "database",
      "command": "postgres",
      "color": "blue",
      "show_pid": true
    }
  ]
}
```

### TOML Example

```toml
[[processes]]
name = "web-server"
command = "python"
args = ["-m", "http.server"]
color = "green"
on_failure = "restart"
restart_delay = 5

[[processes]]
name = "database"
command = "postgres"
color = "blue"
show_pid = true
```

## Interactive CLI Usage

When running Process Party, you can interact with processes using these commands:

- `all:<input>`: Send input to all running processes
- `<process-name>:<input>` or `<process-prefix>:<input>`: Send input to a specific process
- `status`: Display status of all processes
- `exit`: Terminate all processes
- `quit` or `Ctrl+C`: Exit the application

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

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT

## Author

Michael
