pipeline {
    agent any

    environment {
        DOCKER_IMAGE = 'sundayyogurt/flyup'
        DOCKER_TAG   = "${BUILD_NUMBER}"
        COMPOSE_FILE = '/home/ubuntu/flyup/docker-compose.yml'
    }

    stages {
        stage('Checkout') {
            steps {
                cleanWs()
                checkout scm
            }
        }

        stage('Build & Push') {
            steps {
                withCredentials([usernamePassword(
                            credentialsId: 'Flyup',
                            usernameVariable: 'DOCKER_USER',
                            passwordVariable: 'DOCKER_PASS'
                        )]) {
                    sh """
                        echo "\$DOCKER_PASS" | docker login -u "\$DOCKER_USER" --password-stdin
                        docker build -t ${DOCKER_IMAGE}:${DOCKER_TAG} -t ${DOCKER_IMAGE}:latest .
                        docker push ${DOCKER_IMAGE}:${DOCKER_TAG}
                        docker push ${DOCKER_IMAGE}:latest
                    """
                }
            }
        }

       stage('Deploy') {
           steps {
               sshagent(['ec2-ssh-cred']) {
                   sh '''
                       ssh -o StrictHostKeyChecking=no ubuntu@<EC2_PRIVATE_OR_HOST> "
                           docker pull sundayyogurt/flyup:latest &&
                           docker compose -f /home/ubuntu/flyup/docker-compose.yml up -d --no-deps --force-recreate app &&
                           echo Deploy Done
                       "
                   '''
               }
           }
       }
    }

    post {
        always {
            sh "docker logout || true"
            sh "docker rmi ${DOCKER_IMAGE}:${DOCKER_TAG} || true"
        }
        success {
            script {
                def payload = [
                    job: env.JOB_NAME, status: "SUCCESS",
                    url: env.BUILD_URL, build: env.BUILD_NUMBER
                ]
                try {
                    httpRequest acceptType: 'APPLICATION_JSON',
                    contentType: 'APPLICATION_JSON',
                    httpMode: 'POST',
                    requestBody: groovy.json.JsonOutput.toJson(payload),
                    url: 'https://n8n.flyupapi.dev/webhook/jenkins-alert',
                    validResponseCodes: '100:599'
                } catch (e) {
                    echo "Webhook notification failed (non-critical): ${e.message}"
                }
            }
        }
        failure {
            script {
                def payload = [
                    job: env.JOB_NAME, status: "FAILURE",
                    url: env.BUILD_URL, build: env.BUILD_NUMBER
                ]
                try {
                    httpRequest acceptType: 'APPLICATION_JSON',
                    contentType: 'APPLICATION_JSON',
                    httpMode: 'POST',
                    requestBody: groovy.json.JsonOutput.toJson(payload),
                    url: 'https://n8n.flyupapi.dev/webhook/jenkins-alert',
                    validResponseCodes: '100:599'
                } catch (e) {
                    echo "Webhook notification failed (non-critical): ${e.message}"
                }
            }
        }
    }
}
