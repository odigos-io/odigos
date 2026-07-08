package format

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/grafana/jfr-parser/pprof"
)

type FormatterPprof struct{}

func NewFormatterPprof() *FormatterPprof {
	return &FormatterPprof{}
}

func (f *FormatterPprof) Format(buf []byte, dest string) ([]string, [][]byte, error) {
	pi := &pprof.ParseInput{
		StartTime:  time.Now(),
		EndTime:    time.Now(),
		SampleRate: 100,
	}
	profiles, err := pprof.ParseJFR(buf, pi, nil)
	if err != nil {
		return nil, nil, err
	}

	data := make([][]byte, 0)
	dests := make([]string, 0)
	destDir := filepath.Dir(dest)
	destBase := filepath.Base(dest)
	slices.SortFunc(profiles.Profiles, func(i, j pprof.Profile) int {
		if c := strings.Compare(i.Metric, j.Metric); c != 0 {
			return c
		}
		return strings.Compare(
			i.Profile.StringTable[i.Profile.SampleType[0].Type],
			j.Profile.StringTable[j.Profile.SampleType[0].Type],
		)
	})
	for i := 0; i < len(profiles.Profiles); i++ {
		filename := fmt.Sprintf("%s.%d.%s", profiles.Profiles[i].Metric, i, destBase)
		dests = append(dests, filepath.Join(destDir, filename))

		bs, err := profiles.Profiles[i].Profile.MarshalVT()
		if err != nil {
			return nil, nil, err
		}
		data = append(data, bs)
	}
	return dests, data, nil
}
