"""Regressions for project discovery inside a sandbox with an AIO workbench."""
import contextlib
import http.server
import io
from pathlib import Path
import tempfile
import threading
import types
import unittest
from unittest.mock import patch

SERVICE = Path(__file__).resolve().parents[1] / 'internal/application/service'
preview = types.ModuleType('workspace_web_preview')
exec(compile((SERVICE / 'workspace_project.py').read_text(), 'workspace_project.py', 'exec'), preview.__dict__)
with contextlib.redirect_stdout(io.StringIO()), patch.object(preview.sys, 'argv', ['preview', '{}']):
    exec(compile((SERVICE / 'workspace_web_preview.py').read_text(), 'workspace_web_preview.py', 'exec'), preview.__dict__)


@contextlib.contextmanager
def html_server(page):
    class Handler(http.server.BaseHTTPRequestHandler):
        def do_GET(self):
            body = page.encode()
            self.send_response(200)
            self.send_header('Content-Type', 'text/html')
            self.send_header('Content-Length', str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def log_message(self, *args):
            pass

    server = http.server.ThreadingHTTPServer(('127.0.0.1', 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        yield server.server_port
    finally:
        server.shutdown()
        server.server_close()
        thread.join()


class WorkspaceWebPreviewTests(unittest.TestCase):
    def proc_listener(self, proc, pid, directory, port, inode, table='tcp'):
        process = proc / str(pid)
        (process / 'fd').mkdir(parents=True)
        (process / 'cwd').symlink_to(directory, target_is_directory=True)
        (process / 'fd/7').symlink_to(f'socket:[{inode}]')
        (proc / 'net').mkdir(exist_ok=True)
        path = proc / 'net' / table
        if not path.exists():
            path.write_text('sl local_address rem_address st tx_queue tr tm->when retrnsmt uid timeout inode\n')
        with path.open('a') as output:
            output.write(f'0: 00000000:{port:04X} 00000000:0000 0A 0:0 00:0 0 0 0 {inode}\n')

    def test_only_workspace_process_listeners_are_candidates(self):
        with tempfile.TemporaryDirectory() as tmp:
            base = Path(tmp)
            root, proc = base / 'workspace', base / 'proc'
            project, other = root / 'frontend', base / 'workspace-other'
            project.mkdir(parents=True)
            other.mkdir()
            self.proc_listener(proc, 1, other, 18789, 101)
            self.proc_listener(proc, 2, root, 3000, 102)
            self.proc_listener(proc, 3, project, 5173, 103, 'tcp6')
            # Workspace containment must use path components, not a prefix.
            self.assertEqual(preview.listening_ports(root, proc), [5173, 3000])

    def test_openclaw_outside_workspace_is_not_a_project_preview(self):
        for has_project in (False, True):
            with self.subTest(has_project=has_project), tempfile.TemporaryDirectory() as tmp:
                base = Path(tmp)
                root, proc, admin = base / 'workspace', base / 'proc', base / 'openclaw'
                root.mkdir()
                admin.mkdir()
                if has_project:
                    (root / 'index.html').write_text('<html>User project</html>')
                self.proc_listener(proc, 1, admin, 18789, 101)
                discover = preview.listening_ports
                with patch.object(preview.Path, 'home', return_value=base), patch.object(preview, 'listening_ports', side_effect=lambda directory: discover(directory, proc)), patch.object(preview, 'serves_html') as probe, patch.object(preview.subprocess, 'Popen') as spawn:
                    result = preview.web_preview({'root': str(root)})
                self.assertEqual(result['status'], 'not_running' if has_project else 'project_missing')
                self.assertNotIn('url', result)
                probe.assert_not_called()
                spawn.assert_not_called()

    def test_arbitrary_workspace_html_server_remains_discoverable(self):
        # Running Python/static projects need no Node manifest to preview.
        with html_server('<html><title>My Python app</title></html>') as port, tempfile.TemporaryDirectory() as tmp:
            root, proc = Path(tmp) / 'workspace', Path(tmp) / 'proc'
            root.mkdir()
            self.proc_listener(proc, 1, root, port, 101)
            discover = preview.listening_ports
            with patch.object(preview, 'listening_ports', side_effect=lambda directory: discover(directory, proc)):
                self.assertEqual(preview.running_url(root), f'http://127.0.0.1:{port}/')

    def test_missing_process_metadata_never_falls_back_to_global_ports(self):
        with tempfile.TemporaryDirectory() as tmp:
            root, proc = Path(tmp) / 'workspace', Path(tmp) / 'proc'
            root.mkdir()
            with patch.object(preview, 'serves_html') as probe:
                self.assertEqual(preview.listening_ports(root, proc), [])
                with patch.object(preview, 'listening_ports', return_value=[]):
                    self.assertEqual(preview.running_url(root, 'http://127.0.0.1:18789/'), '')
                probe.assert_not_called()

    def test_empty_workspace_does_not_preview_or_start_aio(self):
        with html_server('<html><head><title>AIO Sandbox</title></head><body>Code Browser Terminal</body></html>') as port:
            for start in (False, True):
                with self.subTest(start=start), tempfile.TemporaryDirectory() as tmp:
                    root = Path(tmp) / 'workspace'
                    root.mkdir()
                    (root / 'input').mkdir()
                    (root / 'output').mkdir()
                    with patch.object(preview.Path, 'home', return_value=Path(tmp)), patch.object(preview, 'listening_ports', return_value=[port]), patch.object(preview.subprocess, 'Popen') as spawn:
                        spawn.return_value.pid = 1234
                        result = preview.web_preview({'root': str(root), 'start': start, 'relay_script': ''})
                    self.assertEqual(result['status'], 'project_missing')
                    self.assertNotIn('url', result)
                    spawn.assert_not_called()

    def test_stopped_project_does_not_fall_back_to_aio(self):
        with html_server('<html><title> AIO Sandbox </title></html>') as port, tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / 'workspace'
            root.mkdir()
            (root / 'index.html').write_text('<html>My project</html>')
            with patch.object(preview.Path, 'home', return_value=Path(tmp)), patch.object(preview, 'listening_ports', return_value=[port]), patch.object(preview.subprocess, 'Popen') as spawn:
                spawn.return_value.pid = 1234
                self.assertEqual(preview.web_preview({'root': str(root), 'relay_script': ''})['status'], 'not_running')
                spawn.assert_not_called()

    def test_running_project_is_selected_instead_of_aio(self):
        with html_server('<html><title>AIO Sandbox</title></html>') as aio, html_server('<html><title>User management</title></html>') as project:
            with patch.object(preview, 'listening_ports', return_value=[aio, project]):
                self.assertEqual(preview.running_url(Path('/workspace')), f'http://127.0.0.1:{project}/')

    def test_ordinary_html_without_a_title_still_previews(self):
        with html_server('<!doctype html><html><body>Project</body></html>') as port:
            self.assertTrue(preview.serves_html(f'http://127.0.0.1:{port}/'))


if __name__ == '__main__':
    unittest.main()
