using System;
using System.Collections.Generic;
using Microsoft.Xna.Framework;
using StardewValley;
using xTile.Dimensions;

namespace StardewMCP;

/// <summary>A* pathfinding for navigating Stardew Valley maps.</summary>
public class Pathfinder
{
    private const int MaxIterations = 50000; // Increased for large maps

    /// <summary>Find a path from start to goal using A* algorithm.</summary>
    /// <param name="location">The game location to pathfind in.</param>
    /// <param name="start">Starting tile position.</param>
    /// <param name="goal">Target tile position.</param>
    /// <returns>List of tile positions forming the path, or null if no path found.</returns>
    public List<Vector2>? FindPath(GameLocation location, Vector2 start, Vector2 goal)
    {
        if (location == null)
            return null;

        // Pre-cache blocked tiles for this pathfinding request
        var blockedTiles = GetBlockedTiles(location);

        // Quick check: if goal is not passable, no path possible
        if (!IsTileWalkable(location, (int)goal.X, (int)goal.Y, blockedTiles))
            return null;

        // If already at goal, return empty path
        if (start == goal)
            return new List<Vector2>();

        var openSet = new PriorityQueue<Vector2, float>();
        var cameFrom = new Dictionary<Vector2, Vector2>();
        var gScore = new Dictionary<Vector2, float> { [start] = 0 };
        var fScore = new Dictionary<Vector2, float> { [start] = Heuristic(start, goal) };

        openSet.Enqueue(start, fScore[start]);
        var inOpenSet = new HashSet<Vector2> { start };

        int iterations = 0;

        while (openSet.Count > 0 && iterations < MaxIterations)
        {
            iterations++;

            var current = openSet.Dequeue();
            inOpenSet.Remove(current);

            if (current == goal)
            {
                return ReconstructPath(cameFrom, current);
            }

            foreach (var neighbor in GetNeighbors(current))
            {
                int nx = (int)neighbor.X;
                int ny = (int)neighbor.Y;

                // Skip if not walkable
                if (!IsTileWalkable(location, nx, ny, blockedTiles))
                    continue;

                float tentativeGScore = gScore[current] + 1; // Cost of 1 per tile

                if (!gScore.ContainsKey(neighbor) || tentativeGScore < gScore[neighbor])
                {
                    cameFrom[neighbor] = current;
                    gScore[neighbor] = tentativeGScore;
                    fScore[neighbor] = tentativeGScore + Heuristic(neighbor, goal);

                    if (!inOpenSet.Contains(neighbor))
                    {
                        openSet.Enqueue(neighbor, fScore[neighbor]);
                        inOpenSet.Add(neighbor);
                    }
                }
            }
        }

        // No path found
        return null;
    }

    /// <summary>Get a set of tiles occupied by large features (resource clumps, buildings, furniture).</summary>
    private HashSet<Vector2> GetBlockedTiles(GameLocation location)
    {
        var blocked = new HashSet<Vector2>();

        // Cache resource clumps
        foreach (var clump in location.resourceClumps)
        {
            for (int dx = 0; dx < clump.width.Value; dx++)
            {
                for (int dy = 0; dy < clump.height.Value; dy++)
                {
                    int tx = (int)clump.Tile.X + dx;
                    int ty = (int)clump.Tile.Y + dy;
                    if (clump.occupiesTile(tx, ty))
                        blocked.Add(new Vector2(tx, ty));
                }
            }
        }

        // Cache buildings
        if (location is Farm farm)
        {
            foreach (var building in farm.buildings)
            {
                for (int dx = 0; dx < building.tilesWide.Value; dx++)
                {
                    for (int dy = 0; dy < building.tilesHigh.Value; dy++)
                    {
                        int tx = building.tileX.Value + dx;
                        int ty = building.tileY.Value + dy;
                        var tileVector = new Vector2(tx, ty);
                        if (building.occupiesTile(tileVector))
                            blocked.Add(tileVector);
                    }
                }
            }
        }

        // Cache furniture
        foreach (var furniture in location.furniture)
        {
            // Furniture can be rotated and have complex bounding boxes
            var bbox = furniture.boundingBox.Value;
            int minX = bbox.X / 64;
            int maxX = (bbox.X + bbox.Width) / 64;
            int minY = bbox.Y / 64;
            int maxY = (bbox.Y + bbox.Height) / 64;

            for (int tx = minX; tx <= maxX; tx++)
            {
                for (int ty = minY; ty <= maxY; ty++)
                {
                    if (furniture.TileLocation == new Vector2(tx, ty) ||
                        furniture.boundingBox.Value.Contains(tx * 64 + 32, ty * 64 + 32))
                    {
                        blocked.Add(new Vector2(tx, ty));
                    }
                }
            }
        }

        return blocked;
    }

    /// <summary>Check if a tile is walkable for the player.</summary>
    private bool IsTileWalkable(GameLocation location, int x, int y, HashSet<Vector2> blockedTiles)
    {
        // Check map bounds
        if (x < 0 || y < 0)
            return false;

        var map = location.Map;
        if (map == null || map.Layers.Count == 0)
            return false;

        var layer = map.Layers[0];
        if (x >= layer.LayerWidth || y >= layer.LayerHeight)
            return false;

        // Use game's built-in passability check
        var tileLocation = new Location(x, y);

        // Check if tile itself is passable (map layer check)
        if (!location.isTilePassable(tileLocation, Game1.viewport))
            return false;

        // Check for objects blocking the tile
        var tileVector = new Vector2(x, y);
        if (location.Objects.ContainsKey(tileVector))
        {
            var obj = location.Objects[tileVector];
            if (!obj.isPassable())
                return false;
        }

        // Check for terrain features (trees, etc.)
        if (location.terrainFeatures.ContainsKey(tileVector))
        {
            var feature = location.terrainFeatures[tileVector];
            if (!feature.isPassable())
                return false;
        }

        // Check pre-cached blocked tiles (resource clumps, buildings, furniture)
        if (blockedTiles.Contains(tileVector))
            return false;

        // Check water tiles (player can't walk on water)
        if (location.isWaterTile(x, y) && !location.isTilePassable(tileLocation, Game1.viewport))
            return false;

        return true;
    }

    /// <summary>Get the 4-directional neighbors of a tile.</summary>
    private IEnumerable<Vector2> GetNeighbors(Vector2 tile)
    {
        yield return new Vector2(tile.X + 1, tile.Y);     // Right
        yield return new Vector2(tile.X - 1, tile.Y);     // Left
        yield return new Vector2(tile.X, tile.Y + 1);     // Down
        yield return new Vector2(tile.X, tile.Y - 1);     // Up
    }

    /// <summary>Manhattan distance heuristic.</summary>
    private float Heuristic(Vector2 a, Vector2 b)
    {
        return Math.Abs(a.X - b.X) + Math.Abs(a.Y - b.Y);
    }

    /// <summary>Reconstruct the path from start to goal.</summary>
    private List<Vector2> ReconstructPath(Dictionary<Vector2, Vector2> cameFrom, Vector2 current)
    {
        var path = new List<Vector2> { current };

        while (cameFrom.ContainsKey(current))
        {
            current = cameFrom[current];
            path.Add(current);
        }

        // Reverse to get start-to-goal order and remove starting position (we're already there)
        path.Reverse();
        if (path.Count > 0)
            path.RemoveAt(0);

        return path;
    }
}
