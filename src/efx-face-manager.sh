#!/bin/bash

# ╔═══════════════════════════════════════════════════════════════╗
# ║                      efx-face-manager                         ║
# ║              MLX Hugging Face Model Manager                   ║
# ╚═══════════════════════════════════════════════════════════════╝
# Uses gum for interactive selection
# https://github.com/charmbracelet/gum

VERSION="0.1.8"

clear

# Show version
show_version() {
    echo "efx-face-manager v$VERSION"
    exit 0
}

# Get display-friendly model directory path
get_model_dir_display() {
    local dir="$MODEL_DIR"
    if [[ "$dir" == "$HOME"* ]]; then
        echo "~${dir#$HOME}"
    else
        echo "$dir"
    fi
}

# Show ASCII header
show_header() {
    gum style \
        --foreground 99 \
        --border-foreground 99 \
        --border double \
        --align center \
        --width 50 \
        --margin "1 2" \
        --padding "1 2" \
        '┌─┐┌─┐─┐ ┬   ┌─┐┌─┐┌─┐┌─┐
├┤ ├┤ ┌┴┬┘───├┤ ├─┤│  ├┤ 
└─┘└  ┴ └─   └  ┴ ┴└─┘└─┘' \
        '' \
        'MLX Hugging Face Manager' \
        "by Efx - ${VERSION}"
}

# Handle version flag
if [[ "$1" == "--version" || "$1" == "-v" ]]; then
    show_version
fi

# Define available model storage paths
EXTERNAL_MODEL_PATH="/Volumes/T7/mlx-server"
LOCAL_MODEL_PATH="$HOME/mlx-server"

# Configuration file for storing selected path
CONFIG_FILE="$HOME/.efx-face-manager.conf"

# Function to detect and set default MODEL_DIR
detect_default_path() {
    # Check if external drive is mounted
    if [[ -d "/Volumes/T7" ]]; then
        echo "$EXTERNAL_MODEL_PATH"
    else
        echo "$LOCAL_MODEL_PATH"
    fi
}

# Function to load saved path from config
load_model_path() {
    if [[ -f "$CONFIG_FILE" ]]; then
        local saved_path=$(cat "$CONFIG_FILE")
        # Validate saved path exists or is creatable
        if [[ -n "$saved_path" ]]; then
            echo "$saved_path"
            return
        fi
    fi
    # No saved config, use auto-detection
    detect_default_path
}

# Function to save selected path to config
save_model_path() {
    local path="$1"
    echo "$path" > "$CONFIG_FILE"
}

# Function to get path status and description
get_path_status() {
    local path="$1"
    local model_count=0
    
    if [[ -d "$path" ]]; then
        model_count=$(find "$path" -maxdepth 1 -type l 2>/dev/null | wc -l | tr -d ' ')
        if [[ $model_count -gt 0 ]]; then
            echo "✓ Active ($model_count models)"
        else
            echo "✓ Available (no models)"
        fi
    elif [[ "$path" == "$EXTERNAL_MODEL_PATH" ]]; then
        echo "✗ External drive not mounted"
    else
        echo "○ Not created"
    fi
}

# Function to show and select model storage path
configure_model_path() {
    local external_status=$(get_path_status "$EXTERNAL_MODEL_PATH")
    local local_status=$(get_path_status "$LOCAL_MODEL_PATH")
    
    # Build menu items with current marker
    local external_marker=""
    [[ "$MODEL_DIR" == "$EXTERNAL_MODEL_PATH" ]] && external_marker=" ← Current"
    local external_line="External: $EXTERNAL_MODEL_PATH [$external_status]$external_marker"
    
    local local_marker=""
    [[ "$MODEL_DIR" == "$LOCAL_MODEL_PATH" ]] && local_marker=" ← Current"
    local local_line="Local: $LOCAL_MODEL_PATH [$local_status]$local_marker"
    
    local selection=$(gum choose \
        --header "Configure Model Storage Path" \
        "$external_line" \
        "$local_line" \
        "Auto-detect (External → Local fallback)" \
        "✖ Back")
    
    case "$selection" in
        "External:"*)
            if [[ -d "/Volumes/T7" ]]; then
                MODEL_DIR="$EXTERNAL_MODEL_PATH"
                mkdir -p "$MODEL_DIR"
                save_model_path "$MODEL_DIR"
                gum style --foreground 212 "✓ Switched to External path: $MODEL_DIR"
                sleep 1
            else
                gum style --foreground 196 "✗ External drive not mounted at /Volumes/T7"
                sleep 2
                configure_model_path  # Show menu again
            fi
            ;;
        "Local:"*)
            MODEL_DIR="$LOCAL_MODEL_PATH"
            mkdir -p "$MODEL_DIR"
            save_model_path "$MODEL_DIR"
            gum style --foreground 212 "✓ Switched to Local path: $MODEL_DIR"
            sleep 1
            ;;
        *"Auto-detect"*)
            MODEL_DIR=$(detect_default_path)
            mkdir -p "$MODEL_DIR"
            save_model_path "$MODEL_DIR"
            gum style --foreground 212 "✓ Auto-detected path: $MODEL_DIR"
            sleep 1
            ;;
        *"Back"*|"")
            return
            ;;
    esac
}

# Initialize MODEL_DIR - Load from config or auto-detect
export MODEL_DIR=$(load_model_path)

# Ensure directory exists
mkdir -p "$MODEL_DIR"

# Global array for command arguments (used for Bash 3.x compatibility)
CMD_ARGS=()

# Preset configurations for each model type
# Returns: preset description for display
get_preset_description() {
    local preset=$1
    case "$preset" in
        "lm") echo "lm (text-only)" ;;
        "multimodal") echo "multimodal (vision, audio)" ;;
        "image-generation") echo "image-generation (qwen-image, q16)" ;;
        "image-edit") echo "image-edit (qwen-image-edit, q16)" ;;
        "embeddings") echo "embeddings" ;;
        "whisper") echo "whisper (audio transcription)" ;;
    esac
}

# Apply preset configuration to CMD_ARGS
apply_preset_config() {
    local preset=$1
    local model_path=$2
    
    CMD_ARGS=("--model-path" "$model_path" "--model-type" "$preset" "--host" "0.0.0.0" "--port" "8000")
    
    case "$preset" in
        "image-generation")
            CMD_ARGS+=("--config-name" "qwen-image" "--quantize" "16")
            ;;
        "image-edit")
            CMD_ARGS+=("--config-name" "qwen-image-edit" "--quantize" "16")
            ;;
    esac
}

# Show command preview and confirm launch
confirm_and_launch() {
    local full_cmd="mlx-openai-server launch ${CMD_ARGS[*]}"
    local display_dir
    display_dir=$(get_model_dir_display)
    
    echo ""
    gum style \
        --border rounded \
        --border-foreground 99 \
        --padding "1 2" \
        --width 80 \
        "$(gum style --bold --foreground 212 "Command to execute:")

Models: $display_dir

$full_cmd"
    echo ""
    
    local confirm=$(gum choose \
        --header "Models: $display_dir | Ready to launch?" \
        "▶ Run" \
        "✖ Cancel")
    
    if [[ "$confirm" == "▶ Run" ]]; then
        echo "Starting MLX OpenAI Server..."
        mlx-openai-server launch "${CMD_ARGS[@]}"
        return 0
    else
        return 1
    fi
}

