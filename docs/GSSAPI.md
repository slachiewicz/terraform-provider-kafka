# GSSAPI (Kerberos) Authentication

This document describes how to configure the Terraform Kafka provider to use GSSAPI (Kerberos) authentication.

## Overview

GSSAPI is a standardized interface for Kerberos authentication. This provider supports three authentication types:

1. **KRB5_KEYTAB_AUTH** (Type 2) - Keytab-based authentication (default)
2. **KRB5_USER_AUTH** (Type 1) - Username/password authentication
3. **KRB5_CCACHE_AUTH** (Type 3) - Credential cache authentication

## Configuration Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `sasl_mechanism` | string | Yes | - | Must be set to `"gssapi"` |
| `gssapi_auth_type` | string | No | `"KRB5_KEYTAB_AUTH"` | Authentication type: `"KRB5_USER_AUTH"` (1), `"KRB5_KEYTAB_AUTH"` (2), or `"KRB5_CCACHE_AUTH"` (3) |
| `gssapi_username` | string | Yes | - | Kerberos principal name (e.g., `kafka-client@EXAMPLE.COM`) |
| `gssapi_keytab_path` | string | Conditional | - | Path to keytab file (required for `KRB5_KEYTAB_AUTH`) |
| `gssapi_password` | string | Conditional | - | Password (required for `KRB5_USER_AUTH`) |
| `gssapi_kerberos_config_path` | string | Yes | - | Path to Kerberos config file (e.g., `/etc/krb5.conf`) |
| `gssapi_service_name` | string | No | `"kafka"` | Service name for Kerberos |
| `gssapi_realm` | string | No | - | Kerberos realm |
| `gssapi_disable_pafx_fast` | bool | No | `false` | Disable PA-FX-FAST |

## Environment Variables

All GSSAPI parameters can also be set via environment variables:

- `KAFKA_SASL_MECHANISM`
- `KAFKA_GSSAPI_AUTH_TYPE`
- `KAFKA_GSSAPI_USERNAME`
- `KAFKA_GSSAPI_KEYTAB_PATH`
- `KAFKA_GSSAPI_PASSWORD`
- `KAFKA_GSSAPI_KERBEROS_CONFIG_PATH`
- `KAFKA_GSSAPI_SERVICE_NAME`
- `KAFKA_GSSAPI_REALM`
- `KAFKA_GSSAPI_DISABLE_PAFX_FAST`

## Examples

### Keytab Authentication (Recommended for Production)

```hcl
provider "kafka" {
  bootstrap_servers = ["kafka.example.com:9092"]
  
  sasl_mechanism               = "gssapi"
  gssapi_auth_type             = "KRB5_KEYTAB_AUTH"
  gssapi_username              = "kafka-client@EXAMPLE.COM"
  gssapi_keytab_path           = "/path/to/kafka-client.keytab"
  gssapi_kerberos_config_path  = "/etc/krb5.conf"
  gssapi_service_name          = "kafka"
  gssapi_realm                 = "EXAMPLE.COM"
  
  tls_enabled = true
}
```

### Username/Password Authentication

```hcl
provider "kafka" {
  bootstrap_servers = ["kafka.example.com:9092"]
  
  sasl_mechanism               = "gssapi"
  gssapi_auth_type             = "KRB5_USER_AUTH"
  gssapi_username              = "kafka-client@EXAMPLE.COM"
  gssapi_password              = var.kerberos_password
  gssapi_kerberos_config_path  = "/etc/krb5.conf"
  gssapi_service_name          = "kafka"
  gssapi_realm                 = "EXAMPLE.COM"
  
  tls_enabled = true
}

variable "kerberos_password" {
  description = "Kerberos password"
  type        = string
  sensitive   = true
}
```

### Using Environment Variables

```bash
export KAFKA_SASL_MECHANISM="gssapi"
export KAFKA_GSSAPI_AUTH_TYPE="KRB5_KEYTAB_AUTH"
export KAFKA_GSSAPI_USERNAME="kafka-client@EXAMPLE.COM"
export KAFKA_GSSAPI_KEYTAB_PATH="/path/to/kafka-client.keytab"
export KAFKA_GSSAPI_KERBEROS_CONFIG_PATH="/etc/krb5.conf"
export KAFKA_GSSAPI_SERVICE_NAME="kafka"
export KAFKA_GSSAPI_REALM="EXAMPLE.COM"
```

```hcl
provider "kafka" {
  bootstrap_servers = ["kafka.example.com:9092"]
  # GSSAPI config will be read from environment variables
  tls_enabled = true
}
```

## Prerequisites

1. **Kerberos Configuration File**: Ensure `/etc/krb5.conf` (or your custom path) is properly configured with your realm information
2. **Keytab File** (for keytab auth): Generate and secure the keytab file
3. **Network Access**: Ensure network connectivity to both Kafka brokers and Kerberos KDC
4. **Time Synchronization**: Kerberos requires synchronized clocks (typically via NTP)

## Troubleshooting

### Common Issues

1. **Clock Skew**: Kerberos is sensitive to time differences. Ensure all systems have synchronized clocks.
2. **Keytab Permissions**: The keytab file should have restricted permissions (e.g., `chmod 600`).
3. **Service Principal Name**: Verify that the service name matches your Kafka broker configuration.
4. **Realm Case Sensitivity**: Realm names are case-sensitive and typically uppercase.

### Debug Logging

Enable debug logging in the provider configuration to troubleshoot authentication issues:

```bash
export TF_LOG=DEBUG
terraform plan
```

## Security Best Practices

1. **Use Keytab Authentication**: Preferred for automation and production environments
2. **Secure Keytab Files**: Restrict file permissions and store securely
3. **Sensitive Variables**: Use Terraform variables with `sensitive = true` for passwords
4. **Enable TLS**: Always use TLS encryption (`tls_enabled = true`) with GSSAPI
5. **Rotate Credentials**: Regularly rotate Kerberos credentials and keytabs

## References

- [Confluent GSSAPI/Kerberos Documentation](https://docs.confluent.io/platform/current/kafka/authentication_sasl/authentication_sasl_gssapi.html)
- [Apache Kafka Security Documentation](https://kafka.apache.org/documentation/#security)
- [IBM Sarama GSSAPI Implementation](https://github.com/IBM/sarama)
