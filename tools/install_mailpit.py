"""Install the pinned Windows Mailpit binary into ignored .tools (no service)."""
import hashlib
import io
import platform
from pathlib import Path
import urllib.request
import zipfile

VERSION = "v1.31.6"
SHA256 = "5d5ad094bfb3bac9004f9d1105d2335fea9f8952c09951df73ae0a280e34d3b1"
ROOT = Path(__file__).resolve().parent.parent

if __name__ == "__main__":
    if platform.system() != "Windows" or platform.machine().lower() not in ("amd64", "x86_64"):
        raise SystemExit("Use https://mailpit.axllent.org/docs/install/ for your operating system.")
    url = f"https://github.com/axllent/mailpit/releases/download/{VERSION}/mailpit-windows-amd64.zip"
    archive = urllib.request.urlopen(url, timeout=60).read()
    if hashlib.sha256(archive).hexdigest() != SHA256:
        raise SystemExit("Mailpit checksum mismatch; nothing installed.")
    target = ROOT / ".tools" / "mailpit"
    target.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(io.BytesIO(archive)) as bundle:
        # Extract only the named executable, never arbitrary archive paths.
        (target / "mailpit.exe").write_bytes(bundle.read("mailpit.exe"))
    print(f"Mailpit {VERSION} installed locally; SHA256 verified. No service started.")
