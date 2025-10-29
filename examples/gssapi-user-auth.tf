# Example: Using GSSAPI (Kerberos) authentication with username/password

terraform {
  required_providers {
    kafka = {
      source = "Mongey/kafka"
    }
  }
}

provider "kafka" {
  bootstrap_servers = ["kafka.example.com:9092"]
  
  # SASL/GSSAPI configuration using username/password authentication
  sasl_mechanism               = "gssapi"
  gssapi_auth_type             = "KRB5_USER_AUTH"  # or "1"
  gssapi_username              = "kafka-client@EXAMPLE.COM"
  gssapi_password              = var.kerberos_password  # Use variable for sensitive data
  gssapi_kerberos_config_path  = "/etc/krb5.conf"
  gssapi_service_name          = "kafka"
  gssapi_realm                 = "EXAMPLE.COM"
  gssapi_disable_pafx_fast     = false
  
  tls_enabled = true
}

variable "kerberos_password" {
  description = "Kerberos password for authentication"
  type        = string
  sensitive   = true
}

# Example resources
resource "kafka_topic" "example" {
  name               = "example-topic"
  replication_factor = 3
  partitions         = 6
  
  config = {
    "retention.ms" = "86400000"  # 1 day
  }
}
