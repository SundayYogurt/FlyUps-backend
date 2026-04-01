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
                // บังคับชื่อโปรเจกต์ (-p) ให้เป็น flyup_backend ทุกครั้ง
                // เพื่อให้ docker compose down สามารถหาและปิดของเก่าได้ถูกต้องแม้ชื่อโฟลเดอร์ของ Jenkins รันจะเปลี่ยนไป
                sh '''
                echo "Deploying the application..."
                docker compose -p flyup_backend down
                docker compose -p flyup_backend up -d --build
                '''
            }
            post {
                success {
                    script {
                        echo "Deployment Successful!"
                        def payload = [
                            job: env.JOB_NAME,
                            status: "SUCCESS",
                            url: env.BUILD_URL
                        ]
                        httpRequest acceptType: 'APPLICATION_JSON',
                                    contentType: 'APPLICATION_JSON',
                                    httpMode: 'POST',
                                    requestBody: groovy.json.JsonOutput.toJson(payload),
                                    url: 'https://n8n.flyupapi.dev/webhook/jenkins-alert'
                    }
                }
                failure {
                    script {
                        echo "Deployment Failed!"
                        def payload = [
                            job: env.JOB_NAME,
                            status: "FAILURE",
                            url: env.BUILD_URL
                        ]
                        httpRequest acceptType: 'APPLICATION_JSON',
                                    contentType: 'APPLICATION_JSON',
                                    httpMode: 'POST',
                                    requestBody: groovy.json.JsonOutput.toJson(payload),
                                    url: 'https://n8n.flyupapi.dev/webhook/jenkins-alert'
                    }
                }
            }
        }
    }
}