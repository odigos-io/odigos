// Package interrogation reads transaction call-path tries from the
// interrogation ClickHouse database written by the insights service.
package interrogation

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/services"
)

// Must match insights interrogation ClickHouse storage layout.
const (
	clickhouseUser     = "odigos_insights"
	clickhouseDatabase = "interrogation"
	callTrieTable      = "tx_call_trie"
	transactionsTable  = "tx_transactions"
	trieRootParent     = "root"

	clickhouseDialTimeout = 5 * time.Second
	clickhouseReadTimeout = 5 * time.Second
)

var (
	// ErrNotEnabled is returned when interrogation is off in effective config.
	ErrNotEnabled = errors.New("interrogation is not enabled")
	// ErrUnavailable is returned when ClickHouse cannot be reached or is not configured.
	ErrUnavailable = errors.New("interrogation clickhouse is unavailable")
)

// Function is one aggregated profile function for a transaction.
type Function struct {
	Name       string
	FrameType  string
	SampleType string
	// SeenCount is how many times this function was observed for the transaction.
	SeenCount int64
}

// Transaction is one transaction id with its aggregated function set.
type Transaction struct {
	ID string
	// SeenCount is how many times this transaction was observed (sample-root sum).
	SeenCount int64
	Functions []Function
}

// CallTrieNode is one flat node in a transaction call-path trie.
// ParentID is empty for roots (ClickHouse Parent "root").
type CallTrieNode struct {
	ID         string
	ParentID   string
	Name       string
	FrameType  string
	SampleType string
	SeenCount  int64
}

// ContainerTransactions is the result for one workload container.
type ContainerTransactions struct {
	Namespace     string
	Kind          string
	Name          string
	ContainerName string
	Transactions  []Transaction
}

type callTrieEdgeRow struct {
	TransactionID string
	NodeID        string
	Parent        string
	FunctionName  string
	FrameType     string
	SampleType    string
	Count         uint64
}

// TransactionsFilter narrows ListContainerTransactions. The zero value matches
// every transaction of the container.
type TransactionsFilter struct {
	// IDs restricts results to these transaction ids when non-nil. An empty
	// non-nil slice matches nothing.
	IDs []string
	// Search is a case-insensitive substring match on the transaction id.
	Search string
	// Limit caps the number of transactions returned; 0 means no limit.
	Limit int
	// Offset skips this many matching transactions in result order.
	Offset int
}

// Client talks to interrogation ClickHouse (read-only).
type Client struct {
	db             driver.Conn
	callTrieFQN    string
	callTrieSQL    string
	sampleTraceSQL string
}

// NewClient builds a ClickHouse client for the interrogation database.
// endpoint is a tcp:// host:port (or host:port). Empty endpoint or password
// leaves the client unconfigured; reads then return ErrUnavailable. Does not
// ping at construction so UI bootstrap succeeds when ClickHouse is not deployed.
func NewClient(endpoint, password string) *Client {
	endpoint = strings.TrimSpace(endpoint)
	password = strings.TrimSpace(password)
	if endpoint == "" || password == "" {
		return &Client{}
	}
	opts, err := buildClickHouseOptions(endpoint, password)
	if err != nil {
		return &Client{}
	}
	db, err := clickhouse.Open(opts)
	if err != nil {
		return &Client{}
	}
	return newClientWithConn(db)
}

// NewClientWithConn is for tests.
func NewClientWithConn(db driver.Conn) *Client {
	if db == nil {
		return &Client{}
	}
	return newClientWithConn(db)
}

func newClientWithConn(db driver.Conn) *Client {
	fqn := fmt.Sprintf("%q.%q", clickhouseDatabase, callTrieTable)
	txFqn := fmt.Sprintf("%q.%q", clickhouseDatabase, transactionsTable)
	return &Client{
		db:          db,
		callTrieFQN: fqn,
		callTrieSQL: fmt.Sprintf(`
SELECT
	TransactionId,
	NodeId,
	Parent,
	FunctionName,
	FrameType,
	SampleType,
	sum(Count) AS Count
FROM %s
WHERE Namespace = ?
	AND WorkloadKind = ?
	AND WorkloadName = ?
	AND ContainerName = ?
	AND TransactionId = ?
GROUP BY TransactionId, NodeId, Parent, FunctionName, FrameType, SampleType
`, fqn),
		sampleTraceSQL: fmt.Sprintf(`
SELECT SampleTrace
FROM %s
WHERE Namespace = ?
	AND WorkloadKind = ?
	AND WorkloadName = ?
	AND ContainerName = ?
	AND TransactionId = ?
LIMIT 1
`, txFqn),
	}
}

