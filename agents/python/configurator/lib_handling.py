import sys
import importlib
from importlib import metadata as md

def reorder_python_path():
    paths_to_move = [path for path in sys.path if path.startswith('/var/odigos/python')]
    
    for path in paths_to_move:
        sys.path.remove(path)
        sys.path.append(path)
    
    
def reload_distro_modules() -> None:
    # Reload distro modules, as they may have been imported before the path was reordered.
    # Add any new distro modules to this list.
    needed_modules = [
        'google.protobuf',
        'requests',
        'charset_normalizer',
        'certifi',
        'asgiref'
        'idna',
        'deprecated',
        'importlib_metadata',
        'packaging',
        'psutil',
        'zipp',
        'urllib3',
        'uuid_extensions.uuid7',
        'typing_extensions',
    ]

    for module_name in needed_modules:
        if module_name in sys.modules:
            module = sys.modules[module_name]
            importlib.reload(module)
