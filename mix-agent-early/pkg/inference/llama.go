package inference

/*
#cgo CFLAGS: -I${SRCDIR}/../../../../vendor/llama.cpp
#cgo LDFLAGS: -L${SRCDIR}/../../../../vendor/llama.cpp -lllama -lm -lstdc++
#cgo linux LDFLAGS: -lpthread -ldl

#include <stdlib.h>
#include <stdbool.h>

// Forward declarations for llama.cpp
typedef struct llama_model llama_model;
typedef struct llama_context llama_context;
typedef int32_t llama_token;

// Model parameters
struct llama_model_params {
    int32_t n_gpu_layers;
    int32_t main_gpu;
    const float * tensor_split;
    void (*progress_callback)(float progress, void * ctx);
    void * progress_callback_user_data;
    bool vocab_only;
    bool use_mmap;
    bool use_mlock;
};

// Context parameters
struct llama_context_params {
    uint32_t seed;
    uint32_t n_ctx;
    uint32_t n_batch;
    uint32_t n_threads;
    uint32_t n_threads_batch;
    int8_t rope_scaling_type;
    float rope_freq_base;
    float rope_freq_scale;
    float yarn_ext_factor;
    float yarn_attn_factor;
    float yarn_beta_fast;
    float yarn_beta_slow;
    uint32_t yarn_orig_ctx;
    bool mul_mat_q;
    bool f16_kv;
    bool logits_all;
    bool embedding;
    bool offload_kqv;
};

// Stub functions - actual implementation links to llama.cpp
extern struct llama_model_params llama_model_default_params(void);
extern struct llama_context_params llama_context_default_params(void);
extern llama_model * llama_load_model_from_file(const char * path_model, struct llama_model_params params);
extern void llama_free_model(llama_model * model);
extern llama_context * llama_new_context_with_model(llama_model * model, struct llama_context_params params);
extern void llama_free(llama_context * ctx);
extern int llama_tokenize(llama_model * model, const char * text, int text_len, llama_token * tokens, int n_max_tokens, bool add_bos, bool special);
extern int llama_n_vocab(const llama_model * model);
extern int llama_n_ctx(const llama_context * ctx);
extern int llama_decode(llama_context * ctx, struct llama_batch batch);
extern float * llama_get_logits(llama_context * ctx);
extern llama_token llama_token_bos(const llama_model * model);
extern llama_token llama_token_eos(const llama_model * model);
extern const char * llama_token_get_text(const llama_model * model, llama_token token);

// Batch structure
struct llama_batch {
    int32_t n_tokens;
    llama_token * token;
    float * embd;
    int32_t * pos;
    int32_t * n_seq_id;
    int32_t ** seq_id;
    int8_t * logits;
    int32_t all_pos_0;
    int32_t all_pos_1;
    int32_t all_seq_id;
};

extern struct llama_batch llama_batch_init(int32_t n_tokens, int32_t embd, int32_t n_seq_max);
extern void llama_batch_free(struct llama_batch batch);

// Sampling
extern llama_token llama_sample_token_greedy(llama_context * ctx, float * logits, int n_vocab);
*/
import "C"

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"
)

var (
	ErrModelNotLoaded  = errors.New("model not loaded")
	ErrContextNotReady = errors.New("context not ready")
	ErrTokenization    = errors.New("tokenization failed")
	ErrGeneration      = errors.New("generation failed")
)

// LlamaModel wraps a llama.cpp model
type LlamaModel struct {
	model   *C.llama_model
	ctx     *C.llama_context
	nVocab  int
	nCtx    int
	mu      sync.Mutex
	loaded  bool
	path    string
	threads int
}

// ModelConfig holds model configuration
type ModelConfig struct {
	Path          string
	ContextLength int
	Threads       int
	GPULayers     int
	UseMMap       bool
	UseMlock      bool
}

// DefaultModelConfig returns default configuration
func DefaultModelConfig() ModelConfig {
	return ModelConfig{
		ContextLength: 2048,
		Threads:       4,
		GPULayers:     0,
		UseMMap:       true,
		UseMlock:      false,
	}
}

// GenerateConfig holds generation configuration
type GenerateConfig struct {
	MaxTokens     int
	Temperature   float32
	TopP          float32
	TopK          int
	RepeatPenalty float32
	StopTokens    []string
}

// DefaultGenerateConfig returns default generation configuration
func DefaultGenerateConfig() GenerateConfig {
	return GenerateConfig{
		MaxTokens:     256,
		Temperature:   0.7,
		TopP:          0.95,
		TopK:          40,
		RepeatPenalty: 1.1,
	}
}

// NewLlamaModel creates a new LlamaModel instance
func NewLlamaModel() *LlamaModel {
	return &LlamaModel{
		threads: 4,
	}
}

