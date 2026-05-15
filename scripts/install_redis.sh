#!/bin/bash

set -e

# =========================
# Redis Docker 一键部署脚本
# =========================

REDIS_VERSION="7.2"
REDIS_PASSWORD="test123"# your_redis_password

BASE_DIR="/data/redis"
DATA_DIR="${BASE_DIR}/data"
CONF_DIR="${BASE_DIR}/conf"
LOG_DIR="${BASE_DIR}/logs"

CONTAINER_NAME="redis"

echo "========== 创建目录 =========="

mkdir -p ${DATA_DIR}
mkdir -p ${CONF_DIR}
mkdir -p ${LOG_DIR}

echo "========== 生成 redis.conf =========="

cat > ${CONF_DIR}/redis.conf <<EOF
bind 0.0.0.0
protected-mode yes
port 6379
timeout 0
tcp-keepalive 300

daemonize no

# 日志
loglevel notice
logfile ""

# 数据目录
dir /data

# RDB
save 900 1
save 300 10
save 60 10000

dbfilename dump.rdb

# AOF
appendonly yes
appendfilename "appendonly.aof"
appendfsync everysec

# 内存策略
maxmemory-policy allkeys-lru

# 密码
requirepass ${REDIS_PASSWORD}

# Keyspace notifications
notify-keyspace-events Ex

# 慢查询
slowlog-log-slower-than 10000
slowlog-max-len 128

# 客户端
tcp-backlog 511

# 最大连接数
maxclients 10000
EOF

echo "========== 拉取 Redis 镜像 =========="

docker pull redis:${REDIS_VERSION}

echo "========== 删除旧容器 =========="

docker rm -f ${CONTAINER_NAME} >/dev/null 2>&1 || true

echo "========== 启动 Redis =========="

docker run -d \
  --name ${CONTAINER_NAME} \
  --restart=always \
  -p 6379:6379 \
  -v ${DATA_DIR}:/data \
  -v ${CONF_DIR}/redis.conf:/usr/local/etc/redis/redis.conf \
  -v ${LOG_DIR}:/logs \
  redis:${REDIS_VERSION} \
  redis-server /usr/local/etc/redis/redis.conf

echo "========== Redis 启动完成 =========="

echo ""
echo "Redis 地址: 127.0.0.1:6379"
echo "Redis 密码: ${REDIS_PASSWORD}"
echo ""

echo "========== 容器状态 =========="
docker ps | grep redis || true

echo ""
echo "========== 测试连接 =========="

sleep 3

docker exec -it ${CONTAINER_NAME} redis-cli -a ${REDIS_PASSWORD} ping

echo ""
echo "========== 常用命令 =========="
echo "查看日志:"
echo "docker logs -f redis"

echo ""
echo "进入 Redis:"
echo "docker exec -it redis redis-cli -a ${REDIS_PASSWORD}"

echo ""
echo "停止 Redis:"
echo "docker stop redis"

echo ""
echo "启动 Redis:"
echo "docker start redis"