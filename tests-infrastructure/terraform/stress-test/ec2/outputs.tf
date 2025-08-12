output "instance_public_ip" {
  value = aws_instance.monitoring.public_ip
}

output "grafana_url" {
  value = "http://${aws_instance.monitoring.public_ip}:3000"
}

output "attached_sg_id" {
  value = data.terraform_remote_state.eks.outputs.eks_node_sg_id
}
