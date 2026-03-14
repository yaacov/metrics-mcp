# Installation

## Quick Install (Linux / macOS)

Download the latest release, verify its checksum, install the binary and shell completion helpers:

```bash
curl -sSL https://raw.githubusercontent.com/yaacov/kubectl-metrics/main/install.sh | bash
```

By default the script installs to `~/.local/bin`. Override with environment variables:

```bash
# Install a specific version
curl -sSL https://raw.githubusercontent.com/yaacov/kubectl-metrics/main/install.sh | VERSION=v0.1.0 bash

# Install to a different directory
curl -sSL https://raw.githubusercontent.com/yaacov/kubectl-metrics/main/install.sh | INSTALL_DIR=/usr/local/bin bash
```

The script installs three files:

| File | Purpose |
|------|---------|
| `kubectl-metrics` | Main binary (kubectl plugin) |
| `kubectl_complete-metrics` | Shell completion helper for `kubectl metrics` |
| `oc_complete-metrics` | Shell completion helper for `oc metrics` |

If the install directory is not in your `PATH`, the script prints instructions for adding it.

## Build from Source

Requires Go 1.21+.

```bash
git clone https://github.com/yaacov/kubectl-metrics.git
cd kubectl-metrics
make build

# Copy the binary to a directory in your PATH
sudo cp kubectl-metrics /usr/local/bin/kubectl-metrics
```

## Manual Download

Download a release archive from the [GitHub Releases](https://github.com/yaacov/kubectl-metrics/releases) page. Archives are available for:

| OS | Architecture | Archive |
|----|-------------|---------|
| Linux | amd64 | `kubectl-metrics-VERSION-linux-amd64.tar.gz` |
| Linux | arm64 | `kubectl-metrics-VERSION-linux-arm64.tar.gz` |
| macOS | amd64 | `kubectl-metrics-VERSION-darwin-amd64.tar.gz` |
| macOS | arm64 | `kubectl-metrics-VERSION-darwin-arm64.tar.gz` |
| Windows | amd64 | `kubectl-metrics-VERSION-windows-amd64.zip` |

Extract and install:

```bash
VERSION=v0.1.0   # replace with desired version
OS=darwin         # linux, darwin, or windows
ARCH=arm64        # amd64 or arm64

tar -xzf kubectl-metrics-${VERSION}-${OS}-${ARCH}.tar.gz
install -m 0755 kubectl-metrics-${OS}-${ARCH} ~/.local/bin/kubectl-metrics
```

## Shell Completion

Tab completion works for both `kubectl metrics` and `oc metrics`. The [install script](#quick-install-linux--macos) sets this up automatically. If you installed via another method, create the helpers manually:

```bash
# Find the directory where kubectl-metrics is installed
d="$(dirname "$(which kubectl-metrics)")"

# Create the kubectl completion helper
cat > "$d/kubectl_complete-metrics" << 'SCRIPT'
#!/usr/bin/env bash
kubectl-metrics __complete "$@"
SCRIPT
chmod +x "$d/kubectl_complete-metrics"

# Create the oc completion helper (symlink to the kubectl one)
ln -sf "$d/kubectl_complete-metrics" "$d/oc_complete-metrics"
```

Requires kubectl 1.26+ or oc 4.x with shell completions loaded.

## Uninstall

Remove the three installed files:

```bash
rm -f ~/.local/bin/kubectl-metrics
rm -f ~/.local/bin/kubectl_complete-metrics
rm -f ~/.local/bin/oc_complete-metrics
```

If you installed to a different directory, replace `~/.local/bin` with that path.
