import os
import yaml
import json
import re


# Helper functions


def indent_lines(str="", spaces=0):
    indented = "\n".join(
        f"{" " * spaces}{line}" if line.strip() else line for line in str.splitlines()
    )

    return indented


def replace_section(mdx_content, start_block, end_block, new_content, default_append_to_end, replace_end_block):
    """
    Replace or update a section in the content between start_block and end_block.

    Args:
        mdx_content (str): Original content to modify.
        start_block (str): Start marker for the block.
        end_block (str): End marker for the block.
        new_content (str): New content to insert.
        default_append_to_end (bool): If true, append to the end; otherwise append to start.
        replace_end_block (bool): If true, replace the end block; otherwise exclude it.

    Returns:
        str: Modified content.
    """

    # Compile the regex pattern to find the section
    section_pattern = re.compile(
        rf"({re.escape(start_block)}[\s\S]+?)({re.escape(end_block)})",
        re.DOTALL
    )

    if section_pattern.search(mdx_content):
        # If the section is found, determine replacement logic
        if replace_end_block:
            # Replace including the end block
            mdx_content = section_pattern.sub(new_content, mdx_content)
        else:
            # Replace the content excluding the end block
            mdx_content = section_pattern.sub(
                lambda m: f"{new_content}{m.group(2)}", mdx_content
            )
    else:
        # If the section is not found, append the entire section
        if default_append_to_end:
            # Append to the end
            mdx_content += f"\n\n{new_content}"
        else:
            # Append to the start
            mdx_content = f"{new_content}{mdx_content}"

    return mdx_content


# Generate functions
# (generate content from within YAML files)


def generate_fields(yaml_content):
    yaml_fields = yaml_content.get("spec", {}).get("fields", [])
    fields = ""

    for f in yaml_fields:
        id = f.get("name", "")
        fcp = f.get("componentProps", {})
        display_name = f.get("displayName", "")
        is_secret = f.get("secret", False)
        is_required = fcp.get("required", False)
        tooltip = fcp.get("tooltip", "")
        placeholder = fcp.get("placeholder", "")
        initial_value = f.get("initialValue", {})

        # !! skipped fields:
        # componentType, componentProps.type, customReadDataLabels, renderCondition, hideFromReadData,

        field = (
            f"- **{id}** - {display_name}"
            + (f", {tooltip}" if tooltip else "")
            + "."
            + f"\n  - This field is {'required' if is_required else 'optional'}"
            + (f" and defaults to `{initial_value}`" if initial_value else "")
            + (f"\n  - Example: `{placeholder}`" if placeholder else "")
            + (f"\n  - Secured Secret 🔑" if is_secret else "")
        )

        if fields:
            fields += "\n"
        fields += field

    return fields


