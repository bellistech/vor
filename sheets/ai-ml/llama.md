# LLaMA (Meta's Large Language Model Architecture)

A practical guide to Meta's LLaMA model family covering model sizes and capabilities across generations, llama.cpp and GGUF format for local inference, fine-tuning workflows with Axolotl and Unsloth, deployment via Ollama and vLLM, context extension with RoPE scaling, chat templates, and licensing.

## LLaMA Family Overview
### Model Generations
```
| Generation | Models           | Context | Key Features                     |
|-----------|------------------|---------|----------------------------------|
| LLaMA 1   | 7B/13B/33B/65B   | 2048    | First open-weight LLM at scale   |
| LLaMA 2   | 7B/13B/70B       | 4096    | Chat fine-tuning, RLHF           |
| LLaMA 3   | 8B/70B           | 8192    | GQA, larger vocab (128K tokens)  |
| LLaMA 3.1 | 8B/70B/405B      | 128K    | Tool use, multilingual, longest ctx|
| LLaMA 3.2 | 1B/3B/11B/90B    | 128K    | Vision models, edge deployment   |
```

### Architecture Details
```
| Component        | LLaMA 2 7B | LLaMA 2 70B | LLaMA 3 8B | LLaMA 3.1 405B |
|-----------------|------------|-------------|------------|----------------|
| Hidden size      | 4096       | 8192        | 4096       | 16384          |
| Layers           | 32         | 80          | 32         | 126            |
| Attention heads  | 32         | 64          | 32         | 128            |
| KV heads (GQA)   | 32 (MHA)   | 8           | 8          | 8              |
| FFN dimension    | 11008      | 28672       | 14336      | 53248          |
| Vocab size       | 32000      | 32000       | 128256     | 128256         |
| Positional enc.  | RoPE       | RoPE        | RoPE       | RoPE           |
| Norm             | RMSNorm    | RMSNorm     | RMSNorm    | RMSNorm        |
| Activation       | SwiGLU     | SwiGLU      | SwiGLU     | SwiGLU         |
```

## llama.cpp and GGUF
### Building llama.cpp
```bash
# Clone and build
git clone https://github.com/ggerganov/llama.cpp
cd llama.cpp

# Build with CUDA support
cmake -B build -DGGML_CUDA=ON
cmake --build build --config Release -j$(nproc)

# Build with Metal (macOS)
cmake -B build -DGGML_METAL=ON
cmake --build build --config Release -j$(sysctl -n hw.ncpu)

# Build CPU-only
cmake -B build
cmake --build build --config Release -j$(nproc)
```

### Running Inference
```bash
# Interactive chat
./build/bin/llama-cli \
    -m models/llama-2-7b-chat.Q4_K_M.gguf \
    -n 512 \
    --ctx-size 4096 \
    --temp 0.7 \
    --top-p 0.9 \
    --repeat-penalty 1.1 \
    -i -r "User:" \
    --color

# Server mode (OpenAI-compatible API)
./build/bin/llama-server \
    -m models/llama-2-7b-chat.Q4_K_M.gguf \
    --ctx-size 4096 \
    --host 0.0.0.0 \
    --port 8080 \
    -ngl 35          # Number of layers to offload to GPU

# Batch processing
./build/bin/llama-cli \
    -m models/llama-2-7b.Q4_K_M.gguf \
    -f prompts.txt \
    -n 256 \
    --log-disable
```

### GGUF Quantization
```bash
# Convert HuggingFace to GGUF
python convert_hf_to_gguf.py \
    /path/to/hf/model \
    --outtype f16 \
    --outfile model-f16.gguf

# Quantize
./build/bin/llama-quantize model-f16.gguf model-Q4_K_M.gguf Q4_K_M

# Quantization options and quality/size tradeoff:
# Q2_K   — 2.6 bpw, smallest, significant quality loss
# Q4_K_M — 4.6 bpw, recommended default
# Q5_K_M  — 5.7 bpw, good quality, larger
# Q6_K    — 6.6 bpw, near-original quality
# Q8_0    — 8.5 bpw, minimal quality loss
# F16     — 16 bpw, original quality
```

