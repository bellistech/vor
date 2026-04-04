# Transformers (The Transformer Architecture and HuggingFace Ecosystem)

A practical reference covering the transformer architecture from self-attention to full model stacks, positional encoding variants, encoder-decoder vs decoder-only designs, layer normalization, residual connections, and the HuggingFace Transformers library for loading, fine-tuning, and running inference.

## Architecture Overview
### Full Transformer Block
```
Input Embeddings + Positional Encoding
        |
   +----v----+
   | Multi-Head Self-Attention |
   +----+----+
        |  + Residual Connection
        v
   Layer Normalization
        |
   +----v----+
   | Feed-Forward Network (MLP) |
   +----+----+
        |  + Residual Connection
        v
   Layer Normalization
        |
   (Repeat N times)
        |
   Output / LM Head
```

### Encoder-Decoder vs Decoder-Only
```
| Architecture     | Examples           | Attention Type        | Use Case          |
|-----------------|--------------------|-----------------------|-------------------|
| Encoder-Decoder  | T5, BART, mBART    | Bidirectional + Cross | Translation, Summ |
| Encoder-Only     | BERT, RoBERTa      | Bidirectional         | Classification    |
| Decoder-Only     | GPT, LLaMA, Mistral| Causal (masked)       | Generation        |

Decoder-only with causal mask:
  Token at position i can only attend to positions [0, 1, ..., i]
  Implemented via upper-triangular mask filled with -inf

Encoder-decoder:
  Encoder sees all tokens bidirectionally
  Decoder attends causally to itself + cross-attends to encoder
```

## Self-Attention
### Scaled Dot-Product Attention
```python
import torch
import torch.nn.functional as F

def scaled_dot_product_attention(Q, K, V, mask=None):
    """
    Q, K, V: (batch, heads, seq_len, d_k)
    mask: (batch, 1, 1, seq_len) or (batch, 1, seq_len, seq_len)
    """
    d_k = Q.size(-1)
    scores = torch.matmul(Q, K.transpose(-2, -1)) / (d_k ** 0.5)

    if mask is not None:
        scores = scores.masked_fill(mask == 0, float('-inf'))

    attention_weights = F.softmax(scores, dim=-1)
    output = torch.matmul(attention_weights, V)
    return output, attention_weights
```

### Multi-Head Attention
```python
import torch.nn as nn

class MultiHeadAttention(nn.Module):
    def __init__(self, d_model, num_heads):
        super().__init__()
        self.d_model = d_model
        self.num_heads = num_heads
        self.d_k = d_model // num_heads

        self.W_q = nn.Linear(d_model, d_model)
        self.W_k = nn.Linear(d_model, d_model)
        self.W_v = nn.Linear(d_model, d_model)
        self.W_o = nn.Linear(d_model, d_model)

    def forward(self, Q, K, V, mask=None):
        batch_size = Q.size(0)

        # Project and reshape: (batch, seq, d_model) -> (batch, heads, seq, d_k)
        Q = self.W_q(Q).view(batch_size, -1, self.num_heads, self.d_k).transpose(1, 2)
        K = self.W_k(K).view(batch_size, -1, self.num_heads, self.d_k).transpose(1, 2)
        V = self.W_v(V).view(batch_size, -1, self.num_heads, self.d_k).transpose(1, 2)

        # Attention
        attn_output, _ = scaled_dot_product_attention(Q, K, V, mask)

        # Concatenate heads: (batch, heads, seq, d_k) -> (batch, seq, d_model)
        attn_output = attn_output.transpose(1, 2).contiguous().view(
            batch_size, -1, self.d_model
        )

        return self.W_o(attn_output)
```

### Causal Mask (Decoder)
```python
def create_causal_mask(seq_len):
    """Lower-triangular mask for autoregressive decoding."""
    mask = torch.tril(torch.ones(seq_len, seq_len))
    return mask.unsqueeze(0).unsqueeze(0)  # (1, 1, seq, seq)

# Example for seq_len=4:
# [[1, 0, 0, 0],
#  [1, 1, 0, 0],
#  [1, 1, 1, 0],
#  [1, 1, 1, 1]]
```

