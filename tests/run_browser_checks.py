"""Chrome ile bağımlılıksız arayüz kontrolleri ve masaüstü/mobil ekran görüntüleri.

İsteğe bağlı geliştirme kontrolü: python tests/run_browser_checks.py
Yalnızca yerel dosyaları 127.0.0.1 üzerinde sunar. Harici paket gerektirmez.
"""
import base64
import hashlib
import http.server
import json
import os
from pathlib import Path
import socket
import struct
import subprocess
import tempfile
import threading
import time
import urllib.request

ROOT = Path(__file__).resolve().parent.parent


class DevTools:
    def __init__(self, url):
        from urllib.parse import urlsplit
        parsed = urlsplit(url)
        self.socket = socket.create_connection((parsed.hostname, parsed.port), timeout=25)
        key = base64.b64encode(os.urandom(16)).decode()
        request = (f"GET {parsed.path} HTTP/1.1\r\nHost: {parsed.netloc}\r\n"
                   f"Upgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: {key}\r\n"
                   "Sec-WebSocket-Version: 13\r\n\r\n")
        self.socket.sendall(request.encode())
        response = b""
        while not response.endswith(b"\r\n\r\n"):
            response += self.socket.recv(1)
        expected = base64.b64encode(hashlib.sha1((key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11").encode()).digest())
        assert response.split(b"\r\n", 1)[0].split()[1] == b"101" and expected in response
        self.counter = 0
        self.errors = []

    def read(self, count):
        result = b""
        while len(result) < count:
            chunk = self.socket.recv(count - len(result))
            if not chunk:
                raise ConnectionError("DevTools connection closed")
            result += chunk
        return result

    def receive(self):
        accumulated = b""
        while True:
            first, second = self.read(2)
            length = second & 127
            if length == 126:
                length = struct.unpack("!H", self.read(2))[0]
            elif length == 127:
                length = struct.unpack("!Q", self.read(8))[0]
            mask = self.read(4) if second & 128 else None
            payload = self.read(length)
            if mask:
                payload = bytes(value ^ mask[i % 4] for i, value in enumerate(payload))
            if first & 15 == 8:
                raise ConnectionError("DevTools websocket closed")
            if first & 15 in (0, 1):
                accumulated += payload
            if first & 128 and accumulated:
                return json.loads(accumulated)

    def call(self, method, params=None):
        self.counter += 1
        request_id = self.counter
        payload = json.dumps({"id": request_id, "method": method, "params": params or {}}).encode()
        mask = os.urandom(4)
        header = bytes((0x81, 0x80 | len(payload))) if len(payload) < 126 else bytes((0x81, 0xFE)) + struct.pack("!H", len(payload))
        self.socket.sendall(header + mask + bytes(value ^ mask[i % 4] for i, value in enumerate(payload)))
        while True:
            response = self.receive()
            if response.get("method") == "Runtime.exceptionThrown":
                self.errors.append(response["params"]["exceptionDetails"])
            if response.get("id") == request_id:
                if "error" in response:
                    raise RuntimeError(response["error"])
                return response.get("result", {})

    def evaluate(self, expression):
        result = self.call("Runtime.evaluate", {"expression": expression, "awaitPromise": True, "returnByValue": True, "userGesture": True})
        if "exceptionDetails" in result:
            raise RuntimeError(result["exceptionDetails"])
        return result.get("result", {}).get("value")


class QuietHandler(http.server.SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=str(ROOT), **kwargs)

    def log_message(self, *args):
        pass


def main():
    chrome = Path(os.environ.get("PROGRAMFILES", "C:/Program Files")) / "Google/Chrome/Application/chrome.exe"
    if not chrome.exists():
        raise SystemExit("Chrome bulunamadı. Betikteki chrome yolunu düzenleyin.")
    server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), QuietHandler)
    threading.Thread(target=server.serve_forever, daemon=True).start()
    output = ROOT / "tests" / "artifacts"
    output.mkdir(exist_ok=True)
    process = None
    devtools = None
    profile_context = tempfile.TemporaryDirectory(prefix="kocbenefits-browser-", ignore_cleanup_errors=True)
    try:
        profile = profile_context.name
        if profile:
            process = subprocess.Popen([
                str(chrome), "--headless=new", "--disable-gpu", "--no-first-run",
                "--no-default-browser-check", "--disable-background-networking", "--disable-sync",
                "--disable-extensions", "--disable-component-update", "--remote-debugging-port=0",
                f"--user-data-dir={profile}", "about:blank"
            ], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, creationflags=getattr(subprocess, "CREATE_NO_WINDOW", 0))
            port_file = Path(profile) / "DevToolsActivePort"
            port = None
            for _ in range(150):
                try:
                    lines = port_file.read_text().splitlines()
                    if lines and lines[0].isdigit():
                        port = lines[0]
                        break
                except (FileNotFoundError, PermissionError):
                    pass
                if process.poll() is not None:
                    raise RuntimeError("Chrome exited before DevTools was ready")
                time.sleep(0.1)
            if not port:
                raise RuntimeError("Chrome DevTools port was not ready")
            with urllib.request.urlopen(f"http://127.0.0.1:{port}/json/list") as response:
                targets = json.load(response)
            devtools = DevTools(next(target["webSocketDebuggerUrl"] for target in targets if target["type"] == "page"))
            devtools.call("Runtime.enable")
            devtools.call("Page.enable")
            devtools.call("Page.bringToFront")
            devtools.call("Emulation.setFocusEmulationEnabled", {"enabled": True})
            devtools.call("Emulation.setDeviceMetricsOverride", {"width": 1440, "height": 1040, "deviceScaleFactor": 1, "mobile": False})
            devtools.call("Page.navigate", {"url": f"http://127.0.0.1:{server.server_port}/"})
            devtools.call("Browser.grantPermissions", {"origin": f"http://127.0.0.1:{server.server_port}", "permissions": ["clipboardReadWrite", "clipboardSanitizedWrite"]})
            for _ in range(100):
                if devtools.evaluate("Boolean(document.querySelector('#auth-form'))"):
                    break
                time.sleep(0.1)
            def screenshot(name):
                devtools.evaluate("Promise.all([...document.images].map(i=>i.decode().catch(()=>{})))")
                image = devtools.call("Page.captureScreenshot", {"format": "png", "captureBeyondViewport": False})
                (output / name).write_bytes(base64.b64decode(image["data"]))

            screenshot("login-desktop.png")
            checks = devtools.evaluate((ROOT / "tests" / "checks.js").read_text(encoding="utf-8"))
            checks["runtimeErrors"] = devtools.errors
            screenshot("overview-desktop.png")
            devtools.call("Emulation.setDeviceMetricsOverride", {"width": 390, "height": 844, "deviceScaleFactor": 1, "mobile": True})
            mobile = devtools.evaluate("({width:innerWidth, documentWidth:document.documentElement.scrollWidth, navigationLinks:document.querySelectorAll('#mobile-nav a').length, navigationVisible:getComputedStyle(document.querySelector('#mobile-nav')).display!=='none'})")
            checks["mobile"] = mobile
            screenshot("overview-mobile.png")
            devtools.evaluate("location.hash='benefit/kort';new Promise(resolve=>setTimeout(resolve,100))")
            devtools.evaluate("document.querySelector('[data-action=reveal]').click()")
            screenshot("detail-mobile.png")
            checks["mobileDetailOverflow"] = devtools.evaluate("document.documentElement.scrollWidth > innerWidth")
            devtools.call("Emulation.setDeviceMetricsOverride", {"width": 320, "height": 740, "deviceScaleFactor": 1, "mobile": True})
            checks["smallMobileOverflow"] = devtools.evaluate("document.documentElement.scrollWidth > innerWidth")
            devtools.evaluate("document.querySelector('[data-action=logout]').click()")
            checks["smallLoginOverflow"] = devtools.evaluate("document.documentElement.scrollWidth > innerWidth")
            screenshot("login-mobile.png")
            # Önceki öğrenci kontrollerinin ardından ek yönetici senaryoları.
            devtools.call("Emulation.setDeviceMetricsOverride", {"width": 1440, "height": 1040, "deviceScaleFactor": 1, "mobile": False})
            devtools.evaluate("document.querySelector('[data-action=sample-login]').click();new Promise(resolve=>setTimeout(resolve,100))")
            checks["admin"] = devtools.evaluate((ROOT / "tests" / "admin-checks.js").read_text(encoding="utf-8"))
            screenshot("admin-overview-desktop.png")
            rail_width = devtools.evaluate("document.querySelector('.sidebar').getBoundingClientRect().width")
            devtools.call("Input.dispatchMouseEvent", {"type": "mouseMoved", "x": 36, "y": 200})
            time.sleep(0.3)
            expanded_width = devtools.evaluate("document.querySelector('.sidebar').getBoundingClientRect().width")
            checks["adminBrandLayout"] = devtools.evaluate("[...document.querySelectorAll('.sidebar .brand, .sidebar .brand > span')].map(e=>({text:e.textContent,y:e.getBoundingClientRect().y,height:e.getBoundingClientRect().height}))")
            screenshot("admin-menu-desktop.png")
            devtools.call("Input.dispatchMouseEvent", {"type": "mouseMoved", "x": 500, "y": 50})
            time.sleep(0.3)
            collapsed_width = devtools.evaluate("document.querySelector('.sidebar').getBoundingClientRect().width")
            checks["adminDesktopMenu"] = {"collapsed": rail_width, "hovered": expanded_width, "afterLeave": collapsed_width}
            devtools.call("Emulation.setDeviceMetricsOverride", {"width": 390, "height": 844, "deviceScaleFactor": 1, "mobile": True})
            devtools.evaluate("document.querySelector('#admin-menu-button').click();new Promise(resolve=>setTimeout(resolve,250))")
            checks["adminMobileMenuOpen"] = devtools.evaluate("document.body.classList.contains('admin-menu-open') && !document.querySelector('#menu-scrim').hidden && document.querySelectorAll('#desktop-nav a').length===12")
            screenshot("admin-menu-mobile.png")
            devtools.evaluate("document.querySelector('#desktop-nav a[href=\"#admin/benefits\"]').click();new Promise(resolve=>setTimeout(resolve,250))")
            checks["adminMobileMenuClosesAfterNavigation"] = devtools.evaluate("!document.body.classList.contains('admin-menu-open') && document.querySelector('#menu-scrim').hidden")
            checks["adminMobileTableOverflow"] = devtools.evaluate("document.documentElement.scrollWidth > innerWidth")
            screenshot("admin-benefits-mobile.png")
            devtools.evaluate("document.querySelector('#admin-menu-button').click()")
            devtools.call("Input.dispatchKeyEvent", {"type": "keyDown", "key": "Escape", "code": "Escape", "windowsVirtualKeyCode": 27})
            checks["adminMobileEscapeClosesMenu"] = devtools.evaluate("!document.body.classList.contains('admin-menu-open')")
            devtools.evaluate("location.hash='admin/benefits?new=1';new Promise(resolve=>setTimeout(resolve,100))")
            checks["adminMobileFormOverflow"] = devtools.evaluate("document.documentElement.scrollWidth > innerWidth")
            screenshot("admin-form-mobile.png")
            devtools.call("Emulation.setEmulatedMedia", {"features": [{"name": "prefers-reduced-motion", "value": "reduce"}]})
            checks["reducedMotion"] = devtools.evaluate("getComputedStyle(document.querySelector('.sidebar')).transitionDuration==='0s' && getComputedStyle(document.querySelector('main')).animationName==='none'")
            devtools.call("Emulation.setEmulatedMedia", {"features": []})
            devtools.call("Page.reload")
            for _ in range(100):
                if devtools.evaluate("Boolean(document.querySelector('#auth-form'))"):
                    break
                time.sleep(0.1)
            checks["managementPersistsOnReload"] = devtools.evaluate("KOC_STORE.state.audit.length>0 && KOC_STORE.benefits.some(b=>b.title==='Test avantajı %25')")
            # Dosyaya çift tıklama akışı da HTTP önizlemesinden bağımsız kontrol edilir.
            devtools.call("Page.navigate", {"url": (ROOT / "index.html").as_uri()})
            for _ in range(100):
                if devtools.evaluate("Boolean(document.querySelector('#auth-form'))"):
                    break
                time.sleep(0.1)
            devtools.evaluate("document.querySelector('[data-action=sample-login]').click();new Promise(resolve=>setTimeout(resolve,100))")
            checks["directFileLaunch"] = devtools.evaluate("!document.querySelector('#app-view').hidden && document.querySelectorAll('.category-tile').length===6")
            checks["runtimeErrors"] = devtools.errors
            (output / "results.json").write_text(json.dumps(checks, ensure_ascii=False, indent=2), encoding="utf-8")
            print(json.dumps(checks, ensure_ascii=True, indent=2))
            try:
                devtools.call("Browser.close")
            except (ConnectionError, OSError):
                pass
            process.wait(timeout=15)
            devtools.socket.close()
            devtools = None
            # Chromium alt süreçlerinin profil dosyalarını bırakmasına kısa süre tanı.
            time.sleep(0.5)
            passed = all(check["passed"] for check in checks["checks"])
            assert passed and not checks["runtimeErrors"] and checks["directFileLaunch"]
            assert all(check["passed"] for check in checks["admin"]["checks"])
            assert rail_width == collapsed_width == 76 and expanded_width == 268
            assert checks["adminMobileMenuOpen"] and checks["adminMobileMenuClosesAfterNavigation"] and checks["adminMobileEscapeClosesMenu"]
            assert checks["reducedMotion"] and checks["managementPersistsOnReload"]
            assert not checks["adminMobileTableOverflow"] and not checks["adminMobileFormOverflow"]
            assert mobile["documentWidth"] <= mobile["width"] and mobile["navigationVisible"] and mobile["navigationLinks"] == 6
            assert not any(checks[key] for key in ("mobileDetailOverflow", "smallMobileOverflow", "smallLoginOverflow"))
    finally:
        if devtools:
            devtools.socket.close()
        if process and process.poll() is None:
            process.terminate()
            process.wait(timeout=15)
        server.shutdown()
        server.server_close()
        profile_context.cleanup()


if __name__ == "__main__":
    main()
