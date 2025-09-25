package cli

import (
	"fmt"

	"github.com/tiktoken-go/tokenizer"
)

type TokenEstimation struct {
	TokensCount int
}

func estimateTokens(text string) (TokenEstimation, error) {
	// Count tokens using cl100k_base encoding (used by GPT-4, GPT-3.5-turbo)
	enc, err := tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		return TokenEstimation{}, fmt.Errorf("failed to get tokenizer: %w", err)
	}

	// Convert bytes to string and encode
	tokens, _, _ := enc.Encode(text)
	tokenCount := len(tokens)
	fmt.Printf("Text size: %d bytes\n", len(text))
	fmt.Printf("Token count: %d tokens\n", tokenCount)

	// Show costs for all supported models
	fmt.Println("Estimated costs (input tokens):")
	for model, costPerMillion := range modelCosts {
		cost := float64(tokenCount) * costPerMillion / 1000000
		fmt.Printf("  %s: $%.4f\n", model, cost)
	}

	return TokenEstimation{
		TokensCount: tokenCount,
	}, nil
}

// Cost per million tokens (input) in USD
var modelCosts = map[Model]float64{
	ModelGPT5Nano: 0.05, // $0.05 per 1M tokens
	ModelGPT5Mini: 0.25, // $0.25 per 1M tokens
	ModelGPT5:     1.25, // $1.25 per 1M tokens
	ModelGPT51:    1.25, // $1.25 per 1M tokens
}
