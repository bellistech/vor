# Java (Programming Language)

Statically-typed, class-based, object-oriented language compiled to bytecode and executed on the JVM with garbage collection, JIT compilation, a vast standard library, and a deep ecosystem (Maven/Gradle, Spring, JUnit, Jackson) used across enterprise services, Android, big data, and high-frequency trading.

## Setup

JDK distributions: pick one and stay on LTS (8, 11, 17, 21, 25). Most teams run Temurin (Eclipse Adoptium) or Corretto in production.

```bash
# macOS via Homebrew (Temurin / Eclipse Adoptium)
# brew install --cask temurin@21
# brew install openjdk@21    # OpenJDK build, sym-link to /usr/local/opt

# Linux via package manager
# sudo apt install openjdk-21-jdk          # Debian/Ubuntu
# sudo dnf install java-21-openjdk-devel   # Fedora/RHEL

# SDKMAN — multi-JDK manager (recommended for devs)
# curl -s "https://get.sdkman.io" | bash
# sdk list java
# sdk install java 21.0.4-tem    # Temurin
# sdk install java 21.0.4-amzn   # Corretto (Amazon)
# sdk install java 21.0.4-zulu   # Zulu (Azul)
# sdk install java 21.0.4-librca # Liberica (BellSoft)
# sdk install java 21.0.4-graal  # GraalVM
# sdk default java 21.0.4-tem
# sdk use java 17.0.12-tem       # this shell only

# JAVA_HOME (must point to JDK, not JRE)
# export JAVA_HOME=$(/usr/libexec/java_home -v 21)        # macOS
# export JAVA_HOME=/usr/lib/jvm/java-21-openjdk-amd64     # Linux
# export PATH="$JAVA_HOME/bin:$PATH"

# Verify
# java -version          # runtime version
# javac -version         # compiler version (must match)
# echo $JAVA_HOME

# Distribution differences
# Temurin   : Eclipse Foundation, TCK-certified, free, default for most
# Corretto  : Amazon, free, used in AWS Lambda
# Liberica  : BellSoft, includes JavaFX bundles
# Zulu      : Azul, commercial support tiers
# Oracle JDK: requires NFTC license — free for dev/test, paid for prod
# GraalVM   : polyglot + native-image AOT compiler
```

## Tooling

```bash
# javac — compiler
# javac Hello.java                              # outputs Hello.class
# javac -d out src/Main.java                    # output dir
# javac -cp libs/*:.                            # classpath (colon on UNIX, ; on Windows)
# javac --release 17 Hello.java                 # compile against JDK 17 API
# javac -source 17 -target 17 Hello.java        # legacy (use --release)
# javac -Xlint:all -Werror src/Main.java        # warnings as errors
# javac -parameters Hello.java                  # keep param names for reflection
# javac -g Hello.java                           # debug info (lines, vars, source)
# javac --module-path libs --add-modules ALL-MODULE-PATH

# java — runtime / launcher
# java Hello                                    # run class with default classpath
# java -cp out:libs/* com.example.Main          # explicit classpath
# java -jar app.jar                             # run executable JAR
# java -p mods -m com.example/com.example.Main  # module path + main module
# java Hello.java                               # JEP 330: single-file source-code program
# java --version                                # JDK version
# java --enable-preview ...                     # opt-in preview features
# java -ea Main                                 # enable assertions
# java -Dkey=value Main                         # system property

# jshell — REPL (9+)
# jshell                                        # starts REPL
# jshell> int x = 1 + 2
# jshell> System.out.println(x)
# jshell> /vars      /methods   /imports   /list   /save out.jsh   /exit
# jshell --execution local script.jsh

# jpackage — native installer (14+)
# jpackage --name MyApp --input out --main-jar app.jar --type pkg
# jpackage --type dmg --mac-package-identifier com.example.MyApp ...
# jpackage --type msi --win-menu --win-shortcut ...
# jpackage --type deb ...

# jlink — custom runtime image
# jlink --module-path $JAVA_HOME/jmods:mods \
#       --add-modules com.example.app \
#       --launcher app=com.example.app/com.example.Main \
#       --output dist --strip-debug --compress 2 --no-header-files

# jcmd — JVM diagnostic command (does most of jstack/jmap/jstat work)
# jcmd                                # list JVMs
# jcmd <pid> help
# jcmd <pid> Thread.print             # thread dump
# jcmd <pid> GC.heap_info
# jcmd <pid> GC.run                   # request GC
# jcmd <pid> VM.system_properties
# jcmd <pid> VM.flags
# jcmd <pid> JFR.start name=rec duration=60s filename=rec.jfr
# jcmd <pid> JFR.dump filename=now.jfr
# jcmd <pid> VM.native_memory summary

# jstack — thread dump
# jstack -l <pid>                     # include lock info

# jmap — heap inspection
# jmap -histo:live <pid>              # live object histogram
# jmap -dump:format=b,file=heap.hprof <pid>

# jstat — GC stats
# jstat -gc <pid> 1000 10             # 10 samples, 1s apart
# jstat -gcutil <pid> 1000

# jfr — flight recorder CLI (file analysis)
# jfr print --events jdk.GarbageCollection rec.jfr
# jfr summary rec.jfr

# JConsole / VisualVM / JDK Mission Control — GUI profilers
# jconsole                            # bundled with JDK
# jvisualvm                           # separate download since JDK 9
# jmc                                 # Mission Control — JFR analysis GUI

# javap — disassembler
# javap -c -p Hello                   # bytecode + private members
# javap -v Hello                      # verbose (constant pool, flags)
# javap -s Hello                      # JVM internal type signatures

# javadoc — API doc generator
# javadoc -d docs -sourcepath src -subpackages com.example
```

## Build Tools

Maven and Gradle dominate. Maven uses XML and convention; Gradle uses a Groovy or Kotlin DSL and is faster on incremental builds.

```bash
# Maven directory layout (standard)
# myapp/
#   pom.xml
#   src/main/java/com/example/...
#   src/main/resources/
#   src/test/java/com/example/...
#   target/                  # build output

# pom.xml (minimum viable)
# <project xmlns="http://maven.apache.org/POM/4.0.0">
#   <modelVersion>4.0.0</modelVersion>
#   <groupId>com.example</groupId>
#   <artifactId>myapp</artifactId>
#   <version>1.0.0</version>
#   <packaging>jar</packaging>
#   <properties>
#     <maven.compiler.release>21</maven.compiler.release>
#     <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>
#   </properties>
#   <dependencies>
#     <dependency>
#       <groupId>org.junit.jupiter</groupId>
#       <artifactId>junit-jupiter</artifactId>
#       <version>5.10.2</version>
#       <scope>test</scope>
#     </dependency>
#   </dependencies>
# </project>

# Maven lifecycle phases (each phase runs all earlier phases)
# validate -> compile -> test -> package -> verify -> install -> deploy
# clean phase: pre-clean -> clean -> post-clean
# site phase:  pre-site -> site -> post-site -> site-deploy

# Maven CLI flags
# mvn clean package              # most common
# mvn -DskipTests package        # skip tests
# mvn -Dtest=MyTest test         # run only one test class
# mvn -Dtest=MyTest#one test     # run one test method
# mvn -pl mod-a -am package      # build module + dependencies
# mvn -T 4 package               # parallel build, 4 threads
# mvn -X package                 # debug output
# mvn -o package                 # offline mode
# mvn -U package                 # force update snapshots
# mvn dependency:tree            # show dep graph
# mvn dependency:analyze         # find unused/undeclared deps
# mvn versions:display-dependency-updates
# mvn help:effective-pom         # final merged POM
# mvn help:effective-settings

# Gradle Kotlin DSL — build.gradle.kts
# plugins {
#   `java-library`
#   application
# }
# repositories { mavenCentral() }
# dependencies {
#   implementation("com.fasterxml.jackson.core:jackson-databind:2.17.0")
#   testImplementation("org.junit.jupiter:junit-jupiter:5.10.2")
# }
# java { toolchain { languageVersion = JavaLanguageVersion.of(21) } }
# tasks.test { useJUnitPlatform() }
# application { mainClass.set("com.example.Main") }

# Gradle Groovy DSL — build.gradle (legacy)
# plugins { id 'java-library'; id 'application' }
# dependencies { implementation 'com.google.guava:guava:33.0.0-jre' }

# Gradle CLI flags
# ./gradlew build                # full build (assemble + check)
# ./gradlew test                 # tests only
# ./gradlew run                  # run mainClass
# ./gradlew --daemon             # default — keeps JVM warm
# ./gradlew --no-daemon          # one-shot
# ./gradlew --parallel           # parallel module builds
# ./gradlew --offline
# ./gradlew --refresh-dependencies
# ./gradlew dependencies         # dep graph
# ./gradlew dependencyInsight --dependency jackson-databind
# ./gradlew tasks                # list available tasks
# ./gradlew -i / -d / -s         # info / debug / stacktrace
# ./gradlew --scan               # generate build scan URL
```

## Project Layout

```bash
# Standard Maven / Gradle layout
# myapp/
#   pom.xml | build.gradle.kts
#   src/
#     main/
#       java/                       # production code
#         com/example/MyApp.java
#       resources/                  # bundled into JAR
#         application.properties
#         logback.xml
#       webapp/                     # WAR projects only
#     test/
#       java/                       # JUnit / TestNG
#         com/example/MyAppTest.java
#       resources/                  # test-only resources
#   target/ | build/                # compiled output

# Package convention: reverse-DNS, lowercase, no underscores
# package com.example.myapp.service;
# Source file path MUST mirror package: src/main/java/com/example/myapp/service/UserService.java

# One public top-level class per file, file name == class name (case-sensitive).
# javac error if mismatched:
#   error: class UserService is public, should be declared in a file named UserService.java
```

## Variables & Primitives

