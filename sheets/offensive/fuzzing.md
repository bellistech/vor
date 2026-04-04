# Fuzzing (Coverage-Guided Vulnerability Discovery)

> For authorized security testing, CTF competitions, and educational purposes only.

Fuzzing is an automated testing technique that feeds malformed, unexpected, or random data to programs to discover crashes, memory corruption, assertion failures, and undefined behavior. Coverage-guided fuzzers instrument target code to track which paths have been exercised, then mutate inputs to maximize code coverage and find bugs faster than blind random testing.

---

## AFL++ (American Fuzzy Lop Plus Plus)

### Installation and Setup

```bash
# Install AFL++ from source
git clone https://github.com/AFLplusplus/AFLplusplus
cd AFLplusplus && make distrib
sudo make install

# Install from package manager
apt install afl++                # Debian/Ubuntu
brew install aflplusplus         # macOS

# Verify installation
afl-fuzz --version
afl-cc --version
```

### Compiling Targets with Instrumentation

```bash
# Compile with AFL++ compiler wrappers
CC=afl-cc CXX=afl-c++ ./configure
make clean && make

# Compile a single file
afl-cc -o target target.c

# Use LTO mode for best performance (requires LLVM)
CC=afl-cc CXX=afl-c++ AFL_CC_COMPILER=LTO ./configure
make

# Use CMPLOG for better input-to-state inference
afl-cc -o target_cmplog target.c
# Run with: afl-fuzz -c ./target_cmplog ...

# Compile with AddressSanitizer for crash details
AFL_USE_ASAN=1 afl-cc -o target_asan target.c

# Persistent mode (harness for speed, avoids fork overhead)
# In source: __AFL_FUZZ_INIT(); while (__AFL_LOOP(10000)) { ... }
afl-cc -o target_persist target_persist.c
```

### Running Fuzzing Campaigns

```bash
# Basic fuzzing run
afl-fuzz -i corpus/ -o findings/ -- ./target @@

# @@ is replaced with the test case file path
# Use stdin instead of @@:
afl-fuzz -i corpus/ -o findings/ -- ./target

# Set memory limit (default 50MB)
afl-fuzz -m 256 -i corpus/ -o findings/ -- ./target @@

# Set timeout per execution (default auto-detected)
afl-fuzz -t 1000 -i corpus/ -o findings/ -- ./target @@

# Enable deterministic mutations first
afl-fuzz -D -i corpus/ -o findings/ -- ./target @@

# Use dictionary for format-aware fuzzing
afl-fuzz -x dict/xml.dict -i corpus/ -o findings/ -- ./target @@

# Power schedules (explore, fast, coe, lin, quad, exploit)
afl-fuzz -p explore -i corpus/ -o findings/ -- ./target @@
```

### Parallel Fuzzing

```bash
# Master instance
afl-fuzz -M main -i corpus/ -o findings/ -- ./target @@

# Secondary instances (run on additional cores)
afl-fuzz -S secondary01 -i corpus/ -o findings/ -- ./target @@
afl-fuzz -S secondary02 -i corpus/ -o findings/ -- ./target @@

# Secondary with different power schedule
afl-fuzz -S sec_fast -p fast -i corpus/ -o findings/ -- ./target @@

# Check parallel status
afl-whatsup -s findings/

# Use all available cores automatically
export AFL_AUTORESUME=1
for i in $(seq 1 $(nproc)); do
    afl-fuzz -S "worker_$i" -i corpus/ -o findings/ -- ./target @@ &
done
```

## libFuzzer (LLVM In-Process Fuzzer)

### Writing Fuzz Targets

```bash
# Minimal fuzz target (fuzz_target.c)
# #include <stdint.h>
# #include <stddef.h>
# int LLVMFuzzerTestOneInput(const uint8_t *data, size_t size) {
#     // Call your function with fuzz data
#     parse_input(data, size);
#     return 0;
# }

# Compile with libFuzzer
clang -g -O1 -fno-omit-frame-pointer \
    -fsanitize=fuzzer,address \
    fuzz_target.c target_lib.c \
    -o fuzz_binary

# Run the fuzzer
./fuzz_binary corpus/

# Run with options
./fuzz_binary corpus/ \
    -max_len=4096 \
    -timeout=10 \
    -jobs=4 \
    -workers=4 \
    -dict=xml.dict \
    -only_ascii=1

# Minimize corpus
./fuzz_binary -merge=1 minimized_corpus/ corpus/

# Print coverage report
./fuzz_binary corpus/ -runs=0 -print_coverage=1

# Run specific crash reproducer
./fuzz_binary crash-abc123def456
```

