# Ruby (Dynamic, Object-Oriented Scripting Language)

An expressive, everything-is-an-object language designed for programmer happiness.

## Variables and Data Types

### Variables

```ruby
# Local variable
name = "Alice"

# Instance variable (belongs to an object)
@name = "Alice"

# Class variable (shared across instances)
@@count = 0

# Global variable (avoid these)
$debug = true

# Constant
MAX_SIZE = 100

# Parallel assignment
a, b, c = 1, 2, 3

# Swap
a, b = b, a
```

### Data Types

```ruby
# Numbers
age = 30              # Integer
price = 19.99         # Float
big = 1_000_000       # underscores for readability

# Strings
name = "Alice"
greeting = 'Hello'

# Symbols (immutable, memory-efficient identifiers)
status = :active

# Booleans and nil
active = true
deleted = false
value = nil            # represents "nothing"
```

## Strings

```ruby
# Interpolation (double quotes only)
name = "world"
puts "Hello, #{name}!"        # Hello, world!

# Common methods
"hello".upcase                 # "HELLO"
"HELLO".downcase               # "hello"
"hello".capitalize             # "Hello"
"hello".reverse                # "olleh"
"hello".length                 # 5
"hello world".split            # ["hello", "world"]
"hello".include?("ell")        # true
"hello".gsub("l", "r")         # "herro"
"  hello  ".strip              # "hello"
"hello" * 3                    # "hellohellohello"
"hello".chars                  # ["h", "e", "l", "l", "o"]
"hello".freeze                 # make immutable

# Multiline (heredoc)
text = <<~HEREDOC
  Line one
  Line two
HEREDOC
```

## Arrays

```ruby
# Creation
arr = [1, 2, 3, 4, 5]
words = %w[foo bar baz]        # array of strings without quotes

# Access
arr[0]                         # 1 (first)
arr[-1]                        # 5 (last)
arr[1..3]                      # [2, 3, 4] (range slice)

# Common methods
arr.push(6)                    # append (also arr << 6)
arr.pop                        # remove and return last
arr.shift                      # remove and return first
arr.unshift(0)                 # prepend
arr.flatten                    # flatten nested arrays
arr.compact                    # remove nils
arr.uniq                       # remove duplicates
arr.sort                       # sort ascending
arr.reverse                    # reverse order
arr.zip([10, 20, 30])          # pair elements
arr.sample                     # random element
```

## Hashes

```ruby
# Symbol keys (preferred in modern Ruby)
person = { name: "Alice", age: 30, role: :admin }

# String keys
config = { "host" => "localhost", "port" => 3000 }

# Access and mutation
person[:name]                  # "Alice"
person[:email] = "a@test.com"  # add key
person.fetch(:name)            # "Alice" (raises if missing)
person.fetch(:zip, "N/A")     # "N/A" (default if missing)
person.keys                    # [:name, :age, :role]
person.values                  # ["Alice", 30, :admin]
person.key?(:name)             # true
person.merge(active: true)     # returns new hash with addition
person.delete(:role)           # removes and returns value
```

## Control Flow

```ruby
# if / elsif / else
if score > 90
  "A"
elsif score > 80
  "B"
else
  "C"
end

# Inline if (modifier form)
puts "adult" if age >= 18

# unless (inverse of if)
puts "minor" unless age >= 18

# Ternary
status = age >= 18 ? "adult" : "minor"

# case / when
case day
when "Monday"..  "Friday"
  "weekday"
when "Saturday", "Sunday"
  "weekend"
end

# case with pattern matching (Ruby 3+)
case [1, 2, 3]
in [Integer => a, Integer => b, *]
  puts "first two: #{a}, #{b}"
end
```

## Loops and Iteration

```ruby
# each (preferred for iteration)
[1, 2, 3].each { |n| puts n }

# each with block
[1, 2, 3].each do |n|
  puts n * 2
end

# map (transform elements, returns new array)
doubled = [1, 2, 3].map { |n| n * 2 }       # [2, 4, 6]

# select (filter elements that match)
evens = (1..10).select { |n| n.even? }       # [2, 4, 6, 8, 10]

# reject (filter elements that don't match)
odds = (1..10).reject { |n| n.even? }        # [1, 3, 5, 7, 9]

# reduce (accumulate into single value)
sum = [1, 2, 3, 4].reduce(0) { |acc, n| acc + n }   # 10

# times, upto, downto
5.times { |i| puts i }                       # 0 through 4
1.upto(5) { |i| puts i }                     # 1 through 5
5.downto(1) { |i| puts i }                   # 5 through 1

# while and until
i = 0
while i < 5
  i += 1
end
```

## Methods

```ruby
# Basic method
def greet(name)
  "Hello, #{name}!"               # implicit return of last expression
end

# Default parameters
def greet(name = "world")
  "Hello, #{name}!"
end

# Variable arguments (splat)
def sum(*numbers)
  numbers.reduce(0, :+)
end

# Keyword arguments
def connect(host:, port: 3000, ssl: false)
  "#{ssl ? 'https' : 'http'}://#{host}:#{port}"
end
connect(host: "example.com", ssl: true)

# Method returning boolean (convention: end with ?)
def empty?(arr)
  arr.length == 0
end

# Destructive method (convention: end with !)
def upcase!(str)
  str.replace(str.upcase)
end
```

## Blocks, Procs, and Lambdas

