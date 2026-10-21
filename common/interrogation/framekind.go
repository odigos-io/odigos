// Package interrogation classifies JVM profile frame labels for interrogation call tries.
// The enterprise collector exporter, the interrogation loop and the UI API all classify
// with these rules, so ingest, AI evidence and the function map stay aligned.
package interrogation

import "strings"

// FrameKind is what a profile frame belongs to.
type FrameKind string

const (
	// FrameKindApp is customer code: probe candidates and AI evidence.
	FrameKindApp FrameKind = "app"
	// FrameKindLibrary is third-party framework or library code. Kept in call tries so
	// the function map can draw app → library → app flows, but never probed.
	FrameKindLibrary FrameKind = "library"
	// FrameKindRuntime is the JDK, the VM and agent plumbing. Dropped at ingest.
	FrameKindRuntime FrameKind = "runtime"
)

var runtimePrefixes = []string{
	"java.", "javax.", "jakarta.", "sun.", "jdk.", "kotlin.", "scala.", "com.sun.", "org.jcp.",
	"net.bytebuddy.", "io.opentelemetry.",
}

var libraryPrefixes = []string{
	"reactor.", "io.netty.", "io.micrometer.",
	"org.springframework.", "org.hibernate.", "org.apache.", "org.postgresql.", "org.h2.",
	"org.slf4j.", "ch.qos.logback.", "com.zaxxer.",
	// Common servers, clients and drivers.
	"org.eclipse.jetty.", "io.undertow.", "com.mysql.", "org.mariadb.jdbc.", "oracle.jdbc.",
	"com.microsoft.sqlserver.jdbc.", "redis.clients.jedis.", "io.lettuce.core.", "com.mongodb.",
	"io.grpc.", "okhttp3.", "software.amazon.awssdk.", "com.amazonaws.", "org.elasticsearch.client.",
	"com.rabbitmq.client.", "com.datastax.oss.driver.",
}

// ClassifyFrame reports which FrameKind a JVM frame label belongs to.
func ClassifyFrame(name string) FrameKind {
	name = strings.TrimSpace(name)
	if name == "" {
		return FrameKindRuntime
	}
	lower := strings.ToLower(name)
	if strings.Contains(lower, "vtable") ||
		strings.HasPrefix(lower, "stubroutines") ||
		strings.Contains(lower, "_new_instance_java") {
		return FrameKindRuntime
	}
	class := ClassFromFrameName(name)
	if class == "" || hasAnyPrefix(class, runtimePrefixes) {
		return FrameKindRuntime
	}
	if hasAnyPrefix(class, libraryPrefixes) {
		return FrameKindLibrary
	}
	return FrameKindApp
}

// IsApplicationFrame reports whether name is customer code worth probe
// recommendations and stable evidence hashing.
func IsApplicationFrame(name string) bool {
	return ClassifyFrame(name) == FrameKindApp
}

// IsRuntimeFrame is true when the frame should be dropped at ingest.
func IsRuntimeFrame(name string) bool {
	return ClassifyFrame(name) == FrameKindRuntime
}

// ClassFromFrameName extracts the qualified class/method prefix from a JVM frame label.
func ClassFromFrameName(name string) string {
	if i := strings.IndexByte(name, ' '); i > 0 {
		name = name[i+1:]
	}
	if j := strings.IndexByte(name, '('); j > 0 {
		name = name[:j]
	}
	if !strings.Contains(name, ".") {
		return ""
	}
	return name
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
