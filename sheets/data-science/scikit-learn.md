# scikit-learn (Machine Learning in Python)

scikit-learn is Python's most widely used machine learning library, providing a consistent estimator API with fit/predict/transform methods, comprehensive tools for preprocessing, model selection, evaluation, and pipelining, supporting classification, regression, clustering, and dimensionality reduction across dozens of well-tested algorithms.

## Estimator API
### Core Interface Pattern
```python
from sklearn.ensemble import RandomForestClassifier
from sklearn.linear_model import LinearRegression
from sklearn.cluster import KMeans

# Every estimator follows the same pattern:
# 1. Instantiate with hyperparameters
model = RandomForestClassifier(n_estimators=100, max_depth=10, random_state=42)

# 2. Fit to training data
model.fit(X_train, y_train)

# 3. Predict
y_pred = model.predict(X_test)
y_proba = model.predict_proba(X_test)    # classification probability

# Transformers add transform()
from sklearn.preprocessing import StandardScaler
scaler = StandardScaler()
scaler.fit(X_train)                       # learn mean/std
X_scaled = scaler.transform(X_test)       # apply to new data
X_scaled = scaler.fit_transform(X_train)  # fit + transform in one step

# Attributes available after fit (trailing underscore convention)
model.feature_importances_
model.classes_
model.n_features_in_
scaler.mean_
scaler.scale_
```

## Pipelines
### Pipeline and ColumnTransformer
```python
from sklearn.pipeline import Pipeline, make_pipeline
from sklearn.compose import ColumnTransformer
from sklearn.preprocessing import StandardScaler, OneHotEncoder
from sklearn.impute import SimpleImputer
from sklearn.ensemble import GradientBoostingClassifier

# Simple pipeline
pipe = Pipeline([
    ('scaler', StandardScaler()),
    ('classifier', GradientBoostingClassifier(n_estimators=200))
])
pipe.fit(X_train, y_train)
pipe.predict(X_test)                      # scaler + classifier in sequence

# make_pipeline (auto-names steps)
pipe = make_pipeline(StandardScaler(), GradientBoostingClassifier())

# ColumnTransformer for heterogeneous data
numeric_features = ['age', 'salary', 'experience']
categorical_features = ['department', 'role', 'location']

preprocessor = ColumnTransformer(
    transformers=[
        ('num', Pipeline([
            ('imputer', SimpleImputer(strategy='median')),
            ('scaler', StandardScaler())
        ]), numeric_features),
        ('cat', Pipeline([
            ('imputer', SimpleImputer(strategy='most_frequent')),
            ('encoder', OneHotEncoder(handle_unknown='ignore', sparse_output=False))
        ]), categorical_features)
    ],
    remainder='drop'                       # or 'passthrough'
)

# Full pipeline: preprocessing + model
full_pipeline = Pipeline([
    ('preprocessor', preprocessor),
    ('classifier', GradientBoostingClassifier(
        n_estimators=200, learning_rate=0.1, max_depth=5
    ))
])

full_pipeline.fit(X_train, y_train)
y_pred = full_pipeline.predict(X_test)

# Access nested parameters (for GridSearchCV)
# format: stepname__parametername
full_pipeline.set_params(classifier__n_estimators=300)
```

## Cross-Validation
### Validation Strategies
```python
from sklearn.model_selection import (
    cross_val_score, cross_validate,
    KFold, StratifiedKFold, TimeSeriesSplit,
    LeaveOneOut, RepeatedStratifiedKFold
)

# Basic cross-validation
scores = cross_val_score(model, X, y, cv=5, scoring='accuracy')
print(f"Accuracy: {scores.mean():.3f} (+/- {scores.std():.3f})")

# Multiple metrics
results = cross_validate(
    model, X, y, cv=5,
    scoring=['accuracy', 'f1_weighted', 'roc_auc_ovr'],
    return_train_score=True
)

# Stratified K-Fold (maintains class proportions)
skf = StratifiedKFold(n_splits=5, shuffle=True, random_state=42)
for train_idx, test_idx in skf.split(X, y):
    X_train, X_test = X[train_idx], X[test_idx]
    y_train, y_test = y[train_idx], y[test_idx]

# Time series (no future data leakage)
tscv = TimeSeriesSplit(n_splits=5, gap=10)
scores = cross_val_score(model, X, y, cv=tscv)

# Repeated stratified for more robust estimates
rskf = RepeatedStratifiedKFold(n_splits=5, n_repeats=10, random_state=42)
scores = cross_val_score(model, X, y, cv=rskf, scoring='f1')
```

