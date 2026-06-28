package Orchestration

// Standardized MCP Schemas
type MCPSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required"`
}
