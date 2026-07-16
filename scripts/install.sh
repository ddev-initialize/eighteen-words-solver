#!/bin/sh
set -eu

repo="${EIGHTEEN_WORDS_REPO:-ddevpost/eighteen-words-solver}"
install_dir="${INSTALL_DIR:-$HOME/.local/bin}"

case "$(uname -s)" in
    Darwin) os="darwin" ;;
    Linux) os="linux" ;;
    *) echo "Unsupported operating system" >&2; exit 1 ;;
esac

case "$(uname -m)" in
    x86_64|amd64) arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *) echo "Unsupported architecture" >&2; exit 1 ;;
esac

asset="eighteen-words-solver-${os}-${arch}.tar.gz"
url="https://github.com/${repo}/releases/latest/download/${asset}"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

curl -fsSL "$url" -o "$tmp_dir/$asset"
tar -xzf "$tmp_dir/$asset" -C "$tmp_dir"
mkdir -p "$install_dir"
install -m 0755 "$tmp_dir/eighteen-words-solver" "$install_dir/eighteen-words-solver"

echo "Installed eighteen-words-solver to $install_dir/eighteen-words-solver"
case ":$PATH:" in
    *":$install_dir:"*) ;;
    *) echo "Add $install_dir to PATH to run it from any directory." ;;
esac
