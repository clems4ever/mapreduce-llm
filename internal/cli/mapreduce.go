package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	myopenai "github.com/clems4ever/big-context/internal/openai"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
	"github.com/tiktoken-go/tokenizer"
	"golang.org/x/sync/errgroup"
)

func Process(ctx context.Context, apiKey string, model Model, prompt, filePath string) error {
	openaiClient, err := myopenai.NewClient(apiKey, nil)
	if err != nil {
		return fmt.Errorf("failed to instantiate openai client: %w", err)
	}

	return ProcessWithClient(ctx, openaiClient, model, prompt, filePath, true)
}

// ProcessWithClient processes a file with a custom ChatGenerator client.
// This function is designed for testing and allows injection of mock clients.
func ProcessWithClient(ctx context.Context, client myopenai.ChatGenerator, model Model, prompt, filePath string, requireConfirmation bool) error {
	fmt.Printf("File path provided: %s\n", filePath)

	b, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	text := string(b)
	totalEstimation, err := estimateTokens(text)
	if err != nil {
		return fmt.Errorf("failed to estimate tokens: %w", err)
	}

	fmt.Printf("Total tokens: %d\n", totalEstimation.TokensCount)

	chunks, err := splitIntoTokenChunks(text, 2000)
	if err != nil {
		return fmt.Errorf("failed to split into chunks: %w", err)
	}

	fmt.Printf("Split into %d chunks\n", len(chunks))

	// Ask for user confirmation before proceeding
	if requireConfirmation {
		fmt.Print("\nDo you want to proceed with processing? (yes/no): ")
		var response string
		fmt.Scanln(&response)

		if strings.ToLower(strings.TrimSpace(response)) != "yes" && strings.ToLower(strings.TrimSpace(response)) != "y" {
			fmt.Println("Processing cancelled by user.")
			return nil
		}

		fmt.Println("Proceeding with processing...")
	}

	// Create directory for chunks and results at the same level as the original file
	baseFileName := strings.TrimSuffix(filePath, filepath.Ext(filePath))
	chunkDir := baseFileName // Keep the full path, just remove extension
	err = os.MkdirAll(chunkDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create chunk directory: %w", err)
	}
	fmt.Printf("Using chunk directory: %s/\n", chunkDir)

	// Check for existing cached results
	cachedCount := 0
	for i := range chunks {
		resultFileName := filepath.Join(chunkDir, fmt.Sprintf("result%d.txt", i+1))
		if _, err := os.Stat(resultFileName); err == nil {
			cachedCount++
		}
	}

	if cachedCount > 0 {
		fmt.Printf("Found %d cached results, will process %d new chunks\n", cachedCount, len(chunks)-cachedCount)
	}

	fmt.Printf("Starting parallel processing of %d chunks...\n", len(chunks))

	prompt = prompt + "\nReturn the lines that you want to keep."

	g, gCtx := errgroup.WithContext(ctx)

	// Process each chunk with OpenAI
	results := make([]string, len(chunks))

	// Progress tracking
	var completed int64
	totalChunks := int64(len(chunks))
	var mu sync.Mutex

	for i, chunk := range chunks {
		i, chunk := i, chunk
		g.Go(func() error {
			result, err := processChunk(gCtx, model, i, chunkDir, client, prompt, chunk)
			if err != nil {
				return err
			}
			results[i] = result

			// Update progress
			current := atomic.AddInt64(&completed, 1)
			progress := float64(current) / float64(totalChunks) * 100

			mu.Lock()
			fmt.Printf("Progress: %d/%d chunks completed (%.1f%%)\n", current, totalChunks, progress)
			mu.Unlock()

			return nil
		})
	}

	err = g.Wait()
	if err != nil {
		return fmt.Errorf("failed to wait for all subtasks to complete: %w", err)
	}

	fmt.Printf("\n✓ All %d chunks processed successfully!\n", len(chunks))

	var combinedResults strings.Builder

	for _, result := range results {
		// Add to combined results (just append without separators)
		combinedResults.WriteString(result)
	}

	// Write combined results to file
	filePathWithoutExt := strings.TrimSuffix(filePath, filepath.Ext(filePath))
	combinedFileName := fmt.Sprintf("%s.combined_results.txt", filePathWithoutExt)
	err = os.WriteFile(combinedFileName, []byte(combinedResults.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write combined results: %w", err)
	}

	fmt.Printf("\n=== Combined results written to: %s ===\n", combinedFileName)

	return nil
}

func processChunk(ctx context.Context, model Model, i int, chunkDir string, client myopenai.ChatGenerator, prompt, chunk string) (string, error) {
	chunkFileName := filepath.Join(chunkDir, fmt.Sprintf("chunk%d.txt", i+1))
	resultFileName := filepath.Join(chunkDir, fmt.Sprintf("result%d.txt", i+1))

	// Check if result already exists
	if existingResult, err := os.ReadFile(resultFileName); err == nil {
		fmt.Printf("Chunk %d: Using cached result -> %s\n", i+1, resultFileName)
		return string(existingResult), nil
	}

	// Write chunk to disk
	err := os.WriteFile(chunkFileName, []byte(chunk), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write chunk %d: %w", i+1, err)
	}

	fmt.Printf("Chunk %d: %s (processing...)\n", i+1, chunkFileName)

	res, err := client.GenerateChatCompletion(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(prompt),
			openai.UserMessage(chunk),
		},
		Model:       shared.ChatModel(model),
		ServiceTier: openai.ChatCompletionNewParamsServiceTierFlex,
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate chat completion for chunk %d: %w", i+1, err)
	}

	// Extract the content from the response
	if len(res.Choices) > 0 && res.Choices[0].Message.Content != "" {
		content := res.Choices[0].Message.Content

		// Cache the result to disk
		err = os.WriteFile(resultFileName, []byte(content), 0644)
		if err != nil {
			fmt.Printf("Warning: failed to cache result for chunk %d: %v\n", i+1, err)
		} else {
			fmt.Printf("Chunk %d: Result cached -> %s\n", i+1, resultFileName)
		}

		return content, nil
	}

	return "", fmt.Errorf("no content in response for chunk %d", i+1)
}

