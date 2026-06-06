package securitymetrics

import (
	"encoding/json"
	"os"
	"time"
)

// baselineFile is the on-disk form of a Baseline's learned state. Persisting it means an
// agent restart does NOT re-learn from scratch (and thus does not re-flag every known edge /
// destination / port as "new" drift during the warm-up window).
type baselineFile struct {
	Edges   map[string]time.Time `json:"edges"`
	ExtDest map[string]time.Time `json:"ext_dest"`
	Listens map[string]time.Time `json:"listens"`
	SavedAt time.Time            `json:"saved_at"`
}

// SaveBaseline atomically writes the baseline's learned maps to path (write-tmp-then-rename),
// so a crash mid-write never corrupts the file.
func SaveBaseline(b *Baseline, path string) error {
	edges, ext, listens := b.Snapshot()
	data, err := json.Marshal(baselineFile{Edges: edges, ExtDest: ext, Listens: listens, SavedAt: time.Now()})
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o640); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadBaseline restores a baseline from path. A missing file is not an error (first run);
// it just leaves the baseline empty. After loading, drift treats the persisted edges/dests/
// ports as known, so only genuinely new activity is flagged after a restart.
func LoadBaseline(b *Baseline, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var bf baselineFile
	if err := json.Unmarshal(data, &bf); err != nil {
		return err
	}
	b.Restore(bf.Edges, bf.ExtDest, bf.Listens)
	return nil
}
