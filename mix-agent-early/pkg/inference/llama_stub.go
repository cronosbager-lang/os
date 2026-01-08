// +build !cgo

package inference

import (
	"errors"
	"sync"
)

// Stub implementation when CGO is not available
// This allows the code to compile and run in mock mode

var ErrCGODisabled = errors.New("CGO is disabled, using mock mode")

type LlamaModel struct {
	mu      sync.Mutex
	loaded  bool
	path    string
	threads int
	nVocab  int
	nCtx    int
}

func NewLlamaModel() *LlamaModel {
	return &LlamaModel{
		threads: 4,
		nVocab:  32000,
		nCtx:    2048,
	}
}

func (m *LlamaModel) Load(config ModelConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// In stub mode, we just pretend to load
	m.path = config.Path
	m.threads = config.Threads
	m.nCtx = config.ContextLength
	m.loaded = false // Keep as false to use mock generation

	return nil
}

func (m *LlamaModel) Unload() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loaded = false
}

func (m *LlamaModel) IsLoaded() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.loaded
}

func (m *LlamaModel) Tokenize(text string, addBOS bool) ([]int32, error) {
	// Simple mock tokenization
	tokens := make([]int32, 0, len(text)/4)
	for i := 0; i < len(text); i += 4 {
		tokens = append(tokens, int32(i/4))
	}
	return tokens, nil
}

func (m *LlamaModel) Generate(prompt string, config GenerateConfig) (string, error) {
	return "", ErrCGODisabled
}

func (m *LlamaModel) GetInfo() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	return map[string]interface{}{
		"loaded":      m.loaded,
		"path":        m.path,
		"n_vocab":     m.nVocab,
		"n_ctx":       m.nCtx,
		"threads":     m.threads,
		"stub_mode":   true,
		"cgo_enabled": false,
	}
}
