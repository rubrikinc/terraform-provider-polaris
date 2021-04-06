pipeline {
    agent any
    tools {
        go 'go-1.16.2'
    }
    stages {
        stage('lint') {
            environment {
                GO_GET_TOKEN = credentials('go-get-token')
            }
            steps {
                sh 'echo "machine github.com login ${GO_GET_TOKEN}" > ~/.netrc'
                sh 'go vet ./...'
            }
        }
        stage('build') {
            steps {
                sh 'make all'
                zip zipFile: 'terraform-provider-polaris.zip', dir: 'build', overwrite: true
            }
        }
        stage('test') {
            steps {
                sh 'make test'
            }
        }
    }
    post {
        always {
            archiveArtifacts artifacts: 'terraform-provider-polaris.zip', onlyIfSuccessful: true
        }
    }
}
