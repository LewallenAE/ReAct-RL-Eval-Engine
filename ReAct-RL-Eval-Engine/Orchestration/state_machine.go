package Orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)
// Standardized MCP Schemas
type MCPSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required"`
}

type Tool struct {
	Name string `json:"name"`
	Description string `json:"description"`
	InputSchema MCPSchema `json:"inputSchema"`
}

// Production interfaces
// Using interfaces we can mock the DM and LLM for deterministic unit testing.
type StateStore interface {
	LoadState(ctx context.Context, sessionID string) ([]Message, error)
	SaveState(ctx context.Context, sessionID string, messages []Message) error
}

type LLMClient interface {
	// calls the Python LLM Metal Layer
	Generate(ctx context.Context, messages []Message, tools []Tool) (*Message, error)
}

type ToolGateway interface {
	GetAvailableTools() []Tool
	ExecuteTool(ctx context.Context, name string, args json.RawMessage) (string, error)
}

// Orchestration Engine
type ReActEngine struct {
	store StateStore
	llm LLMClient
	gateway ToolGateway
	maxSteps int
	stepTimeout time.Duration
}

func newReactEngine(s StateStore, l LLMClient, g ToolGateway) *ReActEngine {
	return &ReActEngine{
		store: s,
		llm: l,
		gateway: g,
		maxSteps: 5, // Circuit breaker: prevents an infinitely running agent
		stepTimeout: 10 * time.Second, // Hard timeout for any single action.
	}
}


func (e *ReActEngine) safeExecute(baseCtx context.Context, toolName, string, args json.RawMessage) string {
	ctx, cancel := context.timeout(baseCtx, e.stepTimeout)
	defer cancel()

	resultChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	go func() {
		res, err := e.gateway.ExecuteTool(ctx, toolName, args)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- res
	}()

	select {
	case res := <-resultChan:
		return res
	case err := <-errorChan:
		return fmt.Sprintf("SYSTEM_ERROR: Tool failed with error: %v", err)
	case <-ctx.Done():
		return fmt.Sprintf("System_ERROR: Tool execution timed out after %v", e.stepTimeout)
	}
}