```bash
# Eight primitives (NOT objects, no methods, no null)
# boolean — true|false                  (1 bit logically; size JVM-defined)
# byte    — signed 8-bit                (-128 .. 127)
# short   — signed 16-bit               (-32_768 .. 32_767)
# int     — signed 32-bit               (-2_147_483_648 .. 2_147_483_647)
# long    — signed 64-bit               (suffix L)
# float   — IEEE 754 32-bit             (suffix f or F — REQUIRED)
# double  — IEEE 754 64-bit             (default for decimal literals)
# char    — unsigned 16-bit UTF-16 unit (single quotes 'a', 'é')

# Underscore digit separators (7+)
# long maxInt   = 2_147_483_647L;
# double pi     = 3.141_592_653_589_793;
# int hex       = 0xFF_FF;
# int binary    = 0b1010_1010;
# int octal     = 0177;

# Wrapper classes (objects, nullable, in java.lang)
# Boolean Byte Short Integer Long Float Double Character
# Integer i = Integer.valueOf(42);    // boxing
# int n     = i.intValue();           // unboxing
# Integer.parseInt("42");
# Integer.toBinaryString(42);

# Autoboxing / unboxing — implicit conversion to/from wrapper
# Integer x = 5;          // autobox
# int y = x;              // unbox — NPE if x is null
# Map<String,Integer> m = ...;
# int v = m.get("x");     // throws NullPointerException if key missing!

# var — local-variable type inference (10+) — ONLY local vars / for / try-with-resources
# var list = new ArrayList<String>();
# var stream = list.stream();
# // var x = null;          // ERROR: cannot infer type from null
# // var x;                 // ERROR: cannot use 'var' on variable without initializer

# final — value cannot be reassigned (reference still mutable for objects)
# final int MAX = 100;
# final List<String> names = new ArrayList<>();
# names.add("a");          // OK — list is final, contents mutable

# Default values (instance/static fields, NOT locals)
# numeric -> 0, boolean -> false, char -> ' ', reference -> null
# Local variables MUST be definitely assigned before use:
#   error: variable x might not have been initialized
```

## Strings

```bash
# Strings are immutable. All "mutating" methods return new String.
# String s = "hello";
# String t = s.toUpperCase();   // s unchanged

# String pool — string literals are interned
# String a = "hi";
# String b = "hi";
# a == b                  // true  (same pooled reference)
# String c = new String("hi");
# a == c                  // false (heap copy)
# a.equals(c)             // true
# c.intern() == a         // true

# Equality — ALWAYS use .equals(), not ==
# if (s.equals("yes")) ...
# if ("yes".equals(s)) ...   // null-safe form

# Common methods
# s.length(); s.isEmpty(); s.isBlank()         // 11+
# s.charAt(i); s.indexOf("x"); s.lastIndexOf("x")
# s.substring(2, 5)                            // [2,5)
# s.replace("a", "b"); s.replaceAll("\\s+", " ")
# s.split(","); s.split(",", -1)               // -1 keeps trailing empties
# s.trim(); s.strip(); s.stripLeading(); s.stripTrailing()  // 11+
# s.contains("x"); s.startsWith("a"); s.endsWith("z")
# s.toLowerCase(Locale.ROOT)                   // ALWAYS pass locale
# String.join(",", list)
# s.chars().mapToObj(c -> (char)c)             // IntStream of code units
# s.codePoints()                               // IntStream of code points
# s.repeat(3)                                  // 11+
# s.lines()                                    // 11+ -> Stream<String>

# Formatting
# String.format("%-10s | %5d | %.2f%n", name, qty, price)
# "%s".formatted(name)                         // 15+ instance method
# // %s string  %d int  %f float  %x hex  %o octal  %b boolean  %c char
# // %n newline  %% literal %  %5d width 5  %.2f 2 decimals  %-5d left-align

# MessageFormat — i18n / positional
# MessageFormat.format("{0} owes {1,number,currency}", name, amt);

# Text blocks (15+)
# String json = """
#         {
#           "name": "%s",
#           "age": %d
#         }
#         """.formatted(name, age);
# // Indentation: incidental whitespace stripped relative to closing """
# // Newlines preserved; trailing whitespace trimmed; \ at line end suppresses newline.

# StringBuilder vs StringBuffer
# StringBuilder — NOT thread-safe, faster — DEFAULT choice
# StringBuffer  — synchronized, slower, legacy
# var sb = new StringBuilder(64);
# sb.append("hello").append(' ').append(name).append('!');
# String result = sb.toString();
# sb.reverse(); sb.insert(0, "<"); sb.delete(0, 3); sb.setLength(0);

# DON'T use + in a loop — quadratic time
# String r = "";
# for (var x : list) r += x;     // O(n^2) — creates n new Strings

# DO
# var sb = new StringBuilder();
# for (var x : list) sb.append(x);
# String r = sb.toString();
```

## Numbers & Math

```bash
# Integer overflow does NOT throw — wraps silently
# int x = Integer.MAX_VALUE + 1;   // -2_147_483_648  (no exception!)

# Use Math.addExact / subtractExact / multiplyExact / negateExact / toIntExact for checked ops
# Math.addExact(Integer.MAX_VALUE, 1)   // throws ArithmeticException: integer overflow

# Floating-point pitfalls
# 0.1 + 0.2 == 0.3        // false  (IEEE 754)
# Math.abs(a - b) < 1e-9  // safe equality
# Double.NaN == Double.NaN          // false
# Double.isNaN(x); Double.isInfinite(x)
# Float / Double POSITIVE_INFINITY, NEGATIVE_INFINITY, NaN, MIN_VALUE (smallest positive!)

# BigInteger — arbitrary-precision integers
# BigInteger n = new BigInteger("123456789012345678901234567890");
# n.add(BigInteger.ONE); n.multiply(other); n.mod(m); n.modPow(e, m);
# n.gcd(other); n.isProbablePrime(40);
# BigInteger.valueOf(42)         // small ints — cached

# BigDecimal — arbitrary-precision decimal — USE FOR MONEY
# BigDecimal price = new BigDecimal("19.99");        // String ctor — exact
# new BigDecimal(0.1)                                 // BAD: 0.1000000000000000055...
# BigDecimal.valueOf(0.1)                             // OK: uses Double.toString -> "0.1"
# price.add(tax).setScale(2, RoundingMode.HALF_EVEN)
# RoundingMode: UP, DOWN, CEILING, FLOOR, HALF_UP, HALF_DOWN, HALF_EVEN (banker's), UNNECESSARY

# Math (java.lang.Math) — common methods
# abs, signum, min, max, floor, ceil, round, rint
# pow, sqrt, cbrt, exp, log, log10, log1p
# sin, cos, tan, asin, acos, atan, atan2, sinh, cosh, tanh
# toRadians, toDegrees, hypot, IEEEremainder, fma
# random()                          // [0.0, 1.0) — for serious use ThreadLocalRandom or SecureRandom
# floorDiv(a,b); floorMod(a,b)      // sign of divisor (matches Python %)

# StrictMath — bit-for-bit reproducible (slower)
# ThreadLocalRandom.current().nextInt(0, 100)
# SecureRandom — cryptographic
```

## Arrays vs Collections

```bash
# Arrays — fixed size, type-erased generics not allowed, primitive specialisations
# int[] a = new int[10];          // zeroed
# int[] b = {1, 2, 3};            // initializer
# int[][] grid = new int[3][3];   // 2D
# a.length;                       // field, NOT method
# Arrays.fill(a, 0); Arrays.sort(a); Arrays.binarySearch(a, 5);
# Arrays.equals(a, b); Arrays.hashCode(a); Arrays.toString(a);
# Arrays.copyOf(a, 20); Arrays.copyOfRange(a, 1, 5);
# Arrays.stream(a)                // IntStream

# When to use which:
# Array      — fixed size, primitive perf, low-level / interop / hot paths
# ArrayList  — default mutable list, random access O(1), append amortized O(1)
# LinkedList — rarely the right answer; only if you need queue/deque AND removal mid-list
# HashMap    — default key-value, O(1) avg
# TreeMap    — sorted, O(log n)
# HashSet    — default unique-set, O(1) avg
```

## Control Flow

```bash
# if / else if / else
# if (x > 0) { ... } else if (x == 0) { ... } else { ... }

# Ternary
# String s = n >= 0 ? "pos" : "neg";

# Classic switch (statement)
# switch (day) {
#   case MON: case TUE: doWeekday(); break;
#   case SAT: case SUN: doWeekend(); break;
#   default: throw new AssertionError();
# }

# Arrow / expression switch (14+) — no fall-through, exhaustive
# int n = switch (day) {
#   case MON, TUE, WED, THU, FRI -> 8;
#   case SAT, SUN                -> 0;
# };
# // For block bodies with logic, use yield:
# String label = switch (s) {
#   case "a" -> "alpha";
#   case "b" -> {
#     log.info("b chosen");
#     yield "bravo";
#   }
#   default  -> "other";
# };

# Pattern matching for switch (21+)
# String s = switch (obj) {
#   case Integer i when i > 0 -> "pos int " + i;
#   case Integer i            -> "non-pos int " + i;
#   case String str           -> "string " + str;
#   case null                 -> "null";
#   default                   -> "other";
# };

# Pattern matching for instanceof (16+)
# if (obj instanceof String s && !s.isBlank()) {
#   System.out.println(s.toUpperCase());
# }

# Common switch ERROR
#   error: 'switch' expression does not cover all possible input values
# Fix: add `default ->` or, for sealed types, ensure all permitted subtypes are covered.
```

## Loops

```bash
# Classic for
# for (int i = 0; i < n; i++) { ... }

# Enhanced-for (foreach) — works on Iterable<T> and arrays
# for (var item : list) { ... }
# for (var entry : map.entrySet()) {
#     System.out.println(entry.getKey() + "=" + entry.getValue());
# }

# while / do-while
# while (cond) { ... }
# do { ... } while (cond);

# break / continue
# for (int i = 0; i < n; i++) {
#     if (skip(i)) continue;
#     if (done(i)) break;
# }

# Labels — break out of nested loops
# outer:
# for (int i = 0; i < n; i++) {
#     for (int j = 0; j < m; j++) {
#         if (grid[i][j] == TARGET) break outer;
#         if (grid[i][j] == SKIP)   continue outer;
#     }
# }

# Iterator (manual)
# Iterator<String> it = list.iterator();
# while (it.hasNext()) { String s = it.next(); if (s.isBlank()) it.remove(); }
```

## Methods

```bash
# Visibility   — public | protected | (package-private — no keyword) | private
# Modifiers    — static, final, abstract, synchronized, native, strictfp, default
# Signature    — name + parameter TYPES (return type and throws not part of signature)

# Overloading — same name, different parameters
# void log(String s) { ... }
# void log(String s, Throwable t) { ... }
# void log(int code, String s) { ... }

# Varargs — Type... must be LAST parameter; treated as Type[] inside method
# void printf(String fmt, Object... args) { System.out.printf(fmt, args); }
# printf("%d / %d%n", a, b);           // OK
# printf("%s%n", (Object) null);       // careful: null Object[] vs null Object
# // Warning: Possible heap pollution from parameterized vararg type
# // Add @SafeVarargs on final/static/private methods to suppress.

# No default arguments — use overloads or builders
# void send(Msg m)                          { send(m, 30, true); }
# void send(Msg m, int timeoutSec)          { send(m, timeoutSec, true); }
# void send(Msg m, int timeoutSec, boolean retry) { ... }

# Return / multiple return — wrap in record or array (Java has no tuple)
# record Pair<A,B>(A first, B second) {}
# Pair<Integer,String> getResult() { return new Pair<>(200, "OK"); }

# main signature — exact
# public static void main(String[] args) { ... }
```

