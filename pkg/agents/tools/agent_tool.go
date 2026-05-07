package tools

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/hastekit/hastekit-sdk-go/pkg/agents"
	"github.com/hastekit/hastekit-sdk-go/pkg/gateway/llm/constants"
	"github.com/hastekit/hastekit-sdk-go/pkg/gateway/llm/responses"
	"github.com/hastekit/hastekit-sdk-go/pkg/utils"
)

type SubAgentContextMode string

const (
	SubAgentContextModeNone     SubAgentContextMode = "None"
	SubAgentContextModeIsolated SubAgentContextMode = "Isolated"
)

type AgentTool struct {
	*agents.BaseTool
	agent       *agents.Agent
	contextMode SubAgentContextMode
}

type agentToolArgument struct {
	Message  string `json:"message"`
	ThreadID string `json:"thread_id"`
}

func NewAgentTool(name string, description string, agent *agents.Agent, contextMode SubAgentContextMode) *AgentTool {
	toolUnion := responses.ToolUnion{
		OfFunction: &responses.FunctionTool{
			Name:        name,
			Description: utils.Ptr(description),
		},
	}

	switch contextMode {
	case SubAgentContextModeNone:
		toolUnion.OfFunction.Parameters = map[string]any{
			"type":     "object",
			"required": []string{"message"},
			"properties": map[string]any{
				"message": map[string]any{
					"type":        "string",
					"description": "Message for the agent",
				},
				"thread_id": map[string]any{
					"type":        "string",
					"description": "Thread ID for the agent conversation. Leave empty to start a new conversation.",
				},
			},
			"additionalProperties": false,
		}
	case SubAgentContextModeIsolated:
		toolUnion.OfFunction.Parameters = map[string]any{
			"type":     "object",
			"required": []string{"message"},
			"properties": map[string]any{
				"message": map[string]any{
					"type":        "string",
					"description": "Message for the agent",
				},
			},
			"additionalProperties": false,
		}
	}

	return &AgentTool{
		BaseTool: &agents.BaseTool{
			ToolUnion: toolUnion,
		},
		agent:       agent,
		contextMode: contextMode,
	}
}

func (t *AgentTool) Execute(ctx context.Context, params *agents.ToolCall) (*agents.ToolCallResponse, error) {
	namespace := params.Namespace + "/" + params.Name

	var agentArgs agentToolArgument
	if err := sonic.Unmarshal([]byte(params.Arguments), &agentArgs); err != nil {
		return nil, err
	}

	var threadId string
	if t.contextMode == SubAgentContextModeIsolated {
		subAgentThreadId, exists := params.State[t.getSubAgentThreadIdStateKey()]
		if exists {
			threadId = subAgentThreadId
		} else {
			threadId = uuid.NewString()
		}
	} else {
		if agentArgs.ThreadID != "" {
			threadId = agentArgs.ThreadID
		} else {
			threadId = uuid.NewString()
		}
	}

	handle, err := t.agent.Execute(ctx, &agents.AgentInput{
		Namespace: namespace,
		ThreadID:  threadId,
		Messages: []responses.InputMessageUnion{
			{
				OfEasyInput: &responses.EasyMessage{
					Role:    constants.RoleUser,
					Content: responses.EasyInputContentUnion{OfString: &agentArgs.Message},
				},
			},
		},
		SessionID: params.SessionID, // Using conversation id as the shared session id
	})
	if err != nil {
		return nil, err
	}

	// The parent agent reports tool output, not chunks. Result drains
	// the sub-agent's chunk stream and returns the aggregated output.
	result, err := handle.Result()
	if err != nil {
		return nil, err
	}

	data := ""
	for _, out := range result.Output {
		if out.OfOutputMessage != nil {
			for _, content := range *out.OfOutputMessage.Content {
				if content.OfOutputText != nil {
					data += content.OfOutputText.Text
				}
			}
		}

		if out.OfEasyInput != nil {
			if out.OfEasyInput.Content.OfString != nil {
				data += *out.OfEasyInput.Content.OfString
			}

			if out.OfEasyInput.Content.OfInputMessageList != nil {
				for _, message := range out.OfEasyInput.Content.OfInputMessageList {
					if message.OfOutputText != nil {
						data += message.OfOutputText.Text
					}
				}
			}
		}
	}

	if t.contextMode == SubAgentContextModeNone {
		data = data + fmt.Sprintf("\n---\nThread ID: %s", threadId)
	}

	return &agents.ToolCallResponse{
		FunctionCallOutputMessage: &responses.FunctionCallOutputMessage{
			ID:     params.ID,
			CallID: params.CallID,
			Output: responses.FunctionCallOutputContentUnion{
				OfString: utils.Ptr(data),
			},
		},
		StateUpdates: map[string]string{
			t.getSubAgentThreadIdStateKey(): threadId,
		},
	}, nil
}

func (t *AgentTool) getSubAgentThreadIdStateKey() string {
	return fmt.Sprintf("sub_agent_thread_id/%s", t.agent.Name)
}
