package interrogation

import (
	"testing"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCallTrieToModelClassifiesFrames(t *testing.T) {
	nodes := CallTrieToModel([]CallTrieNode{
		{ID: "n1", Name: "void com.rowan.notify.Controller.create(java.lang.String)", FrameType: "jvm"},
		{ID: "n2", ParentID: "n1", Name: "java.util.concurrent.CompletableFuture org.springframework.kafka.core.KafkaTemplate.send(java.lang.String, java.lang.Object)", FrameType: "jvm"},
		{ID: "n3", ParentID: "n2", Name: "void org.springframework.beans.factory.support.AbstractBeanFactory.getBean(java.lang.String)", FrameType: "hotspot"},
		{ID: "n4", ParentID: "n3", Name: "void com.rowan.notify.RenderMessageProducer.publish(java.lang.String)", FrameType: "jvm"},
		{ID: "n5", Name: "net/http.(*conn).serve", FrameType: "go"},
	})
	require.Len(t, nodes, 5)

	assert.Equal(t, model.InterrogationFrameKindApp, nodes[0].FrameKind)

	assert.Equal(t, model.InterrogationFrameKindLibrary, nodes[1].FrameKind)

	assert.Equal(t, model.InterrogationFrameKindLibrary, nodes[2].FrameKind)

	assert.Equal(t, model.InterrogationFrameKindApp, nodes[3].FrameKind)

	// Non-JVM frames are not classified.
	assert.Equal(t, model.InterrogationFrameKindApp, nodes[4].FrameKind)
}

func TestTransactionsToModelClassifiesFunctions(t *testing.T) {
	txs := TransactionsToModel([]Transaction{{
		ID: "tx-a",
		Functions: []Function{
			{Name: "java.sql.ResultSet org.postgresql.jdbc.PgPreparedStatement.executeQuery()", FrameType: "jvm"},
			{Name: "void com.rowan.account.AccountLookup.loadById(java.lang.String)", FrameType: "jvm"},
		},
	}})
	require.Len(t, txs, 1)
	fns := txs[0].Functions
	require.Len(t, fns, 2)
	assert.Equal(t, model.InterrogationFrameKindLibrary, fns[0].FrameKind)
	assert.Equal(t, model.InterrogationFrameKindApp, fns[1].FrameKind)
}
