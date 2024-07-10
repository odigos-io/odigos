import json
from opamp import opamp_pb2


def parse_first_message_to_resource_attributes(first_message_server_to_agent: opamp_pb2.ServerToAgent) -> dict:
    config_map = first_message_server_to_agent.remote_config.config.config_map
    
    if "SDK" in config_map:
        remote_resource_attributes = json.loads(config_map["SDK"].body)
    else:
        remote_resource_attributes = {"remoteResourceAttributes": []}
    
    returned_dict = {}
    for item in remote_resource_attributes.get('remoteResourceAttributes', []):
        key = item['key']
        value = item['value']
        returned_dict[key] = value    
    return returned_dict