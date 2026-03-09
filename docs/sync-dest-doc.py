import os
import yaml
import json
import re


# Helper functions

def indent_lines(str="", spaces=0):
    """
    Indent each line in a string by the specified number of spaces.

    Args:
        str (str): Input string to indent.
        spaces (int): Number of spaces to indent each line.

    Returns:
        str: Indented string.
    """
    indented = "\n".join(
        f"{' ' * spaces}{line}" if line.strip() else line for line in str.splitlines()
    )

    return indented


def replace_section(mdx_content, start_block, end_block, new_content, default_append_to_end):
    """
    Replace or update a section in the content between start_block and end_block.

    Args:
        mdx_content (str): Original content to modify.
        start_block (str): Start marker for the block.
        end_block (str): End marker for the block.
        new_content (str): New content to insert.
        default_append_to_end (bool): If true, append to the end; otherwise append to start.

    Returns:
        str: Modified content.
    """

    # Compile the regex pattern to find the section
    section_pattern = re.compile(
        rf"({re.escape(start_block)}[\s\S]+?)({re.escape(end_block)})",
        re.DOTALL
    )

    if section_pattern.search(mdx_content):
        # If the section is found, replace including the end block
        mdx_content = section_pattern.sub(new_content, mdx_content)
    else:
        # If the section is not found, append the entire section
        if default_append_to_end:
            # Append to the end
            mdx_content += f"{new_content}"
        else:
            # Append to the start
            mdx_content = f"{new_content}{mdx_content}"

    return mdx_content


# Generate functions
# (generate content from within YAML files)

def generate_logo(yaml_content, img_tag=False, img_size=4):
    """
    Function to generate the logo for the destination.

    Args:
        yaml_content (dict): Destination YAML content.
        img_tag (bool): If True, return the image tag; otherwise, return the markdown link.
        img_size (int): Size of the image (in Tailwind format).

    Returns:
        str: Logo content.
    """
    dest_type = yaml_content.get("metadata", {}).get("type", "")
    dest_image = yaml_content.get("spec", {}).get("image", "")

    if img_tag:
        return f"<img src=\"https://d15jtxgb40qetw.cloudfront.net/{dest_image}\" alt=\"{dest_type}\" className=\"not-prose h-{int(img_size)}\" />"

    return f"[![logo with clickable link](https://d15jtxgb40qetw.cloudfront.net/{dest_image})](https://www.google.com/search?q={dest_type})"


def generate_signals(yaml_content):
    """
    Function to generate the 'Supported Signals' section.
    It will generate a list of signals supported by the destination.

    Args:
        yaml_content (dict): Destination YAML content.

    Returns:
        str: Signals content.
    """
    signals = yaml_content.get("spec", {}).get("signals", {})
    with_traces = signals.get("traces", {}).get("supported", False)
    with_metrics = signals.get("metrics", {}).get("supported", False)
    with_logs = signals.get("logs", {}).get("supported", False)

    content = (
        "<Accordion title=\"Supported Signals:\">"
        + f"\n{indent_lines('✅' if with_traces else '❌', 2)} Traces"
        + f"\n{indent_lines('✅' if with_metrics else '❌', 2)} Metrics"
        + f"\n{indent_lines('✅' if with_logs else '❌', 2)} Logs"
        + "\n</Accordion>"
    )

    return content


def generate_note(yaml_content):
    """
    Function to generate the 'Check' note section.
    It will generate a note for the destination.

    Args:
        yaml_content (dict): Destination YAML content.

    Returns:
        str: Note content.
    """
    note = yaml_content.get("spec", {}).get("note", {})
    type = note.get("type", "Note")
    content = note.get("content", "")

    if content:
        note = f"<{type}>\n{indent_lines(content, 2)}\n</{type}>"
    else:
        note = ""

    return note


