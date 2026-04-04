# Prompt Engineering (Effective LLM Prompting Strategies)

A practical guide to crafting effective prompts for large language models, covering system prompts, few-shot examples, chain-of-thought reasoning, structured output, tool use and function calling, prompt templates, evaluation frameworks, temperature tuning, and token optimization techniques.

## System Prompts
### Structure and Patterns
```
# Effective system prompt structure:
# 1. Role/persona definition
# 2. Core instructions and constraints
# 3. Output format specification
# 4. Edge case handling
# 5. Examples (optional)

# Example: Technical documentation assistant
SYSTEM_PROMPT = """You are a senior technical writer specializing in API documentation.

Rules:
- Write in active voice, present tense
- Use second person ("you") for instructions
- Include code examples for every endpoint
- Mark required parameters with (required)
- Mark optional parameters with (optional, default: X)
- Never invent endpoints that don't exist in the provided spec
- If unsure about behavior, say "Verify with the API team"

Output format:
- Use Markdown with H2 for endpoints, H3 for subsections
- Code blocks with language annotation (python, bash)
- Tables for parameter lists

Tone: Professional, concise, technically precise."""
```

### Persona Pattern
```python
personas = {
    "code_reviewer": """You are a senior staff engineer performing a code review.
Focus on: correctness, security, performance, maintainability.
Be direct. Reference line numbers. Suggest fixes, not just problems.""",

    "explainer": """You are a patient CS professor explaining to a sophomore.
Use analogies, build from fundamentals, define jargon before using it.""",
}
```

## Few-Shot Prompting
### Basic Few-Shot
```python
prompt = """Classify the sentiment of each review as POSITIVE, NEGATIVE, or NEUTRAL.

Review: "The battery life is incredible, lasts all day easily."
Sentiment: POSITIVE

Review: "Screen cracked after two days. Terrible build quality."
Sentiment: NEGATIVE

Review: "It's an okay phone. Nothing special but gets the job done."
Sentiment: NEUTRAL

Review: "The camera is amazing in daylight but struggles in low light."
Sentiment:"""
```

### Few-Shot with Diverse Examples
```python
# Cover edge cases and boundary conditions in examples
few_shot_extraction = """Extract structured data from the text.

Text: "Meeting with Dr. Sarah Chen at 3pm on Tuesday in Room 401"
Result: {"person": "Dr. Sarah Chen", "time": "3pm", "day": "Tuesday", "location": "Room 401"}

Text: "Call John tomorrow"
Result: {"person": "John", "time": null, "day": "tomorrow", "location": null}

Text: "Team standup, every morning 9:15, virtual"
Result: {"person": null, "time": "9:15", "day": "every morning", "location": "virtual"}

Text: "Lunch with the CEO at that Italian place next Friday noon"
Result:"""
```

## Chain-of-Thought (CoT)
### Zero-Shot CoT
```python
# Simply append "Let's think step by step"
prompt = """A store has 47 apples. They receive 3 boxes with 12 apples each,
then sell 28 apples. How many apples remain?

Let's think step by step."""

# Output:
# 1. Starting apples: 47
# 2. Received: 3 boxes * 12 apples = 36 apples
# 3. Total after receiving: 47 + 36 = 83 apples
# 4. After selling 28: 83 - 28 = 55 apples
# Answer: 55 apples
```

### Few-Shot CoT
```python
prompt = """Solve the following problems showing your reasoning.

Q: Roger has 5 tennis balls. He buys 2 more cans of tennis balls.
Each can has 3 tennis balls. How many tennis balls does he have now?
A: Roger started with 5 balls. He bought 2 cans, each with 3 balls,
so he bought 2 * 3 = 6 balls. Total: 5 + 6 = 11 tennis balls.

Q: A café sells 23 coffees in the morning and 31 in the afternoon.
If each coffee uses 18g of beans, how many grams were used total?
A: Morning coffees: 23. Afternoon coffees: 31.
Total coffees: 23 + 31 = 54.
Beans per coffee: 18g.
Total beans: 54 * 18 = 972 grams.

Q: A library had 235 books. They donated 1/5 of their books and
then received a shipment of 48 new books. How many books now?
A:"""
```

