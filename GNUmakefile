GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
KAFKA_BOOTSTRAP_SERVERS ?= localhost:9092
# Kafka version for testing - use "3.9.1" for Kafka 3.x or "4.1.0" for Kafka 4.x
KAFKA_VERSION ?= 3.9.1
default: build

build:
	go build .

test:
	 go test ./kafka -v $(TESTARGS)

testacc:
	GODEBUG=x509ignoreCN=0 \
	KAFKA_BOOTSTRAP_SERVERS=$(KAFKA_BOOTSTRAP_SERVERS) \
	KAFKA_CA_CERT=../secrets/ca.crt \
	KAFKA_CLIENT_CERT=../secrets/client.pem \
	KAFKA_CLIENT_KEY=../secrets/client.key \
	KAFKA_CLIENT_KEY_PASSPHRASE=test-pass \
	KAFKA_SKIP_VERIFY=false \
	KAFKA_ENABLE_TLS=true \
	TF_ACC=1 go test ./kafka -v $(TESTARGS) -timeout 9m -count=1

.PHONY: build test testacc

# Docker compose helpers for different Kafka versions
# These set version-specific configuration via environment variables
docker-up:
	KAFKA_VERSION=$(KAFKA_VERSION) docker compose up -d --wait

docker-up-kafka3:
	KAFKA_VERSION=3.9.1 \
	KAFKA_LOG4J_ROOT_LOGLEVEL=INFO \
	KAFKA_LOG4J_LOGGERS=kafka.server.ClientQuotaManager=WARN \
	KAFKA_AUTHORIZER_CLASS_NAME=org.apache.kafka.metadata.authorizer.StandardAuthorizer \
	KAFKA_ALLOW_EVERYONE_IF_NO_ACL_FOUND=true \
	docker compose up -d --wait

docker-up-kafka4:
	KAFKA_VERSION=4.1.0 \
	KAFKA_LOG4J_ROOT_LOGLEVEL=INFO \
	KAFKA_LOG4J_LOGGERS=kafka.server.ClientQuotaManager=WARN \
	KAFKA_AUTHORIZER_CLASS_NAME=org.apache.kafka.metadata.authorizer.StandardAuthorizer \
	KAFKA_ALLOW_EVERYONE_IF_NO_ACL_FOUND=true \
	docker compose up -d --wait

docker-down:
	docker compose down -v
