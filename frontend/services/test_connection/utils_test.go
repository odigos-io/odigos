package testconnection

import (
	"testing"

	"github.com/odigos-io/odigos/common/config"
	"github.com/stretchr/testify/assert"
)

func TestReplacePlaceholders(t *testing.T) {
	// Fields map with replacements
	fields := map[string]string{
		"MY_KEY1": "MY_VALUE1",
		"MY_KEY2": "MY_VALUE2",
	}

	gmap := config.GenericMap{
		"key1": "${MY_KEY1}",
		"key2": 123,
		"key3": config.GenericMap{
			"nestedKey1": "${MY_KEY2}",              // replaced with MY_VALUE2
			"nestedKey2": "someValue",               // no replacement
			"nestedKey3": "${MY_KEY3}",              // no replacement
			"nestedKey4": "some prefix: ${MY_KEY2}", // replaced with "some prefix: MY_VALUE2"
		},
	}

	replacePlaceholders(gmap, fields)
	assert.Equal(t, "MY_VALUE1", gmap["key1"])
	assert.Equal(t, config.GenericMap{
		"nestedKey1": "MY_VALUE2",
		"nestedKey2": "someValue",
		"nestedKey3": "${MY_KEY3}",
		"nestedKey4": "some prefix: MY_VALUE2",
	}, gmap["key3"])
	assert.Equal(t, 123, gmap["key2"])

	// don't change the original map if no placeholders are found
	gmap = config.GenericMap{
		"key1": "value1",
		"key2": 123,
		"key3": config.GenericMap{
			"nestedKey1": "value2",
			"nestedKey2": "someValue",
		},
	}

	replacePlaceholders(gmap, fields)
	assert.Equal(t, "value1", gmap["key1"])
	assert.Equal(t, config.GenericMap{
		"nestedKey1": "value2",
		"nestedKey2": "someValue",
	}, gmap["key3"])
	assert.Equal(t, 123, gmap["key2"])

}