## Go Native Fuzzing (go test -fuzz)

### Writing Go Fuzz Tests

```bash
# Fuzz test in Go (parser_fuzz_test.go)
# func FuzzParseConfig(f *testing.F) {
#     // Seed corpus
#     f.Add([]byte(`{"key": "value"}`))
#     f.Add([]byte(`{"nested": {"a": 1}}`))
#     f.Add([]byte(""))
#     f.Add([]byte("{"))
#
#     f.Fuzz(func(t *testing.T, data []byte) {
#         result, err := ParseConfig(data)
#         if err != nil { return }
#         // Re-encode and check round-trip
#         encoded, err := EncodeConfig(result)
#         if err != nil { t.Fatal(err) }
#         result2, err := ParseConfig(encoded)
#         if err != nil { t.Fatal(err) }
#         if !reflect.DeepEqual(result, result2) {
#             t.Fatal("round-trip mismatch")
#         }
#     })
# }

# Run the fuzz test
go test -fuzz=FuzzParseConfig -fuzztime=60s ./pkg/parser/

# Run with race detector
go test -fuzz=FuzzParseConfig -fuzztime=5m -race ./pkg/parser/

# Run with specific corpus directory
go test -fuzz=FuzzParseConfig -fuzztime=10m \
    -test.fuzzcachedir=/tmp/fuzz_cache ./pkg/parser/

# Reproduce a crash
go test -run=FuzzParseConfig/corpus_entry_name ./pkg/parser/

# Corpus entries stored in testdata/fuzz/<FuncName>/
ls testdata/fuzz/FuzzParseConfig/
```

## Honggfuzz

```bash
# Install honggfuzz
apt install honggfuzz     # Debian
brew install honggfuzz    # macOS

# Compile with honggfuzz instrumentation
hfuzz-cc -o target target.c
hfuzz-clang -o target target.c

# Basic fuzzing run
honggfuzz -i corpus/ -o crashes/ -- ./target ___FILE___
# ___FILE___ is replaced with test case path

# Feedback-driven (coverage-guided) mode
honggfuzz --input corpus/ --output crashes/ \
    --threads $(nproc) \
    --max_file_size 4096 \
    --timeout 5 \
    -- ./target ___FILE___

# Use hardware performance counters (Linux)
honggfuzz --linux_perf_branch -- ./target ___FILE___

# Persistent mode (fastest)
# In source: HF_ITER(uint8_t **buf, size_t *len)
honggfuzz --persistent -- ./target_persist
```

## Corpus Construction

### Building Effective Seed Corpora

```bash
# Gather valid inputs from test suites and real data
find /path/to/testdata -name "*.json" -exec cp {} corpus/ \;
find /path/to/testdata -name "*.xml" -exec cp {} corpus/ \;

# Create boundary value seeds manually
echo -n "" > corpus/empty                           # empty input
echo -n "A" > corpus/single_byte                    # minimal
python3 -c "print('A'*65536)" > corpus/oversized    # large input
echo -n '{"' > corpus/truncated_json                # truncated
printf '\x00\x00\x00\x00' > corpus/null_bytes       # null bytes
printf '\xff\xfe' > corpus/bom_marker                # BOM

# Format string seeds
echo -n '%s%s%s%s%s%s' > corpus/fmt_string
echo -n '%n%n%n%n' > corpus/fmt_write
echo -n '%x%x%x%x' > corpus/fmt_hex

# Integer boundary values
python3 -c "import struct; open('corpus/int_max','wb').write(struct.pack('<I', 0xFFFFFFFF))"
python3 -c "import struct; open('corpus/int_neg','wb').write(struct.pack('<i', -1))"
python3 -c "import struct; open('corpus/int_zero','wb').write(struct.pack('<I', 0))"

# UTF-8 edge cases
printf '\xc0\xaf' > corpus/overlong_utf8             # overlong encoding
printf '\xed\xa0\x80' > corpus/surrogate_half        # surrogate half
printf '\xf4\x90\x80\x80' > corpus/above_max_cp     # above U+10FFFF
printf '\xef\xbb\xbf' > corpus/utf8_bom             # UTF-8 BOM

# Minimize corpus to unique coverage paths
afl-cmin -i corpus/ -o corpus_min/ -- ./target @@

# Further minimize individual files
afl-tmin -i crash_input -o crash_min -- ./target @@
```

