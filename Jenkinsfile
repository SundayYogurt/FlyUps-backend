pipeline {
    agent any

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Build & Deploy') {
            when {
                expression {
                    return env.GIT_BRANCH == 'origin/develop' || env.GIT_BRANCH == 'develop' || env.GIT_BRANCH == 'origin/main' || env.GIT_BRANCH == 'main'
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