# RAFT (Retrieval-Augmented Fine-Tuning)

A practical guide to RAFT, a domain adaptation technique that fine-tunes LLMs to reason over retrieved documents while resisting hallucination, combining chain-of-thought distillation with oracle/distractor training to produce models that excel at open-book domain-specific question answering.

## RAFT Overview
### Core Concept
```
Traditional RAG:       Frozen LLM + Retrieved Context -> Answer
Standard Fine-Tuning:  Domain LLM (no retrieval) -> Answer (often hallucinates)
RAFT:                  Fine-tuned LLM + Retrieved Context -> CoT -> Verified Answer

RAFT trains the model to:
1. Identify which retrieved documents are relevant (oracle) vs noise (distractor)
2. Extract and cite specific passages from oracle documents
3. Generate chain-of-thought reasoning grounded in the documents
4. Resist hallucinating when distractors are present
```

### RAFT vs RAG vs Fine-Tuning
```
| Approach          | Retrieval | Training Data        | Hallucination Risk | Domain Accuracy |
|-------------------|-----------|----------------------|--------------------|-----------------|
| RAG               | Yes       | None (frozen model)  | Medium             | Good            |
| Fine-Tuning       | No        | Domain Q&A pairs     | High               | Medium          |
| RAFT              | Yes       | Domain Q&A + CoT     | Low                | Best            |
| RAFT (closed-book)| No        | Same as RAFT         | Medium             | Good            |
```

## Training Data Preparation
### Dataset Format
```python
# RAFT training sample structure
{
    "question": "What is the maximum dosage of Drug X for adults?",
    "oracle_document": "Section 4.2: The recommended dose of Drug X for adults is 200mg twice daily. The maximum daily dose should not exceed 400mg...",
    "distractor_documents": [
        "Drug Y is indicated for treatment of...",
        "Clinical trials showed that Drug Z...",
        "The pharmacokinetics of Drug W..."
    ],
    "chain_of_thought": "The question asks about maximum dosage of Drug X for adults. Looking at the provided documents, Section 4.2 of the oracle document states: 'The maximum daily dose should not exceed 400mg.' Therefore, the maximum dosage is 400mg per day.",
    "answer": "The maximum dosage of Drug X for adults is 400mg per day (200mg twice daily)."
}
```

### Generating Training Data
```python
import json
from openai import OpenAI

client = OpenAI()

def generate_raft_sample(question, oracle_doc, distractor_docs):
    """Generate a RAFT training sample with CoT reasoning."""

    # Step 1: Generate chain-of-thought answer from oracle
    cot_prompt = f"""Given the following document, answer the question with
step-by-step reasoning. Quote specific passages from the document.

Document: {oracle_doc}

Question: {question}

Provide your answer in this format:
REASONING: <step-by-step reasoning citing the document>
ANSWER: <concise final answer>"""

    response = client.chat.completions.create(
        model="gpt-4",
        messages=[{"role": "user", "content": cot_prompt}],
        temperature=0,
    )

    cot_answer = response.choices[0].message.content

    # Step 2: Format as training sample
    # Mix oracle with distractors (oracle position randomized)
    import random
    all_docs = distractor_docs + [oracle_doc]
    random.shuffle(all_docs)

    context = "\n\n---\n\n".join(
        [f"Document {i+1}: {doc}" for i, doc in enumerate(all_docs)]
    )

    return {
        "instruction": f"Given these documents, answer the question.\n\n{context}\n\nQuestion: {question}",
        "output": cot_answer,
    }
```

### Oracle/Distractor Ratio
```python
# RAFT paper recommends:
# - P% of training samples include the oracle document (P = 60-80%)
# - (1-P)% of samples have ONLY distractor documents
#   (teaches the model to say "I don't know" or use memorized knowledge)

def create_raft_dataset(qa_pairs, documents, p_oracle=0.7, n_distractors=4):
    """Create full RAFT training dataset."""
    dataset = []

    for question, answer, oracle_doc_id in qa_pairs:
        oracle_doc = documents[oracle_doc_id]

        # Select random distractor documents
        distractor_ids = [
            d for d in documents if d != oracle_doc_id
        ]
        selected_distractors = random.sample(
            distractor_ids, min(n_distractors, len(distractor_ids))
        )
        distractor_docs = [documents[d] for d in selected_distractors]

        if random.random() < p_oracle:
            # Oracle + distractors (open-book with answer)
            sample = generate_raft_sample(
                question, oracle_doc, distractor_docs
            )
        else:
            # Distractors only (closed-book, no oracle)
            sample = generate_raft_sample(
                question,
                oracle_doc=None,
                distractor_docs=distractor_docs,
            )

        dataset.append(sample)

    return dataset
```

## Fine-Tuning with RAFT
### Using Axolotl
```yaml
# axolotl_raft_config.yml
base_model: meta-llama/Llama-2-7b-hf
model_type: LlamaForCausalLM
tokenizer_type: LlamaTokenizer

load_in_4bit: true
adapter: qlora
lora_r: 16
lora_alpha: 32
lora_dropout: 0.05
lora_target_modules:
  - q_proj
  - v_proj
  - k_proj
  - o_proj

datasets:
  - path: ./raft_training_data.jsonl
    type: instruction
    field_instruction: instruction
    field_output: output

sequence_len: 4096
micro_batch_size: 4
gradient_accumulation_steps: 4
num_epochs: 3
learning_rate: 2e-4
optimizer: adamw_torch
lr_scheduler: cosine
warmup_ratio: 0.1

bf16: true
tf32: true
gradient_checkpointing: true
```

