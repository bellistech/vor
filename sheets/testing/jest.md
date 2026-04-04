# Jest (JavaScript Testing Framework)

Delightful JavaScript testing framework with zero-config defaults, snapshot testing, and built-in mocking.

## Running Tests

### Basic execution

```bash
npx jest                            # run all tests
npx jest --watch                    # watch mode, re-run on changes
npx jest --watchAll                 # watch all files
npx jest src/utils.test.js          # run single file
npx jest -t "should parse"         # run tests matching name pattern
npx jest --verbose                  # show individual test results
npx jest --bail                     # stop on first failure
npx jest --runInBand                # run serially (no workers)
npx jest --changedSince=main        # only files changed since branch
npx jest --maxWorkers=4             # limit parallel workers
npx jest --forceExit                # force exit after tests complete
```

### Coverage

```bash
npx jest --coverage                 # generate coverage report
npx jest --coverage --coverageReporters=text-summary
npx jest --collectCoverageFrom='src/**/*.{js,ts}'
npx jest --coverageThreshold='{"global":{"branches":80,"lines":90}}'
```

## Writing Tests

### Basic structure

```js
describe('Calculator', () => {
  let calc;

  beforeAll(() => {
    // runs once before all tests in this describe
  });

  beforeEach(() => {
    calc = new Calculator();
  });

  afterEach(() => {
    calc.reset();
  });

  afterAll(() => {
    // runs once after all tests in this describe
  });

  it('should add two numbers', () => {
    expect(calc.add(2, 3)).toBe(5);
  });

  test('subtracts correctly', () => {
    expect(calc.subtract(5, 3)).toBe(2);
  });
});
```

### Skipping and focusing

```js
describe.skip('disabled suite', () => { ... });
describe.only('focused suite', () => { ... });
it.skip('disabled test', () => { ... });
it.only('focused test', () => { ... });
it.todo('implement pagination');
```

## Matchers

### Common matchers

```js
// Equality
expect(value).toBe(3);                      // strict ===
expect(obj).toEqual({ a: 1 });              // deep equality
expect(obj).toStrictEqual({ a: 1 });        // deep + type checking

// Truthiness
expect(value).toBeTruthy();
expect(value).toBeFalsy();
expect(value).toBeNull();
expect(value).toBeUndefined();
expect(value).toBeDefined();

// Numbers
expect(value).toBeGreaterThan(3);
expect(value).toBeGreaterThanOrEqual(3);
expect(value).toBeLessThan(5);
expect(0.1 + 0.2).toBeCloseTo(0.3);        // floating point

// Strings
expect(str).toMatch(/regex/);
expect(str).toContain('substring');

// Arrays and iterables
expect(arr).toContain('item');
expect(arr).toContainEqual({ id: 1 });
expect(arr).toHaveLength(3);

// Objects
expect(obj).toHaveProperty('key');
expect(obj).toHaveProperty('nested.key', 'value');
expect(obj).toMatchObject({ subset: true });

// Exceptions
expect(() => badFn()).toThrow();
expect(() => badFn()).toThrow(TypeError);
expect(() => badFn()).toThrow('specific message');
```

### Asymmetric matchers

```js
expect(obj).toEqual({
  id: expect.any(Number),
  name: expect.any(String),
  tags: expect.arrayContaining(['important']),
  metadata: expect.objectContaining({ version: 2 }),
  description: expect.stringContaining('test'),
  optional: expect.anything(),
});
```

## Snapshot Testing

### Basic snapshots

```js
it('renders correctly', () => {
  const tree = renderer.create(<Button label="Click" />).toJSON();
  expect(tree).toMatchSnapshot();
});

// Inline snapshots (stored in test file)
it('serializes config', () => {
  expect(getConfig()).toMatchInlineSnapshot(`
    {
      "debug": false,
      "port": 3000,
    }
  `);
});
```

```bash
npx jest --updateSnapshot              # update all snapshots
npx jest --updateSnapshot -t "config"  # update matching snapshots
```

## Mocking

### jest.fn() -- mock functions

```js
const mockCallback = jest.fn();
mockCallback.mockReturnValue(42);
mockCallback.mockReturnValueOnce(1).mockReturnValueOnce(2);
mockCallback.mockImplementation((x) => x * 2);

forEach([1, 2, 3], mockCallback);

expect(mockCallback).toHaveBeenCalledTimes(3);
expect(mockCallback).toHaveBeenCalledWith(1);
expect(mockCallback).toHaveBeenLastCalledWith(3);
expect(mockCallback).toHaveBeenNthCalledWith(2, 2);
```

### jest.spyOn()

```js
const spy = jest.spyOn(Math, 'random').mockReturnValue(0.5);
expect(rollDice()).toBe(4);
expect(spy).toHaveBeenCalled();
spy.mockRestore();                     // restore original
```