def generate_fields(yaml_content):
    """
    Function to generate the 'Configuring Destination Fields' section.
    It will generate a list of fields with their types and descriptions.

    Args:
        yaml_content (dict): Destination YAML content.

    Returns:
        str: Fields content.
    """
    yaml_fields = yaml_content.get("spec", {}).get("fields", [])
    fields = ""

    for f in yaml_fields:
        # !! Skipped field-values: Check "Allowed properties for Destination Fields" in "docs/adding-new-dest.mdx" for more details
        # secret, componentProps.values , customReadDataLabels, renderCondition, hideFromReadData,

        id = f.get("name", "")
        name = f.get("displayName", "")
        initial_value = f.get("initialValue", {})
        component_props = f.get("componentProps", {})
        tooltip = component_props.get("tooltip", "")
        placeholder = component_props.get("placeholder", "")
        is_required = component_props.get("required", False)

        type = "unknown"
        component_type = f.get("componentType", {})
        if component_type == "checkbox":
            type = "boolean"
        elif component_type == "multiInput":
            type = "string[]"
        elif component_type == "keyValuePairs":
            type = "{ key: string; value: string; }[]"
        elif component_type == "input" or component_type == "textarea" or component_type == "dropdown":
            input_type = component_props.get("type", False)
            if input_type == "number":
                type = "number"
            elif input_type == "password":
                type = "string"
            else:
                type = "string"

        field = (
            f"- **{id}** `{type}` : {name}."
            + (f" {tooltip}" if tooltip else "")
            + f"\n  - This field is {'required' if is_required else 'optional'}"
            + (f" and defaults to `{initial_value}`" if initial_value is not None and (
                initial_value or isinstance(initial_value, (bool, int))
            ) else "")
            + (f"\n  - Example: `{placeholder}`" if placeholder else "")
        )

        if fields:
            fields += "\n"
        fields += field

    return fields


