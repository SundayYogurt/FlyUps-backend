pipeline {
    agent any

    environment {
        SONAR_TOKEN = credentials('SonarQubeTokens')
    }

    stages {

        // ===== Build & Test =====
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

        // ===== Sonar =====
        stage('Sonar Scan') {
            agent {
                docker {
                    image 'sonarsource/sonar-scanner-cli:latest'
                }
            }
            steps {
                withSonarQubeEnv('sonarcloud') {
                    sh '''
                    sonar-scanner \
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

        // ===== Debug =====
        stage('Debug Branch') {
            steps {
                sh '''
                echo "BRANCH_NAME=$BRANCH_NAME"
                echo "GIT_BRANCH=$GIT_BRANCH"
                '''
            }
        }

        // ===== Deploy =====
        stage('Build & Deploy') {
            when {
                anyOf {
                    branch 'develop'
                    expression { env.GIT_BRANCH?.contains('develop') }
                }
            }
            steps {
                sh '''
                echo "== CLEAN OLD CONTAINERS =="
                docker compose down || true

                echo "== START NEW =="
                docker compose up -d --build
                '''
            }
        }
    }
}