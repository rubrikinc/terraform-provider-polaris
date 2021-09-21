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
        go 'go-1.16.2'
    }
    triggers {
        cron(env.BRANCH_NAME == 'main' ? '@midnight' : '')
    }
    stages {
        stage('Lint') {
            steps {
                sh 'go mod tidy'
                sh 'go vet ./...'
            }
        }
        stage('Build') {
            steps {
                sh 'curl -sL https://git.io/goreleaser | bash -s -- --snapshot --skip-publish --skip-sign --rm-dist'
            }
        }
        stage('Test') {
            environment {
                // Polaris
                RUBRIK_POLARIS_ACCOUNT_FILE        = credentials('tf-polaris-account')
                RUBRIK_POLARIS_SERVICEACCOUNT_FILE = 'default'

                // AWS
                TEST_AWSACCOUNT_FILE = credentials('tf-sdk-test-aws-account')
                AWS_CREDENTIALS      = credentials('tf-sdk-test-aws-credentials')
                AWS_CONFIG           = credentials('tf-sdk-test-aws-config')

                // Azure
                TEST_AZURESUBSCRIPTION_FILE     = credentials('tf-sdk-test-azure-subscription')
                AZURE_SERVICEPRINCIPAL_LOCATION = credentials('tf-sdk-test-azure-service-principal')

                // GCP
                TEST_GCPPROJECT_FILE           = credentials('tf-sdk-test-gcp-project')
                GOOGLE_APPLICATION_CREDENTIALS = credentials('tf-sdk-test-gcp-service-account')

                // Run acceptance tests with the nightly build.
                TF_ACC = currentBuild.getBuildCauses('hudson.triggers.TimerTrigger$TimerTriggerCause').size()
            }
            steps {
                sh 'mkdir -p ~/.aws && ln -sf $AWS_CREDENTIALS ~/.aws/credentials && ln -sf $AWS_CONFIG ~/.aws/config'
                sh 'mkdir -p ~/.rubrik && ln -sf $RUBRIK_POLARIS_ACCOUNT_FILE ~/.rubrik/polaris-accounts.json'
                sh 'if [ "$TF_ACC" != "1" ]; then unset TF_ACC; fi; CGO_ENABLED=0 go test -count=1 -timeout=120m -v ./...'
                sh 'rm -r ~/.aws ~/.rubrik'
            }
        }
    }
    post {
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
