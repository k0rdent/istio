"""
Generates a Mermaid diagram representing MultiClusterService and ServiceTemplate dependencies
from a rendered Helm chart template.
"""

import argparse
import subprocess
from abc import ABC, abstractmethod
import re
import yaml

MULTI_CLUSTER_SERVICE_KIND = "MultiClusterService"
SERVICE_TEMPLATE_KIND = "ServiceTemplate"


def parse_arguments() -> argparse.Namespace:
    """
    Parse command-line arguments for the diagram generator.
    Returns:
        argparse.Namespace: Parsed command-line arguments.
    """

    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--chart-dir",
        default="../charts/k0rdent-istio",
        help="path to the chart directory",
    )
    parser.add_argument(
        "--output-file-path",
        default="../dev/mcs-dependencies-diagram.md",
        help="path to the output file with diagram",
    )
    return parser.parse_args()


def render_helm_template(chart_dir: str, release_name: str = "k0rdent-istio") -> str:
    """
    Render a Helm chart template using the 'helm template' command.
    Args:
        chart_dir (str): Path to the Helm chart directory.
        release_name (str): Name of the Helm release.
    Returns:
        str: Rendered Helm template as a string.
    """

    result = subprocess.run(
        ["helm", "template", release_name, chart_dir],
        capture_output=True,
        text=True,
        check=True,
    )
    return result.stdout


def extract_kind(yaml_text: str, kind_to_find: str) -> list[dict]:
    """
    Extract all YAML documents of a specific Kubernetes resource kind from a YAML string.
    Args:
        yaml_text (str): YAML string.
        kind_to_find (str): The Kubernetes resource kind to extract.
    Returns:
        list[dict]: List of YAML documents matching the specified kind.
    """

    found = []
    for doc in yaml.safe_load_all(yaml_text):
        if isinstance(doc, dict) and doc.get("kind") == kind_to_find:
            found.append(doc)
    return found


class Resource(ABC):
    """
    Abstract base class representing a Kubernetes resource.
    """

    @staticmethod
    def deep_get(data: dict, path: str, default=None) -> any:
        """
        Retrieve a nested value from a dictionary using a dot-separated path.
        Args:
            data (dict): The dictionary to search.
            path (str): Dot-separated path to the desired value.
            default: Default value to return if the path does not exist.
        Returns:
            any: The value at the specified path or the default value.
        """

        keys = path.split(".")
        for key in keys:
            if not isinstance(data, dict):
                return default
            data = data.get(key)
            if data is None:
                return default
        return data

    @abstractmethod
    def kind(self) -> str:
        """
        Returns the kind of the resource.
        """

    @abstractmethod
    def name(self) -> str:
        """
        Returns the name of the resource.
        """

    @abstractmethod
    def deps_with_kinds(self) -> list[tuple[str, str]]:
        """
        Returns a list of tuples containing the names and kinds of dependent resources.
        """


class TemplateResource(Resource):
    """
    Represents a template resource with a name and kind.
    """

    def __init__(self, name: str, kind: str):
        self._name = name
        self._kind = kind

    def kind(self) -> str:
        return self._kind

    def name(self) -> str:
        return self._name

    def deps_with_kinds(self) -> list[tuple[str, str]]:
        return []


class ServiceTemplate(Resource):
    """
    Represents a ServiceTemplate resource.
    """

    def __init__(self, template_yaml: dict) -> None:
        self._name: str = self.deep_get(template_yaml, "metadata.name", "")
        local_ref: dict = self.deep_get(template_yaml, "spec.resources.localSourceRef")
        self.service_deps: list[TemplateResource] = []

        if local_ref:
            template_name: str = self.deep_get(local_ref, "name", "")
            template_kind: str = self.deep_get(local_ref, "kind", "")
            self.service_deps.append(TemplateResource(template_name, template_kind))

    def kind(self) -> str:
        return SERVICE_TEMPLATE_KIND

    def name(self) -> str:
        return self._name

    def deps_with_kinds(self) -> list[tuple[str, str]]:
        return [(dep.name(), dep.kind()) for dep in self.service_deps]


