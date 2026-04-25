# Ruby (Programming Language)

A dynamic, object-oriented, expressive scripting language designed for programmer happiness — every value is an object, blocks are first-class, and least-surprise is a guiding principle.

## Setup

```bash
# Version managers (pick ONE)
# asdf — multi-language version manager
asdf plugin add ruby
asdf install ruby 3.3.6
asdf global ruby 3.3.6

# rbenv — lightweight, shim-based
brew install rbenv ruby-build
rbenv install 3.3.6
rbenv global 3.3.6
echo 'eval "$(rbenv init - bash)"' >> ~/.bashrc

# rvm — heavier, manages gemsets too
\curl -sSL https://get.rvm.io | bash -s stable
rvm install 3.3.6
rvm use 3.3.6 --default

# chruby + ruby-install — minimal, fastest shell startup
brew install chruby ruby-install
ruby-install ruby 3.3.6
echo 'source /usr/local/share/chruby/chruby.sh' >> ~/.bashrc
echo 'source /usr/local/share/chruby/auto.sh'   >> ~/.bashrc
chruby ruby-3.3.6

# System ruby on macOS (avoid for development; pinned old version)
ruby -v                       # ruby 2.6.x — OS owns this, do not gem install

# Verify install
ruby -v                       # ruby 3.3.6 (2024-11-05 revision ...)
which ruby                    # /Users/me/.asdf/shims/ruby
gem env                       # paths, version, sources

# Core executables shipped with Ruby
ruby   file.rb                # run a script
irb                           # interactive REPL
erb    template.erb           # eRuby template processor
rdoc   lib/                   # documentation generator
ri     Array#map              # offline docs viewer
rake                          # Make-like build tool
gem    install bundler        # package manager (RubyGems)
bundle install                # Bundler — Gemfile-based deps

# Bundler — install once into the active Ruby
gem install bundler
bundle -v                     # Bundler version 2.5.x

# Common ruby flags
ruby -v                       # version
ruby -e 'puts "hi"'           # one-liner
ruby -ne 'puts $_.upcase' f   # loop, pre-binds $_ per line
ruby -pe '$_.gsub!(/foo/,"bar")' f  # loop + auto-print
ruby -i.bak -pe '...' file    # in-place edit, .bak backup
ruby -c file.rb               # syntax check only
ruby -w file.rb               # warnings on
ruby -W2 file.rb              # very verbose warnings
ruby -d file.rb               # debug mode ($DEBUG = true)
ruby -Ilib script.rb          # add lib/ to $LOAD_PATH
ruby -rjson -e 'pp JSON.parse "[1,2]"'  # require library
ruby -a -ne 'puts $F[0]' f    # autosplit on whitespace into $F
ruby -F: -ane 'puts $F[0]' f  # autosplit on : delimiter
ruby --jit / --yjit script.rb # enable JIT (3.1+ YJIT, 3.2+ default-on with flag)

# irb flags
irb --simple-prompt           # short prompt
irb -r ./lib/foo.rb           # require a file at start
irb --noecho                  # don't echo result of each expression
# Ruby 3.1+ ships reline-based irb with autocomplete and measure command

# Ruby 3.0 (2020) highlights — Christmas release (every Dec 25)
#  - Ractor (parallel execution unit; experimental)
#  - Pattern matching out of experimental (case/in)
#  - One-line method def: def square(x) = x * x
#  - Hash#except built-in
#  - RBS (type signatures), TypeProf
#  - Endless ranges already (2.6); beginless ranges new

# Ruby 3.1 (2021)
#  - YJIT (Shopify's JIT) — opt-in via --yjit
#  - Hash literal value omission: {x:, y:} == {x: x, y: y}
#  - Anonymous block forwarding: def foo(&) = bar(&)
#  - Pattern matching with pinning expressions

# Ruby 3.2 (2022)
#  - YJIT marked production-ready, ARM64 support
#  - WASI WebAssembly target
#  - Data class (immutable value objects, alternative to Struct)
#  - Regex DoS mitigation, Regex.timeout = 1.0

# Ruby 3.3 (2023)
#  - YJIT 2x faster on Rails
#  - RJIT replaces MJIT
#  - it block parameter (preview)
#  - Prism parser preview

# Run a project from a Gemfile
bundle init                   # create empty Gemfile
bundle add rails              # add and install
bundle install                # install all deps
bundle exec rspec             # run binary in bundle context
bundle config set --local path 'vendor/bundle'  # vendor deps
```

## Hello World, Scripts, One-Liners

```bash
# Smallest possible
ruby -e 'puts "Hello, world!"'

# Script file
# hello.rb
#!/usr/bin/env ruby
# # frozen_string_literal: true
# puts "Hello, world!"
chmod +x hello.rb && ./hello.rb

# Shebang variants
##!/usr/bin/env ruby                   # PORTABLE — picks ruby on PATH
##!/usr/bin/ruby                       # Linux distros
##!/usr/bin/env -S ruby --yjit         # pass extra flags (Linux/macOS env -S)

# One-liners (sed/awk replacements)
ruby -ne 'print if /ERROR/' app.log              # grep
ruby -pe '$_.gsub!(/foo/, "bar")' input          # sed s/foo/bar/g
ruby -pi -e 'gsub(/foo/, "bar")' file            # in-place edit
ruby -pi.bak -e 'gsub(/foo/, "bar")' file        # in-place + backup
ruby -ane 'puts $F[1]' /etc/passwd               # awk '{print $2}'
ruby -F: -ane 'puts $F[0]' /etc/passwd           # awk -F: '{print $1}'
ruby -e 'puts ARGV.inspect' a b c                # ["a","b","c"]
ruby -rjson -e 'puts JSON.generate({a:1})'       # require + run
echo 'hi' | ruby -ne 'puts $_.upcase'            # stdin filter
ruby -e 'puts (1..100).select { |n| n % 7 == 0 }'

# Read entire stdin
ruby -e 'puts STDIN.read.length'

# Run as a here-document from shell
ruby <<'RUBY'
3.times { |i| puts "line #{i}" }
RUBY

# Bundler-style executable in a project
bundle exec ruby script.rb

# Encoding (UTF-8 by default since 2.0)
ruby -e 'p "héllo".encoding'                     # #<Encoding:UTF-8>
```

## Variables and Scope

```bash
# Five flavours, distinguished by FIRST character
# local      — lowercase or _   — visible inside def/block scope
# @instance  — @                — per-object, lives on the receiver
# @@class    — @@               — per-class hierarchy (USE SPARINGLY)
# $global    — $                — accessible everywhere (avoid!)
# CONSTANT   — UPPERCASE        — top-level or class-scoped, warns on reassign

# Local
# x = 10
# y, z = 20, 30           # parallel assignment
# a, *b = 1, 2, 3, 4      # a=1, b=[2,3,4]
# *a, b = 1, 2, 3, 4      # a=[1,2,3], b=4

# Instance — must be inside a class/module
# class Person
#   def initialize(name)
#     @name = name        # only this object knows @name
#   end
# end

# Class variable — DANGEROUS, shared with subclasses
# class Counter
#   @@count = 0
#   def self.bump; @@count += 1; end
# end

# Class instance variable (PREFERRED over @@)
# class Counter
#   @count = 0            # belongs to the class object itself
#   class << self
#     attr_accessor :count
#   end
# end

# Global
# $LOG_LEVEL = :info
# $stdout.puts "x"        # PREFERRED over STDOUT — redirectable

# Constant
# PI = 3.14159
# MAX = 100
# # Reassigning a constant only warns; it doesn't error
# PI = 3.14                # warning: already initialized constant PI

# Pseudo-variables (read-only)
# self      — current object
# nil       — the only NilClass instance
# true      — the only TrueClass instance
# false     — the only FalseClass instance
# __FILE__  — current source filename
# __LINE__  — current line number
# __dir__   — directory of current file (Ruby 2.0+)
# __method__ — currently executing method name

# Local-variable visibility rules
# - Block creates new scope BUT can see outer locals
# - def creates a new scope, does NOT see outer locals (except constants)
# - lambda/proc captures surrounding locals as closure

# Tricky: parens distinguish call-vs-local
# foo                   # local var lookup if exists, else method call
# foo()                 # always method call
# self.foo              # always method call (cannot match a local)

# defined? — query
# defined?(x)           # nil if x undefined; "local-variable", "method", etc.
```

## Object Model

```bash
# RULE: everything you can name is an object.
#   42.class                    => Integer
#   :sym.class                  => Symbol
#   nil.class                   => NilClass
#   true.class                  => TrueClass
#   Class.class                 => Class
#   Class.superclass            => Module
#   Module.superclass           => Object
#   Object.superclass           => BasicObject
#   BasicObject.superclass      => nil

# Hierarchy
#   BasicObject  — almost no methods, only ==, equal?, !, instance_eval
#       ↑
#     Object  — your usual root; mixes in Kernel
#       ↑
#     Kernel  — module providing puts, gets, raise, require, lambda, ...
#       ↑
#     YourClass

# Kernel is a MODULE included into Object → that's why `puts` works anywhere
# Object.ancestors                 # [Object, Kernel, BasicObject]

# Integer ancestry
#   Integer.ancestors              # [Integer, Numeric, Comparable, Object, Kernel, BasicObject]

# Methods that EVERY object has (via Object/Kernel)
#   .class .object_id .equal? .== .eql? .hash .inspect .to_s .send
#   .respond_to? .methods .public_methods .instance_variables .frozen?
#   .freeze .dup .clone .tap .then/.yield_self .nil? .is_a? .kind_of? .instance_of?

# .equal? vs == vs eql?
# a, b = "x", "x"
# a == b        # true   — value equality (per class override)
# a.eql? b      # true   — stricter; for Hash
# a.equal? b    # false  — same object (object_id) — DO NOT override
# 1 == 1.0      # true
# 1.eql?(1.0)   # false  — eql? rejects type coercion

# Object inspection
# x.inspect                       # programmer-style "x"
# x.to_s                          # user-style x
# pp x                            # pretty-print to stdout
# p  x                            # puts x.inspect
```

## Numbers

