//go:build darwin && arm64 && cgo

package m1cpu

import (
	"testing"

	"github.com/shoenig/test/must"
)

func Test_PCoreHz(t *testing.T) {
	hz := PCoreHz()
	must.Greater(t, 3_000_000_000, hz)
}

func Test_ECoreHz(t *testing.T) {
	hz := ECoreHz()
	must.Greater(t, 1_000_000_000, hz)
}

func Test_PCoreGHz(t *testing.T) {
	ghz := PCoreGHz()
	must.Greater(t, 3, ghz)
}

func Test_ECoreGHz(t *testing.T) {
	ghz := ECoreGHz()
	must.Greater(t, 1, ghz)
}

func Test_PCoreCount(t *testing.T) {
	n := PCoreCount()
	must.Positive(t, n)
}

func Test_ECoreCount(t *testing.T) {
	n := ECoreCount()
	must.Positive(t, n)
}

func Test_Show(t *testing.T) {
	t.Log("model name", ModelName())
	t.Log("pCore Hz", PCoreHz())
	t.Log("eCore Hz", ECoreHz())
	t.Log("pCore GHz", PCoreGHz())
	t.Log("eCore GHz", ECoreGHz())
	t.Log("pCore count", PCoreCount())
	t.Log("eCoreCount", ECoreCount())

	pL1Inst, pL1Data, pL2 := PCoreCache()
	t.Log("pCore Caches", pL1Inst, pL1Data, pL2)

	eL1Inst, eL1Data, eL2 := ECoreCache()
	t.Log("eCore Caches", eL1Inst, eL1Data, eL2)
}