def generate_kubectl_apply(yaml_content):
    """
    Function to create the 'Using Kubernetes manifests' section.
    It will generate and include the destination YAML in a code-block.
    It will also generate a secret YAML and include it in the same code-block.
    """
    destination_type = yaml_content.get("metadata", {}).get("type", "").lower()

    destination_yaml = {
        "apiVersion": "odigos.io/v1alpha1",
        "kind": "Destination",
        "metadata": {
            "namespace": "odigos-system",
            "name": f"{destination_type}-example",
        },
        "spec": {
            "data": {},  # This will hold the required fields directly
            "destinationName": destination_type,
            "signals": [],
            "type": yaml_content.get("metadata", {}).get("type", ""),
        }
    }

    secret_yaml = {
        "apiVersion": "v1",
        "kind": "Secret",
        "metadata": {
            "namespace": destination_yaml["metadata"]["namespace"],
            "name": f"{destination_type}-secret",
        },
        "type": "Opaque",
        "data": {}
    }

    # Handle signals
    signals = yaml_content.get("spec", {}).get("signals", {})
    if signals.get("traces", {}).get("supported", False):
        destination_yaml["spec"]["signals"].append("TRACES")
    if signals.get("metrics", {}).get("supported", False):
        destination_yaml["spec"]["signals"].append("METRICS")
    if signals.get("logs", {}).get("supported", False):
        destination_yaml["spec"]["signals"].append("LOGS")

    # Separate required, optional, and secret fields
    required_fields = {}
    optional_fields = ""
    has_secrets = False
    secret_optional = False

    # Handle fields
    for field in yaml_content.get("spec", {}).get("fields", []):
        config_name = field.get("name", "")
        # Get field display name
        config_display = field.get('displayName', '')
        # Get values for dropdowns
        if field.get('componentType', '') == 'dropdown':
            values = field.get('componentProps', {}).get('values', [])
            if values:
                config_display += f" [{', '.join(values)}]"

        if field.get("secret", False):
            # Secret fields are added only to the Secret manifest
            has_secrets = True
            secret_yaml["data"][config_name] = f"<Base64 {config_display}>"
            if not field.get("componentProps", {}).get("required", False):
                secret_optional = True
        elif field.get("componentProps", {}).get("required", False):
            # Required fields are added directly to the Destination manifest
            required_fields[config_name] = f"<{config_display}>"
        else:
            # Prepare optional fields as commented lines
            optional_fields += f"\n    # {config_name}: <{config_display}>"

    # Add required fields to the 'data' section
    destination_yaml["spec"]["data"].update(required_fields)

    # Convert the YAML to a string
    destination_yaml = yaml.dump(destination_yaml, default_flow_style=False)

    # Inject the optional fields within the 'data' section
    if optional_fields:
        destination_yaml = re.compile(
            # Capture destinationName in group(2)
            r"(\bdata:\s*\n(?:[ \t]+.+\n)*)(\s*destinationName: .+)",
            re.DOTALL
        ).sub(
            lambda match: match.group(1) +  # Entire data block
            "    # Note: The commented fields below are optional." +
            optional_fields + "\n" +  # Append optional fields
            match.group(2),  # Preserve destinationName
            destination_yaml
        )

    # Handle secrets
    if has_secrets:
        secret_name = secret_yaml["metadata"]["name"]
        # Convert the YAML to a string
        secret_yaml = yaml.dump(secret_yaml, default_flow_style=False)

        if secret_optional:
            # Inject optional 'secretRef' section right after 'destinationName'
            destination_yaml = re.sub(
                # Match the entire destinationName line
                r"^(destinationName: [^\n]+)$",
                lambda match: f"{match.group(1)}"
                f"\n  # Uncomment the 'secretRef' below if you are using the optional Secret."
                f"\n  # secretRef:\n  #   name: {secret_name}",
                destination_yaml,
                flags=re.MULTILINE
            )
            # Comment out the entire secret if it's optional
            secret_yaml = f"# The following Secret is optional. Uncomment the entire block if you need to use it.\n" + \
                re.sub(
                    r"^(.)", r"# \1", secret_yaml, flags=re.MULTILINE
                )
        elif "secretRef:" in destination_yaml:
            # Inject required 'secretRef' by updating the existing 'secretRef' section
            destination_yaml = re.compile(
                r"\bsecretRef:\s*\n(.*?)(?=\n\s*\w+:|\n\s*signals:)", re.DOTALL
            ).sub(
                f"  secretRef:\n    name: {secret_name}", destination_yaml
            )
        else:
            # Inject required 'secretRef' section between the 'destinationName' and the 'signals' blocks
            destination_yaml = re.compile(
                r"(\b(?:destinationName: .+))(?=\n\s*signals:)", re.DOTALL
            ).sub(
                r"\1\n  secretRef:\n    name: " + secret_name, destination_yaml
            )

        # Wrap the combined (destination + secret) YAML content in a single code-block
        code_block = f"```yaml\n{
            destination_yaml
        }\n---\n\n{
            secret_yaml
        }```"
    else:
        # Wrap the destination YAML content in a single code-block
        code_block = f"```yaml\n{
            destination_yaml
        }```"

    return code_block


# Get functions
# (gets generated content for MDX files)


def get_logo(yaml_content, img_tag=False):
    dest_type = yaml_content.get("metadata", {}).get("type", "")
    dest_image = yaml_content.get("spec", {}).get("image", "")

    if img_tag:
        return f"<img src='https://d15jtxgb40qetw.cloudfront.net/{dest_image}' alt='{dest_type}' width=\"18\" height=\"18\" className=\"not-prose\" />"

    return f"[![logo with clickable link](https://d15jtxgb40qetw.cloudfront.net/{dest_image})](https://www.google.com/search?q={dest_type})"


def get_header(yaml_content):
    """
    Function to get the header of the MDX file.
    Variables 'starting_block' and 'closing_block' will be used to dynamically identify the start-to-end of the block, in case of an update.
    """
    dest_name = yaml_content.get("metadata", {}).get("displayName", "")

    starting_block = "---"
    content_block = (
        f"{starting_block}"
        + f"\ntitle: '{dest_name}'"
        + f"\ndescription: 'Configuring the {dest_name} Backend'"
        + f"\nsidebarTitle: '{dest_name}'"
        + "\nicon: 'signal-stream'"
        + "\n---"
        + "\n\n### Getting Started"
        # Just to get started somewhere, this row needs to be edited manually after 1st-time-creation
        + "\n\n{/*"
        + "\n    Add custom content here (under this comment)..."
        + "\n"
        + "\n    e.g:"
        + "\n    [🔗 website](https://odigos.io)"
        + f"\n    {get_logo(yaml_content)}"
        + "\n"
        + "\n    !! Do not remove this comment, this acts as a key indicator in `docs/sync-dest-doc.py` !!"
        + "\n    !! START CUSTOM EDIT !!"
        + "\n*/}"
    )
    closing_block = content_block[-100:]

    return starting_block, closing_block, content_block


def get_config_fields_section(yaml_content):
    """
    Function to get the 'Configuring Destination Fields' section.
    Variables 'starting_block' and 'closing_block' will be used to dynamically identify the start-to-end of the block, in case of an update.
    """
    starting_block = (
        "{/*"
        + "\n    !! Do not remove this comment, this acts as a key indicator in `docs/sync-dest-doc.py` !!"
        + "\n    !! END CUSTOM EDIT !!"
        + "\n*/}"
    )
    content_block = (
        f"{starting_block}"
        + "\n\n### Configuring Destination Fields"
        + f"\n\n{generate_fields(yaml_content)}"
    )
    # The limit is `starting_block` from `get_add_dest_section`, we must ensure to not replace this closing block.
    closing_block = "\n\n### Adding Destination to Odigos"

    return starting_block, closing_block, content_block


def get_add_dest_section(yaml_content):
    """
    Function to get the 'Adding Destination' section.
    Variables 'starting_block' and 'closing_block' will be used to dynamically identify the start-to-end of the block, in case of an update.
    """
    yaml_meta = yaml_content.get("metadata", {})
    dest_type = yaml_meta.get("type", "")
    dest_name = yaml_meta.get("displayName", "")

    starting_block = "### Adding Destination to Odigos"
    content_block = (
        f"{starting_block}"
        + "\n\nThere are two primary methods for configuring destinations in Odigos:"
        + "\n\n##### **Using the UI**"
        + "\n\n1. Use the [Odigos CLI](https://docs.odigos.io/cli/odigos_ui) to access the UI"
        + "\n\n  ```bash"
        + "\n  odigos ui"
        + "\n  ```"
        + "\n\n2. Click on `Add Destination`"
        + "\n3. Select "
        + f"`{dest_name}`"
        + " and follow the on-screen instructions"
        + "\n\n##### **Using Kubernetes manifests**"
        + f"\n\n1. Save the YAML below to a file (e.g. `{dest_type}.yaml`)"
        + f"\n\n{indent_lines(generate_kubectl_apply(yaml_content), 2)}"
        + "\n\n2. Apply the YAML using `kubectl`:"
        + "\n\n  ```bash"
        + f"\n  kubectl apply -f {dest_type}.yaml"
        + "\n  ```"
    )
    # 50 is the limit, any further and we might overlap dynamically changing values, and that cannot be used as a closing block reference.
    closing_block = content_block[-50:]

    return starting_block, closing_block, content_block