```bash
# Integer — arbitrary precision since 2.4 (was Fixnum/Bignum split)
# 1.class                         # Integer
# 10**100                         # huge — no overflow
# 1_000_000                       # underscores ignored, just for readability
# 0xff                            # hex 255
# 0o77                            # octal 63
# 0b1010                          # binary 10
# 100.bit_length                  # 7

# Float (double-precision IEEE-754)
# 3.14
# 1.5e3                           # 1500.0
# Float::INFINITY                 # Infinity
# Float::NAN                      # NaN
# 0.1 + 0.2 == 0.3                # FALSE — same as everywhere
# (0.1 + 0.2).round(10) == 0.3    # true

# Rational — exact fractions
# r = Rational(1, 3)              # (1/3)
# r * 3                           # (1/1)
# 1.5r                            # (3/2) — literal suffix

# Complex
# c = Complex(1, 2)               # (1+2i)
# 2i + 3                          # (3+2i)

# BigDecimal — arbitrary precision decimals (require 'bigdecimal')
# require 'bigdecimal'
# BigDecimal("0.1") + BigDecimal("0.2")    # 0.3e0 (exact)

# Conversions
# "42".to_i                       # 42 — silent on garbage
# "42abc".to_i                    # 42
# Integer("42")                   # 42 — strict, raises ArgumentError
# Integer("0x10", 16)             # 16
# "3.14".to_f                     # 3.14
# Float("abc") rescue nil         # nil
# 1.to_r                          # (1/1)
# "1/2".to_r                      # (1/2)
# 2.5.to_i                        # 2
# 2.5.round                       # 3 (banker's rounding for .5 since 2.4)
# (-2.5).round                    # -2
# 2.5.ceil                        # 3
# 2.5.floor                       # 2
# 2.5.truncate                    # 2

# Loops by number
# 5.times { |i| puts i }          # 0..4
# 1.upto(3) { |i| puts i }        # 1, 2, 3
# 5.downto(1) { |i| puts i }      # 5, 4, 3, 2, 1
# 1.step(10, 2) { |i| puts i }    # 1, 3, 5, 7, 9

# Useful methods
# 10.even? / 7.odd?                # true / true
# 10.zero? / 0.zero?               # false / true
# (-5).abs                         # 5
# 17.divmod(5)                     # [3, 2]
# 17 / 5                           # 3 (integer)
# 17.0 / 5                         # 3.4
# 17 % 5                           # 2
# 2 ** 10                          # 1024
# 7.gcd(21)                        # 7
# 7.lcm(21)                        # 21
# 255.to_s(16)                     # "ff"
# "ff".to_i(16)                    # 255
# 10.digits                        # [0, 1] — least significant first
# 10.digits(2)                     # [0,1,0,1] — base 2 digits

# Math module
# Math::PI / Math::E
# Math.sqrt(2)  Math.log(8, 2)  Math.sin(Math::PI/2)
```

## Strings

```bash
# Single quotes — literal, NO interpolation, only \\ and \' escape
# 'hello\nworld'                   # 12 chars: hello\nworld
# 'it\'s'                          # it's

# Double quotes — interpolation + all escapes
# x = 5
# "x = #{x}"                       # "x = 5"
# "tab\there"                      # tab → real tab
# "unié"                      # é

# %q — single-quote with chosen delimiter
# %q[don't]                        # don't
# %q{a\nb}                         # a\nb (literal)

# %Q — double-quote with chosen delimiter
# %Q{x = #{x}}                     # x = 5
# %{x = #{x}}                      # %{} default == %Q

# Heredocs
# text = <<EOF
# hello
# world
# EOF
#                                  # IDENTIFIER must start at col 0, content not indented

# <<-EOF — closing identifier may be indented (content kept as-is)
# text = <<-EOF
#   hello
#   EOF

# <<~EOF — squiggly heredoc, removes COMMON LEADING WHITESPACE (Ruby 2.3+) — preferred
# text = <<~EOF
#   hello
#   world
# EOF                              # "hello\nworld\n"

# <<'EOF' — no interpolation, literal
# text = <<~'EOF'
#   #{not_interpolated}
# EOF

# Frozen string literal magic comment — at TOP OF FILE
# frozen_string_literal: true     # all string literals .freeze'd
# Saves memory, hardens against mutation. RECOMMENDED.

# Common operations
# "hello".upcase                   # "HELLO"
# "HELLO".downcase                 # "hello"
# "hello".capitalize               # "Hello"
# "Hello World".swapcase           # "hELLO wORLD"
# "hello".reverse                  # "olleh"
# "hello".length / .size           # 5
# "héllo".bytesize                 # 6 (UTF-8)
# "hello".chars                    # ["h","e","l","l","o"]
# "hello".bytes                    # [104, 101, 108, 108, 111]
# "hello".each_char { |c| puts c }
# "a,b,c".split(",")               # ["a","b","c"]
# "a,b,,c".split(",", -1)          # ["a","b","","c"] keep trailing empties
# %w[a b c].join("-")              # "a-b-c"
# "  pad  ".strip / .lstrip / .rstrip
# "abc".chomp("c")                 # "ab"
# "abc\n".chomp                    # "abc" — removes ONLY trailing newline
# "abc".chop                       # "ab" — removes last char
# "hello".sub("l","L")             # "heLlo" (first only)
# "hello".gsub("l","L")            # "heLLo"
# "hello".gsub(/l+/) { |m| m.upcase }
# "hello".tr("el","ip")            # "hippo"
# "hello".tr_s("l","x")            # "hexo" (squeeze)
# "  ".empty?                      # false (whitespace counted)
# "".empty?                        # true
# "hello".include?("ell")          # true
# "hello".start_with?("he","ha")   # true
# "hello".end_with?("lo")          # true
# "hello"[1]                       # "e"
# "hello"[1, 3]                    # "ell"
# "hello"[1..3]                    # "ell"
# "hello"[/l+/]                    # "ll"
# "hello".center(11, "*")          # "***hello***"
# "x".ljust(5, ".")                # "x...."
# "x".rjust(5, ".")                # "....x"
# "%-10s|%5d" % ["foo", 42]        # "foo       |   42"
# format("%.2f", 3.14159)          # "3.14"
# "abc" * 3                        # "abcabcabc"
# "abc" + "def"                    # "abcdef"
# "abc" << "def"                   # mutates! NoMethodError on frozen string
# "abc".freeze.frozen?             # true
# "abc".dup                        # unfrozen copy

# Encoding
# "héllo".encoding                 # UTF-8
# "héllo".force_encoding("ASCII-8BIT")
# "abc".valid_encoding?            # true
```

## Symbols

```bash
# A symbol is an INTERNED, IMMUTABLE name. Same symbol == same object_id.
# :name.class                      # Symbol
# :name.object_id == :name.object_id  # true
# "name".object_id == "name".object_id # false (different string objects)

# Why symbols
#  - Hash keys (cheap to compare, no GC churn)
#  - Method names (define_method :foo)
#  - State enums :pending / :done / :failed

# Conversions
# :name.to_s                       # "name"
# "name".to_sym                    # :name
# "name".intern                    # alias of to_sym

# Symbol arrays
# %i[red green blue]               # [:red, :green, :blue]
# %I[a #{x}]                       # interpolating symbol array

# DO NOT user-input → to_sym in long-running processes (symbols WERE never GC'd
# pre-2.2; now they ARE for dynamic ones, but it remains a footgun if abused)

# Symbol#to_proc — block shorthand
# %w[a b].map(&:upcase)            # ["A", "B"]
# Same as .map { |s| s.upcase }
```

## Arrays

```bash
# Literal forms
# []                               # empty
# [1, 2, 3]
# [1, "x", :sym, nil, true]        # heterogeneous fine
# %w[foo bar baz]                  # ["foo","bar","baz"]
# %W[foo #{x}]                     # interpolating
# %i[a b c]                        # [:a,:b,:c]
# Array.new(3)                     # [nil, nil, nil]
# Array.new(3, 0)                  # [0, 0, 0]
# Array.new(3) { |i| i*i }         # [0, 1, 4]
# Array(nil)                       # [] — coerce
# Array(1..3)                      # [1,2,3]
# (1..5).to_a                      # [1,2,3,4,5]

# Access
# a = [10, 20, 30, 40, 50]
# a[0]                             # 10
# a.first / a.last                 # 10 / 50
# a.first(2)                       # [10, 20]
# a[-1]                            # 50
# a[1..3]                          # [20,30,40]
# a[1...3]                         # [20,30]
# a[1, 2]                          # [20,30] (start, length)
# a.dig(0)                         # 10  — nested-safe
# a[100]                           # nil  — no IndexError
# a.fetch(100)                     # IndexError
# a.fetch(100, :default)           # :default
# a.fetch(100) { |i| "miss #{i}" } # block fallback

# Mutating
# a.push(60)        / a << 60      # append
# a.pop                            # remove last
# a.shift                          # remove first
# a.unshift(0)                     # prepend
# a.insert(2, :x)                  # insert at index
# a.delete(20)                     # remove all 20s
# a.delete_at(0)                   # remove by index
# a.delete_if { |x| x > 30 }
# a.clear
# a.concat([100, 200])             # mutates
# a.fill(0)                        # all to 0
# a.fill(0, 1, 2)                  # idx 1..2 to 0

# Non-mutating (preferred)
# a + [60]                         # new array
# a - [20]                         # set difference
# a & [20, 99]                     # intersection
# a | [60]                         # union (deduped)
# a.flatten                        # 1 level by default
# a.flatten(1)
# a.compact                        # remove nils
# a.uniq                           # remove dups
# a.uniq { |x| x % 10 }
# a.sort
# a.sort { |x,y| y <=> x }         # desc
# a.sort_by { |x| -x }
# a.reverse
# a.rotate(1)                      # [20,30,40,50,10]
# a.zip([1,2,3])                   # [[10,1],[20,2],[30,3],[40,nil],[50,nil]]
# a.take(2)                        # [10, 20]
# a.drop(2)                        # [30, 40, 50]
# a.take_while { |x| x < 30 }      # [10, 20]
# a.drop_while { |x| x < 30 }      # [30, 40, 50]
# a.partition { |x| x.even? }      # [[10,20,30,40,50], []]
# a.group_by { |x| x % 3 }         # {1=>[10,40],2=>[20,50],0=>[30]}
# a.tally                          # {10=>1,20=>1,...}  — Ruby 2.7+
# a.chunk_while { |i,j| j-i == 10 } # consecutive groups
# a.each_slice(2).to_a             # [[10,20],[30,40],[50]]
# a.each_cons(2).to_a              # [[10,20],[20,30],...]

# Searching / predicates
# a.include?(20)                   # true
# a.any?                           # true (non-empty)
# a.any? { |x| x > 100 }           # false
# a.all? { |x| x > 0 }             # true
# a.none? { |x| x.nil? }           # true
# a.one? { |x| x > 40 }            # true
# a.count                          # 5
# a.count(20)                      # 1
# a.count { |x| x > 25 }           # 3
# a.find { |x| x > 25 }            # 30 (first match)  — alias detect
# a.index(30)                      # 2
# a.find_index { |x| x > 25 }      # 2
# a.min / .max                     # 10 / 50
# a.minmax                         # [10, 50]
# a.min_by { |x| -x }              # 50

# Transform
# a.map { |x| x * 2 }              # alias collect
# a.flat_map { |x| [x, -x] }       # flatten 1 level after map
# a.select { |x| x > 20 }          # alias filter
# a.reject { |x| x > 20 }
# a.reduce(:+)                     # 150  — alias inject
# a.reduce(0) { |s,x| s + x }
# a.sum                            # 150 (Ruby 2.4+)
# a.each_with_index { |x, i| ... }
# a.each_with_object({}) { |x, h| h[x] = x*x }
# a.lazy                           # lazy enumerator (infinite chains)

# Splat * — unpack/pack
# first, *rest = [1,2,3,4]         # first=1, rest=[2,3,4]
# a, b, c = *[1,2,3]
# def f(*args); end                # collect args
# f(*[1,2,3])                      # spread

# Comparison
# [1,2,3] <=> [1,2,4]              # -1
# Arrays compare element-wise lexicographically
```

