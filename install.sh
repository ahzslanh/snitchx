#!/bin/sh
set -e

REPO="ahzslanh/snitchx"
BINARY_NAME="snitchx"

# allow override via environment
INSTALL_DIR="${INSTALL_DIR:-}"
KEEP_QUARANTINE="${KEEP_QUARANTINE:-}"

detect_install_dir() {
    if [ -n "$INSTALL_DIR" ]; then
        echo "$INSTALL_DIR"
        return
    fi

    # prefer user-local directory if it exists and is in PATH
    if [ -d "$HOME/.local/bin" ] && echo "$PATH" | grep -q "$HOME/.local/bin"; then
        echo "$HOME/.local/bin"
        return
    fi

    # fallback to /usr/local/bin
    echo "/usr/local/bin"
}

detect_os() {
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$os" in
        darwin) echo "darwin" ;;
        linux) echo "linux" ;;
        *)
            echo "error: unsupported operating system: $os" >&2
            exit 1
            ;;
    esac
}

detect_arch() {
    arch=$(uname -m)
    case "$arch" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        armv7l) echo "armv7" ;;
        *)
            echo "error: unsupported architecture: $arch" >&2
            exit 1
            ;;
    esac
}

fetch_latest_version() {
    version=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | head -1 | cut -d'"' -f4)
    if [ -z "$version" ]; then
        echo "error: failed to fetch latest version" >&2
        exit 1
    fi
    echo "$version"
}

main() {
    os=$(detect_os)
    arch=$(detect_arch)
    install_dir=$(detect_install_dir)
    version=$(fetch_latest_version)
    version_no_v="${version#v}"

    archive_name="${BINARY_NAME}_${version_no_v}_${os}_${arch}.tar.gz"
    download_url="https://github.com/${REPO}/releases/download/${version}/${archive_name}"

    echo "installing ${BINARY_NAME} ${version} for ${os}/${arch}..."

    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT

    echo "downloading ${download_url}..."
    if ! curl -sL --fail "$download_url" -o "${tmp_dir}/${archive_name}"; then
        echo "error: failed to download ${download_url}" >&2
        exit 1
    fi

    echo "extracting..."
    tar -xzf "${tmp_dir}/${archive_name}" -C "$tmp_dir"

    if [ ! -f "${tmp_dir}/${BINARY_NAME}" ]; then
        echo "error: binary not found in archive" >&2
        exit 1
    fi

    # remove macos quarantine attribute unless disabled
    if [ "$os" = "darwin" ] && [ -z "$KEEP_QUARANTINE" ]; then
        if xattr -d com.apple.quarantine "${tmp_dir}/${BINARY_NAME}" 2>/dev/null; then
            echo "warning: removed macOS quarantine attribute from binary"
        fi
    fi

    # install binary
    if [ -w "$install_dir" ]; then
        mv "${tmp_dir}/${BINARY_NAME}" "${install_dir}/${BINARY_NAME}"
    else
        echo "elevated permissions required to install to ${install_dir}"
        sudo mv "${tmp_dir}/${BINARY_NAME}" "${install_dir}/${BINARY_NAME}"
    fi

    chmod +x "${install_dir}/${BINARY_NAME}"

    echo "installed ${BINARY_NAME} to ${install_dir}/${BINARY_NAME}"
    echo ""
    echo "run '${BINARY_NAME} --help' to get started"
}

main

