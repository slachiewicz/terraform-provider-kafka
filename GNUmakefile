GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
KAFKA_BOOTSTRAP_SERVERS ?= localhost:9092
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
docker-up:
	KAFKA_VERSION=$(KAFKA_VERSION) docker compose up -d --wait

docker-up-3.9.1:
	docker compose -f docker-compose.yaml -f docker-compose.kafka-3.9.1.yaml up -d --wait

docker-up-4.1.0:
	docker compose -f docker-compose.yaml -f docker-compose.kafka-4.1.0.yaml up -d --wait

docker-down:
	docker compose down -v
