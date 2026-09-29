"""Runs as the sandbox user, never on the application host."""
import concurrent.futures
import fcntl
import hashlib
import http.client
import json
import os
from pathlib import Path
import shutil
import socket
import subprocess
import sys
import time
from urllib.parse import urlsplit

RESERVED_PORTS = {49983, 5900, 6080, 9222}
PREVIEW_WINDOW_CLASS = "WeKnoraPreview"
SKIP_DIRS = {"node_modules", "input", "output", "vendor", "venv", "__pycache__"}


def listening_ports():
    ports = set()
    for table in ("/proc/net/tcp", "/proc/net/tcp6"):
        try:
            for line in Path(table).read_text().splitlines()[1:]:
                fields = line.split()
                if fields[3] == "0A":
                    port = int(fields[1].split(":")[-1], 16)
                    if port >= 1024 and port not in RESERVED_PORTS:
                        ports.add(port)
        except OSError:
            pass
    preferred = [5173, 3000, 4173, 8080, 8000, 3001]
    return sorted(ports, key=lambda p: (preferred.index(p) if p in preferred else 99, p))[:32]


def serves_html(url):
    # Direct loopback request: ignore HTTP_PROXY and do not follow redirects.
    parsed = urlsplit(url)
    connection = http.client.HTTPConnection(parsed.hostname, parsed.port, timeout=0.6)
    try:
        connection.request("GET", (parsed.path or "/") + ("?" + parsed.query if parsed.query else ""))
        response = connection.getresponse()
        sample = response.read(1024).lower()
        return response.status < 400 and ("text/html" in (response.getheader("Content-Type") or "") or b"<!doctype html" in sample or b"<html" in sample)
    except (OSError, http.client.HTTPException):
        return False
    finally:
        connection.close()


def running_url(preferred=""):
    if preferred and serves_html(preferred):
        return preferred
    urls = ["http://127.0.0.1:%d/" % port for port in listening_ports()]
    with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
        results = list(pool.map(serves_html, urls))
    return next((url for url, ready in zip(urls, results) if ready), "")


def project_command(root):
    """Bounded discovery; prefer frontend packages over a sibling API server."""
    candidates = []
    seen = 0
    for folder, dirs, files in os.walk(root, followlinks=False):
        directory = Path(folder)
        depth = len(directory.relative_to(root).parts)
        dirs[:] = sorted(d for d in dirs if d not in SKIP_DIRS and not d.startswith(".") and not (directory / d).is_symlink()) if depth < 3 else []
        seen += 1
        if seen > 160:
            break
        score = 0 if directory.name in ("frontend", "web", "client", "app") else 10
        if "package.json" in files and not (directory / "package.json").is_symlink():
            try:
                with (directory / "package.json").open() as manifest:
                    package = json.loads(manifest.read(262144))
                scripts = package.get("scripts", {})
                script = next((key for key in ("dev", "start", "serve") if isinstance(scripts.get(key), str)), None)
                if script:
                    manager = "pnpm" if (directory / "pnpm-lock.yaml").exists() else "yarn" if (directory / "yarn.lock").exists() else "npm"
                    if not shutil.which(manager):
                        manager = "npm"
                    candidates.append((score, str(directory), [manager, "run", script]))
                    continue
                # A framework's index.html contains modules that need its server.
                continue
            except (OSError, ValueError, TypeError, AttributeError):
                continue
        if "index.html" in files and not (directory / "index.html").is_symlink():
            candidates.append((score + 20, str(directory), None))
    if not candidates:
        return None
    _, directory, command = sorted(candidates, key=lambda c: (c[0], c[1]))[0]
    if command is None:
        with socket.socket() as sock:
            sock.bind(("127.0.0.1", 0))
            port = sock.getsockname()[1]
        command = [sys.executable, "-m", "http.server", str(port), "--bind", "127.0.0.1"]
    return directory, command


