package interrogation

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssembleTransactions(t *testing.T) {
	edges := []callTrieEdgeRow{
		{TransactionID: "tx-a", NodeID: "n1", Parent: "root", FunctionName: "Pay", FrameType: "jvm", SampleType: "events", Count: 7},
		{TransactionID: "tx-a", NodeID: "n2", Parent: "root", FunctionName: "Encrypt", FrameType: "hotspot", SampleType: "samples", Count: 3},
		{TransactionID: "tx-a", NodeID: "n3", Parent: "n2", FunctionName: "Encrypt", FrameType: "hotspot", SampleType: "samples", Count: 2},
		{TransactionID: "tx-b", NodeID: "n4", Parent: "root", FunctionName: "Cart", FrameType: "jvm", SampleType: "events", Count: 4},
		{TransactionID: "tx-b", NodeID: "n5", Parent: "root", FunctionName: "Cart", FrameType: "hotspot", SampleType: "samples", Count: 4},
	}

	txs := assembleTransactions(edges)
	require.Len(t, txs, 2)

	byID := map[string]Transaction{}
	for _, tx := range txs {
		byID[tx.ID] = tx
	}
	require.Contains(t, byID, "tx-a")
	require.Contains(t, byID, "tx-b")

	assert.Equal(t, int64(3), byID["tx-a"].SeenCount)
	assert.ElementsMatch(t, []Function{
		{Name: "Pay", FrameType: "jvm", SampleType: "events", SeenCount: 7},
		{Name: "Encrypt", FrameType: "hotspot", SampleType: "samples", SeenCount: 5},
	}, byID["tx-a"].Functions)

	assert.Equal(t, int64(4), byID["tx-b"].SeenCount)
	assert.ElementsMatch(t, []Function{
		{Name: "Cart", FrameType: "jvm", SampleType: "events", SeenCount: 4},
		{Name: "Cart", FrameType: "hotspot", SampleType: "samples", SeenCount: 4},
	}, byID["tx-b"].Functions)

	// Sorted by seenCount desc: tx-b(4), tx-a(3)
	assert.Equal(t, "tx-b", txs[0].ID)
	assert.Equal(t, "tx-a", txs[1].ID)
}

func TestCallTrieFromEdgesSharedPrefix(t *testing.T) {
	const (
		nodeB  = "node-b"
		nodeBA = "node-ba"
		nodeBC = "node-bc"
	)
	edges := []callTrieEdgeRow{
		{TransactionID: "tx", NodeID: nodeB, Parent: "root", FunctionName: "b", FrameType: "hotspot", SampleType: "samples", Count: 2},
		{TransactionID: "tx", NodeID: nodeBA, Parent: nodeB, FunctionName: "a", FrameType: "hotspot", SampleType: "samples", Count: 1},
		{TransactionID: "tx", NodeID: nodeBC, Parent: nodeB, FunctionName: "c", FrameType: "hotspot", SampleType: "samples", Count: 1},
	}

	nodes := callTrieFromEdges(edges)
	require.Len(t, nodes, 3)

	byID := map[string]CallTrieNode{}
	for _, n := range nodes {
		byID[n.ID] = n
	}
	require.Contains(t, byID, nodeB)
	assert.Equal(t, "", byID[nodeB].ParentID)
	assert.Equal(t, "b", byID[nodeB].Name)
	assert.Equal(t, int64(2), byID[nodeB].SeenCount)

	require.Contains(t, byID, nodeBA)
	assert.Equal(t, nodeB, byID[nodeBA].ParentID)
	assert.Equal(t, "a", byID[nodeBA].Name)
	assert.Equal(t, int64(1), byID[nodeBA].SeenCount)

	require.Contains(t, byID, nodeBC)
	assert.Equal(t, nodeB, byID[nodeBC].ParentID)
	assert.Equal(t, "c", byID[nodeBC].Name)

	// Sorted by seenCount desc, then name: b(2), a(1), c(1)
	assert.Equal(t, nodeB, nodes[0].ID)
	assert.Equal(t, nodeBA, nodes[1].ID)
	assert.Equal(t, nodeBC, nodes[2].ID)
}

func TestGetTransactionSampleTraceUnavailable(t *testing.T) {
	c := NewClient("", "")
	got, err := c.GetTransactionSampleTrace(context.Background(), "default", "Deployment", "checkout", "app", "tx-1")
	assert.ErrorIs(t, err, ErrUnavailable)
	assert.Empty(t, got)
}

func TestListContainerTransactionsUnavailable(t *testing.T) {
	c := NewClient("", "")
	_, err := c.ListContainerTransactions(context.Background(), "default", "Deployment", "checkout", "app", TransactionsFilter{})
	assert.ErrorIs(t, err, ErrUnavailable)
}

func TestGetTransactionCallTrieUnavailable(t *testing.T) {
	c := NewClient("tcp://localhost:9000", "")
	_, ok, err := c.GetTransactionCallTrie(context.Background(), "default", "Deployment", "checkout", "app", "tx-1")
	assert.ErrorIs(t, err, ErrUnavailable)
	assert.False(t, ok)
}

func TestNewClientEmptyConfig(t *testing.T) {
	c := NewClient("", "secret")
	assert.Nil(t, c.db)
	c = NewClient("tcp://localhost:9000", "")
	assert.Nil(t, c.db)
}

