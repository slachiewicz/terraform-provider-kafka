# Example: Using GSSAPI (Kerberos) authentication with keytab

terraform {
  required_providers {
    kafka = {
      source = "Mongey/kafka"
    }
  }
}

provider "kafka" {
  bootstrap_servers = ["kafka.example.com:9092"]
  
  # SASL/GSSAPI configuration using keytab authentication
  sasl_mechanism               = "gssapi"
  gssapi_auth_type             = "KRB5_KEYTAB_AUTH"  # or "2"
  gssapi_username              = "kafka-client@EXAMPLE.COM"
  gssapi_keytab_path           = "/path/to/kafka-client.keytab"
  gssapi_kerberos_config_path  = "/etc/krb5.conf"
  gssapi_service_name          = "kafka"
  gssapi_realm                 = "EXAMPLE.COM"
  gssapi_disable_pafx_fast     = false
  
  # TLS can still be enabled with GSSAPI
  tls_enabled = true
  # ca_cert     = file("/path/to/ca.crt")
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
