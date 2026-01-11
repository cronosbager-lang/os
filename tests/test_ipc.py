#!/usr/bin/env python3
"""Test IPC client functionality"""

import asyncio
import json
import os
import socket
import struct
import sys
import tempfile
import threading
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'agent'))

from ipc.messages import IPCMessage, MessageType


def test_message_serialization():
    """Test message JSON serialization"""
    msg = IPCMessage(
        version=1,
        msg_type=MessageType.REQUEST,
        msg_id=12345,
        source="test",
        target="broker",
        method="register",
        payload=b'{"service": "test"}',
        timestamp=1704931200000,
    )
    
    json_str = msg.to_json()
    parsed = json.loads(json_str)
    
    assert parsed["version"] == 1
    assert parsed["msg_type"] == "REQUEST"
    assert parsed["msg_id"] == 12345
    assert parsed["source"] == "test"
    assert parsed["target"] == "broker"
    assert parsed["method"] == "register"
    
    print("✓ Message serialization test passed")


def test_message_deserialization():
    """Test message JSON deserialization"""
    json_str = json.dumps({
        "version": 1,
        "msg_type": "RESPONSE",
        "msg_id": 12345,
        "source": "broker",
        "target": "test",
        "method": "register",
        "payload": '{"status": "ok"}',
        "timestamp": 1704931200000,
    })
    
    msg = IPCMessage.from_json(json_str)
    
    assert msg.version == 1
    assert msg.msg_type == MessageType.RESPONSE
    assert msg.msg_id == 12345
    assert msg.source == "broker"
    assert msg.target == "test"
    
    print("✓ Message deserialization test passed")


def test_message_bytes():
    """Test message byte encoding with length prefix"""
    msg = IPCMessage(
        version=1,
        msg_type=MessageType.REQUEST,
        msg_id=1,
        source="test",
        target="broker",
        method="ping",
        payload=b"{}",
    )
    
    data = msg.to_bytes()
    
    # First 4 bytes should be length
    length = struct.unpack(">I", data[:4])[0]
    payload = data[4:]
    
    assert length == len(payload)
    assert json.loads(payload.decode())["method"] == "ping"
    
    print("✓ Message bytes test passed")


class MockBroker:
    """Simple mock broker for testing"""
    
    def __init__(self, socket_path):
        self.socket_path = socket_path
        self.running = False
        self.server = None
        self.clients = []
        
    def start(self):
        """Start mock broker"""
        if os.path.exists(self.socket_path):
            os.unlink(self.socket_path)
            
        self.server = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self.server.bind(self.socket_path)
        self.server.listen(5)
        self.server.settimeout(1.0)
        self.running = True
        
        self.thread = threading.Thread(target=self._accept_loop)
        self.thread.start()
        
    def _accept_loop(self):
        while self.running:
            try:
                client, _ = self.server.accept()
                self.clients.append(client)
                threading.Thread(target=self._handle_client, args=(client,)).start()
            except socket.timeout:
                continue
            except Exception as e:
                if self.running:
                    print(f"Accept error: {e}")
                break
                
    def _handle_client(self, client):
        try:
            while self.running:
                # Read length
                len_data = client.recv(4)
                if not len_data:
                    break
                length = struct.unpack(">I", len_data)[0]
                
                # Read message
                msg_data = client.recv(length)
                msg = json.loads(msg_data.decode())
                
                # Send response
                response = {
                    "version": 1,
                    "msg_type": "RESPONSE",
                    "msg_id": msg.get("msg_id", 0),
                    "source": "broker",
                    "target": msg.get("source", ""),
                    "method": msg.get("method", ""),
                    "payload": '{"status": "ok"}',
                    "timestamp": int(time.time() * 1000),
                }
                
                resp_data = json.dumps(response).encode()
                client.send(struct.pack(">I", len(resp_data)) + resp_data)
                
        except Exception as e:
            if self.running:
                print(f"Client error: {e}")
        finally:
            client.close()
            
    def stop(self):
        self.running = False
        for client in self.clients:
            try:
                client.close()
            except:
                pass
        if self.server:
            self.server.close()
        if os.path.exists(self.socket_path):
            os.unlink(self.socket_path)
        self.thread.join(timeout=2.0)


def test_mock_broker():
    """Test communication with mock broker"""
    with tempfile.TemporaryDirectory() as tmpdir:
        socket_path = os.path.join(tmpdir, "test.sock")
        
        # Start mock broker
        broker = MockBroker(socket_path)
        broker.start()
        time.sleep(0.1)
        
        try:
            # Connect client
            client = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            client.connect(socket_path)
            
            # Send message
            msg = IPCMessage(
                msg_type=MessageType.REQUEST,
                msg_id=1,
                source="test",
                target="broker",
                method="register",
                payload=b'{"service": "test"}',
            )
            client.send(msg.to_bytes())
            
            # Read response
            len_data = client.recv(4)
            length = struct.unpack(">I", len_data)[0]
            resp_data = client.recv(length)
            response = json.loads(resp_data.decode())
            
            assert response["msg_type"] == "RESPONSE"
            assert response["msg_id"] == 1
            assert response["source"] == "broker"
            
            client.close()
            print("✓ Mock broker test passed")
            
        finally:
            broker.stop()


async def test_async_client():
    """Test async IPC client"""
    from ipc.client import IPCClient
    
    with tempfile.TemporaryDirectory() as tmpdir:
        socket_path = os.path.join(tmpdir, "test.sock")
        
        # Start mock broker
        broker = MockBroker(socket_path)
        broker.start()
        time.sleep(0.1)
        
        try:
            # Create client
            client = IPCClient("test", socket_path)
            
            # Connect (should succeed)
            connected = await client.connect(max_retries=3, retry_delay=0.1)
            assert connected, "Failed to connect"
            
            # Close
            await client.close()
            print("✓ Async client test passed")
            
        finally:
            broker.stop()


def run_tests():
    """Run all tests"""
    print("Running IPC tests...\n")
    
    test_message_serialization()
    test_message_deserialization()
    test_message_bytes()
    test_mock_broker()
    
    # Run async test
    asyncio.run(test_async_client())
    
    print("\n✓ All tests passed!")


if __name__ == "__main__":
    run_tests()