## Metrics
### Classification Metrics
```python
from sklearn.metrics import (
    accuracy_score, precision_score, recall_score, f1_score,
    roc_auc_score, classification_report, confusion_matrix,
    precision_recall_curve, roc_curve, log_loss,
    average_precision_score
)

# Single metric
accuracy_score(y_true, y_pred)
precision_score(y_true, y_pred, average='weighted')
recall_score(y_true, y_pred, average='macro')
f1_score(y_true, y_pred, average='binary')

# ROC-AUC (requires probability scores)
roc_auc_score(y_true, y_proba[:, 1])               # binary
roc_auc_score(y_true, y_proba, multi_class='ovr')   # multiclass

# Comprehensive report
print(classification_report(y_true, y_pred, target_names=class_names))

# Confusion matrix
cm = confusion_matrix(y_true, y_pred)
# [[TN, FP],
#  [FN, TP]]

# ROC curve data
fpr, tpr, thresholds = roc_curve(y_true, y_proba[:, 1])

# Precision-recall curve
precision, recall, thresholds = precision_recall_curve(y_true, y_proba[:, 1])
ap = average_precision_score(y_true, y_proba[:, 1])
```

### Regression Metrics
```python
from sklearn.metrics import (
    mean_squared_error, mean_absolute_error, r2_score,
    mean_absolute_percentage_error, root_mean_squared_error
)

mse = mean_squared_error(y_true, y_pred)
rmse = root_mean_squared_error(y_true, y_pred)
mae = mean_absolute_error(y_true, y_pred)
r2 = r2_score(y_true, y_pred)
mape = mean_absolute_percentage_error(y_true, y_pred)
```

## Preprocessing
### Scalers and Encoders
```python
from sklearn.preprocessing import (
    StandardScaler,         # zero mean, unit variance
    MinMaxScaler,           # scale to [0, 1]
    RobustScaler,           # median/IQR (outlier robust)
    MaxAbsScaler,           # scale to [-1, 1] by max absolute value
    Normalizer,             # L1 or L2 normalize each sample
    OneHotEncoder,          # categorical -> binary columns
    OrdinalEncoder,         # categorical -> integer
    LabelEncoder,           # target labels -> integers
    PolynomialFeatures,     # generate interaction/polynomial terms
    FunctionTransformer,    # wrap arbitrary function as transformer
    KBinsDiscretizer,       # continuous -> binned
    PowerTransformer,       # Yeo-Johnson or Box-Cox transform
)

# Standard scaling
scaler = StandardScaler()
X_scaled = scaler.fit_transform(X_train)
X_test_scaled = scaler.transform(X_test)  # use train stats

# OneHotEncoder
ohe = OneHotEncoder(sparse_output=False, handle_unknown='ignore')
X_encoded = ohe.fit_transform(X_cat)
ohe.get_feature_names_out()               # column names

# Polynomial features
poly = PolynomialFeatures(degree=2, interaction_only=True)
X_poly = poly.fit_transform(X)            # adds x1*x2, x1*x3, etc.

# Custom transformer
from sklearn.preprocessing import FunctionTransformer
log_transformer = FunctionTransformer(np.log1p, inverse_func=np.expm1)
```

