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
                                # 1. พิสูจน์เรื่องไฟล์ .env (ถ้าไม่มี ให้สร้างสดๆ ตรงนี้เลย)
                                if [ ! -f .env ]; then
                                    echo "DSN=host=db user=root password=root dbname=flyup port=5432 sslmode=disable" > .env
                                    echo "HTTP_PORT=3000" >> .env
                                    echo "--- Created .env file because it was missing ---"
                                fi

                                # 2. ล้างตัวเก่าให้สะอาด
                                docker compose down --remove-orphans

                                # 3. รันใหม่และบังคับให้รอ (The Secret Sauce)
                                docker compose up -d --build

                                echo "Waiting 10 seconds for DB to be READY (Jenkins Context)..."
                                sleep 10

                                # 4. เช็ค Log เพื่อยืนยัน
                                docker ps -a
                                docker logs flyup-backend-app-1
                                '''
                            }
                        }
                    }
                }