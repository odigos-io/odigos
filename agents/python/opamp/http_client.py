import os
import threading
import time
from uuid_extensions import uuid7
import requests
import logging

from retry import retry

from opamp import opamp_pb2, anyvalue_pb2, utils


# Setup the logger
opamp_logger = logging.getLogger(__name__)
opamp_logger.setLevel(logging.DEBUG)
opamp_logger.disabled = True # Comment this line to enable the logger


class OpAMPHTTPClient:
    def __init__(self, event, condition: threading.Condition):
        self.server_host = os.getenv('ODIGOS_OPAMP_SERVER_HOST')
        self.instrumented_device_id = os.getenv('ODIGOS_INSTRUMENTATION_DEVICE_ID')
        self.server_url = f"http://{self.server_host}/v1/opamp"
        self.resource_attributes = {}
        self.running = True
        self.condition = condition
        self.event = event
        self.next_sequence_num = 0
        self.instance_uid = uuid7().__str__()

    def run(self):
        self.send_first_message_with_retry()
            
        if self.resource_attributes:
            self.event.set()
            
        self.fetch_data()
        
    @retry(tries=5, delay=2, exceptions=(requests.HTTPError,))
    def send_first_message_with_retry(self) -> None:
        first_message_server_to_agent = self.send_full_state()
        try:
            self.resource_attributes = utils.parse_first_message_to_resource_attributes(first_message_server_to_agent, opamp_logger)
        except Exception as e:
            opamp_logger.error(f"Error sending full state to OpAMP server: {e}")        

    def fetch_data(self):
        while self.running:
            with self.condition:
                try:
                    server_to_agent = self.send_heartbeat()
                    if server_to_agent.flags & opamp_pb2.ServerToAgentFlags_ReportFullState:
                        opamp_logger.debug("Received request to report full state")
                        self.send_full_state()

                except requests.RequestException as e:
                    opamp_logger.error(f"Error fetching data: {e}")
                time.sleep(30)
            
    def send_heartbeat(self):
        opamp_logger.debug("Sending heartbeat to OpAMP server...") 
        try:
            return self.send_agent_to_server_message(opamp_pb2.AgentToServer())
        except requests.RequestException as e:
            opamp_logger.error(f"Error sending heartbeat to OpAMP server: {e}")

    def send_full_state(self):
        opamp_logger.debug("Sending full state to OpAMP server...")
        
        identifying_attributes = [
            anyvalue_pb2.KeyValue(
                key="service.instance.id",
                value=anyvalue_pb2.AnyValue(string_value=self.instance_uid)
            ),
            anyvalue_pb2.KeyValue(
                key="process.pid",
                value=anyvalue_pb2.AnyValue(int_value=os.getpid())
            ),
            anyvalue_pb2.KeyValue(
                key="telemetry.sdk.language",
                value=anyvalue_pb2.AnyValue(string_value="python")
            )
        ]
        
        agent_description = opamp_pb2.AgentDescription(
            identifying_attributes=identifying_attributes,
            non_identifying_attributes=[]
        )
        return self.send_agent_to_server_message(opamp_pb2.AgentToServer(agent_description=agent_description))
    
    def send_agent_to_server_message(self, message: opamp_pb2.AgentToServer) -> opamp_pb2.ServerToAgent:
        message.instance_uid = self.instance_uid.encode('utf-8')
        message.sequence_num = self.next_sequence_num
        self.next_sequence_num += 1
        message_bytes = message.SerializeToString()
        
        headers = {
            "Content-Type": "application/x-protobuf",
            "X-Odigos-DeviceId": self.instrumented_device_id
        }
        
        try:
            response = requests.post(self.server_url, data=message_bytes, headers=headers, timeout=5)
            response.raise_for_status()
            
        except requests.Timeout:
            opamp_logger.error("Timeout sending message to OpAMP server")
            return opamp_pb2.ServerToAgent()
        except requests.ConnectionError as e:
            opamp_logger.error(f"Error sending message to OpAMP server: {e}")
            return opamp_pb2.ServerToAgent()
        
        server_to_agent = opamp_pb2.ServerToAgent()
        server_to_agent.ParseFromString(response.content)
        return server_to_agent

    def stop(self):
        self.running = False
        
        # Send agent disconnect message
        opamp_logger.debug("Sending agent disconnect message to OpAMP server...")
        disconnect_message = opamp_pb2.AgentToServer(agent_disconnect=opamp_pb2.AgentDisconnect())
        self.send_agent_to_server_message(disconnect_message)
        
        