# Handle preset action menu (Run / Modify config / Cancel)
handle_preset_action() {
    local model_name="$1"
    local preset="$2"
    local model_path="$MODEL_DIR/$model_name"
    local preset_desc=$(get_preset_description "$preset")
    local trust_remote_code=false
    
    # Apply preset configuration
    apply_preset_config "$preset" "$model_path"
    
    # Build command preview
    local full_cmd="mlx-openai-server launch ${CMD_ARGS[*]}"
    local display_dir
    display_dir=$(get_model_dir_display)
    
    # Show combined page with preset info and command details
    while true; do
        # Rebuild command with trust_remote_code flag if enabled
        if [[ $trust_remote_code == true ]]; then
            # Check if --trust-remote-code is not already in CMD_ARGS
            if [[ ! " ${CMD_ARGS[*]} " =~ " --trust-remote-code " ]]; then
                CMD_ARGS+=("--trust-remote-code")
            fi
        else
            # Remove --trust-remote-code if present
            local new_args=()
            for arg in "${CMD_ARGS[@]}"; do
                [[ "$arg" != "--trust-remote-code" ]] && new_args+=("$arg")
            done
            CMD_ARGS=("${new_args[@]}")
        fi
        full_cmd="mlx-openai-server launch ${CMD_ARGS[*]}"
        
        # Clear screen before redrawing
        clear
        show_header
        
        echo ""
        gum style \
            --border rounded \
            --border-foreground 99 \
            --padding "1 2" \
            --width 80 \
            "$(gum style --bold --foreground 212 "Model: $model_name")
$(gum style --bold --foreground 212 "Configuration: $preset_desc")

Models: $display_dir

$(gum style --bold --foreground 212 "Command to execute:")

$full_cmd"
        echo ""
        
        # Build menu based on preset type (only lm/multimodal need trust_remote_code)
        local menu_items=()
        if [[ "$preset" == "lm" || "$preset" == "multimodal" ]]; then
            local trust_status="disabled"
            [[ $trust_remote_code == true ]] && trust_status="enabled"
            menu_items=("▶ Run" "🔐 Trust remote code: $trust_status" "⚙️  Modify config..." "✖ Cancel")
        else
            menu_items=("▶ Run" "⚙️  Modify config..." "✖ Cancel")
        fi
        
        local action=$(gum choose \
            --header "Models: $display_dir | Ready to launch?" \
            "${menu_items[@]}")
        
        case "$action" in
            "▶ Run")
                echo "Starting MLX OpenAI Server..."
                mlx-openai-server launch "${CMD_ARGS[@]}"
                # Server has stopped (either completed or interrupted)
                echo ""
                gum style --foreground 212 "Server stopped."
                gum confirm "Press Enter to return to main menu..."
                return 0  # Success, exit to main menu
                ;;
            "🔐 Trust remote code:"*)
                # Toggle trust_remote_code
                [[ $trust_remote_code == true ]] && trust_remote_code=false || trust_remote_code=true
                ;;
            "⚙️  Modify config...")
                # Open type-specific configuration
                local config_result=0
                case "$preset" in
                    "lm"|"multimodal")
                        configure_lm_options "$preset" || config_result=1
                        ;;
                    "image-generation")
                        configure_image_gen_options_with_defaults || config_result=1
                        ;;
                    "image-edit")
                        configure_image_edit_options_with_defaults || config_result=1
                        ;;
                    "whisper")
                        configure_whisper_options || config_result=1
                        ;;
                    "embeddings")
                        configure_server_options || config_result=1
                        ;;
                esac
                
                if [[ $config_result -eq 0 ]]; then
                    # Update command preview after configuration
                    full_cmd="mlx-openai-server launch ${CMD_ARGS[*]}"
                    # Loop back to show updated command
                else
                    # If cancelled, stay in preset action menu
                    continue
                fi
                ;;
            *)
                return 1  # Cancel, go back to preset selection
                ;;
        esac
    done
}

# Function to run an installed model with default settings
run_model_default() {
    local model_name="$1"
    local model_path="$MODEL_DIR/$model_name"
    
    if [[ -L "$model_path" ]]; then
        echo "Starting MLX OpenAI Server with $model_name..."
        mlx-openai-server launch --model-path "$model_path" --model-type lm
    else
        echo "Error: Model not found or not a symlink: $model_path"
    fi
}

# Function to run model with custom options
run_model_with_options() {
    local model_name="$1"
    local model_path="$MODEL_DIR/$model_name"
    
    if [[ ! -L "$model_path" ]]; then
        echo "Error: Model not found or not a symlink: $model_path"
        return 1
    fi
    
    # Reset global command arguments
    CMD_ARGS=("--model-path" "$model_path")
    
    # Model type selection loop
    while true; do
        local model_type=$(gum choose \
            --header "Select Model Type" \
            "lm (text-only)" \
            "multimodal (text, vision, audio)" \
            "image-generation" \
            "image-edit" \
            "embeddings" \
            "whisper (audio transcription)" \
            "✖ Back")
        
        if [[ -z "$model_type" || "$model_type" == "✖ Back" ]]; then
            return 1
        fi
        
        # Extract model type value
        local type_value
        case "$model_type" in
            "lm (text-only)") type_value="lm" ;;
            "multimodal (text, vision, audio)") type_value="multimodal" ;;
            "whisper (audio transcription)") type_value="whisper" ;;
            *) type_value="$model_type" ;;
        esac
        
        CMD_ARGS+=("--model-type" "$type_value")
        
        # Get type-specific options (if cancelled, return to run mode menu)
        local config_result=0
        case "$type_value" in
            "lm"|"multimodal")
                configure_lm_options "$type_value" || config_result=1
                ;;
            "image-generation")
                configure_image_gen_options || config_result=1
                ;;
            "image-edit")
                configure_image_edit_options || config_result=1
                ;;
            "whisper")
                configure_whisper_options || config_result=1
                ;;
            "embeddings")
                configure_server_options || config_result=1
                ;;
        esac
        
        # If cancelled, return to run mode menu
        if [[ $config_result -eq 1 ]]; then
            return 1
        fi
        
        # Show final command and confirm
        local full_cmd="mlx-openai-server launch ${CMD_ARGS[*]}"
        echo ""
        gum style \
            --border rounded \
            --border-foreground 99 \
            --padding "1 2" \
            --width 80 \
            "$(gum style --bold --foreground 212 "Command to execute:")

$full_cmd"
        echo ""
        
        local confirm=$(gum choose \
            --header "Ready to launch?" \
            "▶ Run" \
            "✖ Cancel")
        
        if [[ "$confirm" == "▶ Run" ]]; then
            echo "Starting MLX OpenAI Server..."
            mlx-openai-server launch "${CMD_ARGS[@]}"
            # Server has stopped (either completed or interrupted)
            echo ""
            gum style --foreground 212 "Server stopped."
            gum confirm "Press Enter to return to main menu..."
            return 0
        else
            return 1
        fi
    done
}

