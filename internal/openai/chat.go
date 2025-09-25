// Defines the ChatGenerator interface for generating chat completions using the OpenAI API.
package myopenai

import (
	"context"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/ssestream"
)

// ChatGenerator provides an interface for generating chat completions using the OpenAI API.
// Implementations should provide a method to generate a chat completion from the given parameters.
type ChatGenerator interface {
	GenerateChatCompletion(ctx context.Context, body openai.ChatCompletionNewParams) (*openai.ChatCompletion, error)
	GenerateChatCompletionStream(ctx context.Context, body openai.ChatCompletionNewParams) *ssestream.Stream[openai.ChatCompletionChunk]
}
