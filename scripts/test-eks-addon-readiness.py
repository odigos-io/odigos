import importlib.util
from pathlib import Path
import tempfile
import unittest


spec = importlib.util.spec_from_file_location("preflight", Path(__file__).with_name("check-eks-addon-readiness.py"))
preflight = importlib.util.module_from_spec(spec)
spec.loader.exec_module(preflight)


class PreflightTests(unittest.TestCase):
    def test_template_scan_ignores_comments_and_similar_names(self):
        source = '''{{/* lookup "v1" "Secret" .Release.Revision */}}
lookupKey: safe
name: {{ .Release.Name }}
namespace: {{ .Release.Namespace }}
value: {{ lookup "v1" "Secret" .Release.Namespace "example" }}
revision: {{ .Release.Revision }}
annotations:
  "helm.sh/hook": pre-delete
'''
        with tempfile.TemporaryDirectory() as directory:
            chart = Path(directory)
            (chart / "templates").mkdir()
            (chart / "templates/test.yaml").write_text(source)
            findings = preflight.scan_templates(chart)
        self.assertEqual([(f["code"], f["line"]) for f in findings],
                         [("helm_lookup", 5), ("helm_release_object", 6), ("helm_hook", 8)])

    def test_inventory_includes_injected_images(self):
        docs = [{"kind": "Deployment", "apiVersion": "apps/v1", "metadata": {"name": "test"},
                 "spec": {"template": {"spec": {"initContainers": [{"image": "marketplace/init:v1"}],
                    "containers": [{"image": "marketplace/runtime:v1", "env": [
                        {"name": "ODIGOS_INIT_CONTAINER_IMAGE", "value": "external/agent:v1"},
                        {"name": "ODIGOS_COLLECTOR_IMAGE", "value": "marketplace/collector:v1"},
                        {"name": "SECRET", "value": "must-not-be-in-inventory"}]}]}}}}]
        images, findings = preflight.inspect_render(docs, "marketplace")
        self.assertEqual(set(images), {"marketplace/init:v1", "marketplace/runtime:v1",
                                       "external/agent:v1", "marketplace/collector:v1"})
        self.assertEqual([f["code"] for f in findings], ["image_outside_marketplace"])

    def test_crd_and_its_resource_are_flagged(self):
        docs = [{"kind": "CustomResourceDefinition", "spec": {"group": "example.com", "names": {"kind": "Widget"}}},
                {"kind": "Widget", "apiVersion": "example.com/v1", "metadata": {"name": "test"}}]
        _, findings = preflight.inspect_render(docs, "marketplace")
        self.assertEqual([f["code"] for f in findings], ["crd_and_custom_resource", "missing_workload"])


if __name__ == "__main__":
    unittest.main()