# Configure options for lm/multimodal models
configure_lm_options() {
    local model_type=$1
    
    # Extract model path from existing CMD_ARGS
    local model_path=$(echo "${CMD_ARGS[@]}" | grep -o "\-\-model-path [^ ]*" | cut -d' ' -f2)
    
    # Initialize option values
    local context_length_val=""
    local auto_tool_choice=false
    local tool_call_parser_val=""
    local reasoning_parser_val=""
    local message_converter_val=""
    local trust_remote_code=false
    local chat_template_file_val=""
    local debug_mode=false
    local disable_auto_resize=false
    local port_val="8000"
    local host_val="0.0.0.0"
    local log_level_val=""
    
    # Interactive settings loop
    while true; do
        clear
        show_header
        
        # Build options list based on model type
        local multimodal_line=""
        [[ "$model_type" == "multimodal" ]] && multimodal_line="Disable auto resize: $([[ $disable_auto_resize == true ]] && echo 'enabled' || echo 'disabled')
"
        
        local menu="Context length: ${context_length_val:-(not set)}
Auto tool choice: $([[ $auto_tool_choice == true ]] && echo 'enabled' || echo 'disabled')
Tool call parser: ${tool_call_parser_val:-(not set)}
Reasoning parser: ${reasoning_parser_val:-(not set)}
Message converter: ${message_converter_val:-(not set)}
Chat template file: ${chat_template_file_val:-(not set)}
Debug mode: $([[ $debug_mode == true ]] && echo 'enabled' || echo 'disabled')
Trust remote code: $([[ $trust_remote_code == true ]] && echo 'enabled' || echo 'disabled')
${multimodal_line}Port: $port_val
Host: $host_val
Log level: ${log_level_val:-INFO (default)}
──────────
✅ Done - continue to launch
✖ Cancel"
        
        local selection=$(echo "$menu" | gum choose --height 20 \
            --header "Configure $model_type options - Select to edit")
        
        case "$selection" in
            "Context length:"*)
                context_length_val=$(gum input --placeholder "8192" --header "Context length" --value "$context_length_val")
                ;;
            "Auto tool choice:"*)
                [[ $auto_tool_choice == true ]] && auto_tool_choice=false || auto_tool_choice=true
                ;;
            "Tool call parser:"*)
                tool_call_parser_val=$(gum choose --header "Tool call parser" \
                    "qwen3" "glm4_moe" "qwen3_coder" "qwen3_moe" "qwen3_next" "qwen3_vl" "harmony" "minimax_m2" "(clear)")
                [[ "$tool_call_parser_val" == "(clear)" ]] && tool_call_parser_val=""
                ;;
            "Reasoning parser:"*)
                reasoning_parser_val=$(gum choose --header "Reasoning parser" \
                    "qwen3" "glm4_moe" "qwen3_coder" "qwen3_moe" "qwen3_next" "qwen3_vl" "harmony" "minimax_m2" "(clear)")
                [[ "$reasoning_parser_val" == "(clear)" ]] && reasoning_parser_val=""
                ;;
            "Message converter:"*)
                message_converter_val=$(gum choose --header "Message converter" \
                    "glm4_moe" "minimax_m2" "nemotron3_nano" "qwen3_coder" "(clear)")
                [[ "$message_converter_val" == "(clear)" ]] && message_converter_val=""
                ;;
            "Trust remote code:"*)
                [[ $trust_remote_code == true ]] && trust_remote_code=false || trust_remote_code=true
                ;;
            "Chat template file:"*)
                chat_template_file_val=$(gum input --placeholder "/path/to/template.jinja" --header "Chat template file path" --value "$chat_template_file_val")
                ;;
            "Debug mode:"*)
                [[ $debug_mode == true ]] && debug_mode=false || debug_mode=true
                ;;
            "Disable auto resize:"*)
                [[ $disable_auto_resize == true ]] && disable_auto_resize=false || disable_auto_resize=true
                ;;
            "Port:"*)
                port_val=$(gum input --placeholder "8000" --header "Server port" --value "$port_val")
                ;;
            "Host:"*)
                host_val=$(gum input --placeholder "0.0.0.0" --header "Server host" --value "$host_val")
                ;;
            "Log level:"*)
                log_level_val=$(gum choose --header "Log level" "DEBUG" "INFO" "WARNING" "ERROR" "CRITICAL" "(clear)")
                [[ "$log_level_val" == "(clear)" ]] && log_level_val=""
                ;;
            *"Done"*)
                # Rebuild CMD_ARGS from scratch to avoid conflicts
                CMD_ARGS=("--model-path" "$model_path" "--model-type" "$model_type")
                [[ -n "$context_length_val" ]] && CMD_ARGS+=("--context-length" "$context_length_val")
                [[ $auto_tool_choice == true ]] && CMD_ARGS+=("--enable-auto-tool-choice")
                [[ -n "$tool_call_parser_val" ]] && CMD_ARGS+=("--tool-call-parser" "$tool_call_parser_val")
                [[ -n "$reasoning_parser_val" ]] && CMD_ARGS+=("--reasoning-parser" "$reasoning_parser_val")
                [[ -n "$message_converter_val" ]] && CMD_ARGS+=("--message-converter" "$message_converter_val")
                [[ $trust_remote_code == true ]] && CMD_ARGS+=("--trust-remote-code")
                [[ -n "$chat_template_file_val" ]] && CMD_ARGS+=("--chat-template-file" "$chat_template_file_val")
                [[ $debug_mode == true ]] && CMD_ARGS+=("--debug")
                [[ $disable_auto_resize == true ]] && CMD_ARGS+=("--disable-auto-resize")
                CMD_ARGS+=("--port" "$port_val")
                CMD_ARGS+=("--host" "$host_val")
                [[ -n "$log_level_val" ]] && CMD_ARGS+=("--log-level" "$log_level_val")
                break
                ;;
            *"Cancel"*|""|"─"*)
                return 1
                ;;
        esac
    done
}

