import os
import yaml
import re

def get_display_text(field):
    """Helper function to generate display text, including dropdown values if present."""
    display_name = field.get('displayName', '')
    if field.get('componentType', '') == 'dropdown':
        values = field.get('componentProps', {}).get('values', [])
        if values:
            display_name += f" [{', '.join(values)}]"
    return display_name

def shorten_yaml(yaml_content):
    """
    Function to generate a shortened YAML structure in the Kubernetes manifest format based on the provided content.
    Also generates a Secret manifest for fields marked as secret.
    """
    destination_name = yaml_content.get("metadata", {}).get("type", "").lower()
    destination = {
        "apiVersion": "odigos.io/v1alpha1",
        "kind": "Destination",
        "metadata": {
            "name": f"{destination_name}-example",  # Adding '-example' suffix to the name
            "namespace": "<odigos namespace>"
        },
        "spec": {
            "data": {},  # This will hold the required fields directly
            "destinationName": destination_name,
            "signals": [],
            "type": yaml_content.get("metadata", {}).get("type", "")
        }
    }

    secret_manifest = {
        "apiVersion": "v1",
        "kind": "Secret",
        "metadata": {
            "name": f"{destination_name}-secret",
            "namespace": "<odigos namespace>"  # Updated default namespace
        },
        "type": "Opaque",
        "data": {}
    }

    # Separate required, optional, and secret fields
    fields = yaml_content.get("spec", {}).get("fields", [])
    required_fields = {}
    optional_fields = []
    has_secrets = False
    secret_optional = False

    for field in fields:
        config_name = field.get("name", "")
        config_display = get_display_text(field)
        if field.get("secret", False):
            # Secret fields are added only to the Secret manifest
            secret_manifest["data"][config_name] = f"<base64 {config_display}>"
            has_secrets = True
            if not field.get("componentProps", {}).get("required", False):
                secret_optional = True
        elif field.get("componentProps", {}).get("required", False):
            required_fields[config_name] = f"<{config_display}>"
        else:
            optional_fields.append(f"    # {config_name}: <{config_display}>")

    # Add required fields to the 'data' section
    destination["spec"]["data"].update(required_fields)

    # Handle signals
    signals = yaml_content.get("spec", {}).get("signals", {})
    if signals.get("traces", {}).get("supported", False):
        destination["spec"]["signals"].append("TRACES")
    if signals.get("metrics", {}).get("supported", False):
        destination["spec"]["signals"].append("METRICS")
    if signals.get("logs", {}).get("supported", False):
        destination["spec"]["signals"].append("LOGS")

    # Handle optional secrets
    secret_ref_section = ""
    if has_secrets:
        if secret_optional:
            secret_ref_section = "  # Uncomment the secretRef below if you are using the optional Secret.\n"
            secret_ref_section += f"  # secretRef:\n  #   name: {secret_manifest['metadata']['name']}\n"
        else:
            destination["spec"]["secretRef"] = {
                "name": secret_manifest["metadata"]["name"]
            }

    # Prepare optional fields as commented lines under the 'data' section
    optional_section = "\n".join(optional_fields)
    if optional_fields:
        optional_section += "\n    # Note: The commented fields above are optional."

    return destination, optional_section, secret_manifest if has_secrets else None, secret_ref_section

def update_mdx_with_yaml(mdx_path, shortened_yaml, optional_section, secret_manifest, secret_ref_section):
    """
    Function to update the MDX file by adding the shortened YAML under the '## Deploying using yaml' section.
    If a Secret manifest is generated, it is also added within the same yaml code block.
    """
    instructions = (
        "\n\n### Applying the Configuration\n"
        "Save the below YAML to a file (e.g., `destination.yaml`) and apply it using kubectl:\n\n"
        "```bash\n"
        "kubectl apply -f destination.yaml\n"
        "```\n"
    )

    yaml_section_title = "## Deploying using yaml"
    new_yaml_content = yaml.dump(shortened_yaml, default_flow_style=False)

    # Inject the optional fields under the 'data' section
    if optional_section:
        data_section_pattern = re.compile(r"\bdata:\s*\n(.*?)(?=\n\s*\w+:|\n\s*destinationName:)", re.DOTALL)
        new_yaml_content = data_section_pattern.sub(r"\g<0>\n" + optional_section, new_yaml_content)

    # Add secretRef section correctly if it exists
    if secret_ref_section:
        # Ensure secretRef is added right after destinationName
        new_yaml_content = re.sub(r"(destinationName: .+)", r"\1\n" + secret_ref_section, new_yaml_content)

    # If a secret manifest exists, add it to the same yaml code block
    if secret_manifest:
        secret_yaml_content = yaml.dump(secret_manifest, default_flow_style=False)
        if secret_ref_section:
            # Comment out the entire secret if it's optional
            secret_yaml_content = re.sub(r"^(.)", r"# \1", secret_yaml_content, flags=re.MULTILINE)
            secret_yaml_content = f"# The following Secret is optional. Uncomment the entire block if you need to use it.\n{secret_yaml_content}"
        new_yaml_content += f"\n---\n{secret_yaml_content}"

    # Wrap the combined YAML content in a single yaml code block
    code_block = f"```yaml\n{new_yaml_content}```"

    # Construct the final content to insert
    final_content = f"{yaml_section_title}{instructions}\n\n{code_block}"

    with open(mdx_path, 'r') as mdx_file:
        mdx_content = mdx_file.read()

    # Find the existing section and replace it with the updated content
    section_pattern = re.compile(rf"({yaml_section_title}\n\n### Applying the Configuration[\s\S]*?)```yaml\n[\s\S]*?```", re.DOTALL)
    if section_pattern.search(mdx_content):
        mdx_content = section_pattern.sub(final_content, mdx_content)
    else:
        # If the section is not found, append the entire content
        mdx_content += f"\n\n{final_content}"

    with open(mdx_path, 'w') as mdx_file:
        mdx_file.write(mdx_content)

def process_files(docs_dir, backends_dir):
    """
    Main function to process the files in the directories.
    """
    for root, dirs, files in os.walk(docs_dir):
        for file in files:
            if file.endswith('.mdx'):
                mdx_path = os.path.join(root, file)
                yaml_path = os.path.join(backends_dir, file.replace('.mdx', '.yaml'))

                if os.path.exists(yaml_path):
                    with open(yaml_path, 'r') as yaml_file:
                        yaml_content = yaml.safe_load(yaml_file)
                    
                    shortened_yaml, optional_section, secret_manifest, secret_ref_section = shorten_yaml(yaml_content)
                    update_mdx_with_yaml(mdx_path, shortened_yaml, optional_section, secret_manifest, secret_ref_section)

if __name__ == "__main__":
    docs_dir = "./backends"
    backends_dir = "../destinations/data"
    
    process_files(docs_dir, backends_dir)
