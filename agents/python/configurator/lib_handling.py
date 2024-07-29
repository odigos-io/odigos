import sys

def reorder_python_path():
    paths_to_move = [path for path in sys.path if path.startswith('/var/odigos/python')]
    
    for path in paths_to_move:
        sys.path.remove(path)
        sys.path.append(path)
    
    
def reload_distro_modules() -> None:
    # Delete distro modules and their sub-modules, as they have been imported before the path was reordered.
    # The distro modules will be re-imported from the new path.
    needed_module_prefixes = [
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
    
    for module in list(sys.modules):
        # Check if the module starts with any of the needed prefixes
        if any(module.startswith(prefix) for prefix in needed_module_prefixes):
            del sys.modules[module]    