# Configure options for image-generation models
configure_image_gen_options() {
    # Config name is required for image-generation
    local config=$(gum choose --header "Select config-name (required)" \
        "flux-schnell" "flux-dev" "flux-krea-dev" "qwen-image" "z-image-turbo" "fibo")
    
    if [[ -z "$config" ]]; then
        config="flux-schnell"
    fi
    CMD_ARGS+=("--config-name" "$config")
    
    # Initialize option values
    local quantize_val=""
    local lora_paths_val=""
    local lora_scales_val=""
    local port_val=""
    local host_val=""
    local log_level_val=""
    
    # Interactive settings loop
    while true; do
        clear
        show_header
        
        local menu="Quantize level: ${quantize_val:-(not set)}
LoRA paths: ${lora_paths_val:-(not set)}
LoRA scales: ${lora_scales_val:-(not set)}
Port: ${port_val:-8000 (default)}
Host: ${host_val:-0.0.0.0 (default)}
Log level: ${log_level_val:-INFO (default)}
──────────
✅ Done - continue to launch
✖ Cancel"
        
        local selection=$(echo "$menu" | gum choose --height 15 \
            --header "Configure image-generation options ($config) - Select to edit")
        
        case "$selection" in
            "Quantize level:"*)
                quantize_val=$(gum choose --header "Quantize level" "4" "8" "16" "(clear)")
                [[ "$quantize_val" == "(clear)" ]] && quantize_val=""
                ;;
            "LoRA paths:"*)
                lora_paths_val=$(gum input --placeholder "/path/to/lora1.safetensors,/path/to/lora2.safetensors" --header "LoRA paths (comma-separated)" --value "$lora_paths_val")
                if [[ -n "$lora_paths_val" ]]; then
                    lora_scales_val=$(gum input --placeholder "0.8,0.6" --header "LoRA scales (must match paths count)" --value "$lora_scales_val")
                fi
                ;;
            "LoRA scales:"*)
                lora_scales_val=$(gum input --placeholder "0.8,0.6" --header "LoRA scales" --value "$lora_scales_val")
                ;;
            "Port:"*)
                port_val=$(gum input --placeholder "8000" --header "Server port" --value "$port_val")
                ;;
            "Host:"*)
                host_val=$(gum input --placeholder "0.0.0.0" --header "Server host" --value "$host_val")
                ;;
            "Log level:"*)
                log_level_val=$(gum choose --header "Log level" "DEBUG" "INFO" "WARNING" "ERROR" "CRITICAL" "(clear)")
                [[ "$log_level_val" == "(clear)" ]] && log_level_val=""
                ;;
            *"Done"*)
                [[ -n "$quantize_val" ]] && CMD_ARGS+=("--quantize" "$quantize_val")
                [[ -n "$lora_paths_val" ]] && CMD_ARGS+=("--lora-paths" "$lora_paths_val")
                [[ -n "$lora_scales_val" ]] && CMD_ARGS+=("--lora-scales" "$lora_scales_val")
                [[ -n "$port_val" ]] && CMD_ARGS+=("--port" "$port_val")
                [[ -n "$host_val" ]] && CMD_ARGS+=("--host" "$host_val")
                [[ -n "$log_level_val" ]] && CMD_ARGS+=("--log-level" "$log_level_val")
                break
                ;;
            *"Cancel"*|""|"─"*)
                return 1
                ;;
        esac
    done
}

# Configure options for image-generation with preset defaults (qwen-image, q16)
configure_image_gen_options_with_defaults() {
    # Initialize option values with preset defaults
    local config_val="qwen-image"
    local quantize_val="16"
    local lora_paths_val=""
    local lora_scales_val=""
    local port_val="8000"
    local host_val="0.0.0.0"
    local log_level_val=""
    
    # Interactive settings loop
    while true; do
        clear
        show_header
        
        local menu="Config name: $config_val
Quantize level: $quantize_val
LoRA paths: ${lora_paths_val:-(not set)}
LoRA scales: ${lora_scales_val:-(not set)}
Port: $port_val
Host: $host_val
Log level: ${log_level_val:-INFO (default)}
──────────
✅ Done - continue to launch
✖ Cancel"
        
        local selection=$(echo "$menu" | gum choose --height 15 \
            --header "Configure image-generation options - Select to edit")
        
        case "$selection" in
            "Config name:"*)
                config_val=$(gum choose --header "Select config-name" \
                    "flux-schnell" "flux-dev" "flux-krea-dev" "qwen-image" "z-image-turbo" "fibo")
                [[ -z "$config_val" ]] && config_val="qwen-image"
                ;;
            "Quantize level:"*)
                quantize_val=$(gum choose --header "Quantize level" "4" "8" "16")
                [[ -z "$quantize_val" ]] && quantize_val="16"
                ;;
            "LoRA paths:"*)
                lora_paths_val=$(gum input --placeholder "/path/to/lora1.safetensors" --header "LoRA paths (comma-separated)" --value "$lora_paths_val")
                if [[ -n "$lora_paths_val" ]]; then
                    lora_scales_val=$(gum input --placeholder "0.8,0.6" --header "LoRA scales (must match paths count)" --value "$lora_scales_val")
                fi
                ;;
            "LoRA scales:"*)
                lora_scales_val=$(gum input --placeholder "0.8,0.6" --header "LoRA scales" --value "$lora_scales_val")
                ;;
            "Port:"*)
                port_val=$(gum input --placeholder "8000" --header "Server port" --value "$port_val")
                [[ -z "$port_val" ]] && port_val="8000"
                ;;
            "Host:"*)
                host_val=$(gum input --placeholder "0.0.0.0" --header "Server host" --value "$host_val")
                [[ -z "$host_val" ]] && host_val="0.0.0.0"
                ;;
            "Log level:"*)
                log_level_val=$(gum choose --header "Log level" "DEBUG" "INFO" "WARNING" "ERROR" "CRITICAL" "(clear)")
                [[ "$log_level_val" == "(clear)" ]] && log_level_val=""
                ;;
            *"Done"*)
                # Rebuild CMD_ARGS with configured values
                local model_path=$(echo "${CMD_ARGS[@]}" | grep -o "\-\-model-path [^ ]*" | cut -d' ' -f2)
                CMD_ARGS=("--model-path" "$model_path" "--model-type" "image-generation")
                CMD_ARGS+=("--config-name" "$config_val")
                CMD_ARGS+=("--quantize" "$quantize_val")
                [[ -n "$lora_paths_val" ]] && CMD_ARGS+=("--lora-paths" "$lora_paths_val")
                [[ -n "$lora_scales_val" ]] && CMD_ARGS+=("--lora-scales" "$lora_scales_val")
                CMD_ARGS+=("--port" "$port_val")
                CMD_ARGS+=("--host" "$host_val")
                [[ -n "$log_level_val" ]] && CMD_ARGS+=("--log-level" "$log_level_val")
                break
                ;;
            *"Cancel"*|""|"─"*)
                return 1
                ;;
        esac
    done
}