## Hashes

```bash
# Literal forms
# h = { "a" => 1, "b" => 2 }       # rocket syntax — any keys
# h = { a: 1, b: 2 }               # symbol keys (shorthand) — same as { :a => 1 }
# h = { "a": 1 }                   # ALSO becomes :a => 1, NOT "a" => 1 (gotcha!)
# h = Hash.new                     # {}
# h = Hash.new(0)                  # default value 0 for missing keys
# h = Hash.new { |hash, k| hash[k] = [] }  # default block (memoizing)

# Insertion-ordered since Ruby 1.9 — iteration is in insertion order

# Value omission shorthand (Ruby 3.1+)
# x, y = 1, 2
# {x:, y:}                         # == {x: 1, y: 2}

# Access
# h[:a]                            # 1
# h[:zzz]                          # nil (or default value)
# h.fetch(:a)                      # 1
# h.fetch(:zzz)                    # KeyError
# h.fetch(:zzz, :default)          # :default
# h.fetch(:zzz) { |k| "no #{k}" }  # block default
# h.dig(:user, :address, :city)    # nested-safe; nil if any miss
# h.key?(:a)   .has_key?  .include?  .member?
# h.value?(1)  .has_value?
# h.keys / .values
# h.length / .size / .count
# h.empty?

# Iteration
# h.each { |k, v| puts "#{k}=#{v}" }
# h.each_pair  / each_key / each_value
# h.map { |k, v| [k, v * 2] }.to_h
# h.transform_values { |v| v * 2 }
# h.transform_keys(&:to_s)
# h.filter_map { |k, v| [k, v] if v > 0 }.to_h
# h.select { |k, v| v > 1 }
# h.reject { |k, v| v.nil? }
# h.partition { |k, v| v > 1 }
# h.group_by { |k, v| v.class }
# h.find { |k, v| v == 1 }         # [:a, 1]
# h.any? { |k, v| v.nil? }
# h.sort_by { |_, v| v }           # array of [k,v] pairs
# h.min_by { |_, v| v }
# h.sum    { |_, v| v }
# h.reduce(0) { |acc, (_, v)| acc + v }

# Mutation
# h[:c] = 3
# h.store(:d, 4)
# h.delete(:a)                     # removes and returns value
# h.delete_if { |k, v| v.nil? }
# h.clear
# h.compact!                       # remove nil values
# h.merge!(other)
# h.update(other)                  # alias of merge!

# Non-mutating
# h.merge(other)                   # right-hand wins
# h.merge(other) { |k, v1, v2| v1 + v2 }  # custom resolver
# h.invert                         # swap keys and values
# h.except(:a, :b)                 # Ruby 3.0+
# h.slice(:a, :b)
# h.to_a                           # [[k,v], ...]

# Comparison
# h1 == h2                         # same key/value pairs (order ignored)
# h1.eql?(h2)                      # stricter (no type coercion)

# Symbol vs String keys
# {a: 1}["a"]                      # nil — different keys
# {a: 1}[:a]                       # 1
# Common bug: HTTP/JSON inputs come as STRING keys; convert with
# JSON.parse(s, symbolize_names: true)
```

## Ranges

```bash
# Inclusive/exclusive
# (1..5)                           # 1,2,3,4,5
# (1...5)                          # 1,2,3,4
# ('a'..'e').to_a                  # ["a","b","c","d","e"]

# Endless ranges (Ruby 2.6+)
# (1..)                            # infinite forward
# arr[2..]                         # idiom: from index 2 to end

# Beginless ranges (Ruby 2.7+)
# (..10)                           # everything up to and including 10
# arr[..2]                         # idiom: first three

# Methods
# (1..10).each { |i| ... }
# (1..10).to_a
# (1..10).step(2).to_a             # [1,3,5,7,9]
# (1..10).cover?(7)                # true (fast for ranges)
# (1..10).include?(7)              # true (slower for char ranges)
# (1..).first(3)                   # [1,2,3]
# (1..).lazy.map { |i| i*i }.first(5)
# (1.0..2.0).step(0.5).to_a        # [1.0, 1.5, 2.0]

# In case/when (range === value)
# case score
# when 90.. then "A"
# when 80..89 then "B"
# end

# Range as array slicer
# arr[1..3]    arr[1...3]   arr[..2]   arr[2..]
```

## Control Flow

```bash
# if / elsif / else / end — note SPELLING (no e in elsif)
# if x > 0
#   "pos"
# elsif x < 0
#   "neg"
# else
#   "zero"
# end

# Trailing modifier (preferred for short guards)
# return unless valid?
# raise "bad" if x < 0

# unless == if not
# unless logged_in?
#   redirect_to_login
# end

# Ternary
# msg = age >= 18 ? "adult" : "minor"

# if is an EXPRESSION — returns the last value
# x = if cond then 1 else 2 end

# case/when — uses === (case-equality, NOT ==)
# case value
# when Integer  then "int"          # Class === instance
# when /^\d+$/  then "digits"       # Regexp === string
# when 1..10    then "small"        # Range === value
# when "yes"    then "ok"
# else "unknown"
# end

# case/in — pattern matching (3.0+, no longer experimental)
# case point
# in [0, 0]              then "origin"
# in [x, 0]              then "on x-axis at #{x}"
# in [x, y] if x == y    then "diagonal"
# in {name:, age: 18..}  then "adult #{name}"
# in Integer => n        then "got int #{n}"
# in [1, *rest]          then "starts with 1, rest=#{rest}"
# else                   "other"
# end

# in expression form (3.0+) — ONE-LINER
# point in [0, 0]                  # true/false

# `=>` form for binding (1-line)
# {a: 1} => {a:}                   # binds local a = 1 (raises NoMatchingPatternError on miss)
```

## Loops

```bash
# Iteration is preferred over explicit loops
# 5.times { |i| puts i }
# arr.each { |x| ... }
# (1..n).each { |i| ... }

# while
# i = 0
# while i < 5
#   i += 1
# end

# until == while not
# until done?
#   work
# end

# loop — infinite, exits via break or StopIteration
# loop do
#   break if done?
# end

# for — exists but DISCOURAGED (leaks loop variable into outer scope!)
# for i in 1..3; ...; end

# break — exit loop, optionally with a value
# x = [1,2,3,4].each { |n| break n if n > 2 }   # x == 3
# arr.find { |x| x > 5 } || raise

# next — skip to next iteration
# arr.each { |x| next if x.nil?; process(x) }

# redo — repeat current iteration with same value (rare; debugging)

# while/until as expression
# loop_result = i = 0; while i < 3 do i += 1 end  # nil

# Idiomatic alternatives
# 5.times { ... }                  # over while-counter
# arr.each_with_index { |x, i| ... }
# (a..b).each { ... }
```

## Methods

```bash
# Basic def
# def add(a, b)
#   a + b                          # last expression returned implicitly
# end

# Explicit return — usually unneeded except for early exit
# def safe_div(a, b)
#   return 0 if b == 0
#   a / b
# end

# Default args
# def greet(name = "world", greeting = "Hello")
#   "#{greeting}, #{name}!"
# end

# Variable positional args (splat)
# def avg(*nums)
#   nums.sum.fdiv(nums.size)
# end
# avg(1,2,3)                       # 2.0

# Forwarding (... since 2.7)
# def forward(...)
#   inner(...)
# end

# Keyword args (preferred for readable APIs)
# def connect(host:, port: 80, ssl: false)
#   ...
# end
# connect(host: "x.com", ssl: true)
# connect()                        # ArgumentError: missing keyword :host

# Double-splat — collect remaining keywords
# def options(**opts)
#   opts                           # Hash of remaining
# end
# options(a:1, b:2)                # {a:1, b:2}

# Mix everything
# def f(req, opt = 0, *rest, kw:, kw_opt: nil, **extra, &blk)
#   ...
# end

# Block (&blk) — callable with .call or yielded
# def each_pair
#   yield 1, 2
#   yield 3, 4
# end
# each_pair { |a, b| puts "#{a},#{b}" }

# Endless method (Ruby 3.0+)
# def square(x) = x * x
# def url(host, port: 80) = "http://#{host}:#{port}"
# Note: endless def CANNOT have parens-less call site `def x = 1; x` works,
# but `def x = expr` body is a single expression; no `end` keyword.

# Predicate / bang methods
# def empty?; ... end              # convention: returns boolean
# def update!; ... end             # convention: mutates / raises

# Method visibility
# class C
#   def pub_a; end
#
#   private
#   def priv_a; end
#
#   public
#   def pub_b; end
#
#   protected
#   def prot_a; end
# end

# Caller without explicit receiver only — for private
# Receiver allowed for protected (within same class hierarchy)

# Method arity
# def f(a, b); end
# method(:f).arity                 # 2
# method(:f).parameters            # [[:req,:a],[:req,:b]]

# Removing a method
# undef_method :foo
# remove_method :foo               # subclass-only removal
```

## Blocks

```bash
# Two equivalent block syntaxes
# arr.each { |x| puts x }
# arr.each do |x|
#   puts x
# end
# Convention: braces for one-liners, do/end for multi-line.

# yield — execute the block passed to the current method
# def with_logging
#   puts "start"
#   yield 42
#   puts "end"
# end
# with_logging { |x| puts "got #{x}" }

# block_given? — check if a block was passed
# def maybe
#   if block_given?
#     yield
#   else
#     "no block"
#   end
# end

# Capturing the block as a Proc with &name
# def around(&blk)
#   t0 = Time.now
#   blk.call                       # or yield
#   Time.now - t0
# end

# Passing a Proc/Lambda/Method as a block with &
# squarer = ->(n) { n*n }
# [1,2,3].map(&squarer)            # [1,4,9]
# [1,2,3].map(&:to_s)              # uses Symbol#to_proc

# Anonymous block forwarding (Ruby 3.1+)
# def wrapper(&)
#   inner(&)
# end

# `it` block parameter (Ruby 3.4 stable / 3.3 preview) — avoid until 3.4 baseline
# arr.map { it * 2 }

# Closures — blocks capture surrounding locals
# x = 10
# inc = ->(y) { x + y }
# inc.call(5)                      # 15

# break / next inside blocks
# # break — stops the enclosing iterator and returns from the method
# # next  — moves to next iteration; with value, becomes block result
# arr.map { |x| next 0 if x.nil?; x*2 }
```

## Procs vs Lambdas vs Methods