# CRUD functions
# (apply generated content in MDX files)


def update_mdx(mdx_path, yaml_content):
    """
    Function to update the MDX file by replacing the existing section with the updated content.
    Note: we do not update the 'Getting Started' section. This is meant to be created only once, and give the developer a starting point for typing-out custom guidelines.
    """
    with open(mdx_path, 'r') as mdx_file:
        mdx_content = mdx_file.read()

    mdx_content = replace_section(
        mdx_content, *get_header(yaml_content), False, True
    )
    mdx_content = replace_section(
        mdx_content, *get_config_fields_section(yaml_content), True, False
    )
    mdx_content = replace_section(
        mdx_content, *get_add_dest_section(yaml_content), True, True
    )

    with open(mdx_path, 'w') as mdx_file:
        mdx_file.write(mdx_content)


def create_mdx(mdx_path, yaml_content):
    """
    Function to create the MDX file by appending the newly generated content.
    """
    dest_name = yaml_content.get("metadata", {}).get("displayName", "")

    _, _, header = get_header(yaml_content)
    _, _, config_dest = get_config_fields_section(yaml_content)
    _, _, add_dest = get_add_dest_section(yaml_content)

    mdx_content = (
        f"{header}"
        # Logo only on-create
        + f"\n\n{get_logo(yaml_content)}"
        + f"\n\n{config_dest}"
        + f"\n\n{add_dest}"
    )

    with open(mdx_path, 'w') as mdx_file:
        mdx_file.write(mdx_content)