def browser_command(url, state_dir):
    browser = next((shutil.which(name) for name in ("chromium", "chromium-browser", "google-chrome", "google-chrome-stable", "firefox") if shutil.which(name)), None)
    if not browser:
        for base in (Path.home() / ".cache/ms-playwright", Path("/ms-playwright"), Path("/opt/ms-playwright")):
            matches = sorted(base.glob("chromium-*/chrome-linux*/chrome"))
            if matches:
                browser = str(matches[-1])
                break
    if not browser:
        return None
    if "firefox" in browser:
        return [browser, "--new-window", url]
    return [browser, "--no-sandbox", "--disable-dev-shm-usage", "--no-first-run",
            "--class=" + PREVIEW_WINDOW_CLASS, "--window-position=0,0", "--window-size=1280,800",
            "--user-data-dir=" + str(state_dir / "preview-browser"), "--app=" + url]


def desktop_command(*args):
    """Only fixed commands and a window from our own WM_CLASS reach X11."""
    return subprocess.run(args, capture_output=True, text=True, timeout=4,
                          env={**os.environ, "DISPLAY": ":0"}, check=True).stdout.strip()


def preview_windows():
    try:
        output = desktop_command("xdotool", "search", "--onlyvisible", "--class", "^" + PREVIEW_WINDOW_CLASS + "$")
        return [window for window in output.splitlines() if window.isdecimal()]
    except (OSError, subprocess.SubprocessError):
        return []


def apply_viewport(window, viewport):
    """Resize the real app, then crop VNC to that window. No RFB resize opcode.

    Xvfb has a fixed framebuffer; cropping through x11vnc keeps the existing
    connection and correct pointer coordinates while the browser reflows.
    Never resize an arbitrary active desktop application.
    """
    if window not in preview_windows():
        raise RuntimeError("The preview browser window is no longer available.")
    screen_width, screen_height = map(int, desktop_command("xdotool", "getdisplaygeometry").split())
    width = min(390 if viewport == "mobile" else 1280, screen_width)
    height = min(780 if viewport == "mobile" else 800, screen_height)
    desktop_command("xprop", "-id", window, "-f", "_MOTIF_WM_HINTS", "32c", "-set", "_MOTIF_WM_HINTS", "2,0,0,0,0")
    desktop_command("xdotool", "windowsize", "--sync", window, str(width), str(height))
    desktop_command("xdotool", "windowmove", "--sync", window, "0", "0")
    desktop_command("xdotool", "windowactivate", "--sync", window)
    geometry = dict(line.split("=", 1) for line in desktop_command("xdotool", "getwindowgeometry", "--shell", window).splitlines() if "=" in line)
    actual_width, actual_height = int(geometry["WIDTH"]), int(geometry["HEIGHT"])
    x, y = int(geometry["X"]), int(geometry["Y"])
    if actual_width != width or actual_height != height or x < 0 or y < 0 or x + width > screen_width or y + height > screen_height:
        raise RuntimeError("The window manager could not apply the requested preview size.")
    desktop_command("x11vnc", "-display", ":0", "-sync", "-R", "clip:%dx%d+%d+%d" % (width, height, x, y))
    return {"viewport": viewport, "width": width, "height": height}


def control_preview(options, window, url):
    if not window:
        return {"status": "browser_unavailable", "url": url, "detail": "Open the project preview before changing its viewport or refreshing it."}
    try:
        viewport = apply_viewport(window, options.get("viewport") or "desktop")
        if options.get("action") == "reload":
            desktop_command("xdotool", "key", "--clearmodifiers", "--window", window, "ctrl+r")
        return {"status": "ready", "url": url, **viewport}
    except (OSError, subprocess.SubprocessError, ValueError, KeyError, RuntimeError):
        return {"status": "start_failed", "url": url, "detail": "This desktop could not adjust the preview window. It requires Chromium, xdotool, xprop and x11vnc remote control."}


