# efx-face-manager

```
┌─┐┌─┐─┐ ┬   ┌─┐┌─┐┌─┐┌─┐
├┤ ├┤ ┌┴┬┘───├┤ ├─┤│  ├┤ 
└─┘└  ┴ └─   └  ┴ ┴└─┘└─┘

MLX Hugging Face Manager
by Laurent Marques
```

**Version: 0.0.2**

A terminal-based LLM model manager for Apple Silicon Macs. Browse, install, and run MLX-optimized models from Hugging Face with an intuitive TUI interface.

## Demo

![efx-face-manager Demo](./efx-face.gif)

> 📹 Browse, search, install, and run MLX-optimized LLMs from Hugging Face

## Features

- **Browse Models** - Access 3000+ MLX models from Hugging Face with pagination
- **Search API** - Search models directly via Hugging Face API
- **Model Details** - View downloads, likes, size, and more before installing
- **Install Models** - Download and set up models with one click
- **Run Models** - Launch models with MLX OpenAI Server
- **Uninstall** - Clean removal of models and cache
- **Multiple Sources** - Browse mlx-community, lmstudio-community, or all models

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
| **Run an Installed LLM** | Select and launch a model with MLX OpenAI Server |
| **Install a New Hugging Face LLM** | Browse and download models from Hugging Face |
| **Uninstall an LLM** | Remove installed models and clean cache |
| **Exit** | Quit the application |

### Installing Models

1. Select **Install a New Hugging Face LLM**
2. Choose a source:
   - `mlx-community` - MLX-optimized models (recommended)
   - `lmstudio-community` - LMStudio models
   - `All Models` - Browse all Hugging Face models
3. Browse models (100 per page) or use **Search API** to filter
4. Select a model to view details
5. Choose **Install this LLM** to download

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