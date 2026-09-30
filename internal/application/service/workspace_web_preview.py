"""Loaded after workspace_preview.py's helpers; no desktop/browser calls."""
import secrets


def project_server_environment():
    # A dev script may enable --open itself. Keep project servers disconnected
    # from the user's desktop even when the sandbox has a graphical session.
    env = dict(os.environ)
    for name in ('DISPLAY', 'WAYLAND_DISPLAY', 'XAUTHORITY', 'DBUS_SESSION_BUS_ADDRESS'):
        env.pop(name, None)
    env['BROWSER'] = 'none'
    return env


def web_preview(options):
    root = Path(options['root']).resolve()
    state_dir = Path.home() / '.cache/weknora-preview' / hashlib.sha256(str(root).encode()).hexdigest()[:16]
    state_dir.mkdir(parents=True, exist_ok=True)
    with (state_dir / 'web.lock').open('w') as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            return {'status': 'starting'}
        # Relay ports must never be rediscovered as project servers.
        relays = []
        for config_file in state_dir.glob('relay-*.json'):
            try:
                relays.append(json.loads(config_file.read_text()))
            except (OSError, ValueError):
                pass
        RESERVED_PORTS.update(item['relay_port'] for item in relays)
        url = running_url()
        if not url and options.get('start'):
            project = project_command(root)
            if not project:
                return {'status': 'not_running'}
            pid_file = state_dir / 'server.pid'
            pid = pid_file.read_text().strip() if pid_file.exists() else ''
            log_path = state_dir / 'server.log'
            process = None
            if not alive(pid):
                directory, command = project
                with log_path.open('w') as log:
                    process = subprocess.Popen(command, cwd=directory, stdin=subprocess.DEVNULL, stdout=log, stderr=log, start_new_session=True, env=project_server_environment())
                pid_file.write_text(str(process.pid))
            deadline = time.monotonic() + 25
            while time.monotonic() < deadline:
                url = running_url()
                if url or (process and process.poll() is not None):
                    break
                time.sleep(.5)
            if not url:
                return {'status': 'start_failed', 'detail': log_tail(log_path)}
        if not url:
            return {'status': 'not_running'}
        port = urlsplit(url).port
        config_file = state_dir / ('relay-%d.json' % port)
        config = next((item for item in relays if item['port'] == port and alive(item.get('pid'))), None)
        if config is None:
            with socket.socket() as sock:
                sock.bind(('0.0.0.0', 0))
                relay_port = sock.getsockname()[1]
            config = {'port': port, 'relay_port': relay_port, 'secret': secrets.token_hex(32)}
            process = subprocess.Popen([sys.executable, '-u', '-c', options['relay_script'], json.dumps(config)], stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, start_new_session=True)
            config['pid'] = process.pid
            config_file.write_text(json.dumps(config))
            config_file.chmod(0o600)
        # Do not report ready before the forwarding listener has started.
        for _ in range(30):
            try:
                with socket.create_connection(('127.0.0.1', config['relay_port']), timeout=.1):
                    return {'status': 'ready', 'url': url, **config}
            except OSError:
                time.sleep(.1)
        return {'status': 'start_failed', 'detail': 'Could not start the project relay.'}

try:
    print(json.dumps(web_preview(json.loads(sys.argv[1]))))
except Exception as error:
    print(json.dumps({'status': 'start_failed', 'detail': str(error)}))