## Model Selection
### GridSearchCV and RandomizedSearchCV
```python
from sklearn.model_selection import GridSearchCV, RandomizedSearchCV
from scipy.stats import randint, uniform

# Grid search (exhaustive)
param_grid = {
    'classifier__n_estimators': [100, 200, 500],
    'classifier__max_depth': [5, 10, 20, None],
    'classifier__min_samples_split': [2, 5, 10],
    'classifier__learning_rate': [0.01, 0.1, 0.2]
}

grid_search = GridSearchCV(
    full_pipeline,
    param_grid,
    cv=5,
    scoring='f1_weighted',
    n_jobs=-1,                             # all cores
    verbose=1,
    refit=True                             # refit best on full training set
)
grid_search.fit(X_train, y_train)

print(grid_search.best_params_)
print(grid_search.best_score_)
best_model = grid_search.best_estimator_

# Randomized search (sample from distributions)
param_distributions = {
    'classifier__n_estimators': randint(50, 500),
    'classifier__max_depth': randint(3, 30),
    'classifier__learning_rate': uniform(0.001, 0.3),
    'classifier__min_samples_split': randint(2, 20),
    'classifier__subsample': uniform(0.5, 0.5)
}

random_search = RandomizedSearchCV(
    full_pipeline,
    param_distributions,
    n_iter=100,                            # number of random combinations
    cv=5,
    scoring='f1_weighted',
    n_jobs=-1,
    random_state=42
)
random_search.fit(X_train, y_train)

# Results as DataFrame
import pandas as pd
results_df = pd.DataFrame(random_search.cv_results_)
results_df.sort_values('rank_test_score').head(10)
```

## Common Algorithms Overview
### Quick Reference
```python
# Classification
from sklearn.linear_model import LogisticRegression
from sklearn.svm import SVC
from sklearn.neighbors import KNeighborsClassifier
from sklearn.tree import DecisionTreeClassifier
from sklearn.ensemble import (
    RandomForestClassifier, GradientBoostingClassifier,
    AdaBoostClassifier, BaggingClassifier
)
from sklearn.naive_bayes import GaussianNB

# Regression
from sklearn.linear_model import (
    LinearRegression, Ridge, Lasso, ElasticNet,
    SGDRegressor
)
from sklearn.svm import SVR
from sklearn.ensemble import (
    RandomForestRegressor, GradientBoostingRegressor
)

# Clustering
from sklearn.cluster import (
    KMeans, DBSCAN, AgglomerativeClustering,
    SpectralClustering, MeanShift
)

# Dimensionality reduction
from sklearn.decomposition import PCA, TruncatedSVD, NMF
from sklearn.manifold import TSNE
from sklearn.discriminant_analysis import LinearDiscriminantAnalysis

# Feature selection
from sklearn.feature_selection import (
    SelectKBest, f_classif, mutual_info_classif,
    RFE, SelectFromModel
)
```

## Tips
- Always split data before any preprocessing; fitting scalers or encoders on test data causes data leakage
- Use `Pipeline` for every workflow, even simple ones, to prevent leakage during cross-validation and ensure clean production deployment
- Prefer `RandomizedSearchCV` over `GridSearchCV` when the hyperparameter space has more than 3-4 dimensions; it finds good solutions faster
- Set `random_state` on every estimator that uses randomness for reproducible results across runs
- Use `StratifiedKFold` for classification and `TimeSeriesSplit` for temporal data; default `KFold` can produce misleading results
- Check `learning_curve` to diagnose bias (underfitting) vs variance (overfitting) before tuning hyperparameters
- Use `class_weight='balanced'` in classifiers for imbalanced datasets rather than resampling as a first approach
- Serialize trained pipelines with `joblib.dump()` instead of `pickle` for efficient handling of large NumPy arrays
- Inspect `feature_importances_` or use `permutation_importance` from `sklearn.inspection` for model interpretability
- Use `make_scorer()` to create custom scoring functions when built-in metrics do not match your business objective
- Profile with `sklearn.utils.estimator_html_repr` to visualize complex pipeline structures

## See Also
- pandas, numpy, jupyter, xgboost, lightgbm, tensorflow

## References
- [scikit-learn Official Documentation](https://scikit-learn.org/stable/)
- [scikit-learn User Guide](https://scikit-learn.org/stable/user_guide.html)
- [scikit-learn API Reference](https://scikit-learn.org/stable/modules/classes.html)
- [Hands-On Machine Learning (Aurelien Geron)](https://www.oreilly.com/library/view/hands-on-machine-learning/9781098125974/)
