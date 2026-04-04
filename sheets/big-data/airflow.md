# Apache Airflow (Workflow Orchestration)

Apache Airflow is an open-source platform for programmatically authoring, scheduling, and monitoring workflows as directed acyclic graphs (DAGs), widely used for ETL pipelines, ML workflows, and data engineering automation.

## Core Concepts
### DAGs (Directed Acyclic Graphs)
```python
# Define a DAG
from airflow import DAG
from datetime import datetime, timedelta

default_args = {
    'owner': 'data-team',
    'depends_on_past': False,
    'email_on_failure': True,
    'email': ['alerts@example.com'],
    'retries': 3,
    'retry_delay': timedelta(minutes=5),
}

dag = DAG(
    'my_etl_pipeline',
    default_args=default_args,
    description='Daily ETL pipeline',
    schedule_interval='@daily',
    start_date=datetime(2024, 1, 1),
    catchup=False,
    max_active_runs=1,
    tags=['etl', 'production'],
)
```

### TaskFlow API (Airflow 2.x)
```python
from airflow.decorators import dag, task
from datetime import datetime

@dag(schedule='@daily', start_date=datetime(2024, 1, 1), catchup=False)
def my_pipeline():
    @task
    def extract():
        return {'data': [1, 2, 3]}

    @task
    def transform(raw):
        return {'transformed': [x * 2 for x in raw['data']]}

    @task
    def load(data):
        print(f"Loading {data}")

    raw = extract()
    transformed = transform(raw)
    load(transformed)

my_pipeline()
```

## Operators
### BashOperator
```python
from airflow.operators.bash import BashOperator

run_script = BashOperator(
    task_id='run_etl_script',
    bash_command='python /opt/scripts/etl.py --date {{ ds }}',
    env={'API_KEY': '{{ var.value.api_key }}'},
    dag=dag,
)
```

### PythonOperator
```python
from airflow.operators.python import PythonOperator

def process_data(**context):
    execution_date = context['ds']
    ti = context['ti']
    upstream_data = ti.xcom_pull(task_ids='extract_task')
    return {'processed': True, 'rows': 1500}

process = PythonOperator(
    task_id='process_data',
    python_callable=process_data,
    provide_context=True,
    dag=dag,
)
```

### KubernetesPodOperator
```python
from airflow.providers.cncf.kubernetes.operators.kubernetes_pod import KubernetesPodOperator

k8s_task = KubernetesPodOperator(
    task_id='spark_job',
    name='spark-etl',
    namespace='airflow',
    image='spark:3.5',
    cmds=['spark-submit'],
    arguments=['--master', 'k8s://https://k8s:6443', '/app/job.py'],
    resources={'request_memory': '2Gi', 'request_cpu': '1'},
    is_delete_operator_pod=True,
    dag=dag,
)
```

## XComs (Cross-Communication)
```python
# Push XCom explicitly
def push_data(**context):
    context['ti'].xcom_push(key='row_count', value=42)

# Pull XCom in Jinja template
templated_task = BashOperator(
    task_id='use_xcom',
    bash_command='echo "Rows: {{ ti.xcom_pull(task_ids=\'push_task\', key=\'row_count\') }}"',
)

# TaskFlow handles XCom automatically via return values
```

## Connections & Variables
```bash
# Set connections via CLI
airflow connections add 'postgres_prod' \
    --conn-type 'postgres' \
    --conn-host 'db.example.com' \
    --conn-port 5432 \
    --conn-login 'airflow' \
    --conn-password 'secret' \
    --conn-schema 'warehouse'

# Set variables
airflow variables set 'env' 'production'
airflow variables get 'env'

# Import/export variables from JSON
airflow variables export variables.json
airflow variables import variables.json
```

## Pools & Priority
```python
# Limit concurrency with pools
task = PythonOperator(
    task_id='db_query',
    python_callable=run_query,
    pool='postgres_pool',        # max N concurrent tasks
    pool_slots=1,                # slots this task consumes
    priority_weight=10,          # higher = runs first
    weight_rule='downstream',    # downstream | upstream | absolute
    dag=dag,
)
```

```bash
# Create a pool via CLI
airflow pools set postgres_pool 5 "Postgres connection pool"
airflow pools list
```

## Scheduling & Executors
```python
# Schedule presets
# @once, @hourly, @daily, @weekly, @monthly, @yearly
# Cron: '0 6 * * 1-5'  (weekdays at 6am)
# Timetable (Airflow 2.x):
from airflow.timetables.interval import CronDataIntervalTimetable
```

