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
                        # ใช้ docker compose down เพื่อหยุดและลบ container ทั้งหมดที่เกี่ยวข้องกับโปรเจกต์นี้
                        # โดยอ้างอิงจากไฟล์ docker-compose.yml ใน directory ปัจจุบัน
                        docker compose down --remove-orphans
                        '''
                    }
                }

                stage('Build & Deploy') {
                    steps {
                        sh '''
                        # build แบบไม่ใช้ cache เพื่อความมั่นใจว่าได้โค้ดใหม่ล่าสุด
                        docker compose build --no-cache

                        # รันขึ้นมาใหม่
                        docker compose up -d
                        '''
                    }
                }
            }
        }