pipeline {
    agent {
        docker {
            image 'golang:1.25.3'
            args '-v /var/run/docker.sock:/var/run/docker.sock'
        }
    }

    environment {
        SONAR_TOKEN = credentials('SonarQubeTokens')
    }

    stages {
        stage('Check Go') {
            steps {
                sh 'go version'
            }
        }

        stage('Install') {
            steps {
                sh 'go mod tidy'
            }
        }

        stage('Test & Coverage') {
            steps {
                sh 'go test ./... -coverprofile=coverage.out'
            }
        }

        stage('Sonar Scan') {
            steps {
                withSonarQubeEnv('sonarcloud') {
                    sh '''
                    apt-get update
                    apt-get install -y curl unzip

                    curl -sSLo sonar-scanner.zip https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/sonar-scanner-cli-5.0.1.3006-linux.zip
                    unzip sonar-scanner.zip
                    mv sonar-scanner-* sonar-scanner

                    ./sonar-scanner/bin/sonar-scanner \
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
                branch 'main'
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