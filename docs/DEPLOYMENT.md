# Deployment Guide

This guide covers deploying your REST API to various cloud platforms.

## Table of Contents

- [Pre-deployment Checklist](#pre-deployment-checklist)
- [Docker Deployment](#docker-deployment)
- [AWS Deployment](#aws-deployment)
- [Google Cloud Run](#google-cloud-run)
- [Azure Container Instances](#azure-container-instances)
- [Heroku](#heroku)
- [DigitalOcean App Platform](#digitalocean-app-platform)

## Pre-deployment Checklist

Before deploying to production:

- [ ] Change JWT secret to a strong, random value
- [ ] Set `ENVIRONMENT=production`
- [ ] Configure database with production credentials
- [ ] Set up HTTPS/SSL certificates
- [ ] Configure CORS for your frontend domain
- [ ] Review and adjust rate limiting settings
- [ ] Set up monitoring and logging
- [ ] Configure backup strategy for database
- [ ] Review security settings
- [ ] Test all endpoints
- [ ] Set up CI/CD pipeline (optional)

## Docker Deployment

### Build Docker Image

```bash
# Build the image
docker build -t gin-rest-api:latest .

# Test locally
docker run -p 8080:8080 --env-file .env gin-rest-api:latest
```

### Push to Docker Registry

```bash
# Tag for Docker Hub
docker tag gin-rest-api:latest yourusername/gin-rest-api:latest

# Push to Docker Hub
docker push yourusername/gin-rest-api:latest
```

## AWS Deployment

### Option 1: AWS ECS (Elastic Container Service)

1. **Create ECR Repository**

```bash
# Create repository
aws ecr create-repository --repository-name gin-rest-api

# Login to ECR
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin YOUR_ACCOUNT_ID.dkr.ecr.us-east-1.amazonaws.com

# Tag and push image
docker tag gin-rest-api:latest YOUR_ACCOUNT_ID.dkr.ecr.us-east-1.amazonaws.com/gin-rest-api:latest
docker push YOUR_ACCOUNT_ID.dkr.ecr.us-east-1.amazonaws.com/gin-rest-api:latest
```

2. **Create ECS Cluster**

```bash
aws ecs create-cluster --cluster-name gin-rest-cluster
```

3. **Create Task Definition**

Create `task-definition.json`:

```json
{
  "family": "gin-rest-api",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "containerDefinitions": [
    {
      "name": "gin-rest-api",
      "image": "YOUR_ACCOUNT_ID.dkr.ecr.us-east-1.amazonaws.com/gin-rest-api:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "ENVIRONMENT",
          "value": "production"
        },
        {
          "name": "DATABASE_TYPE",
          "value": "mysql"
        }
      ],
      "secrets": [
        {
          "name": "JWT_SECRET",
          "valueFrom": "arn:aws:secretsmanager:region:account:secret:jwt-secret"
        }
      ]
    }
  ]
}
```

Register task:
```bash
aws ecs register-task-definition --cli-input-json file://task-definition.json
```

4. **Create Service**

```bash
aws ecs create-service \
  --cluster gin-rest-cluster \
  --service-name gin-rest-service \
  --task-definition gin-rest-api \
  --desired-count 2 \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-xxx],securityGroups=[sg-xxx],assignPublicIp=ENABLED}"
```

### Option 2: AWS Elastic Beanstalk

1. Install EB CLI:
```bash
pip install awsebcli
```

2. Initialize EB application:
```bash
eb init -p docker gin-rest-api
```

3. Create environment:
```bash
eb create production-env
```

4. Deploy:
```bash
eb deploy
```

5. Set environment variables:
```bash
eb setenv ENVIRONMENT=production DATABASE_TYPE=mysql JWT_SECRET=your-secret
```

### Option 3: AWS RDS for MySQL

1. **Create RDS Instance**

```bash
aws rds create-db-instance \
  --db-instance-identifier gin-rest-db \
  --db-instance-class db.t3.micro \
  --engine mysql \
  --master-username admin \
  --master-user-password yourpassword \
  --allocated-storage 20
```

2. **Update environment variables** with RDS endpoint

## Google Cloud Run

1. **Build and push to Google Container Registry**

```bash
# Configure gcloud
gcloud auth configure-docker

# Build image
gcloud builds submit --tag gcr.io/PROJECT_ID/gin-rest-api

# Or build locally and push
docker build -t gcr.io/PROJECT_ID/gin-rest-api .
docker push gcr.io/PROJECT_ID/gin-rest-api
```

2. **Deploy to Cloud Run**

```bash
gcloud run deploy gin-rest-api \
  --image gcr.io/PROJECT_ID/gin-rest-api \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars ENVIRONMENT=production,DATABASE_TYPE=mysql \
  --set-secrets JWT_SECRET=jwt-secret:latest
```

3. **Set up Cloud SQL (MySQL)**

```bash
# Create instance
gcloud sql instances create gin-rest-mysql \
  --database-version=MYSQL_8_0 \
  --tier=db-f1-micro \
  --region=us-central1

# Create database
gcloud sql databases create gin_rest_db --instance=gin-rest-mysql

# Connect Cloud Run to Cloud SQL
gcloud run services update gin-rest-api \
  --add-cloudsql-instances PROJECT_ID:us-central1:gin-rest-mysql
```

## Azure Container Instances

1. **Create Resource Group**

```bash
az group create --name gin-rest-rg --location eastus
```

2. **Create Container Registry**

```bash
az acr create --resource-group gin-rest-rg --name ginrestacr --sku Basic
az acr login --name ginrestacr
```

3. **Build and Push Image**

```bash
docker tag gin-rest-api ginrestacr.azurecr.io/gin-rest-api:latest
docker push ginrestacr.azurecr.io/gin-rest-api:latest
```

4. **Deploy Container**

```bash
az container create \
  --resource-group gin-rest-rg \
  --name gin-rest-api \
  --image ginrestacr.azurecr.io/gin-rest-api:latest \
  --cpu 1 \
  --memory 1 \
  --registry-login-server ginrestacr.azurecr.io \
  --registry-username USERNAME \
  --registry-password PASSWORD \
  --dns-name-label gin-rest-api \
  --ports 8080 \
  --environment-variables ENVIRONMENT=production DATABASE_TYPE=mysql \
  --secure-environment-variables JWT_SECRET=your-secret
```

## Heroku

1. **Install Heroku CLI**

```bash
# macOS
brew tap heroku/brew && brew install heroku

# Or download from heroku.com
```

2. **Login and Create App**

```bash
heroku login
heroku create gin-rest-api
```

3. **Add MySQL Addon**

```bash
heroku addons:create cleardb:ignite
```

4. **Set Environment Variables**

```bash
heroku config:set ENVIRONMENT=production
heroku config:set DATABASE_TYPE=mysql
heroku config:set JWT_SECRET=your-secret-key
```

5. **Deploy**

```bash
# Using Container Registry
heroku container:login
heroku container:push web
heroku container:release web

# Or using Git
git push heroku main
```

6. **Open App**

```bash
heroku open
heroku logs --tail
```

## DigitalOcean App Platform

1. **Create `app.yaml`**

```yaml
name: gin-rest-api
services:
- name: api
  github:
    repo: yourusername/gin-rest-template
    branch: main
    deploy_on_push: true
  dockerfile_path: Dockerfile
  http_port: 8080
  instance_count: 1
  instance_size_slug: basic-xxs
  routes:
  - path: /
  envs:
  - key: ENVIRONMENT
    value: production
  - key: DATABASE_TYPE
    value: mysql
  - key: JWT_SECRET
    value: ${JWT_SECRET}
    type: SECRET
databases:
- name: mysql-db
  engine: MYSQL
  version: "8"
```

2. **Deploy via CLI**

```bash
# Install doctl
brew install doctl

# Authenticate
doctl auth init

# Create app
doctl apps create --spec app.yaml
```

3. **Or deploy via Web UI**
- Go to DigitalOcean App Platform
- Click "Create App"
- Connect your GitHub repository
- Configure environment variables
- Deploy

## Production Best Practices

### Security

1. **Use HTTPS**
   - Set up SSL/TLS certificates
   - Use Let's Encrypt or cloud provider certificates

2. **Environment Variables**
   - Never commit secrets to repository
   - Use secret management services
   - Rotate credentials regularly

3. **Database Security**
   - Use strong passwords
   - Enable encryption at rest
   - Restrict network access
   - Regular backups

### Monitoring

1. **Application Monitoring**
   - Set up health check endpoints
   - Monitor response times
   - Track error rates

2. **Infrastructure Monitoring**
   - Monitor CPU and memory usage
   - Track database performance
   - Set up alerts

3. **Logging**
   - Centralized logging (ELK, CloudWatch, Stackdriver)
   - Log rotation
   - Error tracking (Sentry)

### Scaling

1. **Horizontal Scaling**
   - Use load balancers
   - Auto-scaling groups
   - Multiple instances

2. **Database Scaling**
   - Read replicas
   - Connection pooling
   - Caching (Redis)

### CI/CD Pipeline

Example GitHub Actions workflow (`.github/workflows/deploy.yml`):

```yaml
name: Deploy

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Build Docker image
        run: docker build -t gin-rest-api .
      
      - name: Run tests
        run: docker run gin-rest-api go test ./...
      
      - name: Deploy to production
        run: |
          # Your deployment commands here
```

## Rollback Strategy

Always have a rollback plan:

```bash
# AWS ECS
aws ecs update-service --cluster CLUSTER --service SERVICE --task-definition PREVIOUS_TASK_DEF

# Google Cloud Run
gcloud run services update-traffic SERVICE --to-revisions=REVISION=100

# Heroku
heroku rollback

# Docker
docker pull gin-rest-api:previous-tag
docker stop current-container
docker run gin-rest-api:previous-tag
```

## Cost Optimization

- Start with small instances and scale as needed
- Use auto-scaling to handle traffic spikes
- Implement caching to reduce database load
- Use CDN for static assets
- Monitor and optimize database queries
- Consider reserved instances for stable workloads

## Support

For deployment issues, check:
- Cloud provider documentation
- Application logs
- Database connection status
- Environment variables configuration

Need help? Open an issue on GitHub!
