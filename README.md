# MapReduce LLM

A command-line tool that performs MapReduce-style processing on large text files using OpenAI's language models. Split large datasets into manageable chunks, process them in parallel with LLM prompts, and combine the results.

## Overview

MapReduce LLM enables you to apply AI-powered transformations to large text files that would otherwise exceed token limits of language models. The tool:

1. **Splits** your data file into token-sized chunks (default: 2000 tokens)
2. **Maps** each chunk through an OpenAI model with your custom prompt to save prompt tokens
3. **Reduces** the results by combining all processed chunks into a single output file

Perfect for filtering, transforming, or analyzing large datasets with natural language instructions.

## Features

- 🚀 **Parallel Processing**: Processes chunks concurrently for faster results
- 💾 **Caching**: Automatically caches intermediate results to resume interrupted jobs
- 📊 **Progress Tracking**: Real-time progress updates during processing
- 🔍 **Token Estimation**: Pre-flight token counting before processing begins
- ✅ **Confirmation Prompts**: Interactive confirmation before running costly operations
- 🎯 **Multiple Models**: Support for GPT-5-nano, GPT-5-mini, GPT-5, and GPT-5.1

## Installation

### Prerequisites

- Go 1.25.1 or later
- OpenAI API key

### Build from Source

```bash
git clone https://github.com/clems4ever/mapreduce-llm.git
cd mapreduce-llm
go build -o mapred-llm ./cmd/cli
```

## Usage

### Basic Usage

```bash
export OPENAI_API_KEY="your-api-key-here"
./mapred-llm "your prompt here" path/to/data.txt
```

### Example: Filter Kitchen Product Reviews

Given a file with mixed product reviews, filter only kitchen-related items:

```bash
./mapred-llm "Select the lines with reviews that are about objects from the kitchen." examples/product-ratings/reviews.txt
```

**Input** (`reviews.txt`):
```
The blender rattles like it's trying to escape the kitchen...
This book left me more confused than enlightened...
The coffee maker sputters like a grumpy dragon...
The jacket fits perfectly and somehow makes me feel taller...
```

**Output** (`reviews.combined_results.txt`):
```
The blender rattles like it's trying to escape the kitchen...
The coffee maker sputters like a grumpy dragon...
The frying pan cooks evenly and cleans easily...
```

### Example: Extract Specific Items

```bash
./mapred-llm "Extract all fruit names, one per line" data/test-fruits.txt
```

## How It Works

1. **Read & Estimate**: Reads the input file and estimates total tokens
2. **Chunk**: Splits content into chunks of ~2000 tokens each
3. **Confirm**: Asks for user confirmation (shows chunk count and estimated cost)
4. **Process**: Sends each chunk to OpenAI with your prompt in parallel
5. **Cache**: Saves individual chunk results to `<filename>/result{N}.txt` for resuming if needed.
6. **Combine**: Merges all results into `<filename>.combined_results.txt`

### Directory Structure After Processing

```
data/
├── reviews.txt                      # Original file
├── reviews.combined_results.txt     # Final combined output
└── reviews/                         # Chunk directory
    ├── chunk1.txt                   # Input chunk 1
    ├── result1.txt                  # Processed result 1
    ├── chunk2.txt                   # Input chunk 2
    ├── result2.txt                  # Processed result 2
    └── ...
```

## Configuration

### Environment Variables

- `OPENAI_API_KEY` (required): Your OpenAI API key

### Models

The tool currently uses `gpt-5-nano` by default. Supported models are defined in `internal/cli/models.go`:

- `gpt-5-nano`
- `gpt-5-mini`
- `gpt-5`
- `gpt-5.1`

## Development

### Running Tests

```bash
go test -v ./...
```

### Running Tests with Coverage

```bash
go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
```

### Project Structure

```
.
├── cmd/cli/              # CLI entry point
├── internal/
│   ├── cli/              # Core MapReduce logic
│   └── openai/           # OpenAI client wrapper
├── data/                 # Sample data files
├── examples/             # Usage examples
│   └── product-ratings/  # Product review filtering example
└── .github/workflows/    # CI/CD configuration
```

## Use Cases

- **Data Filtering**: Remove irrelevant entries from large datasets
- **Text Classification**: Categorize text items across large files
- **Content Extraction**: Pull specific information from documents
- **Data Transformation**: Reformat or restructure text data
- **Sentiment Analysis**: Analyze sentiment across thousands of reviews
- **Data Cleaning**: Normalize or clean messy text data

## Tips

- **Cost Optimization**: Start with small test files to verify your prompt works as expected
- **Resume Processing**: Cached results allow you to interrupt and resume without reprocessing
- **Chunk Size**: Default 2000 tokens balances API limits with parallelization efficiency
- **Prompt Design**: Be specific and clear in your prompts for best results

## License

MIT License - see [LICENSE.md](LICENSE.md) for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Author

[@clems4ever](https://github.com/clems4ever)
