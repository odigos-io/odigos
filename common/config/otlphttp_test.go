package config

import "testing"

func TestParseOtlpHttpEndpoint(t *testing.T) {
	type args struct {
		rawURL string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "valid url with http scheme",
			args: args{
				rawURL: "http://localhost:4318",
			},
			want:    "http://localhost:4318",
			wantErr: false,
		},
		{
			name: "valid url with https scheme",
			args: args{
				rawURL: "https://localhost:4318",
			},
			want:    "https://localhost:4318",
			wantErr: false,
		},
		{
			name: "invalid scheme",
			args: args{
				rawURL: "invalid://localhost:4318",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "path allowed",
			args: args{
				rawURL: "http://localhost:4318/some-path",
			},
			want:    "http://localhost:4318/some-path",
			wantErr: false,
		},
		{
			name: "ipv4",
			args: args{
				rawURL: "http://127.0.0.1:4318",
			},
			want:    "http://127.0.0.1:4318",
			wantErr: false,
		},
		{
			name: "ipv6",
			args: args{
				rawURL: "http://[::1]:4318",
			},
			want:    "http://[::1]:4318",
			wantErr: false,
		},
		{
			name: "do not add port when missing",
			args: args{
				rawURL: "http://localhost",
			},
			want:    "http://localhost",
			wantErr: false,
		},
		{
			name: "do not add port when missing with ipv6",
			args: args{
				rawURL: "http://[::1]",
			},
			want:    "http://[::1]",
			wantErr: false,
		},
		{
			name: "host with dots",
			args: args{
				rawURL: "http://jaeger.tracing:4318",
			},
			want:    "http://jaeger.tracing:4318",
			wantErr: false,
		},
		{
			name: "non numeric port",
			args: args{
				rawURL: "http://localhost:port",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "non numeric port with ipv6",
			args: args{
				rawURL: "http://[::1]:port",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "non default port",
			args: args{
				rawURL: "http://localhost:1234",
			},
			want:    "http://localhost:1234",
			wantErr: false,
		},
		{
			name: "whitespaces",
			args: args{
				rawURL: "  http://localhost:4318  ",
			},
			want:    "http://localhost:4318",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOtlpHttpEndpoint(tt.args.rawURL, "", "")
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOtlpHttpEndpoint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseOtlpHttpEndpoint() = %v, want %v", got, tt.want)
			}
		})
	}
}
