# Agent-to-Agent Presentation

This is a reveal.js presentation about the Agent-to-Agent (A2A) architecture for generative AI.

## Color Scheme

The presentation uses the **OCTO Technology** brand colors:
- **Primary:** Bleu Marine (`#0E2356`)
- **Secondary:** Turquoise (`#00D2DD`)
- **Background:** White (`#FFFFFF`)

All color combinations follow WCAG accessibility guidelines for proper contrast ratios.

## How to View

### Option 1: Direct File Access
Simply open the `index.html` file in a web browser:
```bash
open presentation/index.html
```

### Option 2: Local Server (Recommended)
For the best experience, serve the presentation with a local HTTP server:

```bash
cd presentation
python3 -m http.server 8000
```

Then visit: http://localhost:8000

### Option 3: Using Go
```bash
cd presentation
go run -m http.server 8000
```

## Presentation Controls

- **Next slide:** Space, Arrow Right, Arrow Down, Page Down
- **Previous slide:** Arrow Left, Arrow Up, Page Up
- **Full screen:** F
- **Overview mode:** ESC or O
- **Speaker notes:** S
- **Pause:** B or .

## Structure

The presentation contains 22 slides covering:
1. Introduction
2. Problem statement (synchronous AI limitations)
3. Omnichannel concept
4. Event-driven architecture solution
5. A2A protocol from Google
6. Cortex meta-intelligence architecture
7. Key takeaways (5 main points)
8. Conclusion and call to action

## Features

- **Responsive design** - Works on desktop and mobile
- **Code highlighting** - Syntax highlighting for JSON, Protobuf, and text
- **Animations** - Fragment animations for progressive disclosure
- **Accessible colors** - OCTO brand colors with proper contrast
- **Self-contained** - No external dependencies needed (uses CDN)

## Customization

The presentation uses CSS custom properties (CSS variables) for colors. To customize:

1. Edit the `:root` section in the `<style>` tag
2. Modify color variables:
   - `--octo-marine` (primary)
   - `--octo-turquoise` (secondary)
   - Various opacity levels (10-90%)

## Export to PDF

To export the presentation to PDF:

1. Open in Chrome/Chromium browser
2. Add `?print-pdf` to the URL: `http://localhost:8000?print-pdf`
3. Open Print dialog (Ctrl/Cmd + P)
4. Select "Save as PDF"
5. Configure:
   - Destination: Save as PDF
   - Layout: Landscape
   - Margins: None
   - Background graphics: Enabled

## Technologies Used

- [Reveal.js 4.5.0](https://revealjs.com/) - HTML presentation framework
- CSS Grid and Flexbox - Layout
- CSS Custom Properties - Theming
- Monokai - Code syntax highlighting theme

## License

This presentation is part of the AgentHub project.
