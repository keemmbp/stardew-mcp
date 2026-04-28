# Stardew Valley MCP Bridge

Stardew Valley MCP Bridge connects Stardew Valley to MCP clients such as Gemini CLI or Claude Desktop. A SMAPI mod streams live game state over WebSocket, and a Go MCP server exposes tools for AI-assisted play.

## Architecture

```text
Stardew Valley + SMAPI mod
  ModEntry -> GameStateSerializer -> CommandExecutor -> WebSocketServer
  ws://localhost:8765/game

Go MCP server
  GameClient tracks WebSocket state
  MCP tools/prompts over stdio

MCP client
  Gemini CLI, Claude Desktop, or another stdio MCP client
```

## Requirements

- Stardew Valley 1.6+
- SMAPI 4.0+
- .NET 6 SDK for building the mod
- Go 1.23+ for building the MCP server
- An MCP client such as Gemini CLI

## Build

Build the SMAPI mod:

```bash
cd mod/StardewMCP
dotnet build
```

`Pathoschild.Stardew.ModBuildConfig` deploys the mod to the detected SMAPI Mods folder when possible. On macOS Steam, SMAPI commonly uses:

```text
~/Library/Application Support/Steam/steamapps/common/Stardew Valley/Contents/MacOS/Mods/StardewMCP
```

Build the MCP server:

```bash
cd mcp-server
go build -o stardew-mcp
```

Do not commit build outputs such as `bin/`, `obj/`, release zips, or `mcp-server/stardew-mcp`.

## Mod Configuration

After the first launch, SMAPI creates `Mods/StardewMCP/config.json`:

```json
{
  "WebSocketPort": 8765,
  "EnableCheats": false,
  "StateBroadcastIntervalSeconds": 1
}
```

Cheat commands are disabled by default. To use them, both layers must opt in:

- Set `EnableCheats` to `true` in the mod config.
- Start the MCP server with `-enable-cheats`.
- Call `cheat_mode_enable` before any other cheat tool.

## Gemini CLI

Normal mode:

```json
{
  "mcpServers": {
    "stardew-mcp": {
      "command": "/absolute/path/to/stardew-mcp/mcp-server/stardew-mcp",
      "args": ["-url", "ws://localhost:8765/game"],
      "timeout": 30000,
      "trust": false
    }
  }
}
```

Cheat-enabled mode:

```json
{
  "mcpServers": {
    "stardew-mcp": {
      "command": "/absolute/path/to/stardew-mcp/mcp-server/stardew-mcp",
      "args": ["-url", "ws://localhost:8765/game", "-enable-cheats"],
      "timeout": 30000,
      "trust": false
    }
  }
}
```

Typical run order:

1. Open Stardew Valley through SMAPI.
2. Load a save file.
3. Start or restart Gemini CLI.
4. Run `/mcp` and confirm `stardew-mcp` is connected.
5. Ask for `get_surroundings` to confirm player, time, map, and inventory state are visible.

## Docker

Only the Go MCP server is suitable for Docker. Stardew Valley and SMAPI should run on the host desktop.

```dockerfile
FROM golang:1.25-alpine AS build
WORKDIR /app
COPY mcp-server/go.mod mcp-server/go.sum ./
RUN go mod download
COPY mcp-server/*.go ./
RUN go build -o /stardew-mcp

FROM alpine:3.20
COPY --from=build /stardew-mcp /usr/local/bin/stardew-mcp
ENTRYPOINT ["stardew-mcp"]
```

On macOS/Windows Docker, point the container at the host game:

```bash
docker run --rm -i stardew-mcp -url ws://host.docker.internal:8765/game
```

## Tools

Default tools:

| Tool | Description |
| --- | --- |
| `get_surroundings` | Current game state plus 61x61 ASCII vision |
| `move_to` | A* pathfinding to a walkable tile |
| `face_direction` | Face `up`, `down`, `left`, or `right` |
| `interact` | Trigger the game action button at the tile in front |
| `use_tool` | Use the equipped tool on the tile in front |
| `use_tool_repeat` | Use the equipped tool multiple times |
| `select_item` | Equip an inventory item by name |
| `switch_tool` | Equip an inventory slot |
| `eat_item` | Eat an item from an inventory slot |
| `enter_door` | Use a door or warp point in front |
| `find_best_target` | Suggest an accessible target and action sequence |
| `clear_target` | Ask the mod to clear the nearest target type |

Cheat tools are registered only with `-enable-cheats` and require mod config opt-in. They include teleportation, farm cleanup, crop automation, inventory/resource edits, friendship edits, and upgrades.

## WebSocket Protocol

The mod and MCP server exchange JSON over WebSocket:

```json
{
  "id": "uuid",
  "type": "command",
  "action": "move_to",
  "params": { "x": 10, "y": 20 }
}
```

```json
{
  "id": "uuid",
  "type": "response",
  "success": true,
  "message": "Moved to position",
  "data": {}
}
```

State messages are sent automatically while a save is loaded.

## Troubleshooting

`/mcp` shows `stardew-mcp - Disconnected`:

- Rebuild the MCP server after code changes: `cd mcp-server && go build -o stardew-mcp`.
- Make sure the configured Gemini path points to that binary.
- Start Gemini CLI after updating `~/.gemini/settings.json`.
- Check that the MCP server starts without schema panic by running `./stardew-mcp -h`.

Gemini connects but cannot see game data:

- Launch Stardew Valley through SMAPI and load a save.
- Confirm SMAPI loaded the mod. The log should include `Stardew MCP Bridge loaded!`.
- Confirm the WebSocket port is open:

```bash
lsof -nP -iTCP:8765 -sTCP:LISTEN
```

SMAPI does not load the mod:

- Put the mod in the Mods folder printed near the top of `SMAPI-latest.txt`.
- On macOS Steam, this is usually inside `Contents/MacOS/Mods`, not `~/.config/StardewValley/Mods`.
- Rebuild with .NET 6 and restart the game; SMAPI loads mods only at game launch.

Pathfinding fails:

- `move_to` must target a walkable tile, not the object/tree/rock tile itself.
- Use `find_best_target` to get an adjacent approach tile and facing direction.