## Classes

```bash
# Top-level class — only one public per file
# package com.example;
#
# public class Account {
#     private final String owner;       // final field — must init in constructor
#     private long balance;             // mutable field
#     private static final long FEE = 100;  // class-level constant
#
#     public Account(String owner) {
#         this.owner = owner;
#     }
#
#     public Account(String owner, long opening) {
#         this(owner);                  // ctor delegation — must be FIRST stmt
#         this.balance = opening;
#     }
#
#     public long getBalance()        { return balance; }
#     public void deposit(long n)     { this.balance += n; }
#
#     @Override public String toString() { return "Account[" + owner + "]"; }
# }

# abstract — partial impl, cannot be instantiated
# public abstract class Shape {
#     public abstract double area();
#     public double perimeter() { return 0; }
# }

# final — cannot subclass / override / reassign
# public final class Money { ... }      // class final
# public final void run() { ... }       // method final

# static nested vs inner class
# class Outer {
#     static class Nested { /* no implicit Outer reference */ }
#     class Inner        { /* implicit Outer.this reference */ }
#     void method() {
#         class Local { /* method-local */ }
#         Runnable r = new Runnable() { @Override public void run(){} }; // anonymous
#         Runnable r2 = () -> { };                                       // lambda
#     }
# }

# Initializer blocks
# class C {
#     static { /* runs once when class loaded */ }
#     {        /* runs before each ctor */          }
# }
```

## Records

```bash
# Records (16+) — concise immutable data carriers
# auto-generated: private final fields, accessors (no get prefix), equals/hashCode/toString, canonical ctor
# public record Point(double x, double y) {}
# Point p = new Point(1.0, 2.0);
# p.x();      // accessor — note: x(), NOT getX()
# p.equals(new Point(1.0, 2.0));     // true — value-based equality
# System.out.println(p);             // Point[x=1.0, y=2.0]

# Compact constructor — for validation; assignments to fields are implicit
# public record Range(int lo, int hi) {
#     public Range {
#         if (lo > hi) throw new IllegalArgumentException("lo > hi");
#     }
# }

# Add methods, static methods, static factories
# public record Vec(double x, double y) {
#     public double mag() { return Math.hypot(x, y); }
#     public static Vec zero() { return new Vec(0, 0); }
# }

# Records can implement interfaces but NOT extend classes (always final, extend Record)
# Records cannot declare instance fields outside the header.

# Record patterns (21+)
# if (obj instanceof Point(double x, double y)) { ... }
# switch (shape) {
#   case Circle(double r)     -> Math.PI * r * r;
#   case Square(double side)  -> side * side;
# }
```

## Sealed Classes & Interfaces

```bash
# Sealed (17+) — restrict who may extend / implement
# public sealed interface Shape permits Circle, Square, Triangle {}
# public record Circle(double r) implements Shape {}
# public record Square(double s) implements Shape {}
# public final class Triangle implements Shape { ... }

# Permitted subclasses must be:
#   - in same module (or unnamed module if non-modular)
#   - in same package if non-modular
#   - declared final, sealed, or non-sealed (explicit)

# non-sealed — re-opens the hierarchy at that point
# public non-sealed class Triangle implements Shape { ... }

# Sealed + pattern-matching switch is exhaustive (no default needed)
# double area(Shape s) {
#     return switch (s) {
#         case Circle c   -> Math.PI * c.r() * c.r();
#         case Square sq  -> sq.s() * sq.s();
#         case Triangle t -> t.area();
#     };
# }

# Common error if not exhaustive:
#   error: the switch expression does not cover all possible input values
```

## Interfaces

```bash
# Interface — public abstract methods by default
# public interface Repository<T> {
#     T findById(long id);                       // implicitly public abstract
#     List<T> findAll();
#
#     // default method (8+) — provides body, may be overridden
#     default Optional<T> findOptional(long id) {
#         return Optional.ofNullable(findById(id));
#     }
#
#     // static method (8+) — utility, NOT inherited
#     static <T> Repository<T> empty() { return new Empty<>(); }
#
#     // private method (9+) — code-share between defaults
#     private void log(String m) { System.out.println(m); }
#
#     // private static (9+)
#     private static <T> T nonNull(T v) { return Objects.requireNonNull(v); }
# }

# Interfaces can be sealed too
# public sealed interface Animal permits Dog, Cat {}

# Functional interface — exactly ONE abstract method
# @FunctionalInterface
# public interface Mapper<A,B> { B map(A a); }
# // @FunctionalInterface causes javac error if more than one abstract method.

# Marker interface (no methods) — Serializable, Cloneable, RandomAccess
```

## Generics

```bash
# Type parameter — single uppercase letter convention: T, E, K, V, R, N
# public class Box<T> {
#     private T value;
#     public T get()         { return value; }
#     public void set(T v)   { this.value = v; }
# }
# Box<String> b = new Box<>();   // diamond operator (7+)

# Bounded type parameter
# <T extends Number>                  // T must be Number or subclass
# <T extends Comparable<T>>           // T must be self-comparable
# <T extends Number & Comparable<T>>  // multiple bounds with &

# Wildcards
# List<? extends Number>   // covariant — read Numbers, can't add (except null)
# List<? super Integer>    // contravariant — write Integers, read as Object
# List<?>                  // unbounded — read as Object only

# PECS — Producer Extends, Consumer Super
#   Source / produces values  -> ? extends T
#   Sink   / consumes values  -> ? super   T
# public static <T> void copy(List<? extends T> src, List<? super T> dst) {
#     for (T t : src) dst.add(t);
# }

# Type erasure — generics ERASED at runtime
# new ArrayList<String>().getClass() == new ArrayList<Integer>().getClass();   // true
# // You cannot do:
# // T t = new T();              // error: type parameter cannot be instantiated
# // if (x instanceof List<String>) ...   // error: cannot perform instanceof check
# // T[] arr = new T[10];        // error: generic array creation
# Workarounds: pass a Class<T> token, use a Supplier<T>, or use @SafeVarargs.

# Generic method
# public static <T> T firstOrNull(List<T> xs) { return xs.isEmpty() ? null : xs.get(0); }
# String s = Util.<String>firstOrNull(list);     // explicit witness (rarely needed)
```

## Enums

```bash
# Enum — type-safe constants
# public enum Day { MON, TUE, WED, THU, FRI, SAT, SUN }
# Day.MON.name();          // "MON"
# Day.MON.ordinal();       // 0  — DON'T persist; ordering may change
# Day.valueOf("MON");      // Day.MON  — throws IllegalArgumentException if absent
# Day.values();            // Day[] — fresh array each call (cache if hot)

# Enum with fields/ctor
# public enum Status {
#     ACTIVE(1, "active"),
#     INACTIVE(0, "inactive");
#     private final int code;
#     private final String label;
#     Status(int code, String label) { this.code = code; this.label = label; }
#     public int code() { return code; }
# }

# Methods per constant — anonymous body per constant
# public enum Op {
#     PLUS  { public int apply(int a, int b) { return a + b; } },
#     MINUS { public int apply(int a, int b) { return a - b; } };
#     public abstract int apply(int a, int b);
# }

# EnumSet / EnumMap — bitset / array-backed; faster than HashSet/HashMap
# EnumSet<Day> weekend = EnumSet.of(Day.SAT, Day.SUN);
# EnumSet<Day> weekday = EnumSet.complementOf(weekend);
# EnumMap<Day, String> tasks = new EnumMap<>(Day.class);
# tasks.put(Day.MON, "deploy");
```

## Annotations

```bash
# Built-in annotations
# @Override            — verifies you actually override; javac error if not
# @Deprecated          — generates warning; @Deprecated(since="1.5", forRemoval=true)
# @SuppressWarnings("unchecked")
# @FunctionalInterface — verifies exactly one abstract method
# @SafeVarargs         — suppresses heap-pollution warning (final/static/private only)

# Declaration of custom annotation
# import java.lang.annotation.*;
#
# @Retention(RetentionPolicy.RUNTIME)        // SOURCE | CLASS | RUNTIME
# @Target({ElementType.METHOD, ElementType.TYPE})
# @Documented
# @Inherited
# public @interface Audited {
#     String value() default "";
#     String[] tags() default {};
#     Class<? extends Throwable> ignore() default RuntimeException.class;
# }
#
# // Use:
# @Audited(value = "important", tags = {"a","b"})
# public void doWork() { ... }
#
# // Read at runtime via reflection:
# Audited a = method.getAnnotation(Audited.class);
# if (a != null) System.out.println(a.value());

# Common @Target ElementTypes:
#   TYPE, FIELD, METHOD, PARAMETER, CONSTRUCTOR, LOCAL_VARIABLE, ANNOTATION_TYPE,
#   PACKAGE, TYPE_PARAMETER, TYPE_USE, MODULE
```

## Lambdas & Functional Interfaces

```bash
# Lambda syntax
# () -> 42
# x -> x * 2
# (x, y) -> x + y
# (int x, int y) -> { return x + y; }       // explicit types + block body

# Standard functional interfaces (java.util.function)
# Function<T,R>          R apply(T t)            // x -> x*2
# BiFunction<T,U,R>      R apply(T,U)
# Consumer<T>            void accept(T)          // System.out::println
# BiConsumer<T,U>        void accept(T,U)
# Supplier<T>            T get()                 // () -> new ArrayList<>()
# Predicate<T>           boolean test(T)         // s -> s.isEmpty()
# BiPredicate<T,U>
# UnaryOperator<T>       T apply(T)              // String::trim
# BinaryOperator<T>      T apply(T,T)            // Integer::sum
# Runnable               void run()
# Callable<V>            V call() throws Exception
# Comparator<T>          int compare(T,T)

# Specialized primitive functional interfaces (avoid boxing)
# IntFunction, IntPredicate, IntUnaryOperator, IntBinaryOperator, IntConsumer, IntSupplier
# (and the Long, Double variants); ToIntFunction<T>, ToLongFunction<T>, ToDoubleFunction<T>

# Method references (::)
# String::toUpperCase            // unbound instance method
# System.out::println            // bound instance method
# Integer::parseInt              // static method
# ArrayList::new                 // constructor
# String[]::new                  // array constructor

# Effectively final capture — local vars used inside lambda must be final or never reassigned
# int n = 10;                    // effectively final
# Runnable r = () -> System.out.println(n);
# // n = 20;   // ERROR if uncommented:
# //   error: local variables referenced from a lambda expression must be final or effectively final
```

