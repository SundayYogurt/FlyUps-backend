pipeline {
    agent any

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

//        stage('Test & Coverage') {
//            steps {
//                sh 'go test ./... -coverprofile=coverage.out'
//            }
//        }
//
//        stage('Sonar Scan') {
//            steps {
//                withSonarQubeEnv('sonarcloud') {
//                    sh '''
//                    rm -rf sonar-scanner*
//
//                    apt-get update
//                    apt-get install -y curl unzip
//
//                    curl -sSLo sonar-scanner.zip https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/sonar-scanner-cli-5.0.1.3006-linux.zip
//                    unzip sonar-scanner.zip
//
//                    ./sonar-scanner-*/bin/sonar-scanner \
//                      -Dsonar.projectKey=sundayyogurt_flyup \
//                      -Dsonar.organization=sundayyogurt \
//                      -Dsonar.sources=. \
//                      -Dsonar.exclusions=**/*_test.go \
//                      -Dsonar.go.coverage.reportPaths=coverage.out \
//                      -Dsonar.login=$SONAR_TOKEN
//                    '''
//                }
//            }
//        }

        stage('Build & Deploy') {
            when {
                expression {
                    return env.GIT_BRANCH == 'origin/develop' || env.GIT_BRANCH == 'develop'
                }
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