func buildClickHouseOptions(endpoint, password string) (*clickhouse.Options, error) {
	if !strings.Contains(endpoint, "://") {
		endpoint = "tcp://" + endpoint
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse clickhouse endpoint: %w", err)
	}
	q := u.Query()
	if !q.Has("compress") {
		q.Set("compress", "lz4")
	}
	u.RawQuery = q.Encode()
	u.User = url.UserPassword(clickhouseUser, password)

	opts, err := clickhouse.ParseDSN(u.String())
	if err != nil {
		return nil, fmt.Errorf("parse clickhouse dsn: %w", err)
	}
	opts.Auth.Database = clickhouseDatabase
	opts.DialTimeout = clickhouseDialTimeout
	opts.ReadTimeout = clickhouseReadTimeout
	return opts, nil
}

// Close closes the underlying ClickHouse connection.
func (c *Client) Close() error {
	if c == nil || c.db == nil {
		return nil
	}
	return c.db.Close()
}

// IsEnabled reports whether interrogation is enabled in effective config.
func IsEnabled(ctx context.Context, c client.Client) (bool, error) {
	cfg, err := services.GetEffectiveConfig(ctx, c)
	if err != nil {
		return false, err
	}
	return EnabledFromOdigosConfig(cfg), nil
}

// EnabledFromOdigosConfig reports whether interrogation is enabled for a loaded config.
func EnabledFromOdigosConfig(cfg *common.OdigosConfiguration) bool {
	return cfg.InterrogationEnabled()
}

// ListContainerTransactions returns the transactions of the workload container
// that match filter, with aggregated functions from tx_call_trie, ordered by
// seenCount desc, then id.
func (c *Client) ListContainerTransactions(ctx context.Context, namespace, kind, name, containerName string, filter TransactionsFilter) (*ContainerTransactions, error) {
	if c == nil || c.db == nil {
		return nil, ErrUnavailable
	}
	namespace = strings.TrimSpace(namespace)
	kind = strings.TrimSpace(kind)
	name = strings.TrimSpace(name)
	containerName = strings.TrimSpace(containerName)
	if namespace == "" || kind == "" || name == "" || containerName == "" {
		return nil, fmt.Errorf("namespace, kind, name, and containerName are required")
	}
	if filter.Limit < 0 || filter.Offset < 0 {
		return nil, fmt.Errorf("limit and offset must not be negative")
	}

	result := &ContainerTransactions{
		Namespace:     namespace,
		Kind:          kind,
		Name:          name,
		ContainerName: containerName,
		Transactions:  []Transaction{},
	}
	if filter.IDs != nil && len(filter.IDs) == 0 {
		return result, nil
	}

	query, args := buildListEdgesQuery(c.callTrieFQN, namespace, kind, name, containerName, filter)
	rows, err := c.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: query call trie: %v", ErrUnavailable, err)
	}
	defer rows.Close()

	edges, err := scanCallTrieEdges(rows)
	if err != nil {
		return nil, fmt.Errorf("%w: scan call trie: %v", ErrUnavailable, err)
	}

	result.Transactions = assembleTransactions(edges)
	return result, nil
}