```bash
# Three callable kinds: Proc, Lambda (special Proc), Method.

# Proc — lenient arity, `return` returns from the enclosing METHOD
# p1 = Proc.new { |a, b| [a, b] }
# p1.call(1)                       # [1, nil]   — extras nil, missing nil
# p1.call(1, 2, 3)                 # [1, 2]    — extras dropped
# p1 = proc { |a, b| [a, b] }      # same as Proc.new

# Lambda — strict arity, `return` returns from the LAMBDA only
# l1 = lambda { |a, b| [a, b] }
# l1 = ->(a, b) { [a, b] }         # stabby lambda
# l1.call(1)                       # ArgumentError: wrong number of arguments
# l1.lambda?                       # true

# Calling syntaxes
# p1.call(1,2)
# p1.(1,2)                         # alias of call
# p1[1, 2]                         # subscript alias
# p1.yield(1,2)                    # alias

# Demonstration: return semantics
# def proc_demo
#   p = proc { return 42 }
#   p.call
#   :never_reached                 # method already returned
# end
# proc_demo                        # 42

# def lambda_demo
#   l = lambda { return 42 }
#   l.call                         # 42 — local return from lambda
#   :reached
# end
# lambda_demo                      # :reached

# Method object — wraps a method as callable
# class C
#   def greet(name); "hi #{name}"; end
# end
# m = C.new.method(:greet)
# m.call("Sue")                    # "hi Sue"
# m.unbind                         # UnboundMethod
# arr.map(&m)                      # bind via &

# UnboundMethod — class-level
# um = C.instance_method(:greet)
# um.bind(C.new).call("Sue")
```

## Iteration / Enumerable

```bash
# Enumerable is the heart of Ruby. Mix in by defining .each — and you get the rest.

# Most-used
# [1,2,3].each            { |x| puts x }
# [1,2,3].map             { |x| x*2 }                  # [2,4,6]
# [1,2,3].select          { |x| x.odd? }               # [1,3]   alias filter
# [1,2,3].reject          { |x| x.odd? }               # [2]
# [1,2,3].reduce(0)       { |s, x| s + x }             # 6   alias inject
# [1,2,3].sum                                          # 6
# [1,2,3,1,2].tally                                    # {1=>2,2=>2,3=>1}
# [1,2,3].group_by        { |x| x.odd? }               # {true=>[1,3], false=>[2]}
# [1,2,3,4].partition     { |x| x.odd? }               # [[1,3],[2,4]]
# [1,2,3].each_with_index { |x, i| ... }
# [1,2,3].each_with_object([]) { |x, acc| acc << x*2 } # [2,4,6]

# Slicing/grouping
# (1..10).each_slice(3).to_a         # [[1,2,3],[4,5,6],[7,8,9],[10]]
# (1..10).each_cons(3).to_a          # [[1,2,3],[2,3,4],...]
# arr.chunk_while { |a, b| b - a == 1 }.to_a
# arr.slice_when  { |a, b| b - a != 1 }.to_a

# Take / drop
# (1..10).take(3)                    # [1,2,3]
# (1..10).drop(3)                    # [4,5,6,7,8,9,10]
# (1..10).take_while { |n| n < 5 }   # [1,2,3,4]
# (1..10).drop_while { |n| n < 5 }   # [5,6,7,8,9,10]

# Find / index
# arr.find    { |x| x > 5 }          # alias detect, returns first match
# arr.index   { |x| x > 5 }          # index of first match

# Min / max / sort
# arr.min / arr.max
# arr.minmax                         # [min, max]
# arr.min_by { |x| -x }
# arr.sort
# arr.sort_by { |x| -x }
# arr.sort   { |a, b| b <=> a }      # custom comparator

# Predicates
# arr.any?  arr.all?  arr.none?  arr.one?

# Lazy chains — avoid building large intermediate arrays
# (1..Float::INFINITY).lazy
#   .map  { |n| n*n }
#   .select { |n| n.odd? }
#   .first(5)                         # [1,9,25,49,81]

# zip — pair element-wise
# [1,2,3].zip([4,5,6])               # [[1,4],[2,5],[3,6]]
# [1,2,3].zip([4,5],[7,8,9])         # [[1,4,7],[2,5,8],[3,nil,9]]

# uniq with block (de-dup by key)
# users.uniq { |u| u[:email] }
```

## Enumerators

```bash
# An Enumerator is a lazy sequence of values you can consume on demand.

# From a method that yields
# enum = [10, 20, 30].each              # Enumerator (no block!)
# enum.next                              # 10
# enum.next                              # 20
# enum.next                              # 30
# enum.next                              # raises StopIteration

# From scratch
# fib = Enumerator.new do |y|
#   a, b = 0, 1
#   loop do
#     y << a
#     a, b = b, a + b
#   end
# end
# fib.take(10)                           # [0,1,1,2,3,5,8,13,21,34]

# Lazy — chain transforms WITHOUT realizing the intermediates
# (1..Float::INFINITY).lazy
#   .map    { |x| x * 2 }
#   .select { |x| x % 3 == 0 }
#   .first(5)

# Convert any block-yielding method
# "abc".each_char                        # Enumerator (no block)
# enum = "abc".each_char.with_index
# enum.to_a                              # [["a",0],["b",1],["c",2]]

# Useful one-liners
# Enumerator.new { |y| 5.times { |i| y.yield i } }.to_a
```

## Classes

```bash
# Definition
# class Person
#   # readers, writers, accessors generate methods
#   attr_reader   :name
#   attr_writer   :email
#   attr_accessor :age
#
#   def initialize(name:, age:, email: nil)
#     @name  = name
#     @age   = age
#     @email = email
#   end
#
#   def greet
#     "Hi, I'm #{@name}"
#   end
#
#   # Class method 1: explicit self
#   def self.from_hash(h)
#     new(name: h[:name], age: h[:age])
#   end
#
#   # Class method 2: open singleton class (preferred for several at once)
#   class << self
#     def default
#       new(name: "Anon", age: 0)
#     end
#   end
# end
#
# p = Person.new(name: "Sue", age: 30)
# p.greet                                # "Hi, I'm Sue"
# p.age                                  # 30
# p.age = 31                             # writer

# Inheritance via <
# class Employee < Person
#   def initialize(salary:, **rest)
#     super(**rest)                      # call Person#initialize
#     @salary = salary
#   end
# end

# super — calls same-named method on superclass
# super        — passes the SAME args as current method (no parens!)
# super()      — passes NO args
# super(x, y)  — explicit args

# Visibility
# class C
#   def pub; end
#   private
#   def hidden; end
#   protected
#   def shared; end
# end
# private/public/protected without args sets default for following defs
# private :foo, :bar                    # also works as method-form

# Ancestors
# Employee.ancestors
# # => [Employee, Person, Object, Kernel, BasicObject]
# Employee.superclass                   # Person

# Class introspection
# Person.instance_methods(false)        # methods defined directly
# Person.instance_methods               # plus inherited
# Person.method_defined?(:greet)
# p.respond_to?(:greet)
# p.is_a?(Person)                       # true (also via inheritance)
# p.instance_of?(Person)                # true (exact)

# Comparison via Comparable mixin (define <=>)
# class Version
#   include Comparable
#   def initialize(s); @parts = s.split(".").map(&:to_i); end
#   def <=>(other); @parts <=> other.instance_variable_get(:@parts); end
# end
# Version.new("1.2") < Version.new("1.10")  # true (1.2 < 1.10 numerically)

# Open classes (monkey-patching) — use with caution
# class String
#   def shout; upcase + "!"; end
# end
# "hi".shout                            # "HI!"

# Struct — quick value class
# Point = Struct.new(:x, :y) do
#   def magnitude; Math.sqrt(x*x + y*y); end
# end
# Point.new(3, 4).magnitude             # 5.0

# Data (Ruby 3.2+) — IMMUTABLE value object
# Point = Data.define(:x, :y)
# p = Point.new(x: 3, y: 4)
# p.with(y: 9)                          # Point(x=3, y=9)  — frozen, no setters
```

## Modules

```bash
# Modules serve TWO purposes: namespace and mixin (no instances).

# Namespace
# module MyApp
#   class User; end                     # MyApp::User
#   VERSION = "1.0"                     # MyApp::VERSION
#   def self.connect; end               # MyApp.connect (module function)
# end

# Mixin via include — adds INSTANCE methods
# module Speakable
#   def speak; "I say #{phrase}"; end
# end
# class Cat
#   include Speakable
#   def phrase; "meow"; end
# end
# Cat.new.speak                         # "I say meow"

# Mixin via extend — adds methods to ONE OBJECT (or as class methods)
# class Dog
#   extend Speakable                    # now Dog.speak works (class method!)
# end
# obj = Object.new
# obj.extend(Speakable)                 # only obj has speak

# include vs extend — common Idiom
# module Behaviour
#   module ClassMethods
#     def cm; "class method"; end
#   end
#   def im; "instance method"; end
#   def self.included(base)
#     base.extend(ClassMethods)
#   end
# end
# class Foo
#   include Behaviour
# end
# Foo.cm                                # "class method"
# Foo.new.im                            # "instance method"

# prepend — like include but inserted ABOVE in ancestor chain
# # Useful for monkey-patching where super still calls original
# module Auditing
#   def save
#     puts "before"
#     super                             # original save
#     puts "after"
#   end
# end
# class Article
#   prepend Auditing
#   def save; "saved"; end
# end

# module_function — make methods callable as Module.method AND included
# module Logger
#   module_function
#   def log(s); puts "[#{Time.now}] #{s}"; end
# end
# Logger.log "x"

# refine / using — scoped monkey-patch (use sparingly)
# module IntExt
#   refine Integer do
#     def shout; "#{self}!!!"; end
#   end
# end
# # using IntExt          # only inside this file/scope
```

## method_missing & respond_to_missing?

```bash
# When a method is not found, Ruby calls method_missing(name, *args, &blk).
# RULE: if you define method_missing, you MUST also define respond_to_missing?
# to keep .respond_to? and method() honest.

# class StringPainter
#   COLORS = %w[red green blue yellow]
#
#   def method_missing(name, *args, &blk)
#     return super unless COLORS.include?(name.to_s)
#     "<#{name}>#{args.first}</#{name}>"
#   end
#
#   def respond_to_missing?(name, include_private = false)
#     COLORS.include?(name.to_s) || super
#   end
# end
# sp = StringPainter.new
# sp.red("hi")                          # "<red>hi</red>"
# sp.respond_to?(:red)                  # true (because of respond_to_missing?)

# Common DSL idiom — used in Rails-style finders, OpenStruct, ostruct.

# Performance — method_missing is slower than a real method (extra lookup);
# in hot paths, define_method dynamically instead.
```

## Eigenclass / Singleton Class

```bash
# Every object has a HIDDEN class — its singleton class (a.k.a. eigenclass) —
# that holds methods unique to it.

# Define a method on a single object
# obj = "hello"
# def obj.shout; upcase + "!"; end
# obj.shout                             # "HELLO!"
# "world".shout                         # NoMethodError

# class << obj — open the singleton class
# class << obj
#   def whisper; downcase + "..."; end
# end

# For classes — class methods live in the class's singleton class
# class Foo
#   class << self
#     def bar; "hi"; end                # Foo.bar
#   end
# end

# Inspect singleton class
# obj.singleton_class                   # #<Class:#<Object:...>>
# obj.singleton_class.ancestors

# Define a singleton method dynamically
# obj.define_singleton_method(:wave) { "hi" }
```

## Mixin Idioms

