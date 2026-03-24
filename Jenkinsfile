pipeline {
    agent any

    environment {
        SONAR_TOKEN = credentials('SonarQubeTokens')
    }

    stages {

        stage('Go Build & Test') {
            agent {
                docker {
                    image 'golang:1.25'
                }
            }
            steps {
                sh '''
                go mod tidy
                go test ./... -coverprofile=coverage.out
                '''
            }
        }

        stage('Sonar Scan') {
            agent {
                docker {
                    image 'golang:1.25'
                }
            }
            steps {
                withSonarQubeEnv('sonarcloud') {
                    sh '''
                    curl -sSLo sonar-scanner.zip https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/sonar-scanner-cli-5.0.1.3006-linux.zip
                    unzip -o sonar-scanner.zip

                    ./sonar-scanner-*/bin/sonar-scanner \
                      -Dsonar.projectKey=sundayyogurt_flyup \
                      -Dsonar.organization=sundayyogurt \
                      -Dsonar.sources=. \
                      -Dsonar.exclusions=**/*_test.go \
                      -Dsonar.go.coverage.reportPaths=coverage.out \
                      -Dsonar.login=$SONAR_TOKEN
                    '''
                }
            }
        }

        stage('Build & Deploy') {
            when {
                branch 'develop'
            }
            steps {
                sh '''
                docker compose down
                docker compose up -d --build
                '''
            }
        }
    }
}