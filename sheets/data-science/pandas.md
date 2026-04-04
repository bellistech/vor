# pandas (Python Data Analysis Library)

pandas is the foundational Python library for data manipulation and analysis, providing the DataFrame and Series data structures for working with labeled, heterogeneous tabular data with integrated indexing, alignment, grouping, reshaping, and I/O capabilities.

## DataFrame and Series Creation
### From Various Sources
```python
import pandas as pd
import numpy as np

# From dictionary
df = pd.DataFrame({
    'name': ['Alice', 'Bob', 'Charlie', 'Diana'],
    'age': [25, 30, 35, 28],
    'salary': [70000, 85000, 120000, 95000],
    'dept': ['Eng', 'Sales', 'Eng', 'Sales']
})

# From list of dicts
records = [{'x': 1, 'y': 2}, {'x': 3, 'y': 4, 'z': 5}]
df = pd.DataFrame(records)  # missing keys become NaN

# Series from array
s = pd.Series([10, 20, 30], index=['a', 'b', 'c'], name='values')

# From NumPy array with column names
arr = np.random.randn(100, 4)
df = pd.DataFrame(arr, columns=['A', 'B', 'C', 'D'])

# Date range index
dates = pd.date_range('2024-01-01', periods=365, freq='D')
ts = pd.DataFrame({'value': np.random.randn(365)}, index=dates)
```

## Indexing and Selection
### loc (label-based) and iloc (position-based)
```python
# loc: label-based indexing
df.loc[0, 'name']                    # single value
df.loc[0:2, ['name', 'salary']]      # slice rows, select columns
df.loc[df['age'] > 28, 'name']       # boolean mask with column
df.loc[lambda d: d['salary'].gt(80000)]  # callable

# iloc: integer position-based
df.iloc[0]                            # first row as Series
df.iloc[0:3, 1:3]                     # row/col slices
df.iloc[[0, 2], [1, 3]]              # specific positions

# Boolean indexing
mask = (df['age'] > 25) & (df['dept'] == 'Eng')
df[mask]

# query() for readable filtering
df.query('age > 25 and dept == "Eng"')
df.query('salary > @threshold')       # reference Python variables

# Setting values
df.loc[df['dept'] == 'Eng', 'bonus'] = 5000
df.iloc[0, 2] = 75000
```

## GroupBy and Aggregation
### Split-Apply-Combine
```python
# Basic groupby
df.groupby('dept')['salary'].mean()
df.groupby('dept').agg({'salary': ['mean', 'median', 'std'],
                         'age': ['min', 'max']})

# Named aggregations (clean column names)
df.groupby('dept').agg(
    avg_salary=('salary', 'mean'),
    headcount=('name', 'count'),
    oldest=('age', 'max')
)

# Multiple groupby columns
df.groupby(['dept', 'role']).agg(
    total_salary=('salary', 'sum'),
    avg_age=('age', 'mean')
).reset_index()

# Custom aggregation functions
df.groupby('dept')['salary'].agg(lambda x: x.quantile(0.75) - x.quantile(0.25))

# Transform (returns same-shaped DataFrame)
df['salary_zscore'] = df.groupby('dept')['salary'].transform(
    lambda x: (x - x.mean()) / x.std()
)

# Filter groups
df.groupby('dept').filter(lambda g: g['salary'].mean() > 80000)
```

## Merge, Join, and Concat
### Combining DataFrames
```python
# merge (SQL-style joins)
orders = pd.DataFrame({'order_id': [1, 2, 3], 'user_id': [101, 102, 101]})
users = pd.DataFrame({'user_id': [101, 102, 103], 'name': ['A', 'B', 'C']})

pd.merge(orders, users, on='user_id', how='inner')     # default
pd.merge(orders, users, on='user_id', how='left')      # keep all orders
pd.merge(orders, users, on='user_id', how='outer')     # keep everything

# Merge on different column names
pd.merge(df1, df2, left_on='emp_id', right_on='employee_id')

# Multi-key merge
pd.merge(df1, df2, on=['dept', 'year'])

# Merge with indicator
result = pd.merge(df1, df2, on='id', how='outer', indicator=True)
result[result['_merge'] == 'left_only']   # anti-join

# concat (stack DataFrames)
pd.concat([df1, df2], axis=0, ignore_index=True)       # vertical stack
pd.concat([df1, df2], axis=1)                            # horizontal stack

# join (index-based merge)
df1.set_index('key').join(df2.set_index('key'), how='left')
```

## Pivot and Reshape
### pivot_table and melt
```python
# Pivot table (like Excel pivot)
pd.pivot_table(
    df,
    values='salary',
    index='dept',
    columns='role',
    aggfunc='mean',
    fill_value=0,
    margins=True        # add row/column totals
)

# Simple pivot (no aggregation, values must be unique)
df.pivot(index='date', columns='category', values='amount')

# Melt (wide to long)
wide = pd.DataFrame({
    'name': ['Alice', 'Bob'],
    'math': [90, 85],
    'science': [88, 92],
    'english': [95, 78]
})
pd.melt(wide, id_vars='name', var_name='subject', value_name='score')

# Stack/unstack
df.set_index(['dept', 'role']).stack()
df.set_index(['dept', 'role'])['salary'].unstack(fill_value=0)

# Crosstab
pd.crosstab(df['dept'], df['role'], values=df['salary'], aggfunc='mean')
```