```bash
# Comparable — define <=> (spaceship), get <, <=, >, >=, ==, between?
# class Money
#   include Comparable
#   attr_reader :cents
#   def initialize(c); @cents = c; end
#   def <=>(other); cents <=> other.cents; end
# end
# Money.new(100) < Money.new(200)       # true
# [Money.new(50), Money.new(10)].min    # Money(10)

# Enumerable — define .each, get map/select/reduce/min/max/sort/...
# class Box
#   include Enumerable
#   def initialize(*items); @items = items; end
#   def each(&blk); @items.each(&blk); end
# end
# b = Box.new(3, 1, 2)
# b.sort                                # [1,2,3]
# b.include?(2)                         # true
# b.map { |x| x*2 }                     # [6,2,4]

# Forwardable (stdlib) — delegate methods
# require 'forwardable'
# class CarFleet
#   extend Forwardable
#   def_delegators :@cars, :size, :each, :include?
#   def initialize; @cars = []; end
# end
```

## Exceptions

```bash
# Exception class hierarchy (top-down):
#   Exception                  — DO NOT rescue this; includes SystemExit, etc.
#     SignalException
#       Interrupt              — Ctrl+C
#     SystemExit               — `exit` — rescuing breaks shutdown
#     StandardError            — your default rescue boundary
#       RuntimeError           — `raise "msg"` defaults to this
#       ArgumentError
#       NameError              — bad constant/local
#         NoMethodError
#       TypeError
#       IOError
#       SystemCallError        — Errno::ENOENT, etc.
#       ZeroDivisionError
#       FrozenError            — mutating frozen object (was RuntimeError pre-3.0)
#       KeyError
#       StopIteration
#       NotImplementedError
#     ScriptError
#       LoadError, SyntaxError

# Basic begin/rescue/else/ensure
# begin
#   risky
# rescue ArgumentError => e
#   warn "bad arg: #{e.message}"
# rescue StandardError => e
#   warn "other: #{e.class}: #{e.message}"
#   warn e.backtrace.first(5).join("\n")
#   raise                                # re-raise
# else
#   puts "ran without raising"
# ensure
#   cleanup                              # ALWAYS runs (open files, locks)
# end

# Just `rescue` defaults to StandardError — NOT Exception
# rescue => e                             # same as: rescue StandardError => e

# Inline rescue (do NOT abuse — silently swallows errors)
# x = Integer("abc") rescue 0

# Raise
# raise                                  # re-raise current
# raise "boom"                           # RuntimeError
# raise ArgumentError                    # ArgumentError, default msg
# raise ArgumentError, "bad x"
# raise ArgumentError.new("bad x")
# raise CustomError, "bad x", caller

# Custom hierarchy
# module MyApp
#   class Error < StandardError; end
#   class NotFound < Error; end
#   class ValidationError < Error
#     def initialize(field); super("invalid #{field}"); @field = field; end
#     attr_reader :field
#   end
# end
# raise MyApp::NotFound

# retry — re-run the begin block (BEWARE infinite loops!)
# attempts = 0
# begin
#   attempts += 1
#   call_flaky_api
# rescue Net::OpenTimeout
#   retry if attempts < 3
#   raise
# end

# Methods can have rescue/ensure WITHOUT begin
# def fetch(url)
#   http.get(url)
# rescue => e
#   warn e
#   nil
# ensure
#   http.close
# end

# Refining your rescue: rescue StandardError, NOT Exception
# Reason: rescuing Exception swallows ^C, OutOfMemoryError, exit, threads, etc.
```

## Pattern Matching (3.0+)

```bash
# case/in — pattern matching on structure, types, and conditions

# Array patterns — fixed length unless rest *
# case [1, 2, 3]
# in [1, 2, 3]              then "exact"
# in [1, *rest]             then "starts with 1, rest=#{rest}"
# in [Integer, Integer, *]  then "two ints leading"
# in [*, last]              then "ends with #{last}"
# end

# Hash patterns — match by SUBSET of keys
# case user
# in {name: String => n, age: 18..}        then "adult #{n}"
# in {name: String => n, age: ..17}        then "minor #{n}"
# in {**}                                  then "any hash"
# end

# Variable binding & guards
# case point
# in [x, y] if x == y                      then "diagonal"
# in [x, 0]                                then "on x-axis at #{x}"
# end

# Pinning ^ — match against value of variable, don't bind
# expected = 42
# case [42, 5]
# in [^expected, _]                         then "matches expected"
# end

# Find pattern (Ruby 3.0+)
# case [1, 2, 3, 4, 5]
# in [*, 3, *]                              then "contains 3"
# end

# Class-level deconstruction — define .deconstruct / .deconstruct_keys
# class Point
#   attr_reader :x, :y
#   def initialize(x, y); @x, @y = x, y; end
#   def deconstruct; [x, y]; end
#   def deconstruct_keys(keys); {x:, y:}; end
# end
# case Point.new(1, 2)
# in [a, b]            then "array #{a},#{b}"
# in {x:, y:}          then "hash  #{x},#{y}"
# end

# in expression form (3.0+) — true/false test
# {a: 1, b: 2} in {a: Integer}              # true

# `=>` rightward match — bind or raise NoMatchingPatternError
# {name: "Sue", age: 30} => {name:, age:}
# # name == "Sue", age == 30
```

## Regex

```bash
# Literal /pattern/ or %r{pattern}
# /\d+/                                    # one or more digits
# %r{/api/users/(\d+)}                     # %r{} — slashes don't need escaping

# Flags
# /foo/i                                   # case-insensitive
# /foo.bar/m                               # multi-line: . matches \n
# /foo  bar/x                              # extended: ignore whitespace and # comments
# /foo/u                                   # treat string as UTF-8 (default in 1.9+)

# Match — three styles
# "hello world" =~ /world/                 # 6 (offset of match) or nil
# /world/ =~ "hello world"                 # also 6
# "hello world".match?(/world/)            # true (no MatchData created — fast)

# MatchData
# m = "2026-04-25".match(/(\d{4})-(\d{2})-(\d{2})/)
# m[0]  m[1]  m[2]  m[3]                   # full match, captures
# m.pre_match / m.post_match
# m.captures                               # ["2026","04","25"]
# m.named_captures                         # {} or {"y"=>...}
# # named groups
# m = "name=Sue".match(/name=(?<name>\w+)/)
# m[:name]                                 # "Sue"
# m["name"]                                # "Sue"

# After =~ (NOT match), specials are populated (thread-local!)
# "hi" =~ /(.)i/
# $~       # MatchData
# $1, $2   # capture groups
# $`       # pre-match
# $'       # post-match
# $&       # entire match

# Substitution
# "hello".sub(/l/, "L")                    # "heLlo"  — first only
# "hello".gsub(/l/, "L")                   # "heLLo"
# "hello".gsub(/l/) { |m| m.upcase }
# "USD 5, EUR 10".gsub(/(\w+) (\d+)/) { "#{$1}=#{$2}" }
# "abc abc".gsub(/(?<w>\w+)/, '\k<w>!')    # "abc! abc!"

# Hash form (since 1.9)
# "a b c".gsub(/\w/, "a"=>"1","b"=>"2","c"=>"3") # "1 2 3"

# scan — all matches as array
# "1 a 2 b 3".scan(/\d+/)                  # ["1","2","3"]
# "x=1 y=2".scan(/(\w+)=(\d+)/)            # [["x","1"],["y","2"]]

# split with regex
# "a, b,  c".split(/,\s*/)                 # ["a","b","c"]

# Common patterns
# /\A\d+\z/                                # \A start of string, \z end (vs ^/$ which match each line)
# /\bword\b/                               # word boundary
# /(?:foo|bar)/                            # non-capture
# /(?<year>\d{4})/                         # named capture
# /(?=foo)/                                # positive lookahead
# /(?!foo)/                                # negative lookahead
# /(?<=foo)/                               # positive lookbehind
# /(?<!foo)/                               # negative lookbehind

# DoS protection (3.2+)
# Regexp.timeout = 1.0                     # global timeout in seconds
# /(a+)+$/.timeout = 0.5                   # per-Regexp timeout
```

## File I/O

```bash
# Whole-file
# File.read("data.txt")                    # whole content as String
# File.binread("img.png")                  # binary mode
# File.readlines("data.txt", chomp: true)  # ["line1","line2",...]
# File.write("out.txt", "data")            # whole-file write
# File.write("out.txt", "data", mode: "a") # append

# Streaming
# File.open("big.txt", "r") do |f|
#   f.each_line { |line| puts line }
# end
# File.foreach("big.txt") { |line| ... }   # iterator, auto-closes

# Modes — passed as 2nd arg
# "r"   read (default)
# "w"   write, truncate, create
# "a"   append, create
# "r+"  read+write (no truncate)
# "w+"  read+write, truncate
# "a+"  read+append
# "rb" / "wb"  binary on Windows
# "r:UTF-8"     specify encoding

# File predicates
# File.exist?("p")                          # was File.exists? (deprecated)
# File.file?("p")     File.directory?("p")
# File.readable?  .writable?  .executable?
# File.zero?      .size("p")    .size?("p")  # size? returns nil for empty
# File.mtime("p") .ctime("p")  .atime("p")
# File.expand_path("~/x")                   # absolute path
# File.basename("/a/b/c.rb")                # "c.rb"
# File.dirname("/a/b/c.rb")                 # "/a/b"
# File.extname("c.rb")                      # ".rb"
# File.join("a","b","c.rb")                 # "a/b/c.rb"

# Pathname (require 'pathname') — OO file paths
# require 'pathname'
# p = Pathname.new("/tmp/foo.rb")
# p.exist?  p.read  p.write("x")  p.join("sub")
# p.parent  p.basename  p.extname

# Dir
# Dir.glob("**/*.rb")                       # recursive
# Dir.glob("*.{rb,md}")
# Dir.children(".")
# Dir.mkdir("new")                          # one level only
# Dir.exist?("p")
# Dir.pwd                                   # current
# Dir.chdir("/tmp") { ... }                 # block restores cwd

# FileUtils
# require 'fileutils'
# FileUtils.cp("a","b")
# FileUtils.cp_r("a","b")                   # recursive
# FileUtils.mv("a","b")
# FileUtils.rm("a")
# FileUtils.rm_rf("dir")                    # rm -rf
# FileUtils.mkdir_p("a/b/c")                # mkdir -p
# FileUtils.touch("x")
# FileUtils.chmod(0644, "x")
# FileUtils.chmod_R("u+rwX", "dir")

# Tempfile
# require 'tempfile'
# Tempfile.create("prefix") do |f|
#   f.write("data")
#   f.path
# end                                       # auto-deleted

# StringIO — file-like in-memory
# require 'stringio'
# io = StringIO.new("hello\nworld")
# io.gets                                   # "hello\n"

# Atomic write idiom
# require 'tempfile'
# Tempfile.create(["out", ".tmp"], dir: ".") do |t|
#   t.write(data); t.flush; t.fsync; t.close
#   File.rename(t.path, "out.txt")
# end
```

## JSON

```bash
# require 'json'                            # bundled with Ruby

# Parse
# JSON.parse('{"a":1}')                     # {"a"=>1} — STRING keys
# JSON.parse('{"a":1}', symbolize_names: true)  # {a: 1} — symbol keys
# JSON.parse(File.read("data.json"))