## Streams

```bash
# Source -> 0+ intermediate ops -> 1 terminal op
# Streams are LAZY (intermediate) and SINGLE-USE (consumed by terminal op).

# Sources
# list.stream()
# Stream.of(1,2,3)
# Arrays.stream(arr)
# IntStream.range(0, 10)              // 0..9
# IntStream.rangeClosed(1, 10)        // 1..10
# Stream.iterate(1, x -> x*2).limit(10)
# Stream.generate(Math::random).limit(5)
# Files.lines(path)                   // close with try-with-resources
# Pattern.compile(",").splitAsStream(s)

# Intermediate ops (return Stream)
# .filter(s -> s.length() > 3)
# .map(String::toUpperCase)
# .mapToInt(String::length)           // Stream<String> -> IntStream
# .flatMap(line -> Arrays.stream(line.split(" ")))
# .distinct()                         // uses .equals/.hashCode
# .sorted() / .sorted(comparator)
# .peek(System.out::println)          // debug
# .limit(10) / .skip(5)
# .takeWhile(x -> x > 0)              // 9+
# .dropWhile(x -> x > 0)              // 9+

# Terminal ops
# .count()
# .toList()                           // 16+ — UNMODIFIABLE
# .collect(Collectors.toList())       // mutable ArrayList (most collectors)
# .collect(Collectors.toUnmodifiableList())
# .collect(Collectors.toSet())
# .reduce(0, Integer::sum)
# .reduce((a,b) -> a + b)             // Optional<T>
# .forEach(System.out::println)
# .forEachOrdered(...)                // honor encounter order
# .anyMatch(p) / .allMatch(p) / .noneMatch(p)
# .findFirst() / .findAny()           // Optional<T>
# .min(comparator) / .max(comparator)

# Collectors
# Collectors.toList(), toSet(), toUnmodifiableList()
# Collectors.toMap(User::id, Function.identity())
# Collectors.toMap(k, v, (a,b) -> a)            // merge fn for duplicate keys
# Collectors.groupingBy(User::dept)
# Collectors.groupingBy(User::dept, Collectors.counting())
# Collectors.groupingBy(User::dept, Collectors.mapping(User::name, toList()))
# Collectors.partitioningBy(u -> u.salary() > 100_000)
# Collectors.joining(", ", "[", "]")
# Collectors.summingInt(User::age)
# Collectors.averagingDouble(User::salary)
# Collectors.summarizingInt(User::age)          // count/sum/min/avg/max
# Collectors.teeing(c1, c2, merger)             // 12+

# Parallel streams — fork-join. Only worth it for CPU-bound + large + stateless + no shared mutable state.
# list.parallelStream().filter(...).count();

# Common error
#   java.lang.IllegalStateException: stream has already been operated upon or closed
# Fix: build a fresh stream each time; don't store stream references.
```

## Optional

```bash
# Optional<T> — represents possibly-absent value; ONLY for return types of methods/streams.
# DON'T use as field, parameter, or for collections (use empty collection / Map.containsKey).

# Construct
# Optional.of(value)               // value MUST be non-null, else NPE
# Optional.ofNullable(maybeNull)
# Optional.empty()

# Consume
# o.isPresent(); o.isEmpty()                 // 11+
# o.ifPresent(System.out::println)
# o.ifPresentOrElse(v -> ..., () -> ...)     // 9+
# o.orElse(defaultValue)
# o.orElseGet(() -> expensiveDefault())      // lazy
# o.orElseThrow()                            // 10+ — NoSuchElementException
# o.orElseThrow(() -> new MyException("..."))
# o.or(() -> Optional.of(...))               // 9+ — fallback Optional

# Transform
# o.map(String::toUpperCase)
# o.flatMap(this::lookup)
# o.filter(s -> !s.isEmpty())
# o.stream()                                  // 9+ — empty or single-element

# Anti-pattern (defeats the purpose)
# if (o.isPresent()) { use(o.get()); }       // works but clumsy — prefer ifPresent / map
```

## Exceptions

```bash
# Throwable
#   Error               — JVM/system; do NOT catch (OutOfMemoryError, StackOverflowError)
#   Exception
#     RuntimeException  — UNCHECKED  (NullPointerException, IllegalArgumentException, ...)
#     <other>           — CHECKED    (IOException, SQLException, ClassNotFoundException)

# Checked exceptions MUST be declared (throws) or caught.
# Unchecked may be thrown anywhere; need not be declared.

# try / catch / finally
# try {
#     riskyIO();
# } catch (FileNotFoundException e) {
#     log.warn("missing", e);
# } catch (IOException | SQLException e) {     // multi-catch (7+)
#     throw new RuntimeException(e);
# } finally {
#     cleanup();                                // ALWAYS runs (except System.exit / JVM crash)
# }

# try-with-resources — auto-closes anything implementing AutoCloseable
# try (var in  = Files.newBufferedReader(path);
#      var out = Files.newBufferedWriter(target)) {
#     in.transferTo(out);
# }
# // Suppressed exceptions (close throws while body throws) attached via Throwable.addSuppressed.

# AutoCloseable
# public class MyRes implements AutoCloseable {
#     @Override public void close() { ... }
# }

# Custom exception
# public class NotFoundException extends RuntimeException {
#     public NotFoundException(String msg)             { super(msg); }
#     public NotFoundException(String msg, Throwable t){ super(msg, t); }
# }

# Wrap & re-throw with cause
# try { ... } catch (IOException e) {
#     throw new ServiceException("failed", e);    // sets cause
# }
# // Or: throw (RuntimeException) new RuntimeException("x").initCause(e);

# Common errors
#   error: unreported exception java.io.IOException; must be caught or declared to be thrown
#   error: exception java.io.FileNotFoundException is never thrown in body of corresponding try statement
```

## Collections Framework

```bash
# Hierarchy
#   Collection
#     List       (indexed, duplicates allowed)
#     Set        (no duplicates)
#       SortedSet
#         NavigableSet
#     Queue
#       Deque
#   Map (NOT a Collection)
#     SortedMap -> NavigableMap

# List
# new ArrayList<>(initialCapacity)        // O(1) random access, O(1)* append, O(n) middle insert
# new LinkedList<>()                       // O(n) random access, O(1) head/tail (rarely the right answer)
# List.of(a, b, c)                         // immutable (9+); null-hostile
# List.copyOf(list)                        // immutable snapshot
# Collections.unmodifiableList(list)       // unmodifiable view

# Set
# new HashSet<>()                          // O(1) avg, no order
# new LinkedHashSet<>()                    // O(1) avg, insertion order
# new TreeSet<>(comparator)                // O(log n), sorted
# Set.of(...) / Set.copyOf(...)            // immutable (9+)

# Map
# new HashMap<>()                          // O(1) avg, no order
# new LinkedHashMap<>()                    // O(1) avg, insertion order; LRU via accessOrder
# new TreeMap<>()                          // O(log n), sorted by key
# new ConcurrentHashMap<>()                // thread-safe, lock-striped
# Map.of("k1",1, "k2",2)                   // immutable (9+), max 10 entries
# Map.ofEntries(Map.entry(k,v), ...)       // immutable, any size
# Map.copyOf(m)

# Queue / Deque
# new ArrayDeque<>()                       // PREFER OVER Stack and LinkedList
# new PriorityQueue<>(comparator)          // min-heap
# new LinkedBlockingQueue<>(capacity)      // bounded, blocking

# Common Map idioms
# m.getOrDefault(k, defaultV)
# m.putIfAbsent(k, v)
# m.computeIfAbsent(k, key -> new ArrayList<>()).add(item)
# m.computeIfPresent(k, (k,v) -> v + 1)
# m.compute(k, (k,v) -> v == null ? 1 : v + 1)
# m.merge(k, 1, Integer::sum)              // counter idiom
# m.forEach((k,v) -> ...)
# m.entrySet().removeIf(e -> ...)

# Iteration safety
# Iterating an ArrayList/HashMap and calling .add/.remove on the underlying collection
# throws java.util.ConcurrentModificationException — fail-fast.
# Fix: use Iterator.remove(), removeIf(predicate), or copy first.
```

## NIO.2 File I/O

```bash
# Path / Paths (use Path.of since 11)
# Path p = Path.of("/var/log/app.log");
# Path p2 = Path.of("dir", "sub", "file.txt");
# p.getFileName(); p.getParent(); p.getRoot(); p.toAbsolutePath(); p.normalize();
# p.resolve("child.txt"); p.relativize(other);

# Files — utility class
# Files.exists(p); Files.isRegularFile(p); Files.isDirectory(p);
# Files.size(p); Files.getLastModifiedTime(p);
# Files.createDirectories(p);
# Files.delete(p); Files.deleteIfExists(p);
# Files.move(src, dst, StandardCopyOption.REPLACE_EXISTING, StandardCopyOption.ATOMIC_MOVE);
# Files.copy(src, dst, StandardCopyOption.REPLACE_EXISTING);

# Read whole file
# byte[] bytes  = Files.readAllBytes(p);
# String text   = Files.readString(p, StandardCharsets.UTF_8);     // 11+
# List<String> lines = Files.readAllLines(p, StandardCharsets.UTF_8);

# Stream of lines (CLOSE — try-with-resources)
# try (Stream<String> lines = Files.lines(p, StandardCharsets.UTF_8)) {
#     long count = lines.filter(l -> l.startsWith("ERROR")).count();
# }

# Write
# Files.writeString(p, "hello\n", StandardCharsets.UTF_8,
#     StandardOpenOption.CREATE, StandardOpenOption.TRUNCATE_EXISTING);
# Files.write(p, lines, StandardOpenOption.APPEND);

# StandardOpenOption: READ, WRITE, APPEND, TRUNCATE_EXISTING, CREATE, CREATE_NEW,
#                     DELETE_ON_CLOSE, SPARSE, SYNC, DSYNC

# Walk tree
# try (Stream<Path> s = Files.walk(root, 5)) {
#     s.filter(Files::isRegularFile).forEach(System.out::println);
# }

# Channels / ByteBuffer (low-level)
# try (FileChannel ch = FileChannel.open(p, StandardOpenOption.READ)) {
#     ByteBuffer buf = ByteBuffer.allocate(8192);
#     while (ch.read(buf) > 0) { buf.flip(); ... ; buf.clear(); }
# }
```

