"""Windows/Chrome end-to-end auth check against an isolated PostgreSQL test schema.

Starts and stops its own API, Vite, Mailpit and Chrome. Requires free local ports
8080, 5173, 8025, 1025. Never uses the development database or sends real email.
"""
import base64
import json
import os
from pathlib import Path
import re
import shutil
import socket
import subprocess
import tempfile
import time
import urllib.request
import urllib.parse
import uuid

from run_browser_checks import DevTools

ROOT = Path(__file__).resolve().parent.parent
ARTIFACTS = ROOT / "tests" / "artifacts"
BASE = "http://127.0.0.1:5173/"
PASSWORD = "Local browser test password 2026!"
NEW_PASSWORD = "Changed browser test password 2026!"
ADMIN = "admin@ogrenci.example"
STUDENT = "student@ogrenci.example"
OPERATOR = "TestOperator"
OPERATOR_PASSWORD = "496271835"  # Synthetic operator-only policy fixture.


def wait_for(check, label, seconds=20):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        try:
            if check():
                return
        except (OSError, ValueError):
            pass
        time.sleep(.1)
    raise AssertionError(label)


def get_json(url):
    with urllib.request.urlopen(url, timeout=2) as response:
        return json.load(response)


def mail_token(email, purpose):
    result = None
    def find():
        nonlocal result
        for item in get_json("http://127.0.0.1:8025/api/v1/messages")["messages"]:
            if not any(to["Address"] == email for to in item["To"]):
                continue
            msg = get_json("http://127.0.0.1:8025/api/v1/message/" + item["ID"])
            match = re.search(r"#" + purpose + r"\?token=([A-Za-z0-9_-]+)", msg["Text"])
            if match:
                result = match.group(1)
                return True
        return False
    wait_for(find, "verification/reset email not received locally")
    return result