## Radamsa (Generation-Based Mutator)

```bash
# Install Radamsa
git clone https://gitlab.com/akihe/radamsa && cd radamsa && make && sudo make install

# Generate mutated outputs from a seed
echo "hello world" | radamsa
echo "hello world" | radamsa -n 100 -o output/mut-%n.txt

# Mutate binary files
radamsa -o mutated-%n.bin -n 50 original.png

# Pipe directly to target
for i in $(seq 1 10000); do
    radamsa seed.txt | timeout 5 ./target
    if [ $? -gt 128 ]; then
        echo "Crash on iteration $i"
    fi
done

# Combine with known valid inputs
radamsa -n 100 -o fuzzed/out-%n.json valid1.json valid2.json valid3.json
```

## Syzkaller (Kernel Fuzzer)

```bash
# Syzkaller setup for Linux kernel fuzzing
# Requires: Go, QEMU, kernel source with KCOV

# Build kernel with coverage and sanitizers
# CONFIG_KCOV=y
# CONFIG_KASAN=y
# CONFIG_KCOV_INSTRUMENT_ALL=y
# CONFIG_DEBUG_INFO=y

# Clone and build syzkaller
git clone https://github.com/google/syzkaller
cd syzkaller && make

# Create VM image
./tools/create-image.sh

# Configure syzkaller (my.cfg)
# {
#     "target": "linux/amd64",
#     "http": "127.0.0.1:56741",
#     "workdir": "/path/to/workdir",
#     "kernel_obj": "/path/to/linux",
#     "image": "/path/to/image",
#     "sshkey": "/path/to/ssh/key",
#     "syzkaller": "/path/to/syzkaller",
#     "type": "qemu",
#     "vm": {
#         "count": 4,
#         "kernel": "/path/to/bzImage",
#         "cpu": 2,
#         "mem": 2048
#     }
# }

# Run syzkaller
./bin/syz-manager -config=my.cfg

# Web dashboard at http://127.0.0.1:56741
# Shows coverage, crashes, reproducers
```

## Cargo-Fuzz (Rust Fuzzing)

```bash
# Install cargo-fuzz
cargo install cargo-fuzz

# Initialize fuzz targets in a Rust project
cd my_rust_project
cargo fuzz init

# Add a fuzz target
cargo fuzz add parse_input

# Edit fuzz target (fuzz/fuzz_targets/parse_input.rs)
# #![no_main]
# use libfuzzer_sys::fuzz_target;
# use my_crate::parse;
#
# fuzz_target!(|data: &[u8]| {
#     let _ = parse(data);
# });

# Run the fuzzer
cargo fuzz run parse_input

# Run with options
cargo fuzz run parse_input -- \
    -max_len=4096 \
    -timeout=10 \
    -jobs=4

# Run with sanitizers
cargo fuzz run parse_input --sanitizer=address
cargo fuzz run parse_input --sanitizer=memory

# Minimize corpus
cargo fuzz cmin parse_input

# Minimize a crash input
cargo fuzz tmin parse_input artifacts/parse_input/crash-abc123

# List available targets
cargo fuzz list
```

## Coverage Metrics and Crash Triage

### Measuring Coverage

```bash
# LLVM source-based coverage
clang -fprofile-instr-generate -fcoverage-mapping \
    -o target target.c

LLVM_PROFILE_FILE="cov.profraw" ./target < test_input
llvm-profdata merge -sparse cov.profraw -o cov.profdata
llvm-cov show ./target -instr-profile=cov.profdata \
    -format=html -output-dir=coverage_report/
llvm-cov report ./target -instr-profile=cov.profdata

# Go coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
go tool cover -func=coverage.out

# AFL++ coverage stats (in findings/default/plot_data)
afl-plot findings/ plot_output/
# Columns: unix_time, cycles_done, cur_item, corpus_count,
#           pending_favs, pending_total, bitmap_cvg, ...

# gcov for GCC-compiled targets
gcc --coverage -o target target.c
./target < test_input
gcov target.c
# Produces target.c.gcov with per-line execution counts
```

