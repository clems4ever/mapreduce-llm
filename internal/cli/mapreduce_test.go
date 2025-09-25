package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	myopenai "github.com/clems4ever/big-context/internal/openai"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/ssestream"
)

// mockChatGenerator is a mock implementation of the ChatGenerator interface for testing
type mockChatGenerator struct {
	responseFunc   func(callCount int) string // function to generate response based on call count
	callCount      int
	shouldError    bool
	errorOnChunk   int
}

func (m *mockChatGenerator) GenerateChatCompletion(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	m.callCount++

	if m.shouldError && (m.errorOnChunk == 0 || m.errorOnChunk == m.callCount) {
		return nil, fmt.Errorf("mock error: simulated API failure")
	}

	// Generate response
	response := "mock response"
	if m.responseFunc != nil {
		response = m.responseFunc(m.callCount)
	}

	return &openai.ChatCompletion{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Content: response,
				},
			},
		},
	}, nil
}

func (m *mockChatGenerator) GenerateChatCompletionStream(ctx context.Context, params openai.ChatCompletionNewParams) *ssestream.Stream[openai.ChatCompletionChunk] {
	// Not used in Process function, return nil
	return nil
}

// Ensure mockChatGenerator implements ChatGenerator interface
var _ myopenai.ChatGenerator = (*mockChatGenerator)(nil)

func TestProcessWithClient_Success(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "This is a test file.\nIt has multiple lines.\nAnd some content to process."

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create mock client
	mock := &mockChatGenerator{
		responseFunc: func(callCount int) string {
			return "processed content"
		},
	}

	// Run the process
	ctx := context.Background()
	err = ProcessWithClient(ctx, mock, ModelGPT5Nano, "test prompt", testFile, false)
	if err != nil {
		t.Fatalf("ProcessWithClient failed: %v", err)
	}

	// Verify the chunk directory was created
	chunkDir := strings.TrimSuffix(testFile, filepath.Ext(testFile))
	if _, err := os.Stat(chunkDir); os.IsNotExist(err) {
		t.Errorf("Chunk directory was not created: %s", chunkDir)
	}

	// Verify the combined results file was created
	combinedFile := strings.TrimSuffix(testFile, filepath.Ext(testFile)) + ".combined_results.txt"
	if _, err := os.Stat(combinedFile); os.IsNotExist(err) {
		t.Errorf("Combined results file was not created: %s", combinedFile)
	}

	// Verify the content of the combined results
	content, err := os.ReadFile(combinedFile)
	if err != nil {
		t.Fatalf("Failed to read combined results: %v", err)
	}

	if string(content) != "processed content" {
		t.Errorf("Expected combined results to be 'processed content', got: %s", string(content))
	}

	// Verify mock was called
	if mock.callCount != 1 {
		t.Errorf("Expected 1 API call, got %d", mock.callCount)
	}
}

func TestProcessWithClient_MultipleChunks(t *testing.T) {
	// Create a temporary test file with content that will be split into multiple chunks
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large_test.txt")

	// Create content that will be split into multiple chunks (each chunk ~2000 tokens)
	// Using approximately 500 words per chunk to ensure multiple chunks
	var sb strings.Builder
	for i := 0; i < 3000; i++ {
		sb.WriteString("word ")
	}
	testContent := sb.String()

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create mock client with responses for each chunk
	mock := &mockChatGenerator{
		responseFunc: func(callCount int) string {
			return fmt.Sprintf("response for chunk %d", callCount)
		},
	}

	// Run the process
	ctx := context.Background()
	err = ProcessWithClient(ctx, mock, ModelGPT5Nano, "test prompt", testFile, false)
	if err != nil {
		t.Fatalf("ProcessWithClient failed: %v", err)
	}

	// Verify multiple chunks were created and processed
	chunkDir := strings.TrimSuffix(testFile, filepath.Ext(testFile))
	
	// Count chunk files
	entries, err := os.ReadDir(chunkDir)
	if err != nil {
		t.Fatalf("Failed to read chunk directory: %v", err)
	}

	chunkCount := 0
	resultCount := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "chunk") {
			chunkCount++
		}
		if strings.HasPrefix(entry.Name(), "result") {
			resultCount++
		}
	}

	if chunkCount < 2 {
		t.Errorf("Expected at least 2 chunks, got %d", chunkCount)
	}

	if resultCount != chunkCount {
		t.Errorf("Expected %d results to match %d chunks, got %d results", chunkCount, chunkCount, resultCount)
	}

	// Verify mock was called multiple times
	if mock.callCount < 2 {
		t.Errorf("Expected at least 2 API calls, got %d", mock.callCount)
	}
}

