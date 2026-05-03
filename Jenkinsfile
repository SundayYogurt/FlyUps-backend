pipeline {
    agent any

    environment {
        PROJECT_NAME = "flyup_backend"
    }

    stages {
        stage('Checkout') {
            steps {
                // Clean workspace ก่อน checkout เพื่อป้องกัน git directory error
                cleanWs()
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
                sh '''
                echo "🚀 Deploying..."

                # 🔥 kill container ที่แอบใช้ port 3000 (กันชน)
                docker ps -q --filter "publish=3000" | xargs -r docker rm -f

                # 🔥 ปิดของเก่า (ไม่ลบ DB)
                docker compose --env-file /etc/flyup/.env -p $PROJECT_NAME down --remove-orphans

                # 🔥 build + run ใหม่
                docker compose --env-file /etc/flyup/.env -p $PROJECT_NAME up -d --build

                # 🔥 ensure n8n is running (แยก compose)
                cd /root/n8n && docker compose up -d
                cd /root/FlyUps-backend

                echo "⏳ Waiting for services to boot up..."
                sleep 15

                echo "✅ Deploy Done"
                '''
            }
        }
    }

    post {
        success {
            script {
                echo "Deployment Successful!"
                def payload = [
                    job: env.JOB_NAME,
                    status: "SUCCESS",
                    url: env.BUILD_URL,
                    build: env.BUILD_NUMBER
                ]
                try {
                    httpRequest acceptType: 'APPLICATION_JSON',
                                contentType: 'APPLICATION_JSON',
                                httpMode: 'POST',
                                requestBody: groovy.json.JsonOutput.toJson(payload),
                                url: 'https://n8n.flyupapi.dev/webhook/jenkins-alert',
                                validResponseCodes: '100:599' // ไม่ fail แม้ n8n จะ 502
                } catch (e) {
                    echo "Webhook notification failed (non-critical): ${e.message}"
                }
            }
        }
        failure {
            script {
                echo "Deployment Failed!"
                def payload = [
                    job: env.JOB_NAME,
                    status: "FAILURE",
                    url: env.BUILD_URL,
                    build: env.BUILD_NUMBER
                ]
                try {
                    httpRequest acceptType: 'APPLICATION_JSON',
                                contentType: 'APPLICATION_JSON',
                                httpMode: 'POST',
                                requestBody: groovy.json.JsonOutput.toJson(payload),
                                url: 'https://n8n.flyupapi.dev/webhook/jenkins-alert',
                                validResponseCodes: '100:599' // ไม่ fail แม้ n8n จะ 502
                } catch (e) {
                    echo "Webhook notification failed (non-critical): ${e.message}"
                }
            }
        }
    }
}