class MultiClusterService(Resource):
    """
    Represents a MultiClusterService resource.
    """

    def __init__(self, mcs_yaml: dict) -> None:
        self._name = self.deep_get(mcs_yaml, "metadata.name", "")
        self.deps_template_resources: list[TemplateResource] = []
        self.deps_service_names = []
        self.deps_mcs_names = []

        deps_mcs = self.deep_get(mcs_yaml, "spec.dependsOn", [])
        for name in deps_mcs:
            self.deps_mcs_names.append(name)

        services = self.deep_get(mcs_yaml, "spec.serviceSpec.services", [])
        for service in services:
            self.deps_service_names.append(service["template"])

        template_refs = self.deep_get(
            mcs_yaml, "spec.serviceSpec.templateResourceRefs", []
        )
        for template_ref in template_refs:
            template_name = self.deep_get(template_ref, "resource.name", "")
            template_kind = self.deep_get(template_ref, "resource.kind", "")
            template_resource = TemplateResource(template_name, template_kind)
            self.deps_template_resources.append(template_resource)

    def kind(self) -> str:
        return MULTI_CLUSTER_SERVICE_KIND

    def name(self) -> str:
        return self._name

    def deps_with_kinds(self) -> list[tuple[str, str]]:
        deps_templates = [(tr.name(), tr.kind()) for tr in self.deps_template_resources]
        deps_services = [
            (name, SERVICE_TEMPLATE_KIND) for name in self.deps_service_names
        ]
        deps_mcs = [(name, MULTI_CLUSTER_SERVICE_KIND) for name in self.deps_mcs_names]
        return deps_services + deps_templates + deps_mcs


class DiagramGenerator:
    """
    Generates a Mermaid diagram representing resource dependencies.
    """

    def __init__(self, resources: list[Resource], output_file_path: str) -> None:
        self.resources = resources
        self.output_file_path = output_file_path

    def generate_diagram(self) -> None:
        """
        Generate the Mermaid diagram and write it to the output file.
        """

        with open(self.output_file_path, "w", encoding="utf-8") as f:
            f.write("```mermaid\n")
            f.write("graph TD\n")
            for resource in self.resources:
                resource_diagram_id_name = self.get_node_label(
                    resource.name(), resource.kind()
                )
                for raw_dep_name, raw_dep_kind in resource.deps_with_kinds():
                    dep_diagram_id_name = self.get_node_label(
                        raw_dep_name, raw_dep_kind
                    )
                    f.write(
                        f"    {resource_diagram_id_name} --> {dep_diagram_id_name}\n"
                    )
            f.write("```")

    def clean_template_placeholders(self, name: str) -> str:
        """
        Cleans Helm template placeholders from a given name.
        Args:
            name (str): The name potentially containing Helm template placeholders.
        Returns:
            str: The cleaned name without Helm template placeholders.
        """

        return re.sub(r"\{\{\s*.(.*?)\s*\}\}", r"\1", name)

    def get_node_label(self, resource_name: str, resource_kind: str) -> str:
        """
        Generate a Mermaid diagram node label for a resource.
        Args:
            resource_name (str): The name of the resource.
            resource_kind (str): The kind of the resource.
        Returns:
            str: The Mermaid diagram node label.
        """
        clean_name = self.clean_template_placeholders(resource_name)
        clean_kind = self.clean_template_placeholders(resource_kind)
        return f'{clean_name}/{clean_kind}["{clean_name} ({clean_kind})"]'


if __name__ == "__main__":
    args = parse_arguments()
    template = render_helm_template(args.chart_dir)
    multi_cluster_services = extract_kind(template, MULTI_CLUSTER_SERVICE_KIND)
    service_templates = extract_kind(template, SERVICE_TEMPLATE_KIND)

    resources = []
    for mcs in multi_cluster_services:
        resources.append(MultiClusterService(mcs))

    for st in service_templates:
        resources.append(ServiceTemplate(st))

    diagram_generator = DiagramGenerator(resources, args.output_file_path)
    diagram_generator.generate_diagram()
