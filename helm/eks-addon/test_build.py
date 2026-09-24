import importlib.util
import json
from pathlib import Path
import subprocess
import tarfile
import tempfile
import unittest

import jsonschema
import yaml

spec = importlib.util.spec_from_file_location("addon_build", Path(__file__).with_name("build.py"))
builder = importlib.util.module_from_spec(spec)
spec.loader.exec_module(builder)


class AddonBuildTests(unittest.TestCase):
    def test_artifact_lifecycle_and_regionalization(self):
        images = {name: builder.REGISTRY + "odigos/" + name + ":test" for name in builder.COMPONENTS}
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "release"
            chart = builder.build(output, "1.37.1", images)
            report = json.loads((output / "preflight.json").read_text())
            self.assertEqual(report["findings"], [])
            self.assertEqual(report["resourcesChangingBetweenIdenticalRenders"], [])
            command = ["helm", "template", "odigos", str(chart), "--namespace", "odigos-system", "--kube-version", "1.34.0"]
            documents = list(yaml.safe_load_all(builder.run(*command)))
            resources = {doc["metadata"]["name"]: doc for doc in documents if doc and doc["kind"] == "ConfigMap"}
            manifest = json.loads((output / "release-manifest.json").read_text())
            self.assertEqual(manifest["environmentOverrideParameters"], [
                {"Key": "clusterName", "Value": "${AWS_EKS_CLUSTER_NAME}"},
            ])
            injected = list(yaml.safe_load_all(builder.run(*command, "--set-string", "clusterName=eks-substitution-test")))
            config = next(doc for doc in injected if doc and doc["kind"] == "ConfigMap" and doc["metadata"]["name"] == "odigos-configuration")
            self.assertEqual(yaml.safe_load(config["data"]["config.yaml"])["clusterName"], "eks-substitution-test")
            self.assertNotIn("data", resources["odigos-go-offsets"])
            self.assertNotIn("odigos-deployment-id", resources["odigos-deployment"]["data"])
            scheduler = next(doc for doc in documents if doc and doc["kind"] == "Deployment" and doc["metadata"]["name"] == "odigos-scheduler")
            self.assertIn({"name": "ODIGOS_INSTALLATION_METHOD", "value": "eks-addon"}, scheduler["spec"]["template"]["spec"]["containers"][0]["env"])
            odiglet = next(doc for doc in documents if doc and doc["kind"] == "DaemonSet" and doc["metadata"]["name"] == "odiglet")
            volume = next(v for v in odiglet["spec"]["template"]["spec"]["volumes"] if v["name"] == "odigos-go-offsets")
            self.assertEqual(volume["configMap"]["items"], [{"key": "go_offset_results.json", "path": "go_offset_results.json"}])
            self.assertFalse(any(doc and doc["kind"] == "Job" for doc in documents))
            cleanup = yaml.safe_load((output / "cleanup-job.yaml").read_text())
            container = cleanup["spec"]["template"]["spec"]["containers"][0]
            self.assertEqual(container["image"], images["cli"])
            self.assertIn("--instrumentation-only", container["args"])
            self.assertNotIn("annotations", cleanup["metadata"])
            self.assertNotIn("ttlSecondsAfterFinished", cleanup["spec"])

            values = yaml.safe_load((chart / "values.yaml").read_text())
            values["images"] = {name: uri.replace("us-east-1", "ap-northeast-2") for name, uri in images.items()}
            (chart / "values.yaml").write_text(yaml.safe_dump(values))
            regional = builder.run(*command)
            for uri in values["images"].values():
                if uri != values["images"]["cli"]:
                    self.assertIn(uri, regional)
            self.assertFalse("dkr.ecr.us-east-1.amazonaws.com" in regional, "image configuration did not regionalize")
            for setting in ("marketplace.enabled=false", "insights.enabled=true", "goAutoOffsetsMode=image"):
                result = subprocess.run(command + ["--set", setting], capture_output=True, text=True)
                self.assertNotEqual(result.returncode, 0, setting)

            with tarfile.open(output / "odigos-1.37.1.tgz") as package:
                for filename in ("aws_mp_configuration_schema.json", "aws_mp_addon_parameters.json"):
                    self.assertIn("odigos/" + filename, package.getnames())

    def test_configuration_schema_rejects_unsupported_features_and_secrets(self):
        schema = json.loads((builder.HERE / "aws_mp_configuration_schema.json").read_text())
        jsonschema.Draft7Validator.check_schema(schema)
        validator = jsonschema.Draft7Validator(schema)
        for values in ({}, {"logLevel": "info"}, {"marketplace": {"licenseManagerRegion": "us-east-1"}}):
            validator.validate(values)
        for values in ({"clusterName": "buyer-override"}, {"onPremToken": "secret"}, {"images": {}}, {"marketplace": {"enabled": False}}, {"insights": {"enabled": True}}):
            self.assertFalse(validator.is_valid(values))

    def test_rejects_unreviewed_release_inputs(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "release"
            images = {name: builder.REGISTRY + "odigos/" + name + ":test" for name in builder.COMPONENTS}
            for version in ("0.0.0", "1.38.0-pre4", "latest"):
                with self.assertRaises(ValueError):
                    builder.build(output, version, images)
            images["enterprise-agents"] = "docker.io/example/agents:v1"
            with self.assertRaises(ValueError):
                builder.build(output, "1.37.1", images)
            self.assertFalse(output.exists())


if __name__ == "__main__":
    unittest.main()
