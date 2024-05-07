package config

import (
	"reflect"
	"testing"

	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
)

func TestLokiUrlFromInput(t *testing.T) {
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
				rawUrl: "http://localhost:3100/loki/api/v1/push",
			},
			want:    "http://localhost:3100/loki/api/v1/push",
			wantErr: false,
		},
		{
			name: "add http scheme if missing",
			args: args{
				rawUrl: "localhost:3100/loki/api/v1/push",
			},
			want:    "http://localhost:3100/loki/api/v1/push",
			wantErr: false,
		},
		{
			name: "allow https scheme",
			args: args{
				rawUrl: "https://localhost:3100/loki/api/v1/push",
			},
			want:    "https://localhost:3100/loki/api/v1/push",
			wantErr: false,
		},
		{
			name: "allow only http and https schemes",
			args: args{
				rawUrl: "ftp://localhost:3100/loki/api/v1/push",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "add default path if missing",
			args: args{
				rawUrl: "http://localhost:3100",
			},
			want:    "http://localhost:3100/loki/api/v1/push",
			wantErr: false,
		},
		{
			name: "add default port if missing",
			args: args{
				rawUrl: "http://localhost/loki/api/v1/push",
			},
			want:    "http://localhost:3100/loki/api/v1/push",
			wantErr: false,
		},
		{
			name: "allow non standard path",
			args: args{
				rawUrl: "http://localhost:3100/foo",
			},
			want:    "http://localhost:3100/foo",
			wantErr: false,
		},
		{
			name: "allow non standard port",
			args: args{
				rawUrl: "http://localhost:1234/loki/api/v1/push",
			},
			want:    "http://localhost:1234/loki/api/v1/push",
			wantErr: false,
		},
		{
			name: "remove whitespaces",
			args: args{
				rawUrl: "  http://localhost:3100/loki/api/v1/push  ",
			},
			want:    "http://localhost:3100/loki/api/v1/push",
			wantErr: false,
		},
		{
			name: "non numeric port",
			args: args{
				rawUrl: "http://localhost:port/loki/api/v1/push",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "ipv6",
			args: args{
				rawUrl: "[::1]:3100/loki/api/v1/push",
			},
			want:    "http://[::1]:3100/loki/api/v1/push",
			wantErr: false,
		},
		{
			name: "ipv4",
			args: args{
				rawUrl: "http://127.0.0.1:3100/loki/api/v1/push",
			},
			want:    "http://127.0.0.1:3100/loki/api/v1/push",
			wantErr: false,
		},
		{
			name: "add default loki port ipv6",
			args: args{
				rawUrl: "[::1]",
			},
			want:    "http://[::1]:3100/loki/api/v1/push",
			wantErr: false,
		},
		{
			name: "host with dots",
			args: args{
				rawUrl: "loki.loki:3100/loki/api/v1/push",
			},
			want:    "http://loki.loki:3100/loki/api/v1/push",
			wantErr: false,
		},
		{
			name: "missing host",
			args: args{
				rawUrl: "http://",
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := lokiUrlFromInput(tt.args.rawUrl)
			if (err != nil) != tt.wantErr {
				t.Errorf("lokiUrlFromInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("lokiUrlFromInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLokiLabelsProcessors(t *testing.T) {
	type args struct {
		rawLokiLabels string
		exists        bool
		destName      string
	}
	tests := []struct {
		name    string
		args    args
		want    commonconf.GenericMap
		wantErr bool
	}{
		{
			name: "valid labels",
			args: args{
				rawLokiLabels: `["key1","key2"]`,
				exists:        true,
				destName:      "foo",
			},
			want: commonconf.GenericMap{
				"attributes/loki-foo": commonconf.GenericMap{
					"actions": []commonconf.GenericMap{
						{
							"key":    "loki.attribute.labels",
							"action": "insert",
							"value":  "key1, key2",
						},
					},
				},
				"resource/loki-foo": commonconf.GenericMap{
					"attributes": []commonconf.GenericMap{
						{
							"key":    "loki.resource.labels",
							"action": "insert",
							"value":  "key1, key2",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "single label",
			args: args{
				rawLokiLabels: `["key1"]`,
				exists:        true,
				destName:      "foo",
			},
			want: commonconf.GenericMap{
				"attributes/loki-foo": commonconf.GenericMap{
					"actions": []commonconf.GenericMap{
						{
							"key":    "loki.attribute.labels",
							"action": "insert",
							"value":  "key1",
						},
					},
				},
				"resource/loki-foo": commonconf.GenericMap{
					"attributes": []commonconf.GenericMap{
						{
							"key":    "loki.resource.labels",
							"action": "insert",
							"value":  "key1",
						},
					},
				},
			},
		},
		{
			name: "no labels",
			args: args{
				rawLokiLabels: "[]",
				exists:        true,
				destName:      "foo",
			},
			want:    commonconf.GenericMap{},
			wantErr: false,
		},
		{
			name: "no labels, not exists",
			args: args{
				rawLokiLabels: "",
				exists:        false,
				destName:      "foo",
			},
			want: commonconf.GenericMap{
				"attributes/loki-foo": commonconf.GenericMap{
					"actions": []commonconf.GenericMap{
						{
							"key":    "loki.attribute.labels",
							"action": "insert",
							"value":  "k8s.container.name, k8s.pod.name, k8s.namespace.name",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid json",
			args: args{
				rawLokiLabels: "invalid",
				exists:        true,
				destName:      "foo",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := lokiLabelsProcessors(tt.args.rawLokiLabels, tt.args.exists, tt.args.destName)
			if (err != nil) != tt.wantErr {
				t.Errorf("lokiLabelsProcessors() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("lokiLabelsProcessors() = %v, want %v", got, tt.want)
			}
		})
	}
}