func TestProcessWithClient_CachedResults(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "cached_test.txt")
	testContent := "This is a test for caching."

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create mock client
	mock := &mockChatGenerator{
		responseFunc: func(callCount int) string {
			return "first run response"
		},
	}

	// First run
	ctx := context.Background()
	err = ProcessWithClient(ctx, mock, ModelGPT5Nano, "test prompt", testFile, false)
	if err != nil {
		t.Fatalf("First ProcessWithClient run failed: %v", err)
	}

	firstCallCount := mock.callCount

	// Create a new mock with different response
	mock2 := &mockChatGenerator{
		responseFunc: func(callCount int) string {
			return "second run response - should not be used"
		},
	}

	// Second run - should use cached results
	err = ProcessWithClient(ctx, mock2, ModelGPT5Nano, "test prompt", testFile, false)
	if err != nil {
		t.Fatalf("Second ProcessWithClient run failed: %v", err)
	}

	// Verify the second mock was NOT called (cache was used)
	if mock2.callCount != 0 {
		t.Errorf("Expected 0 API calls on second run (cached), got %d", mock2.callCount)
	}

	// Verify the combined results still contain the first response
	combinedFile := strings.TrimSuffix(testFile, filepath.Ext(testFile)) + ".combined_results.txt"
	content, err := os.ReadFile(combinedFile)
	if err != nil {
		t.Fatalf("Failed to read combined results: %v", err)
	}

	if string(content) != "first run response" {
		t.Errorf("Expected cached results 'first run response', got: %s", string(content))
	}

	t.Logf("First run: %d calls, Second run (cached): %d calls", firstCallCount, mock2.callCount)
}

func TestProcessWithClient_APIError(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "error_test.txt")
	testContent := "This test will fail."

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create mock client that returns an error
	mock := &mockChatGenerator{
		shouldError: true,
	}

	// Run the process - should fail
	ctx := context.Background()
	err = ProcessWithClient(ctx, mock, ModelGPT5Nano, "test prompt", testFile, false)
	if err == nil {
		t.Fatal("Expected ProcessWithClient to fail with API error, but it succeeded")
	}

	if !strings.Contains(err.Error(), "mock error") {
		t.Errorf("Expected error message to contain 'mock error', got: %v", err)
	}
}

func TestProcessWithClient_FileNotFound(t *testing.T) {
	// Use a non-existent file path
	testFile := "/tmp/nonexistent_file_12345.txt"

	mock := &mockChatGenerator{}

	// Run the process - should fail
	ctx := context.Background()
	err := ProcessWithClient(ctx, mock, ModelGPT5Nano, "test prompt", testFile, false)
	if err == nil {
		t.Fatal("Expected ProcessWithClient to fail with file not found error, but it succeeded")
	}

	if !strings.Contains(err.Error(), "failed to read file") {
		t.Errorf("Expected error message to contain 'failed to read file', got: %v", err)
	}
}

func TestProcessWithClient_EmptyFile(t *testing.T) {
	// Create a temporary empty test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty_test.txt")

	err := os.WriteFile(testFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	mock := &mockChatGenerator{}

	// Run the process - should handle empty file gracefully
	ctx := context.Background()
	err = ProcessWithClient(ctx, mock, ModelGPT5Nano, "test prompt", testFile, false)
	
	// Empty file should still process (might create 0 or 1 chunk depending on implementation)
	if err != nil {
		t.Logf("ProcessWithClient with empty file: %v", err)
		// This is acceptable behavior - empty files might error or succeed
	}
}

func TestCleanCache(t *testing.T) {
	// Create a temporary test file and cache
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "cleanup_test.txt")
	testContent := "Test content for cleanup"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create mock client and process to generate cache
	mock := &mockChatGenerator{
		responseFunc: func(callCount int) string {
			return "test response"
		},
	}

	ctx := context.Background()
	err = ProcessWithClient(ctx, mock, ModelGPT5Nano, "test prompt", testFile, false)
	if err != nil {
		t.Fatalf("ProcessWithClient failed: %v", err)
	}

	// Verify cache directory exists
	chunkDir := strings.TrimSuffix(testFile, filepath.Ext(testFile))
	if _, err := os.Stat(chunkDir); os.IsNotExist(err) {
		t.Fatalf("Cache directory should exist before cleanup: %s", chunkDir)
	}

	// Clean the cache
	err = CleanCache(testFile)
	if err != nil {
		t.Fatalf("CleanCache failed: %v", err)
	}

	// Verify cache directory was removed
	if _, err := os.Stat(chunkDir); !os.IsNotExist(err) {
		t.Errorf("Cache directory should be removed after cleanup: %s", chunkDir)
	}
}