```ruby
# Block (passed to a method)
def with_logging
  puts "Start"
  yield                            # execute the block
  puts "End"
end
with_logging { puts "Working..." }

# Proc (stored block, lenient arity)
double = Proc.new { |n| n * 2 }
double.call(5)                     # 10

# Lambda (strict arity, returns to caller)
square = ->(n) { n ** 2 }
square.call(4)                     # 16

# Passing a proc/lambda as a block with &
[1, 2, 3].map(&square)            # [1, 4, 9]

# Symbol#to_proc shorthand
["hello", "world"].map(&:upcase)   # ["HELLO", "WORLD"]
```

## Classes and Modules

```ruby
# Class definition
class Animal
  attr_accessor :name, :sound      # getter and setter
  attr_reader :species             # getter only

  def initialize(name, species)
    @name = name
    @species = species
  end

  def speak
    "#{@name} says #{@sound}"
  end
end

# Inheritance
class Dog < Animal
  def initialize(name)
    super(name, "Canis lupus")
    @sound = "Woof"
  end
end

rex = Dog.new("Rex")
rex.speak                          # "Rex says Woof"

# Module (mixin for shared behavior)
module Printable
  def to_display
    instance_variables.map { |v| "#{v}: #{instance_variable_get(v)}" }.join(", ")
  end
end

class Report
  include Printable                # adds instance methods
end

# extend adds module methods as class methods
class Config
  extend Printable
end
```

## Exception Handling

```ruby
# begin / rescue / ensure
begin
  result = 10 / 0
rescue ZeroDivisionError => e
  puts "Error: #{e.message}"
rescue StandardError => e
  puts "Other error: #{e.message}"
ensure
  puts "This always runs"
end

# Inline rescue
value = Integer("abc") rescue nil   # nil instead of exception

# Raise custom errors
class AppError < StandardError; end

def risky_operation
  raise AppError, "Something went wrong"
end

# Retry
attempts = 0
begin
  attempts += 1
  do_something_flaky
rescue => e
  retry if attempts < 3
  raise
end
```

## File I/O

```ruby
# Read entire file
content = File.read("data.txt")

# Read lines into array
lines = File.readlines("data.txt", chomp: true)

# Write to file (overwrites)
File.write("output.txt", "Hello, world!")

# Append to file
File.open("log.txt", "a") { |f| f.puts "New entry" }

# Read line by line (memory efficient)
File.foreach("large.txt") { |line| puts line }

# Check file existence
File.exist?("config.yml")
```

## Regex

```ruby
# Match test
"hello world" =~ /world/          # 6 (index of match)
"hello world".match?(/world/)     # true

# Capture groups
if m = "2026-03-25".match(/(\d{4})-(\d{2})-(\d{2})/)
  m[1]                            # "2026"
  m[2]                            # "03"
end

# Global substitution
"foo bar foo".gsub(/foo/, "baz")  # "baz bar baz"

# Scan (find all matches)
"one 1 two 2 three 3".scan(/\d+/) # ["1", "2", "3"]
```

## Gems and Bundler

```bash
# Install a gem
gem install httparty

# Create a Gemfile and install dependencies
bundle init
bundle add rails
bundle install

# Execute in bundle context
bundle exec rspec

# Update gems
bundle update
```

## Common One-Liners

```ruby
# Sum an array
[1, 2, 3, 4].sum                          # 10

# Flatten and compact
[[1, nil], [2, [3]]].flatten.compact       # [1, 2, 3]

# Group by
%w[ant bear cat].group_by { |w| w.length } # {3=>["ant","cat"], 4=>["bear"]}

# Tally (count occurrences, Ruby 2.7+)
%w[a b a c b a].tally                      # {"a"=>3, "b"=>2, "c"=>1}

# Chain enumerable methods
(1..100).select(&:even?).map { |n| n ** 2 }.first(5)
```

## Tips

- Ruby evaluates everything as truthy except `false` and `nil`. Zero, empty strings, and empty arrays are all truthy.
- Prefer `each` over `for`; idiomatic Ruby uses iterator methods on collections.
- Use `freeze` on string constants to prevent accidental mutation and improve performance.
- The `&:method_name` shorthand works with any method that takes no arguments.
- Use `pp` (pretty print) instead of `puts` when debugging complex objects.
- Bundler's `Gemfile.lock` should be committed for applications but not for gems/libraries.
- Methods ending in `?` return booleans; methods ending in `!` mutate in place or raise on failure.
- Use `irb` or `pry` for interactive exploration and debugging.
- Prefer keyword arguments for methods with more than two parameters for readability.
- Ruby 3+ introduced Ractors for parallelism and typed signatures via RBS.

## See Also

- python, javascript, lua, yaml, json, regex

## References

- [Ruby Documentation](https://www.ruby-lang.org/en/documentation/) -- official guides and resources
- [Ruby Core API](https://ruby-doc.org/core/) -- core class and module reference
- [Ruby Standard Library](https://ruby-doc.org/stdlib/) -- stdlib module reference
- [Ruby Language Specification (ISO/IEC 30170)](https://www.iso.org/standard/59579.html) -- formal ISO standard
- [RubyGems](https://rubygems.org/) -- package registry for Ruby libraries
- [Bundler](https://bundler.io/) -- dependency management for Ruby projects
- [Ruby Style Guide](https://rubystyle.guide/) -- community-maintained coding conventions
- [Ruby News](https://www.ruby-lang.org/en/news/) -- official release announcements and changelogs
- [RBS (Ruby Signature)](https://github.com/ruby/rbs) -- type signature format for Ruby 3+
- [Ruby Forum](https://discuss.ruby-lang.org/) -- official community discussion