func splitIntoTokenChunks(text string, maxTokensPerChunk int) ([]string, error) {
	// Get the tokenizer
	enc, err := tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokenizer: %w", err)
	}

	var chunks []string
	lines := strings.Split(text, "\n")

	currentChunk := ""
	currentTokens := 0

	for _, line := range lines {
		lineWithNewline := line + "\n"
		tokens, _, _ := enc.Encode(lineWithNewline)
		lineTokenCount := len(tokens)

		// If adding this line would exceed the limit, start a new chunk
		if currentTokens+lineTokenCount > maxTokensPerChunk && currentChunk != "" {
			chunks = append(chunks, strings.TrimSuffix(currentChunk, "\n"))
			currentChunk = lineWithNewline
			currentTokens = lineTokenCount
		} else {
			currentChunk += lineWithNewline
			currentTokens += lineTokenCount
		}

		// Handle case where a single line exceeds the token limit
		if lineTokenCount > maxTokensPerChunk {
			// Split the line into smaller parts
			words := strings.Fields(line)
			wordChunk := ""
			wordTokens := 0

			for _, word := range words {
				wordWithSpace := word + " "
				tokens, _, _ := enc.Encode(wordWithSpace)
				wordTokenCount := len(tokens)

				if wordTokens+wordTokenCount > maxTokensPerChunk && wordChunk != "" {
					chunks = append(chunks, strings.TrimSpace(wordChunk))
					wordChunk = wordWithSpace
					wordTokens = wordTokenCount
				} else {
					wordChunk += wordWithSpace
					wordTokens += wordTokenCount
				}
			}

			if wordChunk != "" {
				currentChunk = strings.TrimSpace(wordChunk) + "\n"
				tokens, _, _ := enc.Encode(currentChunk)
				currentTokens = len(tokens)
			}
		}
	}

	// Add the last chunk if it's not empty
	if currentChunk != "" {
		chunks = append(chunks, strings.TrimSuffix(currentChunk, "\n"))
	}

	return chunks, nil
}

// CleanCache removes the entire chunk directory for a given file path
func CleanCache(filePath string) error {
	chunkDir := strings.TrimSuffix(filePath, filepath.Ext(filePath))

	if _, err := os.Stat(chunkDir); os.IsNotExist(err) {
		fmt.Printf("No cache directory found: %s\n", chunkDir)
		return nil
	}

	err := os.RemoveAll(chunkDir)
	if err != nil {
		return fmt.Errorf("failed to remove cache directory: %w", err)
	}

	fmt.Printf("Removed cache directory: %s/\n", chunkDir)
	return nil
}
