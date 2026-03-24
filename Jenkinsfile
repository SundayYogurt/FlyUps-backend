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

                        stage('Force Cleanup Port 5434') {
                            steps {
                                sh '''
                                echo "--- Starting Aggressive Cleanup ---"

                                # 1. หยุดและลบทุก Container ที่ใช้ Port 5434 (ไม่สนชื่อโปรเจกต์)
                                # คำสั่งนี้จะหา Container ID ที่ Map พอร์ต 5434 แล้วสั่งลบทิ้งทันที
                                docker ps -q --filter "publish=5434" | xargs -r docker rm -f || true

                                # 2. กวาดล้าง Container ที่ชื่อมีคำว่า flyup (ป้องกันเรื่องชื่อมี s หรือไม่มี s)
                                docker ps -aq --filter "name=flyup" | xargs -r docker rm -f || true

                                # 3. ใช้ docker compose down ตามปกติเพื่อล้าง Network
                                docker compose down --remove-orphans || true

                                # 4. ตรวจสอบพอร์ตด้วยคำสั่งพื้นฐาน (เผื่อมี process นอก docker)
                                # ใช้ fuser หรือแก้ด้วยการเช็คผ่าน /proc ถ้า lsof ไม่มี
                                (ss -lntp | grep :5434 | awk -F, '{print $2}' | awk -F= '{print $2}' | xargs -r kill -9) || true
                                '''
                            }
                        }

                        stage('Build & Deploy') {
                            steps {
                                sh '''
                                # Build ใหม่แบบไม่ใช้ cache
                                docker compose build --no-cache

                                # รันขึ้นมาใหม่
                                docker compose up -d --force-recreate
                                '''
                            }
                        }
                    }
                }