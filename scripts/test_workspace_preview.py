"""Regression tests for the helper executed inside remote sandboxes."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch, MagicMock

spec = importlib.util.spec_from_file_location("preview", Path(__file__).resolve().parents[1] / "internal/application/service/workspace_preview.py")
preview = importlib.util.module_from_spec(spec)
spec.loader.exec_module(preview)


class PreviewTests(unittest.TestCase):
    def test_prefers_nested_frontend_to_backend_and_ignores_dependencies(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            for folder in ("demo/backend", "demo/frontend", "node_modules/frontend"):
                directory = root / folder
                directory.mkdir(parents=True)
                (directory / "package.json").write_text(json.dumps({"scripts": {"dev": "vite"}}))
            with patch.object(preview.shutil, "which", return_value="/usr/bin/npm"):
                directory, command = preview.project_command(root)
            self.assertEqual(directory, str(root / "demo/frontend"))
            self.assertEqual(command, ["npm", "run", "dev"])

    def test_does_not_run_external_symlink_or_unbuilt_framework_index(self):
        with tempfile.TemporaryDirectory() as tmp, tempfile.TemporaryDirectory() as outside:
            root = Path(tmp)
            (Path(outside) / "index.html").write_text("<html>outside</html>")
            (root / "linked").symlink_to(outside, target_is_directory=True)
            (root / "package.json").write_text('{"scripts":{"test":"node test.js"}}')
            (root / "index.html").write_text('<script type="module" src="/src/main.ts"></script>')
            self.assertIsNone(preview.project_command(root))

    def test_probe_uses_local_socket_and_does_not_follow_redirects_or_json(self):
        for status, content_type, body, expected in [(200, "text/html", b"page", True), (200, "application/json", b"{}", False), (302, "", b"", False), (500, "text/html", b"<html>error</html>", False)]:
            connection = MagicMock()
            response = connection.getresponse.return_value
            response.status = status
            response.getheader.return_value = content_type
            response.read.return_value = body
            with patch.object(preview.http.client, "HTTPConnection", return_value=connection) as factory:
                self.assertEqual(preview.serves_html("http://localhost:5173/users?q=one"), expected)
                factory.assert_called_once_with("localhost", 5173, timeout=0.6)
                connection.request.assert_called_once_with("GET", "/users?q=one")
                connection.close.assert_called_once()

    def test_no_server_does_not_start_project_on_lookup(self):
        with tempfile.TemporaryDirectory() as tmp, patch.object(preview.Path, "home", return_value=Path(tmp)), patch.object(preview, "running_url", return_value=""), patch.object(preview.subprocess, "Popen") as spawn:
            self.assertEqual(preview.open_preview({"root": tmp})["status"], "not_running")
            spawn.assert_not_called()

    def test_manual_url_does_not_fall_back_to_another_server_or_start(self):
        with tempfile.TemporaryDirectory() as tmp, patch.object(preview.Path, "home", return_value=Path(tmp)), patch.object(preview, "serves_html", return_value=False), patch.object(preview, "running_url") as discover, patch.object(preview.subprocess, "Popen") as spawn:
            result = preview.open_preview({"root": tmp, "url": "http://localhost:9999", "start": True})
            self.assertEqual(result["status"], "not_running")
            discover.assert_not_called()
            spawn.assert_not_called()

    def test_start_detaches_server_and_navigates_browser_on_guest_display(self):
        with tempfile.TemporaryDirectory() as tmp, patch.object(preview.Path, "home", return_value=Path(tmp)), patch.object(preview, "running_url", side_effect=["", "http://127.0.0.1:5173/"]), patch.object(preview, "project_command", return_value=(tmp, ["npm", "run", "dev"])), patch.object(preview, "browser_command", return_value=["chromium", "--app=http://127.0.0.1:5173/"]), patch.object(preview.time, "sleep"), patch.object(preview.subprocess, "Popen") as spawn:
            spawn.return_value.poll.return_value = None
            spawn.return_value.pid = 1234
            result = preview.open_preview({"root": tmp, "start": True})
            self.assertEqual(result["status"], "ready")
            self.assertEqual(spawn.call_count, 2)
            server, browser = spawn.call_args_list
            self.assertEqual(server.args[0], ["npm", "run", "dev"])
            self.assertTrue(server.kwargs["start_new_session"])
            self.assertEqual(server.kwargs["stdin"], preview.subprocess.DEVNULL)
            self.assertEqual(browser.kwargs["env"]["DISPLAY"], ":0")

    def test_browser_missing_is_distinct_from_project_failure(self):
        with tempfile.TemporaryDirectory() as tmp, patch.object(preview.Path, "home", return_value=Path(tmp)), patch.object(preview, "running_url", return_value="http://127.0.0.1:5173/"), patch.object(preview, "browser_command", return_value=None):
            self.assertEqual(preview.open_preview({"root": tmp})["status"], "browser_unavailable")


if __name__ == "__main__":
    unittest.main()
