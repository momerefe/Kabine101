"""Browser acceptance launched by TestManagementBrowser against its isolated DB.

Build first, then set KOC_BROWSER_TEST=1 and run:
go test -tags=integration ./internal/management -run TestManagementBrowser -v
No development account, sponsor record or service is modified.
"""
import base64
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time
import urllib.request

from run_browser_checks import DevTools

ROOT = Path(__file__).resolve().parent.parent


def main():
    base = os.environ.get('KOC_BROWSER_URL', '')
    if not base.startswith('http://127.0.0.1:'):
        raise SystemExit('Run this check through the isolated Go browser test.')
    chrome = Path(os.environ.get('KOC_CHROME', 'C:/Program Files/Google/Chrome/Application/chrome.exe'))
    if not chrome.exists():
        raise SystemExit('Chrome is required; set KOC_CHROME to its executable.')
    profile = tempfile.TemporaryDirectory(prefix='koc-management-browser-', ignore_cleanup_errors=True)
    process = subprocess.Popen([str(chrome), '--headless=new', '--disable-gpu', '--no-first-run', '--remote-debugging-port=0', '--user-data-dir='+profile.name, 'about:blank'], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
    dev = None
    def wait(check, label):
        deadline = time.monotonic()+20
        while time.monotonic()<deadline:
            try:
                if check(): return
            except (OSError, ValueError): pass
            time.sleep(.1)
        raise AssertionError(label)
    try:
        portfile = Path(profile.name)/'DevToolsActivePort'
        wait(lambda: bool(portfile.read_text().splitlines()), 'Chrome startup')
        port = portfile.read_text().splitlines()[0]
        with urllib.request.urlopen(f'http://127.0.0.1:{port}/json/list') as r: targets = json.load(r)
        dev = DevTools(next(t['webSocketDebuggerUrl'] for t in targets if t['type']=='page'))
        for method in ('Runtime.enable','Page.enable','Network.enable'): dev.call(method)
        dev.call('Emulation.setFocusEmulationEnabled', {'enabled':True})
        def present(selector):
            wait(lambda: dev.evaluate(f'!!document.querySelector({json.dumps(selector)})'), 'missing '+selector)
        def text(value):
            wait(lambda: dev.evaluate(f'document.body.innerText.includes({json.dumps(value)})'), 'missing text '+value)
        def fill(selector, value):
            present(selector)
            dev.evaluate(f"""(()=>{{const e=document.querySelector({json.dumps(selector)});const proto=e.tagName==='SELECT'?HTMLSelectElement.prototype:e.tagName==='TEXTAREA'?HTMLTextAreaElement.prototype:HTMLInputElement.prototype;Object.getOwnPropertyDescriptor(proto,'value').set.call(e,{json.dumps(value)});e.dispatchEvent(new Event(e.tagName==='SELECT'?'change':'input',{{bubbles:true}}));}})()""")
        def click(label, scope='document', prefix=False):
            # Keep the button lookup readable; disabled controls must never pass.
            predicate = f'b.textContent.trim().startsWith({json.dumps(label)})' if prefix else f'b.textContent.trim()==={json.dumps(label)}'
            assert dev.evaluate(f"(()=>{{const e=Array.from({scope}.querySelectorAll('button')).find(b=>{predicate});if(!e||e.disabled)return false;e.click();return true}})()"), 'button missing/disabled '+label
        def nav(section):
            dev.evaluate('location.hash='+json.dumps('admin/'+section))
            present(f'#desktop-nav a[href="#admin/{section}"][aria-current="page"]')
        def confirm():
            present('dialog[open]'); click('Onayla', "document.querySelector('dialog[open]')")
            wait(lambda: not dev.evaluate("!!document.querySelector('dialog[open]')"), 'confirmation not closed')
        def saved():
            wait(lambda: not dev.evaluate("!!document.querySelector('.management-editor')"), 'editor did not close after save')
        def api(path):
            return dev.evaluate(f"fetch({json.dumps(path)}).then(r=>r.json())")
        def viewport(width):
            dev.call('Emulation.setDeviceMetricsOverride', {'width':width,'height':950,'deviceScaleFactor':1,'mobile':width<500})
            assert dev.evaluate('document.documentElement.scrollWidth<=innerWidth'), f'overflow {width}'
        def shot(name):
            output=ROOT/'tests'/'artifacts'; output.mkdir(exist_ok=True)
            image=dev.call('Page.captureScreenshot', {'format':'png','captureBeyondViewport':False})
            (output/name).write_bytes(base64.b64decode(image['data']))

        viewport(1440)
        dev.call('Page.navigate', {'url':base+'/#admin/businesses'})
        fill('#username', os.environ['KOC_BROWSER_USERNAME']); fill('#password', os.environ['KOC_BROWSER_PASSWORD'])
        dev.evaluate("document.querySelector('form.auth-form').requestSubmit()")
        present('#desktop-nav')
        present('#desktop-nav a[href="#admin/businesses"][aria-current="page"]')
        assert dev.evaluate("document.querySelectorAll('#desktop-nav a').length===12")
        for section in ('overview','partners','businesses','benefits','students','published','archived','usage','analytics','users','audit','settings'):
            nav(section)
        nav('businesses'); text('Eşleşen kayıt bulunamadı')
        click('İşletme ekle', prefix=True)
        fill('#edit-name','Tarayıcı Test Kafesi'); fill('#edit-description','Yalnızca izole test kaydı.')
        fill('#edit-partnership_status','active'); fill('#edit-contact_person','Test Yetkilisi')
        fill('#edit-notes','Gerçek sponsor değildir.')
        click('Kaydet'); confirm(); saved(); text('Tarayıcı Test Kafesi')
        fill('.admin-toolbar input[type=search]','eşleşmeyen-arama'); text('Eşleşen kayıt bulunamadı')
        fill('.admin-toolbar input[type=search]',''); text('Tarayıcı Test Kafesi')
        business = api('/api/admin/businesses')['businesses'][0]
        nav('benefits'); click('Avantaj taslağı ekle', prefix=True)
        fill('#edit-business_id', business['id']); fill('#edit-title','Test kahve avantajı')
        # Drafts may be incomplete; publishing requires a separate reviewed form.
        click('Taslak olarak kaydet'); confirm(); saved()
        click('Düzenle'); fill('#edit-description','Test için kahvelerde indirim.')
        fill('#edit-perk','%10 indirim'); fill('#edit-conditions','Öğrenci kimliği gösterilir.\nYalnızca test şubesinde.')
        fill('#edit-instructions','Kasada kodu göster.'); fill('#edit-code','SYNTHETIC-COFFEE')
        fill('#edit-products','Kahveler'); fill('#edit-exceptions','Paketli ürünler hariç.')
        fill('#edit-minimum_spend','25.50')
        assert dev.evaluate("Array.from(document.querySelectorAll('.management-editor button')).find(b=>b.textContent.includes('Koşulları incele')).disabled"), 'unsaved conditions could be published'
        for width in (390,320): viewport(width)
        shot('management-form-320.png')
        viewport(1440); click('Taslak olarak kaydet'); confirm(); saved()
        click('Düzenle'); click('Koşulları incele ve yayımla')
        present('dialog[open]'); dev.evaluate("document.querySelector('dialog input[type=checkbox]').click()")
        confirm(); saved()
        benefit=api('/api/admin/benefits')['benefits'][0]
        assert benefit['status']=='active' and benefit['minimum_spend']=='25.50'
        nav('published'); text('Test kahve avantajı'); click('Detay')
        text('Paketli ürünler hariç.'); shot('management-detail-desktop.png')
        click('Düzenle'); click('Askıya al'); present('dialog textarea')
        fill('dialog textarea','Test için geçici askı.'); confirm(); saved()
        nav('archived'); text('Test için geçici askı.')
        click('Düzenle'); click('Koşulları incele ve yayımla')
        dev.evaluate("document.querySelector('dialog input[type=checkbox]').click()")
        confirm(); saved()
        nav('partners'); click('Düzenle'); fill('#edit-partnership_status','ended')
        click('Kaydet'); present('dialog[open]'); text('bağlı yayımlanmış avantajları askıya alır'); confirm(); saved()
        assert api('/api/admin/benefits')['benefits'][0]['status']=='suspended'
        click('Düzenle'); fill('#edit-partnership_status','active'); click('Kaydet'); confirm(); saved()
        assert api('/api/admin/benefits')['benefits'][0]['status']=='suspended', 'reopening republished offer'

        # A second writer must not silently overwrite the open form.
        click('Düzenle'); fill('#edit-name','Korunması gereken yerel düzenleme')
        dev.evaluate("""(async()=>{const session=await fetch('/api/auth/session').then(r=>r.json());const {businesses}=await fetch('/api/admin/businesses').then(r=>r.json());const {id,created_at,updated_at,...body}=businesses[0];body.notes='Concurrent server change';const r=await fetch('/api/admin/businesses/'+id,{method:'PUT',headers:{'Content-Type':'application/json','X-CSRF-Token':session.csrf_token},body:JSON.stringify(body)});if(!r.ok)throw new Error('second writer failed')})()""")
        click('Kaydet'); confirm(); text('Girdiğin bilgiler korundu')
        assert dev.evaluate("document.querySelector('#edit-name').value==='Korunması gereken yerel düzenleme'")
        click('Vazgeç', "document.querySelector('.management-editor')")

        # Desktop rail expands without moving content; mobile closes on selection.
        nav('businesses'); dev.evaluate('document.activeElement.blur()')
        dev.call('Input.dispatchMouseEvent', {'type':'mouseMoved','x':600,'y':100}); time.sleep(.25)
        narrow=dev.evaluate("document.querySelector('.sidebar').getBoundingClientRect().width")
        left=dev.evaluate("document.querySelector('.app-body').getBoundingClientRect().left")
        dev.call('Input.dispatchMouseEvent', {'type':'mouseMoved','x':35,'y':250}); time.sleep(.25)
        assert dev.evaluate("document.querySelector('.sidebar').getBoundingClientRect().width")>narrow
        assert dev.evaluate("document.querySelector('.app-body').getBoundingClientRect().left") == left
        dev.call('Input.dispatchMouseEvent', {'type':'mouseMoved','x':600,'y':100}); time.sleep(.25)
        assert dev.evaluate("document.querySelector('.sidebar').getBoundingClientRect().width") == narrow
        shot('management-businesses-desktop.png')
        for width in (390,320):
            nav('businesses')
            viewport(width); click('☰ Menü'); present('.admin-menu-open')
            dev.evaluate("document.querySelector('#desktop-nav a[href=\"#admin/benefits\"]').click()")
            wait(lambda:not dev.evaluate("document.body.classList.contains('admin-menu-open')"),'mobile menu did not close')
            viewport(width); click('☰ Menü')
            dev.call('Input.dispatchKeyEvent', {'type':'keyDown','key':'Escape','code':'Escape','windowsVirtualKeyCode':27})
            wait(lambda:not dev.evaluate("document.body.classList.contains('admin-menu-open')"),'Escape did not close menu')
            assert dev.evaluate("document.activeElement.classList.contains('admin-menu-button')")
        shot('management-benefits-320.png')
        dev.call('Emulation.setEmulatedMedia', {'features':[{'name':'prefers-reduced-motion','value':'reduce'}]})
        nav('businesses'); viewport(320)
        dev.call('Network.setBlockedURLs', {'urls':['*/api/admin/businesses*']})
        dev.call('Page.reload'); text('İşlem tamamlanamadı')
        dev.call('Network.setBlockedURLs', {'urls':[]}); click('Yeniden dene'); text('Tarayıcı Test Kafesi')
        click('Çıkış yap'); present('#password')
        dev.evaluate("location.hash='login/student'")
        fill('#email', os.environ['KOC_BROWSER_STUDENT']); fill('#password', os.environ['KOC_BROWSER_PASSWORD'])
        dev.evaluate("document.querySelector('form.auth-form').requestSubmit()")
        present('.account-details')
        dev.evaluate("location.hash='admin/businesses'"); text('yönetici yetkisi gerekir')
        assert not dev.evaluate("!!document.querySelector('#desktop-nav')"), 'student saw administration'
        click('Çıkış yap'); present('#password')
        assert not dev.errors, 'browser runtime error'
        print('Management browser acceptance passed: 12 sections, real isolated CRUD/publish/suspend, partial drafts, search, conflict preservation, partnership cascade, desktop rail, 390/320 px, reduced motion, network retry, student access boundary and logout.')
    except Exception:
        if dev:
            output=ROOT/'tests'/'artifacts'; output.mkdir(exist_ok=True)
            (output/'management-failure.txt').write_text(dev.evaluate('document.body.innerText'),encoding='utf-8')
        raise
    finally:
        if dev: dev.socket.close()
        process.terminate(); process.wait(timeout=15); profile.cleanup()


if __name__=='__main__': main()
