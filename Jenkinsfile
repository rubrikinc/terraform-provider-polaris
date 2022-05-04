// MIT License
// 
// Copyright (c) 2021 Rubrik
// 
//  Permission is hereby granted, free of charge, to any person obtaining a copy
//  of this software and associated documentation files (the "Software"), to deal
//  in the Software without restriction, including without limitation the rights
//  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
//  copies of the Software, and to permit persons to whom the Software is
//  furnished to do so, subject to the following conditions:
// 
//  The above copyright notice and this permission notice shall be included in all
//  copies or substantial portions of the Software.
// 
//  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
//  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
//  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
//  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
//  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
//  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//  SOFTWARE.

pipeline {
    agent any
    tools {
        go 'go-1.18'
    }
    triggers {
        cron(env.BRANCH_NAME == 'main' ? 'H 01 * * *' : '')
    }
    parameters {
        booleanParam(name: 'RUN_ACCEPTANCE_TEST', defaultValue: false)
    }
    environment {
        // Polaris
        RUBRIK_POLARIS_SERVICEACCOUNT_FILE = credentials('tf-sdk-test-polaris-service-account')

        // AWS
        TEST_AWSACCOUNT_FILE        = credentials('tf-sdk-test-aws-account')
        AWS_SHARED_CREDENTIALS_FILE = credentials('tf-sdk-test-aws-credentials')
        AWS_CONFIG_FILE             = credentials('tf-sdk-test-aws-config')

        // Azure
        TEST_AZURESUBSCRIPTION_FILE     = credentials('tf-sdk-test-azure-subscription')
        AZURE_SERVICEPRINCIPAL_LOCATION = credentials('tf-sdk-test-azure-service-principal')

        // GCP
        TEST_GCPPROJECT_FILE           = credentials('tf-sdk-test-gcp-project')
        GOOGLE_APPLICATION_CREDENTIALS = credentials('tf-sdk-test-gcp-service-account')

        // Run acceptance tests with the nightly build or when triggered manually.
        TF_ACC = "${currentBuild.getBuildCauses('hudson.triggers.TimerTrigger$TimerTriggerCause').size() > 0 ? 'true' : params.RUN_ACCEPTANCE_TEST}"

        // Enable logging from the terraform cli binary used by acceptance tests
        TF_ACC_LOG_PATH='terraform_cli.log'
    }
    stages {
        stage('Lint') {
            steps {
                sh 'go mod tidy'
                sh 'go vet ./...'
                sh 'go run honnef.co/go/tools/cmd/staticcheck@latest ./...'
                sh 'bash -c "diff -u <(echo -n) <(gofmt -d .)"'
            }
        }
        stage('Build') {
            steps {
                sh 'curl -sL https://git.io/goreleaser | bash -s -- --snapshot --skip-publish --skip-sign --rm-dist'
            }
        }
        stage('Pre-test') {
            when { expression { env.TF_ACC == "true" } }
            steps {
                sh 'go run github.com/rubrikinc/rubrik-polaris-sdk-for-go/cmd/testenv@v0.4.7 -precheck'
            }
        }
        stage('Test') {
            steps {
                sh 'if [ "$TF_ACC" != "true" ]; then unset TF_ACC; fi; CGO_ENABLED=0 go test -count=1 -timeout=120m -v ./...'
            }
        }
    }
    post {
        always {
            archiveArtifacts artifacts: '**/terraform_cli.log', allowEmptyArchive: true
            script {
                if (env.TF_ACC == "true") {
                    sh 'go run github.com/rubrikinc/rubrik-polaris-sdk-for-go/cmd/testenv@v0.4.7 -cleanup'
                }
            }
        }
        success {
            script {
                if (currentBuild.getBuildCauses('hudson.triggers.TimerTrigger$TimerTriggerCause').size() > 0) {
                    slackSend(
                        channel: '#terraform-provider-development',
                        color: 'good',
                        message: "The pipeline ${currentBuild.fullDisplayName} succeeded (runtime: ${currentBuild.durationString.minus(' and counting')})\n${currentBuild.absoluteUrl}"
                    )
                }
            }
        }
        failure {
            script {
                if (currentBuild.getBuildCauses('hudson.triggers.TimerTrigger$TimerTriggerCause').size() > 0) {
                    slackSend(
                        channel: '#terraform-provider-development',
                        color: 'danger',
                        message: "The pipeline ${currentBuild.fullDisplayName} failed (runtime: ${currentBuild.durationString.minus(' and counting')})\n${currentBuild.absoluteUrl}"
                    )
                }
            }
        }
    }
}
