# How to Design Effective Agent Cards

This guide shows you how to design AgentCards that enable effective LLM-based discovery and delegation in the Cortex orchestration system.

## Why AgentCards Matter

When you register an agent with AgentHub, the Cortex orchestrator uses your AgentCard to:

1. **Understand your agent's capabilities** - What can it do?
2. **Match user requests** - Does this request fit this agent?
3. **Generate LLM prompts** - Include your agent in decision-making
4. **Delegate tasks** - Route appropriate work to your agent

The quality of your AgentCard directly impacts how effectively Cortex can use your agent.

## AgentCard Structure

```go
type AgentCard struct {
    ProtocolVersion string             // A2A protocol version (e.g., "0.2.9")
    Name            string             // Unique agent identifier
    Description     string             // Human-readable description
    Version         string             // Agent version (e.g., "1.0.0")
    Url             string             // Service endpoint (optional)
    Capabilities    *AgentCapabilities // Technical capabilities
    Skills          []*AgentSkill      // What the agent can do
    // ... other fields
}
```

The most important field for Cortex integration is **Skills**.

## Designing Skills

Each skill represents a specific capability your agent offers. The LLM uses skill information to decide when to delegate to your agent.

### Skill Structure

```go
type AgentSkill struct {
    Id          string   // Unique skill identifier
    Name        string   // Human-readable skill name
    Description string   // Detailed description of what this skill does
    Tags        []string // Categorization and keywords
    Examples    []string // Example user requests that match this skill
    InputModes  []string // Supported input types (e.g., "text/plain")
    OutputModes []string // Supported output types
}
```

### Writing Effective Descriptions

**❌ Poor description:**
```go
Description: "Processes data"
```

**✅ Good description:**
```go
Description: "Analyzes time-series data to detect anomalies using statistical methods. " +
    "Supports multiple algorithms including Z-score, moving average, and ARIMA. " +
    "Returns anomaly locations, severity scores, and confidence intervals."
```

**Why the good description works:**
- Specific about what it does ("analyzes time-series data")
- Mentions the method ("statistical methods")
- Lists supported features ("Z-score, moving average, ARIMA")
- Describes output ("anomaly locations, severity scores, confidence intervals")

### Writing Powerful Examples

Examples are **critical** - the LLM uses them to recognize when a user request matches your skill.

**❌ Weak examples:**
```go
Examples: []string{
    "analyze data",
    "find problems",
}
```

**✅ Strong examples:**
```go
Examples: []string{
    "Can you detect anomalies in this time series?",
    "Find unusual patterns in the sensor data",
    "Analyze this dataset for outliers",
    "Check if there are any abnormal readings",
    "Identify spikes or drops in the data",
    "Run anomaly detection on this log file",
    "Are there any suspicious values in this series?",
}
```

**Why strong examples work:**
- **Variety**: Different phrasings ("detect anomalies", "find unusual patterns", "outliers")
- **Natural language**: How users actually ask questions
- **Specific**: Mention domain terms ("time series", "sensor data", "log file")
- **Action-oriented**: Clear about what to do
- **Multiple formats**: Questions and commands

### Example Categories to Cover

For each skill, include examples that cover:

1. **Direct requests**: "Translate this text to Spanish"
2. **Questions**: "Can you convert this to French?"
3. **Implied tasks**: "I need this in German"
4. **Variations**: "Spanish translation please"
5. **With context**: "Translate the following paragraph to Japanese: ..."
6. **Different phrasings**: "Convert to Spanish", "Change to Spanish", "Make it Spanish"

## Complete Examples

### Example 1: Translation Agent

```go
agentCard := &pb.AgentCard{
    ProtocolVersion: "0.2.9",
    Name:            "agent_translator",
    Description: "Professional-grade language translation service powered by neural machine translation. " +
        "Supports 100+ languages with context-aware translation and proper handling of idioms, " +
        "technical terms, and cultural nuances.",
    Version: "2.1.0",

    Capabilities: &pb.AgentCapabilities{
        Streaming:         false,
        PushNotifications: false,
    },

    Skills: []*pb.AgentSkill{
        {
            Id:   "translate_text",
            Name: "Text Translation",
            Description: "Translates text between any pair of 100+ supported languages including " +
                "English, Spanish, French, German, Chinese, Japanese, Arabic, Russian, and many more. " +
                "Preserves formatting, handles idioms, and maintains context. " +
                "Supports both short phrases and long documents.",

            Tags: []string{
                "translation", "language", "nlp", "localization",
                "multilingual", "i18n", "communication",
            },

            Examples: []string{
                "Translate this to Spanish",
                "Can you convert this text to French?",
                "I need this paragraph in Japanese",
                "Translate from English to German",
                "What does this mean in Chinese?",
                "Convert this Spanish text to English",
                "Please translate to Portuguese",
                "How do you say this in Italian?",
                "Russian translation needed",
                "Change this to Arabic",
            },

            InputModes:  []string{"text/plain", "text/html"},
            OutputModes: []string{"text/plain", "text/html"},
        },
        {
            Id:   "detect_language",
            Name: "Language Detection",
            Description: "Automatically identifies the language of input text with high accuracy. " +
                "Can detect 100+ languages and provides confidence scores. " +
                "Useful for routing, preprocessing, and automatic translation workflows.",

            Tags: []string{"language", "detection", "nlp", "identification"},

            Examples: []string{
                "What language is this text in?",
                "Detect the language",
                "Can you identify this language?",
                "Which language is this?",
                "Tell me what language this is",
            },

            InputModes:  []string{"text/plain"},
            OutputModes: []string{"text/plain"},
        },
    },
}
```

