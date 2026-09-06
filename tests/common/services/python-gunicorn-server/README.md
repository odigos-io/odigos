docker build -t guniapp:latest .

./scripts/k0s-cluster.sh load guniapp:latest

k apply -f k8s