# Generate
# JSON.generate({a: 1})                     # '{"a":1}'
# {a: 1}.to_json                            # '{"a":1}'
# JSON.pretty_generate({a:1, b:[1,2]})      # multi-line, indented

# Custom serialization
# class Point
#   def to_json(*a); {x: @x, y: @y}.to_json(*a); end
# end

# Loading streams
# require 'json'
# JSON.load_file("data.json")               # like parse(File.read)

# Errors
# JSON.parse("nope")                        # JSON::ParserError

# OpenStruct from JSON
# require 'ostruct'
# o = JSON.parse('{"a":1}', object_class: OpenStruct)
# o.a                                       # 1
```

## YAML / CSV

```bash
# YAML — bundled (`psych` gem)
# require 'yaml'
# YAML.load_file("config.yml")              # UNSAFE for untrusted input
# YAML.safe_load(File.read("config.yml"),
#                permitted_classes: [Symbol, Date, Time])
# YAML.dump({a:1, b:[1,2]})                 # serialize
# obj.to_yaml

# Common gotcha: YAML.load was renamed to YAML.unsafe_load in newer psych;
# always prefer safe_load with explicit permitted_classes.

# Multi-document YAML
# YAML.load_stream(io) { |doc| ... }

# CSV — bundled
# require 'csv'
# CSV.read("users.csv", headers: true).each do |row|
#   row["name"]                             # column by header
# end
# CSV.foreach("users.csv", headers: true) { |row| ... }   # streaming

# Generate
# CSV.open("out.csv", "w") do |csv|
#   csv << ["name", "age"]
#   csv << ["Sue", 30]
# end
# CSV.generate { |csv| csv << [1,2,3] }

# Flags
# CSV.read("f.csv", col_sep: ";", headers: true, converters: :numeric)

# Parse a string
# CSV.parse("a,b\n1,2", headers: true)
```

## Subprocess

```bash
# system — run command, return true/false (exit success), inherits I/O
# system("ls", "-la")                       # true if exit 0
# system("nope") || warn("failed: #{$?.exitstatus}")
# $? is a Process::Status

# Backticks `cmd` — capture stdout, blocking
# out = `date`                              # "Sat Apr 25 ...\n"
# %x[date]                                  # equivalent

# IO.popen — bidirectional pipe
# IO.popen(["grep", "ERROR"], "r+") do |io|
#   io.puts "INFO ok"
#   io.puts "ERROR boom"
#   io.close_write
#   io.read                                 # "ERROR boom\n"
# end

# Open3 — PREFERRED for capture (require 'open3')
# require 'open3'
# stdout, stderr, status = Open3.capture3("ls", "-la", "/tmp")
# status.success?                           # true/false
# status.exitstatus                         # Integer
# # streaming
# Open3.popen3("python3") do |i, o, e, t|
#   i.puts "print(2+2)"
#   i.close
#   o.read                                  # "4\n"
# end
# # combined stdout+stderr
# out, status = Open3.capture2e("cmd", "arg")

# Process.spawn — non-blocking, returns PID; pair with Process.wait
# pid = Process.spawn("sleep", "1")
# Process.wait(pid)
# $?.success?

# Common pitfalls
#  - Use ARRAY form to avoid SHELL injection: system(["bin","arg"])
#  - String form goes through /bin/sh: system("ls #{user_input}")  -> DANGER
#  - Open3.capture3 also takes array-of-args safely

# Exit / abort the script
# exit                                       # 0
# exit 1
# exit!(2)                                   # skip at_exit hooks
# abort "fatal: bad config"                  # write to stderr + exit 1
```

## CLI

```bash
# ARGV — command-line args after script name
# ruby script.rb a b c                       # ARGV == ["a","b","c"]
# ARGV.shift                                 # "a"; ARGV == ["b","c"]

# OptionParser — stdlib (require 'optparse')
# require 'optparse'
# options = {}
# OptionParser.new do |opts|
#   opts.banner = "Usage: tool [options] FILE"
#
#   opts.on("-v", "--verbose", "Verbose output") do |v|
#     options[:verbose] = v
#   end
#
#   opts.on("-nN", "--num=N", Integer, "How many") do |n|
#     options[:num] = n
#   end
#
#   opts.on("-h", "--help") { puts opts; exit }
# end.parse!                                 # mutates ARGV, removes flags
#
# files = ARGV                               # what's left
# OptionParser handles --, abbreviated long options, default error msgs.

# GetoptLong (older) — also stdlib but rarely used now

# Reading input
# gets                                       # ONE line from ARGF (each ARGV file or STDIN)
# gets&.chomp                                # nil if EOF
# STDIN.gets                                 # explicit STDIN
# STDIN.read                                 # all of stdin
# STDIN.each_line { |l| ... }
# ARGF.each_line { |l| ... }                 # like awk: each line of all ARGV files

# Output streams
# puts "x"                                   # adds newline; arrays printed one per line
# print "x"                                  # no newline
# printf "%-10s %5d\n", "x", 42
# warn "broken"                              # stderr (use this for errors)
# pp obj                                     # pretty inspect to stdout
# $stdout.sync = true                        # unbuffered (good for pipes)

# Exit codes
# exit 0    # success
# exit 1    # generic error

# thor gem (richer DSL) — example
# require 'thor'
# class CLI < Thor
#   desc "greet NAME", "say hi"
#   option :loud, type: :boolean
#   def greet(name)
#     puts(options[:loud] ? "HI #{name}" : "hi #{name}")
#   end
# end
# CLI.start(ARGV)
```

## Environment

```bash
# ENV — Hash-like wrapper around environment variables (string keys/values)
# ENV["HOME"]                                # "/Users/me"
# ENV["FOO"] = "bar"                          # set
# ENV.delete("FOO")
# ENV.fetch("REQUIRED")                      # KeyError if missing
# ENV.fetch("OPTIONAL", "default")
# ENV.each { |k, v| puts "#{k}=#{v}" }

# Coerce types
# port = ENV.fetch("PORT", "3000").to_i
# debug = ENV["DEBUG"] == "true"

# dotenv gem — loads .env into ENV
# # Gemfile: gem 'dotenv'
# require 'dotenv/load'                      # auto-loads .env
# require 'dotenv'; Dotenv.load(".env.production")

# $LOAD_PATH (alias $:) — Ruby's require search path
# $LOAD_PATH.unshift(File.expand_path("../lib", __dir__))

# RUBYOPT — flags passed implicitly to ruby
# RUBYOPT='-W2 --yjit' ruby script.rb

# RUBY_VERSION  RUBY_PLATFORM  RUBY_ENGINE   # constants
```

## Concurrency

```bash
# CRITICAL: MRI Ruby has a Global VM Lock (GVL) — only ONE thread runs Ruby
# code at a time. Threads still help with I/O concurrency. For CPU parallelism
# use Process.fork or Ractor (3.0+).

# Threads
# t = Thread.new do
#   sleep 0.1
#   :done
# end
# t.value                                    # :done (joins implicitly)
# t.join                                     # wait
# t.alive?
# Thread.current
# Thread.list

# Mutex — guard shared state
# mu = Mutex.new
# mu.synchronize { @counter += 1 }

# Queue / SizedQueue — thread-safe producer/consumer
# q = Queue.new
# Thread.new { 5.times { |i| q << i }; q << :done }
# loop do
#   x = q.pop
#   break if x == :done
#   puts x
# end
# # SizedQueue.new(N) — bounded, blocks producer if full

# ConditionVariable — signal / wait
# cv = ConditionVariable.new
# mu.synchronize { cv.wait(mu) until ready? }
# mu.synchronize { ready!; cv.broadcast }

# Thread-local
# Thread.current[:user_id] = 42

# Process.fork — true OS-level parallelism, separate memory
# Process.fork do
#   # child
#   exit 0
# end
# Process.wait                               # wait for any child

# Ractor (Ruby 3.0+, experimental) — share-nothing parallelism
# r = Ractor.new do
#   Ractor.receive
# end
# r.send("hello")
# # All passed objects must be Ractor::Shareable (frozen / value-type) or copied.

# Fiber — cooperative concurrency, NOT parallel
# f = Fiber.new do
#   Fiber.yield 1
#   Fiber.yield 2
#   :done
# end
# f.resume                                   # 1
# f.resume                                   # 2
# # Fiber Scheduler (3.0+) used by Async gem to do non-blocking I/O

# Concurrent gems
# concurrent-ruby   — Promise/Future/Atomic/ThreadPool
# async             — Fiber-based event loop
# parallel          — easy fork-join over arrays
```

## Bundler

```bash
# Gemfile — declares dependencies
# # Gemfile
# source "https://rubygems.org"
# ruby "3.3.6"
#
# gem "rails", "~> 7.1"          # ~> 7.1 means >= 7.1, < 8.0 (pessimistic)
# gem "pg",    ">= 1.5.0"
# gem "puma",  "~> 6.0"
#
# group :development, :test do
#   gem "rspec-rails"
#   gem "pry"
# end
#
# group :test do
#   gem "factory_bot_rails"
# end
#
# gem "redis", require: false                # don't auto-load
# gem "mygem", git:    "https://x", branch: "main"
# gem "local", path:   "../local-gem"

# Commands
# bundle install              # install deps; writes/uses Gemfile.lock
# bundle install --jobs 4     # parallel
# bundle update               # update ALL gems (relaxes lock)
# bundle update rails         # update one gem
# bundle exec rspec           # run command in bundle context
# bundle lock                 # update Gemfile.lock without installing
# bundle outdated             # what could be upgraded
# bundle clean --force        # delete unused gems
# bundle config set --local path 'vendor/bundle'   # vendor in project
# bundle config set --local without 'production'   # skip a group
# bundle binstubs rspec       # generate ./bin/rspec
# bundle init                 # create empty Gemfile

# Gemfile.lock — COMMIT for apps; DON'T commit for libraries you publish

# Operators in version specs
# = 1.0           exact
# >= 1.0          at least
# > 1.0           strictly greater
# ~> 1.2          >= 1.2,  < 2.0
# ~> 1.2.3        >= 1.2.3, < 1.3.0  (more conservative)
# ~> 1.2, >= 1.2.5   combination
```

## Gem Authoring

```bash
# Skeleton
# bundle gem mygem            # creates lib/mygem.rb, mygem.gemspec, README, Rakefile, spec/
# cd mygem
# rake build                  # builds .gem in pkg/
# rake release                # tags + pushes to rubygems.org

# .gemspec
# # frozen_string_literal: true
# require_relative "lib/mygem/version"
# Gem::Specification.new do |s|
#   s.name        = "mygem"
#   s.version     = Mygem::VERSION
#   s.authors     = ["Sue"]
#   s.email       = ["sue@example.com"]
#   s.summary     = "Short summary"
#   s.description = "Longer description"
#   s.homepage    = "https://example.com"
#   s.license     = "MIT"
#   s.required_ruby_version = ">= 3.0"
#
#   s.files       = Dir["lib/**/*", "README.md", "LICENSE"]
#   s.bindir      = "exe"
#   s.executables = Dir.children("exe") rescue []
#   s.require_paths = ["lib"]
#
#   s.add_dependency "json", "~> 2.6"
#   s.add_development_dependency "minitest", "~> 5.0"
# end

