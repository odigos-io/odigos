package types

type IDMap[REF interface{ MethodRef | FrameTypeRef }] struct {
	Dict  map[REF]uint32
	Slice []uint32
	Size  int
}

func NewIDMap[REF interface{ MethodRef | FrameTypeRef }](n int) IDMap[REF] {
	return IDMap[REF]{
		Slice: make([]uint32, n+1),
	}
}

func (m *IDMap[REF]) Get(ref REF) int {
	if m.Dict == nil {
		if int(ref) < len(m.Slice) {
			return int(m.Slice[ref])
		}
		return -1
	}
	return m.getDict(ref)
}

func (m *IDMap[REF]) Set(ref REF, idx int) {
	if m.Dict == nil && int(ref) < len(m.Slice) {
		m.Slice[ref] = uint32(idx)
		return
	}
	m.setSlow(ref, idx)
}

func (m *IDMap[REF]) setSlow(ref REF, idx int) {
	if m.Dict == nil {
		m.Dict = make(map[REF]uint32, m.Size)
		for i, v := range m.Slice {
			m.Dict[REF(i)] = v
		}
		m.Slice = nil
	}
	m.Dict[ref] = uint32(idx)
}

func (m *IDMap[REF]) getDict(ref REF) int {
	u, ok := m.Dict[ref]
	if ok {
		return int(u)
	}
	return -1
}
