import importlib.util
from pathlib import Path
import tempfile
import unittest

import yaml


DOCS = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("sync_dest_doc", DOCS / "sync-dest-doc.py")
sync = importlib.util.module_from_spec(spec)
spec.loader.exec_module(sync)


class RetirementDocsTest(unittest.TestCase):
    def test_retirement_replaces_setup_and_preserves_custom_guidance(self):
        catalog = yaml.safe_load((DOCS.parent / "destinations/data/lightstep.yaml").read_text())
        legacy = dict(catalog, spec=dict(catalog["spec"], deprecated=False))
        sections = sync.get_documenation(legacy)
        original = sections["content_before_custom"] + "\nKeep migration guidance\n" + sections["content_after_custom"]

        with tempfile.TemporaryDirectory() as directory:
            page = Path(directory) / "lightstep.mdx"
            page.write_text(original)
            sync.update_mdx(page, catalog)
            retired = page.read_text()
            self.assertIn("Keep migration guidance", retired)
            self.assertIn("type: lightstep", retired)
            self.assertIn("LIGHTSTEP_ACCESS_TOKEN", retired)
            self.assertEqual(retired.count("### Existing Configuration Reference"), 1)
            self.assertNotIn("### Adding Destination to Odigos", retired)
            self.assertNotIn("kubectl apply", retired)
            sync.update_mdx(page, catalog)
            self.assertEqual(page.read_text(), retired)

    def test_overview_omits_retired_destinations(self):
        with tempfile.TemporaryDirectory() as directory:
            docs = Path(directory)
            (docs / "snippets/shared").mkdir(parents=True)
            sync.process_overview(DOCS.parent / "destinations/data", docs)
            overview = (docs / "snippets/shared/backends-overview.mdx").read_text()
            self.assertNotIn("/backends/lightstep", overview)
            self.assertIn("/backends/jaeger", overview)
            self.assertIn("/backends/splunk", overview)


if __name__ == "__main__":
    unittest.main()
