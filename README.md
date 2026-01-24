# efx-face-manager

```
┌─┐┌─┐─┐ ┬   ┌─┐┌─┐┌─┐┌─┐
├┤ ├┤ ┌┴┬┘───├┤ ├─┤│  ├┤ 
└─┘└  ┴ └─   └  ┴ ┴└─┘└─┘

MLX Hugging Face Manager
by Laurent Marques
```

**Version: 0.1.13**

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
- **Multi-Path Storage (v0.1.1)** - Flexible model storage with automatic fallback:
  - **External Path**: `/Volumes/T7/mlx-server` (when drive is mounted)
  - **Local Path**: `~/mlx-server` (fallback when external is unavailable)
  - Auto-detection with External → Local fallback
  - Path persistence across sessions via `~/.efx-face-manager.conf`
  - Real-time status display (models count, availability)

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

**Requirements:**
- Python 3.11 or 3.12 (Python 3.13+ is not yet supported)
- Use pyenv to manage Python versions if needed

#### Installation Method: Using uv (Recommended)

```bash
```bash


# Set your preferred Python version
pyenv global 3.12.8

cd ~/Scripts/mlx-tools

# Clone the mlx-openai-server repository
git clone https://github.com/cubist38/mlx-openai-server.git
cd mlx-openai-server

# Create virtual environment (will use your pyenv Python 3.12.8)
uv venv

# Activate the environment
source .venv/bin/activate

# Install in development mode
uv pip install -e .

# Install latest mlx-lm from GitHub if needed to update to last mlx-lm
# uv pip install git+https://github.com/ml-explore/mlx-lm.git

```
Then
> add in your zshrc
```bash
mlx-openai-server() {
    local original_dir="$PWD"
    cd ~/Scripts/mlx-tools/mlx-openai-server
    source .venv/bin/activate
    command mlx-openai-server "$@"
    cd "$original_dir"
}
```

**Verify installation:**
```bash
# see mlx-openai-server doc here: https://github.com/cubist38/mlx-openai-server
mlx-openai-server --version
# Should show version 1.4.2 or higher for full feature support

# Launch models from this environment
mlx-openai-server launch --model-path /path/to/model --model-type lm
```

> **Note:** Version 1.4.2+ is required for image-generation, image-edit, embeddings, and whisper model types.
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

### Model Storage Path

The application automatically manages model storage paths with smart defaults:

**Default Behavior:**
- First launch: Auto-detects External (`/Volumes/T7/mlx-server`) if drive mounted, else Local (`~/mlx-server`)
- Subsequent launches: Uses saved preference from `~/.efx-face-manager.conf`

**Manual Configuration:**
Use the **Configure Model Storage Path** menu option to:
- Switch between External, Local, or Legacy paths
- View real-time status and model counts
- Enable auto-detection for automatic fallback

**Available Paths:**
```bash
# External (recommended for large collections)
/Volumes/T7/mlx-server