# Configure options for image-edit models
configure_image_edit_options() {
    # Config name is required for image-edit
    local config=$(gum choose --header "Select config-name (required)" \
        "flux-kontext-dev" "qwen-image-edit")
    
    if [[ -z "$config" ]]; then
        config="flux-kontext-dev"
    fi
    CMD_ARGS+=("--config-name" "$config")
    
    # Initialize option values
    local quantize_val=""
    local lora_paths_val=""
    local lora_scales_val=""
    local port_val=""
    local host_val=""
    local log_level_val=""
    
    # Interactive settings loop
    while true; do
        clear
        show_header
        
        local menu="Quantize level: ${quantize_val:-(not set)}
LoRA paths: ${lora_paths_val:-(not set)}
LoRA scales: ${lora_scales_val:-(not set)}
Port: ${port_val:-8000 (default)}
Host: ${host_val:-0.0.0.0 (default)}
Log level: ${log_level_val:-INFO (default)}
──────────
✅ Done - continue to launch
✖ Cancel"
        
        local selection=$(echo "$menu" | gum choose --height 15 \
            --header "Configure image-edit options ($config) - Select to edit")
        
        case "$selection" in
            "Quantize level:"*)
                quantize_val=$(gum choose --header "Quantize level" "4" "8" "16" "(clear)")
                [[ "$quantize_val" == "(clear)" ]] && quantize_val=""
                ;;
            "LoRA paths:"*)
                lora_paths_val=$(gum input --placeholder "/path/to/lora1.safetensors,/path/to/lora2.safetensors" --header "LoRA paths (comma-separated)" --value "$lora_paths_val")
                if [[ -n "$lora_paths_val" ]]; then
                    lora_scales_val=$(gum input --placeholder "0.8,0.6" --header "LoRA scales (must match paths count)" --value "$lora_scales_val")
                fi
                ;;
            "LoRA scales:"*)
                lora_scales_val=$(gum input --placeholder "0.8,0.6" --header "LoRA scales" --value "$lora_scales_val")
                ;;
            "Port:"*)
                port_val=$(gum input --placeholder "8000" --header "Server port" --value "$port_val")
                ;;
            "Host:"*)
                host_val=$(gum input --placeholder "0.0.0.0" --header "Server host" --value "$host_val")
                ;;
            "Log level:"*)
                log_level_val=$(gum choose --header "Log level" "DEBUG" "INFO" "WARNING" "ERROR" "CRITICAL" "(clear)")
                [[ "$log_level_val" == "(clear)" ]] && log_level_val=""
                ;;
            *"Done"*)
                [[ -n "$quantize_val" ]] && CMD_ARGS+=("--quantize" "$quantize_val")
                [[ -n "$lora_paths_val" ]] && CMD_ARGS+=("--lora-paths" "$lora_paths_val")
                [[ -n "$lora_scales_val" ]] && CMD_ARGS+=("--lora-scales" "$lora_scales_val")
                [[ -n "$port_val" ]] && CMD_ARGS+=("--port" "$port_val")
                [[ -n "$host_val" ]] && CMD_ARGS+=("--host" "$host_val")
                [[ -n "$log_level_val" ]] && CMD_ARGS+=("--log-level" "$log_level_val")
                break
                ;;
            *"Cancel"*|""|"─"*)
                return 1
                ;;
        esac
    done
}

# Configure options for image-edit with preset defaults (qwen-image-edit, q16)
configure_image_edit_options_with_defaults() {
    # Initialize option values with preset defaults
    local config_val="qwen-image-edit"
    local quantize_val="16"
    local lora_paths_val=""
    local lora_scales_val=""
    local port_val="8000"
    local host_val="0.0.0.0"
    local log_level_val=""
    
    # Interactive settings loop
    while true; do
        clear
        show_header
        
        local menu="Config name: $config_val
Quantize level: $quantize_val
LoRA paths: ${lora_paths_val:-(not set)}
LoRA scales: ${lora_scales_val:-(not set)}
Port: $port_val
Host: $host_val
Log level: ${log_level_val:-INFO (default)}
──────────
✅ Done - continue to launch
✖ Cancel"
        
        local selection=$(echo "$menu" | gum choose --height 15 \
            --header "Configure image-edit options - Select to edit")
        
        case "$selection" in
            "Config name:"*)
                config_val=$(gum choose --header "Select config-name" \
                    "flux-kontext-dev" "qwen-image-edit")
                [[ -z "$config_val" ]] && config_val="qwen-image-edit"
                ;;
            "Quantize level:"*)
                quantize_val=$(gum choose --header "Quantize level" "4" "8" "16")
                [[ -z "$quantize_val" ]] && quantize_val="16"
                ;;
            "LoRA paths:"*)
                lora_paths_val=$(gum input --placeholder "/path/to/lora1.safetensors" --header "LoRA paths (comma-separated)" --value "$lora_paths_val")
                if [[ -n "$lora_paths_val" ]]; then
                    lora_scales_val=$(gum input --placeholder "0.8,0.6" --header "LoRA scales (must match paths count)" --value "$lora_scales_val")
                fi
                ;;
            "LoRA scales:"*)
                lora_scales_val=$(gum input --placeholder "0.8,0.6" --header "LoRA scales" --value "$lora_scales_val")
                ;;
            "Port:"*)
                port_val=$(gum input --placeholder "8000" --header "Server port" --value "$port_val")
                [[ -z "$port_val" ]] && port_val="8000"
                ;;
            "Host:"*)
                host_val=$(gum input --placeholder "0.0.0.0" --header "Server host" --value "$host_val")
                [[ -z "$host_val" ]] && host_val="0.0.0.0"
                ;;
            "Log level:"*)
                log_level_val=$(gum choose --header "Log level" "DEBUG" "INFO" "WARNING" "ERROR" "CRITICAL" "(clear)")
                [[ "$log_level_val" == "(clear)" ]] && log_level_val=""
                ;;
            *"Done"*)
                # Rebuild CMD_ARGS with configured values
                local model_path=$(echo "${CMD_ARGS[@]}" | grep -o "\-\-model-path [^ ]*" | cut -d' ' -f2)
                CMD_ARGS=("--model-path" "$model_path" "--model-type" "image-edit")
                CMD_ARGS+=("--config-name" "$config_val")
                CMD_ARGS+=("--quantize" "$quantize_val")
                [[ -n "$lora_paths_val" ]] && CMD_ARGS+=("--lora-paths" "$lora_paths_val")
                [[ -n "$lora_scales_val" ]] && CMD_ARGS+=("--lora-scales" "$lora_scales_val")
                CMD_ARGS+=("--port" "$port_val")
                CMD_ARGS+=("--host" "$host_val")
                [[ -n "$log_level_val" ]] && CMD_ARGS+=("--log-level" "$log_level_val")
                break
                ;;
            *"Cancel"*|""|"─"*)
                return 1
                ;;
        esac
    done
}

