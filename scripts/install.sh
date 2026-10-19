#!/usr/bin/env sh
# Install the hydro CLI on Linux or macOS.
#   curl -fsSL https://github.com/ozyildirims/hydrogo/releases/latest/download/install.sh | sh
# Environment: HYDRO_VERSION (default: latest, e.g. v0.2.0), HYDRO_INSTALL_DIR
# (default: /usr/local/bin when writable, else ~/.local/bin).
set -eu

repo="ozyildirims/hydrogo"
version="${HYDRO_VERSION:-latest}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in linux|darwin) ;; *) echo "unsupported OS: $os (use the Windows installer on Windows)" >&2; exit 1;; esac
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

if [ "$version" = latest ]; then
  version="$(curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)"
  [ -n "$version" ] || { echo "could not determine the latest release" >&2; exit 1; }
fi

name="hydrogo_${version}_${os}_${arch}"
base="https://github.com/${repo}/releases/download/${version}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Downloading HydroGo ${version} (${os}/${arch})..."
curl -fsSL "${base}/${name}.tar.gz" -o "${tmp}/${name}.tar.gz"
curl -fsSL "${base}/SHA256SUMS" -o "${tmp}/SHA256SUMS"
expected="$(grep " ${name}.tar.gz\$" "${tmp}/SHA256SUMS" | cut -d' ' -f1)"
if command -v sha256sum >/dev/null 2>&1; then actual="$(sha256sum "${tmp}/${name}.tar.gz" | cut -d' ' -f1)"; else actual="$(shasum -a 256 "${tmp}/${name}.tar.gz" | cut -d' ' -f1)"; fi
[ -n "$expected" ] && [ "$expected" = "$actual" ] || { echo "checksum mismatch, aborting" >&2; exit 1; }

tar -xzf "${tmp}/${name}.tar.gz" -C "$tmp"

dir="${HYDRO_INSTALL_DIR:-}"
if [ -z "$dir" ]; then
  if [ -w /usr/local/bin ]; then dir=/usr/local/bin; else dir="$HOME/.local/bin"; fi
fi
mkdir -p "$dir"
install -m 0755 "${tmp}/${name}/hydro" "${dir}/hydro"
echo "Installed ${dir}/hydro"
case ":$PATH:" in *":${dir}:"*) ;; *) echo "Add ${dir} to your PATH to run \`hydro\` from anywhere." ;; esac
echo "Next: hydro new mysite --preset blog --admin-email you@example.com"
