# efx-face-manager Demo
# VHS tape file for generating animated GIF demo
#
# To run:
#     vhs efx-face.tape
#
# Install VHS: brew install vhs

# Output settings
Output efx-face.gif
Output efx-face.mp4

# Terminal configuration
Set FontSize 18
Set Width 1200
Set Height 800
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms
Set Padding 20
Set Framerate 30
Set PlaybackSpeed 1.0

# Start the demo
Sleep 500ms

# Run the efx-face-manager script
Type "cd src && ./efx-face-manager.sh"
Sleep 300ms
Enter
Sleep 2s

# ============================================
# STEP 1: Browse MLX models with pagination
# ============================================

# Main menu - Navigate to "Install a New Hugging Face LLM" (second option)
Down@100ms 3
Sleep 1s
Down@100ms 2
Sleep 1s
Enter
Sleep 1s

# Model source menu - select "mlx-community" (first option)
Down@100ms 2
Sleep 1s
Up@100ms 2
Enter
Sleep 3s

# Page 1: Models list is displayed (100 models per page)
# Browse through some models on page 1
# Gum uses internal pagination with Right arrow (4 pages total)
# Navigate 3 times to reach the last page (one complete pagination cycle)
Down@100ms 30
Sleep 1s
Right
Sleep 1s
Right
Sleep 1s
# Now at the last gum page - can see navigation options
# Show the pagination options briefly

# ============================================
# STEP 2: Search for Llama models
# ============================================
# Navigate to "🔍 Search API..." option (after --- separator)

Down@100ms 11
Sleep 1s
# Select Search
Enter
Sleep 1s

# Type "Llama" to search for Llama models
Type "Llama"
Sleep 300ms
Enter
Sleep 3s

# Search results for "Llama" are displayed
# Browse through results to find Llama-3.2-1B-Instruct-4bit
Down@150ms 3
Sleep 1s
Up@150ms 3
Sleep 1s

# Select the model
Enter
Sleep 2s

# ============================================
# STEP 3: View model details and open in browser
# ============================================

# Model details panel is shown with options:
# ⬇️  Install this LLM
# 🔗 Open in Browser  
# ❌ Cancel

# Navigate to "⬇️ Install this LLM"
Down@150ms 3
Sleep 500ms
Up@150ms 3

# Select "⬇️  Install this LLM"
Enter

# ============================================
# STEP 4: Installing the LLM until end
# ============================================

Sleep 4s

# Installation complete - press Enter to continue
Enter
Sleep 2s

# ============================================
# STEP 5: Run the Installed LLM
# ============================================

# Back at main menu - "Run an Installed LLM" is first option
Enter
Sleep 1s

# Select the installed model (Llama-3.2-1B-Instruct-4bit should be listed)
Enter
Sleep 2s

# MLX OpenAI Server is starting and loading the model
# Wait for it to load
Sleep 7s

# Stop the server with Ctrl+C
Ctrl+c
Sleep 2s

# ============================================
# STEP 6: Exit the application
# ============================================

# Back at main menu after Ctrl+C
# Navigate to "Exit" (4th option - need to go down 3 times)
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Enter
Sleep 1s

# ============================================
# STEP 7: Show version and final
# ============================================

# Show version command
Type "efx-face-manager.sh --version"
Sleep 300ms
Enter
Sleep 2s
