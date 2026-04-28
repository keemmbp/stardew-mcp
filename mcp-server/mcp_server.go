package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sort"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Parameter structs with jsonschema tags for tool definition
type MoveParams struct {
	X int `json:"x" jsonschema:"description=The X coordinate to move to"`
	Y int `json:"y" jsonschema:"description=The Y coordinate to move to"`
}

type DirectionParams struct {
	Direction string `json:"direction" jsonschema:"description=The direction to face (up, down, left, right),enum=up,enum=down,enum=left,enum=right"`
}

type NameParams struct {
	Name string `json:"name" jsonschema:"description=The name of the item to select"`
}

type TargetTypeParams struct {
	TargetType string `json:"targetType" jsonschema:"description=The type of target to find (e.g. tree, rock, npc, weed)"`
}

type CountParams struct {
	Count int `json:"count" jsonschema:"description=The number of times to repeat the action"`
}

type SlotParams struct {
	Slot int `json:"slot" jsonschema:"description=The inventory slot index to switch to"`
}

type CheatWarpParams struct {
	Location string `json:"location" jsonschema:"description=The location to warp to (e.g., Farm, Town, Mountain, Beach)"`
}

type CheatSetMoneyParams struct {
	Amount int `json:"amount" jsonschema:"description=The amount of money to set"`
}

type CheatAddItemParams struct {
	ItemId string `json:"itemId" jsonschema:"description=The item ID to add, e.g. '(O)465'"`
	Amount int    `json:"amount" jsonschema:"description=The amount of the item to add"`
}

type CheatSetFriendshipParams struct {
	NPCName string `json:"npcName" jsonschema:"description=The name of the NPC"`
	Amount  int    `json:"amount" jsonschema:"description=The amount of friendship points or hearts (if <= 14)"`
}

type CheatMineWarpParams struct {
	Level int `json:"level" jsonschema:"description=The mine level to warp to (1-120 Mines, 121+ Skull Cavern)"`
}

type CheatSpawnOresParams struct {
	Amount int `json:"amount" jsonschema:"description=The amount of ores to spawn"`
}

type CheatTimeSetParams struct {
	Time int `json:"time" jsonschema:"description=The time to set (600=6AM, 1200=noon, 2400=midnight)"`
}

type CheatCompleteQuestParams struct {
	QuestId string `json:"questId" jsonschema:"description=The quest ID to complete, or 'all' for all active quests"`
}

type CheatGiveGiftParams struct {
	NPCName  string `json:"npcName" jsonschema:"description=The name of the NPC to give the gift to"`
	ItemName string `json:"itemName" jsonschema:"description=The name of the item to give"`
}

type CheatHoeAllParams struct {
	Radius int `json:"radius" jsonschema:"description=The radius around the player to hoe (0 means entire location)"`
}

type CheatCutTreesParams struct {
	Radius int `json:"radius" jsonschema:"description=The radius around the player to cut trees (0 means entire location)"`
}

type CheatPlantSeedsParams struct {
	SeedId string `json:"seedId" jsonschema:"description=The ID of the seed to plant, e.g. '472' for parsnips"`
	Radius int    `json:"radius" jsonschema:"description=The radius around the player to plant (0 means entire location)"`
}

type CheatFertilizeAllParams struct {
	FertilizerId string `json:"fertilizerId" jsonschema:"description=The ID of the fertilizer to apply"`
	Radius       int    `json:"radius" jsonschema:"description=The radius around the player to fertilize (0 means entire location)"`
}

type CheatUpgradeBackpackParams struct {
	Size int `json:"size" jsonschema:"description=The size of the backpack (12, 24, or 36)"`
}

type CheatUpgradeToolParams struct {
	ToolName string `json:"toolName" jsonschema:"description=The name of the tool to upgrade (Hoe, Pickaxe, Axe, WateringCan, FishingRod, Trash Can)"`
	Level    int    `json:"level" jsonschema:"description=The level to upgrade to (0=Basic, 1=Copper, 2=Steel, 3=Gold, 4=Iridium)"`
}