type fakeConn struct {
	driver.Conn
	query string
	args  []any
	rows  []callTrieEdgeRow
}

func (c *fakeConn) Query(_ context.Context, query string, args ...any) (driver.Rows, error) {
	c.query = query
	c.args = args
	return &fakeRows{rows: c.rows, next: -1}, nil
}

type fakeRows struct {
	driver.Rows
	rows []callTrieEdgeRow
	next int
}

func (r *fakeRows) Next() bool {
	r.next++
	return r.next < len(r.rows)
}

func (r *fakeRows) Scan(dest ...any) error {
	row := r.rows[r.next]
	*dest[0].(*string) = row.TransactionID
	*dest[1].(*string) = row.NodeID
	*dest[2].(*string) = row.Parent
	*dest[3].(*string) = row.FunctionName
	*dest[4].(*string) = row.FrameType
	*dest[5].(*string) = row.SampleType
	*dest[6].(*uint64) = row.Count
	return nil
}

func (r *fakeRows) Err() error   { return nil }
func (r *fakeRows) Close() error { return nil }

func TestListContainerTransactionsNoFilter(t *testing.T) {
	conn := &fakeConn{rows: []callTrieEdgeRow{
		{TransactionID: "tx-a", NodeID: "n1", Parent: "root", FunctionName: "Pay", FrameType: "jvm", SampleType: "samples", Count: 2},
	}}
	c := NewClientWithConn(conn)

	got, err := c.ListContainerTransactions(context.Background(), "default", "Deployment", "checkout", "app", TransactionsFilter{})
	require.NoError(t, err)
	require.Len(t, got.Transactions, 1)
	assert.Equal(t, "tx-a", got.Transactions[0].ID)

	assert.Equal(t, []any{"default", "Deployment", "checkout", "app"}, conn.args)
	assert.NotContains(t, conn.query, "TransactionId IN")
	assert.NotContains(t, conn.query, "positionCaseInsensitiveUTF8")
	assert.NotContains(t, conn.query, "LIMIT")
}

func TestListContainerTransactionsFilters(t *testing.T) {
	conn := &fakeConn{}
	c := NewClientWithConn(conn)

	_, err := c.ListContainerTransactions(context.Background(), "default", "Deployment", "checkout", "app", TransactionsFilter{
		IDs:    []string{"tx-a", "tx-b"},
		Search: " Pay ",
	})
	require.NoError(t, err)

	assert.Contains(t, conn.query, "AND TransactionId IN ?")
	assert.Contains(t, conn.query, "AND positionCaseInsensitiveUTF8(TransactionId, ?) > 0")
	assert.NotContains(t, conn.query, "LIMIT")
	assert.Equal(t, []any{
		"default", "Deployment", "checkout", "app",
		clickhouse.GroupSet{Value: []any{"tx-a", "tx-b"}},
		"Pay",
	}, conn.args)
}

func TestListContainerTransactionsPage(t *testing.T) {
	conn := &fakeConn{}
	c := NewClientWithConn(conn)

	_, err := c.ListContainerTransactions(context.Background(), "default", "Deployment", "checkout", "app", TransactionsFilter{
		Search: "pay",
		Limit:  25,
		Offset: 50,
	})
	require.NoError(t, err)

	assert.Contains(t, conn.query, "ORDER BY sumIf(Count, Parent = 'root' AND SampleType = 'samples') DESC, TransactionId ASC")
	assert.Contains(t, conn.query, "LIMIT ? OFFSET ?")
	assert.Equal(t, strings.Count(conn.query, "?"), len(conn.args))
	assert.Equal(t, []any{
		"default", "Deployment", "checkout", "app", "pay",
		"default", "Deployment", "checkout", "app", "pay",
		uint64(25), uint64(50),
	}, conn.args)
}

func TestListContainerTransactionsOffsetWithoutLimit(t *testing.T) {
	conn := &fakeConn{}
	c := NewClientWithConn(conn)

	_, err := c.ListContainerTransactions(context.Background(), "default", "Deployment", "checkout", "app", TransactionsFilter{Offset: 10})
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(conn.args), 2)
	assert.Equal(t, []any{uint64(math.MaxUint64), uint64(10)}, conn.args[len(conn.args)-2:])
}

func TestListContainerTransactionsEmptyIDs(t *testing.T) {
	conn := &fakeConn{}
	c := NewClientWithConn(conn)

	got, err := c.ListContainerTransactions(context.Background(), "default", "Deployment", "checkout", "app", TransactionsFilter{IDs: []string{}})
	require.NoError(t, err)
	assert.Empty(t, got.Transactions)
	assert.Empty(t, conn.query)
}

func TestListContainerTransactionsNegativePage(t *testing.T) {
	c := NewClientWithConn(&fakeConn{})

	_, err := c.ListContainerTransactions(context.Background(), "default", "Deployment", "checkout", "app", TransactionsFilter{Limit: -1})
	assert.Error(t, err)
	_, err = c.ListContainerTransactions(context.Background(), "default", "Deployment", "checkout", "app", TransactionsFilter{Offset: -1})
	assert.Error(t, err)
}
