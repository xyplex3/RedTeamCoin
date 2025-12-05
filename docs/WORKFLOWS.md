# GitHub Actions Workflows

This directory contains CI/CD workflows for RedTeamCoin.

## Release Process

## Quick Start

Create a release by pushing a version tag:

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

GitHub Actions automatically builds and publishes:

- CPU binaries for Linux, Windows, macOS
- GPU binaries (OpenCL) for Linux and macOS
- Archives with checksums

## Workflows

### `test.yml` - Test & Build Verification

Runs on every push and PR. Tests:

- Code compilation on all platforms
- Cross-compilation to Linux, Windows, macOS
- GPU build compilation (OpenCL)
- CGO directive validation

### `release.yml` - Release Builder

Triggered by version tags (e.g., `v1.0.0`). Builds:

- Native platform binaries with GPU support
- Aggregates artifacts into unified release
- Generates checksums

### `ci.yml` - Code Quality

Runs linting, formatting, and security scans.

### `gpu-cuda-build.yml` - CUDA Builds (Optional)

Manual workflow for NVIDIA CUDA builds.
Requires self-hosted runner with GPU.

## Architecture Overview

```text
┌─────────────────────────────────────────────────────────┐
│                    GitHub Actions                        │
├─────────────────────────────────────────────────────────┤
│                                                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │   Ubuntu     │  │   Windows    │  │    macOS     │  │
│  │   Runner     │  │    Runner    │  │    Runner    │  │
│  ├──────────────┤  ├──────────────┤  ├──────────────┤  │
│  │ CPU + OpenCL │  │  CPU builds  │  │ CPU + OpenCL │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
│         │                  │                  │          │
│         └──────────────────┴──────────────────┘          │
│                            │                              │
│                            ▼                              │
│                  ┌──────────────────┐                    │
│                  │  Aggregate       │                    │
│                  │  Artifacts       │                    │
│                  └──────────────────┘                    │
│                            │                              │
│                            ▼                              │
│                  ┌──────────────────┐                    │
│                  │  GitHub Release  │                    │
│                  │  - CPU binaries  │                    │
│                  │  - GPU binaries  │                    │
│                  │  - Archives      │                    │
│                  │  - Checksums     │                    │
│                  └──────────────────┘                    │
│                                                           │
│  Optional: Self-Hosted GPU Runner with CUDA              │
│  ┌──────────────────────────────────────┐               │
│  │  GPU Runner (NVIDIA CUDA)            │               │
│  │  - Builds CUDA binaries              │               │
│  │  - Uploads to release                │               │
│  └──────────────────────────────────────┘               │
└─────────────────────────────────────────────────────────┘
```

## What Gets Built

Each release includes:

| Archive | Platform | Binaries Included |
|---------|----------|-------------------|
| `redteamcoin-linux-amd64.tar.gz` | Linux x64 | server, client, client-opencl |
| `redteamcoin-windows-amd64.zip` | Windows x64 | server.exe, client.exe |
| `redteamcoin-macos-amd64.tar.gz` | macOS x64 | server, client, client-opencl |

Plus `checksums.txt` with SHA256 hashes.

## CUDA Support (Optional)

For NVIDIA CUDA builds, set up a self-hosted runner with GPU:

1. Add self-hosted runner with labels: `linux`, `gpu-cuda`
2. Install CUDA Toolkit on the runner
3. Manually trigger **GPU Build (CUDA)** workflow from Actions tab

## Manual Trigger

Go to **Actions** → **Release** → **Run workflow** → Enter version

## All Workflows Summary

- `release.yml` - Creates releases (triggered by tags)
- `ci.yml` - Tests on push/PR
- `gpu-cuda-build.yml` - CUDA builds (manual, requires self-hosted runner)