type CheatUpgradeAllToolsParams struct {
	Level int `json:"level" jsonschema:"description=The level to upgrade all tools to (0=Basic, 1=Copper, 2=Steel, 3=Gold, 4=Iridium)"`
}

type CheatHoeTilesParams struct {
	Tiles string `json:"tiles" jsonschema:"description=The tiles to hoe, format 'x,y;x,y'"`
	X     int    `json:"x,omitempty" jsonschema:"description=The x coordinate for a single tile"`
	Y     int    `json:"y,omitempty" jsonschema:"description=The y coordinate for a single tile"`
}

type CheatClearTilesParams struct {
	Tiles         string `json:"tiles" jsonschema:"description=The tiles to clear, format 'x,y;x,y'"`
	X             int    `json:"x,omitempty" jsonschema:"description=The x coordinate for a single tile"`
	Y             int    `json:"y,omitempty" jsonschema:"description=The y coordinate for a single tile"`
	ClearObjects  bool   `json:"clearObjects,omitempty" jsonschema:"description=Whether to clear objects (default true)"`
	ClearFeatures bool   `json:"clearFeatures,omitempty" jsonschema:"description=Whether to clear terrain features (default true)"`
	ClearDirt     bool   `json:"clearDirt,omitempty" jsonschema:"description=Whether to clear hoed dirt (default true)"`
}

type CheatHoeCustomPatternParams struct {
	Grid string `json:"grid" jsonschema:"description=The ASCII grid representing the pattern to hoe (# for hoe, . for empty)"`
	X    int    `json:"x,omitempty" jsonschema:"description=The center X coordinate (default player's X)"`
	Y    int    `json:"y,omitempty" jsonschema:"description=The center Y coordinate (default player's Y)"`
}

