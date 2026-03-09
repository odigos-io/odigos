import os
import re
import time
import requests
from packaging.version import Version


"""
This script is used to sync the instrumentation documentation with the latest versions of the dependencies.
!! Currently it only supports Node.js (native) instrumentation libraries.
TODO: add support for other languages and types of instrumentation libraries.
"""


instrumentations_dir = './snippets/shared/instrumentations'

api_dependency_key = '@opentelemetry/api'
instrumentation_dependency_prefix = ['@opentelemetry/instrumentation-', '@odigos/instrumentation-']

uncategorized_key = 'Other'
documentation_starting_block = '## Instrumentation Libraries\n\nThe following npm packages will be auto instrumented by Odigos:'
documentation_ending_block = '\n{/* END OF FILE */}'

supported_languages = {
    'nodejs': {
        'native': {
            'package_json_url': 'https://raw.githubusercontent.com/odigos-io/opentelemetry-node/refs/heads/main/package.json',
            'categories': {
                'Node.js Core Modules': {
                    'description': '',
                    'items': [
                        {'dependency': 'node:dns', 'note': '', 'mdx': ''},
                        {'dependency': 'node:fs', 'note': '', 'mdx': ''},
                        {'dependency': 'node:http', 'note': '', 'mdx': ''},
                        {'dependency': 'node:https', 'note': '', 'mdx': ''},
                        {'dependency': 'node:net', 'note': '', 'mdx': ''},
                    ]
                },
                'HTTP Frameworks': {
                    'description': '',
                    'items': [
                        {'dependency': 'connect', 'note': '', 'mdx': ''},
                        {'dependency': 'express', 'note': '', 'mdx': ''},
                        {'dependency': 'fastify', 'note': '', 'mdx': ''},
                        {'dependency': 'graphql', 'note': '', 'mdx': ''},
                        {'dependency': '@hapi/hapi', 'note': '', 'mdx': ''},
                        {'dependency': 'koa', 'note': '', 'mdx': ''},
                        {'dependency': '@koa/router', 'note': '', 'mdx': ''},
                        {'dependency': '@nestjs/core', 'note': '', 'mdx': ''},
                        {'dependency': 'node:http', 'note': '', 'mdx': ''},
                        {'dependency': 'node:https', 'note': '', 'mdx': ''},
                        {'dependency': 'restify', 'note': '', 'mdx': ''},
                        {'dependency': 'router', 'note': '', 'mdx': ''},
                    ]
                },
                'HTTP Clients': {
                    'description': '',
                    'items': [
                        {'dependency': 'node:http', 'note': '', 'mdx': ''},
                        {'dependency': 'node:https', 'note': '', 'mdx': ''},
                        {'dependency': 'undici', 'note': '', 'mdx': ''},
                    ]
                },
                'RPC (Remote Procedure Call)': {
                    'description': '',
                    'items': [
                        {'dependency': 'aws-sdk', 'note': '', 'mdx': ''},
                        {'dependency': '@aws-sdk/client-*', 'note': '', 'mdx': ''},
                        {'dependency': '@grpc/grpc-js', 'note': '', 'mdx': ''},
                    ]
                },
                'Messaging Systems Clients': {
                    'description': '',
                    'items': [
                        {'dependency': 'amqplib', 'note': '', 'mdx': ''},
                        {'dependency': 'kafkajs', 'note': '', 'mdx': ''},
                        {'dependency': 'node:http', 'note': '', 'mdx': ''},
                        {'dependency': 'node:https', 'note': '', 'mdx': ''},
                        {'dependency': 'socket.io', 'note': '', 'mdx': ''},
                    ]
                },
                'Database Clients, ORMs, and data access libraries': {
                    'description': '',
                    'items': [
                        {'dependency': 'aws-sdk', 'note': '', 'mdx': ''},
                        {'dependency': '@aws-sdk/client-*', 'note': '', 'mdx': ''},
                        {'dependency': 'cassandra-driver', 'note': '', 'mdx': ''},
                        {'dependency': 'dataloader', 'note': '', 'mdx': ''},
                        {'dependency': 'generic-pool', 'note': '', 'mdx': ''},
                        {'dependency': 'ioredis', 'note': '', 'mdx': ''},
                        {'dependency': 'knex', 'note': '', 'mdx': ''},
                        {'dependency': 'lru-memoizer', 'note': '', 'mdx': ''},
                        {'dependency': 'memcached', 'note': '', 'mdx': ''},
                        {'dependency': 'mongodb', 'note': '', 'mdx': ''},
                        {'dependency': 'mongoose', 'note': '', 'mdx': ''},
                        {'dependency': 'mysql', 'note': '', 'mdx': ''},
                        {'dependency': 'mysql2', 'note': '', 'mdx': ''},
                        {'dependency': 'pg-pool', 'note': '', 'mdx': ''},
                        {'dependency': 'pg', 'note': '', 'mdx': ''},
                        {'dependency': 'redis', 'note': '', 'mdx': ''},
                        {'dependency': 'tedious', 'note': '', 'mdx': ''},
                    ]
                },
                'Loggers': {
                    'description': 'Automatic injection of trace context (trace id and span id) into log records for the following loggers:',
                    'items': [
                        {'dependency': 'bunyan', 'note': '', 'mdx': ''},
                        {'dependency': 'pino', 'note': '', 'mdx': ''},
                        {'dependency': 'winston', 'note': '', 'mdx': ''},
                    ]
                },
                uncategorized_key: {
                    'description': '',
                    'items': []
                },
            },
        },
        # 'ebpf': {
        #     'package_json_url': 'https://raw.githubusercontent.com/odigos-io/ebpf-nodejs-instrumentation/refs/heads/main/dtrace-injector/package.json',
        #     'categories': {}
        # }
    },
    'python': {
        # 'native': '',
    },
    'golang': {
        # 'ebpf': '',
    },
    'java': {
        # 'native': '',
        # 'ebpf': '',
    },
    'dotnet': {
        # 'native': '',
    }
}


