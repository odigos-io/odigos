docker build -t guniapp:latest .

kind load docker-image guniapp:latest

k apply -f k8s
