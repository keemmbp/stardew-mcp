package main

import (
	"context"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type MyInput struct {
	Foo string `json:"foo"`
}

func myHandler(ctx context.Context, req *mcp.CallToolRequest, input MyInput) (*mcp.CallToolResult, interface{}, error) {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: input.Foo}}}, nil, nil
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "test"}, myHandler)

	transport := &mcp.StdioTransport{}
	_ = transport
	fmt.Println("Compiled successfully")
}
