# Jupyter (Interactive Computing Platform)

Jupyter is an open-source interactive computing platform that provides the Notebook document format for combining live code, equations, visualizations, and narrative text, supporting over 40 programming languages through its kernel architecture, widely used for data exploration, reproducible research, and technical communication.

## Notebook Format and Basics
### .ipynb Structure
```json
{
  "nbformat": 4,
  "nbformat_minor": 5,
  "metadata": {
    "kernelspec": {
      "display_name": "Python 3",
      "language": "python",
      "name": "python3"
    },
    "language_info": {
      "name": "python",
      "version": "3.11.0"
    }
  },
  "cells": [
    {
      "cell_type": "markdown",
      "metadata": {},
      "source": ["# Title\n", "Description text"]
    },
    {
      "cell_type": "code",
      "execution_count": 1,
      "metadata": {},
      "outputs": [],
      "source": ["import pandas as pd"]
    }
  ]
}
```

### Installation and Setup
```bash
# Install JupyterLab (recommended)
pip install jupyterlab

# Install classic Notebook
pip install notebook

# Launch
jupyter lab                         # JupyterLab (modern UI)
jupyter notebook                    # Classic notebook interface
jupyter lab --port 8889 --no-browser  # custom port, headless

# Install additional kernels
pip install ipykernel
python -m ipykernel install --user --name myenv --display-name "My Env"

# R kernel
# install.packages('IRkernel'); IRkernel::installspec()

# Julia kernel
# using IJulia; IJulia.installkernel("Julia")

# List installed kernels
jupyter kernelspec list
jupyter kernelspec remove old_kernel
```

## Kernel Management
### Kernel Lifecycle
```bash
# Kernel operations from command line
jupyter kernel --kernel=python3     # start standalone kernel

# In notebook: Kernel menu
# - Interrupt: stop running cell (like Ctrl+C)
# - Restart: fresh Python process (clears all variables)
# - Restart & Run All: clean execution from top
# - Restart & Clear Output: reset everything
# - Change kernel: switch language/environment
```

```python
# Programmatic kernel info
import IPython
print(IPython.sys_info())

# Check current kernel
import sys
print(sys.executable)
print(sys.version)

# Memory management
%reset -f                           # clear all variables
%who                                # list variables
%whos                               # list with details

# Auto-reload modules during development
%load_ext autoreload
%autoreload 2                       # reload all modules before execution
```

## Cell Magics
### Line Magics (%) and Cell Magics (%%)
```python
# Timing
%time result = df.groupby('col').sum()        # wall time (single run)
%timeit np.dot(a, b)                          # average over multiple runs
%%timeit                                       # time entire cell
result = []
for i in range(1000):
    result.append(i ** 2)

# Profiling
%prun my_function()                            # function profiling
%%prun                                         # profile entire cell
df.merge(other_df, on='key')

%lprun -f my_function my_function(args)        # line-by-line (needs line_profiler)
%memit my_function()                           # memory usage (needs memory_profiler)
%mprun -f my_function my_function()            # line-by-line memory

# Shell commands
!ls -la                                         # run shell command
!pip install package_name
files = !ls *.csv                               # capture output as list
%cd /path/to/directory
%pwd
%env MY_VAR=value                               # set environment variable

# Display
%matplotlib inline                              # static plots in notebook
%matplotlib widget                              # interactive plots (ipympl)
%config InlineBackend.figure_format = 'retina'  # high-DPI plots

# SQL (requires ipython-sql)
%load_ext sql
%sql sqlite:///my_database.db
%%sql
SELECT name, COUNT(*) as cnt
FROM users
GROUP BY name
ORDER BY cnt DESC
LIMIT 10;

# Writing files
%%writefile script.py
import sys
print(f"Hello from {sys.version}")

# Running scripts
%run script.py
%run -t script.py                    # with timing
%run -d script.py                    # with debugger

# LaTeX rendering
%%latex
\begin{align}
\nabla \times \vec{E} &= -\frac{\partial \vec{B}}{\partial t} \\
\nabla \times \vec{B} &= \mu_0 \vec{J} + \mu_0 \epsilon_0 \frac{\partial \vec{E}}{\partial t}
\end{align}

# HTML rendering
%%html
<div style="background: #f0f0f0; padding: 10px; border-radius: 5px;">
  <h3>Custom HTML Output</h3>
  <p>Rendered directly in the notebook</p>
</div>
```