# Root


def process_files(backend_mdx_dir, backend_yaml_dir):
    """
    Main function to process the .yaml files, and create/update relative .mdx files.
    """
    for root, _, files in os.walk(backend_yaml_dir):
        for file in files:
            if file.endswith('.yaml'):
                # Read the YAML file
                yaml_path = os.path.join(root, file)
                with open(yaml_path, 'r') as yaml_file:
                    yaml_content = yaml.safe_load(yaml_file)

                # Generate or update the MDX file
                mdx_path = os.path.join(
                    backend_mdx_dir, file.replace('.yaml', '.mdx')
                )
                if os.path.exists(mdx_path):
                    update_mdx(mdx_path, yaml_content)
                else:
                    create_mdx(mdx_path, yaml_content)


def process_overview(backend_yaml_dir, docs_dir):
    """
    This function will generate the overview.md file with the destinations.
    """
    overview_path = os.path.join(docs_dir, "backends-overview.mdx")

    rows = []
    for root, _, files in os.walk(backend_yaml_dir):
        for file in sorted(files):
            if file.endswith('.yaml'):
                # Read the YAML file
                yaml_path = os.path.join(root, file)
                with open(yaml_path, 'r') as yaml_file:
                    yaml_content = yaml.safe_load(yaml_file)
                    meta = yaml_content.get("metadata", {})
                    type = meta.get("type", "")
                    name = meta.get("displayName", "")
                    category = meta.get("category", "")
                    signals = yaml_content.get("spec", {}).get("signals", {})

                    rows.append(f"{
                        get_logo(yaml_content, True)
                    } | [{name}](/backends/{type}) | {
                        "Managed" if category == "managed" else "Self-Hosted"
                    } | {
                        '✅' if signals.get("traces", {}).get(
                            "supported", False) else ''
                    } | {
                        '✅' if signals.get("metrics", {}).get(
                            "supported", False) else ''
                    } | {
                        '✅' if signals.get("logs", {}).get(
                            "supported", False) else ''
                    } |"
                    )

    content = (
        "---"
        + "\ntitle: 'Overview'"
        + "\n---"
        + "\n\nOdigos makes it simple to add and configure destinations, allowing you to select the specific signals (`traces`,`metrics`,`logs`) that you want to send to each destination."
        + "\n\nOdigos has destinations for many observability backends."
        + "\n\n| Logo | Destination | Category | Traces | Metrics | Logs |"
        + "\n|---|---|---|:---:|:---:|:---:|"
        + "\n"
        + "\n".join(rows)
        + "\n\nCan't find the destination you need? Help us by following our quick [adding new destination](/adding-new-dest) guide and submit a PR."
    )

    with open(overview_path, 'w') as file:
        file.write(content)


def process_mint(backend_mdx_dir, docs_dir):
    """
    This function will update the mint.json file with the new backends.
    """
    mint_path = os.path.join(docs_dir, "mint.json")

    # Load the JSON file
    with open(mint_path, 'r') as file:
        mint_data = json.load(file)

    # Locate the "Destinations" group within "navigation"
    destinations_group = next((
        nav for nav in mint_data.get("navigation", [])
        if nav.get("group") == "Destinations"
    ), None)

    # Locate the group with "Supported Backends" within "pages"
    supported_backends_group = next((
        page for page in destinations_group.get("pages", [])
        if isinstance(page, dict) and page.get("group") == "Supported Backends"
    ), None)

    # Replace the "pages" array with new backend paths
    mint_pages = []
    for _, _, files in os.walk(backend_mdx_dir):
        for file in files:
            if file.endswith('.mdx'):
                mint_pages.append(f"backends/{file.replace('.mdx', '')}")
    supported_backends_group["pages"] = sorted(mint_pages)

    # Save the modified JSON back to the file
    with open(mint_path, 'w') as file:
        json.dump(mint_data, file, indent=2)


if __name__ == "__main__":
    backend_mdx_dir = "./backends"
    backend_yaml_dir = "../destinations/data"
    docs_dir = "."

    process_files(backend_mdx_dir, backend_yaml_dir)
    process_overview(backend_yaml_dir, docs_dir)
    process_mint(backend_mdx_dir, docs_dir)
