# myapp_core/views.py
import random
from django.http import HttpResponse
from .models import ExampleModel
import logging
from packaging.version import parse, Version
import importlib_metadata
import asyncio
from asgiref.sync import async_to_sync

import sys


# Import the protobuf message based on the Python version
# In our min version test (3.6) protobuf 3.19.6 is used 
if sys.version_info < (3, 8):
    from .proto import example_pb2_legacy as example_pb2
else:
    from .proto import example_pb2

logger = logging.getLogger()


async def async_greeting():
    await asyncio.sleep(1)
    return "Hello from async!"

# Create a view to insert a random row
def insert_random_row(request):
    try:
        # Uses some 3rd packages that conflict with odigosodigos-opentelemetry-python
        # These few lines wont have business logic, but just to try to reproduce the conflicts
        ### importlib_metadata
        package_name = "Django"
        metadata = importlib_metadata.metadata(package_name)
        django_version = metadata['Version']
        logger.info(f"Using Django version: {django_version}")
        ### packaging
        # Version comparison using packaging
        v1 = parse("3.1.4")
        v2 = parse(django_version)
        if v1 < v2:
            version_comparison_result = f"v1 ({v1}) is older than Django version ({v2})"
        else:
            version_comparison_result = f"v1 ({v1}) is newer than Django version ({v2})"
        logger.info(version_comparison_result)   
        ### asgiref
        greeting = async_to_sync(async_greeting)()
        logger.info(f"Async greeting: {greeting}")        
        
        # Create a random name
        random_name = f"RandomName{random.randint(1, 1000)}"
        
        # Create a new entry in the database
        new_entry = ExampleModel.objects.create(name=random_name)
        
        # Create a Protobuf message
        message = example_pb2.ExampleMessage(
            name=new_entry.name,
            id=new_entry.id,
            created_at=str(new_entry.created_at)
        )

        # Serialize the message to binary format
        serialized_message = message.SerializeToString()

        # You can now store or send this serialized message
        logger.info(f"Serialized protobuf message: {serialized_message}")
        
        return HttpResponse(f"Inserted random row and serialized protobuf: {new_entry.name}")
    except Exception as e:
        logger.error(f"Error inserting random row: {e}")
        return HttpResponse(f"Error inserting random row: {e}", status=500)


# Create a view to fetch all rows
def fetch_all_rows(request):
    try:
        all_entries = ExampleModel.objects.all()

        # Deserialize protobuf data from each entry
        entries_list = []
        for entry in all_entries:
            message = example_pb2.ExampleMessage(
                name=entry.name,
                id=entry.id,
                created_at=str(entry.created_at)
            )
            serialized_message = message.SerializeToString()
            
            # Deserialize the message back to the object
            deserialized_message = example_pb2.ExampleMessage()
            deserialized_message.ParseFromString(serialized_message)

            entries_list.append(f"{deserialized_message.name} (ID: {deserialized_message.id})")
        
        logger.info(f"Fetched and deserialized rows: {entries_list}")
        return HttpResponse(f"All rows: {', '.join(entries_list)}")
    except Exception as e:
        logger.error(f"Error fetching rows: {e}")
        return HttpResponse(f"Error fetching rows: {e}", status=500)