pipeline {
    agent any

    stages {

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