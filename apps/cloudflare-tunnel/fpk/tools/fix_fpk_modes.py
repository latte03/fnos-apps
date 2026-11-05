#!/usr/bin/env python3
"""Normalize modes in an fpk produced on Windows, keeping it installable.

Windows fnpack reads NTFS metadata and emits 0666 for every file (and 0777 for
directories), including lifecycle scripts. fnOS then fails with 10111. fnpack
also writes the MD5 of app.tgz into manifest as `checksum`; rewriting app.tgz
therefore requires updating that field.

This tool:
  - outer dirs 0755, cmd/* 0755, other outer files 0644
  - app.tgz dirs 0755, cloudflare-tunnel-server/bin/cloudflared 0755, others 0644
  - rewrites app.tgz, then updates manifest `checksum` to MD5(new app.tgz)
  - preserves every file's bytes (only tar headers and gzip framing change)

fnpack 1.2.3 embeds no package-level signature; the only checksum is the
app.tgz MD5 in manifest, which this tool keeps in sync.
"""
import gzip
import hashlib
import io
import re
import sys
import tarfile

EXEC_NAMES = {"cloudflare-tunnel-server", "bin/cloudflared"}
CHECKSUM_RE = re.compile(r"^(checksum\s*=\s*)([0-9a-fA-F]{32})(\s*)(\r?\n)?$")


def normalize_inner(raw: bytes) -> bytes:
    src = io.BytesIO(raw)
    dst = io.BytesIO()
    with tarfile.open(fileobj=src, mode="r:gz") as tin, \
            gzip.GzipFile(filename="", fileobj=dst, mode="wb", mtime=0) as gz, \
            tarfile.open(fileobj=gz, mode="w") as tout:
        for info in tin:
            if info.isdir():
                info.mode = 0o755
            elif info.isfile():
                info.mode = 0o755 if (info.name in EXEC_NAMES or info.name.startswith("cmd/")) else 0o644
            data = tin.extractfile(info) if info.isfile() else None
            tout.addfile(info, data)
    return dst.getvalue()


def update_checksum(manifest: bytes, digest: str) -> bytes:
    text = manifest.decode("utf-8")
    lines = text.splitlines(keepends=True)
    found = False
    for i, line in enumerate(lines):
        m = CHECKSUM_RE.match(line)
        if m:
            lines[i] = m.group(1) + digest + m.group(3) + (m.group(4) or "")
            found = True
            break
    if not found:
        if lines and not lines[-1].endswith(("\n", "\r")):
            lines[-1] += "\n"
        lines.append(f"checksum                   = {digest}\n")
    return "".join(lines).encode("utf-8")


def fix(src_path: str, dst_path: str) -> None:
    with tarfile.open(src_path, "r:gz") as tin:
        members = tin.getmembers()
        app_raw = tin.extractfile("app.tgz").read()
        manifest_raw = tin.extractfile("manifest").read()

        new_app = normalize_inner(app_raw)
        digest = hashlib.md5(new_app).hexdigest()
        new_manifest = update_checksum(manifest_raw, digest)

        with open(dst_path, "wb") as raw, \
                gzip.GzipFile(filename="", fileobj=raw, mode="wb", mtime=0) as gz, \
                tarfile.open(fileobj=gz, mode="w") as tout:
            for info in members:
                if info.isdir():
                    info.mode = 0o755
                elif info.isfile():
                    info.mode = 0o755 if info.name.startswith("cmd/") else 0o644

                if info.name == "app.tgz":
                    info.size = len(new_app)
                    tout.addfile(info, io.BytesIO(new_app))
                elif info.name == "manifest":
                    info.size = len(new_manifest)
                    tout.addfile(info, io.BytesIO(new_manifest))
                else:
                    data = tin.extractfile(info) if info.isfile() else None
                    tout.addfile(info, data)

    print(f"app.tgz md5 = {digest}")
    print(f"normalized: {dst_path}")


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("usage: fix_fpk_modes.py <input.fpk> <output.fpk>", file=sys.stderr)
        raise SystemExit(2)
    fix(sys.argv[1], sys.argv[2])
