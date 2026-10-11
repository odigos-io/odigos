package offsets

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/odigos-io/odigos/common/consts"
)

type GetLatestOffsetsOptions struct {
	Revert   bool
	FromFile string
}

func GetLatestOffsets(opts GetLatestOffsetsOptions) ([]byte, error) {
	if opts.Revert {
		return []byte{}, nil
	}

	// If FromFile is specified, read from local file
	if opts.FromFile != "" {
		data, err := os.ReadFile(opts.FromFile)
		if err != nil {
			return nil, fmt.Errorf("cannot read offsets file: %s", err)
		}
		return data, nil
	}

	resp, err := http.Get(consts.GoOffsetsPublicURL)
	if err != nil {
		return nil, fmt.Errorf("cannot get latest offsets: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cannot get latest offsets: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body: %s", err)
	}
	return data, nil
}