## Threads

```bash
# Thread / Runnable
# Thread t = new Thread(() -> System.out.println("hi"), "worker-1");
# t.start();    // NOT t.run() — run() executes on caller thread

# ExecutorService — high-level
# ExecutorService exec = Executors.newFixedThreadPool(4);
# Future<Integer> f = exec.submit(() -> compute());
# Integer r = f.get(5, TimeUnit.SECONDS);
# exec.shutdown();                  // stop accepting; let queue drain
# exec.shutdownNow();               // attempt cancel + interrupt
# exec.awaitTermination(30, TimeUnit.SECONDS);

# Executors factories
# newFixedThreadPool(n)             // bounded threads, unbounded queue (CAREFUL — OOM)
# newCachedThreadPool()             // unbounded threads, SynchronousQueue (CAREFUL — OOM threads)
# newSingleThreadExecutor()
# newScheduledThreadPool(n)         // schedule with delay/period
# newWorkStealingPool()             // ForkJoinPool, for CPU-bound parallel
# newVirtualThreadPerTaskExecutor() // 21+ — each task on its own virtual thread

# ScheduledExecutorService
# ses.schedule(task, 5, TimeUnit.SECONDS);
# ses.scheduleAtFixedRate(task, 0, 10, TimeUnit.SECONDS);
# ses.scheduleWithFixedDelay(task, 0, 10, TimeUnit.SECONDS);

# Production rule: ALWAYS configure your own ThreadPoolExecutor with bounded queue + reject policy.
# new ThreadPoolExecutor(core, max, idle, TimeUnit.SECONDS,
#     new ArrayBlockingQueue<>(1000),
#     Executors.defaultThreadFactory(),
#     new ThreadPoolExecutor.CallerRunsPolicy());
```

## Virtual Threads

```bash
# Virtual threads (21+) — JEP 444. Lightweight, scheduled by JVM on small set of carrier threads.
# Use for I/O-bound work (HTTP, DB, RPC). NOT useful for CPU-bound work.

# Create directly
# Thread.ofVirtual().start(() -> doWork());
# Thread.ofVirtual().name("v-", 0).start(() -> ...);
# Thread t = Thread.startVirtualThread(() -> ...);

# ExecutorService — one virtual thread per task
# try (var exec = Executors.newVirtualThreadPerTaskExecutor()) {
#     for (var url : urls) exec.submit(() -> fetch(url));
# }
# // close() blocks until all tasks finish.

# Pinning — virtual thread cannot unmount carrier inside synchronized blocks;
# prefer ReentrantLock for I/O paths.
# JFR event: jdk.VirtualThreadPinned

# DON'T pool virtual threads — they're cheap, create one per task.
# DON'T use ThreadLocal heavily on virtual threads — can leak memory at scale.
```

## CompletableFuture

```bash
# Build async pipelines without blocking. Default executor: ForkJoinPool.commonPool.

# Start
# CompletableFuture<String> f = CompletableFuture.supplyAsync(() -> fetch(url));
# CompletableFuture<Void>   v = CompletableFuture.runAsync(() -> doWork());

# Chain (pure transform — same thread by default)
# .thenApply(s  -> parse(s))                   // T -> R
# .thenAccept(r -> log(r))                     // T -> void
# .thenRun(()   -> log("done"))                // -> void (ignores result)

# Async variants (.thenApplyAsync, ...) hop to another thread / executor.

# Compose (flatMap of futures)
# .thenCompose(user -> fetchProfileAsync(user.id()))

# Combine
# CompletableFuture<C> ab = a.thenCombine(b, (x,y) -> merge(x,y));
# CompletableFuture<Void> all = CompletableFuture.allOf(f1, f2, f3);
# all.thenRun(() -> System.out.println(f1.join() + f2.join()));
# CompletableFuture<Object> any = CompletableFuture.anyOf(f1, f2, f3);

# Errors
# .exceptionally(ex -> defaultValue)
# .handle((res, ex) -> ex == null ? res : recover(ex))
# .whenComplete((res, ex) -> log(res, ex))     // side-effect

# Timeout (9+)
# .orTimeout(2, TimeUnit.SECONDS)              // completeExceptionally on timeout
# .completeOnTimeout(default, 2, TimeUnit.SECONDS)

# Block (last resort)
# String r = f.get();                          // throws checked ExecutionException
# String r = f.join();                         // throws unchecked CompletionException

# Always pass a custom Executor for I/O
# var io = Executors.newVirtualThreadPerTaskExecutor();
# CompletableFuture.supplyAsync(() -> fetch(url), io);
```

## Synchronization

```bash
# synchronized — intrinsic monitor lock; on object instance OR Class for static
# private final Object lock = new Object();
# synchronized (lock) { ... }
# public synchronized void method() { ... }                  // locks 'this'
# public static synchronized void method() { ... }           // locks Class<>

# ReentrantLock — explicit, supports tryLock, fairness, interruptible, condition vars
# private final ReentrantLock lock = new ReentrantLock(true);   // fair
# lock.lock();
# try { ... } finally { lock.unlock(); }
# if (lock.tryLock(1, TimeUnit.SECONDS)) { try {...} finally {lock.unlock();} }

# Condition (replaces wait/notify)
# Condition notEmpty = lock.newCondition();
# lock.lock();
# try {
#     while (queue.isEmpty()) notEmpty.await();
#     return queue.removeFirst();
# } finally { lock.unlock(); }

# ReentrantReadWriteLock — multiple readers OR one writer
# var rw = new ReentrantReadWriteLock();
# rw.readLock().lock();   try { ... } finally { rw.readLock().unlock(); }
# rw.writeLock().lock();  try { ... } finally { rw.writeLock().unlock(); }

# Semaphore — limit concurrent access
# Semaphore sem = new Semaphore(5);
# sem.acquire();   try { ... } finally { sem.release(); }

# CountDownLatch — one-time gate
# CountDownLatch latch = new CountDownLatch(N);
# // worker:   latch.countDown();
# // waiter:   latch.await();      latch.await(30, TimeUnit.SECONDS);

# CyclicBarrier — N threads sync at a barrier, reusable
# CyclicBarrier barrier = new CyclicBarrier(N, () -> System.out.println("phase done"));
# barrier.await();

# Phaser — flexible barrier with dynamic registration
# Phaser ph = new Phaser(1);
# ph.register();
# ph.arriveAndAwaitAdvance();
# ph.arriveAndDeregister();
```

## Atomics

```bash
# java.util.concurrent.atomic — lock-free CAS
# AtomicInteger n = new AtomicInteger(0);
# n.incrementAndGet();             // ++n
# n.getAndIncrement();             // n++
# n.compareAndSet(expect, update); // CAS — true on success
# n.updateAndGet(x -> x * 2);
# n.accumulateAndGet(5, Integer::sum);

# AtomicLong, AtomicBoolean, AtomicReference<V>
# AtomicReference<State> ref = new AtomicReference<>(State.INIT);
# ref.compareAndSet(State.INIT, State.READY);

# AtomicIntegerArray, AtomicLongArray, AtomicReferenceArray<E>
# AtomicReferenceFieldUpdater — update specific volatile field via reflection

# LongAdder / DoubleAdder / LongAccumulator — high-contention counters; faster than AtomicLong
# LongAdder hits = new LongAdder();
# hits.increment();
# long total = hits.sum();         // approximate during concurrent updates

# volatile — read/write seen by all threads, prevents reordering, NOT atomic for r-m-w
# private volatile boolean shutdown = false;
# // OK for one-writer flag, NOT for counters (use AtomicInteger).
```

## Memory Model Essentials

```bash
# JMM (JSR-133) defines happens-before relations:
#   - program order within a thread
#   - monitor exit happens-before subsequent monitor enter (synchronized)
#   - volatile write happens-before subsequent volatile read of same field
#   - Thread.start happens-before all actions in started thread
#   - all actions in a thread happen-before any other thread's join()
#   - constructor happens-before finalizer

# Consequence: data published via synchronized / volatile / final / Atomic / Lock IS visible.
# Plain field writes are NOT guaranteed to be seen.

# Double-checked locking — broken WITHOUT volatile
# private static Singleton instance;        // BROKEN — partial-construction visible
# public static Singleton get() {
#     if (instance == null) {
#         synchronized (Singleton.class) {
#             if (instance == null) instance = new Singleton();
#         }
#     }
#     return instance;
# }

# Fix 1: volatile
# private static volatile Singleton instance;

# Fix 2 (preferred): initialization-on-demand holder idiom
# public class Singleton {
#     private Singleton() {}
#     private static class Holder { static final Singleton INSTANCE = new Singleton(); }
#     public static Singleton get() { return Holder.INSTANCE; }
# }
# // Class init is thread-safe per JLS — no volatile/synchronized needed.

# final fields — values seen by ANY thread once constructor returns,
# provided the reference doesn't escape during construction.
```

## Reflection & MethodHandles

```bash
# Class<?> token
# Class<?> c = Foo.class;
# Class<?> c = Class.forName("com.example.Foo");
# Class<?> c = obj.getClass();

# Inspect
# c.getName(); c.getSimpleName(); c.getSuperclass(); c.getInterfaces();
# c.getDeclaredFields();  c.getDeclaredMethods();  c.getDeclaredConstructors();
# c.getFields(); c.getMethods();                   // public, including inherited

# Instantiate
# Object o = c.getDeclaredConstructor().newInstance();

# Invoke
# Method m = c.getDeclaredMethod("greet", String.class);
# m.setAccessible(true);                            // suppress access checks (with --add-opens)
# String r = (String) m.invoke(obj, "world");

# Field
# Field f = c.getDeclaredField("count");
# f.setAccessible(true);
# int v = (int) f.get(obj); f.setInt(obj, 5);

# Common errors with modules:
#   java.lang.reflect.InaccessibleObjectException: Unable to make ... accessible:
#   module java.base does not "opens java.lang" to unnamed module
# Fix: add JVM arg `--add-opens java.base/java.lang=ALL-UNNAMED`

# MethodHandle (faster, type-checked at link)
# MethodHandles.Lookup lk = MethodHandles.lookup();
# MethodHandle mh = lk.findVirtual(String.class, "length", MethodType.methodType(int.class));
# int n = (int) mh.invokeExact("hello");
```

## JDBC