// Load loads a model from file
func (m *LlamaModel) Load(config ModelConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.loaded {
		m.unloadUnsafe()
	}

	// Get default model params
	modelParams := C.llama_model_default_params()
	modelParams.n_gpu_layers = C.int32_t(config.GPULayers)
	modelParams.use_mmap = C.bool(config.UseMMap)
	modelParams.use_mlock = C.bool(config.UseMlock)

	// Load model
	cPath := C.CString(config.Path)
	defer C.free(unsafe.Pointer(cPath))

	m.model = C.llama_load_model_from_file(cPath, modelParams)
	if m.model == nil {
		return fmt.Errorf("failed to load model from %s", config.Path)
	}

	// Get default context params
	ctxParams := C.llama_context_default_params()
	ctxParams.n_ctx = C.uint32_t(config.ContextLength)
	ctxParams.n_threads = C.uint32_t(config.Threads)
	ctxParams.n_threads_batch = C.uint32_t(config.Threads)

	// Create context
	m.ctx = C.llama_new_context_with_model(m.model, ctxParams)
	if m.ctx == nil {
		C.llama_free_model(m.model)
		m.model = nil
		return errors.New("failed to create context")
	}

	m.nVocab = int(C.llama_n_vocab(m.model))
	m.nCtx = int(C.llama_n_ctx(m.ctx))
	m.path = config.Path
	m.threads = config.Threads
	m.loaded = true

	return nil
}

// Unload unloads the model
func (m *LlamaModel) Unload() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unloadUnsafe()
}

func (m *LlamaModel) unloadUnsafe() {
	if m.ctx != nil {
		C.llama_free(m.ctx)
		m.ctx = nil
	}
	if m.model != nil {
		C.llama_free_model(m.model)
		m.model = nil
	}
	m.loaded = false
}

// IsLoaded returns whether a model is loaded
func (m *LlamaModel) IsLoaded() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.loaded
}

// Tokenize converts text to tokens
func (m *LlamaModel) Tokenize(text string, addBOS bool) ([]int32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.loaded {
		return nil, ErrModelNotLoaded
	}

	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))

	// Allocate token buffer (estimate: 1 token per 4 chars + some extra)
	maxTokens := len(text)/4 + 100
	tokens := make([]C.llama_token, maxTokens)

	nTokens := C.llama_tokenize(
		m.model,
		cText,
		C.int(len(text)),
		&tokens[0],
		C.int(maxTokens),
		C.bool(addBOS),
		C.bool(false),
	)

	if nTokens < 0 {
		return nil, ErrTokenization
	}

	result := make([]int32, nTokens)
	for i := 0; i < int(nTokens); i++ {
		result[i] = int32(tokens[i])
	}

	return result, nil
}

// Generate generates text from a prompt
func (m *LlamaModel) Generate(prompt string, config GenerateConfig) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.loaded {
		return "", ErrModelNotLoaded
	}

	// Tokenize prompt
	cPrompt := C.CString(prompt)
	defer C.free(unsafe.Pointer(cPrompt))

	maxTokens := m.nCtx
	tokens := make([]C.llama_token, maxTokens)

	nPromptTokens := C.llama_tokenize(
		m.model,
		cPrompt,
		C.int(len(prompt)),
		&tokens[0],
		C.int(maxTokens),
		C.bool(true),
		C.bool(false),
	)

	if nPromptTokens < 0 {
		return "", ErrTokenization
	}

	// Create batch for prompt evaluation
	batch := C.llama_batch_init(C.int32_t(nPromptTokens), 0, 1)
	defer C.llama_batch_free(batch)

	// Fill batch with prompt tokens
	for i := 0; i < int(nPromptTokens); i++ {
		batch.token = (*C.llama_token)(unsafe.Pointer(uintptr(unsafe.Pointer(batch.token)) + uintptr(i)*unsafe.Sizeof(C.llama_token(0))))
		*batch.token = tokens[i]
	}
	batch.n_tokens = C.int32_t(nPromptTokens)

	// Evaluate prompt
	if C.llama_decode(m.ctx, batch) != 0 {
		return "", ErrGeneration
	}

	// Generate tokens
	var result []byte
	eosToken := C.llama_token_eos(m.model)

	for i := 0; i < config.MaxTokens; i++ {
		// Get logits
		logits := C.llama_get_logits(m.ctx)

		// Sample next token (greedy for simplicity in early boot)
		nextToken := C.llama_sample_token_greedy(m.ctx, logits, C.int(m.nVocab))

		// Check for EOS
		if nextToken == eosToken {
			break
		}

		// Get token text
		tokenText := C.llama_token_get_text(m.model, nextToken)
		if tokenText != nil {
			result = append(result, C.GoString(tokenText)...)
		}

		// Prepare next batch
		batch.n_tokens = 1
		batch.token = &nextToken

		if C.llama_decode(m.ctx, batch) != 0 {
			break
		}
	}

	return string(result), nil
}

// GetInfo returns model information
func (m *LlamaModel) GetInfo() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	return map[string]interface{}{
		"loaded":   m.loaded,
		"path":     m.path,
		"n_vocab":  m.nVocab,
		"n_ctx":    m.nCtx,
		"threads":  m.threads,
	}
}