# Configure options for whisper models
configure_whisper_options() {
    # Extract model path from existing CMD_ARGS
    local model_path=$(echo "${CMD_ARGS[@]}" | grep -o "\-\-model-path [^ ]*" | cut -d' ' -f2)
    
    # Initialize option values
    local max_concurrency_val="1"
    local queue_timeout_val="600"
    local queue_size_val="50"
    local port_val="8000"
    local host_val="0.0.0.0"
    local log_level_val=""
    
    # Interactive settings loop
    while true; do
        clear
        show_header
        
        local menu="Max concurrency: $max_concurrency_val
Queue timeout: $queue_timeout_val
Queue size: $queue_size_val
Port: $port_val
Host: $host_val
Log level: ${log_level_val:-INFO (default)}
──────────
✅ Done - continue to launch
✖ Cancel"
        
        local selection=$(echo "$menu" | gum choose --height 15 \
            --header "Configure whisper options - Select to edit")
        
        case "$selection" in
            "Max concurrency:"*)
                max_concurrency_val=$(gum input --placeholder "1" --header "Max concurrency" --value "$max_concurrency_val")
                ;;
            "Queue timeout:"*)
                queue_timeout_val=$(gum input --placeholder "600" --header "Queue timeout (seconds)" --value "$queue_timeout_val")
                ;;
            "Queue size:"*)
                queue_size_val=$(gum input --placeholder "50" --header "Queue size" --value "$queue_size_val")
                ;;
            "Port:"*)
                port_val=$(gum input --placeholder "8000" --header "Server port" --value "$port_val")
                ;;
            "Host:"*)
                host_val=$(gum input --placeholder "0.0.0.0" --header "Server host" --value "$host_val")
                ;;
            "Log level:"*)
                log_level_val=$(gum choose --header "Log level" "DEBUG" "INFO" "WARNING" "ERROR" "CRITICAL" "(clear)")
                [[ "$log_level_val" == "(clear)" ]] && log_level_val=""
                ;;
            *"Done"*)
                # Rebuild CMD_ARGS from scratch to avoid conflicts
                CMD_ARGS=("--model-path" "$model_path" "--model-type" "whisper")
                CMD_ARGS+=("--max-concurrency" "$max_concurrency_val")
                CMD_ARGS+=("--queue-timeout" "$queue_timeout_val")
                CMD_ARGS+=("--queue-size" "$queue_size_val")
                CMD_ARGS+=("--port" "$port_val")
                CMD_ARGS+=("--host" "$host_val")
                [[ -n "$log_level_val" ]] && CMD_ARGS+=("--log-level" "$log_level_val")
                break
                ;;
            *"Cancel"*|""|"─"*)
                return 1
                ;;
        esac
    done
}

# Configure common server options
configure_server_options() {
    # Extract model path from existing CMD_ARGS
    local model_path=$(echo "${CMD_ARGS[@]}" | grep -o "\-\-model-path [^ ]*" | cut -d' ' -f2)
    
    # Initialize option values
    local port_val="8000"
    local host_val="0.0.0.0"
    local log_level_val=""
    local max_concurrency_val="1"
    local queue_timeout_val="300"
    local queue_size_val="100"
    
    # Interactive settings loop
    while true; do
        clear
        show_header
        
        local menu="Port: $port_val
Host: $host_val
Log level: ${log_level_val:-INFO (default)}
Max concurrency: $max_concurrency_val
Queue timeout: $queue_timeout_val
Queue size: $queue_size_val
──────────
✅ Done - continue to launch
✖ Cancel"
        
        local selection=$(echo "$menu" | gum choose --height 15 \
            --header "Configure server options - Select to edit")
        
        case "$selection" in
            "Port:"*)
                port_val=$(gum input --placeholder "8000" --header "Server port" --value "$port_val")
                ;;
            "Host:"*)
                host_val=$(gum input --placeholder "0.0.0.0" --header "Server host" --value "$host_val")
                ;;
            "Log level:"*)
                log_level_val=$(gum choose --header "Log level" "DEBUG" "INFO" "WARNING" "ERROR" "CRITICAL" "(clear)")
                [[ "$log_level_val" == "(clear)" ]] && log_level_val=""
                ;;
            "Max concurrency:"*)
                max_concurrency_val=$(gum input --placeholder "1" --header "Max concurrency" --value "$max_concurrency_val")
                ;;
            "Queue timeout:"*)
                queue_timeout_val=$(gum input --placeholder "300" --header "Queue timeout (seconds)" --value "$queue_timeout_val")
                ;;
            "Queue size:"*)
                queue_size_val=$(gum input --placeholder "100" --header "Queue size" --value "$queue_size_val")
                ;;
            *"Done"*)
                # Rebuild CMD_ARGS from scratch to avoid conflicts
                CMD_ARGS=("--model-path" "$model_path" "--model-type" "embeddings")
                CMD_ARGS+=("--port" "$port_val")
                CMD_ARGS+=("--host" "$host_val")
                [[ -n "$log_level_val" ]] && CMD_ARGS+=("--log-level" "$log_level_val")
                CMD_ARGS+=("--max-concurrency" "$max_concurrency_val")
                CMD_ARGS+=("--queue-timeout" "$queue_timeout_val")
                CMD_ARGS+=("--queue-size" "$queue_size_val")
                break
                ;;
            *"Cancel"*|""|"─"*)
                return 1
                ;;
        esac
    done
}

# Function to list available models
list_models() {
    echo "Available Models:"
    echo "----------------"
    find "$MODEL_DIR" -maxdepth 1 -type l | while read -r model; do
        model_name=$(basename "$model")
        target=$(readlink "$model")
        echo "• $model_name"
        echo "  → $target"
        echo ""
    done
}

# Function to show model details and confirm installation
show_model_details() {
    local repo_id="$1"
    local display_dir
    display_dir=$(get_model_dir_display)
    
    echo ""
    gum spin --spinner dot --title "Fetching details for $repo_id..." -- sleep 0.5 &
    
    # Fetch model details from HuggingFace API
    local model_info=$(curl -s "https://huggingface.co/api/models/$repo_id")
    
    # Kill spinner
    kill %1 2>/dev/null
    
    if [[ -z "$model_info" || "$model_info" == "null" ]]; then
        echo "Could not fetch model details."
        return 1
    fi
    
    # Extract details
    local downloads=$(echo "$model_info" | jq -r '.downloads // "N/A"')
    local likes=$(echo "$model_info" | jq -r '.likes // "N/A"')
    local pipeline=$(echo "$model_info" | jq -r '.pipeline_tag // "N/A"')
    local library=$(echo "$model_info" | jq -r '.library_name // "N/A"')
    local last_modified=$(echo "$model_info" | jq -r '.lastModified // "N/A"' | cut -d'T' -f1)
    
    # Get size info from siblings if available
    local total_size=$(echo "$model_info" | jq -r '[.siblings[]?.size // 0] | add' 2>/dev/null)
    local size_display="N/A"
    if [[ -n "$total_size" && "$total_size" != "null" && "$total_size" != "0" ]]; then
        if [[ $total_size -gt 1073741824 ]]; then
            size_display="$(awk "BEGIN {printf \"%.2f\", $total_size / 1073741824}") GB"
        elif [[ $total_size -gt 1048576 ]]; then
            size_display="$(awk "BEGIN {printf \"%.2f\", $total_size / 1048576}") MB"
        else
            size_display="$total_size bytes"
        fi
    fi
    
    # Display details in a styled box
    echo ""
    gum style \
        --border double \
        --border-foreground 99 \
        --padding "1 2" \
        --width 60 \
        "$(gum style --bold --foreground 212 "$repo_id")

Models:       $display_dir

Downloads:    $downloads
Likes:        $likes
Pipeline:     $pipeline
Library:      $library
Updated:      $last_modified
Size:         $size_display"
    
    echo ""
    
    # Show action menu
    local action=$(gum choose \
        --header "Models: $display_dir | What would you like to do?" \
        "⬇️  Install this LLM" \
        "🔗 Open in Browser" \
        "❌ Cancel")
    
    case "$action" in
        "⬇️  Install this LLM")
            return 0
            ;;
        "🔗 Open in Browser")
            open "https://huggingface.co/$repo_id" 2>/dev/null || xdg-open "https://huggingface.co/$repo_id" 2>/dev/null
            # After opening browser, ask again
            local action2=$(gum choose \
                --header "Models: $display_dir | Continue with installation?" \
                "⬇️  Install this LLM" \
                "❌ Cancel")
            [[ "$action2" == "⬇️  Install this LLM" ]] && return 0 || return 1
            ;;
        *)
            return 1
            ;;
    esac
}

