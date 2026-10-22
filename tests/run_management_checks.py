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
    student_dev = None
    student_process = None
    student_profile = None
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
        dev.call('Page.addScriptToEvaluateOnNewDocument', {'source': '''
            window.__publishRequests=0;
            window.__managementWrites=[];
            const originalFetch=window.fetch.bind(window);
            window.fetch=async(input,options)=>{
                const path=new URL(typeof input==='string'?input:input.url,location.href).pathname;
                const method=options?.method||(input instanceof Request?input.method:'GET');
                if(path.endsWith('/publish')&&method==='POST')window.__publishRequests++;
                const entry={path,method};
                if(path.startsWith('/api/admin/')&&['PUT','POST'].includes(method))window.__managementWrites.push(entry);
                const response=await originalFetch(input,options);
                entry.status=response.status;
                return response;
            };
        '''})
        def present(selector, page=None):
            page = page or dev
            wait(lambda: page.evaluate(f'!!document.querySelector({json.dumps(selector)})'), 'missing '+selector)
        def text(value, page=None):
            page = page or dev
            wait(lambda: page.evaluate(f'document.body.innerText.includes({json.dumps(value)})'), 'missing text '+value)
        def fill(selector, value, page=None):
            page = page or dev
            present(selector, page)
            wait(lambda:page.evaluate(f"(()=>{{const e=document.querySelector({json.dumps(selector)});return e.tagName!=='SELECT'||Array.from(e.options).some(o=>o.value==={json.dumps(value)})}})()"),'select option not loaded '+selector)
            page.evaluate(f"""(()=>{{const e=document.querySelector({json.dumps(selector)});const proto=e.tagName==='SELECT'?HTMLSelectElement.prototype:e.tagName==='TEXTAREA'?HTMLTextAreaElement.prototype:HTMLInputElement.prototype;Object.getOwnPropertyDescriptor(proto,'value').set.call(e,{json.dumps(value)});e.dispatchEvent(new Event(e.tagName==='SELECT'?'change':'input',{{bubbles:true}}));}})()""")
        def click(label, scope='document', prefix=False, page=None):
            page = page or dev
            # Keep the button lookup readable; disabled controls must never pass.
            predicate = f'b.textContent.trim().startsWith({json.dumps(label)})' if prefix else f'b.textContent.trim()==={json.dumps(label)}'
            wait(lambda: page.evaluate(f"(()=>{{const root={scope};if(!root)return false;const e=Array.from(root.querySelectorAll('button')).find(b=>{predicate});if(!e||e.disabled)return false;e.click();return true}})()"), 'button missing/disabled '+label)
        def nav(section):
            dev.evaluate('location.hash='+json.dumps('admin/'+section))
            view = section.split('?', 1)[0]
            active = 'benefits' if view in ('businesses','benefits','published','archived') else view
            present(f'#desktop-nav a[href="#admin/{active}"][aria-current="page"]')
            if active == 'benefits':
                present(f'nav[aria-label="Sponsorluk bölümleri"] a[href="#admin/{view}"][aria-current="page"]')
                assert dev.evaluate("document.querySelectorAll('main h1').length===1&&document.querySelector('main h1').textContent==='Sponsorluk Yönetimi'"), 'sponsorship views must share one heading'
        def confirm():
            present('dialog[open]'); click('Onayla', "document.querySelector('dialog[open]')")
            wait(lambda: not dev.evaluate("!!document.querySelector('dialog[open]')"), 'confirmation not closed')
        def saved():
            wait(lambda: not dev.evaluate("!!document.querySelector('.management-editor')"), 'editor did not close after save')
        def required_marks(required, optional):
            for key in required:
                selector = '#edit-'+key
                present(selector)
                assert dev.evaluate(f"""(()=>{{const label=document.querySelector({json.dumps(selector)}).closest('label');const mark=label.querySelector('.required-mark');if(!mark||mark.getAttribute('aria-hidden')!=='true'||mark.textContent.trim()!=='*')return false;const rgb=getComputedStyle(mark).color.match(/[\\d.]+/g).map(Number);return rgb[0]>rgb[1]+30&&rgb[0]>rgb[2]+30}})()"""), 'required field lacks a red visible star: '+key
                assert dev.evaluate(f"(()=>{{const field=document.querySelector({json.dumps(selector)});return field.required||field.getAttribute('aria-required')==='true'}})()"), 'required field lacks an accessible requirement: '+key
            for key in optional:
                selector = '#edit-'+key
                present(selector)
                assert dev.evaluate(f"!document.querySelector({json.dumps(selector)}).closest('label').querySelector('.required-mark')"), 'optional field is marked required: '+key
        def blocked_publication(missing):
            before = dev.evaluate('window.__publishRequests')
            publish()
            present('.publication-readiness.has-errors')
            wait(lambda:dev.evaluate(f'document.querySelector(".publication-readiness").textContent.includes({json.dumps(missing)})'), 'missing publication guidance: '+missing)
            assert not dev.evaluate("!!document.querySelector('dialog[open]')"), 'incomplete publication opened a confirmation'
            assert dev.evaluate('window.__publishRequests') == before, 'incomplete publication reached the API'
        def red_button(label, scope='document'):
            assert dev.evaluate(f"""(()=>{{const root={scope};const button=root&&[...root.querySelectorAll('button')].find(b=>b.textContent.trim()==={json.dumps(label)});if(!button||button.disabled)return false;const rgb=getComputedStyle(button).backgroundColor.match(/[\\d.]+/g).map(Number);return rgb[0]>rgb[1]+30&&rgb[0]>rgb[2]+30}})()"""), 'publication action must be enabled and red: '+label
        def publish():
            present('.management-editor')
            label = dev.evaluate("[...document.querySelectorAll('.management-editor button')].some(b=>b.textContent.trim()==='Kaydet ve yayımla')?'Kaydet ve yayımla':'Yayınla'")
            red_button(label, "document.querySelector('.management-editor')")
            click(label, "document.querySelector('.management-editor')")
        def api(path):
            return dev.evaluate(f"fetch({json.dumps(path)}).then(r=>r.json())")
        def viewport(width, page=None):
            page = page or dev
            page.call('Emulation.setDeviceMetricsOverride', {'width':width,'height':950,'deviceScaleFactor':1,'mobile':width<500})
            # Let resize/media listeners and React commit before interacting.
            page.evaluate('new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(()=>resolve(true))))')
            assert page.evaluate('document.documentElement.scrollWidth<=innerWidth'), f'overflow {width}'
        def shot(name, page=None):
            page = page or dev
            output=ROOT/'tests'/'artifacts'; output.mkdir(exist_ok=True)
            image=page.call('Page.captureScreenshot', {'format':'png','captureBeyondViewport':False})
            (output/name).write_bytes(base64.b64decode(image['data']))

        viewport(1440)
        dev.call('Page.navigate', {'url':base+'/#admin/businesses'})
        fill('#username', os.environ['KOC_BROWSER_USERNAME']); fill('#password', os.environ['KOC_BROWSER_PASSWORD'])
        dev.evaluate("document.querySelector('form.auth-form').requestSubmit()")
        present('#desktop-nav')
        present('#desktop-nav a[href="#admin/benefits"][aria-current="page"]')
        sections = ('overview','partners','benefits','students','usage','analytics','users','audit','settings')
        tabs = {'businesses':'İşletmeler','benefits':'Tüm avantajlar','published':'Yayındakiler','archived':'Süresi dolan / askıdaki'}
        assert dev.evaluate("document.querySelectorAll('#desktop-nav a').length===9")
        for width in (320,390,1440):
            viewport(width)
            for section in sections:
                if width < 500:
                    click('☰ Menü'); present('.admin-menu-open')
                selector = f'#desktop-nav a[href="#admin/{section}"]'
                present(selector)
                wait(lambda: dev.evaluate(f"(()=>{{const r=document.querySelector({json.dumps(selector)}).getBoundingClientRect();return r.width>0&&r.left>=0&&r.left<innerWidth}})()"), 'sidebar destination is unreachable')
                dev.evaluate(f'document.querySelector({json.dumps(selector)}).click()')
                present(selector+'[aria-current="page"]')
                if width < 500:
                    wait(lambda:not dev.evaluate("document.body.classList.contains('admin-menu-open')"),'mobile selection did not close menu')
            nav('benefits')
            assert dev.evaluate("document.querySelectorAll('nav[aria-label=\"Sponsorluk bölümleri\"] a').length===4")
            for section, label in tabs.items():
                selector=f'nav[aria-label="Sponsorluk bölümleri"] a[href="#admin/{section}"]'
                assert dev.evaluate(f'document.querySelector({json.dumps(selector)}).textContent.trim()==={json.dumps(label)}')
                dev.evaluate(f'document.querySelector({json.dumps(selector)}).click()')
                present(selector+'[aria-current="page"]')
                viewport(width)
                assert dev.evaluate("document.querySelector('main h1').textContent==='Sponsorluk Yönetimi'")
                assert dev.evaluate(f"(()=>{{const r=document.querySelector({json.dumps(selector)}).getBoundingClientRect();return r.width>0&&r.left>=0&&r.right<=innerWidth+1}})()"), 'sponsorship tab is clipped'
            shot(f'management-sponsorship-tabs-{width}.png')
        for section, status in (('businesses','active'),('benefits','draft'),('published','active'),('archived','suspended')):
            nav(section+'?status='+status)
            wait(lambda:dev.evaluate(f'document.querySelectorAll(".admin-toolbar select")[1]?.value==={json.dumps(status)}'),'legacy filtered link lost its status')
        viewport(1440)
        nav('businesses'); text('Eşleşen kayıt bulunamadı')
        click('İşletme ekle', prefix=True)
        required_marks(('name','category','partnership_status'), ('description','image_path','contact_person','contact_info','last_contact','notes','branches'))
        fill('#edit-name','Tarayıcı Test Kafesi'); fill('#edit-description','Yalnızca izole test kaydı.')
        fill('#edit-partnership_status','active'); fill('#edit-contact_person','Test Yetkilisi')
        fill('#edit-notes','Gerçek sponsor değildir.')
        click('Kaydet'); confirm(); saved(); text('Tarayıcı Test Kafesi')
        fill('.admin-toolbar input[type=search]','eşleşmeyen-arama'); text('Eşleşen kayıt bulunamadı')
        fill('.admin-toolbar input[type=search]',''); text('Tarayıcı Test Kafesi')
        business = api('/api/admin/businesses')['businesses'][0]
        red_button('Yayınla', "document.querySelector('.admin-table tbody tr')")
        click('Yayınla', "document.querySelector('.admin-table tbody tr')")
        present('.business-publication')
        assert api('/api/admin/benefits')['benefits']==[], 'opening publication created or published an advantage'
        click('Avantaj oluştur', "document.querySelector('.business-publication')")
        present('#edit-business_id')
        assert dev.evaluate("document.querySelector('#edit-business_id').value") == business['id'], 'business publication did not preselect its business'
        required_marks(('business_id','title','method','description','perk','conditions','instructions','code'), ('products','branches','exceptions','minimum_spend','valid_from','valid_until','verification_requirement','support_contact'))
        assert dev.evaluate("document.querySelector('#edit-title').required&&!document.querySelector('#edit-perk').required"), 'publication requirements must still permit incomplete draft saves'
        fill('#edit-method','link')
        required_marks(('target_url',), ('products','support_contact'))
        fill('#edit-method','code')
        fill('#edit-business_id', business['id']); fill('#edit-title','Test kahve avantajı')
        # Drafts may be incomplete; publishing requires a separate reviewed form.
        click('Taslak olarak kaydet'); confirm(); saved()
        benefit=api('/api/admin/benefits')['benefits'][0]
        nav('benefits')

        # A separate Chrome profile keeps a normal student's HTTP session alive
        # while the administrator publishes and suspends the same isolated offer.
        student_profile = tempfile.TemporaryDirectory(prefix='koc-management-student-', ignore_cleanup_errors=True)
        student_process = subprocess.Popen([str(chrome), '--headless=new', '--disable-gpu', '--no-first-run', '--remote-debugging-port=0', '--user-data-dir='+student_profile.name, 'about:blank'], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
        student_portfile = Path(student_profile.name)/'DevToolsActivePort'
        wait(lambda: bool(student_portfile.read_text().splitlines()), 'student Chrome startup')
        student_port = student_portfile.read_text().splitlines()[0]
        with urllib.request.urlopen(f'http://127.0.0.1:{student_port}/json/list') as r: student_targets = json.load(r)
        student_dev = DevTools(next(t['webSocketDebuggerUrl'] for t in student_targets if t['type']=='page'))
        for method in ('Runtime.enable','Page.enable','Network.enable'): student_dev.call(method)
        student_dev.call('Page.addScriptToEvaluateOnNewDocument', {'source': '''
            window.__catalogResponses=0;
            const originalFetch=window.fetch.bind(window);
            window.fetch=async(input,options)=>{
                const response=await originalFetch(input,options);
                const path=new URL(typeof input==='string'?input:input.url,location.href).pathname;
                if(path==='/api/student/catalog'&&response.ok)window.__catalogResponses++;
                return response;
            };
        '''})
        viewport(1440, student_dev)
        student_dev.call('Page.navigate', {'url':base+'/#discover'})
        fill('#email', os.environ['KOC_BROWSER_STUDENT'].split('@',1)[0], student_dev)
        fill('#password', os.environ['KOC_BROWSER_PASSWORD'], student_dev)
        student_dev.evaluate("document.querySelector('form.auth-form').requestSubmit()")
        present('.student-panel .catalog-toolbar', student_dev)

        def student_api(path):
            return student_dev.evaluate(f"(async()=>{{const r=await fetch({json.dumps(path)});return {{status:r.status,body:await r.json()}}}})()")

        student_session = student_api('/api/auth/session')['body']['user']
        assert student_session['role']=='student' and student_session['id']==os.environ['KOC_BROWSER_STUDENT_ID'], 'publication check must use the ordinary student account'
        assert api('/api/auth/session')['user']['role']=='admin', 'student profile replaced the administrator session'

        def student_catalog(visible):
            student_dev.evaluate("location.hash='discover'")
            present('.catalog-toolbar', student_dev)
            before = student_dev.evaluate('window.__catalogResponses')
            click('Avantajları yenile', page=student_dev)
            wait(lambda:student_dev.evaluate('window.__catalogResponses')>before, 'student refresh did not reach the real catalogue API')
            result = student_api('/api/student/catalog')
            assert result['status']==200
            assert any(b['id']==benefit['id'] for b in result['body']['benefits']) is visible, 'student catalogue publication boundary'
            selector='.benefit-card a[href="#benefit/'+benefit['id']+'"]'
            wait(lambda:student_dev.evaluate(f'!!document.querySelector({json.dumps(selector)})') is visible, 'student rendered catalogue publication boundary')

        student_catalog(False)
        assert student_api('/api/student/benefits/'+benefit['id'])['status']==404, 'student opened an unpublished draft'
        click('Düzenle'); fill('#edit-description','Test için kahvelerde indirim.')
        fill('#edit-conditions','Öğrenci kimliği gösterilir.\nYalnızca test şubesinde.')
        fill('#edit-instructions','Kasada kodu göster.'); fill('#edit-code','SYNTHETIC-COFFEE')
        fill('#edit-products','Kahveler'); fill('#edit-exceptions','Paketli ürünler hariç.')
        fill('#edit-minimum_spend','25.50')
        red_button('Kaydet ve yayımla', "document.querySelector('.management-editor')")
        for width in (390,320): viewport(width)
        shot('management-form-320.png')
        viewport(1440); click('Taslak olarak kaydet'); confirm(); saved()
        incomplete = api('/api/admin/benefits')['benefits'][0]
        assert incomplete['perk']=='' and incomplete['publication_status']=='draft', 'incomplete draft did not remain saveable'
        click('Düzenle')
        blocked_publication('İndirim / ayrıcalık')
        present('#edit-perk[aria-invalid="true"]')
        assert dev.evaluate("document.activeElement.id==='edit-perk'"), 'missing publication field did not receive focus'
        assert api('/api/admin/benefits')['benefits'][0]['publication_status']=='draft', 'failed local publication changed the draft'
        for width in (320,390,1440):
            viewport(width)
            shot(f'management-publication-missing-perk-{width}.png')
        fill('#edit-perk','%10 indirim')
        writes_before = dev.evaluate('window.__managementWrites.length')
        publish()
        present('dialog[open]'); dev.evaluate("document.querySelector('dialog input[type=checkbox]').click()")
        confirm(); saved()
        operations=dev.evaluate('window.__managementWrites.slice('+str(writes_before)+')')
        assert [(op['method'],op.get('status')) for op in operations]==[('PUT',200),('POST',200)], 'dirty publication must save and publish after one confirmation'
        benefit=api('/api/admin/benefits')['benefits'][0]
        assert benefit['status']=='active' and benefit['minimum_spend']=='25.50'
        student_catalog(True)
        detail = student_api('/api/student/benefits/'+benefit['id'])
        assert detail['status']==200 and detail['body']['benefit']['status']=='active'
        assert not any(key in detail['body']['benefit'] for key in ('code','target_url','qr_content','membership_label')), 'catalogue detail leaked redemption secrets'
        student_dev.evaluate('document.querySelector('+json.dumps('a[href="#benefit/'+benefit['id']+'"]')+').click()')
        present('.student-redemption', student_dev)
        text('Test kahve avantajı', student_dev)
        text('Paketli ürünler hariç.', student_dev)
        for width in (320,390,1440):
            viewport(width, student_dev)
            shot(f'management-student-published-{width}.png', student_dev)
        student_dev.evaluate("location.hash='discover'")
        present('.catalog-toolbar', student_dev)
        nav('published'); text('Test kahve avantajı'); click('Detay')
        text('Paketli ürünler hariç.'); shot('management-detail-desktop.png')
        click('Düzenle'); click('Askıya al'); present('dialog textarea')
        fill('dialog textarea','Test için geçici askı.'); confirm(); saved()
        student_catalog(False)
        suspended = student_api('/api/student/benefits/'+benefit['id'])
        assert suspended['status']==404 and 'benefit' not in suspended['body'], 'suspended detail retained a student payload'
        nav('archived'); text('Test için geçici askı.')
        click('Düzenle'); publish()
        dev.evaluate("document.querySelector('dialog input[type=checkbox]').click()")
        confirm(); saved()
        nav('partners'); click('Düzenle'); fill('#edit-partnership_status','ended')
        click('Kaydet'); present('dialog[open]'); text('bağlı yayımlanmış avantajları askıya alır'); confirm(); saved()
        assert api('/api/admin/benefits')['benefits'][0]['status']=='suspended'
        wait(lambda:dev.evaluate("document.querySelector('.admin-table tbody')?.textContent.includes('Sonlandırıldı')"), 'inactive business row did not refresh')
        assert dev.evaluate("![...document.querySelectorAll('.admin-table tbody tr button')].some(b=>b.textContent.trim()==='Yayınla')"), 'inactive business retained its row publication action'
        click('Düzenle')
        present('#edit-partnership_status')
        assert dev.evaluate("document.querySelector('#edit-partnership_status').value==='ended'")
        assert dev.evaluate("![...document.querySelectorAll('.management-editor button')].some(b=>['Yayınla','Kaydet ve yayına devam et'].includes(b.textContent.trim()))"), 'inactive business retained its form publication action'
        click('Vazgeç', "document.querySelector('.management-editor')")
        nav('benefits'); click('Düzenle')
        present('.publication-business-status')
        wait(lambda:dev.evaluate("document.querySelector('.publication-business-status').textContent.includes('Sonlandırıldı')"), 'inactive business status is not explained inline')
        blocked_publication('Yayın için Aktif olmalı.')
        assert api('/api/admin/businesses')['businesses'][0]['partnership_status']=='ended', 'publication automatically activated the business'
        assert api('/api/admin/benefits')['benefits'][0]['status']=='suspended', 'inactive business publication bypassed the guard'
        click('Vazgeç', "document.querySelector('.management-editor')")
        nav('partners')
        click('Düzenle'); fill('#edit-partnership_status','active'); click('Kaydet'); confirm(); saved()
        assert api('/api/admin/benefits')['benefits'][0]['status']=='suspended', 'reopening republished offer'

        # A second writer must not silently overwrite the open form.
        click('Düzenle'); fill('#edit-name','Korunması gereken yerel düzenleme')
        dev.evaluate("""(async()=>{const session=await fetch('/api/auth/session').then(r=>r.json());const {businesses}=await fetch('/api/admin/businesses').then(r=>r.json());const {id,created_at,updated_at,...body}=businesses[0];body.notes='Concurrent server change';const r=await fetch('/api/admin/businesses/'+id,{method:'PUT',headers:{'Content-Type':'application/json','X-CSRF-Token':session.csrf_token},body:JSON.stringify(body)});if(!r.ok)throw new Error('second writer failed')})()""")
        click('Kaydet'); confirm(); text('Girdiğin bilgiler korundu')
        assert dev.evaluate("document.querySelector('#edit-name').value==='Korunması gereken yerel düzenleme'")
        click('Vazgeç', "document.querySelector('.management-editor')")

        # Business form continuation saves only the business, then asks which
        # advantage to publish. A new offer requires its own reviewed approval.
        nav('businesses'); click('Düzenle')
        red_button('Yayınla', "document.querySelector('.management-editor')")
        fill('#edit-notes','Synthetic business publication continuation')
        red_button('Kaydet ve yayına devam et', "document.querySelector('.management-editor')")
        before = dev.evaluate('window.__publishRequests')
        click('Kaydet ve yayına devam et', "document.querySelector('.management-editor')")
        wait(lambda:dev.evaluate("!!document.querySelector('.business-publication')||!!document.querySelector('dialog[open]')"), 'business save did not continue to publication')
        if dev.evaluate("!!document.querySelector('dialog[open]')"):
            confirm()
        present('.business-publication')
        assert api('/api/admin/businesses')['businesses'][0]['notes']=='Synthetic business publication continuation'
        assert dev.evaluate('window.__publishRequests')==before, 'business form automatically published an advantage'
        click('Avantaj oluştur', "document.querySelector('.business-publication')")
        assert dev.evaluate("document.querySelector('#edit-business_id').value")==business['id']
        fill('#edit-title','İkinci sentetik yayın')
        fill('#edit-description','Yalnızca izole yeni yayın testi.')
        fill('#edit-perk','%5 test indirimi')
        fill('#edit-conditions','Yalnızca sentetik test koşulu.')
        fill('#edit-instructions','Test bilgilerini göster.')
        fill('#edit-code','SYNTHETIC-SECOND')
        red_button('Kaydet ve yayımla', "document.querySelector('.management-editor')")
        # A failed POST must retain the successful real PUT and its new revision.
        writes_before=dev.evaluate('window.__managementWrites.length')
        dev.call('Network.setBlockedURLs', {'urls':['*/api/admin/benefits/*/publish']})
        publish(); present('dialog[open]')
        dev.evaluate("document.querySelector('dialog input[type=checkbox]').click()")
        confirm()
        present('.management-editor [role=alert]')
        second=next(b for b in api('/api/admin/benefits')['benefits'] if b['title']=='İkinci sentetik yayın')
        assert second['publication_status']=='draft' and second['revision']>=1, 'failed publication discarded its saved draft'
        assert dev.evaluate("document.querySelector('#edit-title').value==='İkinci sentetik yayın'&&document.querySelector('#edit-perk').value==='%5 test indirimi'"), 'failed publication discarded form values'
        operations=dev.evaluate('window.__managementWrites.slice('+str(writes_before)+')')
        assert [op['method'] for op in operations]==['PUT','POST'] and operations[0].get('status')==200, 'new publication must save once before attempting publication'
        dev.call('Network.setBlockedURLs', {'urls':[]})
        writes_before=dev.evaluate('window.__managementWrites.length')
        red_button('Yayınla', "document.querySelector('.management-editor')")
        publish(); present('dialog[open]')
        dev.evaluate("document.querySelector('dialog input[type=checkbox]').click()")
        confirm(); saved()
        operations=dev.evaluate('window.__managementWrites.slice('+str(writes_before)+')')
        assert [(op['method'],op.get('status')) for op in operations]==[('POST',200)], 'publication retry duplicated the saved draft'
        second=next(b for b in api('/api/admin/benefits')['benefits'] if b['id']==second['id'])
        assert second['publication_status']=='published'

        nav('benefits'); click('Avantaj taslağı ekle', prefix=True)
        fill('#edit-business_id',business['id']); fill('#edit-title','Seçilmeyen sentetik taslak')
        click('Taslak olarak kaydet'); confirm(); saved()
        unselected=next(b for b in api('/api/admin/benefits')['benefits'] if b['title']=='Seçilmeyen sentetik taslak')
        nav('businesses')
        before=dev.evaluate('window.__publishRequests')
        click('Yayınla', "document.querySelector('.admin-table tbody tr')")
        present('.business-publication')
        for title in ('Test kahve avantajı','İkinci sentetik yayın','Seçilmeyen sentetik taslak'):
            wait(lambda title=title:dev.evaluate('document.querySelector(".business-publication").textContent.includes('+json.dumps(title)+')'), 'business publication omitted an advantage')
        assert dev.evaluate('window.__publishRequests')==before, 'opening multi-advantage selection performed a bulk publication'
        for width in (320,390,1440):
            viewport(width); shot(f'management-business-publication-selection-{width}.png')
        selection='document.querySelector('+json.dumps('.business-publication [data-benefit-id="'+benefit['id']+'"]')+')'
        click('Yayınla',selection)
        present('#edit-title')
        assert dev.evaluate("document.querySelector('#edit-title').value")==benefit['title'], 'business publication selected the wrong advantage'
        publish(); present('dialog[open]')
        dev.evaluate("document.querySelector('dialog input[type=checkbox]').click()")
        confirm(); saved()
        after={b['id']:b for b in api('/api/admin/benefits')['benefits']}
        assert after[benefit['id']]['publication_status']=='published'
        assert after[unselected['id']]['publication_status']=='draft' and after[unselected['id']]['revision']==unselected['revision'], 'publishing one selection changed another draft'
        assert after[second['id']]['revision']==second['revision'], 'publishing one selection republished another offer'
        assert dev.evaluate('window.__publishRequests')==before+1, 'selected publication affected more than one advantage'
        student_catalog(True)

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
            wait(lambda:dev.evaluate("document.activeElement.classList.contains('admin-menu-button')"),'focus not restored after menu closes')
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
        present('.student-panel')
        dev.evaluate("location.hash='admin/businesses'"); text('yönetici yetkisi gerekir')
        assert not dev.evaluate("!!document.querySelector('#desktop-nav')"), 'student saw administration'
        click('Çıkış yap'); present('#password')
        assert not dev.errors, 'browser runtime error'
        assert not student_dev.errors, 'student browser runtime error'
        print('Management browser acceptance passed: 9 sidebar destinations, 4 sponsorship tabs at 320/390/1440 px, red business row/form publication, preselected creation, single-confirmation save and publish, failed POST draft retention, explicit multi-offer selection, required fields, inactive-business guard, normal-student visibility, conflicts, network retry and logout.')
    except Exception:
        if dev:
            output=ROOT/'tests'/'artifacts'; output.mkdir(exist_ok=True)
            (output/'management-failure.txt').write_text(dev.evaluate('document.body.innerText'),encoding='utf-8')
            if student_dev:
                (output/'management-student-failure.txt').write_text(student_dev.evaluate('document.body.innerText'),encoding='utf-8')
        raise
    finally:
        if dev: dev.socket.close()
        if student_dev: student_dev.socket.close()
        if student_process:
            student_process.terminate(); student_process.wait(timeout=15)
        if student_profile: student_profile.cleanup()
        process.terminate(); process.wait(timeout=15); profile.cleanup()


if __name__=='__main__': main()