func runMCPServer(client *GameClient) error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "stardew-mcp",
		Version: "1.0.0",
	}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{
			Tools:   &mcp.ToolCapabilities{},
			Prompts: &mcp.PromptCapabilities{},
		},
	})

	// Register Prompts
	server.AddPrompt(&mcp.Prompt{
		Name: "game_knowledge",
		Description: "Stardew Valley knowledge and instructions",
	}, func(ctx context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Description: "Stardew Valley AI Agent Knowledge Base",
			Messages: []*mcp.PromptMessage{
				{
					Role: "user",
					Content: &mcp.TextContent{Text: gameKnowledge},
				},
			},
		}, nil
	})

	// Helper function for tools to execute GameClient command
	executeCommand := func(action string, params map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
		resp, err := client.SendCommand(action, params)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to execute command: %v", err)}},
				IsError: true,
			}, nil, nil
		}
		if !resp.Success {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Command failed: %s", resp.Message)}},
				IsError: true,
			}, nil, nil
		}

		dataBytes, _ := json.MarshalIndent(resp.Data, "", "  ")
		return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Success: %s\n%s", resp.Message, string(dataBytes))}},
			}, nil, nil
	}

	// Register Tools

	mcp.AddTool(server, &mcp.Tool{
		Name: "move_to",
		Description: "Navigate to specific coordinates using A* pathfinding. Pathfinding might fail if the target is unreachable.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params MoveParams) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("move_to", map[string]interface{}{"x": params.X, "y": params.Y})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_surroundings",
		Description: "Refresh vision to see 61x61 area coordinates and the current game state.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		state := client.GetState()
		if state == nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "Game state not available. Ensure a save is loaded."}},
				IsError: true,
			}, nil, nil
		}
		return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: formatGameStateContext(state)}},
			}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "interact",
		Description: "Interact with the tile directly in front of the player.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("interact", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "use_tool",
		Description: "Use the currently equipped tool once.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("use_tool", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "use_tool_repeat",
		Description: "Execute the currently equipped tool multiple times in succession.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params CountParams) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("use_tool_repeat", map[string]interface{}{"count": params.Count})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "face_direction",
		Description: "Turn character to face direction (up, down, left, right).",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params DirectionParams) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("face_direction", map[string]interface{}{"direction": params.Direction})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "select_item",
		Description: "Find and equip an item from inventory by name.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params NameParams) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("select_item", map[string]interface{}{"name": params.Name})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "switch_tool",
		Description: "Equip an item from a specific inventory slot index.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params SlotParams) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("switch_tool", map[string]interface{}{"slot": params.Slot})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "eat_item",
		Description: "Eat food from an inventory slot to restore energy/health.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params SlotParams) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("eat_item", map[string]interface{}{"slot": params.Slot})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "enter_door",
		Description: "Enter a door or warp point in front of the player.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("enter_door", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "find_best_target",
		Description: "Find the nearest target of a specified type with a walkable approach tile.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params TargetTypeParams) (*mcp.CallToolResult, interface{}, error) {
		state := client.GetState()
		if state == nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "Game state not available"}},
				IsError: true,
			}, nil, nil
		}
		result := findBestTarget(state, params.TargetType)
		return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: result}},
			}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "clear_target",
		Description: "Find and clear the nearest target automatically.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params TargetTypeParams) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("clear_target", map[string]interface{}{"targetType": params.TargetType})
	})

	// --- Cheat Tools ---

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_mode_enable",
		Description: "Enable cheat mode. Required before using other cheat commands.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_mode_enable", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_mode_disable",
		Description: "Disable cheat mode and persistent effects.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_mode_disable", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_warp",
		Description: "Instantly teleport to any location.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params CheatWarpParams) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_warp", map[string]interface{}{"location": params.Location})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_set_money",
		Description: "Set player's gold amount.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params CheatSetMoneyParams) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_set_money", map[string]interface{}{"amount": params.Amount})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_add_item",
		Description: "Add any item to inventory by ID.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params CheatAddItemParams) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_add_item", map[string]interface{}{"itemId": params.ItemId, "amount": params.Amount})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_set_energy",
		Description: "Restore stamina to max.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_set_energy", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_set_health",
		Description: "Restore health to max.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_set_health", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_set_friendship",
		Description: "Instantly set friendship with any NPC.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params CheatSetFriendshipParams) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_set_friendship", map[string]interface{}{"npcName": params.NPCName, "amount": params.Amount})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_max_all_friendships",
		Description: "Max out friendship with ALL NPCs at once.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_max_all_friendships", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_harvest_all",
		Description: "Instantly harvest all ready crops in current location.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_harvest_all", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_water_all",
		Description: "Instantly water all soil in current location.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_water_all", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_grow_crops",
		Description: "Instantly grow all crops to harvest-ready.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_grow_crops", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_clear_debris",
		Description: "Remove all weeds, stones, twigs, grass in current location.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_clear_debris", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_mine_warp",
		Description: "Warp directly to specific mine level.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params CheatMineWarpParams) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_mine_warp", map[string]interface{}{"level": params.Level})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_spawn_ores",
		Description: "Add ores directly to inventory.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params CheatSpawnOresParams) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_spawn_ores", map[string]interface{}{"amount": params.Amount})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_collect_all_forage",
		Description: "Instantly collect all forage items in current location.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_collect_all_forage", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_instant_mine",
		Description: "Mine ALL ore nodes in current mine level instantly.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_instant_mine", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_time_set",
		Description: "Set the game time.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params CheatTimeSetParams) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_time_set", map[string]interface{}{"time": params.Time})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_time_freeze",
		Description: "Toggle time freeze on/off.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_time_freeze", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_infinite_energy",
		Description: "Toggle infinite stamina on/off.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_infinite_energy", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_unlock_recipes",
		Description: "Unlock ALL crafting and cooking recipes.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_unlock_recipes", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_pet_all_animals",
		Description: "Pet ALL farm animals instantly.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_pet_all_animals", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_complete_quest",
		Description: "Complete active quests instantly.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params CheatCompleteQuestParams) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_complete_quest", map[string]interface{}{"questId": params.QuestId})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_give_gift",
		Description: "Give a gift to an NPC instantly.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params CheatGiveGiftParams) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_give_gift", map[string]interface{}{"npcName": params.NPCName, "itemName": params.ItemName})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_hoe_all",
		Description: "Instantly hoe/till all diggable tiles in current location.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params CheatHoeAllParams) (*mcp.CallToolResult, interface{}, error) {
		p := map[string]interface{}{}
		if params.Radius > 0 { p["radius"] = params.Radius }
		return executeCommand("cheat_hoe_all", p)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_cut_trees",
		Description: "Instantly cut/chop ALL trees in current location.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params CheatCutTreesParams) (*mcp.CallToolResult, interface{}, error) {
		p := map[string]interface{}{}
		if params.Radius > 0 { p["radius"] = params.Radius }
		return executeCommand("cheat_cut_trees", p)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_mine_rocks",
		Description: "Instantly mine ALL rocks/stones/boulders in current location.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_mine_rocks", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_dig_artifacts",
		Description: "Instantly dig up ALL artifact spots in current location.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_dig_artifacts", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_plant_seeds",
		Description: "Instantly plant seeds on ALL empty hoed tiles.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params CheatPlantSeedsParams) (*mcp.CallToolResult, interface{}, error) {
		p := map[string]interface{}{"seedId": params.SeedId}
		if params.Radius > 0 { p["radius"] = params.Radius }
		return executeCommand("cheat_plant_seeds", p)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_fertilize_all",
		Description: "Apply fertilizer to ALL hoed tiles.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params CheatFertilizeAllParams) (*mcp.CallToolResult, interface{}, error) {
		p := map[string]interface{}{}
		if params.FertilizerId != "" { p["fertilizerId"] = params.FertilizerId }
		if params.Radius > 0 { p["radius"] = params.Radius }
		return executeCommand("cheat_fertilize_all", p)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_upgrade_backpack",
		Description: "Upgrade backpack to larger size.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params CheatUpgradeBackpackParams) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_upgrade_backpack", map[string]interface{}{"size": params.Size})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_upgrade_tool",
		Description: "Upgrade a specific tool to higher level.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params CheatUpgradeToolParams) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_upgrade_tool", map[string]interface{}{"toolName": params.ToolName, "level": params.Level})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_upgrade_all_tools",
		Description: "Upgrade ALL tools to specified level.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params CheatUpgradeAllToolsParams) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_upgrade_all_tools", map[string]interface{}{"level": params.Level})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_unlock_all",
		Description: "UNLOCK EVERYTHING: Max backpack, tools, recipes, skills, etc.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params struct{}) (*mcp.CallToolResult, interface{}, error) {
		return executeCommand("cheat_unlock_all", nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_hoe_tiles",
		Description: "Hoe SPECIFIC tiles by coordinates.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params CheatHoeTilesParams) (*mcp.CallToolResult, interface{}, error) {
		p := map[string]interface{}{}
		if params.Tiles != "" { p["tiles"] = params.Tiles }
		if params.X != 0 || params.Y != 0 {
			p["x"] = params.X
			p["y"] = params.Y
		}
		return executeCommand("cheat_hoe_tiles", p)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_clear_tiles",
		Description: "Clear SPECIFIC tiles (objects, terrain, hoed dirt).",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params CheatClearTilesParams) (*mcp.CallToolResult, interface{}, error) {
		p := map[string]interface{}{}
		if params.Tiles != "" { p["tiles"] = params.Tiles }
		if params.X != 0 || params.Y != 0 {
			p["x"] = params.X
			p["y"] = params.Y
		}
		p["clearObjects"] = params.ClearObjects
		p["clearFeatures"] = params.ClearFeatures
		p["clearDirt"] = params.ClearDirt
		return executeCommand("cheat_clear_tiles", p)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cheat_hoe_custom_pattern",
		Description: "Draw ANY shape by designing it yourself as an ASCII grid!",
	}, func(ctx context.Context, request *mcp.CallToolRequest, params CheatHoeCustomPatternParams) (*mcp.CallToolResult, interface{}, error) {
		p := map[string]interface{}{"grid": params.Grid}
		if params.X != 0 || params.Y != 0 {
			p["x"] = params.X
			p["y"] = params.Y
		}
		return executeCommand("cheat_hoe_custom_pattern", p)
	})


	// Setup standard input/output transport for MCP
	transport := &mcp.StdioTransport{}

	ctx := context.Background()
	err := server.Run(ctx, transport)



	return err
}