## Apply, Map, and Vectorized Operations
### Element-wise and Row/Column Operations
```python
# Vectorized operations (fastest)
df['tax'] = df['salary'] * 0.3
df['full_name'] = df['first'] + ' ' + df['last']

# apply (row-wise or column-wise)
df['category'] = df['salary'].apply(
    lambda x: 'senior' if x > 100000 else 'junior'
)
df.apply(lambda row: row['salary'] / row['age'], axis=1)

# map (Series only, for element-wise mapping)
df['dept_full'] = df['dept'].map({'Eng': 'Engineering', 'Sales': 'Sales Dept'})

# where/mask (conditional assignment)
df['adjusted'] = df['salary'].where(df['dept'] == 'Eng', df['salary'] * 1.1)

# np.select for multiple conditions
conditions = [df['salary'] > 100000, df['salary'] > 70000]
choices = ['high', 'mid']
df['band'] = np.select(conditions, choices, default='low')
```

## I/O Operations
### Reading and Writing Data
```python
# CSV
df = pd.read_csv('data.csv', parse_dates=['date'], dtype={'id': str})
df = pd.read_csv('data.csv', usecols=['name', 'salary'], nrows=1000)
df.to_csv('output.csv', index=False)

# Parquet (columnar, compressed)
df = pd.read_parquet('data.parquet', columns=['name', 'salary'])
df.to_parquet('output.parquet', engine='pyarrow', compression='snappy')

# SQL
from sqlalchemy import create_engine
engine = create_engine('postgresql://user:pass@host/db')
df = pd.read_sql('SELECT * FROM users WHERE active = true', engine)
df = pd.read_sql_table('users', engine)
df.to_sql('users_backup', engine, if_exists='replace', index=False)

# Excel
df = pd.read_excel('report.xlsx', sheet_name='Q4', header=1)
with pd.ExcelWriter('report.xlsx') as writer:
    df1.to_excel(writer, sheet_name='Summary')
    df2.to_excel(writer, sheet_name='Details')

# JSON
df = pd.read_json('data.json', orient='records')
df.to_json('output.json', orient='records', lines=True)
```

## String and DateTime Accessors
### str and dt Accessors
```python
# String accessor
df['name'].str.lower()
df['name'].str.contains('ali', case=False)
df['name'].str.split(' ', expand=True)
df['email'].str.extract(r'@(\w+\.\w+)')
df['text'].str.replace(r'\d+', '', regex=True)
df['code'].str.zfill(5)              # zero-pad to 5 chars
df['tags'].str.get_dummies(sep=',')   # one-hot from delimited

# DateTime accessor
df['date'] = pd.to_datetime(df['date_str'])
df['year'] = df['date'].dt.year
df['month'] = df['date'].dt.month
df['weekday'] = df['date'].dt.day_name()
df['quarter'] = df['date'].dt.quarter
df['is_weekend'] = df['date'].dt.dayofweek >= 5

# Time-based operations
df.set_index('date').resample('M').sum()       # monthly
df.set_index('date').resample('W').mean()      # weekly average
df['rolling_avg'] = df['value'].rolling(window=7).mean()
df['expanding_sum'] = df['value'].expanding().sum()
```

## Missing Data Handling
### fillna, dropna, and Interpolation
```python
# Detection
df.isna().sum()                    # count NaN per column
df.isna().sum().sum()              # total NaN count
df[df.isna().any(axis=1)]         # rows with any NaN

# Dropping
df.dropna()                        # drop rows with any NaN
df.dropna(subset=['salary'])       # only check specific columns
df.dropna(thresh=3)                # keep rows with at least 3 non-NaN

# Filling
df['salary'].fillna(df['salary'].median(), inplace=False)
df.fillna(method='ffill')          # forward fill
df.fillna(method='bfill', limit=2) # backward fill, max 2
df['value'].interpolate(method='linear')
df['value'].interpolate(method='time')   # time-weighted

# Replace specific values
df.replace({999: np.nan, -1: np.nan})
df['status'].replace({'unknown': np.nan}, inplace=True)
```

## Value Counts and Descriptive Stats
### Quick Exploration
```python
# Value counts
df['dept'].value_counts()                     # counts
df['dept'].value_counts(normalize=True)       # proportions
df['salary'].value_counts(bins=5)             # binned counts

# Descriptive statistics
df.describe()                                  # numeric summary
df.describe(include='all')                     # include categoricals
df.info()                                      # dtypes, memory, non-null
df.memory_usage(deep=True)                     # actual memory per column

# Correlation
df[['salary', 'age', 'experience']].corr()     # Pearson correlation
df[['salary', 'age']].corr(method='spearman')  # rank correlation

# Unique values
df['dept'].nunique()
df['dept'].unique()
```

## Tips
- Use `pd.read_csv(..., dtype={...})` to specify dtypes upfront and avoid costly type inference on large files
- Prefer vectorized operations and `np.where`/`np.select` over `apply` with lambdas for 10-100x speed gains
- Use `category` dtype for string columns with low cardinality to reduce memory usage by 90% or more
- Chain operations with `.pipe()` for readable data transformation pipelines
- Use `pd.eval()` and `df.query()` for complex boolean expressions; they are faster and more readable than bracket notation
- Always set `parse_dates` in `read_csv` rather than converting afterward to avoid double-pass overhead
- Use Parquet instead of CSV for production data pipelines; it preserves dtypes and is 5-10x faster to read
- Use `.astype('Int64')` (capital I) for nullable integer columns instead of converting to float
- Profile memory with `df.memory_usage(deep=True)` before and after optimizations to verify improvements
- Avoid chained indexing (`df[col][row]`); use `.loc` or `.iloc` to prevent SettingWithCopyWarning

## See Also
- numpy, scikit-learn, jupyter, datalakes, polars

## References
- [pandas Official Documentation](https://pandas.pydata.org/docs/)
- [pandas User Guide](https://pandas.pydata.org/docs/user_guide/index.html)
- [pandas API Reference](https://pandas.pydata.org/docs/reference/index.html)
- [Modern Pandas (Tom Augspurger)](https://tomaugspurger.github.io/posts/modern-1-intro/)
