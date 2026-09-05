#!/bin/bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
cd "$DIR"

echo "================================================================"
echo "    🌱 息壤 (XiRang) v4.0: 跨平台多架构全矩阵交叉编译自动化"
echo "================================================================"

mkdir -p dist

build_target() {
    GOOS=$1
    GOARCH=$2
    OUTNAME=$3
    EXTRA_ENV=$4
    echo -n "[*] 正在构建 -> ${GOOS}/${GOARCH} (${OUTNAME}) ... "
    env CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH $EXTRA_ENV go build -ldflags="-s -w" -o "dist/${OUTNAME}" main.go embedded_default.go
    echo "✅ [完成]"
}

# 1. Windows (AMD64 / ARM64 / 386)
build_target "windows" "amd64" "xirang_win_x64.exe"
build_target "windows" "arm64" "xirang_win_arm64.exe"
build_target "windows" "386"   "xirang_win_x86_legacy.exe"
cp "dist/xirang_win_x64.exe" "dist/xirang.exe"
# 保持对旧名称的兼容别名
cp "dist/xirang_win_x64.exe" "dist/zen_agent_win_x64.exe"
cp "dist/xirang.exe" "dist/zen_agent.exe"

# 2. macOS (Apple Silicon / Intel)
build_target "darwin" "arm64" "xirang_mac_apple_silicon"
build_target "darwin" "amd64" "xirang_mac_intel"
cp "dist/xirang_mac_apple_silicon" "dist/xirang"

# 3. Linux (AMD64 / ARM64 / ARMv7 / RISC-V / LoongArch / MIPS64LE)
build_target "linux" "amd64"       "xirang_linux_x64"
build_target "linux" "arm64"       "xirang_linux_arm64_aarch64"
build_target "linux" "arm"         "xirang_linux_armv7_raspberrypi" "GOARM=7"
build_target "linux" "riscv64"     "xirang_linux_riscv64"
build_target "linux" "loong64"     "xirang_linux_loongarch64"
build_target "linux" "mips64le"    "xirang_linux_mips64le"

# 4. BSD
build_target "freebsd" "amd64" "xirang_freebsd_x64"

# 5. Builder 工具
cd builder && go build -o xirang_builder main.go && cp xirang_builder zen_builder && cd ..

echo "================================================================"
echo "🎉 息壤 (XiRang) 全平台矩阵编译成功！产物均在 dist/ 目录下"
ls -lh dist/
