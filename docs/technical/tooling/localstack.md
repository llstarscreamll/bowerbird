# LocalStack

Emulates AWS for local backend work: S3, SQS, EventBridge, SSM.

- Compose service on `http://localhost:4566`
- Bootstrap: `apps/backend/scripts/init-localstack.sh` (runs on container start)
- Backend points at it with `AWS_ENDPOINT_URL`

Typical resources: `bowerbird-local-sqs`, `bowerbird-local-eventbridge`, bus `bowerbird-local-bus`, rule `bowerbird-local-rule`, bucket `bowerbird-local-bucket`, SSM secrets from `secrets.json`.

Go still runs locally under Air; Lambda handlers are reused via local pollers instead of deploying to LocalStack.

If `secrets.json` changes:

```bash
docker exec bowerbird-localstack /etc/localstack/init/ready.d/init-localstack.sh
```
