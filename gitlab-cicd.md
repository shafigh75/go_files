---

# 🚀 Comprehensive Guide to GitLab CI/CD

## 🏗️ Part 1: Setting Up GitLab Runner

### ✅ Prerequisites

* A GitLab account (self-hosted or GitLab.com)
* A project created in GitLab
* A Linux server (or Windows/macOS) with root access

---

### 🔧 1.1 Install GitLab Runner

```bash
# Debian/Ubuntu
curl -L https://packages.gitlab.com/install/repositories/runner/gitlab-runner/script.deb.sh | sudo bash
sudo apt install gitlab-runner
```

---

### 🔐 1.2 Register the Runner

```bash
sudo gitlab-runner register
```

**During registration, you'll be asked:**

| Prompt              | Example                                                                                        |
| ------------------- | ---------------------------------------------------------------------------------------------- |
| GitLab instance URL | `https://gitlab.com/` or your self-hosted URL                                                  |
| Registration token  | Get from your GitLab project: `Settings > CI/CD > Runners > Set up a specific runner manually` |
| Description         | `docker-runner`                                                                                |
| Tags                | `docker`, `build`, `test`                                                                      |
| Executor            | e.g. `docker`, `shell`, `virtualbox`, etc.                                                     |
| Docker image        | `alpine:latest`, `node:18`, etc.                                                               |

---

### 🛠️ 1.3 Start & Enable the Runner

```bash
sudo gitlab-runner start
sudo systemctl enable gitlab-runner
```

Check status:

```bash
sudo gitlab-runner status
```

---

## 📦 Part 2: Basic `.gitlab-ci.yml` Structure

```yaml
stages:
  - build
  - test
  - deploy

variables:
  APP_NAME: "MyCoolApp"

build-job:
  stage: build
  script:
    - echo "Building $APP_NAME"

test-job:
  stage: test
  script:
    - echo "Running tests..."

deploy-job:
  stage: deploy
  script:
    - echo "Deploying $APP_NAME"
```

---

## ⚙️ Part 3: Full Multi-Stage CI/CD Example

```yaml
stages:
  - prepare
  - build
  - test
  - deploy

variables:
  APP_NAME: "MyCoolApp"
  IMAGE_TAG: "$CI_COMMIT_SHORT_SHA"

# Define a global image
default:
  image: alpine:latest

# Prepare Environment
prepare:
  stage: prepare
  tags: [shell]
  script:
    - echo "Preparing build for $APP_NAME"
    - apk add --no-cache curl git

# Build Docker Image
build:
  stage: build
  tags: [docker]
  image: docker:latest
  services:
    - docker:dind
  script:
    - docker build -t registry.gitlab.com/$CI_PROJECT_PATH:$IMAGE_TAG .
    - docker push registry.gitlab.com/$CI_PROJECT_PATH:$IMAGE_TAG

# Run Tests
unit_test:
  stage: test
  tags: [test-runner]
  image: node:18
  script:
    - npm ci
    - npm run test

# Deploy to production
deploy_prod:
  stage: deploy
  only:
    - main
  tags: [deploy-runner]
  script:
    - echo "Deploying version $IMAGE_TAG to production..."
    - ./deploy.sh
```

---

## 🔐 Part 4: Secrets & Environment Variables

### 🌍 GitLab UI (Recommended)

1. Go to your **project → Settings → CI/CD → Variables**
2. Click `Add Variable`
3. Example:

| Key            | Value    | Type             |
| -------------- | -------- | ---------------- |
| `PROD_API_KEY` | `s3cr3t` | Protected/Masked |

These will be available as environment variables in jobs.

### 🔐 Use in `.gitlab-ci.yml`

```yaml
script:
  - echo "API Key is $PROD_API_KEY"
```

---

## 🏃 Part 5: Run Each Stage on Separate Runner

When registering runners, assign unique **tags** like:

* `docker`
* `test-runner`
* `deploy-runner`

Then in each job:

```yaml
job_name:
  tags:
    - docker
```

Each job will only run on runners with matching tags.

---

## 🧪 Part 6: Artifacts and Caching

### 🔄 Cache Dependencies (e.g. Node.js)

```yaml
cache:
  paths:
    - node_modules/
```

### 📦 Upload Artifacts Between Jobs

```yaml
build:
  stage: build
  script:
    - npm run build
  artifacts:
    paths:
      - dist/
    expire_in: 1 hour
```

---

## 🛑 Part 7: Conditional Execution

```yaml
deploy_prod:
  only:
    - main
```

Or with rules:

```yaml
rules:
  - if: '$CI_COMMIT_BRANCH == "main"'
    when: always
```

---

## 🔍 Part 8: Debugging Jobs

Run failed jobs locally:

```bash
gitlab-runner exec shell <job_name>
```

Use `CI_DEBUG_TRACE` to show more logs:

```yaml
variables:
  CI_DEBUG_TRACE: "true"
```

---

## 🏁 Final Tips

* **Use `before_script`** for shared steps
* **Use `include:`** to split pipeline into reusable templates
* **Use `protected` variables** for secure environments
* **Keep runners updated**
* **Tag runners wisely**

---

## ✅ Resources

* [GitLab CI/CD Docs](https://docs.gitlab.com/ee/ci/)
* [`.gitlab-ci.yml` Reference](https://docs.gitlab.com/ee/ci/yaml/)
* [Runner Installation](https://docs.gitlab.com/runner/install/)
* [Runner Executors](https://docs.gitlab.com/runner/executors/)

---

Would you like an example project repo or a Docker-based CI/CD setup (with Nginx, DB, etc.)? I can generate that too.