# Version file (lib/mygem/version.rb)
# module Mygem
#   VERSION = "0.1.0"
# end

# Layout
#   lib/
#     mygem.rb                # require_relative "mygem/version"; require ...
#     mygem/version.rb
#     mygem/core.rb
#   exe/
#     mygem-tool              # executable; chmod +x; first line `#!/usr/bin/env ruby`
#   sig/                      # RBS type signatures (optional)
#   test/  or  spec/

# Build & publish
# gem build mygem.gemspec     # mygem-0.1.0.gem
# gem push mygem-0.1.0.gem    # to rubygems.org
# gem signin                  # OAuth/API key flow
# gem owner mygem -a name     # add maintainer

# Versioning — follow semver: MAJOR.MINOR.PATCH
#   PATCH — bug fixes
#   MINOR — backwards-compat features
#   MAJOR — breaking changes
```

## Rake

```bash
# Make-like build tool. File: Rakefile.
# # Rakefile
# require "rake/clean"
# CLEAN.include("tmp/**/*")
#
# task default: %i[test lint]
#
# desc "Run tests"
# task :test do
#   sh "ruby -Ilib -Itest test/test_*.rb"
# end
#
# desc "Lint"
# task :lint do
#   sh "rubocop"
# end
#
# namespace :db do
#   desc "Migrate"
#   task :migrate do
#     puts "migrating"
#   end
# end
#
# # File task
# file "out.txt" => "in.txt" do |t|
#   File.write(t.name, File.read("in.txt").upcase)
# end

# Commands
# rake -T                     # list documented tasks
# rake -T --all               # include undocumented
# rake test
# rake db:migrate             # namespaced
# rake -P                     # show prerequisites
# rake -W                     # where each task is defined
```

## Testing

```bash
# Minitest — bundled, lightweight (require 'minitest/autorun')
# require "minitest/autorun"
# require_relative "../lib/calculator"
#
# class CalculatorTest < Minitest::Test
#   def setup; @c = Calculator.new; end
#
#   def test_add
#     assert_equal 4, @c.add(2, 2)
#   end
#
#   def test_div_by_zero_raises
#     err = assert_raises(ZeroDivisionError) { @c.div(1, 0) }
#     assert_match(/zero/, err.message)
#   end
#
#   def test_predicate
#     assert_predicate [], :empty?
#     refute_predicate [1], :empty?
#   end
# end
# Common assertions:
#   assert / refute
#   assert_equal exp, act
#   assert_in_delta exp, act, 0.001
#   assert_nil  / refute_nil
#   assert_kind_of Class, obj
#   assert_match /re/, str
#   assert_includes coll, item
#   assert_raises(ErrClass) { ... }
#   assert_output(/stdout/, /stderr/) { ... }
#   skip "WIP" if true
# Run: ruby -Ilib -Itest test/some_test.rb

# Minitest::Spec — describe/it style
# require "minitest/autorun"
# describe Calculator do
#   it "adds" do
#     _(Calculator.new.add(1,1)).must_equal 2
#   end
# end

# RSpec — popular external gem
# # Gemfile
# group :development, :test do
#   gem "rspec"
# end
# bundle exec rspec --init                  # creates .rspec, spec/spec_helper.rb
# # spec/calc_spec.rb
# require "calculator"
# RSpec.describe Calculator do
#   subject(:calc) { described_class.new }
#
#   describe "#add" do
#     it "adds two ints" do
#       expect(calc.add(2,3)).to eq(5)
#     end
#
#     context "when given non-numeric" do
#       it "raises TypeError" do
#         expect { calc.add(:a, 1) }.to raise_error(TypeError)
#       end
#     end
#   end
# end
# # Run all
# bundle exec rspec
# bundle exec rspec spec/calc_spec.rb:12    # by file:line
# bundle exec rspec --tag focus
# # Mocks
# allow(api).to receive(:get).and_return("ok")
# expect(api).to receive(:post).with(/users/).once

# Other libraries
#  - mocha (mocking for Minitest)
#  - factory_bot (test data)
#  - vcr (record HTTP)
#  - simplecov (coverage)
```

## Tools

```bash
# Syntax check
ruby -c file.rb                              # "Syntax OK"
ruby -wc file.rb                             # plus warnings

# Warnings
ruby -w  file.rb                             # ordinary warnings
ruby -W2 file.rb                             # verbose
ruby -W:no-deprecated -W:no-experimental f.rb

# rubocop — community style linter
# Gemfile: gem "rubocop", require: false
bundle exec rubocop                          # lint everything
bundle exec rubocop -A path/                 # auto-correct (was --auto-correct-all)
bundle exec rubocop --display-cop-names
# .rubocop.yml — config
#   AllCops:
#     TargetRubyVersion: 3.3
#     NewCops: enable
#     Exclude: ['db/**/*']

# standardrb — opinionated wrapper around rubocop
# gem "standard"
bundle exec standardrb
bundle exec standardrb --fix

# Sorbet — static typing (gem "sorbet-static-and-runtime")
srb init
srb tc                                       # type check

# RBS — Ruby's own type signatures (sig/*.rbs)
# class User
#   attr_reader name: String
#   def initialize: (name: String) -> void
# end
# rbs / typeprof / steep validate

# brakeman — Rails security scanner
# gem "brakeman"
bundle exec brakeman

# reek — code smell detector
bundle exec reek lib/

# pry / debug
# # Gemfile (development): gem "debug" (Ruby 3.1+ ships it)
# require "debug"
# binding.break        # drops into debugger
# # commands: step, next, continue, info, eval

# IRB advanced
# irb> measure
# irb> ls Array      # list methods
# irb> show_source Array#map
# irb> help
```

## Performance

```bash
# Benchmark — stdlib
require 'benchmark'
# n = 100_000
# Benchmark.bm(10) do |x|
#   x.report("array")  { n.times { [1,2,3].include?(2) } }
#   x.report("set")    { s = [1,2,3].to_set; n.times { s.include?(2) } }
# end

# benchmark-ips gem — iterations per second (more accurate)
# require 'benchmark/ips'
# Benchmark.ips do |x|
#   x.report("a") { ... }
#   x.report("b") { ... }
#   x.compare!
# end

# YJIT — enable in production
# ruby --yjit script.rb
# RUBY_YJIT_ENABLE=1 ruby script.rb
# yjit-stats: --yjit-stats (in non-release builds)
# In Rails: YJIT auto-enabled when 1+ thread reaches threshold

# ObjectSpace
# ObjectSpace.count_objects                 # {TOTAL: ..., FREE: ..., T_STRING: ...}
# ObjectSpace.each_object(String).count
# ObjectSpace.memsize_of(obj)               # bytes (require 'objspace')

# GC — generational, incremental
# GC.start                                  # force a major GC
# GC.stat                                   # counts, heap pages
# GC.count                                  # gen counts
# GC.disable / GC.enable                    # rare, in tight benchmarks
# GC::Profiler.enable; ...; GC::Profiler.report

# Profiling
#  - stackprof gem      — sampling profiler, fast
#  - vernier   gem      — modern sampler (3.2+)
#  - ruby-prof          — instrumenting (slower, more detail)
#
# require 'stackprof'
# StackProf.run(mode: :cpu, out: "out.dump") { work }
# # stackprof out.dump --text
```

## Memory and GC

```bash
# Generational, incremental, compactable GC.

# Tuning via env vars (set BEFORE ruby starts)
# RUBY_GC_HEAP_INIT_SLOTS=600000          # initial heap size
# RUBY_GC_HEAP_FREE_SLOTS=200000
# RUBY_GC_HEAP_GROWTH_FACTOR=1.1          # default 1.8 — slower growth
# RUBY_GC_HEAP_GROWTH_MAX_SLOTS=300000
# RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR=2.0
# RUBY_GC_MALLOC_LIMIT=64000000
# RUBY_GC_OLDMALLOC_LIMIT=64000000

# GC compaction (Ruby 2.7+)
# GC.compact                                # synchronous compaction
# GC.auto_compact = true                    # background compaction (3.0+)

# Inspect
# GC.stat
# GC.stat[:major_gc_count]
# GC.latest_gc_info
# GC.measure_total_time = true ; GC.total_time

# WeakRef — weak references (require 'weakref')
# require 'weakref'
# obj = SomeBigThing.new
# w = WeakRef.new(obj)
# obj = nil
# GC.start
# w.weakref_alive?                         # false (collected)

# Allocation tracing
# require 'objspace'
# ObjectSpace.trace_object_allocations_start
# x = "hi"
# ObjectSpace.allocation_sourcefile(x)      # the file
# ObjectSpace.allocation_sourceline(x)      # the line

# Reduce allocations
#  - Freeze constants (frozen_string_literal magic comment)
#  - Use Symbol for keys
#  - .map → use Array#map! / .select! when safe
#  - Avoid intermediate arrays in long pipelines: use Enumerable#lazy
```

## Common Gotchas

```bash
# Truthiness — ONLY false and nil are falsy. Everything else (including 0,
# 0.0, "", [], {}) is TRUTHY.
# BROKEN
if response[:count]                         # bug: 0 is truthy!
  puts "has results"
end
# FIXED
if response[:count] && response[:count] > 0
  puts "has results"
end

# = vs == vs ===
# x = 1   assignment
# x == y  general equality (per class .==)
# x === y case-equality; Range/Class/Regexp override; not symmetric!
# Integer === 5    # true
# 5 === Integer    # false
# /\d+/ === "123"  # true
# "123" === /\d+/  # false (NoMatchingPattern... no, just false)

# Method call vs local var (no parens ambiguity)
# def foo; "method"; end
# foo                  # "method"
# foo = 1              # now `foo` is a LOCAL variable
# foo                  # 1
# self.foo             # forces method call → "method"
# foo()                # forces method call → "method"

# return inside a block returns from the ENCLOSING METHOD
# BROKEN
def first_match(items)
  items.each do |x|
    return x if x.even?     # this is fine, returns from first_match
  end
  nil
end
# But beware in a Proc captured outside a method (LocalJumpError)

# Mutable default arg via shared object
# BROKEN
def add(item, list = [])    # default array is REUSED across calls? NO — actually new each call
  list << item
end
# Ruby creates a NEW [] each call (unlike Python). Safe here.
# BUT if you cache: @list ||= [] in initialize and pass to multiple instances, watch aliasing.

# Modifying array while iterating
# BROKEN
arr = [1,2,3,4]
arr.each { |x| arr.delete(x) if x.even? }   # skips elements; "indexes shift"
# FIXED
arr.delete_if { |x| x.even? }
# OR
arr = arr.reject(&:even?)

# `=~` and $~/$1 are thread-local globals — fine across threads, but tricky
# across helper methods called from a regex sub block.
# BROKEN
"hi 5".gsub(/\d+/) { helper_that_uses_$1 }  # $1 set inside the block, but
# helper might run another regex and overwrite $1.
# FIXED
"hi 5".gsub(/\d+/) { |m| helper(m) }        # pass the match string explicitly

