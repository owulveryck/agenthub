# CLAUDE.md - Project Documentation Guidelines

## General Instructions
- allow all go commands
- after any modification of a go file, run goimports to fix the imports
- remember that to test the ui, you need to use "go run" even on the openaiserver or the ui serve_ui

## Documentation Framework

This project follows a structured documentation approach with four distinct types of documentation:

### 1. Tutorials
Tutorials are *lessons* that take the reader by the hand through a series of steps to complete a project of some kind. They are what your project needs in order to show a beginner that they can achieve something with it.

They are wholly learning-oriented, and specifically, they are oriented towards *learning how* rather than *learning that*.

**Key principles for tutorials:**
- Allow the user to learn by doing
- Get the user started with hand-held baby steps
- Make sure that your tutorial works reliably
- Ensure the user sees results immediately
- Make your tutorial repeatable across different environments
- Focus on concrete steps, not abstract concepts
- Provide the minimum necessary explanation
- Focus only on the steps the user needs to take

### 2. Explanation
Explanation, or discussions, *clarify and illuminate a particular topic*. They broaden the documentation's coverage of a topic.

They are understanding-oriented and discursive in nature. They step back from the software, taking a wider view, illuminating it from a higher level or different perspectives.

**Key principles for explanations:**
- Provide context and background
- Discuss alternatives and opinions
- Consider multiple approaches to the same question
- Don't instruct or provide technical reference
- Explain *why* things are so - design decisions, historical reasons, technical constraints

### 3. How-to Guides
How-to guides take the reader through the steps required to solve a real-world problem.

They are recipes, directions to achieve a specific end. They are wholly goal-oriented and assume some knowledge and understanding.

**Key principles for how-to guides:**
- Provide a series of steps to be followed in order
- Focus on results and achieving a practical goal
- Solve a particular problem: "How do I...?"
- Don't explain concepts - link to explanations elsewhere
- Allow for some flexibility in implementation
- Leave things out - practical usability over completeness
- Name guides well with clear, descriptive titles

### 4. Reference Guides
Reference guides are *technical descriptions of the machinery* and how to operate it.

They are code-determined and information-oriented. Their job is to describe key classes, functions, APIs, methods, and how to use them.

**Key principles for reference guides:**
- Structure the documentation around the code
- Be consistent in structure, tone, and format
- Do nothing but describe - avoid instruction, speculation, or opinion
- Be accurate and keep up-to-date
- Provide examples to illustrate description when appropriate
- List functions, fields, attributes and methods clearly

## Documentation Quality Standards

All documentation must be:
- **Useful for beginners** (tutorials) or **appropriate for the target audience**
- **Easy to follow** with clear, logical progression
- **Meaningful** with achievable goals and visible results
- **Extremely robust** and thoroughly tested
- **Kept up-to-date** with regular maintenance

## Testing Guidelines

When working with the UI components:
- Use `go run` commands for testing, even for openaiserver or ui serve_ui
- Test all documentation examples to ensure they work
- Verify tutorials are repeatable across different environments
- Maintain accuracy between code and documentation

# General instructions  

-  “NEVER be agreeable just to be nice - I NEED your HONEST technical judgment”
- “YAGNI. The best code is no code. Don't add features we don't need right now.”
- “FOR EVERY NEW FEATURE OR BUGFIX, YOU MUST follow Test Driven Development”