def main():
    os.chdir(ROOT)
    env = dict(os.environ)
    local_env = ROOT / ".env"
    if local_env.exists():
        for line in local_env.read_text(encoding="utf-8-sig").splitlines():
            if "=" in line and not line.lstrip().startswith("#"):
                key, value = line.split("=", 1)
                env.setdefault(key.strip(), value.strip().strip('"'))
    raw = env.get("TEST_DATABASE_URL", "")
    parsed = urllib.parse.urlsplit(raw)
    if parsed.scheme not in ("postgres", "postgresql") or not parsed.path.endswith("_test"):
        raise SystemExit("TEST_DATABASE_URL must point to a dedicated database ending in _test.")
    for port in (8080, 5173, 8025, 1025):
        with socket.socket() as sock:
            if sock.connect_ex(("127.0.0.1", port)) == 0:
                raise SystemExit(f"Stop the local service on port {port} before this isolated test.")
    node_dirs = list((ROOT / ".tools").glob("node-*-win-x64"))
    node_dir = str(node_dirs[0]) if node_dirs else ""
    env["PATH"] = os.pathsep.join([str(ROOT / ".tools/go/bin"), node_dir, env.get("PATH", "")])
    go = shutil.which("go", path=env["PATH"])
    node = shutil.which("node", path=env["PATH"])
    psql = shutil.which("psql") or "C:/Program Files/PostgreSQL/18/bin/psql.exe"
    mailpit = ROOT / ".tools/mailpit/mailpit.exe"
    chrome = Path("C:/Program Files/Google/Chrome/Application/chrome.exe")
    if not go or not node or not mailpit.exists() or not chrome.exists():
        raise SystemExit("Go, Node, installed Mailpit and Windows Chrome are required.")
    schema = "auth_browser_" + uuid.uuid4().hex
    pg_env = dict(env, PGHOST=parsed.hostname or "127.0.0.1", PGPORT=str(parsed.port or 5432), PGUSER=urllib.parse.unquote(parsed.username or ""), PGPASSWORD=urllib.parse.unquote(parsed.password or ""), PGDATABASE=parsed.path[1:])
    def sql(statement):
        result = subprocess.run([psql, "-X", "-v", "ON_ERROR_STOP=1", "-c", statement], env=pg_env, capture_output=True, text=True)
        if result.returncode:
            raise RuntimeError("Isolated PostgreSQL setup or cleanup failed; no credentials printed.")
    params = dict(urllib.parse.parse_qsl(parsed.query))
    params["search_path"] = schema
    test_url = urllib.parse.urlunsplit(parsed._replace(query=urllib.parse.urlencode(params)))
    env.update(DATABASE_URL=test_url, APP_ENV="test", SCHOOL_EMAIL_DOMAIN="ogrenci.example", APP_ORIGIN=BASE.rstrip("/"), HTTP_ADDR="127.0.0.1:8080", ELIGIBILITY_POLICY="manual", REGISTRATION_OPEN="true", LOCAL_EMAIL="true", SMTP_HOST="127.0.0.1", SMTP_PORT="1025", SMTP_FROM="noreply@kocbenefits.example", SMTP_USERNAME="", SMTP_PASSWORD="")
    processes = []
    logs = []
    schema_created = False
    temp = tempfile.TemporaryDirectory(prefix="koc-auth-browser-", ignore_cleanup_errors=True)
    folder = Path(temp.name)
    def command(args, **kwargs):
        result = subprocess.run(args, env=env, cwd=ROOT, capture_output=True, text=True, encoding="utf-8", **kwargs)
        if result.returncode:
            raise RuntimeError("Test command failed: " + result.stderr[:2000])
        return result
    def start(args, name):
        log = open(folder / (name + ".log"), "w", encoding="utf-8")
        logs.append(log)
        proc = subprocess.Popen(args, env=env, cwd=ROOT, stdout=log, stderr=log, creationflags=subprocess.CREATE_NO_WINDOW)
        processes.append(proc)
        return proc
    try:
        sql(f'CREATE SCHEMA "{schema}"')
        schema_created = True
        for name in ("api", "admin", "db"):
            command([go, "build", "-o", str(folder / (name + ".exe")), "./cmd/" + name])
        command([str(folder / "db.exe"), "up"])
        start([str(mailpit), "--listen", "127.0.0.1:8025", "--smtp", "127.0.0.1:1025", "--database", str(folder / "mail.db"), "--disable-version-check", "--smtp-disable-rdns", "--quiet"], "mail")
        wait_for(lambda: get_json("http://127.0.0.1:8025/api/v1/messages"), "Mailpit unavailable")
        command([str(folder / "admin.exe"), "-name", "Test Yöneticisi", "-email", ADMIN], input=PASSWORD + "\n")
        command([str(folder / "admin.exe"), "-name", "Test Operatörü", "-username", OPERATOR], input=OPERATOR_PASSWORD + "\n")
        start([str(folder / "api.exe")], "api")
        start([node, str(ROOT / "node_modules/vite/bin/vite.js"), "--host", "127.0.0.1"], "web")
        wait_for(lambda: get_json(BASE + "api/ready")["status"] == "ok", "API/Vite unavailable", 40)
        profile = folder / "chrome"
        start([str(chrome), "--headless=new", "--disable-gpu", "--no-first-run", "--remote-debugging-port=0", "--user-data-dir=" + str(profile), "about:blank"], "chrome")
        portfile = profile / "DevToolsActivePort"
        debug_port = None
        def read_debug_port():
            nonlocal debug_port
            lines = portfile.read_text().splitlines()
            if lines:
                debug_port = lines[0]
                return True
            return False
        wait_for(read_debug_port, "Chrome unavailable")
        targets = get_json(f"http://127.0.0.1:{debug_port}/json/list")
        dev = DevTools(next(t["webSocketDebuggerUrl"] for t in targets if t["type"] == "page"))
        dev.call("Runtime.enable"); dev.call("Page.enable"); dev.call("Network.enable")
        def navigate(fragment):
            dev.call("Page.navigate", {"url": BASE + "#" + fragment})
        def present(selector):
            wait_for(lambda: dev.evaluate(f"Boolean(document.querySelector({json.dumps(selector)}))"), "missing element " + selector)
        def text_present(text):
            wait_for(lambda: dev.evaluate(f"document.body.innerText.includes({json.dumps(text)})"), "expected screen text missing: " + text)
        def fill(name, value):
            present("#" + name)
            dev.evaluate(f"document.getElementById({json.dumps(name)}).value={json.dumps(value)}")
        def submit(selector="form.auth-form"):
            dev.evaluate(f"document.querySelector({json.dumps(selector)}).requestSubmit()")
        def click_text(text, scope="document"):
            assert dev.evaluate(f"(() => {{const b=Array.from({scope}.querySelectorAll('button')).find(b=>b.textContent.trim()==={json.dumps(text)});if(!b)return false;b.click();return true}})()"), "missing button " + text
        def login(email, password=PASSWORD, role="student"):
            navigate("login/" + role); fill("username" if role == "admin" else "email", email); fill("password", password); submit()
        def logout():
            click_text("Çıkış yap"); present("#password")
        ARTIFACTS.mkdir(exist_ok=True)
        navigate("login/student"); present("#password")
        for width in (1440, 390, 320):
            dev.call("Emulation.setDeviceMetricsOverride", {"width":width,"height":1000,"deviceScaleFactor":1,"mobile":width<500})
            assert dev.evaluate("document.documentElement.scrollWidth<=innerWidth"), f"login overflow {width}"
            shot=dev.call("Page.captureScreenshot", {"format":"png"})
            (ARTIFACTS / f"auth-login-{width}.png").write_bytes(base64.b64decode(shot["data"]))
        navigate("verify-email?token=" + mail_token(ADMIN,"verify-email")); present("form.auth-form"); submit()
        present("#password"); assert "token=" not in dev.evaluate("location.hash")
        navigate("register"); fill("name","Test Öğrencisi"); fill("email",STUDENT); fill("password",PASSWORD); submit()
        text_present("Doğrulama e-postasını yeniden gönder")
        student_token=mail_token(STUDENT,"verify-email")
        navigate("verify-email?token="+student_token); present("form.auth-form"); submit(); present("#password")
        login(STUDENT); text_present("yönetici onayı bekliyor")
        login(ADMIN, role="admin"); present(".admin-account")
        card=f"Array.from(document.querySelectorAll('.admin-account')).find(a=>a.textContent.includes({json.dumps(STUDENT)}))"
        click_text("Onayla",card); present("dialog[open]"); fill("current_password",PASSWORD)
        dev.evaluate("document.querySelector('dialog input[type=checkbox]').click()")
        submit("dialog form"); wait_for(lambda:not dev.evaluate("Boolean(document.querySelector('dialog[open]'))"),"approval dialog did not close")
        text_present("Hesap güncellendi")
        assert dev.evaluate("document.documentElement.scrollWidth<=innerWidth"), "admin overflow320"
        shot=dev.call("Page.captureScreenshot",{"format":"png"});(ARTIFACTS/"auth-admin-320.png").write_bytes(base64.b64decode(shot["data"]))
        logout(); login(STUDENT); present(".account-details"); text_present("Test Öğrencisi")
        assert not dev.evaluate("document.cookie.includes('kb_session')"), "session cookie readable by JS"
        navigate("admin"); text_present("yönetici yetkisi gerekir")
        logout(); navigate("forgot-password"); fill("email",STUDENT); submit()
        reset_token=mail_token(STUDENT,"reset-password")
        navigate("reset-password?token="+reset_token); fill("password",NEW_PASSWORD); submit(); present("#email")
        assert "token=" not in dev.evaluate("location.hash")
        login(STUDENT,NEW_PASSWORD); present(".account-details"); logout()
        navigate("verify-email?token="+student_token); present("form.auth-form"); submit(); text_present("Bağlantı geçersiz")
        navigate("login/student"); present("#password")
        dev.call("Network.setBlockedURLs",{"urls":["*/api/auth/session"]})
        dev.call("Page.reload")
        text_present("Şu anda bağlantı kurulamıyor")
        dev.call("Network.setBlockedURLs",{"urls":[]}); click_text("Yeniden dene"); present("#password")
        dev.call("Emulation.setEmulatedMedia",{"features":[{"name":"prefers-reduced-motion","value":"reduce"}]})
        navigate("register"); present("#name")
        assert dev.evaluate("document.documentElement.scrollWidth<=innerWidth"), "register overflow320"
        login(OPERATOR, OPERATOR_PASSWORD, role="admin"); present(".admin-account")
        navigate("account"); present(".account-details"); text_present(OPERATOR)
        assert not dev.evaluate("document.querySelector('.account-details').textContent.includes('E-posta doğrulaması')"), "operator incorrectly shown as email verified"
        assert dev.evaluate("document.documentElement.scrollWidth<=innerWidth"), "operator account overflow320"
        logout()
        assert not dev.errors, "browser runtime exceptions"
        print("Auth browser checks passed: SMTP capture, email admin verification, email-free operator login, student registration/verification/approval/login, RBAC UI, HttpOnly cookie, password reset, token replay rejection, error/retry, reduced motion, 1440/390/320 widths.")
    finally:
        for proc in reversed(processes):
            if proc.poll() is None:
                proc.terminate()
                try: proc.wait(timeout=15)
                except subprocess.TimeoutExpired: proc.kill(); proc.wait(timeout=5)
        for log in logs: log.close()
        if schema_created: sql(f'DROP SCHEMA "{schema}" CASCADE')
        temp.cleanup()


if __name__ == "__main__":
    main()