### Self-Consistency (Multiple CoT Paths)
```python
from collections import Counter

def self_consistent_answer(question, n_samples=5):
    """Generate multiple CoT paths and take majority vote."""
    answers = []
    for _ in range(n_samples):
        response = client.chat.completions.create(
            model="gpt-4", temperature=0.7,
            messages=[
                {"role": "system", "content": "Solve step by step. End with 'ANSWER: <number>'"},
                {"role": "user", "content": question},
            ],
        )
        text = response.choices[0].message.content
        if "ANSWER:" in text:
            answers.append(text.split("ANSWER:")[-1].strip())
    return Counter(answers).most_common(1)[0]  # (answer, count)
```

## Structured Output
### JSON Mode
```python
from openai import OpenAI

client = OpenAI()

response = client.chat.completions.create(
    model="gpt-4-turbo",
    response_format={"type": "json_object"},
    messages=[
        {"role": "system", "content": """Extract event details as JSON with keys:
            name (string), date (ISO 8601), location (string),
            attendees (array of strings), description (string).
            Always return valid JSON."""},
        {"role": "user", "content": "Annual company retreat on March 15, 2025 at Lake Tahoe Resort. Attendees: Alice, Bob, Charlie. Team building and strategy planning."},
    ],
)

import json
event = json.loads(response.choices[0].message.content)
```

### Structured Output with Pydantic
```python
from pydantic import BaseModel, Field

class EventDetails(BaseModel):
    name: str = Field(description="Event name")
    date: str = Field(description="ISO 8601 date")
    location: str
    attendees: list[str]

response = client.beta.chat.completions.parse(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Annual retreat, March 15, Lake Tahoe."}],
    response_format=EventDetails,
)
event = response.choices[0].message.parsed  # Typed EventDetails object
```

## Tool Use / Function Calling
### OpenAI Function Calling
```python
from openai import OpenAI
client = OpenAI()

tools = [{
    "type": "function",
    "function": {
        "name": "get_weather",
        "description": "Get current weather for a location",
        "parameters": {
            "type": "object",
            "properties": {
                "location": {"type": "string", "description": "City and state"},
                "unit": {"type": "string", "enum": ["celsius", "fahrenheit"]},
            },
            "required": ["location"],
        },
    },
}]

response = client.chat.completions.create(
    model="gpt-4-turbo",
    messages=[{"role": "user", "content": "What's the weather in Boston?"}],
    tools=tools, tool_choice="auto",
)

message = response.choices[0].message
if message.tool_calls:
    for call in message.tool_calls:
        print(f"Function: {call.function.name}, Args: {call.function.arguments}")
```

### Anthropic Tool Use
```python
import anthropic
client = anthropic.Anthropic()

response = client.messages.create(
    model="claude-sonnet-4-20250514", max_tokens=1024,
    tools=[{
        "name": "get_weather",
        "description": "Get current weather for a location",
        "input_schema": {
            "type": "object",
            "properties": {"location": {"type": "string"}},
            "required": ["location"],
        },
    }],
    messages=[{"role": "user", "content": "Weather in Tokyo?"}],
)

for block in response.content:
    if block.type == "tool_use":
        print(f"Tool: {block.name}, Input: {block.input}")
```

## Common Patterns
### Constraint Pattern
```
# Set explicit boundaries
"Answer in exactly 3 bullet points."
"Use only information from the provided context."
"If you don't know, say 'I don't have enough information.'"
"Respond in under 100 words."
"Do not include any code in your response."
"List pros and cons. Minimum 5 of each."
```

### Output Priming
```python
# Start the response to constrain format
messages = [
    {"role": "user", "content": "List the top 5 programming languages for data science."},
    {"role": "assistant", "content": "1."},  # Prime the numbered list format
]
```

