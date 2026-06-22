<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# Client

## Purpose
Remote CLI client for controlling and monitoring a running StormSIM emulator instance via HTTP/JSON API. Provides both interactive shell mode and single-command mode for scripting and automation.

## Key Files
| File | Description |
|------|-------------|
| `client.go` | HTTP client with interactive shell (ishell), watch mode, and context-based command routing |

## Subdirectories
None - this is a leaf directory containing a single entry point.

## For AI Agents

### Working In This Directory
- **Single source file** - All client logic is in `client.go`
- **Two execution modes**:
  - Single command: `./client <command> [args]` - executes one command and exits
  - Interactive: `./client` - launches ishell REPL with prompt `stormsim>`
- **Watch mode** - Add `-w` or `--watch` to any command for periodic refresh
- **Context tracking** - Client maintains `ctxId` for navigating command hierarchy (stormsim → ue:xxx → gnb:xxx)

### Testing Requirements
```bash
# Build client binary
go build -o bin/client ./cmd/client

# Test single command mode (requires running emulator on port 4000)
./bin/client stats

# Test watch mode
./bin/client stats -w -n 2s

# Test interactive mode
./bin/client
```

### Common Patterns

**Request/Response Types:**
```go
type CmdRequest struct {
    ContextId string   `json:"ContextId"`  // Current context (e.g., "stormsim", "ue:12345")
    Name      string   `json:"Name"`       // Command name (e.g., "stats", "list-ue")
    Args      []string `json:"Args"`       // Command arguments
}

type CmdResponse struct {
    Message string       `json:"Message"`  // Output text from command
    Error   string       `json:"Error"`    // Error message if failed
    Context *ContextInfo `json:"Context"`  // New context info (prompt, commands)
}
```

**Single Command Mode:**
```go
func handleSingleCommand(args []string) {
    ctxId := "stormsim"
    availableCommands := convertCommands(oambackend.EmuCmds)
    executeCommandLogic(nil, &ctxId, args, nil, &availableCommands)
}
```

**Watch Mode Pattern:**
```go
// Parse watch flags
if arg == "--watch" || arg == "-w" {
    watch = true
    continue
}
if arg == "--interval" || arg == "-n" {
    if d, err := time.ParseDuration(args[i+1]); err == nil {
        interval = d
        i++
        continue
    }
}

// Watch loop with signal handling
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt)
for {
    sendRequest(...)
    select {
    case <-sigChan:
        break loop
    case <-time.After(interval):
    }
}
```

**HTTP Request:**
```go
req := CmdRequest{ContextId: *ctxId, Name: name, Args: args}
jsonData, _ := json.Marshal(req)
resp, _ := http.Post("http://localhost:4000/cmd", "application/json", bytes.NewBuffer(jsonData))
```

### Available Commands
Commands are defined in `monitoring/oambackend.EmuCmds`. Root context commands include:
- `stats` - Show emulator statistics
- `list-ue` - List all UEs
- `list-gnb` - List all gNBs
- `select-ue <msin>` - Enter UE context
- `select-gnb <gnbid>` - Enter gNB context
- `group-delay-stats` - Show group delay statistics

### Client-Server Protocol
1. Client sends `CmdRequest` JSON via POST to `http://localhost:4000/cmd`
2. Server routes to appropriate handler based on `ContextId` and `Name`
3. Server returns `CmdResponse` with output and optional new context
4. Client updates prompt and available commands from response

### Limitations
- **Hardcoded port**: Server URL is `localhost:4000` - not configurable
- **No authentication**: Assumes trusted local network
- **No TLS**: HTTP only, not HTTPS

## Dependencies

### Internal
- `monitoring/oambackend` - `EmuCmds` map for command definitions, shared `cli.Command` types

### External
- `github.com/abiosoft/ishell` - Interactive shell library with readline, command history, tab completion
- `github.com/urfave/cli/v3` - CLI types (`cli.Command`) for command structure

<!-- MANUAL: Add configuration options (server URL, port) if remote emulator control needed -->
