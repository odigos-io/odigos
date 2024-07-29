import os
import time
import threading
import requests
import logging

from uuid_extensions import uuid7
from opentelemetry.semconv.resource import ResourceAttributes
from opentelemetry.context import (
    _SUPPRESS_HTTP_INSTRUMENTATION_KEY,
    attach,
    detach,
    set_value,
)

from opamp import opamp_pb2, anyvalue_pb2, utils


# Setup the logger
opamp_logger = logging.getLogger(__name__)
opamp_logger.setLevel(logging.DEBUG)
opamp_logger.disabled = True # Comment this line to enable the logger


class OpAMPHTTPClient:
    def __init__(self, event, condition: threading.Condition):
        self.server_host = os.getenv('ODIGOS_OPAMP_SERVER_HOST')
        self.instrumentation_device_id = os.getenv('ODIGOS_INSTRUMENTATION_DEVICE_ID')
        self.server_url = f"http://{self.server_host}/v1/opamp"
        self.resource_attributes = {}
        self.running = True
        self.condition = condition
        self.event = event
        self.next_sequence_num = 0
        self.instance_uid = uuid7().__str__()
        self.remote_config_status = None

    def start(self):
        self.client_thread = threading.Thread(target=self.run, name="OpAMPClientThread", daemon=True)
        self.client_thread.start()
        
    def run(self):
        if not self.mandatory_env_vars_set():
            self.event.set()
            return
        
        self.send_first_message_with_retry()
        
        self.event.set()
        
        self.worker()
        
    def send_first_message_with_retry(self) -> None:
        max_retries = 5
        delay = 2
        for attempt in range(1, max_retries + 1):
            try:
                first_message_server_to_agent = self.send_full_state()
                self.update_remote_config_status(first_message_server_to_agent)
                self.resource_attributes = utils.parse_first_message_to_resource_attributes(first_message_server_to_agent, opamp_logger)
                break
            except Exception as e:
                opamp_logger.error(f"Error sending full state to OpAMP server: {e}")
            
            if attempt < max_retries:
                time.sleep(delay)

    def worker(self):
        while self.running:
            with self.condition:
                try:
                    server_to_agent = self.send_heartbeat()
                    if self.update_remote_config_status(server_to_agent):
                        opamp_logger.info("Remote config updated, applying changes...")
                        # TODO: implement changes based on the remote config

                    if server_to_agent.flags & opamp_pb2.ServerToAgentFlags_ReportFullState:
                        opamp_logger.info("Received request to report full state")
                        server_to_agent = self.send_full_state()
                        self.update_remote_config_status(server_to_agent)

                except requests.RequestException as e:
                    opamp_logger.error(f"Error fetching data: {e}")
                self.condition.wait(30)

    def send_heartbeat(self) -> opamp_pb2.ServerToAgent:
        opamp_logger.debug("Sending heartbeat to OpAMP server...") 
        try:
            agent_to_server = opamp_pb2.AgentToServer(remote_config_status=self.remote_config_status)
            return self.send_agent_to_server_message(agent_to_server)
        except requests.RequestException as e:
            opamp_logger.error(f"Error sending heartbeat to OpAMP server: {e}")

    def send_full_state(self):
        opamp_logger.info("Sending full state to OpAMP server...")
        
        identifying_attributes = [
            anyvalue_pb2.KeyValue(
                key=ResourceAttributes.SERVICE_INSTANCE_ID,
                value=anyvalue_pb2.AnyValue(string_value=self.instance_uid)
            ),
            anyvalue_pb2.KeyValue(
                key=ResourceAttributes.PROCESS_PID,
                value=anyvalue_pb2.AnyValue(int_value=os.getpid())
            ),
            anyvalue_pb2.KeyValue(
                key=ResourceAttributes.TELEMETRY_SDK_LANGUAGE,
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
        if self.remote_config_status:
            message.remote_config_status.CopyFrom(self.remote_config_status)
    
        self.next_sequence_num += 1
        message_bytes = message.SerializeToString()
        
        headers = {
            "Content-Type": "application/x-protobuf",
            "X-Odigos-DeviceId": self.instrumentation_device_id
        }
        
        try:
            agent_message = attach(set_value(_SUPPRESS_HTTP_INSTRUMENTATION_KEY, True))
            response = requests.post(self.server_url, data=message_bytes, headers=headers, timeout=5)
            response.raise_for_status()
        except requests.Timeout:
            opamp_logger.error("Timeout sending message to OpAMP server")
            return opamp_pb2.ServerToAgent()
        except requests.ConnectionError as e:
            opamp_logger.error(f"Error sending message to OpAMP server: {e}")
            return opamp_pb2.ServerToAgent()
        finally:
            detach(agent_message)
        
        server_to_agent = opamp_pb2.ServerToAgent()
        try:
            server_to_agent.ParseFromString(response.content)
        except NotImplementedError as e:
            opamp_logger.error(f"Error parsing response from OpAMP server: {e}")
            return opamp_pb2.ServerToAgent()
        return server_to_agent
        
    def mandatory_env_vars_set(self):
        mandatory_env_vars = {
            "ODIGOS_OPAMP_SERVER_HOST": self.server_host,
            "ODIGOS_INSTRUMENTATION_DEVICE_ID": self.instrumentation_device_id
        }
        
        for env_var, value in mandatory_env_vars.items():
            if not value:
                opamp_logger.error(f"{env_var} environment variable not set")
                return False
        
        return True        
    
    def shutdown(self):
        self.running = False
        
        opamp_logger.info("Sending agent disconnect message to OpAMP server...")
        disconnect_message = opamp_pb2.AgentToServer(agent_disconnect=opamp_pb2.AgentDisconnect())
        
        with self.condition:
            self.condition.notify_all()
        self.client_thread.join()
        
        self.send_agent_to_server_message(disconnect_message)
        
    def update_remote_config_status(self, server_to_agent: opamp_pb2.ServerToAgent) -> bool:
        if server_to_agent.HasField("remote_config"):
            remote_config_hash = server_to_agent.remote_config.config_hash
            remote_config_status = opamp_pb2.RemoteConfigStatus(last_remote_config_hash=remote_config_hash)
            self.remote_config_status = remote_config_status
            return True
        
        return False        