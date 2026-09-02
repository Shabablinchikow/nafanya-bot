# nafanya-bot

Щитпост бот для Telegram на основе ChatGPT

## Deployment

Push to `master` builds the image and deploys it to a Kubernetes cluster with Helm
(`.github/workflows/build-docker.yml`). Cluster, registry and application configuration come
from repository secrets and variables; nothing environment-specific lives in this repo.
