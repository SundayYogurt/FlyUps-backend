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

                stage('Debug Branch') {
                            steps {
                                sh 'echo BRANCH_NAME=$BRANCH_NAME'
                                sh 'echo GIT_BRANCH=$GIT_BRANCH'
                            }
                        }

                        stage('Cleanup & Down') {
                            steps {
                                sh '''
                                # หยุด container เก่า และลบ orphan container
                                docker compose down --remove-orphans

                                # kill process ที่อาจใช้ port 5434 (DB) ถ้ามี
                                lsof -i :5434 | awk 'NR>1 {print $2}' | xargs -r kill -9
                                '''
                            }
                        }

                        stage('Build & Deploy') {
                            steps {
                                sh '''
                                # rebuild image แบบไม่ใช้ cache → ได้ code ใหม่ล่าสุด
                                docker compose build --no-cache

                                # run container ใหม่
                                docker compose up -d --force-recreate
                                '''
                            }
                        }
            }
        }