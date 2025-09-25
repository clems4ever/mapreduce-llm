package cli

import (
	"strings"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		expectedMin int
		expectedMax int
		expectError bool
	}{
		{
			name:        "empty string",
			text:        "",
			expectedMin: 0,
			expectedMax: 0,
			expectError: false,
		},
		{
			name:        "simple text",
			text:        "Hello, world!",
			expectedMin: 2,
			expectedMax: 5,
			expectError: false,
		},
		{
			name:        "longer text",
			text:        "This is a longer text with multiple words and sentences. It should have more tokens.",
			expectedMin: 15,
			expectedMax: 25,
			expectError: false,
		},
		{
			name:        "text with newlines",
			text:        "Line 1\nLine 2\nLine 3",
			expectedMin: 8,
			expectedMax: 12,
			expectError: false,
		},
		{
			name:        "repeated words",
			text:        strings.Repeat("test ", 100),
			expectedMin: 95,
			expectedMax: 105,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := estimateTokens(tt.text)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.TokensCount < tt.expectedMin || result.TokensCount > tt.expectedMax {
				t.Errorf("token count %d not in expected range [%d, %d]",
					result.TokensCount, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

func TestModelCosts(t *testing.T) {
	tests := []struct {
		model        Model
		expectedCost float64
	}{
		{ModelGPT5Nano, 0.05},
		{ModelGPT5Mini, 0.25},
		{ModelGPT5, 1.25},
		{ModelGPT51, 1.25},
	}

	for _, tt := range tests {
		t.Run(string(tt.model), func(t *testing.T) {
			cost, exists := modelCosts[tt.model]
			if !exists {
				t.Errorf("model %s not found in modelCosts", tt.model)
				return
			}

			if cost != tt.expectedCost {
				t.Errorf("expected cost %f for model %s, got %f",
					tt.expectedCost, tt.model, cost)
			}
		})
	}
}

func TestModelCostsCompleteness(t *testing.T) {
	// Ensure all models have a cost defined
	expectedModels := []Model{
		ModelGPT5Nano,
		ModelGPT5Mini,
		ModelGPT5,
		ModelGPT51,
	}

	for _, model := range expectedModels {
		if _, exists := modelCosts[model]; !exists {
			t.Errorf("model %s missing from modelCosts", model)
		}
	}

	// Ensure modelCosts has the expected number of entries
	if len(modelCosts) != len(expectedModels) {
		t.Errorf("expected %d models in modelCosts, got %d",
			len(expectedModels), len(modelCosts))
	}
}

func TestTokenEstimationConsistency(t *testing.T) {
	// Test that the same text always produces the same token count
	text := "This is a test sentence to verify consistency."

	result1, err1 := estimateTokens(text)
	if err1 != nil {
		t.Fatalf("first estimation failed: %v", err1)
	}

	result2, err2 := estimateTokens(text)
	if err2 != nil {
		t.Fatalf("second estimation failed: %v", err2)
	}

	if result1.TokensCount != result2.TokensCount {
		t.Errorf("token count inconsistent: first=%d, second=%d",
			result1.TokensCount, result2.TokensCount)
	}
}