# Function to download a model
download_model() {
    local repo_id="$1"
    local model_name="$2"
    local cache_dir="$MODEL_DIR/cache"
    local display_dir
    display_dir=$(get_model_dir_display)
    
    # Validate repo_id before passing to hf
    if [[ -z "$repo_id" || "$repo_id" == *"Fetching"* || "$repo_id" == *"..."* ]]; then
        echo "Error: Invalid repository ID: '$repo_id'"
        return 1
    fi
    
    echo "Installing $repo_id into $display_dir..."
    
    # Ensure cache directory exists
    mkdir -p "$cache_dir"
    
    # Download with hf into cache directory
    if hf download "$repo_id" \
        --cache-dir "$cache_dir" \
        --no-quiet; then
        
        echo "Download complete!"
        
        # Convert repo_id to cache directory format (e.g., "mlx-community/model-name" -> "models--mlx-community--model-name")
        local cache_model_name="models--${repo_id//\//--}"
        local model_cache_dir="$cache_dir/$cache_model_name"
        
        # Find the snapshot directory for THIS specific model
        if [[ -d "$model_cache_dir/snapshots" ]]; then
            local snapshot_path=$(find "$model_cache_dir/snapshots" -mindepth 1 -maxdepth 1 -type d | head -n 1)
            if [[ -n "$snapshot_path" ]]; then
                # Create symlink at MODEL_DIR root pointing to this model's snapshot
                ln -sf "$snapshot_path" "$MODEL_DIR/$model_name"
                echo "LLM ready: $model_name"
                return 0
            else
                echo "Error: No snapshot found in $model_cache_dir/snapshots"
                return 1
            fi
        else
            echo "Error: Snapshots directory not found at $model_cache_dir/snapshots"
            return 1
        fi
    else
        echo "Installation failed!"
        return 1
    fi
}

# Function to remove a model
remove_model() {
    local model_name="$1"
    local model_path="$MODEL_DIR/$model_name"
    local target=$(readlink "$model_path")
    local display_dir
    display_dir=$(get_model_dir_display)
    
    if gum confirm "Models: $display_dir | Are you sure you want to uninstall $model_name?"; then
        # Remove symlink
        rm "$model_path"
        
        # Find and remove the cache directory
        # The target path is like: .../cache/models--org--name/snapshots/hash
        # We want to remove the models--org--name directory
        if [[ -n "$target" ]]; then
            # Go up from snapshots/hash to get the model cache folder
            local cache_model_dir=$(dirname "$(dirname "$target")")
            if [[ -d "$cache_model_dir" && "$cache_model_dir" == *"/cache/"* ]]; then
                rm -rf "$cache_model_dir"
                echo "Cache cleaned for $model_name"
            fi
        fi
        
        echo "LLM $model_name uninstalled!"
    fi
}

# Function to get available models from Hugging Face
get_available_models() {
    local source="$1"
    local page="${2:-1}"
    local search_term="${3:-}"
    local per_page=100
    local offset=$((( page - 1 ) * per_page))
    
    local api_url=""
    
    case "$source" in
        "mlx-community")
            # Get models from mlx-community org
            api_url="https://huggingface.co/api/models?author=mlx-community&sort=downloads&direction=-1&limit=${per_page}&skip=${offset}"
            ;;
        "lmstudio-community")
            # Get models from lmstudio-community org
            api_url="https://huggingface.co/api/models?author=lmstudio-community&sort=downloads&direction=-1&limit=${per_page}&skip=${offset}"
            ;;
        "All Models")
            # Search all HuggingFace models
            api_url="https://huggingface.co/api/models?sort=downloads&direction=-1&limit=${per_page}&skip=${offset}"
            ;;
    esac
    
    # Add additional search filter if provided
    if [[ -n "$search_term" ]]; then
        local encoded_search=$(echo "$search_term" | jq -sRr @uri)
        api_url="${api_url}&search=${encoded_search}"
    fi
    
    # Debug: show URL (comment out in production)
    # echo "DEBUG: $api_url" >&2
    
    # Fetch results
    curl -s "$api_url" | jq -r '.[].id' 2>/dev/null
}

