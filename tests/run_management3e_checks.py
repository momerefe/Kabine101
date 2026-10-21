"""Real 3E UI/API acceptance; launched only by TestManagement3EBrowser.

The Go harness supplies a built application and an isolated PostgreSQL schema.
No request mocking, development database, backend subprocess or frontend server.
"""
import base64
import http.cookiejar
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time
import urllib.error
import urllib.request

from run_browser_checks import DevTools

ROOT = Path(__file__).resolve().parent.parent


def main():
    base = os.environ.get('KOC_BROWSER_URL', '')
    if not base.startswith('http://127.0.0.1:'):
        raise SystemExit('Run through the isolated TestManagement3EBrowser harness.')
    chrome = Path(os.environ.get('KOC_CHROME', 'C:/Program Files/Google/Chrome/Application/chrome.exe'))
    if not chrome.is_file():
        raise SystemExit('Chrome required; set KOC_CHROME to its executable.')
    password = os.environ['KOC_BROWSER_PASSWORD']
    username = os.environ['KOC_BROWSER_USERNAME']
    student = os.environ['KOC_BROWSER_STUDENT']
    student_id = os.environ['KOC_BROWSER_STUDENT_ID']
    benefit = os.environ['KOC_BROWSER_BENEFIT']
    artifacts = ROOT / 'tests' / 'artifacts'
    artifacts.mkdir(exist_ok=True)
    checks = []
    profile = tempfile.TemporaryDirectory(prefix='koc-management3e-', ignore_cleanup_errors=True)
    process = subprocess.Popen([
        str(chrome), '--headless=new', '--disable-gpu', '--no-first-run',
        '--remote-debugging-port=0', '--user-data-dir=' + profile.name, 'about:blank',
    ], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
        creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
    dev = None

    def wait(check, label):
        deadline = time.monotonic() + 25
        while time.monotonic() < deadline:
            try:
                if check():
                    return
            except (OSError, ValueError):
                pass
            time.sleep(.1)
        raise AssertionError(label)

    try:
        port_file = Path(profile.name) / 'DevToolsActivePort'
        wait(lambda: bool(port_file.read_text().splitlines()), 'Chrome startup')
        port = port_file.read_text().splitlines()[0]
        with urllib.request.urlopen(f'http://127.0.0.1:{port}/json/list') as r:
            targets = json.load(r)
        dev = DevTools(next(t['webSocketDebuggerUrl'] for t in targets if t['type'] == 'page'))
        for method in ('Runtime.enable', 'Page.enable', 'Network.enable'):
            dev.call(method)
        dev.call('Emulation.setFocusEmulationEnabled', {'enabled': True})
        # Observe real requests. Every response continues to come from the real API.
        dev.call('Page.addScriptToEvaluateOnNewDocument', {'source': '''
          window.__managementRequests=[];
          const originalFetch=window.fetch.bind(window);
          window.fetch=async(input,options)=>{
            const response=await originalFetch(input,options);
            const path=typeof input==='string'?input:input.url;
            window.__managementRequests.push({path,method:options?.method||input?.method||'GET',status:response.status});
            return response;
          };
        '''})

        def present(selector):
            wait(lambda: dev.evaluate('!!document.querySelector(' + json.dumps(selector) + ')'), 'missing ' + selector)

        def text(value):
            wait(lambda: dev.evaluate('document.body.innerText.includes(' + json.dumps(value) + ')'), 'missing text ' + value)

        def fill(selector, value):
            present(selector)
            dev.evaluate(f'''(()=>{{const e=document.querySelector({json.dumps(selector)});
              const proto=e.tagName==='SELECT'?HTMLSelectElement.prototype:HTMLInputElement.prototype;
              Object.getOwnPropertyDescriptor(proto,'value').set.call(e,{json.dumps(value)});
              e.dispatchEvent(new Event(e.tagName==='SELECT'?'change':'input',{{bubbles:true}}));}})()''')

        def click(label, scope='document'):
            wait(lambda: dev.evaluate(f'''(()=>{{const e=[...{scope}.querySelectorAll('button')].find(e=>e.textContent.trim()==={json.dumps(label)}&&!e.disabled);if(!e)return false;e.click();return true}})()'''), 'button ' + label)

        def nav(section, selector):
            dev.evaluate('location.hash=' + json.dumps('admin/' + section))
            present(selector)
            text({'audit': 'İşlem Günlüğü', 'settings': 'Platform Ayarları', 'analytics': 'Analitik', 'users': 'Kullanıcı ve Rol Yönetimi'}.get(section, 'Yönetim'))

        def api(path, method='GET', body=None):
            return dev.evaluate(f'''(async()=>{{
              const session=await fetch('/api/auth/session').then(r=>r.json());
              const options={{method:{json.dumps(method)},headers:{{'Content-Type':'application/json','X-CSRF-Token':session.csrf_token}}}};
              if(options.method!=='GET')options.body=JSON.stringify({json.dumps(body)});
              const r=await fetch({json.dumps(path)},options);
              return {{status:r.status,body:await r.json()}};
            }})()''')

        def layout(name):
            for width, height in ((320, 740), (390, 844), (1440, 1000)):
                dev.call('Emulation.setDeviceMetricsOverride', {'width': width, 'height': height, 'deviceScaleFactor': 1, 'mobile': width < 500})
                dev.evaluate('new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(()=>resolve(true))))')
                size = dev.evaluate('({viewport:innerWidth,page:document.documentElement.scrollWidth})')
                assert size['page'] <= width + 1 and size['viewport'] <= width + 1, f'page overflow {name} {width}: {size}'
                controls = dev.evaluate('''[...document.querySelectorAll('main input,main select,dialog[open] input')].filter(e=>e.getClientRects().length).map(e=>{const r=e.getBoundingClientRect();return {name:e.name,type:e.type,left:r.left,right:r.right,width:r.width}})''')
                assert all(c['left'] >= -1 and c['right'] <= width + 1 and c['width'] > 0 for c in controls), f'form controls clipped {name} {width}: {controls}'
                for table in dev.evaluate('''[...document.querySelectorAll('.table-scroll')].map(e=>({width:e.clientWidth,scroll:e.scrollWidth,overflow:getComputedStyle(e).overflowX}))'''):
                    assert table['scroll'] <= table['width'] or table['overflow'] in ('auto', 'scroll'), f'inaccessible table {name} {width}'
                screenshot = dev.call('Page.captureScreenshot', {'format': 'png', 'captureBeyondViewport': False})
                (artifacts / f'management3e-{name}-{width}.png').write_bytes(base64.b64decode(screenshot['data']))
                checks.append({'page': name, 'width': width})

        def requests(path, method, status):
            return dev.evaluate(f'''window.__managementRequests.some(r=>r.path.includes({json.dumps(path)})&&r.method==={json.dumps(method)}&&r.status==={status})''')

        dev.call('Emulation.setDeviceMetricsOverride', {'width': 1440, 'height': 1000, 'deviceScaleFactor': 1, 'mobile': False})
        dev.call('Page.navigate', {'url': base + '/#admin/settings'})
        fill('#username', username)
        fill('#password', password)
        dev.evaluate("document.querySelector('form.auth-form').requestSubmit()")
        present('input[name=platform_name]')
        assert dev.evaluate("document.querySelector('input[aria-describedby=domain-help]').readOnly"), 'domain must be read-only'
        fill('input[name=platform_name]', 'Synthetic browser platform')
        fill('input[name=support_contact]', 'browser-support@example.test')
        dev.evaluate("document.querySelector('input[name=show_featured]').click()")
        click('Ayarları kaydet')
        text('Platform ayarları kaydedildi.')
        saved = api('/api/admin/settings')['body']['settings']
        assert saved['platform_name'] == 'Synthetic browser platform' and saved['show_featured'] is False
        assert requests('/api/admin/settings', 'PUT', 200), 'settings form did not save to API'
        layout('settings')
        fill('input[name=platform_name]', 'Preserved local edit')
        concurrent = dict(saved, platform_name='Concurrent server value')
        assert api('/api/admin/settings', 'PUT', concurrent)['status'] == 200
        click('Ayarları kaydet')
        text('Girdiğin bilgiler korundu')
        assert dev.evaluate("document.querySelector('input[name=platform_name]').value") == 'Preserved local edit'
        assert api('/api/admin/settings')['body']['settings']['platform_name'] == 'Concurrent server value'
        click('Güncel ayarları yeniden yükle')
        text('Sunucudaki güncel ayarlar')
        assert dev.evaluate("document.querySelector('input[name=platform_name]').value") == 'Preserved local edit'
        click('Güncel değerleri forma uygula')
        wait(lambda: dev.evaluate("document.querySelector('input[name=platform_name]').value==='Concurrent server value'"), 'explicit conflict resolution')

        nav('audit', '.management-audit input[type=search]')
        fill('.management-audit input[type=search]', 'Synthetic business')
        fill('.management-audit select', 'business.save')
        text('Sayfa 1 / 2')
        present('.management-audit tbody tr')
        first_page = dev.evaluate("document.querySelector('.management-audit tbody').innerText")
        assert 'Synthetic business' in first_page
        click('Sonraki sayfa')
        text('Sayfa 2 / 2')
        assert dev.evaluate("document.querySelectorAll('.management-audit tbody tr').length") == 6
        fill('.management-audit input[type=search]', 'no-such-synthetic-record')
        text('Eşleşen işlem bulunamadı.')
        fill('.management-audit input[type=search]', 'Synthetic business')
        text('Sayfa 1 / 2')
        layout('audit')
        audit = api('/api/admin/audit')['body']
        allowed = {'id', 'action', 'actor_name', 'entity_type', 'entity_id', 'entity_label', 'created_at'}
        assert all(set(event) == allowed for event in audit['events']), 'audit private field leak'
        nav('analytics', '.management-analytics .analytics-totals')
        assert api('/api/admin/analytics')['body']['totals']['views'] == 0
        assert api('/api/admin/analytics')['body']['totals']['favorites'] == 0
        assert not dev.evaluate("document.body.innerText.includes('Sonraki teslim')"), 'analytics placeholder'

        # Student opens actual detail and favorites through rendered controls.
        assert api('/api/auth/logout', 'POST', {})['status'] == 200
        dev.call('Page.navigate', {'url': base + '/#benefit/' + benefit})
        fill('#email', student.split('@', 1)[0])
        fill('#password', password)
        dev.evaluate("document.querySelector('form.auth-form').requestSubmit()")
        present('.student-redemption')
        wait(lambda: requests('/api/student/benefits/' + benefit + '/view', 'POST', 200), 'detail did not record real student view')
        assert api('/api/student/settings')['body']['show_featured'] is False
        dev.evaluate("location.hash='overview'")
        text('Ne keşfetmek istersin?')
        assert not dev.evaluate("document.body.innerText.includes('Bir göz at, sevebilirsin.')"), 'disabled featured recommendations remained visible'
        dev.evaluate("location.hash='discover'")
        selector = '.benefit-card:has(a[href="#benefit/' + benefit + '"]) .favorite-toggle'
        present(selector)
        dev.evaluate('document.querySelector(' + json.dumps(selector) + ').click()')
        wait(lambda: dev.evaluate('document.querySelector(' + json.dumps(selector) + ').getAttribute("aria-pressed")==="true"'), 'favorite form did not update')
        assert any(b['id'] == benefit for b in api('/api/student/favorites')['body']['benefits'])
        assert api('/api/student/benefits/' + benefit + '/view', 'POST', {})['status'] == 200
        assert api('/api/admin/analytics')['status'] in (401, 403), 'student reached analytics'

        # Keep a separate real student session to prove server-side revocation.
        opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))

        def student_request(path, method='GET', body=None, csrf=''):
            headers = {'Content-Type': 'application/json', 'Origin': base, 'X-CSRF-Token': csrf}
            req = urllib.request.Request(base + path, data=json.dumps(body).encode() if body is not None else None, headers=headers, method=method)
            try:
                with opener.open(req) as r:
                    return r.status, json.load(r)
            except urllib.error.HTTPError as error:
                return error.code, json.load(error)

        _, session = student_request('/api/auth/session')
        assert student_request('/api/auth/login', 'POST', {'email': student, 'password': password}, session['csrf_token'])[0] == 200
        assert api('/api/auth/logout', 'POST', {})['status'] == 200
        assert api('/api/auth/admin-login', 'POST', {'username': username, 'password': password})['status'] == 200
        dev.call('Page.navigate', {'url': base + '/#admin/analytics'})
        present('.analytics-totals')
        stats = api('/api/admin/analytics')['body']
        assert stats['totals']['views'] == 1 and stats['totals']['favorites'] == 1, stats
        assert dev.evaluate("[...document.querySelectorAll('.analytics-totals dd')].map(e=>e.textContent)") == ['1', '1', '1']
        assert 'Synthetic 3E coffee' in dev.evaluate("document.querySelector('.management-analytics tbody').innerText")
        layout('analytics')
        nav('users', '.admin-toolbar input[type=search]')
        fill('.admin-toolbar input[type=search]', student)
        present('.admin-account')
        assert dev.evaluate("document.querySelectorAll('.admin-account').length") == 1
        click('Devre dışı bırak', "document.querySelector('.admin-account')")
        present('dialog[open]')
        fill('dialog input[name=current_password]', 'incorrect synthetic password')
        dev.evaluate("document.querySelector('dialog input[type=checkbox]').click()")
        click('Devre dışı bırak', "document.querySelector('dialog[open]')")
        present('dialog[open] [role=alert]')
        assert student_request('/api/student/catalog')[0] == 200, 'bad reauth disabled account'
        fill('dialog input[name=current_password]', password)
        layout('status-dialog')
        click('Devre dışı bırak', "document.querySelector('dialog[open]')")
        wait(lambda: not dev.evaluate("!!document.querySelector('dialog[open]')"), 'status dialog did not finish')
        assert student_request('/api/student/catalog')[0] in (401, 403), 'disabled session retained access'
        records = api('/api/admin/accounts?q=' + student)['body']['accounts']
        assert len(records) == 1 and records[0]['id'] == student_id and records[0]['status'] == 'disabled'
        click('Hesabı etkinleştir', "document.querySelector('.admin-account')")
        fill('dialog input[name=current_password]', password)
        dev.evaluate("document.querySelector('dialog input[type=checkbox]').click()")
        click('Hesabı etkinleştir', "document.querySelector('dialog[open]')")
        wait(lambda: not dev.evaluate("!!document.querySelector('dialog[open]')"), 'reactivation did not finish')
        assert student_request('/api/student/catalog')[0] in (401, 403), 'reactivation revived old session'
        _, session = student_request('/api/auth/session')
        assert student_request('/api/auth/login', 'POST', {'email': student, 'password': password}, session['csrf_token'])[0] == 200
        assert student_request('/api/student/catalog')[0] == 200
        layout('users')
        nav('audit', '.management-audit input[type=search]')
        fill('.management-audit input[type=search]', 'Synthetic 3E Student')
        fill('.management-audit select', 'account.status.disabled')
        present('.management-audit tbody tr')
        assert 'Synthetic 3E Student' in dev.evaluate("document.querySelector('.management-audit tbody').innerText")
        nav('usage', '.management-intro')
        text('Doğrulanmış kullanım kaydı bulunmuyor.')
        assert not dev.errors, f'browser runtime errors: {dev.errors}'
        (artifacts / 'management3e-checks.json').write_text(json.dumps({'checks': checks, 'result': 'passed'}, indent=2), encoding='utf-8')
        print(f'3E browser acceptance passed: {len(checks)} responsive checks and real API workflows.')
    finally:
        if dev is not None:
            dev.socket.close()
        process.terminate()
        try:
            process.wait(timeout=10)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait(timeout=10)
        profile.cleanup()


if __name__ == '__main__':
    main()
