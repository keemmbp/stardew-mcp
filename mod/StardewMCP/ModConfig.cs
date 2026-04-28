namespace StardewMCP;

public sealed class ModConfig
{
    public int WebSocketPort { get; set; } = 8765;
    public bool EnableCheats { get; set; } = false;
    public int StateBroadcastIntervalSeconds { get; set; } = 1;
}
