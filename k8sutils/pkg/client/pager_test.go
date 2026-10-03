package client

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// pagedLister serves a fixed sequence of pages and records the ListOptions it was called with.
type pagedLister struct {
	pages     []*corev1.PodList
	err       error
	errOnCall int
	calls     []metav1.ListOptions
}

func (p *pagedLister) list(_ context.Context, opts metav1.ListOptions) (*corev1.PodList, error) {
	p.calls = append(p.calls, opts)
	if p.err != nil && len(p.calls) == p.errOnCall {
		return nil, p.err
	}
	page := p.pages[len(p.calls)-1]
	return page, nil
}

// podPage builds one list page carrying a single pod named podName; continueToken is the token
// the apiserver would hand back to fetch the next page ("" means this is the last page).
func podPage(podName, continueToken string) *corev1.PodList {
	return &corev1.PodList{
		ListMeta: metav1.ListMeta{Continue: continueToken},
		Items:    []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: podName}}},
	}
}

func collectPodNames(seen *[]string) func(*corev1.PodList) error {
	return func(list *corev1.PodList) error {
		for _, pod := range list.Items {
			*seen = append(*seen, pod.Name)
		}
		return nil
	}
}

func TestListWithPagesWalksEveryPage(t *testing.T) {
	lister := &pagedLister{pages: []*corev1.PodList{
		podPage("first", "token-after-first"),
		podPage("second", "token-after-second"),
		podPage("third", ""),
	}}

	var seen []string
	err := ListWithPages(7, lister.list, context.Background(), &metav1.ListOptions{}, collectPodNames(&seen))

	require.NoError(t, err)
	assert.Equal(t, []string{"first", "second", "third"}, seen)
	require.Len(t, lister.calls, 3)
	// Each request must resume from the token the previous page returned, or the walk either
	// restarts from the beginning (duplicates) or skips ahead (missing workloads in the UI).
	assert.Equal(t, "", lister.calls[0].Continue)
	assert.Equal(t, "token-after-first", lister.calls[1].Continue)
	assert.Equal(t, "token-after-second", lister.calls[2].Continue)
}

func TestListWithPagesSendsThePageSizeAsTheLimit(t *testing.T) {
	lister := &pagedLister{pages: []*corev1.PodList{podPage("only", "")}}

	var seen []string
	err := ListWithPages(DefaultPageSize, lister.list, context.Background(), &metav1.ListOptions{}, collectPodNames(&seen))

	require.NoError(t, err)
	require.Len(t, lister.calls, 1)
	assert.Equal(t, int64(DefaultPageSize), lister.calls[0].Limit)
}

// DefaultPageSize is a tuning knob, so its exact value is deliberately not pinned - but a
// non-positive default would ask the apiserver for an unbounded list on every call site.
func TestDefaultPageSizeIsPositive(t *testing.T) {
	assert.Positive(t, DefaultPageSize)
}

func TestListWithPagesKeepsTheCallersSelectorsOnEveryPage(t *testing.T) {
	lister := &pagedLister{pages: []*corev1.PodList{
		podPage("first", "token-after-first"),
		podPage("second", ""),
	}}

	var seen []string
	err := ListWithPages(3, lister.list, context.Background(), &metav1.ListOptions{
		LabelSelector: "app=odigos",
		FieldSelector: "metadata.namespace=odigos-system",
	}, collectPodNames(&seen))

	require.NoError(t, err)
	require.Len(t, lister.calls, 2)
	for i, call := range lister.calls {
		assert.Equal(t, "app=odigos", call.LabelSelector, "label selector dropped on page %d", i+1)
		assert.Equal(t, "metadata.namespace=odigos-system", call.FieldSelector, "field selector dropped on page %d", i+1)
		assert.Equal(t, int64(3), call.Limit, "limit dropped on page %d", i+1)
	}
}

func TestListWithPagesAcceptsNilOptions(t *testing.T) {
	lister := &pagedLister{pages: []*corev1.PodList{podPage("only", "")}}

	var seen []string
	err := ListWithPages(4, lister.list, context.Background(), nil, collectPodNames(&seen))

	require.NoError(t, err)
	assert.Equal(t, []string{"only"}, seen)
	require.Len(t, lister.calls, 1)
	assert.Equal(t, int64(4), lister.calls[0].Limit)
	// Nothing beyond the paging fields may be invented for a caller that asked for no filtering.
	assert.Equal(t, metav1.ListOptions{Limit: 4}, lister.calls[0])
}