## Widgets (ipywidgets)
### Interactive Controls
```python
import ipywidgets as widgets
from IPython.display import display

# Basic widgets
slider = widgets.IntSlider(value=50, min=0, max=100, step=1,
                           description='Threshold:')
display(slider)

dropdown = widgets.Dropdown(options=['linear', 'poly', 'rbf'],
                            value='rbf', description='Kernel:')

text = widgets.Text(value='', placeholder='Enter query',
                    description='Search:')

checkbox = widgets.Checkbox(value=True, description='Normalize')

# Interactive function binding
@widgets.interact(
    n=(10, 1000, 10),
    distribution=['normal', 'uniform', 'exponential'],
    show_stats=True
)
def plot_distribution(n=100, distribution='normal', show_stats=True):
    import matplotlib.pyplot as plt
    import numpy as np
    rng = np.random.default_rng(42)
    if distribution == 'normal':
        data = rng.standard_normal(n)
    elif distribution == 'uniform':
        data = rng.uniform(0, 1, n)
    else:
        data = rng.exponential(1, n)
    plt.hist(data, bins=30, edgecolor='black')
    if show_stats:
        plt.title(f'mean={data.mean():.3f}, std={data.std():.3f}')
    plt.show()

# Interactive output with manual update
output = widgets.Output()
button = widgets.Button(description='Run Analysis')

def on_click(b):
    with output:
        output.clear_output()
        print("Running analysis...")

button.on_click(on_click)
display(widgets.VBox([button, output]))

# Linked widgets
a = widgets.FloatSlider(description='a')
b = widgets.FloatSlider(description='b')
widgets.link((a, 'value'), (b, 'value'))   # bidirectional link
```

## JupyterHub Deployment
### Multi-User Server Configuration
```bash
# Install JupyterHub
pip install jupyterhub
npm install -g configurable-http-proxy
jupyterhub --generate-config

# Key config options (jupyterhub_config.py)
c.JupyterHub.ip = '0.0.0.0'
c.JupyterHub.port = 8000
c.JupyterHub.spawner_class = 'dockerspawner.DockerSpawner'
c.DockerSpawner.image = 'jupyter/datascience-notebook:latest'
c.DockerSpawner.mem_limit = '4G'
c.DockerSpawner.cpu_limit = 2.0
c.Authenticator.admin_users = {'admin1', 'admin2'}

# Kubernetes deployment (Zero to JupyterHub)
helm repo add jupyterhub https://hub.jupyter.org/helm-chart/
helm upgrade --install jhub jupyterhub/jupyterhub \
  --namespace jhub --create-namespace \
  --values config.yaml
```

## nbconvert (Export and Conversion)
### Converting Notebooks
```bash
# HTML export
jupyter nbconvert --to html notebook.ipynb
jupyter nbconvert --to html --no-input notebook.ipynb   # hide code cells

# PDF export (requires LaTeX)
jupyter nbconvert --to pdf notebook.ipynb
jupyter nbconvert --to pdf --template classic notebook.ipynb

# Slide presentation (Reveal.js)
jupyter nbconvert --to slides notebook.ipynb
jupyter nbconvert --to slides --post serve notebook.ipynb  # live serve

# Python script
jupyter nbconvert --to script notebook.ipynb

# Markdown
jupyter nbconvert --to markdown notebook.ipynb

# Execute notebook programmatically
jupyter nbconvert --to notebook --execute notebook.ipynb \
  --output executed_notebook.ipynb

# Batch conversion
for f in *.ipynb; do
  jupyter nbconvert --to html "$f"
done

# Parameterized execution with papermill
pip install papermill
papermill input.ipynb output.ipynb \
  -p dataset "sales_2024" \
  -p threshold 0.95
```

## Extensions and Configuration
### JupyterLab Extensions
```bash
# Install extensions
pip install jupyterlab-git               # Git integration
pip install jupyterlab-lsp               # Language Server Protocol
pip install jupyterlab_code_formatter    # Code formatting
jupyter labextension list                # list installed

# Configuration
jupyter lab --generate-config            # generates ~/.jupyter/jupyter_lab_config.py
```

### Notebook Configuration
```python
# Display options
%config InlineBackend.figure_format = 'retina'

# Pandas display options
import pandas as pd
pd.set_option('display.max_rows', 100)
pd.set_option('display.max_columns', 50)
pd.set_option('display.float_format', '{:.4f}'.format)

# Startup file (~/.ipython/profile_default/startup/00-imports.py)
# Runs automatically when kernel starts
```

## Tips
- Always use "Restart & Run All" before sharing a notebook to ensure cells execute in order and produce reproducible results
- Pin your environment with `%pip install` or `%conda install` inside notebooks instead of `!pip install` to ensure the correct kernel environment
- Use `%%capture` magic to suppress verbose output from library imports or long-running operations while still executing them
- Structure notebooks with markdown headers and a table of contents for navigability; JupyterLab renders them as a sidebar
- Set `%config InlineBackend.figure_format = 'retina'` for crisp plots on high-DPI displays
- Use papermill for parameterized notebook execution in production pipelines rather than running notebooks manually
- Keep notebooks under version control with `nbstripout` as a pre-commit hook to strip output and reduce diff noise
- Create a shared startup file in `~/.ipython/profile_default/startup/` for imports you use in every session
- Use `%store` magic to persist variables between notebook sessions without writing to files
- Tag cells as "parameters" for papermill or "skip" for nbconvert to control execution and export granularly

## See Also
- pandas, numpy, matplotlib, scikit-learn, datalakes

## References
- [Jupyter Official Documentation](https://docs.jupyter.org/en/latest/)
- [JupyterLab Documentation](https://jupyterlab.readthedocs.io/en/latest/)
- [JupyterHub Documentation](https://jupyterhub.readthedocs.io/en/latest/)
- [nbconvert Documentation](https://nbconvert.readthedocs.io/en/latest/)
- [ipywidgets Documentation](https://ipywidgets.readthedocs.io/en/latest/)
