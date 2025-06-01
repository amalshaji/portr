#!/bin/bash

# Portr Installation Script
# Usage: curl -sSL https://raw.githubusercontent.com/amalshaji/portr/main/install.sh | sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# GitHub repository information
REPO_OWNER="amalshaji"
REPO_NAME="portr"
GITHUB_REPO="https://github.com/${REPO_OWNER}/${REPO_NAME}"
GITHUB_API="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}"

# Installation directory
INSTALL_DIR="$HOME/.portr/bin"
BINARY_NAME="portr"

# Function to print colored output
print_info() {
    printf "${GREEN}[INFO]${NC} %s\n" "$1"
}

print_warning() {
    printf "${YELLOW}[WARNING]${NC} %s\n" "$1"
}

print_error() {
    printf "${RED}[ERROR]${NC} %s\n" "$1"
}

# Function to detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)     echo "Linux";;
        Darwin*)    echo "Darwin";;
        CYGWIN*|MINGW*|MSYS*) echo "Windows";;
        *)
            print_error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac
}

# Function to detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "x86_64";;
        arm64|aarch64) echo "arm64";;
        *)
            print_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
}

# Function to get the latest release version
get_latest_version() {
    if command -v curl >/dev/null 2>&1; then
        curl -s "${GITHUB_API}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "${GITHUB_API}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    else
        print_error "Neither curl nor wget is available. Please install one of them."
        exit 1
    fi
}

# Function to download file
download_file() {
    local url="$1"
    local output="$2"

    if command -v curl >/dev/null 2>&1; then
        curl -sSL "$url" -o "$output"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$url" -O "$output"
    else
        print_error "Neither curl nor wget is available. Please install one of them."
        exit 1
    fi
}

# Function to verify checksum (optional)
verify_checksum() {
    local file="$1"
    local expected_checksum="$2"

    if command -v sha256sum >/dev/null 2>&1; then
        local actual_checksum=$(sha256sum "$file" | cut -d' ' -f1)
    elif command -v shasum >/dev/null 2>&1; then
        local actual_checksum=$(shasum -a 256 "$file" | cut -d' ' -f1)
    else
        print_warning "No checksum utility found. Skipping verification."
        return 0
    fi

    if [ "$actual_checksum" = "$expected_checksum" ]; then
        print_info "Checksum verification passed"
        return 0
    else
        print_error "Checksum verification failed"
        return 1
    fi
}

# Function to detect shell profile file
detect_shell_profile() {
    local shell_name=$(basename "$SHELL")
    case "$shell_name" in
        bash)
            if [[ -f "$HOME/.bashrc" ]]; then
                echo "$HOME/.bashrc"
            elif [[ -f "$HOME/.bash_profile" ]]; then
                echo "$HOME/.bash_profile"
            else
                echo "$HOME/.profile"
            fi
            ;;
        zsh)
            echo "$HOME/.zshrc"
            ;;
        fish)
            echo "fish"
            ;;
        *)
            echo "$HOME/.profile"
            ;;
    esac
}

# Function to add directory to PATH
add_to_path() {
    local profile_file=$(detect_shell_profile)
    local path_line="export PATH=\"$INSTALL_DIR:\$PATH\""

    if [[ "$profile_file" == "fish" ]]; then
        # Fish shell uses fish_add_path
        if command -v fish >/dev/null 2>&1; then
            # Check if path is already added to fish
            if fish -c "contains $INSTALL_DIR \$fish_user_paths" 2>/dev/null; then
                print_info "$INSTALL_DIR already in fish PATH configuration"
                return 0
            fi

            fish -c "fish_add_path $INSTALL_DIR" 2>/dev/null || {
                print_warning "Failed to add to fish PATH automatically"
                print_info "Add manually: fish_add_path $INSTALL_DIR"
                return 1
            }
            print_info "Added $INSTALL_DIR to fish PATH"
        else
            print_warning "Fish shell not found in PATH"
            return 1
        fi
    else
        # Check if the exact PATH export line already exists in profile
        if grep -Fq "$path_line" "$profile_file" 2>/dev/null; then
            print_info "$INSTALL_DIR already in PATH configuration"
            return 0
        fi

        # Also check for any existing PATH modification with this directory
        if grep -q "PATH.*$INSTALL_DIR" "$profile_file" 2>/dev/null; then
            print_info "$INSTALL_DIR already referenced in PATH configuration"
            return 0
        fi

        echo "" >> "$profile_file"
        echo "# Added by Portr installer" >> "$profile_file"
        echo "$path_line" >> "$profile_file"
        print_info "Added $INSTALL_DIR to PATH in $profile_file"
    fi
    return 0
}

