# efx-face-manager

```
┌─┐┌─┐─┐ ┬   ┌─┐┌─┐┌─┐┌─┐
├┤ ├┤ ┌┴┬┘───├┤ ├─┤│  ├┤ 
└─┘└  ┴ └─   └  ┴ ┴└─┘└─┘

MLX Hugging Face Manager
by Laurent Marques
```

**Version: 0.1.0**

A terminal-based LLM model manager for Apple Silicon Macs. Browse, install, and run MLX-optimized models from Hugging Face with an intuitive TUI interface.

## Demo

![efx-face-manager Demo](./efx-face.gif)

> 📹 Browse, search, install, and run MLX-optimized LLMs from Hugging Face

## Features

### Core Features
- **Browse Models** - Access 3000+ MLX models from Hugging Face with pagination
- **Unified Search** - Search models directly via Hugging Face API with live filtering
- **Model Details** - View downloads, likes, size, and more before installing
- **Install Models** - Download and set up models with one click
- **Run Models** - Launch models with MLX OpenAI Server
- **Uninstall** - Clean removal of models and cache
- **Multiple Sources** - Browse mlx-community, lmstudio-community, or all models

### Advanced Configuration (v0.1.0)
- **🎯 Preset Configurations** - Quick-launch with 6 model type presets:
  - `lm` - Text-only language models
  - `multimodal` - Vision, audio, and text processing
  - `image-generation` - Qwen image generation (default: q16)
  - `image-edit` - Qwen image editing (default: q16)
  - `embeddings` - Text embeddings generation
  - `whisper` - Audio transcription

- **⚙️ Interactive Configuration** - Single-page settings with live preview:
  - Configure all parameters on one screen
  - Direct editing by clicking parameter lines
  - See current values while modifying
  - Full control over quantization, LoRA, server settings

- **🔧 Model-Type Specific Options**:
  - **LM/Multimodal**: Context length, tool calling, parsers, chat templates
  - **Image Generation**: Config name (flux/qwen), quantization (4/8/16), LoRA adapters
  - **Image Edit**: Context-aware editing, quantization, LoRA support
  - **Whisper**: Queue management, concurrency control
  - **Embeddings**: Server configuration, memory optimization

- **📋 Command Preview** - Review full command before execution
- **🔄 Modify & Re-run** - Edit presets, preview, and relaunch without restarting

## Prerequisites

### 1. Install Homebrew (if not installed)
```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

### 2. Install gum (Terminal UI toolkit)
```bash
brew install gum
```

### 3. Install jq (JSON processor)
```bash
brew install jq
```

### 4. Install Hugging Face CLI
```bash
brew install huggingface-cli
```

### 5. Install MLX OpenAI Server (for running models)
```bash
pipx install mlx-openai-server
```
> See: https://github.com/cubist38/mlx-openai-server

### 6. Login to Hugging Face (optional, for private models)
```bash
hf auth login
```

## Installation

### Option 1: Clone the repository
```bash
git clone https://github.com/electroheadfx/efx-face-manager.git
cd efx-face-manager
chmod +x efx-face-manager.sh
```

### Option 2: Direct download
```bash
curl -O https://raw.githubusercontent.com/electroheadfx/efx-face-manager/main/efx-face-manager.sh
chmod +x efx-face-manager.sh
```

### Option 3: Add to PATH
```bash
# Add to your .zshrc or .bashrc
export PATH="$PATH:/path/to/efx-face-manager"
```

## Configuration

Set the model directory in your shell config (`.zshrc` or `.bashrc`):

```bash
# Default location for downloaded models
export MODEL_DIR="/path/to/your/models"
```

## Usage

### Display version
```bash
./efx-face-manager.sh --version
# or
./efx-face-manager.sh -v
```

### Run the manager
```bash
./efx-face-manager.sh
```

### Main Menu Options

| Option | Description |
|--------|-------------|
| **Run an Installed LLM** | Select and launch a model with preset configurations or custom settings |
| **Install a New Hugging Face LLM** | Browse and download models from Hugging Face |
| **Uninstall an LLM** | Remove installed models and clean cache |
| **Exit** | Quit the application |

### Running Models (v0.1.0)

#### Quick Launch with Presets
1. Select **Run an Installed LLM**
2. Choose a model from installed models
3. Select a preset configuration:
   - **lm (text-only)** - Default language model
   - **multimodal (vision, audio)** - Multimodal processing
   - **image-generation (qwen-image, q16)** - Image generation with Qwen
   - **image-edit (qwen-image-edit, q16)** - Image editing with Qwen
   - **embeddings** - Text embeddings
   - **whisper (audio transcription)** - Speech-to-text
4. Choose action:
   - **▶ Run** - Launch immediately with preset defaults
   - **⚙️  Modify config...** - Customize parameters before launch
   - **✖ Cancel** - Go back

#### Interactive Configuration
When you select **Modify config...**:
- All parameters displayed on one screen
- Click any parameter line to edit its value
- See current values and defaults in real-time
- Configure:
  - Model type-specific options (context length, quantization, etc.)
  - Server settings (port, host, logging)
  - Advanced options (LoRA adapters, tool calling, parsers)
- Preview full command before execution
- Cancel returns to preset menu (preserves navigation stack)

### Installing Models

1. Select **Install a New Hugging Face LLM**
2. Choose a source:
   - `mlx-community` - MLX-optimized models (recommended)
   - `lmstudio-community` - LMStudio models
   - `All Models` - Browse all Hugging Face models
3. Choose browse mode:
   - **🔍 Search a model...** - Enter search term to filter models
   - **📋 Show all models...** - Browse paginated list (100 per page)
4. Select a model to view details (downloads, likes, size, last update)
5. Choose **Install this LLM** to download
6. Press ESC or Back to return to browse mode menu (not root)

### Model Storage Structure

```
$MODEL_DIR/
├── cache/                         # Downloaded model files
│   └── models--org--ModelName/
│       └── snapshots/
│           └── abc123/
├── ModelName -> cache/.../        # Symlink to model
└── AnotherModel -> cache/.../     # Clean root with symlinks
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑/↓` | Navigate menu |
| `Enter` | Select option |
| `ESC` | Go back / Exit |

## Requirements

- macOS with Apple Silicon (M1/M2/M3/M4)
- Python 3.8+
- Disk space varies by model (1GB - 50GB+)

## Troubleshooting

### "gum: command not found"
```bash
brew install gum
```

### "jq: command not found"
```bash
brew install jq
```

### "hf: command not found"
```bash
brew install huggingface-cli
```

### "mlx-openai-server: command not found"
```bash
pipx install mlx-openai-server
```

### Models not showing
- Check your internet connection
- Verify Hugging Face API is accessible
- Try searching with a specific term

## License

MIT License - Feel free to use, modify, and distribute.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Credits

- [Hugging Face](https://huggingface.co) - Model hosting
- [mlx-openai-server](https://github.com/cubist38/mlx-openai-server) - OpenAI-compatible server for MLX (Apple's ML framework)
- [gum](https://github.com/charmbracelet/gum) - Terminal UI toolkit
- [VHS](https://github.com/charmbracelet/vhs) - Terminal GIF recorder for scripted CLI demos
- Coded with [Qoder](https://qoder.com/)