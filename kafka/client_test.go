package kafka

import "testing"

func Test_NewClient(t *testing.T) {
	config := &Config{}
	_, err := NewClient(config)
	if err == nil {
		t.Errorf("Expected error, got none")
	}
}

// Test_ClientAPIVersion has been removed as manual API version detection
// is no longer needed with Sarama 1.46+, which handles API version
// negotiation automatically using ApiVersionRequest.
