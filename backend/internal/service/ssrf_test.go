package service

import (
	"context"
	"net"
	"testing"
)

func TestIsPublicIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{ip: "8.8.8.8", want: true},
		{ip: "1.1.1.1", want: true},
		{ip: "2606:4700:4700::1111", want: true},
		{ip: "127.0.0.1", want: false},
		{ip: "10.0.0.1", want: false},
		{ip: "172.16.0.1", want: false},
		{ip: "192.168.1.1", want: false},
		{ip: "169.254.169.254", want: false},
		{ip: "100.64.0.1", want: false},
		{ip: "::1", want: false},
		{ip: "fc00::1", want: false},
		{ip: "fe80::1", want: false},
		{ip: "2001:db8::1", want: false},
		{ip: "224.0.0.1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			got := isPublicIP(net.ParseIP(tt.ip))

			if got != tt.want {
				t.Errorf("isPublicIP(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestSafeDialContextRejectsPrivateIP(t *testing.T) {
	_, err := safeDialContext(
		context.Background(),
		"tcp",
		"127.0.0.1:80",
	)

	if err == nil {
		t.Fatal("expected connection to private IP to be rejected")
	}
}

func TestSafeDialContextRejectsUnsupportedPort(t *testing.T) {
	_, err := safeDialContext(
		context.Background(),
		"tcp",
		"8.8.8.8:8080",
	)

	if err == nil {
		t.Fatal("expected unsupported port to be rejected")
	}
}
