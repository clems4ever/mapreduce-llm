package cli

// Model represents an AI model name
type Model string

// Model names
const (
	ModelGPT5Nano Model = "gpt-5-nano"
	ModelGPT5Mini Model = "gpt-5-mini"
	ModelGPT5     Model = "gpt-5"
	ModelGPT51    Model = "gpt-5.1"
)
