"""Stage a bounded source backup; the host persists it through artifact storage."""
import json
import os
import stat
import sys
import tempfile
import time
import zipfile


def main():
    args = json.loads(sys.argv[1])
    root = os.path.abspath(args["root"])
    output = os.path.abspath(args["output"])
    max_bytes = args["max_bytes"]
    if os.path.realpath(root) != root or not output.startswith(root + os.sep):
        raise ValueError("backup paths must stay inside the workspace")
    # Refuse symlinked output directories, including any parent component.
    if os.path.realpath(output) != output:
        raise ValueError("backup output must not be a symlink")
    os.makedirs(output, exist_ok=True)

    skipped_dirs = {".git", "node_modules", ".venv", "venv", "__pycache__",
                    ".cache", ".pytest_cache", ".mypy_cache", ".ruff_cache",
                    ".next", ".nuxt", ".pnpm-store"}
    skipped_roots = {output, os.path.join(root, "input"), os.path.join(root, "output")}
    files = []
    total = 0
    visited = 0

    def walk_error(error):
        raise error  # Never present an unreadable, partial tree as a full backup.

    for directory, dirs, names in os.walk(root, followlinks=False, onerror=walk_error):
        visited += 1
        if visited > 20000:
            raise ValueError("source backup exceeds 20000 directories")
        dirs[:] = sorted(name for name in dirs
                         if name not in skipped_dirs
                         and os.path.join(directory, name) not in skipped_roots
                         and not os.path.islink(os.path.join(directory, name)))
        for name in sorted(names):
            if name == ".git" or name.startswith(".weknora-source-backup-") or name.endswith((".pyc", ".pyo")):
                continue
            path = os.path.join(directory, name)
            info = os.lstat(path)
            if not stat.S_ISREG(info.st_mode):
                continue
            total += info.st_size
            files.append((path, info))
            if total > 200 * 1024 * 1024 or len(files) > 20000:
                raise ValueError("source backup exceeds 200 MiB or 20000 files")
    destination = os.path.join(output, "workspace-source.zip")
    if not files and not os.path.isfile(destination):
        return

    # Stage outside the collected output tree so interruptions cannot upload
    # an incomplete archive. Publication is one atomic rename on the same disk.
    fd, temporary = tempfile.mkstemp(prefix=".weknora-source-backup-", suffix=".zip", dir=root)
    try:
        with os.fdopen(fd, "w+b") as stream:
            with zipfile.ZipFile(stream, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=6) as archive:
                # Identify the turn even when its source reverts to an older
                # version: historical hash de-duplication must not hide the
                # newest recovery point behind a later, different backup.
                archive.comment = ("WeKnora source backup; turn=" + args["turn"]).encode("utf-8")
                for path, before in files:
                    if os.path.realpath(path) != path:
                        raise ValueError("source path changed during backup")
                    source_fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
                    with os.fdopen(source_fd, "rb") as source:
                        info = os.fstat(source.fileno())
                        if (not stat.S_ISREG(info.st_mode) or info.st_ino != before.st_ino
                                or info.st_size != before.st_size or info.st_mtime_ns != before.st_mtime_ns):
                            raise ValueError("source changed during backup")
                        # Use a stable UTC timestamp for retries of this turn.
                        stamp = time.gmtime(max(315532800, min(info.st_mtime, 4354819198)))[:6]
                        entry = zipfile.ZipInfo(os.path.relpath(path, root), stamp)
                        entry.create_system = 3
                        entry.external_attr = (info.st_mode & 0xFFFF) << 16
                        entry.compress_type = zipfile.ZIP_DEFLATED
                        entry.file_size = info.st_size
                        copied = 0
                        with archive.open(entry, "w") as target:
                            while True:
                                chunk = source.read(1024 * 1024)
                                if not chunk:
                                    break
                                copied += len(chunk)
                                if copied > info.st_size:
                                    raise ValueError("source grew during backup")
                                target.write(chunk)
                                if stream.tell() > max_bytes:
                                    raise ValueError("compressed source backup exceeds artifact size limit")
                        after = os.fstat(source.fileno())
                        if copied != info.st_size or after.st_mtime_ns != info.st_mtime_ns:
                            raise ValueError("source changed during backup")
            if stream.tell() > max_bytes:
                raise ValueError("compressed source backup exceeds artifact size limit")
        os.replace(temporary, destination)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        sys.exit("source backup failed: " + str(error))
