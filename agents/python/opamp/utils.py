import json
import logging
from opamp import opamp_pb2

def parse_first_message_to_resource_attributes(first_message_server_to_agent: opamp_pb2.ServerToAgent, logger: logging.Logger) -> dict:
    config_map = first_message_server_to_agent.remote_config.config.config_map
    
    if "SDK" not in config_map:
        logger.error("SDK not found in config map, returning empty resource attributes")
        return {}
    
    remote_resource_attributes = json.loads(config_map["SDK"].body)
    
    return {item['key']: item['value'] for item in remote_resource_attributes.get('remoteResourceAttributes', [])}