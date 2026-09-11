package testconnection

import (
	"testing"

	"github.com/odigos-io/odigos/common/config"
	"github.com/stretchr/testify/assert"
)

func TestReplacePlaceholders(t *testing.T) {
	fields := map[string]string{
		"MY_KEY1": "MY_VALUE1",
		"MY_KEY2": "MY_VALUE2",
	}
	destID := "odigos.io.dest.otlp-test"

	gmap := config.GenericMap{
		"key1": "${MY_KEY1}",
		"key2": 123,
		"key3": config.GenericMap{
			"nestedKey1": "${MY_KEY2}",
			"nestedKey2": "someValue",
			"nestedKey3": "${MY_KEY3}",
			"nestedKey4": "some prefix: ${MY_KEY2}",
		},
	}

	replacePlaceholders(gmap, fields, destID)
	assert.Equal(t, "MY_VALUE1", gmap["key1"])
	assert.Equal(t, config.GenericMap{
		"nestedKey1": "MY_VALUE2",
		"nestedKey2": "someValue",
		"nestedKey3": "${MY_KEY3}",
		"nestedKey4": "some prefix: MY_VALUE2",
	}, gmap["key3"])
	assert.Equal(t, 123, gmap["key2"])

	gmap = config.GenericMap{
		"key1": "value1",
		"key2": 123,
		"key3": config.GenericMap{
			"nestedKey1": "value2",
			"nestedKey2": "someValue",
		},
	}

	replacePlaceholders(gmap, fields, destID)
	assert.Equal(t, "value1", gmap["key1"])
	assert.Equal(t, config.GenericMap{
		"nestedKey1": "value2",
		"nestedKey2": "someValue",
	}, gmap["key3"])
	assert.Equal(t, 123, gmap["key2"])
}

func TestReplacePlaceholders_ScopedSecretEnv(t *testing.T) {
	fields := map[string]string{
		"OTLP_HTTP_CLIENT_KEY_PEM": "pem-a",
	}
	destID := "odigos.io.dest.otlphttp-aaaa"
	placeholder := "${" + config.SecretEnvVarName("OTLP_HTTP_CLIENT_KEY_PEM", destID) + "}"

	gmap := config.GenericMap{
		"key_pem": placeholder,
	}
	replacePlaceholders(gmap, fields, destID)
	assert.Equal(t, "pem-a", gmap["key_pem"])
}
