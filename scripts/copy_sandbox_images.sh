#!/bin/bash
# =============================================================================
# WeKnora Sandbox 镜像同步脚本 (skopeo 版本)
#
# 使用 skopeo（无需 Docker daemon，无需构建）将官方 sandbox 镜像从 Docker Hub
# 复制到 Cube 集群可访问的仓库。
#
# 优势：不需要重新构建，直接复制官方已发布镜像，速度快，不会出错。
#
# 依赖: skopeo (brew install skopeo)
# 用法:
#   ./scripts/copy_sandbox_images.sh --registry ccr.ccs.tencentyun.com/my-ns
#   ./scripts/copy_sandbox_images.sh --registry harbor.example.com/weknora --image desktop-cube
# =============================================================================

set -euo pipefail

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()  { echo -e "${BLUE}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_ok()    { echo -e "${GREEN}[OK]${NC} $1"; }

usage() {
    cat <<EOF
用法: $0 --registry <target-registry> [选项]

必填:
  --registry  目标仓库地址 (如 ccr.ccs.tencentyun.com/my-ns)

可选:
  --image     要复制的镜像: desktop-cube | cube | all (默认: desktop-cube)
  --source    源仓库 (默认: docker.io/wechatopenai/weknora-sandbox)
  --dry-run   打印命令，不实际操作
  -h, --help  显示帮助

示例:
  # 复制 desktop-cube 镜像到腾讯云 CCR
  $0 --registry ccr.ccs.tencentyun.com/my-namespace

  # 复制所有 sandbox 镜像
  $0 --registry harbor.internal.example.com/weknora --image all

  # 仅预览
  $0 --registry ccr.ccs.tencentyun.com/my-ns --dry-run

说明:
  skopeo 直接从 Docker Hub 复制镜像到目标仓库，无需 pull + tag + push，
  也无需 Docker daemon。镜像 blob 直接从源仓库流式传输到目标仓库。
EOF
    exit 0
}

# 解析参数
REGISTRY=""
IMAGE_SELECT="desktop-cube"
SOURCE="docker.io/wechatopenai/weknora-sandbox"
DRY_RUN=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --registry) REGISTRY="$2"; shift 2 ;;
        --image)    IMAGE_SELECT="$2"; shift 2 ;;
        --source)   SOURCE="$2"; shift 2 ;;
        --dry-run)  DRY_RUN=true; shift ;;
        -h|--help)  usage ;;
        *)          log_error "未知选项: $1"; usage ;;
    esac
done

if [[ -z "$REGISTRY" ]]; then
    log_error "缺少 --registry（目标仓库地址）"; usage
fi

REGISTRY="${REGISTRY%/}"

# 待复制的镜像 tags（按 target 分类）
TAGS=()
case "$IMAGE_SELECT" in
    desktop-cube)
        TAGS=("main-desktop-cube")
        ;;
    cube)
        TAGS=("main-cube")
        ;;
    desktop)
        TAGS=("main-desktop")
        ;;
    sandbox)
        TAGS=("main" "latest")
        ;;
    all)
        TAGS=("main" "latest" "main-cube" "latest-cube" "main-desktop" "latest-desktop" "main-desktop-cube" "latest-desktop-cube")
        ;;
    *)
        log_error "未知 --image: $IMAGE_SELECT"
        log_error "可选: desktop-cube | cube | desktop | sandbox | all"
        exit 1
        ;;
esac

# 检查 skopeo
check_skopeo() {
    if ! command -v skopeo &>/dev/null; then
        log_error "skopeo 未安装。请运行: brew install skopeo"
        exit 1
    fi
    log_info "skopeo 版本: $(skopeo --version 2>&1 | head -1)"
}

# 检查源镜像是否存在
check_source() {
    local tag="$1"
    local src="docker://${SOURCE}:${tag}"
    log_info "检查源镜像: $src"
    if skopeo inspect "$src" &>/dev/null; then
        log_ok "源镜像存在: $src"
        return 0
    else
        log_error "源镜像不存在或无法访问: $src"
        return 1
    fi
}

# 复制单个 tag
copy_tag() {
    local tag="$1"
    local src="docker://${SOURCE}:${tag}"
    local dst="docker://${REGISTRY}/weknora-sandbox:${tag}"

    log_info "复制: $src -> $dst"

    if [[ "$DRY_RUN" == "true" ]]; then
        echo "  skopeo copy --preserve-digests $src $dst"
        return 0
    fi

    skopeo copy --preserve-digests "$src" "$dst"
    log_ok "复制完成: $dst"
}

main() {
    check_skopeo

    log_info "源仓库: $SOURCE"
    log_info "目标仓库: $REGISTRY"
    log_info "待复制 tags: ${TAGS[*]}"

    echo ""

    # 先快速检查源镜像是否可访问
    local first_tag="${TAGS[0]}"
    if ! check_source "$first_tag"; then
        log_error ""
        log_error "无法访问源镜像。请确认："
        log_error "1. 网络可以访问 Docker Hub (index.docker.io)"
        log_error "2. 镜像 tag '$first_tag' 存在"
        log_error ""
        log_error "如果本地网络也无法访问 Docker Hub，请使用 build_sandbox_buildah.sh 从源码构建。"
        exit 1
    fi

    echo ""

    for tag in "${TAGS[@]}"; do
        copy_tag "$tag"
    done

    echo ""
    log_ok "全部同步完成！"
    echo ""
    log_info "=== Cube 配置步骤 ==="
    echo ""
    local desktop_image="${REGISTRY}/weknora-sandbox:main-desktop-cube"
    echo "1. 如果复制了 desktop-cube 镜像，在 WeKnora 管理后台中："
    echo "   「设置 → 沙箱 → 编辑 → 模板 → 桌面模板构建镜像」"
    echo "   填入: $desktop_image"
    echo ""
    echo "2. 在失败的桌面模板卡片上点击「重建」"
    echo ""
    echo "3. 或者通过 API 更新："
    echo '   PATCH /api/v1/workspaces/{id}/sandbox'
    echo '   { "cube": { "desktop_template_image": "'"$desktop_image"'" } }'
}

main