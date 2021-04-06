pipeline {
    agent any
    tools {
        go 'go-1.16.2'
    }
    stages {
        stage('lint') {
            steps {
                sh 'go vet ./...'
            }
        }
        stage('build') {
            environment {
                CGO_ENABLED = '0'
            }
            steps {
                sh 'go build ./...'
            }
        }
        stage('test') {
            environment {
                CGO_ENABLED = '0'
            }
            steps {
                sh 'go test -cover ./...'
            }
        }
    }
}
