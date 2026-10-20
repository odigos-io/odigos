package interrogation

import (
	"strings"
	"testing"
)

func TestClassifyFrame(t *testing.T) {
	cases := map[string]FrameKind{
		"":                                FrameKindRuntime,
		"vtable chunks":                   FrameKindRuntime,
		"StubRoutines::call_stub":         FrameKindRuntime,
		"JavaThread::thread_main_inner()": FrameKindRuntime,
		"java.lang.Object java.util.HashMap.get(java.lang.Object)":                      FrameKindRuntime,
		"void jdk.internal.reflect.DirectMethodHandleAccessor.invoke(java.lang.Object)": FrameKindRuntime,
		"void io.opentelemetry.javaagent.shaded.Foo.bar()":                              FrameKindRuntime,

		"java.lang.Object org.springframework.data.repository.core.support.RepositoryMethodInvoker.invoke(java.lang.Object, java.lang.reflect.Method, java.lang.Object[])": FrameKindLibrary,
		"java.lang.Object org.hibernate.internal.SessionImpl.find(java.lang.Class, java.lang.Object, java.util.Map)":                                                       FrameKindLibrary,
		"void org.apache.kafka.clients.consumer.KafkaConsumer.poll(java.time.Duration)":                                                                                    FrameKindLibrary,
		"void org.apache.tomcat.util.threads.ThreadPoolExecutor.runWorker(org.apache.tomcat.util.threads.ThreadPoolExecutor$Worker)":                                       FrameKindLibrary,
		"okhttp3.Response okhttp3.internal.connection.RealCall.execute()":                                                                                                  FrameKindLibrary,

		"org.springframework.http.ResponseEntity com.rowan.notify.DocumentController.create(com.rowan.notify.DocumentRequest)": FrameKindApp,
		"void com.rowan.notify.RenderMessageProducer.publish(java.lang.String, com.rowan.notify.DocumentRequest)":              FrameKindApp,
		"com.rowan.notify.Controller.entry": FrameKindApp,
	}
	for name, want := range cases {
		if got := ClassifyFrame(name); got != want {
			t.Errorf("ClassifyFrame(%q) = %q, want %q", name, got, want)
		}
		if got := IsApplicationFrame(name); got != (want == FrameKindApp) {
			t.Errorf("IsApplicationFrame(%q) = %v", name, got)
		}
		if got := IsRuntimeFrame(name); got != (want == FrameKindRuntime) {
			t.Errorf("IsRuntimeFrame(%q) = %v", name, got)
		}
	}
}

func TestInstrumentationLibrary(t *testing.T) {
	cases := map[string]string{
		"void org.springframework.web.servlet.DispatcherServlet.doDispatch(jakarta.servlet.http.HttpServletRequest, jakarta.servlet.http.HttpServletResponse)": "spring-webmvc",
		"java.lang.Object org.springframework.data.repository.core.support.RepositoryMethodInvoker.invoke(java.lang.Object)":                                   "spring-data",
		"java.util.concurrent.CompletableFuture org.springframework.kafka.core.KafkaTemplate.send(java.lang.String, java.lang.Object)":                         "spring-kafka",
		"void org.apache.kafka.clients.consumer.KafkaConsumer.poll(java.time.Duration)":                                                                        "kafka-clients",
		"java.sql.ResultSet org.postgresql.jdbc.PgPreparedStatement.executeQuery()":                                                                            "jdbc",
		"java.sql.Connection com.zaxxer.hikari.HikariDataSource.getConnection()":                                                                               "hikaricp",
		"void org.apache.catalina.core.StandardWrapperValve.invoke(org.apache.catalina.connector.Request, org.apache.catalina.connector.Response)":             "tomcat",
		// Library, but no default instrumentation.
		"void org.apache.tomcat.util.threads.ThreadPoolExecutor.runWorker(org.apache.tomcat.util.threads.ThreadPoolExecutor$Worker)": "",
		"void org.springframework.beans.factory.support.AbstractBeanFactory.getBean(java.lang.String)":                               "",
		// Not a library frame.
		"void com.rowan.notify.RenderMessageProducer.publish(java.lang.String)": "",
		"java.lang.Object java.util.HashMap.get(java.lang.Object)":              "",
	}
	for name, want := range cases {
		if got := InstrumentationLibrary(name); got != want {
			t.Errorf("InstrumentationLibrary(%q) = %q, want %q", name, got, want)
		}
	}
}

// Every instrumented prefix must classify as a library, or InstrumentationLibrary could
// never report it.
func TestInstrumentedLibrariesAreLibraries(t *testing.T) {
	for prefix, lib := range instrumentedLibraries {
		name := "void " + prefix + "Example.method()"
		if kind := ClassifyFrame(name); kind != FrameKindLibrary {
			t.Errorf("prefix %q classifies as %q, want library", prefix, kind)
		}
		if got := InstrumentationLibrary(name); got != lib {
			t.Errorf("InstrumentationLibrary(%q) = %q, want %q", name, got, lib)
		}
		if !strings.HasSuffix(prefix, ".") {
			t.Errorf("prefix %q must end with a dot to match whole packages", prefix)
		}
	}
}
