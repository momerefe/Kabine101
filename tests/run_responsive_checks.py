"""Full responsive audit of the real application, launched by isolated Go harness.

KOC_BROWSER_TEST=1 KOC_RESPONSIVE_TEST=1 go test -tags=integration
./internal/management -run TestManagementBrowser -count=1 -v
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
OUT = ROOT / 'tests' / 'artifacts' / 'responsive'
SIZES = [(320,740),(360,800),(390,844),(430,932),(768,1024),(1024,768),(1440,1000),(1920,1080),(844,390)]


def main():
    base = os.environ.get('KOC_BROWSER_URL','')
    if not base.startswith('http://127.0.0.1:'):
        raise SystemExit('Use the isolated Go browser harness.')
    OUT.mkdir(parents=True,exist_ok=True)
    profile = tempfile.TemporaryDirectory(prefix='koc-responsive-',ignore_cleanup_errors=True)
    chrome=os.environ.get('KOC_CHROME','C:/Program Files/Google/Chrome/Application/chrome.exe')
    process=subprocess.Popen([chrome,'--headless=new','--disable-gpu','--no-first-run','--remote-debugging-port=0','--user-data-dir='+profile.name,'about:blank'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,creationflags=getattr(subprocess,'CREATE_NO_WINDOW',0))
    checks=[]; issues=[]; dev=None
    def wait(test,label):
        until=time.monotonic()+15
        while time.monotonic()<until:
            try:
                if test(): return
            except (OSError,ValueError): pass
            time.sleep(.05)
        raise AssertionError(label)
    try:
        portfile=Path(profile.name)/'DevToolsActivePort'
        wait(lambda: bool(portfile.read_text().splitlines()),'Chrome startup')
        port=portfile.read_text().splitlines()[0]
        with urllib.request.urlopen(f'http://127.0.0.1:{port}/json/list') as r: targets=json.load(r)
        dev=DevTools(next(x['webSocketDebuggerUrl'] for x in targets if x['type']=='page'))
        for method in ('Runtime.enable','Page.enable','Network.enable'): dev.call(method)
        dev.call('Emulation.setFocusEmulationEnabled',{'enabled':True})
        def present(selector): wait(lambda:dev.evaluate(f'!!document.querySelector({json.dumps(selector)})'),'missing '+selector)
        def text(value): wait(lambda:dev.evaluate(f'document.body.innerText.includes({json.dumps(value)})'),'missing '+value)
        def route(hash,selector):
            dev.evaluate('location.hash='+json.dumps(hash)); present(selector)
            time.sleep(.07)
        def size(w,h,touch=None):
            if touch is None: touch=w<=1100
            dev.call('Emulation.setDeviceMetricsOverride',{'width':w,'height':h,'deviceScaleFactor':1,'mobile':touch})
            dev.call('Emulation.setTouchEmulationEnabled',{'enabled':touch,'maxTouchPoints':1})
            time.sleep(.12)
        def shot(name):
            image=dev.call('Page.captureScreenshot',{'format':'png','captureBeyondViewport':False})
            (OUT/(name+'.png')).write_bytes(base64.b64decode(image['data']))
        def layout(name,w,h):
            size(w,h)
            result=dev.evaluate("""(()=>{const root=document.documentElement;const dialog=document.querySelector('dialog[open]');const d=dialog?.getBoundingClientRect();return {width:innerWidth,pageWidth:root.scrollWidth,dialogWidth:dialog?.scrollWidth,dialogClient:dialog?.clientWidth,dialogBounds:d?{left:d.left,right:d.right,top:d.top,bottom:d.bottom}:null,brokenImages:Array.from(document.images).filter(i=>i.getClientRects().length&&i.complete&&!i.naturalWidth).map(i=>i.src)}})()""")
            checks.append({'page':name,'viewport':[w,h],**result})
            if result['pageWidth']>w+1: issues.append(f'{name} {w}x{h}: page overflows {result["pageWidth"]}')
            if result.get('dialogWidth',0)>result.get('dialogClient',0)+1: issues.append(f'{name} {w}x{h}: dialog content overflows')
            if result['dialogBounds']:
                d=result['dialogBounds']
                if d['left']<0 or d['right']>w+1 or d['top']<0 or d['bottom']>h+1: issues.append(f'{name} {w}x{h}: dialog outside viewport')
            if result['brokenImages']: issues.append(f'{name}: broken image')
            if w in (320,390,1440) or (w,h)==(844,390): shot(f'{name}-{w}x{h}')
        def matrix(name):
            for w,h in SIZES: layout(name,w,h)
        def fill(selector,value):
            present(selector)
            dev.evaluate(f"""(()=>{{const e=document.querySelector({json.dumps(selector)});const proto=e.tagName==='SELECT'?HTMLSelectElement.prototype:e.tagName==='TEXTAREA'?HTMLTextAreaElement.prototype:HTMLInputElement.prototype;Object.getOwnPropertyDescriptor(proto,'value').set.call(e,{json.dumps(value)});e.dispatchEvent(new Event(e.tagName==='SELECT'?'change':'input',{{bubbles:true}}));}})()""")
        def button(label,scope='document'):
            assert dev.evaluate(f"(()=>{{const e=Array.from({scope}.querySelectorAll('button')).find(e=>e.textContent.trim()==={json.dumps(label)});if(!e||e.disabled)return false;e.click();return true}})()"),'button '+label
        def tap(selector):
            present(selector)
            hit=dev.evaluate(f"""(()=>{{const e=document.querySelector({json.dumps(selector)});e.scrollIntoView({{block:'center',inline:'center'}});const r=e.getBoundingClientRect();const x=r.x+r.width/2,y=r.y+r.height/2;const hit=document.elementFromPoint(x,y);return {{x,y,reachable:hit===e||e.contains(hit)}}}})()""")
            assert hit['reachable'],'covered/unreachable '+selector
            dev.call('Input.dispatchTouchEvent',{'type':'touchStart','touchPoints':[{'x':hit['x'],'y':hit['y']}]})
            dev.call('Input.dispatchTouchEvent',{'type':'touchEnd','touchPoints':[]})
            time.sleep(.1)
        def admin(section):
            grouped = section in ('businesses','benefits','published','archived')
            active = 'benefits' if grouped else section
            route('admin/'+section,f'#desktop-nav a[href="#admin/{active}"][aria-current="page"]')
            if grouped:
                present(f'nav[aria-label="Sponsorluk bölümleri"] a[href="#admin/{section}"][aria-current="page"]')
            if section in ('businesses','benefits','published','archived','partners'):
                wait(lambda:not dev.evaluate("document.querySelector('#management-content')?.innerText.includes('Kayıtlar yükleniyor')"),'list loading')
            if section in ('students','users'): present('.admin-account-list')
        size(1440,1000,False)
        dev.call('Page.navigate',{'url':base+'/#login/student'}); present('#email')
        auth_pages=[('login/student','#password'),('login/admin','#username'),('register','#name'),('forgot-password','#email'),('resend-verification','#email'),('verify-email?token='+'A'*43,'form.auth-form'),('reset-password?token='+'A'*43,'#password'),('verify-email','[role=alert]'),('system','.foundation-card [role=status]')]
        for fragment,selector in auth_pages:
            route(fragment,selector); matrix(fragment.split('?')[0].replace('/','-')+('-token' if '?'in fragment else ''))
        route('login/admin','#username'); size(390,844)
        fill('#username',os.environ['KOC_BROWSER_USERNAME']); fill('#password',os.environ['KOC_BROWSER_PASSWORD'])
        tap('form.auth-form button[type=submit]'); present('#desktop-nav')
        # Long, clearly synthetic input exercises details and confirmations, not development data.
        seed=dev.evaluate("""(async()=>{const session=await fetch('/api/auth/session').then(r=>r.json());const write=async(method,path,body)=>{const r=await fetch(path,{method,headers:{'Content-Type':'application/json','X-CSRF-Token':session.csrf_token},body:JSON.stringify(body)});if(!r.ok)throw new Error('fixture request '+r.status);return r.json()};const id=crypto.randomUUID();await write('PUT','/api/admin/businesses/'+id,{revision:0,name:'Mobil görünüm test işletmesi',category:'food',description:'Gerçek sponsor değildir. '+ 'Uzun açıklama '.repeat(30),image_path:'assets/cafe.svg',partnership_status:'active',contact_person:'İlgili kişi',contact_info:'https://example.com/'+ 'contact'.repeat(30),branches:'Test şubesi',last_contact:'',notes:'Test notları'});const ids={};for(const method of ['code','link','qr','membership']){const bid=crypto.randomUUID();await write('PUT','/api/admin/benefits/'+bid,{revision:0,business_id:id,title:'Ekran testi '+method,description:'Yalnızca izole tarayıcı testi. '+ 'Açıklama '.repeat(25),perk:'%10 test indirimi',conditions:['Koşul '+ 'UZUNMETİN'.repeat(120),'Öğrenci kimliği gerekir.'],instructions:['Kasada göster.'],products:'Test ürünleri',branches:'Test şubesi',exceptions:'Test istisnası',minimum_spend:'25.50',currency:'TRY',verification_requirement:'Öğrenci kimliği',valid_from:'',valid_until:'',method,code:method==='code'?'TEST-'+ 'ABC'.repeat(45):'',target_url:method==='link'?'https://example.com/'+ 'path'.repeat(30):'',qr_content:method==='qr'?'SYNTHETIC-QR':'',membership_label:method==='membership'?'Test kartı':'',support_contact:'',featured:false});ids[method]=bid;}return {business:id,benefits:ids}})()""")
        for section in ('overview','partners','businesses','benefits','students','published','archived','usage','analytics','users','audit','settings'):
            admin(section); matrix('admin-'+section)
        # All reachable menu links via actual touch, including tablet and landscape.
        for w,h in [(320,740),(390,844),(768,1024),(820,1180),(1024,768),(844,390)]:
            size(w,h); admin('businesses')
            tap('.admin-menu-button')
            wait(lambda:dev.evaluate("document.body.classList.contains('admin-menu-open')"),'touch menu not opened')
            layout('menu-open',w,h)
            tap('#desktop-nav a[href="#admin/benefits"]')
            wait(lambda:not dev.evaluate("document.body.classList.contains('admin-menu-open')"),'menu not closed on touch selection')
            tap('.admin-menu-button')
            # Keyboard focus wraps between visible controls, including on touch tablets.
            dev.evaluate("window.menuItems=Array.from(document.querySelectorAll('.sidebar a,.sidebar button')).filter(e=>e.getClientRects().length&&getComputedStyle(e).visibility!=='hidden');window.menuItems[0].focus()")
            dev.call('Input.dispatchKeyEvent',{'type':'keyDown','key':'Tab','code':'Tab','windowsVirtualKeyCode':9,'modifiers':8})
            assert dev.evaluate("document.activeElement===window.menuItems.at(-1)"),'menu backward focus wrap'
            dev.call('Input.dispatchKeyEvent',{'type':'keyDown','key':'Tab','code':'Tab','windowsVirtualKeyCode':9})
            assert dev.evaluate("document.activeElement===window.menuItems[0]"),'menu forward focus wrap'
            dev.call('Input.dispatchKeyEvent',{'type':'keyDown','key':'Escape','code':'Escape','windowsVirtualKeyCode':27})
            wait(lambda:not dev.evaluate("document.body.classList.contains('admin-menu-open')"),'Escape close')
            wait(lambda:dev.evaluate("document.activeElement.classList.contains('admin-menu-button')"),'Escape focus restoration')
            # Closed drawer links must not receive sequential keyboard focus.
            dev.evaluate("document.querySelector('.admin-menu-button').focus()")
            dev.call('Input.dispatchKeyEvent',{'type':'keyDown','key':'Tab','code':'Tab','windowsVirtualKeyCode':9})
            if dev.evaluate("!!document.activeElement.closest('.sidebar')"): issues.append(f'hidden menu keyboard focus {w}')
        admin('businesses'); button('Düzenle'); present('#edit-name'); matrix('business-form')
        button('Vazgeç',"document.querySelector('.management-editor')")
        button('Detay'); text('İlgili kişi'); matrix('business-detail')
        for method in ('code','link','qr','membership'):
            admin('benefits')
            fill('.admin-toolbar input[type=search]','Ekran testi '+method)
            wait(lambda:dev.evaluate("document.querySelectorAll('.admin-table tbody tr').length===1"),'filter unique method')
            button('Detay'); matrix('benefit-detail-'+method)
            button('Düzenle'); present('#edit-method'); matrix('benefit-form-'+method)
            assert dev.evaluate('document.querySelector("#edit-method").value')==method
            if method=='code':
                button('Yayınla'); present('dialog[open]'); matrix('publish-dialog')
                size(390,430); layout('publish-dialog-keyboard',390,430)
                button('Vazgeç',"document.querySelector('dialog[open]')")
            button('Vazgeç',"document.querySelector('.management-editor')")
        admin('students')
        card="Array.from(document.querySelectorAll('.admin-account')).find(e=>e.textContent.includes('pending-layout@ogrenci.example'))"
        button('Onayla',card); present('dialog[open]'); matrix('approval-dialog')
        button('Vazgeç',"document.querySelector('dialog[open]')")
        route('account','.account-details'); matrix('admin-account')
        button('Çıkış yap'); present('#password')
        route('login/student','#email'); fill('#email',os.environ['KOC_BROWSER_STUDENT']); fill('#password',os.environ['KOC_BROWSER_PASSWORD'])
        dev.evaluate("document.querySelector('form.auth-form').requestSubmit()"); present('.student-panel')
        route('profile','.student-profile'); matrix('student-account')
        button('Çıkış yap'); present('#password')
        if dev.errors: issues.append('JavaScript runtime exceptions')
        report={'checks':checks,'issues':issues,'runtimeErrors':dev.errors,'engine':'Chrome desktop + touch/device emulation; no physical device claim'}
        (OUT/'report.json').write_text(json.dumps(report,ensure_ascii=False,indent=2),encoding='utf-8')
        print(f'Responsive audit: {len(checks)} page/viewport checks, {len(issues)} issues; screenshots tests/artifacts/responsive')
        for issue in issues: print(issue)
        if issues: raise AssertionError('Responsive issues require correction')
    except Exception as error:
        (OUT/'failure.json').write_text(json.dumps({'error':str(error),'checks':checks,'issues':issues},ensure_ascii=False,indent=2),encoding='utf-8')
        if dev:
            (OUT/'failure.txt').write_text(dev.evaluate('document.body.innerText'),encoding='utf-8')
        raise
    finally:
        if dev: dev.socket.close()
        process.terminate(); process.wait(timeout=15); profile.cleanup()


if __name__=='__main__': main()
