# Java

Statically-typed, object-oriented language running on the JVM with automatic memory management, strong concurrency primitives, and a vast ecosystem spanning enterprise, mobile, and distributed systems.

## JVM Architecture

```
┌─────────────────────────────────────────────┐
│                  JVM                        │
│  ┌──────────┐  ┌──────────┐  ┌───────────┐ │
│  │ ClassLoad│  │ Runtime   │  │ Execution │ │
│  │   er     │──│ Data Areas│──│  Engine   │ │
│  └──────────┘  └──────────┘  └───────────┘ │
│                     │                       │
│         ┌───────────┼───────────┐           │
│         │           │           │           │
│     ┌───┴──┐   ┌────┴───┐  ┌───┴──┐        │
│     │ Heap │   │ Stack  │  │Method│        │
│     │      │   │(per    │  │ Area │        │
│     │      │   │thread) │  │      │        │
│     └──────┘   └────────┘  └──────┘        │
└─────────────────────────────────────────────┘
```

## Compile, Run, Package

```bash
# Compile
javac -d out src/com/example/Main.java

# Run
java -cp out com.example.Main

# Compile with module path
javac -d out --module-source-path src -m com.example.app

# Create JAR
jar cf app.jar -C out .

# Create executable JAR
jar cfe app.jar com.example.Main -C out .
java -jar app.jar

# jlink - create custom runtime image
jlink --module-path $JAVA_HOME/jmods:mods \
      --add-modules com.example.app \
      --output custom-runtime

# JVM tuning flags
java -Xms512m -Xmx4g -XX:+UseZGC -jar app.jar
java -XX:+PrintFlagsFinal -version 2>&1 | grep -i gc
```

## Module System (JPMS)

```java
// module-info.java
module com.example.app {
    requires java.net.http;
    requires java.sql;
    requires transitive com.example.core;

    exports com.example.app.api;
    exports com.example.app.internal to com.example.tests;

    opens com.example.app.models to com.fasterxml.jackson.databind;

    provides com.example.spi.Plugin
        with com.example.app.DefaultPlugin;

    uses com.example.spi.Plugin;
}
```

## Collections Framework

```java
// List
List<String> list = List.of("a", "b", "c");           // immutable
List<String> mutable = new ArrayList<>(list);
List<String> linked = new LinkedList<>();

// Map
Map<String, Integer> map = Map.of("x", 1, "y", 2);    // immutable
Map<String, Integer> hm = new HashMap<>();
Map<String, Integer> sorted = new TreeMap<>();
Map<String, List<String>> grouped = new ConcurrentHashMap<>();

// Set
Set<Integer> set = Set.of(1, 2, 3);                    // immutable
Set<Integer> hs = new LinkedHashSet<>();                // insertion order

// Queue / Deque
Queue<String> queue = new ArrayDeque<>();
Deque<String> stack = new ArrayDeque<>();               // prefer over Stack
PriorityQueue<Integer> pq = new PriorityQueue<>(Comparator.reverseOrder());

// Concurrent collections
ConcurrentHashMap<String, Integer> concMap = new ConcurrentHashMap<>();
CopyOnWriteArrayList<String> cowList = new CopyOnWriteArrayList<>();
BlockingQueue<String> bq = new LinkedBlockingQueue<>(100);
```

## Streams API

```java
// Basic pipeline
List<String> result = names.stream()
    .filter(n -> n.length() > 3)
    .map(String::toUpperCase)
    .sorted()
    .distinct()
    .collect(Collectors.toList());

// Reduction
int total = numbers.stream().reduce(0, Integer::sum);
OptionalInt max = IntStream.of(1, 5, 3).max();

// Grouping and partitioning
Map<String, List<Employee>> byDept = employees.stream()
    .collect(Collectors.groupingBy(Employee::department));

Map<Boolean, List<Employee>> partitioned = employees.stream()
    .collect(Collectors.partitioningBy(e -> e.salary() > 100_000));

// Flat mapping
List<String> allWords = sentences.stream()
    .flatMap(s -> Arrays.stream(s.split("\\s+")))
    .collect(Collectors.toList());

// Parallel streams
long count = hugeList.parallelStream()
    .filter(item -> item.isValid())
    .count();

// Collectors
String joined = names.stream().collect(Collectors.joining(", "));
DoubleSummaryStatistics stats = employees.stream()
    .collect(Collectors.summarizingDouble(Employee::salary));
```

## Concurrency

```java
// CompletableFuture
CompletableFuture<String> future = CompletableFuture
    .supplyAsync(() -> fetchData("url"))
    .thenApply(data -> parse(data))
    .thenCompose(parsed -> saveAsync(parsed))
    .exceptionally(ex -> handleError(ex));

// Combine multiple futures
CompletableFuture<Void> all = CompletableFuture.allOf(f1, f2, f3);
CompletableFuture<Object> any = CompletableFuture.anyOf(f1, f2, f3);

// Virtual threads (Java 21+)
try (var executor = Executors.newVirtualThreadPerTaskExecutor()) {
    List<Future<String>> futures = urls.stream()
        .map(url -> executor.submit(() -> fetch(url)))
        .toList();
    for (var f : futures) {
        System.out.println(f.get());
    }
}

// Structured concurrency (preview)
try (var scope = new StructuredTaskScope.ShutdownOnFailure()) {
    Subtask<String> user = scope.fork(() -> findUser(id));
    Subtask<Order> order = scope.fork(() -> fetchOrder(id));
    scope.join().throwIfFailed();
    return new Response(user.get(), order.get());
}

// Classic synchronization
synchronized (lock) { sharedState.update(); }
ReentrantLock lock = new ReentrantLock();
Semaphore semaphore = new Semaphore(10);
CountDownLatch latch = new CountDownLatch(3);
```