# String mutation on frozen strings
# BROKEN
"x".freeze << "y"                            # FrozenError
# FIXED
String.new("x") << "y"                       # mutable
# OR
"x" + "y"                                    # new string

# Default argument referencing instance var (none yet on first call)
# BROKEN
class C
  def f(x = @y)   # @y is nil if not set yet
    x
  end
end
# FIXED — set defaults in initialize, not in signature

# `for` leaks loop variable
# BROKEN
for i in 1..3; end
i  # => 3 (leaks!)
# FIXED — use each
(1..3).each { |i| ... }
i  # NameError: undefined local

# Hash short syntax with strings makes SYMBOLS
# BROKEN
h = {"a": 1}                                 # this is :a => 1, not "a" => 1
# FIXED
h = {"a" => 1}                               # explicit string key

# Symbol vs string lookup
# BROKEN
h = {a: 1}; h["a"]                           # nil
# FIXED
h[:a]   or  h.transform_keys(&:to_s)["a"]
```

## Idioms

```bash
# Block over loop
# # bad
# arr = []
# for i in 0..2
#   arr << i*2
# end
# # good
# arr = (0..2).map { |i| i*2 }

# Use safe-navigation operator &. for nil-safe chains
# user&.address&.city                        # nil if any is nil
# Equivalent to: user && user.address && user.address.city

# tap — execute a side effect and return self
# User.new(name: "Sue").tap { |u| u.save! }

# then / yield_self — pipe self through a block
# 5.then { |x| x*x }.then { |x| x + 1 }      # 26

# Numbered block params (2.7+)
# [[1,2],[3,4]].map { _1 + _2 }              # [3,7]
# # Discouraged for readability; explicit params usually clearer.

# Conditional assignment
# @cache ||= {}                              # set if nil/false
# h[k] ||= []                                # default-then-append idiom

# Multiple assignment
# a, b = b, a                                # swap
# first, *rest = arr

# Memoize
# def expensive
#   @result ||= compute
# end

# String building — prefer << for mutating concatenation in loops
# s = String.new
# items.each { |x| s << x.to_s << "\n" }

# Idiomatic each_with_object instead of inject + return acc
# people.each_with_object({}) { |p, h| h[p.id] = p.name }

# Comparable + Enumerable cover most "data class" needs
```

## Frozen / Immutability

```bash
# Symbols, Integers, Floats, true, false, nil — always immutable.
# Strings, Arrays, Hashes — mutable by default; can be frozen.

# Magic comment — at top of file
# # frozen_string_literal: true            # all string LITERALS in this file are frozen
# "hi".frozen?                              # true

# .freeze
# arr = [1,2,3].freeze
# arr << 4                                  # FrozenError

# Deep freeze idiom
# def deep_freeze(obj)
#   case obj
#   when Hash  then obj.each_value(&method(:deep_freeze)); obj.freeze
#   when Array then obj.each(&method(:deep_freeze));        obj.freeze
#   else obj.freeze
#   end
# end

# Ractor::Shareable — required for Ractor-passed objects
# Ractor.make_shareable(obj)                # deep-freeze + verify

# Data (3.2+) — by-design immutable value objects
# Point = Data.define(:x, :y)
# p = Point.new(x: 1, y: 2)
# p.x                                       # 1
# p.with(x: 9)                              # NEW Point(9, 2)
# p.x = 9                                   # NoMethodError — no setters
```

## Meta-programming

```bash
# define_method — define a method dynamically
# class C
#   [:red, :green, :blue].each do |color|
#     define_method("#{color}!") { @color = color }
#   end
# end

# class_eval / instance_eval / module_eval
# String.class_eval do
#   def shout; upcase + "!"; end
# end
# obj.instance_eval { @secret = 42 }        # set instance var from outside

# send / public_send — invoke by name (allows private with send)
# obj.send(:internal_method, *args)
# obj.public_send(:safe_method)             # raises if not public

# method / instance_method
# m = obj.method(:foo)                      # bound Method
# um = SomeClass.instance_method(:foo)      # UnboundMethod
# um.bind(other).call(...)
# Method#unbind  Method#owner  Method#source_location

# respond_to? / respond_to_missing?
# obj.respond_to?(:save)                    # true if it would respond
# obj.respond_to?(:save, true)              # include private

# instance_variables / instance_variable_get/set
# obj.instance_variables                    # [:@name, :@age]
# obj.instance_variable_get(:@name)
# obj.instance_variable_set(:@name, "x")

# const_get / const_set
# Object.const_get("MyApp::User")
# MyApp.const_set(:VERSION, "2.0")

# ancestors / included_modules
# Foo.ancestors
# Foo.included_modules

# Hooks — invoked at extension points
# class Module
#   def included(base);   end                # when this module is `include`d
#   def extended(base);   end                # when `extend`ed
#   def prepended(base);  end
# end
# class Class
#   def inherited(sub);   end
#   def method_added(name); end
#   def method_removed(name); end
#   def method_undefined(name); end
# end

# TracePoint — runtime introspection
# tp = TracePoint.new(:call) { |t| puts "#{t.path}:#{t.lineno} #{t.method_id}" }
# tp.enable { do_stuff }

# Caller stack
# caller                                    # array of "file:line:in `method`"
# caller_locations(0,5)                     # array of Thread::Backtrace::Location
```

## Common Error Messages

```bash
# NoMethodError: undefined method `foo' for #<C:0x...>
#   You called .foo on something that doesn't define it. Often a typo or wrong type.
# # Fix
# obj.respond_to?(:foo) or raise "no foo"

# NoMethodError: undefined method `foo' for nil:NilClass
#   You expected an object but got nil. Use &. or check earlier.
# # Fix
# user&.profile&.name
# user.profile&.name || "anon"

# NameError: undefined local variable or method `foo' for main:Object
#   Either typo or you didn't require/define it.
# # Fix — require, fix typo, or define before use

# NameError: uninitialized constant Foo
#   Constant not loaded. require it, or check namespace (Bar::Foo).
# # Fix
# require "foo"  # or  require_relative "foo"

# ArgumentError: wrong number of arguments (given X, expected Y)
#   Method arity mismatch.
# # Fix — match signature, or use *splat

# ArgumentError: missing keyword: :name
#   Required keyword arg not passed.
# # Fix
# foo(name: "Sue")

# TypeError: no implicit conversion of String into Integer
#   You tried arr["0"] or "x" + 1; types mismatch and no auto-coercion.
# # Fix
# "x" + 1.to_s   or  arr[0]

# FrozenError: can't modify frozen Array: [1,2,3]
#   Mutating a frozen object.
# # Fix
# arr.dup << x

# ZeroDivisionError: divided by 0
#   Integer / 0. Float / 0 returns Infinity instead.
# # Fix
# b.zero? ? 0 : a / b

# StopIteration: iteration reached an end
#   Enumerator.next called past the end.
# # Fix — use loop {} which catches StopIteration

# LoadError: cannot load such file -- foo
#   Can't find the file in $LOAD_PATH.
# # Fix
# require "bundler/setup"  or  $LOAD_PATH.unshift "lib"

# SyntaxError: ...
#   Missing end / def, mismatched braces.
# # Fix
# ruby -c file.rb         # locate

# SystemStackError: stack level too deep
#   Infinite recursion.

# RegexpError: invalid pattern in look-behind
#   Lookbehind needs fixed-width pattern in some engines (Ruby is OK with most).

# Encoding::CompatibilityError: incompatible character encodings: UTF-8 and ASCII-8BIT
#   Mixing encodings.
# # Fix
# str.force_encoding("UTF-8").encode("UTF-8", invalid: :replace)
```

## Tips

```bash
# Use `# frozen_string_literal: true` at the top of every .rb file.
# Prefer keyword arguments for any method with > 2 params or boolean params.
# Prefer Hash#fetch over Hash#[] when missing must error.
# Prefer Open3.capture3 over backticks/system for capturing output safely.
# Prefer ENV.fetch("KEY") over ENV["KEY"] for required vars.
# Prefer `unless x` over `if !x`; but never `unless` with `else` (rewrite as `if`).
# Prefer .each over .for; .for leaks the index variable.
# Prefer .find over .select(...).first.
# Prefer .map(&:foo) over .map { |x| x.foo }.
# Prefer `pp` over `puts` for non-trivial inspection; `p` is shortest.
# Use `binding.break` (require "debug") to drop into the stdlib debugger.
# Use `bundle exec` for any binary backed by a Gemfile dep.
# Use `~/` paths via File.expand_path("~/foo") — `~` is shell-only.
# Use `__dir__` and `__FILE__` instead of relative paths inside libraries.
# Use SecureRandom (require 'securerandom') for tokens, not rand.
# Use Time.now.utc.iso8601 for log timestamps; Time#strftime for custom.
# Run `ruby -W2 file.rb` periodically to catch latent warnings.
# Prefer Rake tasks named with verb-noun: `db:migrate`, `assets:precompile`.
# Always rescue StandardError, NEVER Exception (only catch the Exception base if you really know why).
# Always pair method_missing with respond_to_missing?.
# Always close I/O via the block form: File.open("p") { |f| ... }.
# Run YJIT in production (--yjit) for ~10-30% speedup on Rails workloads.
```

## See Also

- python, javascript, lua, regex, bash, polyglot, make

## References

- [ruby-lang.org documentation hub](https://www.ruby-lang.org/en/documentation/) -- canonical entry point
- [ruby-doc.org core API](https://docs.ruby-lang.org/en/master/) -- core class reference (per version)
- [ruby-doc.org stdlib](https://docs.ruby-lang.org/en/master/stdlibs.html) -- standard library reference
- [rubydoc.info](https://www.rubydoc.info/) -- gem and library docs (yard-based)
- [Ruby Style Guide (rubocop)](https://github.com/rubocop/ruby-style-guide) -- community conventions
- [Bundler](https://bundler.io/) -- dependency management
- [RubyGems](https://rubygems.org/) -- gem registry
- [Programming Ruby (Pickaxe)](https://pragprog.com/titles/ruby5/programming-ruby-3-3-5th-edition/) -- definitive book
- [Eloquent Ruby](https://www.amazon.com/Eloquent-Ruby-Addison-Wesley-Professional/dp/0321584104) -- idioms and conventions
- [Metaprogramming Ruby 2](https://pragprog.com/titles/ppmetr2/metaprogramming-ruby-2/) -- advanced metaprogramming
- [Ruby News](https://www.ruby-lang.org/en/news/) -- official release announcements
- [RBS (Ruby Signature)](https://github.com/ruby/rbs) -- type signature format
- [Sorbet](https://sorbet.org/) -- gradual typing
- [YJIT design doc](https://github.com/ruby/ruby/blob/master/doc/yjit/yjit.md) -- in-process JIT compiler
- [Ruby Forum / Discourse](https://discuss.ruby-lang.org/) -- community discussion
- [Ruby Weekly](https://rubyweekly.com/) -- newsletter, links to releases