### Crash Triage and Deduplication

```bash
# Examine AFL++ crash outputs
ls findings/default/crashes/
# id:000000,sig:11,src:000123,time:456789,... -> SIGSEGV
# id:000001,sig:06,src:000456,time:789012,... -> SIGABRT

# Reproduce a crash
./target < findings/default/crashes/id:000000,sig:11,*

# Reproduce with ASAN for details
./target_asan < findings/default/crashes/id:000000,sig:11,*

# Crash triage with afl-collect
pip install afl-utils
afl-collect -d crashes.db -e gdb_script \
    -r findings/default/ ./target -- @@

# Stack hash deduplication
# Group crashes by unique stack trace hash
for crash in findings/default/crashes/id:*; do
    hash=$(./target_asan < "$crash" 2>&1 | \
        grep "^    #" | md5sum | cut -d' ' -f1)
    echo "$crash -> $hash"
done | sort -t'>' -k2 | uniq -f1 -c | sort -rn

# GDB batch crash analysis
gdb -batch -ex run -ex bt -ex quit \
    --args ./target findings/default/crashes/id:000000,sig:11,*

# Casr (Crash Analysis and Severity Rating)
# cargo install casr
casr-gdb -o report.json -- ./target crash_input
casr-cli report.json
```

## Campaign Design and Strategy

```bash
# Phase 1: Quick coverage with minimal corpus (1-2 hours)
afl-fuzz -i corpus_min/ -o findings/ -V 7200 -- ./target @@

# Phase 2: Dictionary-driven exploration
afl-fuzz -i corpus_min/ -o findings/ -x custom.dict -- ./target @@

# Phase 3: CMPLOG for magic byte discovery
afl-fuzz -i corpus_min/ -o findings/ \
    -c ./target_cmplog -- ./target @@

# Phase 4: Parallel scaling
for i in $(seq 1 $(nproc)); do
    afl-fuzz -S "worker_$i" -i corpus_min/ -o findings/ \
        -p $(echo "fast explore coe" | tr ' ' '\n' | \
        shuf -n1) -- ./target @@ &
done

# Monitor campaign health
watch -n 30 afl-whatsup -s findings/
# Key metrics to watch:
#   - cycles_done: >1 means full corpus has been fuzzed
#   - corpus_count: should grow then plateau
#   - bitmap_cvg: percentage of edge coverage
#   - pending_favs: should decrease over time
#   - unique_crashes: the goal

# Environment tuning for Linux
echo core | sudo tee /proc/sys/kernel/core_pattern
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor
```

---

## Tips

- Always compile separate binaries for fuzzing (instrumented), ASAN (crash details), and CMPLOG (comparison logging) to get the best results from each
- Start with a small, minimized corpus using afl-cmin; too many redundant seeds waste cycles
- Use dictionaries for structured formats (JSON, XML, HTTP, SQL) to help the fuzzer construct syntactically meaningful inputs
- Enable ASAN during fuzzing to catch memory bugs that do not cause immediate crashes
- Monitor bitmap coverage; if it plateaus early, your harness may not be reaching deep code paths
- Run campaigns for at least 24 hours; many bugs only surface after millions of executions
- Triage crashes by stack hash to deduplicate; focus on unique root causes rather than raw crash count
- For network services, use AFL++ persistent mode or desock (preeny) to avoid socket overhead
- Combine generation-based tools (Radamsa) with coverage-guided fuzzers for protocol fuzzing
- Keep seed corpora in version control; they represent reusable test knowledge

---

## See Also

- sanitizers
- checksec
- reverse-engineering

## References

- [AFL++ Documentation](https://aflplus.plus/docs/)
- [libFuzzer Reference](https://llvm.org/docs/LibFuzzer.html)
- [Go Fuzzing Documentation](https://go.dev/doc/security/fuzz/)
- [Honggfuzz Documentation](https://github.com/google/honggfuzz)
- [Syzkaller Documentation](https://github.com/google/syzkaller/blob/master/docs/linux/setup.md)
- [Cargo-Fuzz Book](https://rust-fuzz.github.io/book/cargo-fuzz.html)
- [Google OSS-Fuzz](https://google.github.io/oss-fuzz/)
- [Radamsa](https://gitlab.com/akihe/radamsa)