```bash
# Executor configuration (airflow.cfg)
# LocalExecutor   — single machine, multiprocess
# CeleryExecutor  — distributed across workers
# KubernetesExecutor — one pod per task
# CeleryKubernetesExecutor — hybrid

# Start Celery worker
airflow celery worker --concurrency 16 --queues default,high_priority

# Start scheduler
airflow scheduler

# Start webserver
airflow webserver --port 8080
```

## Sensors
```python
from airflow.sensors.filesystem import FileSensor
from airflow.providers.http.sensors.http import HttpSensor
from airflow.sensors.external_task import ExternalTaskSensor

wait_for_file = FileSensor(
    task_id='wait_for_file',
    filepath='/data/incoming/{{ ds }}.csv',
    poke_interval=60,
    timeout=3600,
    mode='reschedule',   # reschedule | poke
    dag=dag,
)

wait_for_api = HttpSensor(
    task_id='wait_for_api',
    http_conn_id='api_conn',
    endpoint='/health',
    response_check=lambda resp: resp.json()['status'] == 'ready',
    poke_interval=30,
    timeout=600,
    dag=dag,
)

wait_for_upstream = ExternalTaskSensor(
    task_id='wait_for_upstream',
    external_dag_id='upstream_dag',
    external_task_id='final_task',
    execution_delta=timedelta(hours=1),
    dag=dag,
)
```

## CLI Commands
```bash
# DAG management
airflow dags list
airflow dags trigger my_dag --conf '{"key": "value"}'
airflow dags pause my_dag
airflow dags unpause my_dag
airflow dags show my_dag          # render DAG structure

# Task operations
airflow tasks list my_dag
airflow tasks test my_dag my_task 2024-01-01
airflow tasks run my_dag my_task 2024-01-01
airflow tasks clear my_dag -s 2024-01-01 -e 2024-01-31
airflow tasks failed-deps my_dag my_task 2024-01-01

# Backfill
airflow dags backfill my_dag \
    --start-date 2024-01-01 \
    --end-date 2024-01-31 \
    --reset-dagruns

# Database
airflow db init
airflow db upgrade
airflow db check

# Users
airflow users create --username admin --role Admin \
    --firstname Admin --lastname User --email admin@example.com

# Info & debugging
airflow info
airflow config list
airflow providers list
```

## Task Dependencies
```python
# Bitshift operators
extract >> transform >> load

# Fan-out / fan-in
extract >> [validate, enrich] >> load

# Cross-DAG dependencies with TriggerDagRunOperator
from airflow.operators.trigger_dagrun import TriggerDagRunOperator

trigger_downstream = TriggerDagRunOperator(
    task_id='trigger_reporting',
    trigger_dag_id='reporting_dag',
    conf={'source_date': '{{ ds }}'},
    wait_for_completion=True,
    dag=dag,
)
```

## Tips
- Use `catchup=False` on new DAGs to prevent massive backfill on first deploy
- Prefer `mode='reschedule'` on sensors to free up worker slots during waits
- Keep DAG file parsing fast -- avoid heavy imports or DB calls at module level
- Use `max_active_runs=1` for DAGs that must not overlap executions
- Store secrets in connections or a secrets backend (Vault, AWS SSM), never in DAG code
- Set `depends_on_past=True` only when task order across runs truly matters
- Use TaskFlow API for cleaner XCom passing instead of manual push/pull
- Tag DAGs consistently (`tags=['team', 'domain']`) for filtering in the UI
- Test tasks locally with `airflow tasks test` before deploying
- Use pools to throttle concurrent access to shared resources like databases
- Configure SLA misses and email alerts for critical pipelines
- Pin provider package versions to avoid unexpected breakage on upgrade

## See Also
- celery, kubernetes, docker, redis, terraform

## References
- [Airflow Documentation](https://airflow.apache.org/docs/apache-airflow/stable/)
- [Airflow Best Practices](https://airflow.apache.org/docs/apache-airflow/stable/best-practices.html)
- [Airflow CLI Reference](https://airflow.apache.org/docs/apache-airflow/stable/cli-and-env-variables-ref.html)
- [Astronomer Guides](https://www.astronomer.io/guides/)
- [Airflow GitHub Repository](https://github.com/apache/airflow)
