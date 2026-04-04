# LoRA (Low-Rank Adaptation of Large Language Models)

A practical guide to parameter-efficient fine-tuning with LoRA and QLoRA, covering rank selection, alpha scaling, target module configuration, adapter merging, and real-world training workflows using the PEFT library and HuggingFace ecosystem.

## LoRA Core Concept
### How It Works
```
Standard Fine-Tuning:
  W_new = W_original + delta_W       (delta_W is full-rank, same size as W)
  Parameters updated: d_in * d_out   (e.g., 4096 * 4096 = 16.7M per layer)

LoRA Fine-Tuning:
  W_new = W_original + B * A          (B: d_out x r, A: r x d_in)
  Parameters updated: r * (d_in + d_out)  (e.g., 8 * (4096+4096) = 65K per layer)

  W_original is FROZEN (no gradients)
  Only A and B are trained

  Rank r << min(d_in, d_out)
  Typical r: 4, 8, 16, 32, 64
```

### Parameter Efficiency
```
| Model    | Full FT Params | LoRA r=8 | LoRA r=16 | LoRA r=64 | % of Full |
|----------|---------------|----------|-----------|-----------|-----------|
| 7B       | 6.7B          | 4.2M     | 8.4M      | 33.6M     | 0.06-0.5% |
| 13B      | 13.0B         | 6.6M     | 13.1M     | 52.4M     | 0.05-0.4% |
| 70B      | 68.9B         | 13.1M    | 26.2M     | 104.9M    | 0.02-0.15%|
```

## PEFT Library Usage
### Basic LoRA Setup
```python
from peft import LoraConfig, get_peft_model, TaskType
from transformers import AutoModelForCausalLM, AutoTokenizer

# Load base model
model = AutoModelForCausalLM.from_pretrained(
    "meta-llama/Llama-2-7b-hf",
    torch_dtype=torch.float16,
    device_map="auto",
)

# Configure LoRA
lora_config = LoraConfig(
    r=16,                          # Rank
    lora_alpha=32,                 # Alpha scaling factor
    lora_dropout=0.05,             # Dropout on LoRA layers
    target_modules=[               # Which layers to adapt
        "q_proj", "v_proj",        # Minimum recommended
        "k_proj", "o_proj",        # Additional attention
        "gate_proj", "up_proj",    # MLP layers
        "down_proj",
    ],
    bias="none",                   # Don't train biases
    task_type=TaskType.CAUSAL_LM,
)

# Apply LoRA
model = get_peft_model(model, lora_config)
model.print_trainable_parameters()
# trainable params: 13,107,200 || all params: 6,751,944,704 || trainable%: 0.194
```

### Alpha and Rank Relationship
```python
# The effective LoRA scaling factor is alpha/r
# This means the adaptation magnitude is:
#   delta_W = (alpha/r) * B * A

# Common configurations:
# r=8,  alpha=16  -> scale = 2.0 (moderate adaptation)
# r=16, alpha=32  -> scale = 2.0 (same scale, more capacity)
# r=16, alpha=16  -> scale = 1.0 (conservative)
# r=64, alpha=128 -> scale = 2.0 (high capacity)

# Rule of thumb: set alpha = 2 * r for a starting point
# Then adjust based on training loss and validation performance
```

## QLoRA (4-bit + LoRA)
### Setup
```python
from transformers import AutoModelForCausalLM, BitsAndBytesConfig
from peft import LoraConfig, get_peft_model, prepare_model_for_kbit_training
import torch

# 4-bit quantization config
bnb_config = BitsAndBytesConfig(
    load_in_4bit=True,
    bnb_4bit_quant_type="nf4",           # NormalFloat4
    bnb_4bit_compute_dtype=torch.bfloat16,
    bnb_4bit_use_double_quant=True,       # Nested quantization
)

# Load quantized model
model = AutoModelForCausalLM.from_pretrained(
    "meta-llama/Llama-2-7b-hf",
    quantization_config=bnb_config,
    device_map="auto",
)

# Prepare for k-bit training (handles gradient checkpointing etc.)
model = prepare_model_for_kbit_training(model)

# Apply LoRA on top of quantized model
lora_config = LoraConfig(
    r=16,
    lora_alpha=32,
    lora_dropout=0.05,
    target_modules=["q_proj", "v_proj", "k_proj", "o_proj",
                     "gate_proj", "up_proj", "down_proj"],
    bias="none",
    task_type="CAUSAL_LM",
)

model = get_peft_model(model, lora_config)
```

