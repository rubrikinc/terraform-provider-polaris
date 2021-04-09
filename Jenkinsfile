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
    stages {
        stage('Lint') {
            environment {
                // Install tokens required to access private repositories using
                // go get.
                PROVIDER_NETRC = credentials('provider-netrc-file')
            }
            steps {
                sh 'cp -f ${PROVIDER_NETRC} ~/.netrc'
                sh 'go vet ./...'
            }
        }
        stage('Build') {
            environment {
                // Extract version information from tags named as vX.Y.Z. Other
                // tags and branches are defaulted to v0.0.1.
                PROVIDER_VERSION = eval(env.TAG_NAME ==~ /^v[0-9]+.[0-9]+.[0-9]+$/ ? env.TAG_NAME.substring(1) : '0.0.1')
            }
            steps {
                sh 'make clean all'
            }
        }
        stage('Test') {
            steps {
                sh 'make test'
                sh 'rm ~/.netrc'
            }
        }
    }
    post {
        always {
            archiveArtifacts artifacts: 'build/terraform-provider-polaris*', onlyIfSuccessful: true
        }
    }
}

// Trick to allow groovy script to be evaluated when assigning a value to an
// environment variable.
def eval(expr) {
    return expr
}
