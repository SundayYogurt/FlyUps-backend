pipeline {
    agent any

    environment {
        SONAR_TOKEN = credentials('SonarQubeTokens')
    }

    stages {

//        stage('Go Build & Test') {
//            agent {
//                docker {
//                    image 'golang:1.25'
//                }
//            }
//            steps {
//                sh '''
//                go mod tidy
//                go test ./... -coverprofile=coverage.out
//                '''
//            }
//        }

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

        stage('Debug Branch') {
            steps {
                sh 'echo BRANCH_NAME=$BRANCH_NAME'
                sh 'echo GIT_BRANCH=$GIT_BRANCH'
            }
        }

            stage('Build & Deploy') {
                when {
                    expression {
                        return env.GIT_BRANCH?.contains('develop')
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
}