const gameKnowledge = `# Stardew Valley AI Agent: High-Intelligence Protocol

## CORE LOGIC: PLANNING VS EXECUTION

**1. LONG-TERM PLANNING**: When you receive a goal, think about the sequence of areas you need to clear.
**2. SPATIAL AWARENESS**: Check your surroundings (61x61 map) to find the nearest cluster of targets.
**3. EXECUTION**:
   - Move to a tile NEXT to the target.
   - FACE the target.
   - **CONFIRM** the target is in the "Tile in front" data.
   - Use the **Lowest-Energy** tool required.

## ASCII MAP LEGEND (61x61 Vision)
- @ : YOU (The Player)
- . : BLANK GROUND (Walkable)
- # : WALL / BUILDING / IMPASSABLE (Blocked)
- ~ : WATER (Blocked)
- T : TREE / BUSH (Blocked - Chop with AXE to clear)
- O : OBJECT / STONE / TWIG / WEED (Blocked - Break with Pickaxe/Axe/Scythe)
- C : CROP (Blocked - Do not trample if possible)
- H : HOE DIRT (Walkable)
- " : GRASS (Walkable - Cut with Scythe for 0 energy)
- > : WARP / DOOR / ENTRANCE
- ; : ARTIFACT SPOT (Hoe it!)
- ! : NPC
- M : MONSTER

## SPATIAL COORDINATION & PRECISION

- **Coordinates**: X is horizontal (0=left), Y is vertical (0=top).
- **Tool Range**: You can ONLY hit the tile directly in front of you.
- **Distance Rule**: You must be exactly 1 tile away from your target.
  - To hit Target (10, 10): Stand at (9, 10) and face "right", OR at (11, 10) and face "left", etc.
  - **DO NOT** stand on the same tile as the target (10, 10).
  - **DO NOT** use move_to to go TO a target coordinate typed 'O' or 'T'. Use move_to to go to a '.' tile NEXT to it.

## TOOL EFFICIENCY

- **SCYTHE**: Use for weeds/grass. It costs **0 ENERGY**. Highly efficient for cleanup.
- **AXE**: Use for wood/twigs/stumps. Costs energy.
- **PICKAXE**: Use for stones/ore. Costs energy.
- **VERIFICATION**: After using a tool, check if the objectName/terrainType in "Tile in front" has changed to "." (walkable ground). If not, your action failed—do not keep moving, FIX it.

## INTELLIGENCE & AUTO-CORRECTION

- **No Path Found?**: The tile you clicked is blocked. Try moving to a tile 1-step away from it.
- **IsMoving Error?**: Movement is now BLOCKING. If a move tool finishes, you are at your destination. Do not issue 10 move commands in a row; wait for each.
- **Cleaning Goals**: Don't just swing randomly. Find a target, move to it, clear it, move to the next.

## SURVIVAL & NIGHT

- **2:00 AM** is a hard game-over. You MUST be in bed by **1:00 AM**.
- Farmhouse Entrance is usually around (60, 15) on the standard farm layout, but check surroundings for "FarmHouse" warp.

## CHEAT MODE (Optional Power Tools)

Cheat mode provides instant, god-mode capabilities. **You must call cheat_mode_enable first** before any cheat commands work.
`

