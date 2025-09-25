// Provides a client implementation for interacting with the OpenAI API for speech synthesis.
// Defines a Client interface and a concrete implementation for generating speech audio.
package myopenai

import (
	"context"
	"net/http"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/ssestream"
)

// Client defines an interface for generating speech audio using the OpenAI API.
type Client interface {
	GenerateSpeech(ctx context.Context, params openai.AudioSpeechNewParams) (*http.Response, error)
}

// clientImpl is a concrete implementation of the Client interface using the OpenAI Go SDK.
type clientImpl struct {
	client openai.Client
}

// NewClient creates a new clientImpl using the OPENAI_API_KEY environment variable.
// Returns an error if the API key is not set.
func NewClient(apiKey string, httpClient *http.Client) (*clientImpl, error) {
	clientOpts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithRequestTimeout(5 * time.Minute),
	}

	if httpClient != nil {
		clientOpts = append(clientOpts, option.WithHTTPClient(httpClient))
	}
	return &clientImpl{
		client: openai.NewClient(clientOpts...),
	}, nil
}

func (o *clientImpl) GenerateChatCompletion(ctx context.Context, body openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	return o.client.Chat.Completions.New(ctx, body)
}

func (o *clientImpl) GenerateChatCompletionStream(ctx context.Context, body openai.ChatCompletionNewParams) *ssestream.Stream[openai.ChatCompletionChunk] {
	return o.client.Chat.Completions.NewStreaming(ctx, body)
}
