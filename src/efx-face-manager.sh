#!/bin/bash

# ╔═══════════════════════════════════════════════════════════════╗
# ║                      efx-face-manager                         ║
# ║              MLX Hugging Face Model Manager                   ║
# ╚═══════════════════════════════════════════════════════════════╝
# Uses gum for interactive selection
# https://github.com/charmbracelet/gum

VERSION="0.0.2"

clear

# Show version
show_version() {
    echo "efx-face-manager v$VERSION"
    exit 0
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
        'MLX Hugging Face Manager'
}

# Handle version flag
if [[ "$1" == "--version" || "$1" == "-v" ]]; then
    show_version
fi

# Configuration - Set MODEL_DIR in your .zshrc or here
export MODEL_DIR="${MODEL_DIR:-/Volumes/T7/Téléchargements/ollama/mlx-server}"

# Ensure directory exists
mkdir -p "$MODEL_DIR"

# Function to run an installed model
run_model() {
    local model_name="$1"
    local model_path="$MODEL_DIR/$model_name"
    
    if [[ -L "$model_path" ]]; then
        echo "Starting MLX OpenAI Server with $model_name..."
        mlx-openai-server launch --model-path "$model_path" --model-type lm
    else
        echo "Error: Model not found or not a symlink: $model_path"
    fi
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

Downloads:    $downloads
Likes:        $likes
Pipeline:     $pipeline
Library:      $library
Updated:      $last_modified
Size:         $size_display"
    
    echo ""
    
    # Show action menu
    local action=$(gum choose \
        --header "What would you like to do?" \
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
                --header "Continue with installation?" \
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
    
    # Validate repo_id before passing to hf
    if [[ -z "$repo_id" || "$repo_id" == *"Fetching"* || "$repo_id" == *"..."* ]]; then
        echo "Error: Invalid repository ID: '$repo_id'"
        return 1
    fi
    
    echo "Installing $repo_id..."
    
    # Ensure cache directory exists
    mkdir -p "$cache_dir"
    
    # Download with hf into cache directory
    if hf download "$repo_id" \
        --cache-dir "$cache_dir" \
        --no-quiet; then
        
        echo "Download complete!"
        
        # Find the snapshot directory in cache
        local snapshot_dir=$(find "$cache_dir" -name "snapshots" -type d | head -n 1)
        if [[ -n "$snapshot_dir" ]]; then
            local snapshot_path=$(find "$snapshot_dir" -mindepth 1 -maxdepth 1 -type d | head -n 1)
            if [[ -n "$snapshot_path" ]]; then
                # Create symlink at MODEL_DIR root pointing into cache
                ln -sf "$snapshot_path" "$MODEL_DIR/$model_name"
                echo "LLM ready: $model_name"
            else
                echo "Error: No snapshot directory found"
                return 1
            fi
        else
            echo "Error: No snapshots directory found"
            return 1
        fi
        
        return 0
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
    
    if gum confirm "Are you sure you want to uninstall $model_name?"; then
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
        --header "Models: $MODEL_DIR" \
        "Run an Installed LLM" \
        "Install a New Hugging Face LLM" \
        "Uninstall an LLM" \
        "Exit")
    
    # Handle ESC (empty selection)
    if [[ -z "$choice" ]]; then
        exit 0
    fi
    
    case $choice in
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
                model_to_run=$(gum choose --header "Select an LLM to run" "${installed_models[@]}")
                if [[ -n "$model_to_run" ]]; then
                    run_model "$model_to_run"
                fi
            fi
            ;;
            
        "Install a New Hugging Face LLM")
            # Get model source
            model_source=$(gum choose \
                --header "Select Model Source" \
                "mlx-community" \
                "lmstudio-community" \
                "All Models")
            
            if [[ -z "$model_source" ]]; then
                continue
            fi
            
            # Pagination for model selection
            page=1
            search_term=""
            
            while true; do
                # Calculate offset for display
                offset=$(( (page - 1) * 100 ))
                echo "Fetching models (page $page, offset $offset)..."
                
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
                header_text="Page $page | $model_source | $model_count models"
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
                model_to_remove=$(gum choose --header "Select an LLM to uninstall" "${models_to_remove[@]}")
                if [[ -n "$model_to_remove" ]]; then
                    remove_model "$model_to_remove"
                fi
            fi
            ;;
            
        "Exit")
            exit 0
            ;;
    esac
done