type Target struct {
	X            int
	Y            int
	Name         string
	Type         string
	RequiredTool string
	HitsRequired int
	Distance     int
}

func findBestTarget(state *GameState, targetType string) string {
	px, py := int(state.Player.X), int(state.Player.Y)
	var targets []Target

	targetTypeLower := strings.ToLower(targetType)

	switch targetTypeLower {
	case "weed", "weeds", "grass", "stone", "stones", "rock", "rocks", "twig", "twigs", "stick", "sticks", "object", "objects":
		targetTypeLower = "debris"
	case "trees", "wood", "log", "logs":
		targetTypeLower = "tree"
	case "crops", "harvest", "vegetables", "fruit":
		targetTypeLower = "crop"
	case "npcs", "villager", "villagers", "person", "people":
		targetTypeLower = "npc"
	case "warps", "doors", "exit", "entrance", "portal":
		targetTypeLower = "warp"
	case "all", "everything", "anything":
		targetTypeLower = "any"
	}

	if targetTypeLower == "debris" || targetTypeLower == "any" {
		for _, obj := range state.Surroundings.NearbyObjects {
			if !obj.IsPassable {
				hitsRequired := obj.HitsRequired
				if hitsRequired == 0 {
					hitsRequired = 1
				}
				t := Target{
					X:            obj.X,
					Y:            obj.Y,
					Name:         obj.DisplayName,
					Type:         "object",
					RequiredTool: obj.RequiredTool,
					HitsRequired: hitsRequired,
					Distance:     abs(obj.X-px) + abs(obj.Y-py),
				}
				if t.RequiredTool == "Scythe" {
					t.Distance -= 100
				}
				targets = append(targets, t)
			}
		}
	}

	if targetTypeLower == "tree" || targetTypeLower == "any" {
		for _, tf := range state.Surroundings.NearbyTerrainFeatures {
			if !tf.IsPassable && (tf.Type == "tree" || tf.Type == "fruit_tree") {
				hitsRequired := tf.HitsRequired
				if hitsRequired == 0 {
					hitsRequired = 10
				}
				targets = append(targets, Target{
					X:            tf.X,
					Y:            tf.Y,
					Name:         tf.Type,
					Type:         "terrain",
					RequiredTool: tf.RequiredTool,
					HitsRequired: hitsRequired,
					Distance:     abs(tf.X-px) + abs(tf.Y-py),
				})
			}
		}
	}

	if targetTypeLower == "crop" || targetTypeLower == "any" {
		for _, tf := range state.Surroundings.NearbyTerrainFeatures {
			if tf.HasCrop && tf.IsReadyForHarvest {
				targets = append(targets, Target{
					X:            tf.X,
					Y:            tf.Y,
					Name:         tf.CropName,
					Type:         "crop",
					RequiredTool: "Scythe",
					HitsRequired: 1,
					Distance:     abs(tf.X-px) + abs(tf.Y-py),
				})
			}
		}
	}

	if targetTypeLower == "npc" || targetTypeLower == "any" {
		for _, npc := range state.Surroundings.NearbyNPCs {
			targets = append(targets, Target{
				X:            npc.X,
				Y:            npc.Y,
				Name:         npc.DisplayName,
				Type:         "npc",
				RequiredTool: "",
				HitsRequired: 0,
				Distance:     abs(npc.X-px) + abs(npc.Y-py),
			})
		}
	}

	if targetTypeLower == "warp" || targetTypeLower == "door" || targetTypeLower == "any" {
		for _, warp := range state.Surroundings.WarpPoints {
			targets = append(targets, Target{
				X:            warp.X,
				Y:            warp.Y,
				Name:         warp.TargetLocation,
				Type:         "warp",
				RequiredTool: "",
				HitsRequired: 0,
				Distance:     abs(warp.X-px) + abs(warp.Y-py),
			})
		}
		for _, bldg := range state.Surroundings.NearbyBuildings {
			if bldg.Type == "FarmHouse" || bldg.Type == "Cabin" {
				targets = append(targets, Target{
					X:            bldg.DoorX,
					Y:            bldg.DoorY,
					Name:         bldg.Type + " door",
					Type:         "door",
					RequiredTool: "",
					HitsRequired: 0,
					Distance:     abs(bldg.DoorX-px) + abs(bldg.DoorY-py),
				})
			}
		}
	}

	if len(targets) == 0 {
		return fmt.Sprintf("No targets of type '%s' found nearby.", targetType)
	}

	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Distance < targets[j].Distance
	})

	for _, target := range targets {
		adjacents := []struct {
			x, y      int
			direction string
		}{
			{target.X - 1, target.Y, "right"},
			{target.X + 1, target.Y, "left"},
			{target.X, target.Y - 1, "down"},
			{target.X, target.Y + 1, "up"},
		}

		for _, adj := range adjacents {
			if isTileWalkable(state, adj.x, adj.y) {
				toolName := strings.ToLower(target.RequiredTool)
				if toolName == "" {
					toolName = "none"
				}

				finalAction := "use_tool"
				if target.HitsRequired > 1 {
					finalAction = fmt.Sprintf("use_tool_repeat with count=%d", target.HitsRequired)
				} else if target.HitsRequired == 0 {
					finalAction = "interact"
				}

				return fmt.Sprintf(`TARGET: %s at (%d,%d) - Tool: %s - Hits: %d

NOW DO THESE IN ORDER (do NOT call find_best_target again):
Step 1: select_item name="%s"
Step 2: move_to x=%d y=%d
Step 3: face_direction direction="%s"
Step 4: %s`,
					target.Name, target.X, target.Y, target.RequiredTool, target.HitsRequired,
					toolName,
					adj.x, adj.y,
					adj.direction,
					finalAction)
			}
		}
	}

	return fmt.Sprintf("Found %d targets but none have accessible approach tiles. Try moving to a different area.", len(targets))
}

