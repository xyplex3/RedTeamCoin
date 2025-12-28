# Building RedTeamCoin on Windows

This guide covers building RedTeamCoin client on Windows with GPU (OpenCL) support,
including authentication token configuration for connecting to remote mining pools.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Method 1: Native Windows Build (Recommended)](#method-1-native-windows-build-recommended)
- [Method 2: Cross-Compilation from Linux](#method-2-cross-compilation-from-linux)
- [Connecting to Mining Pool Server](#connecting-to-mining-pool-server)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### For Native Windows Build

**Required Software:**

- Go 1.21 or later for Windows: https://golang.org/dl/
- Git for Windows: https://git-scm.com/download/win
- MinGW-w64 (for CGO): https://www.mingw-w64.org/ or via MSYS2
- Protocol Buffer Compiler (protoc): https://github.com/protocolbuffers/protobuf/releases

**For GPU Support:**

- **NVIDIA GPU**: CUDA Toolkit for Windows: https://developer.nvidia.com/cuda-downloads
- **AMD/Intel GPU**: OpenCL SDK (included with GPU drivers)
  - AMD: Install AMD Radeon Software Adrenalin
  - Intel: Install Intel Graphics Driver
  - OpenCL headers and libraries are typically at: `C:\Program Files\NVIDIA Corporation\OpenCL\` or with GPU drivers

### For Cross-Compilation from Linux

**Required on Linux:**

- MinGW-w64 cross-compiler: `sudo apt install mingw-w64`
- Windows OpenCL SDK files (headers and libraries)

## Method 1: Native Windows Build (Recommended)

This method builds directly on Windows and is more reliable for GPU support.

### Step 1: Install Prerequisites

**1.1 Install Go:**

```powershell
# Download and install Go from https://golang.org/dl/
# Verify installation
go version
```

**1.2 Install MinGW-w64 via MSYS2 (Recommended):**

```powershell
# Download and install MSYS2 from https://www.msys2.org/
# In MSYS2 terminal:
pacman -S mingw-w64-x86_64-gcc mingw-w64-x86_64-go

# Add to PATH (in Windows System Environment Variables):
C:\msys64\mingw64\bin
```

**1.3 Install Protocol Buffer Compiler:**

```powershell
# Download protoc from GitHub releases
# Extract to C:\protoc
# Add to PATH:
C:\protoc\bin

# Verify
protoc --version
```

**1.4 Install Go protoc plugins:**

```powershell
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Step 2: Clone Repository

```powershell
git clone https://github.com/xyplex3/RedTeamCoin.git
cd RedTeamCoin
```

### Step 3: Install Go Dependencies

```powershell
go mod download
```

### Step 4: Generate Protocol Buffer Code

```powershell
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/mining.proto
```

### Step 5: Build Client

**Option A: CPU-only build (simplest):**

```powershell
go build -o bin/client.exe ./client
```

**Option B: OpenCL build (GPU support):**

```powershell
# Set CGO compiler
$env:CC = "gcc"
$env:CXX = "g++"

# Build with OpenCL
go build -tags opencl -o bin/client.exe ./client
```

**Option C: CUDA build (NVIDIA only):**

```powershell
# Compile CUDA kernel
nvcc -c -m64 -O3 client/mine.cu -o client/mine.o

# Build with CUDA
$env:CGO_LDFLAGS = "-L. -lcuda -lcudart"
go build -tags cuda -o bin/client.exe ./client
```

### Step 6: Verify OpenCL Runtime

For OpenCL builds, ensure OpenCL.dll is available:

```powershell
# Check for OpenCL.dll
dir "C:\Windows\System32\OpenCL.dll"

# Or check GPU driver installation directory
dir "C:\Program Files\NVIDIA Corporation\OpenCL\OpenCL.dll"
```

If not found, install/update your GPU drivers.

## Method 2: Cross-Compilation from Linux

### Step 1: Install Cross-Compilation Tools (Linux)

```bash
# Install MinGW-w64
sudo apt update
sudo apt install mingw-w64
```

### Step 2: Obtain Windows OpenCL SDK

You need Windows OpenCL headers and libraries. Several options:

**Option A: Extract from OpenCL SDK:**

1. Download Intel OpenCL SDK for Windows
2. Extract `CL/` headers and `OpenCL.lib`
3. Copy to MinGW paths:

```bash
# Create directories
sudo mkdir -p /usr/x86_64-w64-mingw32/include/CL
sudo mkdir -p /usr/x86_64-w64-mingw32/lib

# Copy OpenCL headers (from SDK)
sudo cp -r path/to/opencl-sdk/include/CL/* /usr/x86_64-w64-mingw32/include/CL/

# Convert OpenCL.lib to .a format
# Download OpenCL.lib from Windows OpenCL SDK, then:
sudo x86_64-w64-mingw32-dlltool -d OpenCL.def -l /usr/x86_64-w64-mingw32/lib/libOpenCL.a
```

**Option B: Use pre-built libraries:**

```bash
# Download from: https://github.com/KhronosGroup/OpenCL-Headers
git clone https://github.com/KhronosGroup/OpenCL-Headers.git
sudo cp -r OpenCL-Headers/CL /usr/x86_64-w64-mingw32/include/

# For library, use MinGW OpenCL stubs or extract from Windows
```

### Step 3: Cross-Compile

```bash
# From RedTeamCoin directory
make build-windows-opencl
```

### Step 4: Deploy to Windows

Copy the built executable to Windows:

```bash
scp bin/client-windows-opencl.exe user@windows-machine:C:/RedTeamCoin/
```

On Windows, ensure OpenCL.dll is available (comes with GPU drivers).

## Connecting to Mining Pool Server

### Understanding Authentication Tokens

Each RedTeamCoin server generates a unique authentication token when it starts. This token is required for:

- API access (viewing stats, controlling miners)
- Web dashboard access

**Important:** The authentication token is displayed in the server console when it starts and is unique per server instance.

### Configuration Options

#### Method 1: Command-Line Flag (Recommended)

```powershell
# Connect to remote server with specific address
client.exe -server 192.168.1.100:50051

# Or using shorthand
client.exe -s mining-pool.example.com:50051
```

#### Method 2: Environment Variable

```powershell
# PowerShell
$env:RTC_CLIENT_SERVER_ADDRESS = "192.168.1.100:50051"
client.exe

# Command Prompt
set RTC_CLIENT_SERVER_ADDRESS=192.168.1.100:50051
client.exe
```

#### Method 3: Default (localhost)

```powershell
# Connects to localhost:50051 by default
client.exe
```

### Full Connection Example with GPU

```powershell
# Connect to remote server with GPU mining enabled
$env:RTC_CLIENT_MINING_GPU_ENABLED = "true"
client.exe -server mining-pool.example.com:50051

# Connect with hybrid CPU+GPU mode
$env:RTC_CLIENT_MINING_HYBRID_MODE = "true"
$env:RTC_CLIENT_MINING_GPU_ENABLED = "true"
client.exe -s 192.168.1.100:50051

# Connect with CPU only (disable GPU)
$env:RTC_CLIENT_MINING_GPU_ENABLED = "false"
client.exe -server 192.168.1.100:50051
```

### Accessing the Web Dashboard

The mining pool server provides a web dashboard for monitoring and control.

#### Step 1: Get the Authentication Token

When the server starts, it displays output like:

```text
===========================================
RedTeamCoin Mining Pool Server
===========================================
Authentication token: abc123def456ghi789xyz

Web Dashboard: http://localhost:8080?token=abc123def456ghi789xyz
===========================================
```

#### Step 2: Access the Dashboard

Copy the complete URL with token and paste it into your browser:

```text
http://<server-ip>:8080?token=abc123def456ghi789xyz
```

**For HTTPS (if TLS is enabled):**

```text
https://<server-ip>:8443?token=abc123def456ghi789xyz
```

**Note:** Accept the security warning for self-signed certificates (click "Advanced" → "Proceed").

### API Access with Authentication

If you need to access the API programmatically:

**PowerShell:**

```powershell
$token = "abc123def456ghi789xyz"
$headers = @{
    "Authorization" = "Bearer $token"
}

Invoke-RestMethod -Uri "http://192.168.1.100:8080/api/stats" -Headers $headers
```

**Python:**

```python
import requests

token = "abc123def456ghi789xyz"
headers = {"Authorization": f"Bearer {token}"}

response = requests.get("http://192.168.1.100:8080/api/stats", headers=headers)
print(response.json())
```

**cURL (Git Bash on Windows):**

```bash
curl -H "Authorization: Bearer abc123def456ghi789xyz" http://192.168.1.100:8080/api/stats
```

### Security Best Practices

1. **Keep Token Secret:** The authentication token grants full control over the mining pool
2. **Use HTTPS:** Enable TLS on the server for encrypted communications
3. **Firewall Rules:** Restrict access to ports 50051 (gRPC) and 8080/8443 (HTTP/HTTPS)
4. **Rotate Tokens:** Restart the server periodically to generate new tokens, or set custom tokens:

```bash
# On server (Linux)
export RTC_AUTH_TOKEN="your-strong-custom-token-here"
./bin/server
```

## Environment Variables Reference

**Client Configuration:**

- `RTC_CLIENT_SERVER_ADDRESS` - Server address (default: `localhost:50051`)
- `RTC_CLIENT_MINING_GPU_ENABLED` - Enable GPU mining (`true`/`false`, default: `true`)
- `RTC_CLIENT_MINING_HYBRID_MODE` - Enable CPU+GPU hybrid mode (`true`/`false`, default: `false`)

**Examples:**

```powershell
# Disable GPU mining
$env:RTC_CLIENT_MINING_GPU_ENABLED = "false"
client.exe

# Enable hybrid mode
$env:RTC_CLIENT_MINING_HYBRID_MODE = "true"
client.exe

# Connect to specific server
$env:RTC_CLIENT_SERVER_ADDRESS = "192.168.1.100:50051"
client.exe
```

## Troubleshooting

### Build Issues

#### Error: "gcc: command not found"

```powershell
# Install MinGW-w64 via MSYS2
# Add C:\msys64\mingw64\bin to PATH
```

#### Error: "cannot find -lOpenCL"

```powershell
# Install GPU drivers with OpenCL support
# For NVIDIA: Install CUDA Toolkit
# For AMD: Install AMD Radeon Software
# For Intel: Install Intel Graphics Driver
```

**Error: "undefined reference to `clGetPlatformIDs`"**

```powershell
# OpenCL library not linked correctly
# Ensure GPU drivers are installed
# Try: $env:CGO_LDFLAGS = "-lOpenCL"
```

### Runtime Issues

#### Error: "OpenCL.dll not found"

- Install/update GPU drivers
- Copy OpenCL.dll to same directory as client.exe
- Check: `C:\Windows\System32\OpenCL.dll`

#### Error: "No OpenCL platforms found"

- Update GPU drivers
- Verify GPU is detected: Run `clinfo` (if available)
- Use CPU-only build if no GPU available

**Connection refused errors:**

- Verify server is running
- Check firewall allows port 50051
- Use correct server IP address
- Test with: `telnet server-ip 50051`

**Authentication errors (API/Dashboard):**

- Verify token matches server output
- Check token is included in URL or Authorization header
- Ensure no extra spaces in token string

#### "Failed to connect to pool"

```powershell
# Test network connectivity
ping server-ip
Test-NetConnection -ComputerName server-ip -Port 50051

# Check server address is correct
client.exe -server 192.168.1.100:50051
```

### Performance Issues

**Low hash rate:**

- Ensure GPU mining is enabled: `$env:RTC_CLIENT_MINING_GPU_ENABLED = "true"`
- Check GPU is being used (Task Manager → Performance → GPU)
- Try hybrid mode: `$env:RTC_CLIENT_MINING_HYBRID_MODE = "true"`
- Update GPU drivers

**High CPU usage:**

- Use GPU-only mode (don't enable RTC_CLIENT_MINING_HYBRID_MODE)
- Check CPU throttling is not set on server dashboard

## Building from Source - Complete Example

```powershell
# Full build process on Windows

# 1. Clone repository
git clone https://github.com/xyplex3/RedTeamCoin.git
cd RedTeamCoin

# 2. Install dependencies
go mod download

# 3. Generate protobuf code
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/mining.proto

# 4. Build (choose one):

# CPU only
go build -o bin/client.exe ./client

# OpenCL (GPU)
$env:CC = "gcc"
go build -tags opencl -o bin/client.exe ./client

# CUDA (NVIDIA)
nvcc -c -m64 -O3 client/mine.cu -o client/mine.o
go build -tags cuda -o bin/client.exe ./client

# 5. Run client
$env:RTC_CLIENT_SERVER_ADDRESS = "192.168.1.100:50051"
$env:RTC_CLIENT_MINING_GPU_ENABLED = "true"
.\bin\client.exe
```

## Deployment Checklist

- [ ] Go 1.21+ installed
- [ ] MinGW-w64 installed (for OpenCL/CUDA builds)
- [ ] GPU drivers installed (NVIDIA/AMD/Intel)
- [ ] OpenCL.dll present (comes with drivers)
- [ ] Server IP address known
- [ ] Authentication token obtained from server console
- [ ] Firewall rules allow port 50051 (gRPC)
- [ ] RTC_CLIENT_MINING_GPU_ENABLED environment variable set (if using GPU)
- [ ] Client built successfully
- [ ] Connection tested to server

## Additional Resources

- [GPU_MINING.md](GPU_MINING.md) - Detailed GPU mining guide
- [REMOTE_SERVER_SETUP.md](REMOTE_SERVER_SETUP.md) - Remote server configuration
- [TLS_SETUP.md](TLS_SETUP.md) - HTTPS/TLS setup for secure connections
- [README.md](README.md) - Main project documentation

## Getting Help

If you encounter issues:

1. Check this troubleshooting guide
2. Verify all prerequisites are installed
3. Check GPU drivers are up to date
4. Test with CPU-only build first
5. Report issues at: https://github.com/xyplex3/RedTeamCoin/issues
