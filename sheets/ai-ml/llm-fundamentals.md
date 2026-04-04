# LLM Fundamentals (Large Language Model Core Concepts)

A comprehensive reference covering transformer-based large language model architecture, tokenization schemes, inference pipelines, quantization formats, serving frameworks, and sampling strategies essential for deploying and working with modern LLMs.

## Architecture Overview
### Decoder-Only Transformer Stack
```
Input Text
  -> Tokenizer (text -> token IDs)
  -> Embedding Layer (token IDs -> vectors)
  -> N x Transformer Blocks:
       -> Multi-Head Self-Attention
       -> Layer Normalization
       -> Feed-Forward Network (MLP)
       -> Residual Connections
  -> Final Layer Norm
  -> LM Head (logits over vocabulary)
  -> Sampling Strategy
  -> Token Output
  -> Detokenizer (token IDs -> text)
```

### Common Model Sizes
```
| Model      | Parameters | Hidden | Layers | Heads | Context |
|------------|-----------|--------|--------|-------|---------|
| GPT-2      | 1.5B      | 1600   | 48     | 25    | 1024    |
| LLaMA-2 7B | 6.7B      | 4096   | 32     | 32    | 4096    |
| LLaMA-2 70B| 68.9B     | 8192   | 80     | 64    | 4096    |
| Mistral 7B | 7.3B      | 4096   | 32     | 32    | 32768   |
| GPT-4      | ~1.8T*    | —      | —      | —     | 128K    |
```

## Tokenization
### BPE (Byte Pair Encoding)
```python
# tiktoken — OpenAI's BPE tokenizer
import tiktoken

enc = tiktoken.encoding_for_model("gpt-4")
tokens = enc.encode("Hello, world!")
print(tokens)          # [9906, 11, 1917, 0]
print(len(tokens))     # 4 tokens
text = enc.decode(tokens)
print(text)            # "Hello, world!"

# Count tokens before sending to API
def count_tokens(text, model="gpt-4"):
    enc = tiktoken.encoding_for_model(model)
    return len(enc.encode(text))
```

### SentencePiece (LLaMA / Mistral)
```python
import sentencepiece as spm

sp = spm.SentencePieceProcessor()
sp.load("tokenizer.model")

tokens = sp.encode("Hello, world!", out_type=str)
# ['▁Hello', ',', '▁world', '!']

ids = sp.encode("Hello, world!")
text = sp.decode(ids)
```

### HuggingFace Tokenizers
```python
from transformers import AutoTokenizer

tokenizer = AutoTokenizer.from_pretrained("meta-llama/Llama-2-7b-hf")
encoded = tokenizer("Hello, world!", return_tensors="pt")
print(encoded.input_ids)
print(tokenizer.vocab_size)  # 32000
```

## Quantization
### Format Comparison
```
| Format       | Bits | Method          | Use Case               |
|-------------|------|-----------------|------------------------|
| FP16        | 16   | Native          | GPU training/inference  |
| GPTQ        | 4    | Post-training   | GPU inference           |
| AWQ         | 4    | Activation-aware| GPU inference (faster)  |
| GGUF        | 2-8  | llama.cpp       | CPU/hybrid inference    |
| bitsandbytes| 4/8  | Dynamic         | Training (QLoRA)        |
```

### bitsandbytes Quantization
```python
from transformers import AutoModelForCausalLM, BitsAndBytesConfig

bnb_config = BitsAndBytesConfig(
    load_in_4bit=True,
    bnb_4bit_quant_type="nf4",
    bnb_4bit_compute_dtype="float16",
    bnb_4bit_use_double_quant=True,
)

model = AutoModelForCausalLM.from_pretrained(
    "meta-llama/Llama-2-7b-hf",
    quantization_config=bnb_config,
    device_map="auto",
)
```

### GGUF Conversion
```bash
# Convert HuggingFace model to GGUF
python convert_hf_to_gguf.py ./model_dir --outtype f16 --outfile model.f16.gguf

# Quantize GGUF model
./quantize model.f16.gguf model.Q4_K_M.gguf Q4_K_M

# Common quantization levels
# Q4_K_M  — best balance of size/quality (recommended)
# Q5_K_M  — slightly better quality, larger
# Q8_0    — near-original quality, ~2x Q4 size
# Q2_K    — smallest, noticeable quality loss
```

## Serving Frameworks
### vLLM
```bash
# Install
pip install vllm

# Start OpenAI-compatible server
python -m vllm.entrypoints.openai.api_server \
    --model meta-llama/Llama-2-7b-chat-hf \
    --tensor-parallel-size 2 \
    --max-model-len 4096 \
    --gpu-memory-utilization 0.9

# Query the server
curl http://localhost:8000/v1/completions \
    -H "Content-Type: application/json" \
    -d '{"model": "meta-llama/Llama-2-7b-chat-hf",
         "prompt": "Explain quantum computing:",
         "max_tokens": 256, "temperature": 0.7}'
```

### Ollama
```bash
# Pull and run a model
ollama pull llama2
ollama run llama2 "What is machine learning?"

# Run with specific parameters
ollama run llama2 --verbose

# List downloaded models
ollama list

# API endpoint
curl http://localhost:11434/api/generate -d '{
  "model": "llama2",
  "prompt": "Explain transformers",
  "stream": false
}'

# Create custom model with Modelfile
cat > Modelfile <<EOF
FROM llama2
PARAMETER temperature 0.7
PARAMETER num_ctx 4096
SYSTEM "You are a helpful coding assistant."
EOF
ollama create mymodel -f Modelfile
```