def alive(pid):
    try:
        os.kill(int(pid), 0)
        return Path("/proc/%s/stat" % int(pid)).read_text().split(") ", 1)[1][0] != "Z"
    except (OSError, ValueError, IndexError, TypeError):
        return False


def log_tail(path, limit=4000):
    try:
        with path.open("rb") as log:
            log.seek(0, os.SEEK_END)
            log.seek(max(0, log.tell() - limit))
            return log.read(limit).decode("utf-8", errors="replace")
    except OSError:
        return ""


def open_preview(options):
    root = Path(options["root"]).resolve()
    state_dir = Path.home() / ".cache/weknora-preview" / hashlib.sha256(str(root).encode()).hexdigest()[:16]
    state_dir.mkdir(parents=True, exist_ok=True)
    # Cross-request locking prevents duplicate development servers.
    with (state_dir / "lock").open("w") as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            return {"status": "starting"}
        requested = options.get("url", "")
        if options.get("action") in ("resize", "reload"):
            windows = preview_windows()
            return control_preview(options, windows[-1] if windows else "", requested)
        url = requested if requested and serves_html(requested) else "" if requested else running_url()
        if not url and options.get("start") and not requested:
            project = project_command(root)
            if not project:
                return {"status": "not_running", "detail": "No runnable dev/start/serve script or static index.html found."}
            pid_file = state_dir / "server.pid"
            pid = pid_file.read_text().strip() if pid_file.exists() else ""
            log_path = state_dir / "server.log"
            process = None
            if not alive(pid):
                directory, command = project
                with log_path.open("w") as log:
                    process = subprocess.Popen(command, cwd=directory, stdin=subprocess.DEVNULL, stdout=log, stderr=log, start_new_session=True, env={**os.environ, "BROWSER": "none"})
                pid_file.write_text(str(process.pid))
            deadline = time.monotonic() + 25
            while time.monotonic() < deadline:
                url = running_url()
                if url or (process and process.poll() is not None):
                    break
                time.sleep(0.5)
            if not url:
                detail = log_tail(log_path)
                return {"status": "start_failed", "detail": detail or "The project did not serve an HTML page within 25 seconds."}
        if not url:
            return {"status": "not_running"}
        # Reopening the side panel should reuse its browser, preserving forms.
        location_file = state_dir / "location"
        windows = preview_windows()
        if windows and location_file.exists() and location_file.read_text() == url:
            return control_preview(options, windows[-1], url)
        command = browser_command(url, state_dir)
        if not command:
            return {"status": "browser_unavailable", "url": url}
        with (state_dir / "browser.log").open("a") as log:
            browser = subprocess.Popen(command, stdin=subprocess.DEVNULL, stdout=log, stderr=log, start_new_session=True, env={**os.environ, "DISPLAY": ":0"})
        time.sleep(0.5)
        if browser.poll() not in (None, 0):
            return {"status": "browser_unavailable", "url": url, "detail": log_tail(state_dir / "browser.log", 2000)}
        if options.get("viewport"):
            previous_windows = set(windows)
            deadline = time.monotonic() + 4
            window = ""
            while time.monotonic() < deadline:
                fresh = [item for item in preview_windows() if item not in previous_windows]
                if fresh:
                    window = fresh[-1]
                    break
                time.sleep(0.1)
            controlled = control_preview(options, window, url)
            if controlled["status"] == "ready":
                location_file.write_text(url)
                return controlled
            # Custom/older desktops still display their browser even if they
            # cannot supply a cropped responsive viewport.
            return {"status": "ready", "url": url, "detail": controlled.get("detail", "")}
        return {"status": "ready", "url": url}


if __name__ == "__main__":
    try:
        print(json.dumps(open_preview(json.loads(sys.argv[1]))))
    except Exception as error:
        print(json.dumps({"status": "start_failed", "detail": str(error)}))