### Example 2: Data Analysis Agent

```go
agentCard := &pb.AgentCard{
    ProtocolVersion: "0.2.9",
    Name:            "agent_data_analyst",
    Description: "Advanced data analysis and statistical computing agent. Performs exploratory " +
        "data analysis, statistical tests, correlation analysis, and generates insights from datasets.",
    Version: "1.5.2",

    Capabilities: &pb.AgentCapabilities{
        Streaming:         true, // Can stream large results
        PushNotifications: false,
    },

    Skills: []*pb.AgentSkill{
        {
            Id:   "analyze_dataset",
            Name: "Dataset Analysis",
            Description: "Performs comprehensive statistical analysis on datasets including " +
                "descriptive statistics (mean, median, std dev), distribution analysis, " +
                "outlier detection, correlation matrices, and trend identification. " +
                "Supports CSV, JSON, and structured data formats.",

            Tags: []string{
                "data-analysis", "statistics", "analytics", "dataset",
                "eda", "exploratory", "insights",
            },

            Examples: []string{
                "Analyze this dataset",
                "Can you provide statistics for this data?",
                "What are the key insights from this CSV?",
                "Run an analysis on this data file",
                "Give me a statistical summary",
                "Find correlations in this dataset",
                "What patterns do you see in this data?",
                "Analyze the distribution of these values",
                "Calculate descriptive statistics",
                "Identify trends in this time series",
            },

            InputModes:  []string{"text/csv", "application/json", "text/plain"},
            OutputModes: []string{"application/json", "text/plain", "text/html"},
        },
        {
            Id:   "visualize_data",
            Name: "Data Visualization",
            Description: "Creates charts and graphs from data including line charts, bar charts, " +
                "scatter plots, histograms, box plots, and heatmaps. Returns visualization " +
                "specifications in various formats.",

            Tags: []string{"visualization", "charts", "graphs", "plotting"},

            Examples: []string{
                "Create a chart from this data",
                "Visualize this dataset",
                "Make a graph of these values",
                "Plot this time series",
                "Show me a chart",
                "Generate a histogram",
                "Can you create a scatter plot?",
            },

            InputModes:  []string{"text/csv", "application/json"},
            OutputModes: []string{"image/png", "application/json", "text/html"},
        },
    },
}
```

### Example 3: Image Processing Agent

```go
agentCard := &pb.AgentCard{
    ProtocolVersion: "0.2.9",
    Name:            "agent_image_processor",
    Description: "Image processing and computer vision agent with capabilities for transformation, " +
        "enhancement, analysis, and object detection. Supports all major image formats.",
    Version: "3.0.0",

    Skills: []*pb.AgentSkill{
        {
            Id:   "resize_image",
            Name: "Image Resizing",
            Description: "Resizes images to specified dimensions while maintaining aspect ratio " +
                "and quality. Supports various scaling algorithms including bicubic, lanczos, " +
                "and nearest neighbor. Can batch process multiple images.",

            Tags: []string{"image", "resize", "scale", "transform", "dimensions"},

            Examples: []string{
                "Resize this image to 800x600",
                "Make this image smaller",
                "Scale this photo to 50%",
                "Can you resize to thumbnail size?",
                "Change image dimensions",
                "Shrink this image",
                "Make it 1920x1080",
            },

            InputModes:  []string{"image/jpeg", "image/png", "image/webp"},
            OutputModes: []string{"image/jpeg", "image/png", "image/webp"},
        },
        {
            Id:   "detect_objects",
            Name: "Object Detection",
            Description: "Detects and identifies objects in images using deep learning models. " +
                "Can recognize 1000+ object categories including people, animals, vehicles, " +
                "furniture, and more. Returns bounding boxes and confidence scores.",

            Tags: []string{
                "computer-vision", "object-detection", "ai", "recognition",
                "detection", "classification",
            },

            Examples: []string{
                "What objects are in this image?",
                "Detect objects in this photo",
                "What do you see in this picture?",
                "Identify items in this image",
                "Find all people in this photo",
                "Detect cars in this image",
                "What's in this picture?",
            },

            InputModes:  []string{"image/jpeg", "image/png"},
            OutputModes: []string{"application/json", "text/plain"},
        },
    },
}
```

