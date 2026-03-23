package flamegraph

import (
	"debug/dwarf"
	"debug/elf"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Symbolizer resolves (buildID, address) to a function name using debuginfod and DWARF.
// Safe for concurrent use. If debuginfod URLs are not set or buildID is empty, Resolve returns "".
type Symbolizer struct {
	urls    []string
	client  *http.Client
	mu      sync.Mutex
	resCache map[string]string // key: buildID+"\x00"+hex(address)
	dwCache map[string]*dwarf.Data
	tmpDir  string
}

// NewSymbolizer returns a symbolizer that uses DEBUGINFOD_URLS (space-separated).
// If env is empty or not set, Resolve will always return "".
func NewSymbolizer(debuginfodURLs string) *Symbolizer {
	var urls []string
	for _, u := range strings.Fields(strings.TrimSpace(debuginfodURLs)) {
		u = strings.TrimSuffix(u, "/")
		if u != "" {
			urls = append(urls, u)
		}
	}
	return &Symbolizer{
		urls:     urls,
		client:   &http.Client{Timeout: 15 * time.Second},
		resCache: make(map[string]string),
		dwCache:  make(map[string]*dwarf.Data),
		tmpDir:   os.TempDir(),
	}
}

// Resolve returns the function name for (buildID, address). Returns "" if not found or symbolization disabled.
func (s *Symbolizer) Resolve(buildID string, address uint64) string {
	buildID = normalizeBuildID(buildID)
	if buildID == "" || len(s.urls) == 0 {
		return ""
	}
	key := buildID + "\x00" + uint64Hex(address)
	s.mu.Lock()
	if name, ok := s.resCache[key]; ok {
		s.mu.Unlock()
		return name
	}
	s.mu.Unlock()
	name := s.resolveLocked(buildID, address)
	if name != "" {
		s.mu.Lock()
		s.resCache[key] = name
		s.mu.Unlock()
	}
	return name
}

func (s *Symbolizer) resolveLocked(buildID string, address uint64) string {
	dw, err := s.dwarfForBuildID(buildID)
	if err != nil {
		return ""
	}
	entry, err := s.findSubprogram(dw, address)
	if err != nil {
		return ""
	}
	if entry == nil {
		return ""
	}
	if v := entry.Val(dwarf.AttrName); v != nil {
		if n, ok := v.(string); ok && n != "" {
			return n
		}
	}
	if v := entry.Val(dwarf.AttrLinkageName); v != nil {
		if n, ok := v.(string); ok && n != "" {
			return n
		}
	}
	return ""
}

func (s *Symbolizer) dwarfForBuildID(buildID string) (*dwarf.Data, error) {
	s.mu.Lock()
	if dw, ok := s.dwCache[buildID]; ok {
		s.mu.Unlock()
		return dw, nil
	}
	s.mu.Unlock()
	path, err := s.fetchDebuginfo(buildID)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	ef, err := elf.NewFile(f)
	if err != nil {
		return nil, err
	}
	defer ef.Close()
	dw, err := ef.DWARF()
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.dwCache[buildID] = dw
	s.mu.Unlock()
	return dw, nil
}

func (s *Symbolizer) fetchDebuginfo(buildID string) (string, error) {
	for _, base := range s.urls {
		url := base + "/buildid/" + buildID + "/debuginfo"
		resp, err := s.client.Get(url)
		if err != nil {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}
		tmp, err := os.CreateTemp(s.tmpDir, "debuginfo-*")
		if err != nil {
			resp.Body.Close()
			continue
		}
		_, err = io.Copy(tmp, resp.Body)
		resp.Body.Close()
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			continue
		}
		if err := tmp.Close(); err != nil {
			os.Remove(tmp.Name())
			continue
		}
		return tmp.Name(), nil
	}
	return "", os.ErrNotExist
}

// findSubprogram finds the subprogram DIE that contains pc by seeking to the CU and walking children.
func (s *Symbolizer) findSubprogram(dw *dwarf.Data, pc uint64) (*dwarf.Entry, error) {
	r := dw.Reader()
	cu, err := r.SeekPC(pc)
	if err != nil {
		return nil, err
	}
	if cu == nil {
		return nil, nil
	}
	// Reader is now at the first child of the CU; walk to find TagSubprogram containing pc.
	for {
		entry, err := r.Next()
		if err != nil {
			return nil, err
		}
		if entry == nil {
			break
		}
		if entry.Tag == dwarf.TagSubprogram {
			ranges, err := dw.Ranges(entry)
			if err != nil {
				continue
			}
			for _, rng := range ranges {
				if pc >= rng[0] && pc < rng[1] {
					return entry, nil
				}
			}
		}
		if entry.Tag != dwarf.TagSubprogram && entry.Children {
			r.SkipChildren()
		}
	}
	return nil, nil
}

func normalizeBuildID(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "0x")
	s = strings.ToLower(s)
	// Keep only hex chars
	var b strings.Builder
	for _, c := range s {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			b.WriteRune(c)
		}
	}
	return b.String()
}

func uint64Hex(u uint64) string {
	const hex = "0123456789abcdef"
	if u == 0 {
		return "0"
	}
	var b [16]byte
	i := 15
	for u > 0 {
		b[i] = hex[u&0xf]
		u >>= 4
		i--
	}
	return string(b[i+1:])
}