# Local (always available)
~/mlx-server
```

### Environment Variables (Optional)

You can override the path temporarily:
```bash
# Override for single session
export MODEL_DIR="/custom/path"
./efx-face-manager.sh
```

> **Note:** Manual `MODEL_DIR` exports override the saved configuration. Use the in-app menu for persistent changes.

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
| **⚡ Run LM on Template** | Fast-launch predefined language model templates with optimized configurations |
| **Run an Installed LLM** | Select and launch a model with preset configurations or custom settings |
| **Install a New Hugging Face LLM** | Browse and download models from Hugging Face |
| **Uninstall an LLM** | Remove installed models and clean cache |
| **Configure Model Storage Path** | Switch between External, Local, or Legacy storage paths |
| **Exit** | Quit the application |

### Template Models (v0.1.10)

#### Fast Launch Predefined Templates

Select **⚡ Run LM on Template** from the main menu to quickly launch popular language models with optimized configurations:

**Available Templates:**
- **Qwen3-Coder-30B-A3B-Instruct-8bit** - Optimized for coding tasks
- **NVIDIA-Nemotron-3-Nano-30B-A3B-MLX-8Bit** - General purpose language model
- **GLM-4.7-Flash-8bit** - Fast inference Chinese/English language model
- **Qwen3-VL-8B-Thinking-8bit** - Vision-language multimodal model with thinking capabilities

**Features:**
- One-click launch with pre-configured optimal settings
- No manual configuration required
- Uses proven parameter combinations for best performance
- Supports all standard mlx-openai-server launch options

**Template Configuration:**
Each template comes with vendor-recommended settings:
- Appropriate tool-call parsers and message converters
- Optimized quantization levels
- Pre-set server configurations (port 8000, host 0.0.0.0)
- Model-type specific optimizations

> **Note:** Templates require the corresponding model to be installed in your model storage path first.

### Model Storage Configuration (v0.1.1)

#### Path Selection
1. Select **Configure Model Storage Path** from main menu
2. Choose from available options:
   - **External**: `/Volumes/T7/mlx-server` - Best for large collections
   - **Local**: `~/mlx-server` - Always available on internal drive
   - **Auto-detect** - Automatic External → Local fallback based on drive availability
3. Selection is immediate - no confirmation needed
4. Each path shows real-time status:
   - ✓ Active (X models) - Path has models
   - ✓ Available (no models) - Path exists but empty
   - ✗ External drive not mounted - Drive unavailable
5. Current path marked with ← indicator

#### How It Works
- **First Launch**: Auto-detects External if `/Volumes/T7` is mounted, otherwise uses Local
- **Persistence**: Selected path saved to `~/.efx-face-manager.conf`
- **Subsequent Launches**: Loads saved path preference
- **Path Switching**: All operations (install/run/list) immediately use new path
- **Independence**: Each path maintains its own model collection

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
$MODEL_DIR/                        # Selected path (External/Local/Legacy)
├── cache/                         # Downloaded model files
│   └── models--org--ModelName/
│       └── snapshots/
│           └── abc123/
├── ModelName -> cache/.../        # Symlink to model
└── AnotherModel -> cache/.../     # Clean root with symlinks
```

**Multiple Paths:**
- Each path (`External`, `Local`) maintains independent model collections
- Switching paths shows only models in that location
- Models don't move when switching - they stay in their original path

**Bug Fix (v0.1.5):**
- Fixed symlink creation bug where all models pointed to the same cached model
- Each model now correctly links to its own cache directory
- Re-download models if installed with v0.1.4 or earlier

**Enhancement (v0.1.6):**
- Updated documentation with dual installation methods (pipx and uv)
- Added mlx-lm update instructions for newest model architectures
- Added troubleshooting for unsupported model types

**Enhancement (v0.1.7):**
- Added quick "Trust remote code" toggle on preset launch screen for LM/multimodal models
- No need to enter config menu to enable trust_remote_code for models like IQuest-Coder

**Enhancement (v0.1.8):**
- Added logo header to all configuration menus
- Clear screen before redrawing to prevent overlapping displays
- Cleaner UI experience when toggling options or returning from config menus

**Enhancement (v0.1.10):**
- Added **⚡ Run LM on Template** menu option for fast model launching
- Added **GLM-4.7-Flash-8bit** template with optimized configuration
- Added template models documentation with usage instructions
- Improved main menu organization and discoverability

**Fix (v0.1.11):**
- Added **glm47_flash** to --reasoning-parser param from GLM-4.7-Flash-8bit template

**Enhancement (v0.1.12):**
- Added **Qwen3-VL-8B-Thinking-8bit** vision template with multimodal support
- Enhanced template collection with vision-language capabilities
- Optimized queue management for multimodal models

**Fix (v0.1.13):**
- Fixed **mlx-openai-server command not found** error in template execution
- Added intelligent mlx-openai-server detection supporting multiple installation methods (pipx, uv venv)
- Script now automatically detects and uses correct mlx-openai-server installation
- Improved compatibility with both documented installation methods from README

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

### "Model type [architecture] not supported" error

If you encounter an error like `Model type iquestcoder not supported`, your mlx-lm version may be outdated.

**Solution:**
```bash
# Uninstall old mlx-lm
pipx runpip mlx-openai-server uninstall mlx-lm -y

# Install latest from GitHub
pipx runpip mlx-openai-server install git+https://github.com/ml-explore/mlx-lm.git

# Verify it worked
grep "iquestcoder" ~/.local/pipx/venvs/mlx-openai-server/lib/python3.*/site-packages/mlx_lm/utils.py
# Should output: "iquestcoder": "llama",
```

### "trust_remote_code" error

If you see: `Please pass the argument trust_remote_code=True to allow custom code to be run`

**Solution:**
Add the `--trust-remote-code` flag when launching:
```bash
mlx-openai-server launch \
  --model-path /path/to/model \
  --model-type lm \
  --trust-remote-code \
  --host 0.0.0.0 \
  --port 8000
```

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