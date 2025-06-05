package feature

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/version"
)

var fs1, fs2 *featureSupport

func setup() {
	fs1 = &featureSupport{
		alphaVersion: "1.10.0",
		betaVersion:  "1.11.0",
		gaVersion:    "1.15.0",
	}

	fs2 = &featureSupport{
		alphaVersion: "1.21.0",
		betaVersion:  "1.24.0",
		gaVersion:    "1.25.0",
	}
}

func TestFeatureSupport(t *testing.T) {
	t.Run("v1.12.0", func(t *testing.T) {
		setup()
		k8sVersion = version.MustParse("1.12.0")
		t.Cleanup(func() {
			k8sVersion = nil
		})

		assert.True(t, fs1.isEnabled(Alpha))
		assert.True(t, fs1.isEnabled(Beta))
		assert.False(t, fs1.isEnabled(GA))

		assert.False(t, fs2.isEnabled(Alpha))
		assert.False(t, fs2.isEnabled(Beta))
		assert.False(t, fs2.isEnabled(GA))
	})

	t.Run("v1.23.17-eks-ce1d5eb", func(t *testing.T) {
		setup()
		k8sVersion = version.MustParse("v1.23.17-eks-ce1d5eb")
		t.Cleanup(func() {
			k8sVersion = nil
		})

		assert.True(t, fs1.isEnabled(Alpha))
		assert.True(t, fs1.isEnabled(Beta))
		assert.True(t, fs1.isEnabled(GA))

		assert.True(t, fs2.isEnabled(Alpha))
		assert.False(t, fs2.isEnabled(Beta))
		assert.False(t, fs2.isEnabled(GA))
	})

	t.Run("unknown version", func(t *testing.T) {
		setup()
		k8sVersion = nil

		assert.False(t, fs1.isEnabled(Alpha))
		assert.False(t, fs1.isEnabled(Beta))
		assert.False(t, fs1.isEnabled(GA))

		assert.False(t, fs2.isEnabled(Alpha))
		assert.False(t, fs2.isEnabled(Beta))
		assert.False(t, fs2.isEnabled(GA))
	})
}