### Memory Requirements
```
| Model Size | Full FP16 | QLoRA 4-bit | Training VRAM |
|-----------|-----------|-------------|---------------|
| 7B        | ~14 GB    | ~4 GB       | ~7 GB         |
| 13B       | ~26 GB    | ~8 GB       | ~12 GB        |
| 70B       | ~140 GB   | ~36 GB      | ~48 GB        |

# QLoRA enables 7B fine-tuning on a single 24GB GPU
# and 70B fine-tuning on a single 80GB A100
```

## Training Workflow
### Complete Training Script
```python
from transformers import (
    AutoModelForCausalLM, AutoTokenizer,
    TrainingArguments, Trainer, DataCollatorForLanguageModeling,
)
from peft import LoraConfig, get_peft_model
from datasets import load_dataset

# Load tokenizer
tokenizer = AutoTokenizer.from_pretrained("meta-llama/Llama-2-7b-hf")
tokenizer.pad_token = tokenizer.eos_token
tokenizer.padding_side = "right"

# Load and prepare dataset
dataset = load_dataset("json", data_files="train.jsonl")

def format_sample(sample):
    text = f"### Instruction:\n{sample['instruction']}\n\n### Response:\n{sample['output']}"
    return tokenizer(text, truncation=True, max_length=2048, padding="max_length")

tokenized = dataset.map(format_sample, remove_columns=dataset["train"].column_names)

# Training arguments
training_args = TrainingArguments(
    output_dir="./lora_output",
    num_train_epochs=3,
    per_device_train_batch_size=4,
    gradient_accumulation_steps=4,
    learning_rate=2e-4,
    lr_scheduler_type="cosine",
    warmup_ratio=0.1,
    bf16=True,
    logging_steps=10,
    save_strategy="steps",
    save_steps=100,
    eval_strategy="steps",
    eval_steps=100,
    save_total_limit=3,
    gradient_checkpointing=True,
    optim="adamw_torch",
    report_to="wandb",
)

# Data collator
data_collator = DataCollatorForLanguageModeling(
    tokenizer=tokenizer,
    mlm=False,
)

# Train
trainer = Trainer(
    model=model,
    args=training_args,
    train_dataset=tokenized["train"],
    data_collator=data_collator,
)

trainer.train()
trainer.save_model("./lora_adapter")
```

### Using SFTTrainer (TRL)
```python
from trl import SFTTrainer, SFTConfig

sft_config = SFTConfig(
    output_dir="./lora_output",
    num_train_epochs=3,
    per_device_train_batch_size=4,
    gradient_accumulation_steps=4,
    learning_rate=2e-4,
    bf16=True,
    max_seq_length=2048,
    packing=True,  # Pack short sequences for efficiency
)

trainer = SFTTrainer(
    model=model,
    args=sft_config,
    train_dataset=dataset["train"],
    peft_config=lora_config,
)
trainer.train()
```

## Target Module Selection
### Finding Available Modules
```python
# List all linear layers in the model
from peft.utils import TRANSFORMERS_MODELS_TO_LORA_TARGET_MODULES_MAPPING

# Or inspect manually
for name, module in model.named_modules():
    if isinstance(module, torch.nn.Linear):
        print(name, module.in_features, module.out_features)

# Common target module sets:
# Minimal:     ["q_proj", "v_proj"]
# Attention:   ["q_proj", "v_proj", "k_proj", "o_proj"]
# Full:        ["q_proj", "v_proj", "k_proj", "o_proj",
#               "gate_proj", "up_proj", "down_proj"]
# All linear:  target_modules="all-linear"  (PEFT shortcut)
```

