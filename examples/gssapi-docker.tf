# Example Terraform configuration for testing GSSAPI authentication
# with the local Docker Compose setup

terraform {
  required_providers {
    kafka = {
      source = "Mongey/kafka"
    }
  }
}

# Provider configuration using GSSAPI/Kerberos authentication
provider "kafka" {
  bootstrap_servers = ["localhost:9092"]
  
  # GSSAPI/Kerberos configuration
  sasl_mechanism               = "gssapi"
  gssapi_auth_type             = "KRB5_KEYTAB_AUTH"
  gssapi_username              = "kafka-client@KAFKA.LOCAL"
  gssapi_keytab_path           = "./kerberos/kafka-client.keytab"
  gssapi_kerberos_config_path  = "./kerberos/krb5.conf"
  gssapi_service_name          = "kafka"
  gssapi_realm                 = "KAFKA.LOCAL"
  
  # SASL_PLAINTEXT doesn't use TLS
  tls_enabled = false
}

# Example topic
resource "kafka_topic" "gssapi_test" {
  name               = "gssapi-test-topic"
  replication_factor = 1
  partitions         = 3
  
  config = {
    "cleanup.policy" = "delete"
    "retention.ms"   = "86400000"  # 1 day
  }
}

# Example ACL
resource "kafka_acl" "test_acl" {
  resource_name       = "gssapi-test-topic"
  resource_type       = "Topic"
  acl_principal       = "User:kafka-client"
  acl_host            = "*"
  acl_operation       = "Write"
  acl_permission_type = "Allow"
}

output "topic_name" {
  value       = kafka_topic.gssapi_test.name
  description = "The name of the created topic"
}

output "topic_partitions" {
  value       = kafka_topic.gssapi_test.partitions
  description = "Number of partitions"
}