func isTileWalkable(state *GameState, x, y int) bool {
	radius := 30
	px, py := int(state.Player.X), int(state.Player.Y)
	rx := x - px
	ry := y - py

	if ry < -radius || ry > radius || rx < -radius || rx > radius {
		return false
	}

	if state.Surroundings.AsciiMap == "" {
		return true
	}

	lines := strings.Split(state.Surroundings.AsciiMap, "\n")
	gridY := radius + ry
	gridX := radius + rx

	if gridY < 0 || gridY >= len(lines) {
		return false
	}
	line := lines[gridY]
	if gridX < 0 || gridX >= len(line) {
		return false
	}

	char := line[gridX]
	switch char {
	case '.', '>', 'H', '"', ';', '@':
		return true
	default:
		return false
	}
}

func formatGameStateContext(state *GameState) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Equipped Tool: %s (slot %d)\n", state.Player.CurrentTool, state.Player.CurrentToolIndex))

	tif := state.Surroundings.TileInFront
	sb.WriteString(fmt.Sprintf("\n--- TILE IN FRONT (facing %s) ---\n", state.Player.FacingDirectionName))
	sb.WriteString(fmt.Sprintf("Position: (%d, %d)\n", tif.X, tif.Y))
	if tif.ObjectName != "" {
		toolInfo := ""
		if tif.RequiredTool != "" {
			toolInfo = fmt.Sprintf(" [Use: %s]", tif.RequiredTool)
		}
		sb.WriteString(fmt.Sprintf("Object: %s%s\n", tif.ObjectName, toolInfo))
	}
	if tif.TerrainType != "" {
		toolInfo := ""
		if tif.RequiredTool != "" {
			toolInfo = fmt.Sprintf(" [Use: %s]", tif.RequiredTool)
		}
		sb.WriteString(fmt.Sprintf("Terrain: %s%s\n", tif.TerrainType, toolInfo))
	}
	if tif.NPCName != "" {
		sb.WriteString(fmt.Sprintf("NPC: %s\n", tif.NPCName))
	}
	if tif.IsPassable {
		sb.WriteString("Status: WALKABLE\n")
	} else {
		sb.WriteString("Status: BLOCKED\n")
	}

	if len(state.Surroundings.NearbyBuildings) > 0 {
		sb.WriteString("\n--- BUILDINGS ---\n")
		for _, b := range state.Surroundings.NearbyBuildings {
			sb.WriteString(fmt.Sprintf("- %s: Door at (%d, %d)\n", b.Type, b.DoorX, b.DoorY))
		}
	}

	if len(state.Surroundings.WarpPoints) > 0 {
		sb.WriteString("\n--- WARPS/DOORS ---\n")
		for _, w := range state.Surroundings.WarpPoints {
			sb.WriteString(fmt.Sprintf("- (%d, %d) -> %s\n", w.X, w.Y, w.TargetLocation))
		}
	}

	debrisCount := 0
	scytheTargets := 0
	for _, obj := range state.Surroundings.NearbyObjects {
		if !obj.IsPassable {
			debrisCount++
			if obj.RequiredTool == "Scythe" {
				scytheTargets++
			}
		}
	}
	treeCount := 0
	for _, tf := range state.Surroundings.NearbyTerrainFeatures {
		if tf.Type == "tree" || tf.Type == "fruit_tree" {
			treeCount++
		}
	}

	sb.WriteString("\n--- TARGET SUMMARY ---\n")
	sb.WriteString(fmt.Sprintf("Debris (stones/twigs/weeds): %d (%d use Scythe=0 energy)\n", debrisCount, scytheTargets))
	sb.WriteString(fmt.Sprintf("Trees: %d\n", treeCount))
	sb.WriteString(fmt.Sprintf("NPCs: %d\n", len(state.Surroundings.NearbyNPCs)))

	sb.WriteString("\n--- NEAREST TARGETS (use find_best_target for full list) ---\n")
	shown := 0
	for _, obj := range state.Surroundings.NearbyObjects {
		if shown >= 5 {
			break
		}
		if !obj.IsPassable {
			tool := obj.RequiredTool
			if tool == "" {
				tool = "unknown"
			}
			sb.WriteString(fmt.Sprintf("- %s at (%d, %d) [%s]\n", obj.DisplayName, obj.X, obj.Y, tool))
			shown++
		}
	}

	sb.WriteString("\n--- INVENTORY (food items) ---\n")
	hasFood := false
	for _, item := range state.Player.Inventory {
		if item.Category == "Cooking" || strings.Contains(strings.ToLower(item.Name), "salad") ||
			strings.Contains(strings.ToLower(item.Name), "egg") || strings.Contains(strings.ToLower(item.Name), "milk") {
			sb.WriteString(fmt.Sprintf("- Slot %d: %s (x%d)\n", item.Slot, item.DisplayName, item.Stack))
			hasFood = true
		}
	}
	if !hasFood {
		sb.WriteString("No food items found.\n")
	}

	if state.Surroundings.AsciiMap != "" {
		sb.WriteString("\n--- ASCII MAP (center 21x21 of 61x61) ---\n")
		lines := strings.Split(state.Surroundings.AsciiMap, "\n")
		center := 30
		viewRadius := 10
		for y := center - viewRadius; y <= center+viewRadius && y < len(lines); y++ {
			if y >= 0 && y < len(lines) {
				line := lines[y]
				start := center - viewRadius
				end := center + viewRadius + 1
				if start >= 0 && end <= len(line) {
					sb.WriteString(line[start:end] + "\n")
				} else if len(line) > 0 {
					sb.WriteString(line + "\n")
				}
			}
		}
	}

	return sb.String()
}

// abs returns the absolute value of x
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
