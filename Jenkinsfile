pipeline {
    agent any

    environment {
        // สามารถเพิ่มตัวแปร Environment ของ Jenkins ได้ที่นี่ถ้าจำเป็น
        // ตัวอย่าง: DOCKER_IMAGE = 'my-app'
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Build & Deploy') {
            when {
                anyOf {
                    branch 'develop'
                    branch 'main'
                }
            }
            steps {
                // คำสั่งรัน Docker Compose
                // โดย docker-compose.yml จะไปดึง .env จาก /etc/flyup/.env บนเครื่อง Server (Host)
                sh '''
                echo "Deploying the application..."
                docker compose down
                docker compose up -d --build
                '''
            }
            post {
                success {
                    echo "Deployment Successful!"
                }
                failure {
                    echo "Deployment Failed!"
                }
            }
        }
    }
}