### Text Generation Inference (TGI)
```bash
docker run --gpus all --shm-size 1g \
    -p 8080:80 \
    -v $PWD/data:/data \
    ghcr.io/huggingface/text-generation-inference:latest \
    --model-id meta-llama/Llama-2-7b-chat-hf \
    --max-input-length 4096 \
    --max-total-tokens 8192 \
    --quantize gptq
```

## Sampling Parameters
### Temperature, Top-p, Top-k
```python
from transformers import AutoModelForCausalLM, AutoTokenizer

model = AutoModelForCausalLM.from_pretrained("gpt2")
tokenizer = AutoTokenizer.from_pretrained("gpt2")
inputs = tokenizer("The future of AI is", return_tensors="pt")

# Greedy decoding (deterministic)
output = model.generate(**inputs, max_new_tokens=50, do_sample=False)

# Temperature sampling (higher = more random)
output = model.generate(**inputs, max_new_tokens=50,
                         do_sample=True, temperature=0.7)

# Top-p (nucleus) sampling
output = model.generate(**inputs, max_new_tokens=50,
                         do_sample=True, top_p=0.9)

# Top-k sampling
output = model.generate(**inputs, max_new_tokens=50,
                         do_sample=True, top_k=50)

# Combined strategy (recommended)
output = model.generate(**inputs, max_new_tokens=50,
                         do_sample=True, temperature=0.7,
                         top_p=0.9, top_k=50,
                         repetition_penalty=1.1)
```

## KV Cache and Batching
### KV Cache Concept
```
# Without KV cache: recompute attention for all tokens each step
# Step 1: process [A]           -> 1 attention computation
# Step 2: process [A, B]        -> 2 attention computations
# Step 3: process [A, B, C]     -> 3 attention computations
# Total: O(n^2) computations

# With KV cache: cache K,V from previous steps
# Step 1: process [A], cache K1,V1
# Step 2: process [B] with cached K1,V1 -> 1 new computation
# Step 3: process [C] with cached K1..2,V1..2 -> 1 new computation
# Total: O(n) computations

# KV cache memory estimate:
# Memory = 2 * num_layers * hidden_size * seq_len * batch_size * dtype_bytes
# LLaMA-2 7B, 4096 ctx, FP16:
# = 2 * 32 * 4096 * 4096 * 1 * 2 = ~2 GB per sequence
```

### Continuous Batching
```python
# vLLM handles continuous batching automatically
# Key parameters:
# --max-num-batched-tokens  Maximum tokens in a batch
# --max-num-seqs            Maximum concurrent sequences
# --block-size              KV cache block size (default: 16)

# PagedAttention in vLLM manages KV cache like virtual memory
# Reduces memory waste from 60-80% to near 0%
```

## Model Cards
### Reading Model Cards
```python
from huggingface_hub import ModelCard

card = ModelCard.load("meta-llama/Llama-2-7b-hf")
print(card.content[:500])

# Check model config
from transformers import AutoConfig
config = AutoConfig.from_pretrained("meta-llama/Llama-2-7b-hf")
print(f"Hidden size: {config.hidden_size}")
print(f"Num layers: {config.num_hidden_layers}")
print(f"Num heads: {config.num_attention_heads}")
print(f"Vocab size: {config.vocab_size}")
print(f"Max position: {config.max_position_embeddings}")
```

## Tips
- Always count tokens before sending to API -- context limits are hard cutoffs, not soft suggestions
- Q4_K_M quantization is the sweet spot for most GGUF deployments: ~4.5 bits effective, minimal quality loss
- Use vLLM for production GPU serving -- PagedAttention gives 2-4x throughput over naive batching
- Ollama is the fastest path from zero to local LLM: one command to download and run
- Set temperature to 0 for deterministic tasks (code, math, extraction) and 0.7-1.0 for creative tasks
- KV cache memory scales linearly with context length -- a 128K context model needs 64x more cache than 2K
- GPTQ and AWQ quantization require GPU; GGUF works on CPU with optional GPU offloading
- Monitor VRAM usage: a 7B FP16 model needs ~14GB, 4-bit needs ~4GB, 8-bit needs ~7GB
- Use repetition_penalty (1.05-1.2) to reduce repetitive outputs without hurting coherence
- Always verify tokenizer compatibility -- using the wrong tokenizer produces garbage outputs
- Batch prompts when possible: throughput scales much better than latency with batching
- Check model licenses before commercial use: LLaMA has restrictions, Mistral/Mixtral are Apache 2.0

## See Also
- transformers, prompt-engineering, quantization, lora, llama, rag

## References
- [Attention Is All You Need (Vaswani et al., 2017)](https://arxiv.org/abs/1706.03762)
- [vLLM: Easy, Fast, and Cheap LLM Serving](https://docs.vllm.ai/en/latest/)
- [Ollama Documentation](https://ollama.com/)
- [HuggingFace Transformers Docs](https://huggingface.co/docs/transformers/)
- [GGUF Format Specification](https://github.com/ggerganov/ggml/blob/master/docs/gguf.md)
- [tiktoken Tokenizer](https://github.com/openai/tiktoken)