## Ollama Deployment
### Basic Usage
```bash
# Pull official models
ollama pull llama3.1
ollama pull llama3.1:70b
ollama pull llama3.1:8b-instruct-q4_K_M

# Run interactively
ollama run llama3.1 "Explain the theory of relativity"

# List models
ollama list

# Show model details
ollama show llama3.1

# Remove a model
ollama rm llama3.1:70b
```

### Custom Modelfile
```dockerfile
# Modelfile for custom LLaMA setup
FROM llama3.1

# Set parameters
PARAMETER temperature 0.7
PARAMETER top_p 0.9
PARAMETER top_k 40
PARAMETER num_ctx 8192
PARAMETER repeat_penalty 1.1
PARAMETER num_predict 512

# System prompt
SYSTEM """You are a helpful technical assistant specializing in
software engineering. You provide concise, accurate answers
with code examples when relevant."""

# Chat template override (if needed)
TEMPLATE """{{ if .System }}<|begin_of_text|><|start_header_id|>system<|end_header_id|>

{{ .System }}<|eot_id|>{{ end }}{{ if .Prompt }}<|start_header_id|>user<|end_header_id|>

{{ .Prompt }}<|eot_id|>{{ end }}<|start_header_id|>assistant<|end_header_id|>

{{ .Response }}<|eot_id|>"""
```

```bash
# Create and run custom model
ollama create my-assistant -f Modelfile
ollama run my-assistant
```

### Ollama API
```bash
# Generate completion
curl http://localhost:11434/api/generate -d '{
  "model": "llama3.1", "prompt": "What is quantum computing?",
  "stream": false, "options": {"temperature": 0.7, "num_ctx": 4096}
}'

# Chat completion
curl http://localhost:11434/api/chat -d '{
  "model": "llama3.1", "stream": false,
  "messages": [{"role": "user", "content": "Hello!"}]
}'
```

## vLLM Deployment
```bash
# Basic server
python -m vllm.entrypoints.openai.api_server \
    --model meta-llama/Meta-Llama-3.1-8B-Instruct \
    --max-model-len 8192 --gpu-memory-utilization 0.9

# Multi-GPU with tensor parallelism
python -m vllm.entrypoints.openai.api_server \
    --model meta-llama/Meta-Llama-3.1-70B-Instruct \
    --tensor-parallel-size 4 --max-model-len 32768 --dtype bfloat16
```

## Chat Templates
### LLaMA 2 Chat Format
```
<s>[INST] <<SYS>>
You are a helpful assistant.
<</SYS>>

User message here [/INST] Assistant response here </s>
<s>[INST] Follow-up message [/INST]
```

### LLaMA 3 Chat Format
```
<|begin_of_text|><|start_header_id|>system<|end_header_id|>

You are a helpful assistant.<|eot_id|><|start_header_id|>user<|end_header_id|>

User message here<|eot_id|><|start_header_id|>assistant<|end_header_id|>

Assistant response here<|eot_id|>
```

### Applying Chat Template in Python
```python
from transformers import AutoTokenizer
tokenizer = AutoTokenizer.from_pretrained("meta-llama/Meta-Llama-3.1-8B-Instruct")

messages = [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "What is the capital of France?"},
]
formatted = tokenizer.apply_chat_template(messages, tokenize=False, add_generation_prompt=True)
```

## Fine-Tuning
### Axolotl
```yaml
# axolotl_config.yml
base_model: meta-llama/Meta-Llama-3.1-8B-Instruct
model_type: LlamaForCausalLM

load_in_4bit: true
adapter: qlora
lora_r: 16
lora_alpha: 32
lora_target_modules:
  - q_proj
  - v_proj
  - k_proj
  - o_proj

datasets:
  - path: ./train_data.jsonl
    type: sharegpt

sequence_len: 4096
micro_batch_size: 2
gradient_accumulation_steps: 8
num_epochs: 3
learning_rate: 2e-4
optimizer: adamw_torch
lr_scheduler: cosine
warmup_ratio: 0.1

bf16: true
gradient_checkpointing: true
flash_attention: true
```

```bash
accelerate launch -m axolotl.cli.train axolotl_config.yml
```

