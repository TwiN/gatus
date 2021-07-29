package client

import (
	"testing"
	"time"
)

func TestGetHTTPClient(t *testing.T) {
	cfg := &Config{
		Insecure:       false,
		IgnoreRedirect: false,
		Timeout:        0,
	}
	cfg.ValidateAndSetDefaults()
	if GetHTTPClient(cfg) == nil {
		t.Error("expected client to not be nil")
	}
	if GetHTTPClient(nil) == nil {
		t.Error("expected client to not be nil")
	}
}

func TestPing(t *testing.T) {
	if success, rtt := Ping("127.0.0.1", &Config{Timeout: 500 * time.Millisecond}); !success {
		t.Error("expected true")
		if rtt == 0 {
			t.Error("Round-trip time returned on success should've higher than 0")
		}
	}
	if success, rtt := Ping("256.256.256.256", &Config{Timeout: 500 * time.Millisecond}); success {
		t.Error("expected false, because the IP is invalid")
		if rtt != 0 {
			t.Error("Round-trip time returned on failure should've been 0")
		}
	}
	if success, rtt := Ping("192.168.152.153", &Config{Timeout: 500 * time.Millisecond}); success {
		t.Error("expected false, because the IP is valid but the host should be unreachable")
		if rtt != 0 {
			t.Error("Round-trip time returned on failure should've been 0")
		}
	}
}

func TestCanPerformStartTLS(t *testing.T) {
	type args struct {
		address  string
		insecure bool
	}
	tests := []struct {
		name          string
		args          args
		wantConnected bool
		wantErr       bool
	}{
		{
			name: "invalid address",
			args: args{
				address: "test",
			},
			wantConnected: false,
			wantErr:       true,
		},
		{
			name: "error dial",
			args: args{
				address: "test:1234",
			},
			wantConnected: false,
			wantErr:       true,
		},
		{
			name: "valid starttls",
			args: args{
				address: "smtp.gmail.com:587",
			},
			wantConnected: true,
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connected, _, err := CanPerformStartTLS(tt.args.address, &Config{Insecure: tt.args.insecure, Timeout: 5 * time.Second})
			if (err != nil) != tt.wantErr {
				t.Errorf("CanPerformStartTLS() err=%v, wantErr=%v", err, tt.wantErr)
				return
			}
			if connected != tt.wantConnected {
				t.Errorf("CanPerformStartTLS() connected=%v, wantConnected=%v", connected, tt.wantConnected)
			}
		})
	}
}

func TestCanCreateTCPConnection(t *testing.T) {
	if CanCreateTCPConnection("127.0.0.1", &Config{Timeout: 5 * time.Second}) {
		t.Error("should've failed, because there's no port in the address")
	}
}
