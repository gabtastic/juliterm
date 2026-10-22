#!/bin/sh
set -eu

REPO="cihann/juliterm"
BINARY="juli"
INSTALL_DIR="${JULI_INSTALL_DIR:-/usr/local/bin}"

main() {
    need_cmd curl
    need_cmd tar

    case "$(uname -s)" in
        Linux)  os="linux" ;;
        Darwin) os="darwin" ;;
        *)      err "unsupported OS: $(uname -s)" ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64" ;;
        aarch64|arm64)  arch="arm64" ;;
        *)              err "unsupported architecture: $(uname -m)" ;;
    esac

    tag="${1:-$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' | cut -d'"' -f4)}"
    [ -n "${tag}" ] || err "failed to determine latest release tag"

    url="https://github.com/${REPO}/releases/download/${tag}/${BINARY}-${tag}-${os}-${arch}.tar.gz"
    tmp="$(mktemp -d)"
    trap 'rm -rf "${tmp}"' EXIT

    echo "downloading ${BINARY} ${tag} for ${os}/${arch}..."
    curl -fsSL "${url}" | tar xz -C "${tmp}"

    if [ -w "${INSTALL_DIR}" ]; then
        mv "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    else
        echo "installing to ${INSTALL_DIR} (requires sudo)..."
        sudo mv "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    fi

    chmod +x "${INSTALL_DIR}/${BINARY}"
    echo "${BINARY} ${tag} installed to ${INSTALL_DIR}/${BINARY}"
}

need_cmd() {
    command -v "$1" > /dev/null 2>&1 || err "need '$1' (not found)"
}

err() {
    echo "error: $1" >&2
    exit 1
}

main "$@"
