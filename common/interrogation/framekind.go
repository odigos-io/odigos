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
	// Libraries covered by default OTel instrumentation (see instrumentedLibraries).
	"org.eclipse.jetty.", "io.undertow.", "com.mysql.", "org.mariadb.jdbc.", "oracle.jdbc.",
	"com.microsoft.sqlserver.jdbc.", "redis.clients.jedis.", "io.lettuce.core.", "com.mongodb.",
	"io.grpc.", "okhttp3.", "software.amazon.awssdk.", "com.amazonaws.", "org.elasticsearch.client.",
	"com.rabbitmq.client.", "com.datastax.oss.driver.",
}

// instrumentedLibraries maps library package prefixes to the OTel Java agent
// instrumentation that covers them by default. The longest matching prefix wins.
var instrumentedLibraries = map[string]string{
	"org.springframework.web.servlet.":  "spring-webmvc",
	"org.springframework.web.reactive.": "spring-webflux",
	"org.springframework.web.client.":   "spring-web",
	"org.springframework.kafka.":        "spring-kafka",
	"org.springframework.jms.":          "spring-jms",
	"org.springframework.amqp.rabbit.":  "spring-rabbit",
	"org.springframework.scheduling.":   "spring-scheduling",
	"org.springframework.data.":         "spring-data",
	"org.apache.catalina.":              "tomcat",
	"org.apache.coyote.":                "tomcat",
	"org.eclipse.jetty.server.":         "jetty",
	"org.eclipse.jetty.client.":         "jetty-httpclient",
	"io.undertow.":                      "undertow",
	"io.netty.":                         "netty",
	"reactor.netty.":                    "reactor-netty",
	"org.postgresql.":                   "jdbc",
	"com.mysql.":                        "jdbc",
	"org.mariadb.jdbc.":                 "jdbc",
	"org.h2.":                           "jdbc",
	"oracle.jdbc.":                      "jdbc",
	"com.microsoft.sqlserver.jdbc.":     "jdbc",
	"com.zaxxer.hikari.":                "hikaricp",
	"org.hibernate.":                    "hibernate",
	"org.apache.kafka.clients.":         "kafka-clients",
	"redis.clients.jedis.":              "jedis",
	"io.lettuce.core.":                  "lettuce",
	"com.mongodb.":                      "mongo",
	"io.grpc.":                          "grpc",
	"okhttp3.":                          "okhttp",
	"org.apache.http.":                  "apache-httpclient",
	"org.apache.hc.client5.":            "apache-httpclient",
	"software.amazon.awssdk.":           "aws-sdk",
	"com.amazonaws.":                    "aws-sdk",
	"org.elasticsearch.client.":         "elasticsearch-rest",
	"com.rabbitmq.client.":              "rabbitmq",
	"com.datastax.oss.driver.":          "cassandra",
	"org.apache.logging.log4j.":         "log4j",
	"ch.qos.logback.classic.":           "logback",
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

// InstrumentationLibrary names the OTel Java agent instrumentation that covers name's
// library by default, or "" when it is not a library frame or no default covers it.
func InstrumentationLibrary(name string) string {
	if ClassifyFrame(name) != FrameKindLibrary {
		return ""
	}
	class := ClassFromFrameName(strings.TrimSpace(name))
	best, library := "", ""
	for prefix, lib := range instrumentedLibraries {
		if len(prefix) > len(best) && strings.HasPrefix(class, prefix) {
			best, library = prefix, lib
		}
	}
	return library
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