## Positional Encoding
### Sinusoidal (Original Transformer)
```python
import math

def sinusoidal_position_encoding(seq_len, d_model):
    """Fixed sinusoidal positional encoding from Vaswani et al."""
    pe = torch.zeros(seq_len, d_model)
    position = torch.arange(0, seq_len).unsqueeze(1).float()
    div_term = torch.exp(
        torch.arange(0, d_model, 2).float() * (-math.log(10000.0) / d_model)
    )
    pe[:, 0::2] = torch.sin(position * div_term)  # Even dimensions
    pe[:, 1::2] = torch.cos(position * div_term)  # Odd dimensions
    return pe
```

### RoPE (Rotary Position Embedding)
```python
def apply_rope(x, freqs_cos, freqs_sin):
    """Apply Rotary Position Embedding to queries or keys."""
    # x: (batch, heads, seq_len, d_k)
    x_reshape = x.float().reshape(*x.shape[:-1], -1, 2)
    x1, x2 = x_reshape[..., 0], x_reshape[..., 1]

    # Rotation: (x1 + ix2) * (cos + i*sin)
    out1 = x1 * freqs_cos - x2 * freqs_sin
    out2 = x1 * freqs_sin + x2 * freqs_cos

    out = torch.stack([out1, out2], dim=-1).flatten(-2)
    return out.type_as(x)
```

### ALiBi (Attention with Linear Biases)
```python
def alibi_bias(num_heads, seq_len):
    """Compute ALiBi bias matrix (no positional embedding needed)."""
    # Slopes: geometric sequence from 2^(-8/n) to 2^(-8)
    slopes = torch.tensor([2 ** (-8 * i / num_heads) for i in range(1, num_heads + 1)])

    # Distance matrix
    positions = torch.arange(seq_len)
    distance = positions.unsqueeze(0) - positions.unsqueeze(1)  # (seq, seq)
    distance = distance.abs().neg()

    # Bias: slopes * distance
    bias = slopes.unsqueeze(1).unsqueeze(1) * distance.unsqueeze(0)
    return bias  # (heads, seq, seq) -- add to attention scores
```

### Comparison
```
| Encoding    | Type      | Extrapolation | Memory  | Models              |
|------------|-----------|---------------|---------|---------------------|
| Sinusoidal  | Absolute  | Poor          | None    | Original Transformer|
| Learned     | Absolute  | Poor          | O(L*d)  | GPT-2, BERT         |
| RoPE        | Relative  | Good*         | None    | LLaMA, Mistral      |
| ALiBi       | Relative  | Excellent     | None    | BLOOM, MPT           |

* RoPE extrapolation improves with NTK/YaRN scaling
```

## Layer Normalization and Residual Connections
### Pre-Norm vs Post-Norm
```python
# Post-Norm (original Transformer, BERT)
x = x + self_attention(x)
x = layer_norm(x)
x = x + feed_forward(x)
x = layer_norm(x)

# Pre-Norm (GPT-2, LLaMA -- more stable training)
x = x + self_attention(layer_norm(x))
x = x + feed_forward(layer_norm(x))
```

### RMSNorm (LLaMA)
```python
class RMSNorm(nn.Module):
    def __init__(self, d_model, eps=1e-6):
        super().__init__()
        self.weight = nn.Parameter(torch.ones(d_model))
        self.eps = eps

    def forward(self, x):
        rms = torch.sqrt(x.pow(2).mean(-1, keepdim=True) + self.eps)
        return x / rms * self.weight
```

## Feed-Forward Network
### Standard FFN
```python
class FeedForward(nn.Module):
    def __init__(self, d_model, d_ff, dropout=0.1):
        super().__init__()
        self.linear1 = nn.Linear(d_model, d_ff)        # Expand
        self.linear2 = nn.Linear(d_ff, d_model)         # Contract
        self.dropout = nn.Dropout(dropout)
        self.activation = nn.GELU()

    def forward(self, x):
        return self.linear2(self.dropout(self.activation(self.linear1(x))))

# Typical d_ff = 4 * d_model
```

### SwiGLU FFN (LLaMA/Mistral)
```python
class SwiGLUFFN(nn.Module):
    def __init__(self, d_model, d_ff):
        super().__init__()
        self.gate_proj = nn.Linear(d_model, d_ff, bias=False)
        self.up_proj = nn.Linear(d_model, d_ff, bias=False)
        self.down_proj = nn.Linear(d_ff, d_model, bias=False)

    def forward(self, x):
        gate = F.silu(self.gate_proj(x))   # SiLU = Swish
        up = self.up_proj(x)
        return self.down_proj(gate * up)
```