# Main installation function
install_portr() {
    print_info "Starting Portr installation..."

    # Detect system information
    OS=$(detect_os)
    ARCH=$(detect_arch)
    print_info "Detected OS: $OS, Architecture: $ARCH"

    # Get latest version
    print_info "Fetching latest release information..."
    VERSION=$(get_latest_version)
    if [ -z "$VERSION" ]; then
        print_error "Failed to get latest version information"
        exit 1
    fi
    print_info "Latest version: $VERSION"

    # Construct download URL based on goreleaser template
    # Template: portr_{{.Version}}_{{title .Os}}_{{if eq .Arch "amd64"}}x86_64{{else}}{{.Arch}}{{end}}.zip
    ARCHIVE_NAME="${BINARY_NAME}_${VERSION}_${OS}_${ARCH}.zip"
    DOWNLOAD_URL="${GITHUB_REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"

    print_info "Download URL: $DOWNLOAD_URL"

    # Create temporary directory
    TEMP_DIR=$(mktemp -d)
    trap "rm -rf $TEMP_DIR" EXIT

    # Download the archive
    print_info "Downloading $ARCHIVE_NAME..."
    ARCHIVE_PATH="$TEMP_DIR/$ARCHIVE_NAME"

    if ! download_file "$DOWNLOAD_URL" "$ARCHIVE_PATH"; then
        print_error "Failed to download $ARCHIVE_NAME"
        exit 1
    fi

    # Extract the archive
    print_info "Extracting archive..."
    if command -v unzip >/dev/null 2>&1; then
        unzip -q "$ARCHIVE_PATH" -d "$TEMP_DIR"
    else
        print_error "unzip command not found. Please install unzip."
        exit 1
    fi

    # Find the binary
    BINARY_PATH="$TEMP_DIR/$BINARY_NAME"
    if [ "$OS" = "Windows" ]; then
        BINARY_PATH="$TEMP_DIR/${BINARY_NAME}.exe"
    fi

    if [ ! -f "$BINARY_PATH" ]; then
        print_error "Binary not found in archive"
        exit 1
    fi

    # Create installation directory
    print_info "Creating installation directory: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"

    # Install the binary
    print_info "Installing binary to $INSTALL_DIR"
    cp "$BINARY_PATH" "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"

    # Handle PATH configuration
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        echo
        # Check for environment variable to control behavior
        AUTO_ADD_PATH=${PORTR_AUTO_ADD_PATH:-"yes"}

        case "$AUTO_ADD_PATH" in
            "no"|"false"|"0")
                print_warning "$INSTALL_DIR is not in your PATH."
                print_info "Add manually: export PATH=\"$INSTALL_DIR:\$PATH\""
                ;;
            *)
                # Default: automatically add to PATH
                if add_to_path; then
                    print_info "PATH updated! Restart your terminal or run: source $(detect_shell_profile)"
                else
                    print_warning "Failed to add to PATH automatically"
                    print_info "Add manually: export PATH=\"$INSTALL_DIR:\$PATH\""
                fi
                ;;
        esac
    else
        print_info "$INSTALL_DIR is already in your PATH"
    fi

    echo
    print_info "Portr $VERSION installed successfully!"
    print_info "Please restart your terminal or run: source $(detect_shell_profile)"
    print_info "Then run 'portr --help' to get started"
}

# Check if running as root (not recommended)
if [ "$EUID" -eq 0 ]; then
    print_warning "Running as root is not recommended. Consider running as a regular user."
    INSTALL_DIR="/usr/local/bin"
fi

# Run installation
install_portr