## Best Practices Checklist

### ✅ Description Quality
- [ ] Clearly states what the agent does
- [ ] Mentions key features and capabilities
- [ ] Specifies supported formats/types
- [ ] Describes what output users can expect
- [ ] Uses domain-specific terminology appropriately

### ✅ Skill Design
- [ ] Each skill has a focused, specific purpose
- [ ] Skill names are clear and descriptive
- [ ] Descriptions explain benefits, not just features
- [ ] Tags are relevant and searchable
- [ ] Input/output modes accurately reflect capabilities

### ✅ Examples Coverage
- [ ] 5-10 examples per skill
- [ ] Mix of questions and commands
- [ ] Different phrasings and variations
- [ ] Natural language, not technical jargon
- [ ] Cover common use cases
- [ ] Include domain-specific terms
- [ ] Represent how real users ask

### ✅ Metadata
- [ ] Version follows semantic versioning
- [ ] Capabilities accurately reflect agent features
- [ ] Protocol version is current
- [ ] Agent name is unique and descriptive

## Testing Your AgentCard

### 1. Manual Testing

Start your agent and check Cortex logs:

```bash
grep "Agent skills registered" cortex.log
```

You should see your skill descriptions displayed.

### 2. LLM Prompt Testing

Check what the LLM sees by enabling DEBUG logging in Cortex:

```bash
LOG_LEVEL=DEBUG ./bin/cortex
```

Look for prompts that include:
```
Available agents:
- your_agent: Your agent description
  Skills:
    * Skill Name: Skill description
```

### 3. Integration Testing

Test with actual user requests:

```bash
# Start services
./bin/broker &
./bin/cortex &
./bin/your_agent &

# Use chat CLI
./bin/chat_cli

# Try requests that match your examples
> Can you translate this to Spanish?
```

Watch the logs to see if Cortex delegates to your agent.

## Common Mistakes to Avoid

### ❌ Vague Descriptions
```go
Description: "Does useful things"
```
**Problem**: LLM can't determine if this agent is suitable

### ❌ Too Few Examples
```go
Examples: []string{"do the thing"}
```
**Problem**: LLM won't recognize variations

### ❌ Technical Jargon in Examples
```go
Examples: []string{
    "Execute POST /api/v1/translate with payload",
}
```
**Problem**: Users don't talk like this

### ❌ Overly Broad Skills
```go
{
    Name: "Do Everything",
    Description: "This agent can help with anything",
}
```
**Problem**: LLM can't make good decisions

### ❌ Missing Context
```go
{
    Name: "Process",
    Description: "Processes the input",
}
```
**Problem**: What kind of processing? What input?

## Advanced Topics

### Multi-Language Support

Include examples in multiple languages if your agent supports them:

```go
Examples: []string{
    "Translate to Spanish",
    "Traduire en français",
    "Übersetzen Sie nach Deutsch",
    "日本語に翻訳",
}
```

### Conditional Capabilities

Use metadata to indicate conditional features:

```go
Metadata: &structpb.Struct{
    Fields: map[string]*structpb.Value{
        "requires_api_key": structpb.NewBoolValue(true),
        "max_input_size":   structpb.NewNumberValue(10000),
        "rate_limit":       structpb.NewStringValue("100/minute"),
    },
}
```

### Skill Dependencies

Indicate if skills build on each other:

```go
{
    Id: "advanced_analysis",
    Description: "Advanced statistical analysis. Requires dataset to be preprocessed " +
        "using the 'clean_data' skill first.",
}
```

## Iteration and Improvement

Your AgentCard isn't set in stone. Improve it based on:

1. **Usage patterns**: What requests do users actually make?
2. **Delegation success**: Is Cortex routing appropriate tasks?
3. **User feedback**: Are users getting what they expect?
4. **LLM behavior**: What decisions is the LLM making?

Update your AgentCard and restart your agent to reflect improvements.

## Next Steps

- See [Creating an Agent with Cortex](create_agent_with_cortex.md) for implementation
- Read [A2A Protocol](../explanation/the_agent_to_agent_principle.md) for context
- Review [Example Agents](../../agents/) for inspiration
- Check [AGENT_DECIDE.md](../../AGENT_DECIDE.md) for the complete specification

Well-designed AgentCards are the key to effective AI orchestration!