## Adapter Management
### Save and Load
```python
# Save adapter (only LoRA weights, ~50MB for r=16)
model.save_pretrained("./my_adapter")

# Load adapter onto base model
from peft import PeftModel

base_model = AutoModelForCausalLM.from_pretrained("meta-llama/Llama-2-7b-hf")
model = PeftModel.from_pretrained(base_model, "./my_adapter")
```

### Merge Adapter into Base Model
```python
# Merge LoRA weights into base model (no adapter overhead at inference)
merged_model = model.merge_and_unload()
merged_model.save_pretrained("./merged_model")
tokenizer.save_pretrained("./merged_model")

# Now the model runs at full speed without PEFT
```

### Multiple Adapters
```python
from peft import PeftModel

model = AutoModelForCausalLM.from_pretrained("meta-llama/Llama-2-7b-hf")

# Load first adapter
model = PeftModel.from_pretrained(model, "./adapter_coding", adapter_name="coding")

# Load second adapter
model.load_adapter("./adapter_writing", adapter_name="writing")

# Switch between adapters
model.set_adapter("coding")    # Use coding adapter
model.set_adapter("writing")   # Use writing adapter
```

## Rank Selection Guide
### Choosing r
```
| Use Case                | Recommended r | Alpha  | Notes                      |
|------------------------|---------------|--------|----------------------------|
| Simple classification   | 4-8           | 16     | Minimal capacity needed    |
| Instruction following   | 8-16          | 32     | Standard choice            |
| Domain adaptation       | 16-32         | 64     | More capacity for new vocab|
| Complex reasoning       | 32-64         | 128    | High capacity              |
| Code generation          | 16-32         | 64     | Code patterns are complex  |

# Higher r = more capacity but:
#   - More trainable parameters
#   - Higher memory usage
#   - Risk of overfitting on small datasets
# Start with r=16, alpha=32 and adjust
```

## Tips
- Start with r=16, alpha=32 and target only q_proj/v_proj before scaling up -- often this is sufficient
- QLoRA makes 7B fine-tuning possible on a single consumer GPU (RTX 3090/4090 with 24GB VRAM)
- Set learning rate to 2e-4 for LoRA (10x higher than full fine-tuning) since only small matrices are updated
- Always merge adapters before production deployment -- inference with separate adapters adds latency
- Use gradient checkpointing to trade compute for memory: halves VRAM at ~20% speed cost
- Multiple LoRA adapters can share a base model, enabling multi-task serving with adapter switching
- Watch for overfitting on small datasets (< 1000 samples): reduce r, increase dropout, or use early stopping
- Pack short sequences together with SFTTrainer's packing=True to maximize GPU utilization
- Validate that your target_modules match your model architecture -- LLaMA uses different names than GPT/Mistral
- Set bias="none" unless your task specifically needs bias adaptation (rare)
- Use bfloat16 compute dtype with QLoRA for stability -- float16 can cause NaN issues in some configurations

## See Also
- llm-fundamentals, raft, llama, transformers, quantization

## References
- [LoRA: Low-Rank Adaptation of Large Language Models (Hu et al., 2021)](https://arxiv.org/abs/2106.09685)
- [QLoRA: Efficient Finetuning of Quantized LLMs (Dettmers et al., 2023)](https://arxiv.org/abs/2305.14314)
- [PEFT Documentation](https://huggingface.co/docs/peft/)
- [TRL: Transformer Reinforcement Learning](https://huggingface.co/docs/trl/)
- [bitsandbytes Documentation](https://github.com/TimDettmers/bitsandbytes)
