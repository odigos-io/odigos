package interrogation

import (
	"github.com/odigos-io/odigos/common/interrogation"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

// Only JVM frames carry Java package names the classifier understands.
const (
	frameTypeJVM     = "jvm"
	frameTypeHotspot = "hotspot"
)

var frameKindToModel = map[interrogation.FrameKind]model.InterrogationFrameKind{
	interrogation.FrameKindApp:     model.InterrogationFrameKindApp,
	interrogation.FrameKindLibrary: model.InterrogationFrameKindLibrary,
	interrogation.FrameKindRuntime: model.InterrogationFrameKindRuntime,
}

// classifyFrame returns the frame's kind. Non-JVM frames are treated as application
// code, as they were before classification.
func classifyFrame(name, frameType string) model.InterrogationFrameKind {
	if frameType != frameTypeJVM && frameType != frameTypeHotspot {
		return model.InterrogationFrameKindApp
	}
	return frameKindToModel[interrogation.ClassifyFrame(name)]
}

// TransactionsToModel converts ClickHouse-backed transactions to GraphQL models.
func TransactionsToModel(txs []Transaction) []*model.InterrogationTransaction {
	out := make([]*model.InterrogationTransaction, 0, len(txs))
	for _, tx := range txs {
		fns := make([]*model.InterrogationFunction, 0, len(tx.Functions))
		for _, fn := range tx.Functions {
			fns = append(fns, &model.InterrogationFunction{
				Name:       fn.Name,
				FrameType:  fn.FrameType,
				FrameKind:  classifyFrame(fn.Name, fn.FrameType),
				SampleType: fn.SampleType,
				SeenCount:  int(fn.SeenCount),
			})
		}
		out = append(out, &model.InterrogationTransaction{
			ID:        tx.ID,
			SeenCount: int(tx.SeenCount),
			Functions: fns,
		})
	}
	return out
}

// SourcesToModel converts per-container transaction counts to GraphQL models.
// Rows whose workload kind is not a known K8sResourceKind are dropped, since the
// enum could not be serialized.
func SourcesToModel(sources []SourceTransactions) []*model.InterrogationSource {
	out := make([]*model.InterrogationSource, 0, len(sources))
	for _, s := range sources {
		kind := model.K8sResourceKind(s.Kind)
		if !kind.IsValid() {
			continue
		}
		out = append(out, &model.InterrogationSource{
			ID:               &model.K8sWorkloadID{Namespace: s.Namespace, Kind: kind, Name: s.Name},
			ContainerName:    s.ContainerName,
			TransactionCount: int(s.TransactionCount),
		})
	}
	return out
}

// CallTrieToModel converts flat call-path trie nodes to GraphQL models.
func CallTrieToModel(nodes []CallTrieNode) []*model.InterrogationCallTrieNode {
	out := make([]*model.InterrogationCallTrieNode, 0, len(nodes))
	for _, n := range nodes {
		var parentID *string
		if n.ParentID != "" {
			parentID = &n.ParentID
		}
		out = append(out, &model.InterrogationCallTrieNode{
			ID:         n.ID,
			ParentID:   parentID,
			Name:       n.Name,
			FrameType:  n.FrameType,
			FrameKind:  classifyFrame(n.Name, n.FrameType),
			SampleType: n.SampleType,
			SeenCount:  int(n.SeenCount),
		})
	}
	return out
}
