pipeline {
    agent any

    environment {
        DOCKER_IMAGE = 'sundayyogurt/flyup'
        DOCKER_TAG   = "${BUILD_NUMBER}"
        COMPOSE_FILE = '/root/infra/backend/docker-compose.yml'
    }

    stages {
        stage('Checkout') {
            steps {
                cleanWs()
                checkout scm
            }
        }

        stage('Build & Push') {
            when {
                expression {
                    return env.GIT_BRANCH == 'origin/develop' || env.GIT_BRANCH == 'develop' ||
                           env.GIT_BRANCH == 'origin/main'   || env.GIT_BRANCH == 'main'
                }
            }
            steps {
                withCredentials([usernamePassword(
                    credentialsId: 'dockerhub-credentials',
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
            when {
                expression {
                    return env.GIT_BRANCH == 'origin/develop' || env.GIT_BRANCH == 'develop' ||
                           env.GIT_BRANCH == 'origin/main'   || env.GIT_BRANCH == 'main'
                }
            }
            steps {
                sh """
                    docker pull ${DOCKER_IMAGE}:latest
                    docker compose -f ${COMPOSE_FILE} up -d --no-deps --force-recreate app
                    echo "✅ Deploy Done"
                """
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