## HuggingFace Transformers Library
### Model Loading
```python
from transformers import AutoModelForCausalLM, AutoTokenizer, AutoConfig

# Load model and tokenizer
model_name = "meta-llama/Llama-2-7b-hf"
tokenizer = AutoTokenizer.from_pretrained(model_name)
model = AutoModelForCausalLM.from_pretrained(
    model_name,
    torch_dtype=torch.float16,
    device_map="auto",             # Automatic GPU placement
    attn_implementation="flash_attention_2",
)

# Inspect config
config = AutoConfig.from_pretrained(model_name)
print(f"Layers: {config.num_hidden_layers}")
print(f"Heads: {config.num_attention_heads}")
print(f"Hidden: {config.hidden_size}")
```

### Text Generation
```python
inputs = tokenizer("The transformer architecture", return_tensors="pt").to(model.device)

output = model.generate(
    **inputs,
    max_new_tokens=200,
    temperature=0.7,
    top_p=0.9,
    do_sample=True,
    repetition_penalty=1.1,
    num_return_sequences=1,
)

text = tokenizer.decode(output[0], skip_special_tokens=True)
```

### Pipeline API
```python
from transformers import pipeline

# Text generation
generator = pipeline("text-generation", model="gpt2", device=0)
result = generator("The future of AI", max_new_tokens=100, temperature=0.8)

# Feature extraction (embeddings)
extractor = pipeline("feature-extraction", model="bert-base-uncased")
embeddings = extractor("Hello world")

# Text classification
classifier = pipeline("text-classification", model="distilbert-base-uncased-finetuned-sst-2-english")
result = classifier("This movie was fantastic!")
```

### Flash Attention
```python
# Enable Flash Attention 2 (requires flash-attn package)
model = AutoModelForCausalLM.from_pretrained(
    model_name,
    torch_dtype=torch.bfloat16,
    attn_implementation="flash_attention_2",
)

# SDPA (Scaled Dot-Product Attention) -- PyTorch native, no extra install
model = AutoModelForCausalLM.from_pretrained(
    model_name,
    torch_dtype=torch.float16,
    attn_implementation="sdpa",
)

# Memory and speed comparison (approximate, 4K context):
# Standard attention:   O(n^2) memory, baseline speed
# SDPA:                 O(n) memory*, ~1.5x speed
# Flash Attention 2:    O(n) memory, ~2x speed
```

## Tips
- Always use Pre-Norm (layer norm before attention) for training stability -- Post-Norm can diverge
- Flash Attention 2 should be your default -- it is faster and uses less memory with no quality tradeoff
- Use device_map="auto" to automatically split large models across multiple GPUs
- RoPE is the dominant positional encoding: better length generalization than absolute embeddings
- When implementing custom attention, always scale by 1/sqrt(d_k) -- forgetting this causes gradient issues
- SwiGLU FFN gives ~1-3% better quality than standard GELU FFN at the same parameter count
- For classification tasks use encoder-only (BERT-style); for generation use decoder-only (GPT/LLaMA-style)
- The pipeline API is excellent for prototyping but use raw model.generate() for production control
- Causal masking is critical for decoder-only models -- bidirectional attention causes data leakage in generation
- Use bfloat16 over float16 when your hardware supports it (A100, H100) -- better numerical stability
- Gradient checkpointing trades ~20% speed for ~50% memory savings -- enable for large models or long contexts

## See Also
- llm-fundamentals, llama, lora, prompt-engineering, rag

## References
- [Attention Is All You Need (Vaswani et al., 2017)](https://arxiv.org/abs/1706.03762)
- [HuggingFace Transformers Documentation](https://huggingface.co/docs/transformers/)
- [FlashAttention: Fast and Memory-Efficient Attention](https://arxiv.org/abs/2205.14135)
- [RoFormer: Enhanced Transformer with Rotary Position Embedding](https://arxiv.org/abs/2104.09864)
- [ALiBi: Train Short, Test Long](https://arxiv.org/abs/2108.12409)
- [GLU Variants Improve Transformer (Shazeer, 2020)](https://arxiv.org/abs/2002.05202)
