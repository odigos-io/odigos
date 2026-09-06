docker build -t guniapp:latest .

k3d image import guniapp:latest

k apply -f k8s
