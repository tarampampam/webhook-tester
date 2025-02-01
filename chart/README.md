# WebHook Tester

## Usage

```shell
helm repo add tarampampam https://tarampampam.github.io/webhook-tester/helm-charts
helm repo update

helm install webhook-tester tarampampam/webhook-tester
```

Alternatively, add the following lines to your `Chart.yaml`:

```yaml
dependencies:
  - name: webhook-tester
    version: <version>
    repository: https://tarampampam.github.io/webhook-tester/helm-charts
```

And override the default values in your `values.yaml`:

```yaml
webhook-tester:
  # ...
  service: {port: 8800}
  # ...
```