def generate_kubectl_apply(yaml_content):
    """
    Function to generate the `kubectl apply` command for the destination.
    It will generate the YAML content for the destination and the secret (if required).

    Args:
        yaml_content (dict): Destination YAML content.

    Returns:
        str: `kubectl apply` command content.
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
        key = field.get("name", "")
        name = field.get('displayName', '')
        component_props = field.get('componentProps', {})
        required = component_props.get("required", False)
        initial_value = field.get("initialValue", {})

        # Get initial values
        if initial_value:
            name += f" (default: {initial_value})"

        # Get values for dropdowns
        if field.get('componentType', '') == 'dropdown':
            values = component_props.get('values', [])
            if values:
                name += f" (options: [{', '.join(values)}])"

        if field.get("secret", False):
            # Secret fields are added only to the Secret manifest
            has_secrets = True
            secret_yaml["data"][key] = f"<Base64 {name}>"
            if not required:
                secret_optional = True
        elif required:
            # Required fields are added directly to the Destination manifest
            required_fields[key] = f"<{name}>"
        else:
            # Prepare optional fields as commented lines
            optional_fields += f"\n    # {key}: <{name}>"

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
        # Between the 'destinationName' and the 'signals' blocks
        pointer_between = r"(\b(?:destinationName: .+))(?=\n\s*signals:)"
        # The existing 'secretRef' section
        pointer_is = r"\bsecretRef:\s*\n(.*?)(?=\n\s*\w+:|\n\s*signals:)"

        if secret_optional:
            # Inject optional 'secretRef' section between the 'destinationName' and the 'signals' blocks
            destination_yaml = re.sub(
                pointer_between,
                lambda match: f"{match.group(1)}"
                f"\n  # Uncomment the 'secretRef' below if you are using the optional Secret."
                f"\n  # secretRef:\n  #   name: {secret_name}",
                destination_yaml,
                flags=re.MULTILINE
            )
            # Comment out the entire secret if it's optional
            secret_yaml = f"# The following Secret is optional. Uncomment the entire block if you need to use it.\n" + \
                re.sub(r"^(.)", r"# \1", secret_yaml, flags=re.MULTILINE)

        elif "secretRef:" in destination_yaml:
            # Inject required 'secretRef' by updating the existing 'secretRef' section
            destination_yaml = re.compile(pointer_is, re.DOTALL).sub(
                f"  secretRef:\n    name: {secret_name}", destination_yaml
            )
        else:
            # Inject required 'secretRef' section between the 'destinationName' and the 'signals' blocks
            destination_yaml = re.compile(pointer_between, re.DOTALL).sub(
                r"\1\n  secretRef:\n    name: " + secret_name, destination_yaml
            )

        # Wrap the combined (destination + secret) YAML content in a single code-block
        code_block = f"```yaml\n{destination_yaml}\n---\n\n{secret_yaml}```"
    else:
        # Wrap the destination YAML content in a single code-block
        code_block = f"```yaml\n{destination_yaml}```"

    return code_block


# Get functions
# (gets generated content for MDX files)

def get_documenation(yaml_content):
    """
    Function to get the documentation content for the destination.

    Args:
        yaml_content (dict): Destination YAML content.

    Returns:
        dict: Documentation content
    """
    meta = yaml_content.get("metadata", {})
    type = meta.get("type", "")
    name = meta.get("displayName", "")
    category = meta.get("category", "")
    category = "Managed" if category == "managed" else "Self-Hosted"

    signals = generate_signals(yaml_content)
    fields = generate_fields(yaml_content)
    note = generate_note(yaml_content)

    start_before_custom = "---"
    content_before_custom = (
        f"{start_before_custom}"
        + f"\ntitle: '{name}'"
        + f"\ndescription: 'Configuring the {name} backend ({category})'"
        + f"\nsidebarTitle: '{name}'"
        + "\nicon: 'signal-stream'"
        + "\n---"
        + "\n\n### Getting Started"
        + f"\n\n{generate_logo(yaml_content, True, 20)}"
        + "\n\n{/*"
        + "\n    Add custom content here (under this comment)..."
        + "\n"
        + "\n    e.g."
        + "\n\n    **Creating Account**<br />"
        + "\n    Go to the **[🔗 website](https://odigos.io) > Account** and click **Sign Up**"
        + "\n\n    **Obtaining Access Token**<br />"
        + "\n    Go to **⚙️ > Access Tokens** and click **Create New**"
        + "\n"
        + "\n    !! Do not remove this comment, this acts as a key indicator in `docs/sync-dest-doc.py` !!"
        + "\n    !! START CUSTOM EDIT !!"
        + "\n*/}"
    )
    end_before_custom = content_before_custom[-100:]

    start_after_custom = (
        "{/*"
        + "\n    !! Do not remove this comment, this acts as a key indicator in `docs/sync-dest-doc.py` !!"
        + "\n    !! END CUSTOM EDIT !!"
        + "\n*/}"
    )
    signals_content = f"\n\n{signals}" if signals else ""
    fields_content = f"\n\n{fields}" if fields else ""
    note_content = f"\n\n{note}" if note else ""
    content_after_custom = (
        f"{start_after_custom}"
        + "\n\n### Configuring Destination Fields"
        + signals_content
        + fields_content
        + note_content
        + "\n\n### Adding Destination to Odigos"
        + "\n\nThere are two primary methods for configuring destinations in Odigos:"
        + "\n\n##### **Using the UI**"
        + "\n\n<Steps>"
        + "\n  <Step>"
        + "\n    Use the [Odigos CLI](https://docs.odigos.io/cli/odigos_ui) to access the UI"
        + "\n    ```bash"
        + "\n    odigos ui"
        + "\n    ```"
        + "\n  </Step>"
        + "\n  <Step>"
        + "\n    Click on `Add Destination`"
        + f", select `{name}` and follow the on-screen instructions"
        + "\n  </Step>"
        + "\n</Steps>"
        + "\n\n##### **Using Kubernetes manifests**"
        + "\n\n<Steps>"
        + "\n  <Step>"
        + f"\n    Save the YAML below to a file (e.g. `{type}.yaml`)"
        + f"\n{indent_lines(generate_kubectl_apply(yaml_content), 4)}"
        + "\n  </Step>"
        + "\n  <Step>"
        + "\n    Apply the YAML using `kubectl`"
        + "\n    ```bash"
        + f"\n    kubectl apply -f {type}.yaml"
        + "\n    ```"
        + "\n  </Step>"
        + "\n</Steps>"
    )
    end_after_custom = content_after_custom[-50:]

    return {
        "start_before_custom": start_before_custom,
        "content_before_custom": content_before_custom,
        "end_before_custom": end_before_custom,
        "start_after_custom": start_after_custom,
        "content_after_custom": content_after_custom,
        "end_after_custom": end_after_custom,
    }


# CRUD functions
# (apply generated content in MDX files)

def update_mdx(mdx_path, yaml_content):
    """
    Function to update the MDX file by replacing the existing content.

    Args:
        mdx_path (str): Path to the MDX file.
        yaml_content (dict): Destination YAML content.

    Returns:
        None
    """
    documenation = get_documenation(yaml_content)

    with open(mdx_path, 'r') as mdx_file:
        mdx_content = mdx_file.read()

    mdx_content = replace_section(
        mdx_content,
        documenation.get("start_before_custom"),
        documenation.get("end_before_custom"),
        documenation.get("content_before_custom"),
        False
    )
    mdx_content = replace_section(
        mdx_content,
        documenation.get("start_after_custom"),
        documenation.get("end_after_custom"),
        documenation.get("content_after_custom"),
        True
    )

    with open(mdx_path, 'w') as mdx_file:
        mdx_file.write(mdx_content)


def create_mdx(mdx_path, yaml_content):
    """
    Function to create a new MDX file with the generated content.

    Args:
        mdx_path (str): Path to the MDX file.
        yaml_content (dict): Destination YAML content.

    Returns:
        None
    """
    documenation = get_documenation(yaml_content)
    content_before = documenation.get("content_before_custom")
    content_after = documenation.get("content_after_custom")
    mdx_content = (
        f"{content_before}"
        + f"\n\n{content_after}"
    )

    with open(mdx_path, 'w') as mdx_file:
        mdx_file.write(mdx_content)


# Root


def process_files(backend_mdx_dir, backend_yaml_dir):
    """
    This function will generate or update the MDX files for each destination YAML.

    Args:
        backend_mdx_dir (str): Path to the MDX files directory.
        backend_yaml_dir (str): Path to the YAML files directory.

    Returns:
        None
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
    This function will generate the overview page with the list of backends.

    Args:
        backend_yaml_dir (str): Path to the YAML files directory.
        docs_dir (str): Path to the docs directory.

    Returns:
        None
    """
    overview_path = os.path.join(docs_dir, "snippets/shared/backends-overview.mdx")

    rows = []
    for root, _, files in os.walk(backend_yaml_dir):
        for file in sorted(files):
            if file.endswith('.yaml'):
                # Read the YAML file
                yaml_path = os.path.join(root, file)
                with open(yaml_path, 'r') as yaml_file:
                    yaml_content = yaml.safe_load(yaml_file)
                    meta = yaml_content.get("metadata", {})
                    name = meta.get("displayName", "")
                    category = meta.get("category", "")
                    signals = yaml_content.get("spec", {}).get("signals", {})

                    rows.append(
                        f"{generate_logo(yaml_content, True, 4)} | "
                        f"[{name}](/backends/{file.replace('.yaml', '')}) | "
                        f"{'Managed' if category == 'managed' else 'Self-Hosted'} | "
                        f"{'✅' if signals.get('traces', {}).get('supported', False) else ''} | "
                        f"{'✅' if signals.get('metrics', {}).get('supported', False) else ''} | "
                        f"{'✅' if signals.get('logs', {}).get('supported', False) else ''} |"
                    )

    content = (
        "<Tip>"
        + "\n  Can't find your backend in Odigos? Please tell us! We are constantly adding new integrations.<br />"
        + "\n  You can also follow our quick [add new destination](/adding-new-dest) guide and submit a PR."
        + "\n</Tip>"
        + "\n\n| | | | Traces | Metrics | Logs |"
        + "\n|---|---|---|:---:|:---:|:---:|"
        + "\n"
        + "\n".join(rows)
    )

    with open(overview_path, 'w') as file:
        file.write(content)


def process_mint(backend_mdx_dir, docs_dir):
    """
    This function will update the docs.json navigation to include the new backends.

    Args:
        backend_mdx_dir (str): Path to the MDX files directory.
        docs_dir (str): Path to the docs directory.

    Returns:
        None
    """
    docs_json_path = os.path.join(docs_dir, "docs.json")

    # Load the JSON file
    with open(docs_json_path, 'r') as file:
        docs_data = json.load(file)

    # Build sorted list of backend page names
    backend_pages = []
    for _, _, files in os.walk(backend_mdx_dir):
        for file in files:
            if file.endswith('.mdx'):
                backend_pages.append(file.replace('.mdx', ''))
    backend_pages = sorted(backend_pages)

    # Update backend pages for each K8s Agent tab (oss and enterprise)
    k8s_product = next((
        p for p in docs_data.get("navigation", {}).get("products", [])
        if p.get("product") == "K8s Agent"
    ), None)

    if k8s_product:
        for version in k8s_product.get("versions", []):
            for tab in version.get("tabs", []):
                tab_href = tab.get("href", "")
                prefix = f"{tab_href}/backends" if tab_href else "backends"
                for group in tab.get("groups", []):
                    if group.get("group") == "Destinations":
                        for page in group.get("pages", []):
                            if isinstance(page, dict) and page.get("group") == "Supported Backends":
                                page["pages"] = [f"{prefix}/{name}" for name in backend_pages]

    # Save the modified JSON back to the file
    with open(docs_json_path, 'w') as file:
        json.dump(docs_data, file, indent=2)


if __name__ == "__main__":
    backend_mdx_dir = "./snippets/shared/backends"
    backend_yaml_dir = "../destinations/data"
    docs_dir = "."

    process_files(backend_mdx_dir, backend_yaml_dir)
    process_overview(backend_yaml_dir, docs_dir)
    process_mint(backend_mdx_dir, docs_dir)
