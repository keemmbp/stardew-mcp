using System;
using System.Reflection;
using Microsoft.Xna.Framework;
using StardewModdingAPI;
using StardewValley;

namespace StardewMCP;

public sealed class GameActionDriver
{
    private readonly IMonitor _monitor;

    public GameActionDriver(IMonitor monitor)
    {
        _monitor = monitor;
    }

    public void Move(int facingDirection)
    {
        var player = Game1.player;
        player.FacingDirection = facingDirection;
        player.setMoving((byte)facingDirection);
    }

    public void Face(int facingDirection)
    {
        var player = Game1.player;
        player.FacingDirection = facingDirection;
        player.FarmerSprite.StopAnimation();
    }

    public bool Interact(out string message)
    {
        SetCursorToFacingTile();
        return InvokeGame1Button("pressActionButton", out message);
    }

    public bool UseTool(out string message)
    {
        if (Game1.player.CurrentTool == null)
        {
            message = "No tool equipped";
            return false;
        }

        SetCursorToFacingTile();
        return InvokeGame1Button("pressUseToolButton", out message);
    }

    public bool PressButton(SButton button)
    {
        if (MatchesButton(button, Game1.options.actionButton, SButton.MouseRight))
            return Interact(out _);

        if (MatchesButton(button, Game1.options.useToolButton, SButton.MouseLeft))
            return UseTool(out _);

        if (MatchesButton(button, Game1.options.moveUpButton, SButton.W))
        {
            Move(0);
            return true;
        }

        if (MatchesButton(button, Game1.options.moveRightButton, SButton.D))
        {
            Move(1);
            return true;
        }

        if (MatchesButton(button, Game1.options.moveDownButton, SButton.S))
        {
            Move(2);
            return true;
        }

        if (MatchesButton(button, Game1.options.moveLeftButton, SButton.A))
        {
            Move(3);
            return true;
        }

        _monitor.Log($"Unsupported synthetic button: {button}", LogLevel.Warn);
        return false;
    }

    private static bool MatchesButton(SButton button, InputButton[] configuredButtons, SButton fallback)
    {
        if (configuredButtons.Length == 0)
            return button == fallback;

        foreach (var configured in configuredButtons)
        {
            if (configured.ToSButton() == button)
                return true;
        }

        return false;
    }

    private bool InvokeGame1Button(string methodName, out string message)
    {
        try
        {
            var method = typeof(Game1).GetMethod(methodName, BindingFlags.Static | BindingFlags.Public | BindingFlags.NonPublic, null, Type.EmptyTypes, null);
            if (method == null)
            {
                message = $"Stardew Valley API method Game1.{methodName}() was not found.";
                return false;
            }

            method.Invoke(null, null);
            message = $"Invoked Game1.{methodName}()";
            return true;
        }
        catch (TargetInvocationException ex)
        {
            message = ex.InnerException?.Message ?? ex.Message;
            _monitor.Log($"Game action {methodName} failed: {message}", LogLevel.Warn);
            return false;
        }
        catch (Exception ex)
        {
            message = ex.Message;
            _monitor.Log($"Game action {methodName} failed: {message}", LogLevel.Warn);
            return false;
        }
    }

    private static void SetCursorToFacingTile()
    {
        var player = Game1.player;
        int tileX = (int)player.Tile.X;
        int tileY = (int)player.Tile.Y;

        switch (player.FacingDirection)
        {
            case 0: tileY--; break;
            case 1: tileX++; break;
            case 2: tileY++; break;
            case 3: tileX--; break;
        }

        Game1.currentCursorTile = new Vector2(tileX, tileY);
        Game1.lastCursorMotionWasMouse = false;
        Game1.setMousePosition((tileX * 64) + 32 - Game1.viewport.X, (tileY * 64) + 32 - Game1.viewport.Y);
    }
}