### Unsloth (2x Faster)
```python
from unsloth import FastLanguageModel

model, tokenizer = FastLanguageModel.from_pretrained(
    model_name="unsloth/Meta-Llama-3.1-8B-Instruct",
    max_seq_length=4096,
    load_in_4bit=True,
)

model = FastLanguageModel.get_peft_model(
    model,
    r=16,
    lora_alpha=32,
    target_modules=["q_proj", "k_proj", "v_proj", "o_proj",
                     "gate_proj", "up_proj", "down_proj"],
    lora_dropout=0,
    bias="none",
)

# Train with SFTTrainer as usual
from trl import SFTTrainer, SFTConfig

trainer = SFTTrainer(
    model=model,
    tokenizer=tokenizer,
    train_dataset=dataset,
    args=SFTConfig(
        per_device_train_batch_size=2,
        gradient_accumulation_steps=4,
        num_train_epochs=3,
        learning_rate=2e-4,
        bf16=True,
        output_dir="./output",
    ),
)
trainer.train()

# Save as GGUF directly
model.save_pretrained_gguf("./model_gguf", tokenizer, quantization_method="q4_k_m")
```

## Context Extension (RoPE Scaling)
### Scaling Methods
```python
from transformers import AutoModelForCausalLM, AutoConfig

config = AutoConfig.from_pretrained("meta-llama/Llama-2-7b-hf")

# Dynamic NTK-aware scaling (4K -> 16K)
config.rope_scaling = {"type": "dynamic", "factor": 4.0}

# YaRN scaling (4K -> 32K)
config.rope_scaling = {"type": "yarn", "factor": 8.0,
                        "original_max_position_embeddings": 4096}

model = AutoModelForCausalLM.from_pretrained(
    "meta-llama/Llama-2-7b-hf", config=config, device_map="auto")
```

## License Terms
```
| Model      | License                    | Commercial Use | Restrictions              |
|-----------|----------------------------|----------------|---------------------------|
| LLaMA 1   | Research-only              | No             | Academic/research only     |
| LLaMA 2   | LLaMA 2 Community License  | Yes            | 700M MAU limit             |
| LLaMA 3   | Meta LLaMA 3 License       | Yes            | 700M MAU limit             |
| LLaMA 3.1 | LLaMA 3.1 Community License| Yes            | 700M MAU limit            |
```

## Tips
- Use LLaMA 3.1 8B as the default starting point for most tasks -- best quality-to-size ratio in the family
- Q4_K_M quantization via llama.cpp is the sweet spot: fits 7B/8B models in 5GB with minimal quality loss
- Ollama is the fastest path to local LLaMA: single command install, automatic GGUF management
- For multi-GPU serving, vLLM with tensor parallelism gives the best throughput per dollar
- Use Unsloth for fine-tuning when possible -- 2x faster training and direct GGUF export
- Apply the correct chat template for your model version -- LLaMA 2 and 3 have different formats
- RoPE scaling can extend context cheaply but quality degrades beyond 4x the original length without fine-tuning
- Flash Attention 2 reduces memory usage and speeds up inference -- always enable it when available
- The 405B model requires at least 4x A100 80GB (quantized) or 8x (FP16) for inference
- Check the LLaMA license before deploying commercially: 700M monthly active user cap applies
- Always use -Instruct variants for chat/instruction tasks -- base models need careful prompting

## See Also
- llm-fundamentals, lora, transformers, vector-databases, prompt-engineering

## References
- [LLaMA: Open and Efficient Foundation Language Models (Touvron et al., 2023)](https://arxiv.org/abs/2302.13971)
- [LLaMA 2: Open Foundation and Fine-Tuned Chat Models](https://arxiv.org/abs/2307.09288)
- [The LLaMA 3 Herd of Models (Meta, 2024)](https://arxiv.org/abs/2407.21783)
- [llama.cpp GitHub Repository](https://github.com/ggerganov/llama.cpp)
- [Ollama Documentation](https://ollama.com/)
- [Unsloth GitHub Repository](https://github.com/unslothai/unsloth)
- [Axolotl Fine-Tuning Framework](https://github.com/OpenAccess-AI-Collective/axolotl)
