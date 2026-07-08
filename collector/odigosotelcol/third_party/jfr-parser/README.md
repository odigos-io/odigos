# Java Flight Recorder parser library written in Go.

The parser design is generic, and it should be able to support any kind of type and event.

Current implementation is incomplete, with a focus in supporting the types and events generated
by [async-profiler](https://github.com/jvm-profiling-tools/async-profiler).

## References

- [JEP 328](https://openjdk.java.net/jeps/328) introduces Java Flight Recorder.
- [async-profiler](https://github.com/jvm-profiling-tools/async-profiler) supports includes
  a [JFR writer](https://github.com/jvm-profiling-tools/async-profiler/blob/master/src/flightRecorder.cpp)
  and [reader](https://github.com/jvm-profiling-tools/async-profiler/tree/master/src/converter/one/jfr).
- [JMC](https://github.com/openjdk/jmc) project includes its
  own [JFR parser](https://github.com/openjdk/jmc/tree/master/core/org.openjdk.jmc.flightrecorder/src/main/java/org/openjdk/jmc/flightrecorder/parser) (
  in Java).
- [The JDK Flight Recorder File Format](https://www.morling.dev/blog/jdk-flight-recorder-file-format/)
  by [@gunnarmorling](https://github.com/gunnarmorling) has a great overview of the JFR format.
