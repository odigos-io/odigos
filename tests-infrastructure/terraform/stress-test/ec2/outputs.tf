output "monitoring_instance_id" {
  description = "Instance ID of the Prometheus receiver EC2"
  value       = aws_instance.monitoring.id
}

output "monitoring_instance_private_ip" {
  description = "Private IP of the Prometheus receiver EC2 (use in remote_write URL)"
  value       = aws_instance.monitoring.private_ip
}

output "monitoring_ec2_sg_id" {
  description = "Security Group ID attached to the receiver EC2"
  value       = aws_security_group.monitoring_ec2_sg.id
}

output "prometheus_remote_write_url" {
  description = "Agent remote_write target"
  value       = "http://${aws_instance.monitoring.private_ip}:9090/api/v1/write"
}

output "clickhouse_connection_info" {
  description = "ClickHouse connection information"
  value = {
    endpoint = "tcp://${aws_instance.monitoring.private_ip}:9000"
    database = "otel"
    username = "default"
    password = "stresstest"
    namespace = "odigos-system"
  }
}
