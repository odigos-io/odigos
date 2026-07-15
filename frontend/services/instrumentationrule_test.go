package services

import (
	"testing"

	apirules "github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
)

func TestMergePayloadCollectionUpdatePreservesOmittedAdvancedOptions(t *testing.T) {
	maxHTTP := int64(2048)
	dropHTTP := true
	maxDb := int64(512)
	dropDb := false
	maxMessaging := int64(1024)
	dropMessaging := true
	mimeTypes := []string{"application/json", "text/plain"}

	existing := &apirules.PayloadCollection{
		HttpRequest: &apirules.HttpPayloadCollection{
			MimeTypes:           &mimeTypes,
			MaxPayloadLength:    &maxHTTP,
			DropPartialPayloads: &dropHTTP,
		},
		DbQuery: &apirules.DbQueryPayloadCollection{
			MaxPayloadLength:    &maxDb,
			DropPartialPayloads: &dropDb,
		},
		Messaging: &apirules.MessagingPayloadCollection{
			MaxPayloadLength:    &maxMessaging,
			DropPartialPayloads: &dropMessaging,
		},
	}

	out := mergePayloadCollectionUpdate(existing, &model.PayloadCollectionInput{
		HTTPRequest: &model.HTTPPayloadCollectionInput{},
		DbQuery:     &model.DbQueryPayloadCollectionInput{},
		Messaging:   &model.MessagingPayloadCollectionInput{},
	})

	require.NotNil(t, out.HttpRequest)
	require.Equal(t, []string{"application/json", "text/plain"}, *out.HttpRequest.MimeTypes)
	require.Equal(t, int64(2048), *out.HttpRequest.MaxPayloadLength)
	require.True(t, *out.HttpRequest.DropPartialPayloads)
	require.NotSame(t, existing.HttpRequest.MimeTypes, out.HttpRequest.MimeTypes)

	require.NotNil(t, out.DbQuery)
	require.Equal(t, int64(512), *out.DbQuery.MaxPayloadLength)
	require.False(t, *out.DbQuery.DropPartialPayloads)

	require.NotNil(t, out.Messaging)
	require.Equal(t, int64(1024), *out.Messaging.MaxPayloadLength)
	require.True(t, *out.Messaging.DropPartialPayloads)
}

func TestMergePayloadCollectionUpdateReplacesExplicitAdvancedOptions(t *testing.T) {
	oldMax := int64(2048)
	oldDrop := true
	oldMimeTypes := []string{"application/json"}
	newMax := 4096
	newDrop := false

	existing := &apirules.PayloadCollection{
		HttpRequest: &apirules.HttpPayloadCollection{
			MimeTypes:           &oldMimeTypes,
			MaxPayloadLength:    &oldMax,
			DropPartialPayloads: &oldDrop,
		},
		HttpResponse: &apirules.HttpPayloadCollection{
			MimeTypes:           &oldMimeTypes,
			MaxPayloadLength:    &oldMax,
			DropPartialPayloads: &oldDrop,
		},
	}

	out := mergePayloadCollectionUpdate(existing, &model.PayloadCollectionInput{
		HTTPRequest: &model.HTTPPayloadCollectionInput{
			MimeTypes:           []*string{},
			MaxPayloadLength:    &newMax,
			DropPartialPayloads: &newDrop,
		},
	})

	require.NotNil(t, out.HttpRequest)
	require.Empty(t, *out.HttpRequest.MimeTypes)
	require.Equal(t, int64(4096), *out.HttpRequest.MaxPayloadLength)
	require.False(t, *out.HttpRequest.DropPartialPayloads)
	require.Nil(t, out.HttpResponse, "omitted payload sections should still be disabled")
}
