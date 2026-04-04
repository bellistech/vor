# Jenkins (Pipeline-as-Code CI/CD Server)

Automation server for building, testing, and deploying software with declarative or scripted pipelines.

## Declarative Pipeline

### Minimal Jenkinsfile

```groovy
// Jenkinsfile (Declarative)
pipeline {
    agent any

    stages {
        stage('Build') {
            steps {
                sh 'make build'
            }
        }
        stage('Test') {
            steps {
                sh 'make test'
            }
        }
        stage('Deploy') {
            steps {
                sh './deploy.sh'
            }
        }
    }
}
```

### Agent types

```groovy
pipeline {
    // Run on any available agent
    agent any

    // Run on a labeled node
    // agent { label 'linux && docker' }

    // Run inside a Docker container
    // agent {
    //     docker {
    //         image 'node:20-alpine'
    //         args '-v /tmp:/tmp'
    //     }
    // }

    // No top-level agent (each stage picks its own)
    // agent none

    stages {
        stage('Build') {
            // Per-stage agent override
            agent {
                docker { image 'golang:1.24' }
            }
            steps {
                sh 'go build ./...'
            }
        }
    }
}
```

## Stages and Steps

### Common step types

```groovy
pipeline {
    agent any
    stages {
        stage('Build') {
            steps {
                // Shell command
                sh 'echo "building"'
                sh '''
                    echo "multi-line"
                    make build
                '''

                // Change directory
                dir('backend') {
                    sh 'go build ./...'
                }

                // Archive artifacts
                archiveArtifacts artifacts: 'build/**/*', fingerprint: true

                // Stash/unstash files between stages
                stash includes: 'build/**', name: 'build-output'
            }
        }
        stage('Test') {
            steps {
                unstash 'build-output'
                sh 'make test'

                // Publish test results
                junit 'reports/**/*.xml'
            }
        }
    }
}
```

## Post Actions

### Run steps after stage/pipeline completion

```groovy
pipeline {
    agent any
    stages {
        stage('Build') {
            steps {
                sh 'make build'
            }
        }
    }
    post {
        always {
            // Runs regardless of outcome
            echo 'Cleaning up workspace'
            cleanWs()
        }
        success {
            echo 'Build succeeded'
        }
        failure {
            // Send notification on failure
            mail to: 'team@example.com',
                 subject: "FAILED: ${env.JOB_NAME} #${env.BUILD_NUMBER}",
                 body: "Check: ${env.BUILD_URL}"
        }
        unstable {
            echo 'Build is unstable (test failures)'
        }
        changed {
            echo 'Build status changed from last run'
        }
    }
}
```

## Conditional Execution

### When directive

```groovy
pipeline {
    agent any
    stages {
        stage('Deploy Staging') {
            when {
                branch 'develop'
            }
            steps {
                sh './deploy.sh staging'
            }
        }
        stage('Deploy Production') {
            when {
                branch 'main'
                // Additional conditions
                // environment name: 'DEPLOY', value: 'true'
                // expression { return params.DEPLOY_PROD }
                // tag pattern: 'v*', comparator: 'GLOB'
                // changeset '**/*.js'
            }
            steps {
                sh './deploy.sh production'
            }
        }
        stage('PR Build') {
            when {
                changeRequest()    // only on pull requests
            }
            steps {
                sh 'make lint'
            }
        }
    }
}
```

## Parameters

### Build with user inputs

```groovy
pipeline {
    agent any
    parameters {
        string(name: 'DEPLOY_ENV', defaultValue: 'staging', description: 'Target env')
        booleanParam(name: 'RUN_TESTS', defaultValue: true, description: 'Run tests?')
        choice(name: 'REGION', choices: ['us-east-1', 'eu-west-1'], description: 'AWS region')
        password(name: 'API_KEY', description: 'API key for deploy')
    }
    stages {
        stage('Deploy') {
            when {
                expression { params.RUN_TESTS == true }
            }
            steps {
                echo "Deploying to ${params.DEPLOY_ENV} in ${params.REGION}"
            }
        }
    }
}
```

## Credentials

### Access secrets securely

```groovy
pipeline {
    agent any
    stages {
        stage('Deploy') {
            steps {
                // Username/password credential
                withCredentials([usernamePassword(
                    credentialsId: 'docker-hub',
                    usernameVariable: 'DOCKER_USER',
                    passwordVariable: 'DOCKER_PASS'
                )]) {
                    sh 'docker login -u $DOCKER_USER -p $DOCKER_PASS'
                }

                // Secret text
                withCredentials([string(
                    credentialsId: 'api-token',
                    variable: 'TOKEN'
                )]) {
                    sh 'curl -H "Authorization: Bearer $TOKEN" https://api.example.com'
                }

                // SSH key / secret file
                withCredentials([file(
                    credentialsId: 'kubeconfig',
                    variable: 'KUBECONFIG'
                )]) {
                    sh 'kubectl get pods'
                }
            }
        }
    }
}
```

## Environment Variables

### Define and use env vars

