package config

import "testing"

func TestParseUnencryptedOtlpGrpcUrl(t *testing.T) {
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
			name: "valid url",
			args: args{
				rawURL: "localhost:4317",
			},
			want:    "localhost:4317",
			wantErr: false,
		},
		{
			name: "valid url with grpc scheme",
			args: args{
				rawURL: "grpc://localhost:4317",
			},
			want:    "localhost:4317",
			wantErr: false,
		},
		{
			name: "valid url with http scheme",
			args: args{
				rawURL: "http://localhost:4317",
			},
			want:    "localhost:4317",
			wantErr: false,
		},
		{
			name: "no tls grpcs scheme",
			args: args{
				rawURL: "grpcs://localhost:4317",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "no tls https scheme",
			args: args{
				rawURL: "https://localhost:4317",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "invalid scheme",
			args: args{
				rawURL: "invalid://localhost:4317",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "err with path",
			args: args{
				rawURL: "localhost:4317/v1/traces",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "err with query",
			args: args{
				rawURL: "http://localhost:4317?query=1",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "err with user",
			args: args{
				rawURL: "http://user:pass@localhost:4317",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "err with no host",
			args: args{
				rawURL: ":4317",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "ipv4",
			args: args{
				rawURL: "127.0.0.1:4317",
			},
			want:    "127.0.0.1:4317",
			wantErr: false,
		},
		{
			name: "ipv6",
			args: args{
				rawURL: "[::1]:4317",
			},
			want:    "[::1]:4317",
			wantErr: false,
		},
		{
			name: "add default otlp port when missing",
			args: args{
				rawURL: "localhost",
			},
			want:    "localhost:4317",
			wantErr: false,
		},
		{
			name: "add default otlp port when missing with ipv6",
			args: args{
				rawURL: "[::1]",
			},
			want:    "[::1]:4317",
			wantErr: false,
		},
		{
			name: "host with dots",
			args: args{
				rawURL: "jaeger.tracing:4317",
			},
			want:    "jaeger.tracing:4317",
			wantErr: false,
		},
		{
			name: "non numeric port",
			args: args{
				rawURL: "localhost:port",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "non numeric port with ipv6",
			args: args{
				rawURL: "[::1]:port",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "non default port",
			args: args{
				rawURL: "localhost:1234",
			},
			want:    "localhost:1234",
			wantErr: false,
		},
		{
			name: "whitespaces with scheme",
			args: args{
				rawURL: "  http://localhost:4317  ",
			},
			want:    "localhost:4317",
			wantErr: false,
		},
		{
			name: "whitespaces without scheme",
			args: args{
				rawURL: "  localhost:4317  ",
			},
			want:    "localhost:4317",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOtlpGrpcUrl(tt.args.rawURL, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOtlpGrpcUrl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseOtlpGrpcUrl() = %v, want %v", got, tt.want)
			}
		})
	}
}
