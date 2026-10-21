"""Isolated Chrome viewport audit of the static prototype (not the React app).

Run: python tests/prototype_responsive_checks.py
No database, credentials or external packages are required. Mobile is emulated.
"""
import base64
import http.server
import json
import os
from pathlib import Path
import subprocess
import tempfile
import threading
import time
import urllib.request

from run_browser_checks import DevTools, QuietHandler, ROOT


def main():
    chrome = Path(os.environ.get("PROGRAMFILES", "C:/Program Files")) / "Google/Chrome/Application/chrome.exe"
    server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), QuietHandler)
    threading.Thread(target=server.serve_forever, daemon=True).start()
    output = ROOT / "tests/artifacts/prototype-responsive"
    output.mkdir(parents=True, exist_ok=True)
    process = devtools = None
    profile = tempfile.TemporaryDirectory(prefix="prototype-responsive-", ignore_cleanup_errors=True)
    rows = []
    try:
        process = subprocess.Popen([
            str(chrome), "--headless=new", "--disable-gpu", "--no-first-run",
            "--no-default-browser-check", "--disable-background-networking", "--disable-sync",
            "--disable-extensions", "--remote-debugging-port=0",
            f"--user-data-dir={profile.name}", "about:blank",
        ], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
            creationflags=getattr(subprocess, "CREATE_NO_WINDOW", 0))
        port_file = Path(profile.name) / "DevToolsActivePort"
        port = None
        for _ in range(150):
            try:
                lines = port_file.read_text().splitlines()
                if lines and lines[0].isdigit():
                    port = lines[0]
                    break
            except (FileNotFoundError, PermissionError):
                pass
            time.sleep(.1)
        assert port, "Chrome DevTools did not become ready"
        with urllib.request.urlopen(f"http://127.0.0.1:{port}/json/list") as response:
            targets = json.load(response)
        devtools = DevTools(next(t["webSocketDebuggerUrl"] for t in targets if t["type"] == "page"))
        devtools.call("Page.enable")
        devtools.call("Runtime.enable")
        devtools.call("Page.navigate", {"url": f"http://127.0.0.1:{server.server_port}/"})
        for _ in range(100):
            if devtools.evaluate("Boolean(document.querySelector('[data-action=sample-login]'))"):
                break
            time.sleep(.1)
        devtools.evaluate("document.querySelector('[data-action=sample-login]').click()")
        routes = ["overview", "discover", "favorites", "history", "profile", "help",
                  "benefit/mola", "benefit/rota", "benefit/kort", "benefit/sayfa"]
        for width in [320, 390, 768, 1024, 1440]:
            devtools.call("Emulation.setDeviceMetricsOverride", {
                "width": width, "height": 844 if width < 768 else 1040,
                "deviceScaleFactor": 1, "mobile": width < 768})
            for route in routes:
                if not route.startswith("benefit/"):
                    selector = f"{'#mobile-nav' if width <= 760 else '#desktop-nav'} a[href='#{route}']"
                    devtools.evaluate(f"document.querySelector({json.dumps(selector)}).click();new Promise(r=>setTimeout(r,100))")
                else:
                    devtools.evaluate(f"location.hash={json.dumps(route)};new Promise(r=>setTimeout(r,100))")
                if route in ("benefit/mola", "benefit/kort"):
                    devtools.evaluate("document.querySelector('[data-action=reveal]')?.click()")
                devtools.evaluate("Promise.all([...document.querySelectorAll('main img')].map(i=>{i.loading='eager';return i.decode().catch(()=>{});}))")
                row = devtools.evaluate("""(() => {
                    const main=document.querySelector('main');
                    const nav=document.querySelector(innerWidth<=760?'#mobile-nav':'#desktop-nav');
                    return {width:innerWidth, documentWidth:document.documentElement.scrollWidth,
                        content:!!main.querySelector('h1,h2'),
                        brokenImages:[...main.querySelectorAll('img')].filter(i=>!i.naturalWidth).length,
                        navigationVisible:!!nav.getClientRects().length};
                })()""")
                row.update(route=route)
                expected = {
                    "overview": "innerWidth>360 || document.querySelector('.hero-art').getBoundingClientRect().top>=document.querySelector('.hero .btn').getBoundingClientRect().bottom",
                    "benefit/mola": "document.querySelector('#discount-code')?.value==='DEMO-MOLA20'",
                    "benefit/rota": "document.querySelector('.redeem-panel a')?.target==='_blank'",
                    "benefit/kort": "document.querySelector('.qr-box img')?.naturalWidth>0",
                    "benefit/sayfa": "document.querySelector('.expired-panel button')?.disabled===true",
                }.get(route, f"location.hash==={json.dumps('#' + route)}")
                row["interaction"] = devtools.evaluate(expected)
                row["passed"] = (row["documentWidth"] <= width and row["content"]
                                 and not row["brokenImages"] and row["navigationVisible"] and row["interaction"])
                rows.append(row)
                if width in (320, 390, 1440):
                    shot = devtools.call("Page.captureScreenshot", {"format": "png", "captureBeyondViewport": False})
                    (output / f"{width}-{route.replace('/', '-')}.png").write_bytes(base64.b64decode(shot["data"]))
        result = {"scope": "Static prototype; Chrome viewport emulation, not physical devices",
                  "checks": rows, "runtimeErrors": devtools.errors}
        (output / "results.json").write_text(json.dumps(result, indent=2), encoding="utf-8")
        failures = [row for row in rows if not row["passed"]]
        print(json.dumps({"total": len(rows), "failures": failures, "runtimeErrors": devtools.errors}, indent=2))
        assert not failures and not devtools.errors
    finally:
        if devtools:
            devtools.socket.close()
        if process and process.poll() is None:
            process.terminate()
            process.wait(timeout=15)
        server.shutdown()
        server.server_close()
        profile.cleanup()


if __name__ == "__main__":
    main()