```bash
# Run Axolotl training
accelerate launch -m axolotl.cli.train axolotl_raft_config.yml
```

### Using transformers + PEFT
```python
from transformers import (
    AutoModelForCausalLM, AutoTokenizer,
    TrainingArguments, Trainer,
)
from peft import LoraConfig, get_peft_model, prepare_model_for_kbit_training
from datasets import load_dataset

# Load model in 4-bit
model = AutoModelForCausalLM.from_pretrained(
    "meta-llama/Llama-2-7b-hf",
    load_in_4bit=True,
    device_map="auto",
)
model = prepare_model_for_kbit_training(model)

# Apply LoRA
lora_config = LoraConfig(
    r=16, lora_alpha=32, lora_dropout=0.05,
    target_modules=["q_proj", "v_proj", "k_proj", "o_proj"],
    task_type="CAUSAL_LM",
)
model = get_peft_model(model, lora_config)

# Load RAFT dataset
dataset = load_dataset("json", data_files="raft_training_data.jsonl")

# Training
training_args = TrainingArguments(
    output_dir="./raft_output",
    num_train_epochs=3,
    per_device_train_batch_size=4,
    gradient_accumulation_steps=4,
    learning_rate=2e-4,
    bf16=True,
    logging_steps=10,
    save_strategy="epoch",
)

trainer = Trainer(
    model=model,
    args=training_args,
    train_dataset=dataset["train"],
)
trainer.train()
```

## Inference Pipeline
### RAFT Model with Retrieval
```python
from transformers import AutoModelForCausalLM, AutoTokenizer
from peft import PeftModel

# Load base + RAFT adapter
base_model = AutoModelForCausalLM.from_pretrained(
    "meta-llama/Llama-2-7b-hf",
    load_in_4bit=True,
    device_map="auto",
)
model = PeftModel.from_pretrained(base_model, "./raft_output")
tokenizer = AutoTokenizer.from_pretrained("meta-llama/Llama-2-7b-hf")

def raft_inference(question, retrieved_docs, model, tokenizer):
    """Run RAFT inference with retrieved documents."""
    context = "\n\n---\n\n".join(
        [f"Document {i+1}: {doc}" for i, doc in enumerate(retrieved_docs)]
    )

    prompt = f"""Given these documents, answer the question with reasoning.

{context}

Question: {question}

REASONING:"""

    inputs = tokenizer(prompt, return_tensors="pt").to(model.device)
    outputs = model.generate(
        **inputs,
        max_new_tokens=512,
        temperature=0.1,
        do_sample=True,
    )
    return tokenizer.decode(outputs[0], skip_special_tokens=True)
```

## Evaluation
### Comparing RAG vs RAFT
```python
from datasets import load_dataset
from rouge_score import rouge_scorer

scorer = rouge_scorer.RougeScorer(['rouge1', 'rougeL'], use_stemmer=True)

def evaluate_pipeline(pipeline_fn, eval_dataset):
    """Evaluate a RAG or RAFT pipeline."""
    results = {"correct": 0, "total": 0, "rouge1": [], "rougeL": []}

    for sample in eval_dataset:
        prediction = pipeline_fn(
            sample["question"],
            sample["retrieved_docs"],
        )

        scores = scorer.score(sample["ground_truth"], prediction)
        results["rouge1"].append(scores["rouge1"].fmeasure)
        results["rougeL"].append(scores["rougeL"].fmeasure)
        results["total"] += 1

        if sample["answer_keyword"] in prediction.lower():
            results["correct"] += 1

    accuracy = results["correct"] / results["total"]
    avg_rouge1 = sum(results["rouge1"]) / len(results["rouge1"])
    avg_rougeL = sum(results["rougeL"]) / len(results["rougeL"])

    return {
        "accuracy": accuracy,
        "rouge1": avg_rouge1,
        "rougeL": avg_rougeL,
    }
```

## Tips
- Use 60-80% oracle ratio in training data -- too much oracle and the model ignores distractors, too little and it cannot leverage relevant context
- Generate CoT reasoning with a strong teacher model (GPT-4) rather than writing it manually for consistent quality
- Include 3-5 distractor documents per sample to match realistic retrieval noise levels
- RAFT shines on domain-specific corpora (medical, legal, financial) where standard RAG struggles with terminology
- Always include some closed-book samples (no oracle) to teach the model to recognize when context is insufficient
- Fine-tune with the same retrieval pipeline you will use in production so the model learns to handle real retrieval noise
- Start with QLoRA (4-bit + LoRA r=16) to keep training feasible on a single GPU
- Evaluate on held-out questions with both oracle-present and oracle-absent scenarios to measure robustness
- RAFT training data quality matters more than quantity -- 1000 high-quality CoT samples beat 10000 noisy ones
- Compare RAFT against vanilla RAG on your specific domain before committing to the training pipeline

## See Also
- rag, lora, llm-fundamentals, prompt-engineering, transformers

## References
- [RAFT: Adapting Language Model to Domain Specific RAG (Gorilla Team, 2024)](https://arxiv.org/abs/2403.10131)
- [Axolotl Fine-Tuning Framework](https://github.com/OpenAccess-AI-Collective/axolotl)
- [PEFT: Parameter-Efficient Fine-Tuning](https://huggingface.co/docs/peft/)
- [QLoRA: Efficient Finetuning of Quantized LLMs](https://arxiv.org/abs/2305.14314)