```groovy
pipeline {
    agent any
    environment {
        APP_NAME = 'my-app'
        VERSION  = sh(script: 'git describe --tags', returnStdout: true).trim()
        // Bind credentials to env vars
        DOCKER_CREDS = credentials('docker-hub')  // sets _USR and _PSW
    }
    stages {
        stage('Info') {
            environment {
                STAGE_VAR = 'stage-only'
            }
            steps {
                echo "App: ${env.APP_NAME} v${env.VERSION}"
                echo "Job: ${env.JOB_NAME} Build: ${env.BUILD_NUMBER}"
                echo "URL: ${env.BUILD_URL}"
            }
        }
    }
}
```

## Parallel Stages

### Run stages concurrently

```groovy
pipeline {
    agent any
    stages {
        stage('Tests') {
            parallel {
                stage('Unit Tests') {
                    agent { docker { image 'node:20' } }
                    steps {
                        sh 'npm run test:unit'
                    }
                }
                stage('Integration Tests') {
                    agent { docker { image 'node:20' } }
                    steps {
                        sh 'npm run test:integration'
                    }
                }
                stage('Lint') {
                    agent { docker { image 'node:20' } }
                    steps {
                        sh 'npm run lint'
                    }
                }
            }
        }
    }
}
```

## Shared Libraries

### Load and use shared libraries

```groovy
// In Jenkins: Manage Jenkins > System > Global Pipeline Libraries
// Configure: name=my-lib, default version=main, git URL

// Jenkinsfile
@Library('my-lib') _

pipeline {
    agent any
    stages {
        stage('Build') {
            steps {
                // Call a function from vars/deployApp.groovy
                deployApp(env: 'staging', version: '1.2.3')
            }
        }
    }
}

// vars/deployApp.groovy (in the shared library repo)
// def call(Map config) {
//     sh "./deploy.sh ${config.env} ${config.version}"
// }
```

## Scripted Pipeline

### Groovy-based pipeline (more flexible)

```groovy
// Jenkinsfile (Scripted)
node('linux') {
    stage('Checkout') {
        checkout scm
    }

    stage('Build') {
        sh 'make build'
    }

    stage('Test') {
        try {
            sh 'make test'
        } catch (err) {
            currentBuild.result = 'UNSTABLE'
            echo "Tests failed: ${err}"
        } finally {
            junit 'reports/**/*.xml'
        }
    }

    stage('Deploy') {
        if (env.BRANCH_NAME == 'main') {
            sh './deploy.sh production'
        }
    }
}
```

## Docker Agent

### Full Docker workflow

```groovy
pipeline {
    agent none
    stages {
        stage('Build Image') {
            agent any
            steps {
                script {
                    def app = docker.build("myapp:${env.BUILD_NUMBER}")
                    docker.withRegistry('https://registry.example.com', 'docker-creds') {
                        app.push()
                        app.push('latest')
                    }
                }
            }
        }
    }
}
```

## Common Plugins

```text
# Essential plugins and their use:
Pipeline               - core pipeline support
Git                    - git SCM integration
Docker Pipeline        - docker agents and builds
Credentials Binding    - inject secrets into builds
Blue Ocean             - modern UI for pipelines
Job DSL                - programmatic job creation
Timestamper            - add timestamps to console output
Workspace Cleanup      - cleanWs() step
JUnit                  - test result reporting
Slack Notification     - send build status to Slack
```

## Tips

- Use `cleanWs()` in `post { always {} }` to avoid stale workspace issues.
- Use `timeout(time: 30, unit: 'MINUTES') { ... }` to prevent runaway builds.
- Use `retry(3) { sh 'flaky-command' }` for transient failures.
- Use `input` step for manual approval gates before production deploys.
- Use `catchError(buildResult: 'SUCCESS', stageResult: 'FAILURE')` to continue pipeline on stage failure.
- Use `options { disableConcurrentBuilds() }` to prevent parallel runs of the same job.
- Prefer Declarative pipelines for readability; use Scripted for complex logic.

## See Also

- github-actions
- gitlab-ci
- docker
- kubernetes
- ansible
- git

## References

- [Jenkins Pipeline Documentation](https://www.jenkins.io/doc/book/pipeline/)
- [Pipeline Syntax Reference](https://www.jenkins.io/doc/book/pipeline/syntax/)
- [Pipeline Steps Reference](https://www.jenkins.io/doc/pipeline/steps/)
- [Shared Libraries](https://www.jenkins.io/doc/book/pipeline/shared-libraries/)
- [Jenkins User Handbook](https://www.jenkins.io/doc/book/)
- [Jenkinsfile Best Practices](https://www.jenkins.io/doc/book/pipeline/pipeline-best-practices/)
- [Credentials Management](https://www.jenkins.io/doc/book/using/using-credentials/)
- [Blue Ocean Pipeline Editor](https://www.jenkins.io/doc/book/blueocean/)
- [Jenkins Plugin Index](https://plugins.jenkins.io/)
- [Managing Jenkins Agents](https://www.jenkins.io/doc/book/managing/nodes/)
- [Jenkins Security Guide](https://www.jenkins.io/doc/book/security/)
- [Jenkins GitHub Repository](https://github.com/jenkinsci/jenkins)
