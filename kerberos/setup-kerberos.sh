#!/bin/bash

# Script to set up Kerberos principals and keytabs for Kafka GSSAPI testing
# This script should be run after starting the KDC container

set -e

echo "Setting up Kerberos principals for Kafka GSSAPI authentication..."

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REALM="KAFKA.LOCAL"
KDC_CONTAINER="kdc"
ADMIN_PASSWORD="kerberos"

echo -e "${YELLOW}Waiting for KDC to be ready...${NC}"
sleep 5

echo -e "${GREEN}Creating Kafka broker principal...${NC}"
docker exec -i ${KDC_CONTAINER} kadmin.local -q "addprinc -randkey kafka/kafka.kafka.local@${REALM}" || true

echo -e "${GREEN}Creating Kafka client principal...${NC}"
docker exec -i ${KDC_CONTAINER} kadmin.local -q "addprinc -randkey kafka-client@${REALM}" || true

echo -e "${GREEN}Exporting keytab for Kafka broker...${NC}"
docker exec -i ${KDC_CONTAINER} kadmin.local -q "ktadd -k /tmp/kafka.keytab kafka/kafka.kafka.local@${REALM}"

echo -e "${GREEN}Exporting keytab for Kafka client...${NC}"
docker exec -i ${KDC_CONTAINER} kadmin.local -q "ktadd -k /tmp/kafka-client.keytab kafka-client@${REALM}"

echo -e "${GREEN}Copying keytabs from KDC container...${NC}"
docker cp ${KDC_CONTAINER}:/tmp/kafka.keytab ./kerberos/kafka.keytab
docker cp ${KDC_CONTAINER}:/tmp/kafka-client.keytab ./kerberos/kafka-client.keytab

echo -e "${GREEN}Setting proper permissions on keytabs...${NC}"
chmod 600 ./kerberos/*.keytab

echo -e "${GREEN}Verifying keytabs...${NC}"
echo "Kafka broker keytab:"
docker exec -i ${KDC_CONTAINER} klist -kt /tmp/kafka.keytab
echo ""
echo "Kafka client keytab:"
docker exec -i ${KDC_CONTAINER} klist -kt /tmp/kafka-client.keytab

echo -e "${GREEN}✓ Kerberos setup complete!${NC}"
echo ""
echo "Keytabs created:"
echo "  - ./kerberos/kafka.keytab (broker)"
echo "  - ./kerberos/kafka-client.keytab (client)"
echo ""
echo "Principals created:"
echo "  - kafka/kafka.kafka.local@${REALM}"
echo "  - kafka-client@${REALM}"
