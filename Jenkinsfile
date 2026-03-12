pipeline {
    agent any

    stages {

        stage('Checkout') {
            steps {
                git branch: 'develop',
                url: 'https://github.com/SundayYogurt/FlyUps-backend.git'
            }
        }

        stage('Build & Deploy') {
            steps {
                sh '''
                docker compose down
                docker compose up -d --build
                '''
            }
        }

    }
}