```bash
# DriverManager (simple) / DataSource (production with pooling)
# Class.forName("org.postgresql.Driver");           // not needed since JDBC 4 / Java 6
# String url = "jdbc:postgresql://localhost:5432/mydb";

# try-with-resources for everything
# try (Connection cx = DriverManager.getConnection(url, "user", "pass");
#      PreparedStatement ps = cx.prepareStatement(
#          "SELECT id, name FROM users WHERE active = ? AND created > ?")) {
#
#     ps.setBoolean(1, true);
#     ps.setTimestamp(2, Timestamp.from(Instant.now().minus(7, ChronoUnit.DAYS)));
#
#     try (ResultSet rs = ps.executeQuery()) {
#         while (rs.next()) {
#             long id     = rs.getLong("id");
#             String name = rs.getString("name");
#             // process
#         }
#     }
# }

# ALWAYS use PreparedStatement, never string-concat SQL — prevents injection.
# rs.getXxx(int idx) — 1-based; rs.getXxx(String column) — by name (slower).
# Use rs.wasNull() AFTER reading a primitive to detect SQL NULL.

# Transactions
# cx.setAutoCommit(false);
# try {
#     ps1.executeUpdate();
#     ps2.executeUpdate();
#     cx.commit();
# } catch (SQLException e) {
#     cx.rollback();
#     throw e;
# }

# Batch
# for (var item : items) {
#     ps.setLong(1, item.id());
#     ps.addBatch();
# }
# int[] counts = ps.executeBatch();

# DataSource (HikariCP) for pooling
# HikariConfig cfg = new HikariConfig();
# cfg.setJdbcUrl(url); cfg.setUsername("u"); cfg.setPassword("p");
# cfg.setMaximumPoolSize(20);
# DataSource ds = new HikariDataSource(cfg);
# try (Connection cx = ds.getConnection()) { ... }
```

## Modules (JPMS)

```bash
# module-info.java — at top of source tree (src/main/java/module-info.java)
# module com.example.app {
#     requires java.net.http;          // depend on JDK module
#     requires com.example.core;        // depend on app module
#     requires transitive com.example.api;   // re-export to consumers
#     requires static  org.junit;       // compile-only (no runtime requirement)
#
#     exports com.example.app.api;                       // public for everyone
#     exports com.example.app.internal to com.example.tests; // qualified export
#
#     opens com.example.app.models;                      // reflective open at runtime
#     opens com.example.app.beans to com.fasterxml.jackson.databind;
#
#     provides com.example.spi.Plugin with com.example.app.DefaultPlugin;
#     uses com.example.spi.Plugin;
# }

# Build
# javac -d out/com.example.app $(find src/main/java -name '*.java')
# jar --create --file mods/com.example.app.jar --module-version 1.0 -C out/com.example.app .

# Run
# java -p mods -m com.example.app/com.example.app.Main

# Classpath vs Module Path
# - Classpath  : flat — all classes visible, no encapsulation
# - Modulepath : explicit dependencies, strong encapsulation (only `exports` visible)
# - Automatic module: a JAR on modulepath without module-info — derives module name from JAR filename.
# - Unnamed module: a JAR on classpath — reads all modules, but cannot be required.

# Common errors
#   error: package x.y is declared in module a, but module b does not read it
#       -> add `requires a;` to b's module-info
#   error: module a does not export package x.y
#       -> add `exports x.y;` to a's module-info, or use --add-exports at runtime
#   java.lang.reflect.InaccessibleObjectException
#       -> add `opens x.y;` or pass --add-opens
```

## Packaging

```bash
# JAR — zip with META-INF/MANIFEST.MF
# jar cf app.jar -C out .                      # plain JAR
# jar cfe app.jar com.example.Main -C out .    # executable: sets Main-Class
# jar tf app.jar                               # list contents
# jar xf app.jar                               # extract

# Manifest entries
# Main-Class:     com.example.Main
# Class-Path:     lib/guava.jar lib/jackson.jar
# Multi-Release:  true                         # MR-JAR (9+) — versioned classes under META-INF/versions/N/

# Run executable JAR
# java -jar app.jar

# Fat / Uber JAR — bundle ALL deps
#   Maven Shade Plugin    — relocates/merges service files
#   Gradle Shadow Plugin  — same idea
# Pros: single-file deploy
# Cons: licenses bundled; lose service-file merging if mis-configured

# jlink — custom runtime image (smaller than full JDK)
# jlink --module-path $JAVA_HOME/jmods:mods \
#       --add-modules com.example.app \
#       --launcher app=com.example.app/com.example.Main \
#       --output dist --strip-debug --compress 2 --no-header-files --no-man-pages
# dist/bin/app

# jpackage — native installers (14+)
# jpackage --type pkg --name MyApp --input out --main-jar app.jar --main-class com.example.Main
# # types: app-image (no installer), pkg/dmg (mac), msi/exe (windows), deb/rpm (linux)
```

## Logging

```bash
# Default stack: SLF4J (facade) + Logback (impl). Spring Boot brings this by default.
# Maven:
#   org.slf4j:slf4j-api:2.0.13
#   ch.qos.logback:logback-classic:1.5.6

# Use SLF4J in code
# import org.slf4j.Logger;
# import org.slf4j.LoggerFactory;
# private static final Logger log = LoggerFactory.getLogger(MyClass.class);
#
# log.trace("low detail"); log.debug("..."); log.info("..."); log.warn("..."); log.error("...");
# log.info("user={} action={}", user, action);          // {} placeholder — NO string concat
# log.error("failed to send to {}", endpoint, ex);      // last arg Throwable -> stack trace
# if (log.isDebugEnabled()) log.debug(expensive());     // guard expensive arg construction

# Structured (JSON) logging
# Logback encoder: net.logstash.logback:logstash-logback-encoder
# Add MDC for request correlation
# MDC.put("requestId", id);  try { ... } finally { MDC.clear(); }

# DON'T use System.out / System.err in production code — bypasses logging config.
# DON'T use java.util.logging directly — use SLF4J facade.
```

## Testing

```bash
# JUnit 5 (Jupiter) — de facto default
# import org.junit.jupiter.api.*;
# import static org.junit.jupiter.api.Assertions.*;
#
# class CalcTest {
#     @BeforeAll static void setupOnce() { }
#     @AfterAll  static void teardownOnce() { }
#     @BeforeEach void setup() { }
#     @AfterEach  void teardown() { }
#
#     @Test
#     @DisplayName("adds two numbers")
#     void addsTwoNumbers() {
#         assertEquals(5, Calc.add(2, 3));
#     }
#
#     @Test
#     void throwsOnDivByZero() {
#         var ex = assertThrows(ArithmeticException.class, () -> Calc.div(1, 0));
#         assertTrue(ex.getMessage().contains("zero"));
#     }
#
#     @Test
#     void timesOut() {
#         assertTimeoutPreemptively(Duration.ofSeconds(1), () -> ...);
#     }
#
#     @Disabled("flaky on CI")
#     @Test void todo() { }
# }
#
# // Parameterized
# @ParameterizedTest
# @ValueSource(ints = {1, 2, 3})
# void positive(int n) { assertTrue(n > 0); }
#
# @ParameterizedTest
# @CsvSource({ "1,2,3", "10,20,30" })
# void adds(int a, int b, int sum) { assertEquals(sum, a + b); }
#
# @ParameterizedTest
# @MethodSource("cases")
# void run(String input, int expected) { ... }
# static Stream<Arguments> cases() { return Stream.of(arguments("a", 1), arguments("b", 2)); }

# Common asserts
# assertEquals/assertNotEquals/assertSame/assertNotSame
# assertTrue/assertFalse/assertNull/assertNotNull
# assertArrayEquals/assertIterableEquals/assertLinesMatch
# assertThrows/assertDoesNotThrow
# assertAll(() -> ..., () -> ...)         // grouped — reports all failures

# AssertJ — fluent (much nicer than core Assertions)
# import static org.assertj.core.api.Assertions.*;
# assertThat(list).hasSize(3).contains("a","b").doesNotContain("c");
# assertThat(map).containsEntry("k", 1).hasSize(2);
# assertThatThrownBy(() -> svc.do()).isInstanceOf(MyException.class).hasMessageContaining("nope");

# Mockito
# @ExtendWith(MockitoExtension.class)
# class SvcTest {
#     @Mock Repo repo;
#     @InjectMocks Svc svc;
#
#     @Test void findsUser() {
#         when(repo.findById(1L)).thenReturn(Optional.of(new User(1, "alice")));
#         assertEquals("alice", svc.nameOf(1L));
#         verify(repo, times(1)).findById(1L);
#         verifyNoMoreInteractions(repo);
#     }
# }
# // doThrow(...).when(mock).method();   // for void methods
# // any(), eq(), argThat(...);          // matchers — must wrap ALL args if used

# Run
# mvn test
# mvn -Dtest=CalcTest#addsTwoNumbers test
# ./gradlew test
# ./gradlew test --tests "*.CalcTest.addsTwoNumbers"
```

## JVM Flags

```bash
# Heap sizing
# -Xms512m              # initial heap
# -Xmx4g                # max heap (set Xms == Xmx in production)
# -Xss512k              # per-thread stack
# -XX:MaxDirectMemorySize=2g
# -XX:MaxMetaspaceSize=256m

# GC selection
# -XX:+UseG1GC               # G1 — default since 9, balanced
# -XX:+UseZGC                # ZGC — sub-millisecond pauses, scales to TBs
# -XX:+UseShenandoahGC       # Shenandoah — low pause alt
# -XX:+UseParallelGC         # throughput, batch jobs
# -XX:+UseSerialGC           # tiny / single-thread / containers
# -XX:MaxGCPauseMillis=200   # G1 target pause

# Diagnostics
# -XshowSettings:vm                  # print VM settings on launch
# -XshowSettings:properties
# -XX:+PrintFlagsFinal -version | grep -i gc
# -XX:+UnlockDiagnosticVMOptions
# -XX:+UnlockExperimentalVMOptions

# OOM safety
# -XX:+HeapDumpOnOutOfMemoryError
# -XX:HeapDumpPath=/var/dumps
# -XX:+ExitOnOutOfMemoryError
# -XX:OnOutOfMemoryError="kill -9 %p"

# GC logging (unified, 9+)
# -Xlog:gc*:file=gc.log:time,uptime,level,tags:filecount=10,filesize=100M

# JFR (continuous, low-overhead profiling)
# -XX:StartFlightRecording=duration=60s,filename=rec.jfr,settings=profile
# -XX:FlightRecorderOptions=stackdepth=128
# # or with jcmd:  jcmd <pid> JFR.start name=rec settings=profile

# JIT / Compiler
# -XX:CompileThreshold=10000        # invocations before JIT (default ~10k)
# -XX:-TieredCompilation            # disable tiered (rare)
# -XX:+PrintCompilation             # spammy, useful for tuning
# -Xint                             # interpreted only — debugging
# -Xcomp                            # compile everything immediately

# Container awareness (8u191+, 10+)
# -XX:+UseContainerSupport (default on)
# -XX:MaxRAMPercentage=75.0         # use % of cgroup memory limit
# -XX:InitialRAMPercentage=50.0
```

