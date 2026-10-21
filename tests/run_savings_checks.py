"""CI browser acceptance for real purchase forms and student savings.

Starts Chrome only; TestSavingsBrowser supplies the built frontend, API, and
isolated schema. No API interception or development services are used.
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
OUT = ROOT / 'tests/artifacts/savings-real'


def main():
    base = os.environ['KOC_BROWSER_URL'].rstrip('/')
    origin = urllib.parse.urlsplit(base)
    assert origin.scheme == 'http' and origin.hostname in ('127.0.0.1', 'localhost', '::1'), 'isolated loopback harness required'
    username = os.environ['KOC_BROWSER_USERNAME']
    student = os.environ['KOC_BROWSER_STUDENT']
    password = os.environ['KOC_BROWSER_PASSWORD']
    benefit = os.environ['KOC_BROWSER_BENEFIT']
    day = os.environ['KOC_BROWSER_PURCHASE_DAY']
    chrome = os.environ.get('KOC_CHROME', '/usr/bin/google-chrome')
    OUT.mkdir(parents=True, exist_ok=True)
    profile = tempfile.TemporaryDirectory(prefix='koc-savings-real-', ignore_cleanup_errors=True)
    process = subprocess.Popen([
        chrome, '--headless=new', '--disable-gpu', '--disable-dev-shm-usage',
        '--no-first-run', '--no-default-browser-check', '--remote-debugging-port=0',
        '--user-data-dir=' + profile.name, 'about:blank',
    ], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
        creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
    dev = None

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
        dev.call('Emulation.setDeviceMetricsOverride', {'width': 1440, 'height': 1000, 'deviceScaleFactor': 1, 'mobile': False})

        def present(selector):
            wait(lambda: dev.evaluate('!!document.querySelector(' + json.dumps(selector) + ')'), 'missing ' + selector)

        def text(value):
            wait(lambda: dev.evaluate('document.body.innerText.includes(' + json.dumps(value) + ')'), 'missing text ' + value)

        def fill(selector, value):
            present(selector)
            wait(lambda: dev.evaluate(f'''(()=>{{const e=document.querySelector({json.dumps(selector)});
                return e.tagName!=='SELECT'||Array.from(e.options).some(o=>o.value==={json.dumps(value)});}})()'''), 'option not ready ' + selector)
            dev.evaluate(f'''(()=>{{const e=document.querySelector({json.dumps(selector)});
                const prototype=e.tagName==='SELECT'?HTMLSelectElement.prototype:e.tagName==='TEXTAREA'?HTMLTextAreaElement.prototype:HTMLInputElement.prototype;
                Object.getOwnPropertyDescriptor(prototype,'value').set.call(e,{json.dumps(value)});
                e.dispatchEvent(new Event(e.tagName==='SELECT'?'change':'input',{{bubbles:true}}));}})()''')

        def click(label, scope='document'):
            assert dev.evaluate(f'''(()=>{{const e=Array.from({scope}.querySelectorAll('button')).find(b=>b.textContent.trim()==={json.dumps(label)});
                if(!e||e.disabled)return false;e.click();return true;}})()'''), 'missing/disabled button ' + label

        def api(path):
            return dev.evaluate(f'''fetch({json.dumps(path)}).then(async r=>{{if(!r.ok)throw new Error('API '+r.status);return r.json();}})''')

        def route(value, selector):
            dev.evaluate('location.hash=' + json.dumps(value))
            present(selector)

        def login(admin):
            route('login/admin' if admin else 'login/student', '#username' if admin else '#email')
            fill('#username' if admin else '#email', username if admin else student)
            fill('#password', password)
            dev.evaluate("document.querySelector('form.auth-form').requestSubmit()")
            present('#desktop-nav' if admin else '.student-panel')
            route('admin/usage' if admin else 'overview', '.purchase-table' if admin else '.student-savings')

        def logout():
            click('Çıkış yap')
            present('#password')
            # A reload also verifies that the next session cannot inherit a
            # previous account's cached totals or history.
            dev.call('Page.reload')
            present('#password')

        def shot(name):
            image = dev.call('Page.captureScreenshot', {'format': 'png', 'captureBeyondViewport': False})
            (OUT / name).write_bytes(base64.b64decode(image['data']))

        def check_student(total, count):
            login(False)
            text('Bu ayki tasarrufun')
            text('Yönetici onaylı')
            wait(lambda: dev.evaluate("document.querySelector('.student-savings-total')?.textContent.trim()===" + json.dumps('₺' + total.replace('.', ','))), 'rendered savings total ' + total)
            text(f'Yönetici onaylı · {count} alışveriş')
            d = api('/api/student/savings')
            assert d['total_savings'] == total and d['purchase_count'] == count, d
            assert len(d['businesses']) == (1 if count else 0), d
            if count:
                assert d['businesses'][0]['business_name'] == 'Browser savings business'
                assert d['businesses'][0]['savings_amount'] == total
                assert dev.evaluate("document.querySelector('.student-savings-breakdown tbody').innerText.includes('Browser savings business')")
            route('history', '#student-main h1')
            text('Alışveriş Geçmişi')
            history = api('/api/student/purchases')
            assert len(history['purchases']) == count, history
            if count:
                present('.student-purchase')
                assert dev.evaluate("document.querySelectorAll('.student-purchase').length") == count
                assert dev.evaluate("document.querySelector('.student-purchase time').dateTime") == day
                assert dev.evaluate("document.querySelector('.student-purchase').innerText.includes('Browser savings business')")
                assert dev.evaluate("document.querySelector('.student-purchase').innerText.includes('Purchase snapshot benefit')")
                assert dev.evaluate("Array.from(document.querySelectorAll('.student-purchase dd')).map(e=>e.textContent)") == ['₺100,00', '₺80,00', '₺20,00']
                dev.call('Page.reload')
                present('.student-purchase')
                assert len(api('/api/student/purchases')['purchases']) == 1
                for width in (390, 320):
                    dev.call('Emulation.setDeviceMetricsOverride', {'width': width, 'height': 950, 'deviceScaleFactor': 1, 'mobile': True})
                    dev.evaluate('new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(()=>resolve(true))))')
                    assert dev.evaluate('document.documentElement.scrollWidth<=innerWidth'), 'history overflow ' + str(width)
                shot('student-approved-history-320.png')
                dev.call('Emulation.setDeviceMetricsOverride', {'width': 1440, 'height': 1000, 'deviceScaleFactor': 1, 'mobile': False})
            else:
                text('Henüz yönetici onaylı alışverişin yok.')
                assert not dev.evaluate("!!document.querySelector('.student-purchase')")
            assert dev.evaluate("fetch('/api/admin/purchases').then(r=>r.status)") == 403, 'student read administrative purchases'
            body = dev.evaluate('document.body.innerText')
            assert 'BROWSER-RECEIPT' not in body and 'Browser revoke reason' not in body
            logout()

        def review(label, reason=None):
            click(label)
            present('dialog[open]')
            assert dev.evaluate("document.querySelector('dialog button[type=submit]').disabled"), 'review confirmation bypassed'
            if reason:
                fill('dialog textarea[name=reason]', reason)
            dev.evaluate("document.querySelector('dialog input[type=checkbox]').click()")
            click(label, "document.querySelector('dialog[open]')")
            wait(lambda: not dev.evaluate("!!document.querySelector('dialog[open]')"), 'review dialog did not close')

        dev.call('Page.navigate', {'url': base + '/#login/admin'})
        present('#username')
        fill('#username', username)
        fill('#password', password)
        dev.evaluate("document.querySelector('form.auth-form').requestSubmit()")
        present('#desktop-nav')
        route('admin/usage', '.heading-actions')
        text('Henüz alışveriş kaydı yok.')
        click('Alışveriş ekle +')
        fill('[name=student_email]', student)
        fill('[name=benefit_id]', benefit)
        fill('[name=purchase_date]', day)
        fill('[name=reference]', 'BROWSER-RECEIPT')
        # Comma entry exercises the Turkish form conversion before the API call.
        fill('[name=original_amount]', '100,00')
        fill('[name=paid_amount]', '80,00')
        click('Onay bekleyen kayıt oluştur')
        wait(lambda: not dev.evaluate("!!document.querySelector('.purchase-create')"), 'purchase form did not close')
        present('.purchase-table')
        text('Onay bekliyor')
        purchases = api('/api/admin/purchases')['purchases']
        assert len(purchases) == 1 and purchases[0]['status'] == 'pending', purchases
        purchase_id = purchases[0]['id']
        assert purchases[0]['original_amount'] == '100.00' and purchases[0]['paid_amount'] == '80.00'
        logout()
        check_student('0.00', 0)

        login(True)
        review('Alışverişi onayla')
        text('Onaylandı')
        purchases = api('/api/admin/purchases')['purchases']
        assert len(purchases) == 1 and purchases[0]['id'] == purchase_id and purchases[0]['status'] == 'approved'
        shot('admin-approved.png')
        logout()
        check_student('20.00', 1)

        login(True)
        review('Onayı geri al', 'Browser revoke reason')
        text('Onay geri alındı')
        purchases = api('/api/admin/purchases')['purchases']
        assert len(purchases) == 1 and purchases[0]['status'] == 'revoked'
        logout()
        check_student('0.00', 0)
        assert not dev.errors, dev.errors
        print('Savings browser acceptance passed: real purchase creation, pending exclusion, reviewed approval, persisted student totals/history, responsive history, revocation removal and access isolation.')
    except Exception:
        if dev:
            (OUT / 'failure.txt').write_text(dev.evaluate('document.body.innerText'), encoding='utf-8')
            shot('failure.png')
        raise
    finally:
        if dev:
            dev.socket.close()
        process.terminate()
        process.wait(timeout=15)
        profile.cleanup()


if __name__ == '__main__':
    main()
