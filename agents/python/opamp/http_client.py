import os
import time
import requests
import logging

from retry import retry

from opamp import opamp_pb2, anyvalue_pb2, utils


def setup_logger():
    # Create a dedicated logger for the OpAMPHTTPClient
    opamp_logger = logging.getLogger('OpAMPHTTPClient')
    opamp_logger.setLevel(logging.DEBUG)

    # Add a console handler
    console_handler = logging.StreamHandler()
    console_handler.setLevel(logging.DEBUG)
    formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s')
    console_handler.setFormatter(formatter)
    opamp_logger.addHandler(console_handler)

    # Comment the following line to disable the logger
    opamp_logger.disabled = True
    
    return opamp_logger

# Setup the logger
opamp_logger = setup_logger()


class OpAMPHTTPClient:
    def __init__(self, event):
        self.server_host = os.getenv('ODIGOS_OPAMP_SERVER_HOST')
        self.instrumented_device_id = os.getenv('ODIGOS_INSTRUMENTATION_DEVICE_ID')
        self.server_url = f"http://{self.server_host}/v1/opamp"
        self.resource_attributes = {}
        self.running = True
        self.event = event
        self.next_sequence_num = 0
        self.instance_uid = os.urandom(16)  # Generate a random UID for this instance

    def run(self):
        self.send_first_message_with_retry()
            
        if self.resource_attributes:
            self.event.set()
            
        # Start to fetch data
        self.fetch_data()
        
    @retry(tries=6, delay=5, exceptions=(requests.HTTPError,))
    def send_first_message_with_retry(self) -> None:
        first_message_server_to_agent = self.send_full_state()
        try:
            self.resource_attributes = utils.parse_first_message_to_resource_attributes(first_message_server_to_agent)
        except Exception as e:
            opamp_logger.error(f"Error sending full state to OpAMP server: {e}")        

    def fetch_data(self):
        while self.running:
            try:
                server_to_agent = self.send_heartbeat()
                ## TODO: Add logic for processing the server_to_agent messages
                
            except requests.RequestException as e:
                opamp_logger.error(f"Error fetching data: {e}")
            time.sleep(30)  # Poll messages every 30 seconds
            
    def send_heartbeat(self):
        opamp_logger.debug("Sending heartbeat to OpAMP server...")        
        try:
            return self.send_agent_to_server_message(opamp_pb2.AgentToServer())
        except requests.RequestException as e:
            opamp_logger.error(f"Error sending heartbeat to OpAMP server: {e}")

    def send_full_state(self):
        identifying_attributes = [
            anyvalue_pb2.KeyValue(
                key="service.instance.id",
                value=anyvalue_pb2.AnyValue(string_value=self.instance_uid.hex())
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
        
        message.instance_uid = self.instance_uid
        message.sequence_num = self.next_sequence_num
        self.next_sequence_num += 1
        message_bytes = message.SerializeToString()
        
        headers = {
            "Content-Type": "application/x-protobuf",
            "X-Odigos-DeviceId": self.instrumented_device_id
        }
        
        try:
            response = requests.post(self.server_url, data=message_bytes, headers=headers)
            response.raise_for_status()
        except requests.ConnectionError as e:
            opamp_logger.error(f"Error sending message to OpAMP server: {e}") ## TODO: remove this log
            return opamp_pb2.ServerToAgent()
        
        server_to_agent = opamp_pb2.ServerToAgent()
        server_to_agent.ParseFromString(response.content)
        return server_to_agent

    def stop(self):
        self.running = False
        