## Common Compile Errors

```bash
# error: cannot find symbol
#   symbol:   variable foo
#   location: class Bar
# -> typo, missing import, wrong scope, not-yet-declared variable

# error: incompatible types: String cannot be converted to int
# -> wrong type; use Integer.parseInt(s) or fix the declaration

# error: unreported exception java.io.IOException; must be caught or declared to be thrown
# -> wrap call in try/catch, OR add `throws IOException` to enclosing method

# error: not a statement
# -> a line of code that is an expression but not a valid statement
#    (e.g. `x + 1;`, missing `=`, missing semicolon on previous line)

# error: variable x might not have been initialized
# -> local variable used before any definite assignment along all paths

# error: incompatible types: possible lossy conversion from long to int
# -> add explicit cast `(int) value` AFTER you've checked for overflow

# error: <T> is a raw type. References to generic type ArrayList<E> should be parameterized
# -> use ArrayList<String>, not raw ArrayList

# error: attempting to assign weaker access privileges; was public
# -> overriding method must be at least as visible as parent (public stays public)

# error: class X is public, should be declared in a file named X.java
# -> rename file OR class so they match

# error: reached end of file while parsing
# -> missing closing `}`; check brace balance

# error: package x.y does not exist
# -> add module/jar to classpath/modulepath, or fix import

# error: 'switch' expression does not cover all possible input values
# -> add `default ->` or cover all sealed-type permits

# error: lambda expression not expected here
# -> target type isn't a functional interface (e.g. Object o = () -> 1; — invalid)

# warning: [unchecked] unchecked cast
# -> generics and erasure conflict; either redesign or @SuppressWarnings("unchecked")
```

## Common Gotchas

```bash
# 1) == vs .equals on String
# BROKEN
# String a = new String("hi");
# String b = new String("hi");
# if (a == b) { ... }                    // false — different references
# FIXED
# if (a.equals(b)) { ... }
# if ("hi".equals(a)) { ... }            // null-safe
# if (Objects.equals(a, b)) { ... }      // null-safe both sides

# 2) == on boxed Integer (cache 0..127)
# BROKEN
# Integer a = 1000, b = 1000;
# if (a == b) { ... }                    // false (outside cache)
# Integer c = 100,  d = 100;
# if (c == d) { ... }                    // true  (cached)  — accidental success
# FIXED
# if (a.equals(b)) { ... }
# if (Objects.equals(a, b)) { ... }

# 3) Autoboxing NPE in loops
# BROKEN
# Map<String,Integer> m = ...;
# int count = 0;
# for (String k : keys) count += m.get(k);          // NPE if any key absent
# FIXED
# for (String k : keys) count += m.getOrDefault(k, 0);

# 4) ConcurrentModificationException during iteration
# BROKEN
# for (String s : list) {
#     if (s.isEmpty()) list.remove(s);              // CME — fail-fast iterator
# }
# FIXED — Iterator.remove or removeIf
# list.removeIf(String::isEmpty);
# Iterator<String> it = list.iterator();
# while (it.hasNext()) { if (it.next().isEmpty()) it.remove(); }

# 5) Exception swallowing
# BROKEN
# try { riskyIO(); } catch (IOException e) { /* ignored */ }
# FIXED — log with cause and rethrow or handle deliberately
# try { riskyIO(); }
# catch (IOException e) { log.error("riskyIO failed", e); throw new UncheckedIOException(e); }

# 6) String + concat in a loop — quadratic
# BROKEN
# String r = "";
# for (var s : list) r += s;
# FIXED
# var sb = new StringBuilder();
# for (var s : list) sb.append(s);
# String r = sb.toString();
# // or:  String r = String.join("", list);
# // or:  String r = list.stream().collect(Collectors.joining());

# 7) java.util.Date vs java.time.LocalDate
# BROKEN — Date is mutable, broken API, locale issues
# Date d = new Date();          // legacy
# d.setHours(13);               // deprecated
# FIXED — java.time (8+)
# LocalDate today = LocalDate.now();
# LocalDateTime dt = LocalDateTime.now();
# Instant now = Instant.now();              // UTC timestamp

# 8) Forgetting locale in case conversion
# BROKEN
# "TITLE".toLowerCase();        // breaks in Turkish locale: "tıtle"
# FIXED
# "TITLE".toLowerCase(Locale.ROOT);

# 9) Catching Exception (or Throwable)
# BROKEN
# try { ... } catch (Exception e) { ... }   // swallows InterruptedException, RuntimeException, etc
# FIXED — catch narrowest types, or restore interrupt flag for InterruptedException
# } catch (InterruptedException e) {
#     Thread.currentThread().interrupt();
#     throw new RuntimeException(e);
# }

# 10) Treating Optional as nullable container
# BROKEN
# Optional<String> o = ...;
# if (o != null) { ... }                   // Optional should never be null
# FIXED
# o.ifPresent(...); o.orElse(...); o.orElseThrow();

# 11) Returning null collection
# BROKEN
# return null;                              // forces every caller to null-check
# FIXED
# return List.of();                         // empty immutable

# 12) Calling overridable method from constructor
# BROKEN — subclass override sees uninitialized fields
# class Base { Base() { init(); } void init() {} }
# class Sub extends Base { String name = "x"; void init() { name.length(); } }   // NPE
# FIXED — make init() final or private, or call from a factory
```

## Performance Tips

```bash
# Primitive vs wrapper in hot paths
# - Use int[]/long[] over List<Integer> for tight loops; Integer adds 16-byte header + boxing.
# - Use IntStream over Stream<Integer>.
# - For hash maps over int keys, consider Eclipse Collections / fastutil.

# Right collection
# - Default to ArrayList; pre-size with new ArrayList<>(expected) to avoid reallocs.
# - For 1..N elements known at compile time: List.of(...).
# - For frequent contains checks: HashSet, not List.contains (O(n)).
# - For sorted iteration: TreeMap/TreeSet.
# - For high-write concurrency: ConcurrentHashMap, LongAdder.

# Specialized maps when keys are primitive
# - EnumMap (enum keys), IdentityHashMap (identity instead of equals), WeakHashMap (GC keys).

# JIT warmup
# - JIT compiles hot methods after ~10k invocations (-XX:CompileThreshold).
# - Benchmark with JMH (Java Microbenchmark Harness) — handles warmup, deopt, dead-code elim.

# Escape analysis
# - Allocations that don't escape a method may be stack-allocated (no GC pressure).
# - Don't try to outsmart the JIT; profile, don't guess.

# Allocation pressure
# - Reuse buffers (StringBuilder, ByteBuffer) in hot paths.
# - Avoid boxing in tight loops.
# - Prefer ArrayDeque to LinkedList (less per-node allocation).

# Concurrency
# - Pin CPU-bound work to ForkJoinPool or fixed pool sized to cores; I/O to virtual threads.
# - Avoid synchronized on hot paths; use ReentrantLock, atomics, or LongAdder.

# Profile before optimizing
# - JFR + JMC: CPU sampling, allocation, lock contention, GC.
# - async-profiler — flame graphs, mixed-mode (Java + native).
# - Use jcmd <pid> JFR.start ... ; jcmd <pid> JFR.dump filename=...
```

## java.time

```bash
# All immutable, thread-safe. Replaces Date/Calendar.

# Instant — UTC machine time, nanosecond precision
# Instant now = Instant.now();
# Instant later = now.plus(5, ChronoUnit.MINUTES);
# now.getEpochSecond(); now.toEpochMilli();
# Instant.ofEpochMilli(System.currentTimeMillis());

# LocalDate — date without time / zone
# LocalDate today = LocalDate.now();
# LocalDate d = LocalDate.of(2026, Month.APRIL, 25);
# d.plusDays(7); d.minusMonths(2); d.withDayOfMonth(1);
# d.getDayOfWeek(); d.getDayOfYear(); d.lengthOfMonth(); d.isLeapYear();

# LocalTime / LocalDateTime — wall-clock (no zone)
# LocalTime t = LocalTime.of(13, 45, 0);
# LocalDateTime dt = LocalDateTime.of(d, t);

# ZonedDateTime — date+time with timezone (DST-aware)
# ZoneId zone = ZoneId.of("America/Los_Angeles");
# ZonedDateTime z = ZonedDateTime.now(zone);
# z.withZoneSameInstant(ZoneId.of("UTC"));     // same Instant, new zone
# z.withZoneSameLocal(ZoneId.of("UTC"));       // same wall-clock, new zone (changes Instant)
# OffsetDateTime — date+time with fixed offset (no DST rules)

# Duration — machine time amount (seconds + nanos)
# Duration d = Duration.ofSeconds(90);
# Duration d = Duration.between(start, end);
# d.toMinutes(); d.toMillis(); d.toNanos();

# Period — calendar amount (years/months/days)
# Period p = Period.between(birthday, today);
# p.getYears(); p.getMonths(); p.getDays();

# DateTimeFormatter
# DateTimeFormatter.ISO_LOCAL_DATE          // 2026-04-25
# DateTimeFormatter.ISO_DATE_TIME           // 2026-04-25T13:45:00
# DateTimeFormatter.ISO_INSTANT             // 2026-04-25T13:45:00Z
# DateTimeFormatter.ofPattern("yyyy/MM/dd HH:mm")
# d.format(formatter);
# LocalDate.parse("2026-04-25");
# LocalDate.parse("25/04/2026", DateTimeFormatter.ofPattern("dd/MM/yyyy"));

# ALWAYS use Instant for timestamps in storage / wire formats.
# Use java.sql.Timestamp ONLY when talking to JDBC; convert at the boundary.
```

## JSON

