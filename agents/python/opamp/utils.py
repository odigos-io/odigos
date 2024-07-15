import json
import logging
from opamp import opamp_pb2

def parse_first_message_to_resource_attributes(first_message_server_to_agent: opamp_pb2.ServerToAgent, logger: logging.Logger) -> dict:
    config_map = first_message_server_to_agent.remote_config.config.config_map
    
    if "SDK" not in config_map:
        logger.error("SDK not found in config map, returning empty resource attributes")
        return {}
    
    try:
        sdk_config = json.loads(config_map["SDK"].body)
    except json.JSONDecodeError as e:
        logger.error(f"Error decoding SDK config: {e}")
        return {}
    
    remote_resource_attributes = sdk_config.get('remoteResourceAttributes', [])
    
    if not remote_resource_attributes:
        logger.error('missing "remoteResourceAttributes" section in OpAMP server remote config on first server to agent message')
        return {}
    
    return {item['key']: item['value'] for item in remote_resource_attributes}