## Records, Sealed Classes, Pattern Matching

```java
// Records (Java 16+)
public record Point(double x, double y) {
    public double distanceTo(Point other) {
        return Math.sqrt(Math.pow(x - other.x, 2) + Math.pow(y - other.y, 2));
    }
}

// Sealed classes (Java 17+)
public sealed interface Shape
    permits Circle, Rectangle, Triangle {
}
public record Circle(double radius) implements Shape {}
public record Rectangle(double w, double h) implements Shape {}
public record Triangle(double a, double b, double c) implements Shape {}

// Pattern matching with switch (Java 21+)
double area = switch (shape) {
    case Circle c    -> Math.PI * c.radius() * c.radius();
    case Rectangle r -> r.w() * r.h();
    case Triangle t  -> {
        double s = (t.a() + t.b() + t.c()) / 2;
        yield Math.sqrt(s * (s - t.a()) * (s - t.b()) * (s - t.c()));
    }
};
```

## GC Tuning

```bash
# G1 (default since Java 9) - balanced latency/throughput
java -XX:+UseG1GC -XX:MaxGCPauseMillis=200 -Xmx4g -jar app.jar

# ZGC - ultra-low latency (<1ms pauses)
java -XX:+UseZGC -Xmx16g -jar app.jar

# Shenandoah - low latency alternative
java -XX:+UseShenandoahGC -Xmx8g -jar app.jar

# GC logging
java -Xlog:gc*:file=gc.log:time,uptime,level,tags -jar app.jar

# Heap dump on OOM
java -XX:+HeapDumpOnOutOfMemoryError \
     -XX:HeapDumpPath=/tmp/heapdump.hprof -jar app.jar

# Diagnostic tools
jps                       # list JVM processes
jstat -gc <pid> 1000      # GC stats every 1s
jmap -histo <pid>         # heap histogram
jstack <pid>              # thread dump
jcmd <pid> GC.heap_info   # heap info
jfr start name=rec duration=60s filename=rec.jfr  # flight recorder
```

## Maven / Gradle Basics

```xml
<!-- pom.xml (Maven) -->
<project>
    <groupId>com.example</groupId>
    <artifactId>myapp</artifactId>
    <version>1.0.0</version>
    <dependencies>
        <dependency>
            <groupId>com.google.guava</groupId>
            <artifactId>guava</artifactId>
            <version>33.0.0-jre</version>
        </dependency>
    </dependencies>
</project>
```

```bash
mvn clean compile            # compile
mvn test                     # run tests
mvn package                  # build JAR
mvn dependency:tree          # show dependency tree
```

```kotlin
// build.gradle.kts (Gradle Kotlin DSL)
plugins {
    java
    application
}

dependencies {
    implementation("com.google.guava:guava:33.0.0-jre")
    testImplementation("org.junit.jupiter:junit-jupiter:5.10.2")
}

application {
    mainClass.set("com.example.Main")
}
```

```bash
gradle build                 # compile + test + package
gradle test                  # run tests
gradle dependencies          # show dependency tree
gradle run                   # run application
```

## Tips

- Use `var` (Java 10+) for local variables where the type is obvious from context, but keep explicit types in method signatures
- Prefer `List.of()`, `Map.of()`, `Set.of()` for creating immutable collections -- they are null-hostile and fail fast
- Use virtual threads (Java 21+) for I/O-bound workloads instead of thread pools -- they scale to millions of concurrent tasks
- Choose `record` for data carriers, `sealed interface` for type hierarchies, and pattern matching `switch` to eliminate `instanceof` chains
- Use ZGC for latency-sensitive services and G1 for general-purpose workloads; always enable GC logging in production
- Prefer `CompletableFuture` over raw `Thread`/`Runnable` for async workflows; chain with `thenApply`/`thenCompose`
- Use `ConcurrentHashMap.computeIfAbsent()` instead of check-then-put patterns to avoid race conditions
- Enable `-XX:+HeapDumpOnOutOfMemoryError` in all production JVMs -- you will need that heap dump eventually
- Avoid `parallelStream()` for short collections or CPU-light operations; the fork-join overhead often exceeds the benefit
- Use the module system (JPMS) for strong encapsulation, but be aware that many libraries still use automatic modules
- Run `jcmd <pid> VM.flags` to inspect active JVM flags and verify your tuning is actually applied

## See Also

- Kotlin (JVM language)
- Scala (JVM language)
- GraalVM (polyglot VM, native image)
- JMH (Java Microbenchmark Harness)
- JUnit 5 and Mockito

## References

- [Java SE Documentation](https://docs.oracle.com/en/java/javase/21/)
- [JVM Specification](https://docs.oracle.com/javase/specs/jvms/se21/html/index.html)
- [Java Concurrency in Practice (Brian Goetz)](https://jcip.net/)
- [Effective Java (Joshua Bloch)](https://www.oreilly.com/library/view/effective-java/9780134686097/)
- [JEP Index](https://openjdk.org/jeps/0)
- [OpenJDK Wiki](https://wiki.openjdk.org/)
- [GC Tuning Guide](https://docs.oracle.com/en/java/javase/21/gctuning/)