```bash
# Jackson — de facto JSON library
# Maven:  com.fasterxml.jackson.core:jackson-databind:2.17.0
# Optional modules:
#   jackson-datatype-jsr310    # for java.time
#   jackson-module-kotlin      # for Kotlin

# Setup
# ObjectMapper mapper = new ObjectMapper();
# mapper.registerModule(new JavaTimeModule());                 // for java.time
# mapper.disable(SerializationFeature.WRITE_DATES_AS_TIMESTAMPS);
# mapper.configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
# mapper.setSerializationInclusion(JsonInclude.Include.NON_NULL);

# DTO with annotations
# public record User(
#     long id,
#     @JsonProperty("full_name") String fullName,
#     @JsonFormat(shape=JsonFormat.Shape.STRING) Instant created,
#     @JsonIgnore String password) {}

# Serialize
# String json = mapper.writeValueAsString(user);
# mapper.writerWithDefaultPrettyPrinter().writeValueAsString(user);
# mapper.writeValue(new File("u.json"), user);

# Deserialize
# User u  = mapper.readValue(json, User.class);
# List<User> us = mapper.readValue(json, new TypeReference<List<User>>() {});
# JsonNode node = mapper.readTree(json);
# String name   = node.get("user").get("name").asText();

# Streaming (low-mem)
# try (JsonParser p = mapper.getFactory().createParser(file)) {
#     while (p.nextToken() != null) { ... }
# }

# Common errors
#   UnrecognizedPropertyException — set FAIL_ON_UNKNOWN_PROPERTIES=false or annotate @JsonIgnoreProperties
#   InvalidDefinitionException: cannot construct instance of X — needs default ctor or @JsonCreator
```

## HTTP Client (11+)

```bash
# java.net.http.HttpClient — modern, async, HTTP/2.
# import java.net.http.*;
# import java.net.URI;
#
# HttpClient client = HttpClient.newBuilder()
#     .version(HttpClient.Version.HTTP_2)
#     .connectTimeout(Duration.ofSeconds(10))
#     .followRedirects(HttpClient.Redirect.NORMAL)
#     .build();

# GET (sync)
# HttpRequest req = HttpRequest.newBuilder()
#     .uri(URI.create("https://api.example.com/users/1"))
#     .header("Accept", "application/json")
#     .timeout(Duration.ofSeconds(5))
#     .GET()
#     .build();
# HttpResponse<String> resp = client.send(req, HttpResponse.BodyHandlers.ofString());
# resp.statusCode(); resp.body(); resp.headers();

# POST JSON
# HttpRequest post = HttpRequest.newBuilder()
#     .uri(URI.create("https://api.example.com/users"))
#     .header("Content-Type", "application/json")
#     .POST(HttpRequest.BodyPublishers.ofString(payloadJson))
#     .build();

# Async
# CompletableFuture<HttpResponse<String>> f =
#     client.sendAsync(req, HttpResponse.BodyHandlers.ofString());
# f.thenApply(HttpResponse::body).thenAccept(System.out::println);

# Body publishers
# BodyPublishers.ofString(s) / ofByteArray(bytes) / ofFile(path) / ofInputStream(...) / noBody()

# Body handlers (consume response)
# BodyHandlers.ofString() / ofByteArray() / ofFile(path) / ofLines() / discarding()

# Errors:
#   java.net.http.HttpTimeoutException — request or connect timeout
#   java.net.ConnectException          — connection refused
#   javax.net.ssl.SSLHandshakeException — TLS issue
```

## Build / CI Tooling

```bash
# javac warnings — turn them on
# -Xlint:all                                    # everything
# -Xlint:-serial,-processing                    # disable noisy ones
# -Werror                                       # warnings -> errors

# Error Prone — bug-pattern checker (Google)
# Maven plugin / Gradle plugin; runs as javac plugin.
# Catches: RemoveUnusedImports, FormatString, BoxedPrimitiveConstructor, EqualsHashCode, ...

# SpotBugs (formerly FindBugs) — bytecode static analysis
# Maven: spotbugs-maven-plugin
# mvn spotbugs:check
# Gradle: id 'com.github.spotbugs'

# PMD / Checkstyle — style and anti-pattern enforcement
# Checkstyle: enforces formatting, naming, modifier order
# PMD:        finds duplicated code, unused vars, complexity

# JaCoCo — coverage
# Maven: org.jacoco:jacoco-maven-plugin:0.8.12
# mvn test jacoco:report   # -> target/site/jacoco/index.html
# Coverage gate via rule:
#   <rule><element>BUNDLE</element><limit><minimum>0.80</minimum></limit></rule>

# JMH — Java Microbenchmark Harness
# https://github.com/openjdk/jmh — only correct way to micro-benchmark on JVM.
# @State(Scope.Benchmark) class Bench {
#     @Param({"100","10000"}) int n;
#     @Setup public void setup() { ... }
#     @Benchmark public int hash() { return ... ; }
# }
# Run: mvn package && java -jar target/benchmarks.jar -wi 5 -i 10 -f 1
```

## Idioms

```bash
# try-with-resources for ANYTHING AutoCloseable
# try (var rs = ps.executeQuery()) { ... }

# Optional ONLY for return types
# public Optional<User> findById(long id);
# // not: List<Optional<X>>, Optional<Optional<X>>, Map<K, Optional<V>>

# Builder pattern — immutables with many optional fields
# public final class Request {
#     private final String url;
#     private final Map<String,String> headers;
#     private Request(Builder b) { this.url = b.url; this.headers = Map.copyOf(b.headers); }
#     public static Builder builder() { return new Builder(); }
#     public static class Builder {
#         String url; Map<String,String> headers = new HashMap<>();
#         public Builder url(String u)               { this.url = u; return this; }
#         public Builder header(String k, String v)  { headers.put(k,v); return this; }
#         public Request build() { return new Request(this); }
#     }
# }

# Defensive copy at the boundary
# public final class Range {
#     private final List<Integer> xs;
#     public Range(List<Integer> xs) { this.xs = List.copyOf(xs); }    // immutable copy
#     public List<Integer> xs()      { return xs; }                    // already immutable
# }

# Static factory over constructor
# public static List<Integer> range(int lo, int hi) { ... }
# // - meaningful name (vs ctor overload set)
# // - can return cached / subclassed instance
# // - hides type parameter inference (List.of vs new ArrayList<>())

# Objects.requireNonNull — fail fast in constructors / setters
# this.name = Objects.requireNonNull(name, "name must not be null");

# Ternary for null defaulting
# String n = user != null ? user.name() : "anon";
# // or:  Optional.ofNullable(user).map(User::name).orElse("anon");

# Avoid overusing inheritance — prefer composition + interface
```

## Tips

- Pin to an LTS JDK (17 or 21) and run the same on dev, CI, and prod; mismatches surface as `UnsupportedClassVersionError: class file version 65 (this JDK supports up to 61)`.
- Use `var` for local variables only — never lose explicit types in public method signatures or fields.
- Prefer `record` for data carriers and `sealed interface` for type hierarchies — together with pattern-matching `switch` they replace the visitor pattern.
- Use virtual threads (21+) for I/O-bound work; do not pool them and avoid `synchronized` on hot paths (use `ReentrantLock`).
- Always put `Files.lines`, `Stream<Path>`, `Connection`, `PreparedStatement`, `ResultSet`, `InputStream` in a `try-with-resources`.
- Default to `List.of()`, `Map.of()`, `Set.of()` for fixed collections; they fail fast on null and are unmodifiable.
- Use `BigDecimal` (built from `String`!) for money; never `double`.
- Use `java.time` everywhere; never `java.util.Date` / `Calendar` / `SimpleDateFormat` (the last is not thread-safe).
- Always pass `Locale.ROOT` to `toLowerCase`/`toUpperCase`/`String.format` when not user-facing.
- Catch the narrowest exception you can handle; restore the interrupt flag with `Thread.currentThread().interrupt()` after catching `InterruptedException`.
- Set `-Xms == -Xmx` in production; enable `-XX:+HeapDumpOnOutOfMemoryError`; ship JFR with `-XX:StartFlightRecording=...`.
- Profile with JFR + JMC or async-profiler before optimizing; benchmark with JMH, never hand-rolled loops.
- Use `ConcurrentHashMap.computeIfAbsent` / `merge` instead of check-then-put.
- Build fat-jars with relocation (Maven Shade / Gradle Shadow) when needed; otherwise prefer `jlink` for small native runtimes.
- Use SLF4J placeholders (`log.info("k={}", v)`) — never string-concatenate at log sites.
- Run `jcmd <pid> VM.flags` to confirm production JVM tuning is what you think it is.

## See Also

- [polyglot](polyglot.md)
- [c](c.md)
- [rust](rust.md)
- [go](go.md)
- [python](python.md)
- [javascript](javascript.md)
- [typescript](typescript.md)
- [ruby](ruby.md)
- [lua](lua.md)
- [make](make.md)
- [webassembly](webassembly.md)
- [bash](bash.md)
- [regex](regex.md)

## References

- [Java SE 21 API Documentation](https://docs.oracle.com/en/java/javase/21/docs/api/index.html)
- [Java Language Specification (JLS) SE 21](https://docs.oracle.com/javase/specs/jls/se21/html/index.html)
- [Java Virtual Machine Specification SE 21](https://docs.oracle.com/javase/specs/jvms/se21/html/index.html)
- [The Java Tutorials](https://docs.oracle.com/javase/tutorial/)
- [JEP Index (OpenJDK)](https://openjdk.org/jeps/0)
- [GC Tuning Guide (21)](https://docs.oracle.com/en/java/javase/21/gctuning/)
- [JFR / Mission Control](https://docs.oracle.com/en/java/javase/21/jfapi/)
- [Effective Java, 3rd Ed. — Joshua Bloch](https://www.oreilly.com/library/view/effective-java/9780134686097/)
- [Java Concurrency in Practice — Brian Goetz](https://jcip.net/)
- [Modern Java in Action — Urma, Fusco, Mycroft](https://www.manning.com/books/modern-java-in-action)
- [Baeldung — Java Tutorials](https://www.baeldung.com/)
- [SLF4J Manual](https://www.slf4j.org/manual.html)
- [JUnit 5 User Guide](https://junit.org/junit5/docs/current/user-guide/)
- [Mockito Reference](https://javadoc.io/doc/org.mockito/mockito-core/latest/org/mockito/Mockito.html)
- [Jackson Documentation](https://github.com/FasterXML/jackson-docs)
- [Maven — Introduction to the Build Lifecycle](https://maven.apache.org/guides/introduction/introduction-to-the-lifecycle.html)
- [Gradle User Manual](https://docs.gradle.org/current/userguide/userguide.html)
- [SDKMAN!](https://sdkman.io/)
- [Eclipse Adoptium (Temurin)](https://adoptium.net/)
