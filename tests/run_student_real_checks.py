"""CI-only real student UI/API checks against the Go harness supplied origin.

Starts Chrome only. API and built frontend hosting belong to TestStudentBrowser.
Uses the real fetch transport; never intercepts, fulfills, or replaces API responses.
"""
import base64
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time
import urllib.parse
import urllib.request

from run_browser_checks import DevTools

ROOT = Path(__file__).resolve().parent.parent
OUT = ROOT / 'tests/artifacts/student-real'


def main():
    base = os.environ['KOC_BROWSER_URL'].rstrip('/')
    origin = urllib.parse.urlsplit(base)
    assert origin.scheme == 'http' and origin.hostname in ('127.0.0.1', 'localhost', '::1'), 'isolated loopback harness required'
    student = os.environ['KOC_BROWSER_STUDENT']
    password = os.environ['KOC_BROWSER_PASSWORD']
    offers = json.loads(os.environ['KOC_BROWSER_BENEFITS'])
    assert all(offers.get(method) for method in ('code', 'link', 'qr', 'membership'))
    chrome = os.environ.get('KOC_CHROME', '/usr/bin/google-chrome')
    OUT.mkdir(parents=True, exist_ok=True)
    profile = tempfile.TemporaryDirectory(prefix='koc-student-real-', ignore_cleanup_errors=True)
    process = subprocess.Popen([
        chrome, '--headless=new', '--disable-gpu', '--disable-dev-shm-usage',
        '--no-first-run', '--no-default-browser-check', '--remote-debugging-port=0',
        '--user-data-dir=' + profile.name, 'about:blank',
    ], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
        creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
    dev = None
    checks = []

    def wait(check, label):
        deadline = time.monotonic() + 20
        while time.monotonic() < deadline:
            if check():
                return
            if process.poll() is not None:
                raise AssertionError('Chrome exited: ' + label)
            time.sleep(.05)
        raise AssertionError(label)

    try:
        portfile = Path(profile.name) / 'DevToolsActivePort'
        wait(lambda: portfile.exists() and bool(portfile.read_text()), 'Chrome startup')
        port = portfile.read_text().splitlines()[0]
        with urllib.request.urlopen(f'http://127.0.0.1:{port}/json/list', timeout=5) as response:
            targets = json.load(response)
        dev = DevTools(next(t['webSocketDebuggerUrl'] for t in targets if t['type'] == 'page'))
        for method in ('Runtime.enable', 'Page.enable', 'Network.enable'):
            dev.call(method)
        dev.call('Emulation.setFocusEmulationEnabled', {'enabled': True})
        # Observe real transport without capturing passwords, cookies or redemption values.
        dev.call('Page.addScriptToEvaluateOnNewDocument', {'source': '''
            window.__realRequests=[];
            const originalFetch=window.fetch.bind(window);
            window.fetch=async function(input,options){
                const response=await originalFetch(input,options);
                const path=new URL(typeof input==='string'?input:input.url,location.href).pathname;
                window.__realRequests.push({path,method:options?.method||(input instanceof Request?input.method:'GET'),status:response.status});
                return response;
            };
        '''})

        def present(selector):
            wait(lambda: dev.evaluate('!!document.querySelector(' + json.dumps(selector) + ')'), 'missing ' + selector)

        def route(value, selector):
            dev.evaluate('location.hash=' + json.dumps(value))
            wait(lambda: dev.evaluate('location.hash') == '#' + value, 'route ' + value)
            present(selector)
            if value.startswith('benefit/'):
                title = api('benefits/' + value.split('/', 1)[1])['benefit']['title']
                wait(lambda: dev.evaluate('document.querySelector("#student-main h1")?.textContent===' + json.dumps(title)), 'new benefit detail')

        def fill(selector, value):
            present(selector)
            dev.evaluate(f'''(()=>{{const e=document.querySelector({json.dumps(selector)});
                Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set.call(e,{json.dumps(value)});
                e.dispatchEvent(new Event('input',{{bubbles:true}}));}})()''')

        def button(label):
            wait(lambda: dev.evaluate(f'''(()=>{{const e=[...document.querySelectorAll('button')].find(e=>e.textContent.trim()==={json.dumps(label)}&&!e.disabled);if(!e)return false;e.click();return true}})()'''), 'button ' + label)

        def api(path):
            return dev.evaluate(f'''(async()=>{{const r=await fetch({json.dumps('/api/student/' + path)},{{credentials:'same-origin'}});if(!r.ok)throw new Error('API '+r.status);return r.json()}})()''')

        def layout(name):
            for width, height in ((320, 740), (390, 844), (1440, 1000)):
                dev.call('Emulation.setDeviceMetricsOverride', {'width': width, 'height': height, 'deviceScaleFactor': 1, 'mobile': width < 1000})
                time.sleep(.1)
                result = dev.evaluate('({viewport:innerWidth,page:document.documentElement.scrollWidth})')
                assert result['page'] <= width + 1 and result['viewport'] <= width + 1, f'overflow {name} {width}: {result}'
                image = dev.call('Page.captureScreenshot', {'format': 'png', 'captureBeyondViewport': False})
                (OUT / f'{name}-{width}.png').write_bytes(base64.b64decode(image['data']))
                checks.append({'page': name, 'width': width})

        def login():
            fill('#email', student)
            fill('#password', password)
            dev.evaluate("document.querySelector('form.auth-form').requestSubmit()")
            present('.student-panel')

        def copy_result(value):
            wait(lambda: dev.evaluate("!!document.querySelector('.student-redemption [role=status]')"), 'copy feedback')
            manual = dev.evaluate("document.querySelector('.student-manual-copy')?.value||''")
            hint = dev.evaluate("document.querySelector('.student-redemption [role=status]').textContent")
            assert manual == value or 'Kopyalandı' in hint, 'copy must succeed or expose exact manual value'

        code_id = offers['code']
        dev.call('Page.navigate', {'url': base + '/#benefit/' + code_id})
        present('form.auth-form #email')
        assert dev.evaluate("!document.querySelector('.student-panel')"), 'anonymous student content'
        login()
        present('.student-redemption')
        assert dev.evaluate('location.hash') == '#benefit/' + code_id, 'direct link lost through login'
        cookies = dev.call('Network.getAllCookies').get('cookies', [])
        assert any(c.get('httpOnly') for c in cookies), 'real HTTP session cookie missing'
        layout('deep-link')

        public_fields = ('code', 'target_url', 'qr_content', 'membership_label')
        detail = api('benefits/' + code_id)['benefit']
        assert not any(k in detail for k in public_fields), 'secret fields in public detail'
        route('discover', '.benefit-card')
        catalog = api('catalog')['benefits']
        assert all(not any(k in b for k in public_fields) for b in catalog), 'secret fields in public catalog'
        layout('catalog')
        # Add through rendered UI, then confirm the server's own account-scoped response.
        selector = '.benefit-card:has(a[href="#benefit/' + code_id + '"]) .favorite-toggle'
        present(selector)
        assert dev.evaluate('document.querySelector(' + json.dumps(selector) + ').getAttribute("aria-pressed")') == 'false', 'fixture unexpectedly favored'
        dev.evaluate('document.querySelector(' + json.dumps(selector) + ').click()')
        wait(lambda: dev.evaluate('document.querySelector(' + json.dumps(selector) + ').getAttribute("aria-pressed")==="true"'), 'favorite UI PUT')
        assert any(b['id'] == code_id for b in api('favorites')['benefits']), 'favorite did not persist on server'
        requests = dev.evaluate('window.__realRequests')
        assert any(r['path'] == '/api/student/favorites/' + code_id and r['method'] == 'PUT' and r['status'] == 200 for r in requests), 'UI did not issue successful real favorite PUT'
        route('profile', '.student-profile')
        button('Çıkış yap')
        present('form.auth-form #email')
        assert not dev.evaluate("!!document.querySelector('.student-panel')"), 'panel retained after logout'
        login()
        route('favorites', '.benefit-card')
        present('a[href="#benefit/' + code_id + '"]')
        assert any(b['id'] == code_id for b in api('favorites')['benefits']), 'favorite lost after new login session'
        layout('favorites-new-session')

        labels = {'code': 'Kodu görüntüle', 'link': 'Avantaj bağlantısını getir', 'qr': 'QR kodunu göster', 'membership': 'Üyelik bilgisini göster'}
        for method, label in labels.items():
            route('benefit/' + offers[method], '.student-redemption')
            # Read expected values from the real gated API, never from fixture constants.
            expected = api('benefits/' + offers[method] + '/redemption')
            assert expected['method'] == method
            button(label)
            present('.student-payload')
            if method == 'code':
                assert dev.evaluate("document.querySelector('#student-code').value") == expected['code']
                button('Kodu kopyala')
                copy_result(expected['code'])
            elif method == 'link':
                present('.student-payload a')
                link = dev.evaluate("({href:document.querySelector('.student-payload a').href,rel:document.querySelector('.student-payload a').rel})")
                assert link['href'] == expected['target_url'] and 'noopener' in link['rel'], 'redemption link mismatch'
            elif method == 'qr':
                present('.student-qr svg path')
                assert dev.evaluate("document.querySelector('.student-qr svg path').getAttribute('d').length>100"), 'QR matrix missing'
                assert dev.evaluate("document.querySelector('.student-qr svg').getAttribute('viewBox').split(' ').slice(2).every(v=>Number(v)>=29)"), 'QR dimensions invalid'
            else:
                assert dev.evaluate("document.querySelector('.student-membership').textContent") == expected['membership_label']
            requests = dev.evaluate('window.__realRequests')
            assert any(r['path'] == '/api/student/benefits/' + offers[method] + '/redemption' and r['status'] == 200 for r in requests), 'missing real redemption request'
            layout('redemption-' + method)

        route('benefit/' + code_id, '.student-redemption')
        button('Avantajı paylaş · bağlantıyı kopyala')
        copy_result(base + '/#benefit/' + code_id)
        route('history', '.empty-state')
        assert dev.evaluate("document.querySelector('#student-main').innerText.includes('Henüz doğrulanmış bir kullanım yok')"), 'invented usage history'
        assert not dev.evaluate("!!document.querySelector('#student-main table')"), 'unexpected usage rows'
        route('profile', '.student-profile')
        dev.evaluate("document.querySelector('.student-profile a[href=\"#forgot-password\"]').click()")
        present('form.auth-form #email')
        assert dev.evaluate("location.hash==='#forgot-password'&&document.body.innerText.includes('Parolanı yenileyelim')"), 'profile password link did not open request form'
        assert not dev.errors, 'JavaScript runtime errors: ' + json.dumps(dev.errors)
        (OUT / 'report.json').write_text(json.dumps({'scope': 'Real built frontend, real Go API and isolated PostgreSQL via CI harness', 'checks': checks, 'runtimeErrors': dev.errors, 'requests': dev.evaluate('window.__realRequests')}, ensure_ascii=False, indent=2), encoding='utf-8')
        print(f'Real student UI/API checks passed: {len(checks)} layouts, deep-link login, durable favorites, four redemption methods, profile password route, copy and empty history.')
    finally:
        if dev:
            dev.socket.close()
        process.terminate()
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait(timeout=5)
        profile.cleanup()


if __name__ == '__main__':
    main()
