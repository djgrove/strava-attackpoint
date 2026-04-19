---
description: Deploy the Lambda infrastructure via Pulumi
user_invocable: true
---

Run the following command to deploy the infrastructure:

```bash
cd /Users/devon/dev/strava-attackpoint/infra && pulumi up --yes
```

After deployment, check if the API URL changed in the Pulumi output. If it did, update the `ProxyURL` constant in `internal/strava/auth.go` and the `PROXY_URL` constants in `docs/index.html` and `docs/callback.html`, then rebuild with `go build -o strava-ap .`.