// buildListEdgesQuery selects the aggregated call-trie edges of every
// transaction matching filter. Paging runs in a subquery over transaction ids,
// ordered like assembleTransactions, so each page holds whole transactions.
func buildListEdgesQuery(table, namespace, kind, name, containerName string, filter TransactionsFilter) (string, []any) {
	where := `Namespace = ?
	AND WorkloadKind = ?
	AND WorkloadName = ?
	AND ContainerName = ?`
	args := []any{namespace, kind, name, containerName}
	if filter.IDs != nil {
		ids := make([]any, 0, len(filter.IDs))
		for _, id := range filter.IDs {
			ids = append(ids, id)
		}
		where += `
	AND TransactionId IN ?`
		args = append(args, clickhouse.GroupSet{Value: ids})
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		where += `
	AND positionCaseInsensitiveUTF8(TransactionId, ?) > 0`
		args = append(args, search)
	}

	if filter.Limit > 0 || filter.Offset > 0 {
		limit := uint64(math.MaxUint64)
		if filter.Limit > 0 {
			limit = uint64(filter.Limit)
		}
		pageArgs := append(slices.Clone(args), limit, uint64(filter.Offset))
		where += fmt.Sprintf(`
	AND TransactionId IN (
		SELECT TransactionId
		FROM %s
		WHERE %s
		GROUP BY TransactionId
		ORDER BY sumIf(Count, Parent = '%s' AND SampleType = 'samples') DESC, TransactionId ASC
		LIMIT ? OFFSET ?
	)`, table, where, trieRootParent)
		args = append(args, pageArgs...)
	}

	return fmt.Sprintf(`
SELECT
	TransactionId,
	NodeId,
	Parent,
	FunctionName,
	FrameType,
	SampleType,
	sum(Count) AS Count
FROM %s
WHERE %s
GROUP BY TransactionId, NodeId, Parent, FunctionName, FrameType, SampleType
`, table, where), args
}

// GetTransactionSampleTrace returns the stored OTLP JSON sample for a
// transaction (service-invocation ResourceSpans only). Empty when none stored.
func (c *Client) GetTransactionSampleTrace(ctx context.Context, namespace, kind, name, containerName, txID string) (string, error) {
	if c == nil || c.db == nil {
		return "", ErrUnavailable
	}
	namespace = strings.TrimSpace(namespace)
	kind = strings.TrimSpace(kind)
	name = strings.TrimSpace(name)
	containerName = strings.TrimSpace(containerName)
	txID = strings.TrimSpace(txID)
	if namespace == "" || kind == "" || name == "" || containerName == "" || txID == "" {
		return "", fmt.Errorf("namespace, kind, name, containerName, and transactionId are required")
	}

	rows, err := c.db.Query(ctx, c.sampleTraceSQL, namespace, kind, name, containerName, txID)
	if err != nil {
		return "", fmt.Errorf("%w: query sample trace: %v", ErrUnavailable, err)
	}
	defer rows.Close()

	if !rows.Next() {
		return "", rows.Err()
	}
	var sample string
	if err := rows.Scan(&sample); err != nil {
		return "", fmt.Errorf("%w: scan sample trace: %v", ErrUnavailable, err)
	}
	return sample, nil
}

// GetTransactionCallTrie returns flat call-path trie nodes for a transaction.
// ok is false when no edges exist (caller should treat as null).
func (c *Client) GetTransactionCallTrie(ctx context.Context, namespace, kind, name, containerName, txID string) (nodes []CallTrieNode, ok bool, err error) {
	if c == nil || c.db == nil {
		return nil, false, ErrUnavailable
	}
	namespace = strings.TrimSpace(namespace)
	kind = strings.TrimSpace(kind)
	name = strings.TrimSpace(name)
	containerName = strings.TrimSpace(containerName)
	txID = strings.TrimSpace(txID)
	if namespace == "" || kind == "" || name == "" || containerName == "" || txID == "" {
		return nil, false, fmt.Errorf("namespace, kind, name, containerName, and transactionId are required")
	}

	rows, err := c.db.Query(ctx, c.callTrieSQL, namespace, kind, name, containerName, txID)
	if err != nil {
		return nil, false, fmt.Errorf("%w: query call trie: %v", ErrUnavailable, err)
	}
	defer rows.Close()

	edges, err := scanCallTrieEdges(rows)
	if err != nil {
		return nil, false, fmt.Errorf("%w: scan call trie: %v", ErrUnavailable, err)
	}
	if len(edges) == 0 {
		return nil, false, nil
	}
	return callTrieFromEdges(edges), true, nil
}

func scanCallTrieEdges(rows driver.Rows) ([]callTrieEdgeRow, error) {
	var out []callTrieEdgeRow
	for rows.Next() {
		var r callTrieEdgeRow
		if err := rows.Scan(
			&r.TransactionID,
			&r.NodeID,
			&r.Parent,
			&r.FunctionName,
			&r.FrameType,
			&r.SampleType,
			&r.Count,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
