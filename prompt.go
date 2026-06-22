package sdk

import (
	"github.com/hastekit/hastekit-sdk-go/pkg/agents/prompts"
	"github.com/hastekit/hastekit-sdk-go/pkg/hastekitgateway"
)

func (c *SDK) Prompt(prompt string, opts ...prompts.PromptOption) *prompts.SimplePrompt {
	return prompts.New(prompt, opts...)
}

func (c *SDK) RemotePrompt(name, alias string, opts ...prompts.PromptOption) *prompts.SimplePrompt {
	return prompts.NewWithLoader(hastekitgateway.NewExternalPromptPersistence(c.endpoint, c.orgName, c.projectName, name, alias, c.httpClient), opts...)
}

func (c *SDK) CustomPrompt(loader prompts.PromptLoader, opts ...prompts.PromptOption) *prompts.SimplePrompt {
	return prompts.NewWithLoader(loader, opts...)
}
