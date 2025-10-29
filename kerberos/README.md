# Kafka GSSAPI/Kerberos Docker Setup

This directory contains a complete Docker Compose setup for testing Kafka with GSSAPI (Kerberos) authentication.

## Overview

The setup includes:
- **Kerberos KDC** (Key Distribution Center) server
- **Kafka 4.x broker** configured with GSSAPI authentication
- Automated scripts for Kerberos principal and keytab generation

## Quick Start

### 1. Start the environment

```bash
# Start KDC and Kafka broker
docker-compose -f docker-compose.gssapi.yaml up -d

# Wait for services to be ready (about 10 seconds)
sleep 10
```

### 2. Initialize Kerberos

```bash
# Run the setup script to create principals and keytabs
./kerberos/setup-kerberos.sh
```

This will create:
- `kafka/kafka.kafka.local@KAFKA.LOCAL` principal (broker)
- `kafka-client@KAFKA.LOCAL` principal (client)
- `./kerberos/kafka.keytab` (broker keytab)
- `./kerberos/kafka-client.keytab` (client keytab)

### 3. Restart Kafka to pick up the keytab

```bash
docker-compose -f docker-compose.gssapi.yaml restart kafka-gssapi
```

### 4. Test with Terraform

Create a test configuration:

```hcl
provider "kafka" {
  bootstrap_servers = ["localhost:9092"]
  
  sasl_mechanism               = "gssapi"
  gssapi_auth_type             = "KRB5_KEYTAB_AUTH"
  gssapi_username              = "kafka-client@KAFKA.LOCAL"
  gssapi_keytab_path           = "./kerberos/kafka-client.keytab"
  gssapi_kerberos_config_path  = "./kerberos/krb5.conf"
  gssapi_service_name          = "kafka"
  gssapi_realm                 = "KAFKA.LOCAL"
  
  tls_enabled = false  # SASL_PLAINTEXT doesn't use TLS
}

resource "kafka_topic" "test" {
  name               = "gssapi-test"
  replication_factor = 1
  partitions         = 3
}
```

Then run:

```bash
terraform init
terraform plan
terraform apply
```

## Configuration Details

### Kerberos Realm

- **Realm**: `KAFKA.LOCAL`
- **KDC**: `kdc.kafka.local`
- **Admin Password**: `kerberos`

### Kafka Configuration

- **Broker**: `localhost:9092`
- **Protocol**: `SASL_PLAINTEXT`
- **SASL Mechanism**: `GSSAPI`
- **Service Name**: `kafka`

### File Structure

```
kerberos/
├── krb5.conf                  # Kerberos client configuration
├── kafka_server_jaas.conf     # JAAS configuration for Kafka broker
├── setup-kerberos.sh          # Script to initialize Kerberos
├── kafka.keytab              # Broker keytab (generated)
└── kafka-client.keytab       # Client keytab (generated)
```

## Troubleshooting

### Check KDC status

```bash
docker logs kdc
```

### Check Kafka logs

```bash
docker logs kafka-gssapi
```

### Verify Kerberos tickets

```bash
# List principals
docker exec -it kdc kadmin.local -q "listprincs"

# Check keytab
docker exec -it kdc klist -kt /tmp/kafka.keytab
```

### Test Kerberos authentication

```bash
# Get a Kerberos ticket using the client keytab
kinit -kt ./kerberos/kafka-client.keytab kafka-client@KAFKA.LOCAL

# List current tickets
klist
```

### Common Issues

1. **"Clock skew too great"**: Ensure your system clock is synchronized
   ```bash
   # On Linux
   sudo ntpdate -s time.nist.gov
   ```

2. **"Server not found in Kerberos database"**: Restart the Kafka container after running setup-kerberos.sh
   ```bash
   docker-compose -f docker-compose.gssapi.yaml restart kafka-gssapi
   ```

3. **"Cannot contact any KDC"**: Make sure the KDC container is running
   ```bash
   docker-compose -f docker-compose.gssapi.yaml ps
   ```

## Cleanup

Stop and remove all containers and volumes:

```bash
docker-compose -f docker-compose.gssapi.yaml down -v
rm -f ./kerberos/*.keytab
```

## Testing with kafka-console-producer/consumer

For manual testing with Kafka tools:

```bash
# Create client properties
cat > client-gssapi.properties <<EOF
security.protocol=SASL_PLAINTEXT
sasl.mechanism=GSSAPI
sasl.kerberos.service.name=kafka
sasl.jaas.config=com.sun.security.auth.module.Krb5LoginModule required \
  useKeyTab=true \
  storeKey=true \
  keyTab="./kerberos/kafka-client.keytab" \
  principal="kafka-client@KAFKA.LOCAL";
EOF

# Produce messages
docker exec -it kafka-gssapi kafka-console-producer \
  --bootstrap-server localhost:9092 \
  --topic test-topic \
  --producer.config /path/to/client-gssapi.properties

# Consume messages
docker exec -it kafka-gssapi kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic test-topic \
  --from-beginning \
  --consumer.config /path/to/client-gssapi.properties
```

## References

- [Kafka GSSAPI Documentation](https://kafka.apache.org/documentation/#security_sasl_kerberos)
- [Confluent GSSAPI Guide](https://docs.confluent.io/platform/current/kafka/authentication_sasl/authentication_sasl_gssapi.html)
- [MIT Kerberos Documentation](https://web.mit.edu/kerberos/krb5-latest/doc/)