### Delimiter Pattern
```python
# Use clear delimiters for multi-part inputs
prompt = """Translate the text between <source> tags from English to French.
Do not translate anything outside the tags.

Context (do not translate): This is a product description for our EU market.

<source>
Our new product features advanced AI-powered recommendations
tailored to your browsing history and preferences.
</source>

Translation:"""
```

## Evaluation
### Simple Evaluation Framework
```python
def evaluate_prompt(prompt_template, test_cases, model="gpt-4"):
    """Evaluate a prompt against test cases with automated checking."""
    results = []
    for case in test_cases:
        response = get_completion(prompt_template.format(**case["input"]))
        results.append(case["check_fn"](response, case["expected"]))
    return sum(results) / len(results)  # accuracy
```

## Temperature Tuning Guide
```
| Task Type                    | Temperature | Top-p | Rationale                      |
|------------------------------|------------|-------|--------------------------------|
| Code generation              | 0          | 1.0   | Deterministic, correct code    |
| Data extraction              | 0          | 1.0   | Consistent, faithful output    |
| Classification               | 0          | 1.0   | Reproducible labels            |
| Technical writing             | 0.3        | 0.9   | Slight variation, factual      |
| General Q&A                  | 0.5        | 0.9   | Balanced creativity/accuracy   |
| Creative writing             | 0.8        | 0.95  | More diverse, interesting      |
| Brainstorming                | 1.0        | 0.95  | Maximum diversity              |
| Self-consistency (ensemble)  | 0.7        | 0.9   | Diverse reasoning paths        |
```

## Token Optimization
### Reducing Token Count
```python
# Verbose (47 tokens):
prompt_verbose = """I would like you to please analyze the following
piece of text and determine what the overall sentiment of the text is.
The text is as follows:"""

# Optimized (12 tokens):
prompt_optimized = """Classify sentiment (POSITIVE/NEGATIVE/NEUTRAL):"""

# Use abbreviations in system prompts for recurring instructions
# "resp" -> "response", "req" -> "required", "info" -> "information"

# Compress few-shot examples
# Instead of full sentences, use minimal patterns:
# Input: "great product" -> POSITIVE
# Input: "broke immediately" -> NEGATIVE
```

## Tips
- Start with a clear system prompt that defines role, constraints, and output format before adding complexity
- Use few-shot examples that cover edge cases and boundary conditions, not just happy paths
- Chain-of-thought improves reasoning accuracy by 10-40% on math, logic, and multi-step problems
- Set temperature to 0 for extraction, classification, and code -- these need determinism, not creativity
- Test prompts with adversarial inputs: what happens with empty input, nonsense, or prompt injection attempts?
- JSON mode with Pydantic schemas catches structural errors that free-text parsing misses
- Delimiter patterns (XML tags, triple backticks) prevent the model from confusing instructions with content
- Self-consistency (multiple samples + majority vote) improves accuracy but costs proportionally more tokens
- Keep system prompts under 500 tokens for cost efficiency -- most of the context window should be for the task
- Version control your prompts: small wording changes can cause 5-15% accuracy swings
- Use output priming (pre-filling the assistant response) to enforce exact output format
- Evaluate prompts quantitatively on a test set, not by vibes -- gut feeling misleads on edge case performance

## See Also
- llm-fundamentals, rag, transformers, chain-of-thought, function-calling

## References
- [OpenAI Prompt Engineering Guide](https://platform.openai.com/docs/guides/prompt-engineering)
- [Anthropic Prompt Engineering Guide](https://docs.anthropic.com/en/docs/build-with-claude/prompt-engineering)
- [Chain-of-Thought Prompting (Wei et al., 2022)](https://arxiv.org/abs/2201.11903)
- [Self-Consistency (Wang et al., 2022)](https://arxiv.org/abs/2203.11171)
- [OpenAI Function Calling Docs](https://platform.openai.com/docs/guides/function-calling)
- [Anthropic Tool Use Docs](https://docs.anthropic.com/en/docs/build-with-claude/tool-use)
