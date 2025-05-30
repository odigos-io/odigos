package config

import (
	"testing"
)

func TestCauselyUrlFromInput(t *testing.T) {
	type args struct {
		rawUrl string
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
				rawUrl: "http://mediator.causely:4317",
			},
			want:    "http://mediator.causely:4317",
			wantErr: false,
		},
		{
			name: "remove path from url",
			args: args{
				rawUrl: "http://mediator.causely:4317/",
			},
			want:    "http://mediator.causely:4317",
			wantErr: false,
		},
		{
			name: "add http protocol if missing",
			args: args{
				rawUrl: "mediator.causely:4317",
			},
			want:    "http://mediator.causely:4317",
			wantErr: false,
		},
		{
			name: "convert https protocol to http",
			args: args{
				rawUrl: "https://mediator.causely:4317",
			},
			want:    "http://mediator.causely:4317",
			wantErr: false,
		},
		{
			name: "allow only http and https protocols",
			args: args{
				rawUrl: "ftp://mediator.causely:4317",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "add default port if missing",
			args: args{
				rawUrl: "http://mediator.causely",
			},
			want:    "http://mediator.causely:4317",
			wantErr: false,
		},
		{
			name: "allow non standard port",
			args: args{
				rawUrl: "http://mediator.causely:4567",
			},
			want:    "http://mediator.causely:4567",
			wantErr: false,
		},
		{
			name: "remove whitespaces",
			args: args{
				rawUrl: "  http://mediator.causely:4317  ",
			},
			want:    "http://mediator.causely:4317",
			wantErr: false,
		},
		{
			name: "non numeric port",
			args: args{
				rawUrl: "http://mediator.causely:a4317/",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "missing host",
			args: args{
				rawUrl: "http://:4317",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "missing host and port",
			args: args{
				rawUrl: "http://",
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateCauselyUrlInput(tt.args.rawUrl)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCauselyUrlInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("validateCauselyUrlInput() = %v, want %v", got, tt.want)
			}
		})
	}
}