### Module mocking

```js
// Auto-mock entire module
jest.mock('./database');

// Manual mock with implementation
jest.mock('./api', () => ({
  fetchUser: jest.fn().mockResolvedValue({ id: 1, name: 'Alice' }),
  fetchPosts: jest.fn().mockResolvedValue([]),
}));

// Partial mock (keep some real implementations)
jest.mock('./utils', () => ({
  ...jest.requireActual('./utils'),
  generateId: jest.fn().mockReturnValue('test-id'),
}));
```

### Manual mocks (__mocks__ directory)

```js
// __mocks__/axios.js
export default {
  get: jest.fn().mockResolvedValue({ data: {} }),
  post: jest.fn().mockResolvedValue({ data: {} }),
};
// Tests automatically use this when jest.mock('axios') is called
```

## Async Testing

### Promises, async/await, callbacks

```js
// async/await
it('fetches users', async () => {
  const users = await fetchUsers();
  expect(users).toHaveLength(3);
});

// Promises
it('fetches users', () => {
  return fetchUsers().then(users => {
    expect(users).toHaveLength(3);
  });
});

// Resolves/rejects matchers
it('resolves with data', async () => {
  await expect(fetchUsers()).resolves.toHaveLength(3);
  await expect(badRequest()).rejects.toThrow('404');
});

// Callbacks (done)
it('calls back with data', (done) => {
  fetchData((err, data) => {
    expect(data).toBe('peanut butter');
    done();
  });
});
```

## Fake Timers

### Controlling time

```js
beforeEach(() => {
  jest.useFakeTimers();
});

afterEach(() => {
  jest.useRealTimers();
});

it('calls callback after 1 second', () => {
  const callback = jest.fn();
  setTimeout(callback, 1000);

  jest.advanceTimersByTime(999);
  expect(callback).not.toHaveBeenCalled();

  jest.advanceTimersByTime(1);
  expect(callback).toHaveBeenCalledTimes(1);
});

it('runs all pending timers', () => {
  const cb = jest.fn();
  setTimeout(cb, 5000);
  jest.runAllTimers();
  expect(cb).toHaveBeenCalled();
});
```

## Configuration

### jest.config.js

```js
module.exports = {
  testEnvironment: 'node',             // or 'jsdom' for browser
  roots: ['<rootDir>/src'],
  testMatch: ['**/__tests__/**/*.js', '**/*.test.js'],
  transform: { '^.+\\.tsx?$': 'ts-jest' },
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',    // path aliases
    '\\.(css|less)$': 'identity-obj-proxy',  // CSS modules
  },
  setupFilesAfterSetup: ['./jest.setup.js'],
  collectCoverageFrom: ['src/**/*.{js,ts}', '!src/**/*.d.ts'],
  coverageThreshold: {
    global: { branches: 80, functions: 80, lines: 80, statements: 80 },
  },
};
```

### Custom matchers

```js
// jest.setup.js
expect.extend({
  toBeWithinRange(received, floor, ceiling) {
    const pass = received >= floor && received <= ceiling;
    return {
      pass,
      message: () =>
        `expected ${received} ${pass ? 'not ' : ''}to be within [${floor}, ${ceiling}]`,
    };
  },
});

// Usage
expect(100).toBeWithinRange(90, 110);
```

## Tips

- Use `--watch` during development for instant feedback on changed files
- Prefer `toEqual` over `toBe` for objects and arrays -- `toBe` uses `Object.is`
- Use `toMatchObject` for partial object matching instead of cherry-picking fields
- Always `mockRestore()` spies in `afterEach` to prevent test pollution
- Use `jest.requireActual` in partial mocks to keep untouched exports working
- Use inline snapshots for small values to keep assertions visible in the test file
- Set `--maxWorkers=50%` in CI to avoid OOM on resource-constrained runners
- Use `jest.useFakeTimers()` to test debounce, throttle, and setTimeout logic deterministically
- Use `--bail` in CI to fail fast and save build minutes
- Use `describe.each` and `it.each` for data-driven table tests
- Avoid `--forceExit` in production CI -- it masks resource leak bugs
- Place `__mocks__` directories adjacent to the module they mock for automatic resolution

## See Also

- vitest
- mocha
- cypress
- testing-library
- playwright

## References

- [Jest Official Documentation](https://jestjs.io/docs/getting-started)
- [Jest Expect API Reference](https://jestjs.io/docs/expect)
- [Jest Mock Functions](https://jestjs.io/docs/mock-functions)
- [Jest Configuration Reference](https://jestjs.io/docs/configuration)
- [ts-jest Documentation](https://kulshekhar.github.io/ts-jest/)
