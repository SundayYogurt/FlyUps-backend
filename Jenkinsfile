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

//        stage('Sonar Scan') {
//            agent {
//                docker {
//                    image 'sonarsource/sonar-scanner-cli:latest'
//                }
//            }
//            steps {
//                withSonarQubeEnv('sonarcloud') {
//                    sh '''
//                    sonar-scanner \
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

        stages {

                stage('Debug Branch') {
                    steps {
                        sh 'echo BRANCH_NAME=$BRANCH_NAME'
                        sh 'echo GIT_BRANCH=$GIT_BRANCH'
                    }
                }

                stage('Cleanup DB Port') {
                    steps {
                        sh '''
                        # หยุด container DB เก่า
                        docker ps -q --filter "name=flyup-backend-db-1" | xargs -r docker stop
                        docker ps -a -q --filter "name=flyup-backend-db-1" | xargs -r docker rm

                        # kill process ที่ใช้ port 5434 ถ้ายังมี
                        lsof -i :5434 | awk 'NR>1 {print $2}' | xargs -r kill -9
                        '''
                    }
                }

                stage('Build & Deploy') {
                    steps {
                        sh '''
                        # rebuild image ใหม่
                        docker compose build

                        # recreate container ใหม่จาก image ล่าสุด
                        docker compose up -d --force-recreate
                        '''
                    }
                }
            }
        }