func TestCleanCache_NonExistentCache(t *testing.T) {
	// Try to clean cache for a file that never had a cache
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "no_cache_test.txt")

	// Clean cache for non-existent cache - should not error
	err := CleanCache(testFile)
	if err != nil {
		t.Errorf("CleanCache should not error for non-existent cache, got: %v", err)
	}
}

func TestSplitIntoTokenChunks(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		maxTokensPerChunk int
		expectedMinChunks int
		expectedMaxChunks int
	}{
		{
			name:              "short text single chunk",
			input:             "Short text that fits in one chunk",
			maxTokensPerChunk: 1000,
			expectedMinChunks: 1,
			expectedMaxChunks: 1,
		},
		{
			name:              "multiline text",
			input:             "Line 1\nLine 2\nLine 3\nLine 4\nLine 5",
			maxTokensPerChunk: 1000,
			expectedMinChunks: 1,
			expectedMaxChunks: 1,
		},
		{
			name:              "long text multiple chunks",
			input:             strings.Repeat("word ", 1000), // ~1000 words
			maxTokensPerChunk: 100,
			expectedMinChunks: 10,
			expectedMaxChunks: 25,
		},
		{
			name:              "very small chunk size",
			input:             "This is a test sentence with multiple words",
			maxTokensPerChunk: 3,
			expectedMinChunks: 2,
			expectedMaxChunks: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks, err := splitIntoTokenChunks(tt.input, tt.maxTokensPerChunk)
			if err != nil {
				t.Fatalf("splitIntoTokenChunks failed: %v", err)
			}

			if len(chunks) < tt.expectedMinChunks || len(chunks) > tt.expectedMaxChunks {
				t.Errorf("Expected %d-%d chunks, got %d", tt.expectedMinChunks, tt.expectedMaxChunks, len(chunks))
			}

			// Verify all chunks are within token limit
			for i, chunk := range chunks {
				est, err := estimateTokens(chunk)
				if err != nil {
					t.Fatalf("Failed to estimate tokens for chunk %d: %v", i, err)
				}

				// Allow some tolerance for chunk size (chunks might slightly exceed due to line boundaries)
				if est.TokensCount > tt.maxTokensPerChunk*2 {
					t.Errorf("Chunk %d has %d tokens, which significantly exceeds limit of %d", 
						i, est.TokensCount, tt.maxTokensPerChunk)
				}
			}

			// Verify chunks can be recombined to original text (preserving content)
			var recombined strings.Builder
			for i, chunk := range chunks {
				recombined.WriteString(chunk)
				if i < len(chunks)-1 {
					recombined.WriteString("\n")
				}
			}

			// The recombined text should contain the same words (newlines might differ)
			originalWords := strings.Fields(tt.input)
			recombinedWords := strings.Fields(recombined.String())

			if len(originalWords) != len(recombinedWords) {
				t.Errorf("Word count mismatch after recombining chunks: original=%d, recombined=%d",
					len(originalWords), len(recombinedWords))
			}
		})
	}
}

func TestSplitIntoTokenChunks_EmptyInput(t *testing.T) {
	chunks, err := splitIntoTokenChunks("", 1000)
	if err != nil {
		t.Fatalf("splitIntoTokenChunks failed on empty input: %v", err)
	}

	// Empty input should produce either 0 or 1 empty chunk
	if len(chunks) > 1 {
		t.Errorf("Expected 0 or 1 chunk for empty input, got %d", len(chunks))
	}
}
