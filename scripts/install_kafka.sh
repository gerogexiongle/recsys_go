#!/bin/bash
set -euo pipefail

# 与 /data/kafka/install_kafka.sh 保持一致，便于从项目目录部署到 algo-test-01

KAFKA_VERSION="3.7.0"
BASE_DIR="/data/kafka"
DATA_DIR="${BASE_DIR}/data"
CONF_DIR="${BASE_DIR}/conf"
LOG_DIR="${BASE_DIR}/logs"
CONTAINER_NAME="kafka"
HOST_IP=$(hostname -I | awk '{print $1}')

echo "========== 创建目录 =========="
mkdir -p "${DATA_DIR}" "${CONF_DIR}" "${LOG_DIR}"
chown -R 1000:1000 "${DATA_DIR}"
chmod 755 "${DATA_DIR}"

if [[ -n "$(ls -A "${DATA_DIR}" 2>/dev/null)" ]]; then
  echo "========== 清理旧数据 =========="
  rm -rf "${DATA_DIR:?}"/*
  chown -R 1000:1000 "${DATA_DIR}"
fi

echo "========== 生成 server.properties（备查）=========="
cat > "${CONF_DIR}/server.properties" <<EOF
process.roles=broker,controller
node.id=1
controller.quorum.voters=1@localhost:9093
listeners=PLAINTEXT://:9092,CONTROLLER://:9093
advertised.listeners=PLAINTEXT://${HOST_IP}:9092
listener.security.protocol.map=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
controller.listener.names=CONTROLLER
inter.broker.listener.name=PLAINTEXT
log.dirs=/var/lib/kafka/data
num.partitions=3
default.replication.factor=1
offsets.topic.replication.factor=1
transaction.state.log.replication.factor=1
transaction.state.log.min.isr=1
min.insync.replicas=1
log.retention.hours=168
log.segment.bytes=1073741824
socket.send.buffer.bytes=102400
socket.receive.buffer.bytes=102400
socket.request.max.bytes=104857600
EOF

echo "========== 拉取 Kafka 镜像 =========="
docker pull "apache/kafka:${KAFKA_VERSION}"

echo "========== 删除旧容器 =========="
docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true

echo "========== 启动 Kafka =========="
docker run -d \
  --name "${CONTAINER_NAME}" \
  --restart=always \
  -p 9092:9092 \
  -p 9093:9093 \
  -e KAFKA_NODE_ID=1 \
  -e KAFKA_PROCESS_ROLES='broker,controller' \
  -e KAFKA_CONTROLLER_QUORUM_VOTERS='1@localhost:9093' \
  -e KAFKA_LISTENERS='PLAINTEXT://:9092,CONTROLLER://:9093' \
  -e KAFKA_ADVERTISED_LISTENERS="PLAINTEXT://${HOST_IP}:9092" \
  -e KAFKA_LISTENER_SECURITY_PROTOCOL_MAP='CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT' \
  -e KAFKA_CONTROLLER_LISTENER_NAMES='CONTROLLER' \
  -e KAFKA_INTER_BROKER_LISTENER_NAME='PLAINTEXT' \
  -e KAFKA_LOG_DIRS='/var/lib/kafka/data' \
  -e KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1 \
  -e KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=1 \
  -e KAFKA_TRANSACTION_STATE_LOG_MIN_ISR=1 \
  -e KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS=0 \
  -v "${DATA_DIR}:/var/lib/kafka/data:z" \
  "apache/kafka:${KAFKA_VERSION}"

echo "========== 等待 Kafka 就绪 =========="
ready=0
for _ in $(seq 1 30); do
  if docker exec "${CONTAINER_NAME}" /opt/kafka/bin/kafka-broker-api-versions.sh \
    --bootstrap-server localhost:9092 &>/dev/null; then
    ready=1
    break
  fi
  if ! docker ps --format '{{.Names}}' | grep -qx "${CONTAINER_NAME}"; then
    echo "容器已退出，最近日志："
    docker logs --tail 40 "${CONTAINER_NAME}" 2>&1 || true
    exit 1
  fi
  sleep 2
done
if [[ "${ready}" -ne 1 ]]; then
  echo "Kafka 启动超时，最近日志："
  docker logs --tail 40 "${CONTAINER_NAME}" 2>&1 || true
  exit 1
fi

echo ""
echo "========== 容器状态 =========="
docker ps --filter "name=${CONTAINER_NAME}"

echo ""
echo "========== 创建测试 Topic =========="
docker exec "${CONTAINER_NAME}" /opt/kafka/bin/kafka-topics.sh \
  --create --if-not-exists \
  --topic test \
  --bootstrap-server localhost:9092 \
  --partitions 3 \
  --replication-factor 1

echo ""
echo "========== 查看 Topic =========="
docker exec "${CONTAINER_NAME}" /opt/kafka/bin/kafka-topics.sh \
  --list --bootstrap-server localhost:9092

echo ""
echo "========== Kafka 信息 =========="
echo "Bootstrap Server: ${HOST_IP}:9092"
echo ""
echo "docker exec -it ${CONTAINER_NAME} /opt/kafka/bin/kafka-console-producer.sh --topic test --bootstrap-server localhost:9092"
echo "docker exec -it ${CONTAINER_NAME} /opt/kafka/bin/kafka-console-consumer.sh --topic test --from-beginning --bootstrap-server localhost:9092"