// The walk leaves the caller's ListOptions holding the second-to-last continue token, so reusing
// the same options for a second walk would resume mid-way unless Continue is reset up front.
func TestListWithPagesReusedOptionsStillStartFromTheBeginning(t *testing.T) {
	opts := &metav1.ListOptions{}
	pages := []*corev1.PodList{
		podPage("first", "token-after-first"),
		podPage("second", "token-after-second"),
		podPage("third", ""),
	}

	for _, run := range []string{"first walk", "second walk"} {
		lister := &pagedLister{pages: pages}
		var seen []string
		err := ListWithPages(5, lister.list, context.Background(), opts, collectPodNames(&seen))

		require.NoError(t, err, run)
		assert.Equal(t, []string{"first", "second", "third"}, seen, run)
		require.Len(t, lister.calls, 3, run)
		assert.Equal(t, "", lister.calls[0].Continue, run)
	}
}

// A caller that hands in a continue token from an unrelated, already finished walk must not be
// able to make the pager skip the first page.
func TestListWithPagesIgnoresAContinueTokenSuppliedByTheCaller(t *testing.T) {
	lister := &pagedLister{pages: []*corev1.PodList{podPage("first", "")}}

	var seen []string
	err := ListWithPages(5, lister.list, context.Background(), &metav1.ListOptions{Continue: "stale-token"}, collectPodNames(&seen))

	require.NoError(t, err)
	assert.Equal(t, []string{"first"}, seen)
	require.Len(t, lister.calls, 1)
	assert.Equal(t, "", lister.calls[0].Continue)
}

func TestListWithPagesStopsOnAListError(t *testing.T) {
	listErr := errors.New("apiserver is unavailable")
	lister := &pagedLister{
		pages:     []*corev1.PodList{podPage("first", "token-after-first"), podPage("second", "")},
		err:       listErr,
		errOnCall: 2,
	}

	var seen []string
	err := ListWithPages(5, lister.list, context.Background(), &metav1.ListOptions{}, collectPodNames(&seen))

	require.ErrorIs(t, err, listErr)
	// The first page was already handed to the handler; the walk must not silently report success
	// with a partial result.
	assert.Equal(t, []string{"first"}, seen)
	assert.Len(t, lister.calls, 2)
}

func TestListWithPagesStopsOnAHandlerError(t *testing.T) {
	handlerErr := errors.New("handler rejected the page")
	lister := &pagedLister{pages: []*corev1.PodList{
		podPage("first", "token-after-first"),
		podPage("second", ""),
	}}

	var seen []string
	err := ListWithPages(5, lister.list, context.Background(), &metav1.ListOptions{}, func(list *corev1.PodList) error {
		_ = collectPodNames(&seen)(list)
		return handlerErr
	})

	require.ErrorIs(t, err, handlerErr)
	assert.Equal(t, []string{"first"}, seen)
	// The next page must not be requested once the handler has failed.
	assert.Len(t, lister.calls, 1)
}

func TestListWithPagesDoesNotCallTheHandlerWhenTheFirstListFails(t *testing.T) {
	listErr := errors.New("forbidden")
	lister := &pagedLister{pages: []*corev1.PodList{podPage("first", "")}, err: listErr, errOnCall: 1}

	handlerCalls := 0
	err := ListWithPages(5, lister.list, context.Background(), &metav1.ListOptions{}, func(*corev1.PodList) error {
		handlerCalls++
		return nil
	})

	require.ErrorIs(t, err, listErr)
	assert.Zero(t, handlerCalls)
}

func TestListWithPagesPassesTheCallersContext(t *testing.T) {
	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "carried")

	var got any
	err := ListWithPages(5, func(c context.Context, _ metav1.ListOptions) (*corev1.PodList, error) {
		got = c.Value(ctxKey{})
		return podPage("only", ""), nil
	}, ctx, &metav1.ListOptions{}, func(*corev1.PodList) error { return nil })

	require.NoError(t, err)
	assert.Equal(t, "carried", got)
}