def fetch(url, retry_url=None):
    """
    Fetch the content of a URL

    :param url: The URL to fetch
    :param retry_url: The URL to retry fetching if the first one fails
    :return: The response object
    """
    try:
        response = requests.get(url)
        response.raise_for_status()
        return response
    except requests.exceptions.RequestException as e:
        if retry_url:
            return fetch(retry_url, None)
        else:
            if e.response.status_code == 429:
                retry_after = int(e.response.headers.get('Retry-After', 10))
                print(
                    f'\nRate limited ({url})',
                    f'\nRetrying after {retry_after} seconds\n'
                )
                time.sleep(retry_after)
                return fetch(url, retry_url)
            else:
                print(f'Failed to fetch: {e}')
                os._exit(1)


def replace_section(mdx_content, start_block, end_block, new_content):
    """
    Replace or update a section in the content between start_block and end_block.

    :param mdx_content: The content to update
    :param start_block: The start block of the section
    :param end_block: The end block of the section
    :param new_content: The new content to replace or update the section with
    :return: The updated content
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
        mdx_content += f"{new_content}"

    return mdx_content


def merge_versions(current_versions, new_versions):
    """
    Merge the versions of a dependency

    :param current_versions: The current versions of the dependency
    :param new_versions: The new versions of the dependency
    :return: The merged versions of the dependency
    """

    # Split the dependency name and it's versions
    pre_ver, post_ver = current_versions.split(
        ' versions '
    )

    # Current values
    curr_gt, curr_lt = post_ver.replace(
        '`', ''
    ).replace(
        '>=', ''
    ).replace(
        '<', ''
    ).strip().split(
        ' '
    )

    # New values
    new_versions = new_versions.split(
        'versions'
    )[1].replace(
        '`', ''
    ).replace(
        '>=', ''
    ).replace(
        '<', ''
    ).strip().split(
        ' '
    )

    new_gt = new_versions[0]
    try:
        new_lt = new_versions[1]
    except IndexError:
        new_lt = '0'

    # Update the versions
    if Version(new_gt) < Version(curr_gt):
        curr_gt = new_gt
    if Version(new_lt) > Version(curr_lt):
        curr_lt = new_lt
    if Version(curr_lt) == Version(curr_gt) or Version(curr_lt) == Version(new_gt):
        curr_lt = ''

    return (
        pre_ver
        + f' versions `>={curr_gt}'
        + (f' <{curr_lt}`' if curr_lt else '`')
    )


def get_npm_versions(otel_dependency, otel_dependency_version):
    """
    Get the versions of an OTel dependency from the npmjs website

    :param otel_dependency: The OTel dependency to get the versions of
    :param otel_dependency_version: The version of the OTel dependency
    :return: The versions of the OTel dependency
    """
    npm_pack_url = 'https://www.npmjs.com/package'
    content = fetch(
        f'{npm_pack_url}/{otel_dependency}/v/{otel_dependency_version}'
    )._content

    # Extract the block of content that contains the versions
    content_block = content[
        content.find(
            b'href="#supported-versions"'
        ):
        content.find(
            b'href="#usage"'
        )
    ]
    ul_block = content_block[
        content_block.find(
            b'<ul>'
        ):content_block.find(
            b'<div class="markdown-heading">'
        )
    ]

    # Decode the byte string to a normal string
    html_string = ul_block.decode('utf-8')

    # Strip all HTML tags
    inner_text_rows = re.sub(
        r'<[^>]*>', '', html_string
    ).strip().replace(
        ' version ', ' '
    ).replace(
        ' versions ', ' '
    ).replace(
        '&gt;', '>'
    ).replace(
        '&lt;', '<'
    ).replace(
        '&amp;', '&'
    ).split('\n')

    # Extract the names and versions
    versions = []
    filtered_text_rows = [s for s in inner_text_rows if s]
    for row in filtered_text_rows:
        row_words = row.split(' ')
        package_name = row_words[0] if row_words[0].startswith('@') or len(filtered_text_rows) > 1 else otel_dependency.replace(
            instrumentation_dependency_prefix[0], ''
        ).replace(
            instrumentation_dependency_prefix[1], ''
        )
        package_versions = ' '.join(row_words[1:])

        if not package_versions:
            print(f'Failed to extract version info for {otel_dependency}')
            os._exit(1)

        if any(char.isalpha() for char in package_versions):
            print(f'Skipping: {row} (origin: {otel_dependency})')
        else:
            package_url = f'{npm_pack_url}/{package_name}'
            package_versions = f'versions `{package_versions}`'

            # Node specific
            if row_words[0].replace('.', '').lower() == 'nodejs':
                package_url = ''
                clean_package_name = otel_dependency.replace(
                    instrumentation_dependency_prefix[0], ''
                ).replace(
                    instrumentation_dependency_prefix[1], ''
                )
                package_name = f'node:{clean_package_name}'
                package_versions = (
                    f'[`Node.js`](https://nodejs.org/)'
                    + f' {package_versions}'
                )

            versions.append({
                'package_url': package_url,
                'package_name': package_name,
                'package_versions': package_versions
            })

            # Unique case
            if package_name == 'node:http':
                package_name = 'node:https'
                versions.append({
                    'package_url': package_url,
                    'package_name': package_name,
                    'package_versions': package_versions
                })

    return versions


def process_nodejs_dependencies(lang_type_config, current_dir):
    """
    Process the Node.js dependencies

    :param lang_type_config: The configuration for the Node.js dependencies
    :param current_dir: The current directory
    :return: The categories of the dependencies
    """

    # Get the categories
    categories = lang_type_config.get('categories', [])

    # Fetch the package.json file and get it's dependencies
    dependencies = fetch(
        lang_type_config.get('package_json_url', '')
    ).json().get(
        'dependencies', {}
    )

    # Get the versions of the dependencies
    for dep, ver in dependencies.items():
        # Handle OTel API dependency
        if dep == api_dependency_key:
            enrichment_mdx_path = os.path.join(current_dir, 'enrichment.mdx')
            with open(enrichment_mdx_path, 'r') as r_file:
                content = r_file.read()
                with open(enrichment_mdx_path, 'w') as w_file:
                    start_block = '## Required Dependencies'
                    end_block = '## Creating Spans'
                    content = replace_section(
                        content,
                        start_block,
                        end_block,
                        (
                            f'{start_block}'
                            + '\n\nAdd the following npm packages to your service by running:'
                            + '\n\n```bash'
                            + f'\nnpm install {dep}@{ver}'
                            + '\n```'
                            + '\n\n<Warning>'
                            + f'\n  Odigos agent implements OpenTelemetry API version {ver}.'
                            + f' Any version greater than {ver}'
                            + f' may not be compatible with Odigos agent and fail to produce data.<br />'
                            + f'\n  Please do not use caret range ~~`{dep}@^{ver}`~~'
                            + f' for this dependency in your package.json to avoid pulling in incompatible version.'
                            + f'\n</Warning>'
                            + f'\n\n{end_block}'
                        )
                    )

                    w_file.write(content)

        # Handle OTel instrumentation dependencies
        elif dep.startswith(instrumentation_dependency_prefix[0]) or dep.startswith(instrumentation_dependency_prefix[1]):
            for row_obj in get_npm_versions(dep, ver):
                r_url = row_obj.get('package_url')
                r_name = row_obj.get('package_name', '')
                r_ver = row_obj.get('package_versions')

                row_str = (
                    (
                        f'- [`{r_name}`]({r_url})'
                        if r_url
                        else f'- `{r_name}`'
                    )
                    + f' {r_ver}'
                )

                # Append the dependencies to the categories
                has_category = False
                for _, cat in categories.items():
                    cat_deps = cat.get('items', [])

                    for idx, cat_dep in enumerate(cat_deps):
                        if r_name == cat_dep.get('dependency', ''):
                            has_category = True

                            if not cat_deps[idx]['mdx']:
                                cat_deps[idx]['mdx'] = row_str
                            else:
                                cat_deps[idx]['mdx'] = merge_versions(
                                    cat_deps[idx]['mdx'], r_ver
                                )

                if not has_category:
                    categories[uncategorized_key]['items'].append(
                        {
                            'dependency': r_name,
                            'note': '',
                            'mdx': row_str
                        }
                    )

    return categories


if __name__ == '__main__':
    for root, _, files in os.walk(instrumentations_dir):
        # Skip the root directory
        if root is instrumentations_dir:
            continue

        lang = root.replace(f"{instrumentations_dir}/", "")
        lang_config = supported_languages.get(lang, {})

        for file in files:
            if file == 'native.mdx' or file == 'ebpf.mdx':

                # Read the MDX file
                mdx_path = os.path.join(root, file)
                with open(mdx_path, 'r') as r_file:
                    mdx_content = r_file.read()

                    lang_type = file.replace('.mdx', '')
                    lang_type_config = lang_config.get(lang_type)
                    if not lang_type_config:
                        print(f'Config not found for {lang} - {file}')
                        continue

                    if lang == 'nodejs':
                        print(f'\nProcessing: {mdx_path}')
                        categories = process_nodejs_dependencies(
                            lang_type_config,
                            root
                        )
                    else:
                        # TODO: add support for other languages
                        continue

                    # Sort uncategorized items
                    categories[uncategorized_key]['items'] = sorted(
                        categories[uncategorized_key]['items'],
                        key=lambda x: x.get(
                            'dependency', ''
                        ).replace(
                            '@', ''
                        ).split(' ')[0]
                    )

                    # Construct the documentation for this MDX file
                    documentation = ''
                    for cat_name, cat in categories.items():
                        cat_desc = cat.get('description', '')
                        cat_deps = cat.get('items', [])

                        if cat_deps:
                            documentation += f'\n\n### {cat_name}:\n'
                            if cat_desc:
                                documentation += f'\n{cat_desc}\n'

                            for dep in cat_deps:
                                note = dep.get('note', '')
                                mdx_text = dep.get('mdx', '')
                                note_text = f'\n  <Info>{note}</Info>' if note else ''
                                documentation += f'\n{mdx_text} {note_text}'
                            documentation += '\n'

                    mdx_content = replace_section(
                        mdx_content,
                        documentation_starting_block,
                        documentation_ending_block,
                        (
                            documentation_starting_block
                            + documentation
                            + documentation_ending_block
                        )
                    )

                    # Write the updated MDX file
                    with open(mdx_path, 'w') as w_file:
                        w_file.write(mdx_content)
