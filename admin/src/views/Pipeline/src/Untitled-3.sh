#!/bin/bash
set -e

export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"

# 配置信息
ALIYUN_TOKEN="pt-LS4afw0HEeaaMWAFRP44bvow_c588a23d-c807-4c85-bbf5-2df906f80f4a"
ALIYUN_USER="coller"
CODEUP_BASE="https://$ALIYUN_USER:$ALIYUN_TOKEN@codeup.aliyun.com/64dc6e9a9210862005710a57"

# 检测 Podman Compose 命令
compose_cmd() {
    if podman compose version >/dev/null 2>&1; then
        echo "podman compose"
        return 0
    fi
    if command -v podman-compose >/dev/null 2>&1; then
        echo "podman-compose"
        return 0
    fi
    return 1
}

replace_dir() {
    SRC=$1
    DST=$2
    if [ ! -d "$SRC" ]; then
        echo "目录不存在，跳过同步: $SRC"
        return 0
    fi
    rm -rf "$DST"
    mkdir -p "$DST"
    cp -a "$SRC"/. "$DST"/
}

ensure_dir() {
    DST=$1
    mkdir -p "$DST"
}

sync_repo() {
    REPO_URL=$1
    DIR_NAME=$2
    if [ -d "$DIR_NAME/.git" ]; then
        echo "发现缓存 [$DIR_NAME]: 执行 git pull 增量更新..."
        cd "$DIR_NAME" && git pull origin main && cd - >/dev/null
    else
        echo "首次拉取 [$DIR_NAME]: 执行 git clone --depth 1..."
        git clone --depth 1 --single-branch "$REPO_URL" "$DIR_NAME"
    fi
}

sync_app_repo() {
    REPO_NAME=$1
    APP_NAME=$2
    TARGET_DIR="./plugins/app/$APP_NAME"
    REPO_URL="$CODEUP_BASE/$REPO_NAME.git"
    if git ls-remote "$REPO_URL" HEAD >/dev/null 2>&1; then
        if [ -d "$TARGET_DIR/.git" ]; then
            echo "发现应用缓存 [$APP_NAME]: 执行 git fetch + reset..."
            cd "$TARGET_DIR" \
              && git fetch --depth 1 origin \
              && git reset --hard FETCH_HEAD \
              && cd - >/dev/null
        else
            echo "拉取应用仓库 [$REPO_NAME] -> [$APP_NAME]"
            rm -rf "$TARGET_DIR" "$TARGET_DIR.tmp"
            git clone --depth 1 "$REPO_URL" "$TARGET_DIR.tmp"
            mv "$TARGET_DIR.tmp" "$TARGET_DIR"
        fi
    else
        echo "应用仓库不存在，跳过 [$REPO_NAME]"
    fi
}

COMPOSE_CMD=$(compose_cmd) || {
    echo "未找到 podman compose 或 podman-compose，请检查环境"
    exit 127
}

echo "==== 1. 拉取主仓与 Go 依赖仓库 ===="
sync_repo "$CODEUP_BASE/go-core.git" ./secure/go-core
sync_repo "$CODEUP_BASE/go-common.git" ./secure/go-common
sync_repo "$CODEUP_BASE/go-fast.git" ./go-fast
sync_repo "$CODEUP_BASE/go-shoply.git" ./go-shoply
sync_repo "$CODEUP_BASE/go-utils.git" ./go-utils
sync_repo "$CODEUP_BASE/go-es.git" ./go-es
sync_repo "$CODEUP_BASE/go-cache.git" ./go-cache
sync_repo "$CODEUP_BASE/go-storage.git" ./go-storage
sync_repo "$CODEUP_BASE/shoply-dist-site.git" ./template/common/site
sync_repo "$CODEUP_BASE/shoply-dist-admin.git" ./template/admin
sync_repo "$CODEUP_BASE/theme-default.git" ./template/themes/default

echo "当前构建版本: $PIPELINE_VERSION"
echo "==== 4. 宿主机编译 Go 二进制文件 ===="
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 GOPROXY=https://goproxy.cn,direct /usr/local/go/bin/go build -tags plugin -ldflags="-s -w -X fastix.ai/go-fast/constant.AppVersion=${PIPELINE_VERSION}" -trimpath -o shoply main.go

echo "==== 5. 宿主机安装 PHP 依赖 ===="
composer install --no-dev --no-scripts --no-autoloader --ignore-platform-reqs -d plugins
cd plugins && composer dump-autoload --optimize && cd ..

cat <<EOF > ./Dockerfile
FROM php:8.2-cli-alpine

LABEL maintainer="Shoply Base Image"

WORKDIR /var/www/shoply
ENV TZ=Asia/Shanghai

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

RUN apk add --no-cache \
    tzdata \
    libzip \
    libpng \
    libjpeg-turbo \
    libwebp \
    freetype \
    icu-libs \
    oniguruma \
    libstdc++ \
    bash \
    libcurl \
    libxml2

RUN set -eux; \
    apk add --no-cache --virtual .build-deps \
    \$PHPIZE_DEPS \
    libzip-dev libpng-dev libjpeg-turbo-dev libwebp-dev freetype-dev \
    oniguruma-dev icu-dev zlib-dev openssl-dev linux-headers \
    libxml2-dev curl-dev; \
    \
    docker-php-ext-configure gd --with-freetype --with-jpeg --with-webp; \
    \
    docker-php-ext-install -j\$(getconf _NPROCESSORS_ONLN) \
    pdo_mysql \
    bcmath \
    mbstring \
    zip \
    gd \
    intl \
    sockets \
    opcache \
    pcntl \
    curl \
    xml; \
    \
    pecl install protobuf; \
    docker-php-ext-enable protobuf; \
    \
    apk del .build-deps; \
    rm -rf /tmp/* /usr/local/lib/php/test /usr/local/lib/php/doc

COPY --from=composer:latest /usr/bin/composer /usr/bin/composer

EOF

# ==================== 新增：Podman 打包与推送 ====================
echo "==== 6. 使用 Podman 构建 ARM64 生产镜像 ===="
# 这里的命名空间和仓库名，换成你前面在阿里云 ACR 开通的实际名称
REGISTRY_URL="crpi-vvvid8bdlzbcxo4s.cn-shenzhen.personal.cr.aliyuncs.com"
NAMESPACE="aihop"
IMAGE_NAME="shoply-base"
TAG="${PIPELINE_VERSION:-latest}"

FULL_IMAGE_PATH="${REGISTRY_URL}/${NAMESPACE}/${IMAGE_NAME}:${TAG}"

# 强制指定 arm64 架构打包，并直接作为最终标签
podman build --platform linux/arm64 -t "${FULL_IMAGE_PATH}" .