# Main menu
while true; do
    clear
    show_header
    
    choice=$(gum choose \
        --height 15 \
        --header "Models: $(get_model_dir_display)" \
        "⚡ Run LM on Template" \
        "Run an Installed LLM" \
        "Install a New Hugging Face LLM" \
        "Uninstall an LLM" \
        "Configure Model Storage Path" \
        "Exit")
    
    # Handle ESC (empty selection)
    if [[ -z "$choice" ]]; then
        exit 0
    fi
    
    case $choice in
        "⚡ Run LM on Template")
            # Fast launch predefined template models
            template_choice=$(gum choose \
                --header "Select a template model to run" \
                "Qwen3-Coder-30B-A3B-Instruct-8bit" \
                "NVIDIA-Nemotron-3-Nano-30B-A3B-MLX-8Bit" \
                "GLM-4.7-Flash-8bit" \
                "✖ Back")

            case "$template_choice" in
                "Qwen3-Coder-30B-A3B-Instruct-8bit")
                    CMD_ARGS=("--model-path" "$MODEL_DIR/Qwen3-Coder-30B-A3B-Instruct-8bit")
                    CMD_ARGS+=("--model-type" "lm")
                    CMD_ARGS+=("--tool-call-parser" "qwen3_coder")
                    CMD_ARGS+=("--message-converter" "qwen3_coder")
                    CMD_ARGS+=("--port" "8000")
                    CMD_ARGS+=("--host" "0.0.0.0")
                    confirm_and_launch
                    ;;
                "NVIDIA-Nemotron-3-Nano-30B-A3B-MLX-8Bit")
                    CMD_ARGS=("--model-path" "$MODEL_DIR/NVIDIA-Nemotron-3-Nano-30B-A3B-MLX-8Bit")
                    CMD_ARGS+=("--model-type" "lm")
                    CMD_ARGS+=("--tool-call-parser" "qwen3")
                    CMD_ARGS+=("--message-converter" "nemotron3_nano")
                    CMD_ARGS+=("--port" "8000")
                    CMD_ARGS+=("--host" "0.0.0.0")
                    CMD_ARGS+=("--trust-remote-code")
                    confirm_and_launch
                    ;;
                "GLM-4.7-Flash-8bit")
                    CMD_ARGS=("--model-path" "$MODEL_DIR/GLM-4.7-Flash-8bit")
                    CMD_ARGS+=("--model-type" "lm")
                    CMD_ARGS+=("--reasoning-parser" "glm4_moe") # glm47_flash
                    CMD_ARGS+=("--tool-call-parser" "glm4_moe")
                    CMD_ARGS+=("--message-converter" "glm4_moe")
                    CMD_ARGS+=("--debug")
                    CMD_ARGS+=("--port" "8000")
                    CMD_ARGS+=("--host" "0.0.0.0")
                    confirm_and_launch
                    ;;
            esac
            ;;

        "Run an Installed LLM")
            # Get list of installed models
            installed_models=()
            while IFS= read -r model; do
                [[ -n "$model" ]] && installed_models+=("$(basename "$model")")
            done < <(find "$MODEL_DIR" -maxdepth 1 -type l 2>/dev/null)
            
            if [[ ${#installed_models[@]} -eq 0 ]]; then
                echo "No LLMs installed yet."
                gum confirm "Press Enter to continue..."
            else
                # Model selection loop
                while true; do
                    model_to_run=$(gum choose \
                        --header "Models: $(get_model_dir_display) | Select an LLM to run" \
                        "${installed_models[@]}" \
                        "✖ Back")
                    
                    if [[ -z "$model_to_run" || "$model_to_run" == "✖ Back" ]]; then
                        break
                    fi
                    
                    # Run mode selection loop - show preset configurations
                    while true; do
                        preset=$(gum choose \
                            --header "Models: $(get_model_dir_display) | $model_to_run - Select configuration" \
                            "lm (text-only)" \
                            "multimodal (vision, audio)" \
                            "image-generation (qwen-image, q16)" \
                            "image-edit (qwen-image-edit, q16)" \
                            "embeddings" \
                            "whisper (audio transcription)" \
                            "✖ Back")
                        
                        # Extract preset type
                        case "$preset" in
                            "lm (text-only)") preset_type="lm" ;;
                            "multimodal (vision, audio)") preset_type="multimodal" ;;
                            "image-generation"*) preset_type="image-generation" ;;
                            "image-edit"*) preset_type="image-edit" ;;
                            "embeddings") preset_type="embeddings" ;;
                            "whisper"*) preset_type="whisper" ;;
                            *) break ;;  # Back or cancel
                        esac
                        
                        # Handle preset action
                        if handle_preset_action "$model_to_run" "$preset_type"; then
                            break 2  # Exit both loops after successful run
                        fi
                        # If cancelled, stay in preset selection
                    done
                done
            fi
            ;;
            
        "Install a New Hugging Face LLM")
            # Get model source
            model_source=$(gum choose \
                --header "Models: $(get_model_dir_display) | Select Model Source" \
                "mlx-community" \
                "lmstudio-community" \
                "All Models")
            
            if [[ -z "$model_source" ]]; then
                continue
            fi
            
            # Browse mode loop - allows returning to this menu from model list
            while true; do
                # Choose browse mode: search or show all
                browse_mode=$(gum choose \
                    --header "Models: $(get_model_dir_display) | $model_source" \
                    "🔍 Search a model..." \
                    "📋 Show all models..." \
                    "✖ Back")
                
                if [[ -z "$browse_mode" || "$browse_mode" == "✖ Back" ]]; then
                    break
                fi
                
                # Pagination for model selection
                page=1
                search_term=""
                
                # Handle search option
                if [[ "$browse_mode" == "🔍 Search a model..." ]]; then
                    search_term=$(gum input \
                        --placeholder "Enter model name (e.g., llama, qwen, mistral)..." \
                        --header "Search in $model_source" \
                        --width 60)
                    
                    # If user cancelled or empty input, go back to browse mode menu
                    if [[ -z "$search_term" ]]; then
                        continue
                    fi
                fi
                
                while true; do
                    # Calculate offset for display
                    offset=$(( (page - 1) * 100 ))
                    
                    if [[ -n "$search_term" ]]; then
                        echo "Searching '$search_term' in $model_source (page $page)..."
                    else
                        echo "Fetching $model_source models (page $page)..."
                    fi
                    
                    # Get available models from API
                    available_models=$(get_available_models "$model_source" "$page" "$search_term")
                    
                    if [[ -z "$available_models" ]]; then
                        if [[ $page -eq 1 ]]; then
                            echo "No models found matching your criteria."
                            gum confirm "Press Enter to continue..."
                            break
                        else
                            echo "No more models."
                            ((page--))
                            continue
                        fi
                    fi
                    
                    # Count models
                    model_count=$(echo "$available_models" | wc -l | tr -d ' ')
                    
                    # Build menu: models first, then navigation at bottom
                    menu_items="${available_models}\n---"
                    
                    # Add search option
                    if [[ -n "$search_term" ]]; then
                        menu_items="${menu_items}\n🔍 Search: \"$search_term\" [change]"
                    else
                        menu_items="${menu_items}\n🔍 Search API..."
                    fi
                    
                    # Add pagination
                    if [[ $page -gt 1 ]]; then
                        menu_items="${menu_items}\n◀ Previous Page"
                    fi
                    if [[ $model_count -ge 100 ]]; then
                        menu_items="${menu_items}\n▶ Next Page"
                    fi
                    menu_items="${menu_items}\n✖ Back"
                    
                    # Create header with info
                    display_dir=$(get_model_dir_display)
                    header_text="Models: $display_dir | Page $page | $model_source | $model_count models"
                    if [[ -n "$search_term" ]]; then
                        header_text="$header_text | Search: $search_term"
                    fi
                    
                    selection=$(echo -e "$menu_items" | gum choose --height 30 --header "$header_text")
                    
                    case "$selection" in
                        "🔍 Search"*)
                            search_term=$(gum input \
                                --placeholder "Search models (e.g., llama, qwen, mistral)..." \
                                --header "Search HuggingFace API" \
                                --value "$search_term" \
                                --width 60)
                            page=1  # Reset to page 1 with new search
                            ;;
                        "◀ Previous Page")
                            ((page--))
                            ;;
                        "▶ Next Page")
                            ((page++))
                            ;;
                        "✖ Back"|""|"---")
                            break
                            ;;
                        *)
                            # User selected a model - show details first
                            repo_id="$selection"
                            model_name=$(basename "$repo_id")
                            
                            if show_model_details "$repo_id"; then
                                if download_model "$repo_id" "$model_name"; then
                                    gum confirm "LLM installed successfully! Press Enter to continue..."
                                else
                                    gum confirm "Installation failed. Press Enter to continue..."
                                fi
                                break
                            fi
                            # If cancelled, stay in the model list
                            ;;
                    esac
                done
            done
            ;;
            
        "Uninstall an LLM")
            models_to_remove=()
            while IFS= read -r model; do
                [[ -n "$model" ]] && models_to_remove+=("$(basename "$model")")
            done < <(find "$MODEL_DIR" -maxdepth 1 -type l 2>/dev/null)
            
            if [[ ${#models_to_remove[@]} -eq 0 ]]; then
                echo "No LLMs installed."
                gum confirm "Press Enter to continue..."
            else
                model_to_remove=$(gum choose \
                    --header "Models: $(get_model_dir_display) | Select an LLM to uninstall" \
                    "${models_to_remove[@]}")
                if [[ -n "$model_to_remove" ]]; then
                    remove_model "$model_to_remove"
                fi
            fi
            ;;
            
        "Configure Model Storage Path")
            configure_model_path
            ;;
            
        "Exit")
            exit 0
            ;;
    esac
done