"""Student UI acceptance on a user-started Vite server, with synthetic API only.

Set KOC_STUDENT_URL=http://192.168.1.94:5173 and run this script.
No backend/frontend is launched; no development account or database is accessed.
Real authorization/DB behavior is covered separately by Go integration tests.
"""
import base64
import datetime
import ipaddress
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
OUT = ROOT / 'tests/artifacts/student'


def main():
    base=os.environ.get('KOC_STUDENT_URL','').rstrip('/')
    parsed=urllib.parse.urlsplit(base)
    if parsed.scheme!='http' or not parsed.hostname or not ipaddress.ip_address(parsed.hostname).is_private or parsed.path or parsed.query or parsed.fragment or parsed.username:
        raise SystemExit('Set KOC_STUDENT_URL to the existing local/LAN Vite origin.')
    today=datetime.datetime.now(datetime.timezone(datetime.timedelta(hours=3))).date()
    items=[]
    for n,method in enumerate(('code','link','qr','membership','code','code'),1):
        items.append({'id':f'00000000-0000-4000-8000-{n:012d}','business_id':f'10000000-0000-4000-8000-{n:012d}','business_name':f'Test işletmesi {n}','image_path':'assets/cafe.svg' if n%2 else 'assets/books.svg','category':'food' if n%2 else 'other','title':f'Öğrenci avantajı {n}','description':'Yalnızca otomatik arayüz testi için kurgusal içerik.','perk':'%15 test indirimi','conditions':['Öğrenci kimliğini göster.','Başka kampanyalarla birleştirilemez.'],'instructions':['Koşulları kontrol et.','Kasada işletmenin tanımladığı bilgiyi göster.'],'products':'Seçili ürünler','branches':'Test şubesi','exceptions':'Paketli ürünler hariç','minimum_spend':'25.50' if n==1 else '', 'currency':'TRY','verification_requirement':'Öğrenci kimliği','valid_from':str(today-datetime.timedelta(days=5)),'valid_until':str(today+datetime.timedelta(days=7)),'method':method,'featured':n<=2,'status':'expired' if n==5 else 'scheduled' if n==6 else 'active','is_favorite':False,'support_contact':''})
    script='window.__offers='+json.dumps(items,ensure_ascii=False)+r""";
    window.__apiCalls=[];window.__submittedEmails=[];window.__blocked=false;window.__failCatalog=false;window.__long=sessionStorage.getItem('synthetic-long')==='1';
    const storeKey='synthetic-student-favorites';
    const saved=()=>JSON.parse(sessionStorage.getItem(storeKey)||'[]');
    const response=(v,status=200)=>new Response(JSON.stringify(v),{status,headers:{'Content-Type':'application/json'}});
    const account=()=>({id:'20000000-0000-4000-8000-000000000001',name:'Test öğrencisi',email:'browser@ogrenci.example',username:'',role:'student',status:'active',eligibility_status:'approved',email_verified:true});
    const session=()=>({user:sessionStorage.getItem('synthetic-anon')==='1'?null:account(),csrf_token:'synthetic-csrf',school_domain:'ogrenci.example',registration_open:true,local_email:true});
    const publicItem=(item)=>({...item,is_favorite:saved().includes(item.id),status:window.__blocked&&item.id.endsWith('000001')?'expired':item.status,...(window.__long?{title:'UZUNAVANTAJBAŞLIĞI'.repeat(18),description:'Açıklama '.repeat(45),conditions:['UZUNKOŞUL'.repeat(100)]}:{})});
    const original=window.fetch.bind(window);
    window.fetch=async(input,init)=>{
      const u=new URL(input instanceof Request?input.url:String(input),location.href);
      if(!u.pathname.startsWith('/api/'))return original(input,init);
      const method=init?.method||(input instanceof Request?input.method:'GET');
      window.__apiCalls.push({path:u.pathname,query:u.search,method});
      if(u.pathname==='/api/auth/session')return response(session());
      if(u.pathname==='/api/auth/login'||u.pathname==='/api/auth/register'){
        const body=input instanceof Request?await input.clone().json():JSON.parse(init.body);
        window.__submittedEmails.push({path:u.pathname,email:body.email});
        if(u.pathname==='/api/auth/register')return response({message:'Test kayıt talebi alındı.'},202);
        sessionStorage.removeItem('synthetic-anon');return response({user:account()});
      }
      if(u.pathname==='/api/auth/logout'){
        sessionStorage.setItem('synthetic-anon','1');return response({message:'Çıkış yapıldı.'});
      }
      if(u.pathname==='/api/student/settings')return response({platform_name:'KoçBenefits',support_contact:''});
      if(u.pathname==='/api/student/savings'){
        if(window.__failSavings)return response({message:'Test bağlantı hatası'},503);
        return response(window.__savings||{month:new Intl.DateTimeFormat('sv-SE',{year:'numeric',month:'2-digit',timeZone:'Europe/Istanbul'}).format(new Date()),currency:'TRY',total_savings:'0.00',purchase_count:0,businesses:[]});
      }
      if(u.pathname==='/api/student/purchases')return response({purchases:[],truncated:false});
      if(u.pathname==='/api/student/catalog'){
        await new Promise(r=>setTimeout(r,100));
        if(window.__failCatalog)return response({message:'Synthetic failure'},503);
        let offers=window.__offers.map(publicItem);
        if(u.searchParams.get('category'))offers=offers.filter(o=>o.category===u.searchParams.get('category'));
        if(u.searchParams.get('q'))offers=offers.filter(o=>(o.title+' '+o.business_name).toLocaleLowerCase('tr').includes(u.searchParams.get('q').toLocaleLowerCase('tr')));
        return response({benefits:offers,truncated:false});
      }
      if(u.pathname==='/api/student/favorites')return response({benefits:window.__offers.filter(o=>saved().includes(o.id)).map(publicItem),truncated:false});
      const favorite=u.pathname.match(/^\/api\/student\/favorites\/([^/]+)$/);
      if(favorite){
        const headers=new Headers(init?.headers||(input instanceof Request?input.headers:{}));
        if(headers.get('X-CSRF-Token')!=='synthetic-csrf'||headers.get('Content-Type')!=='application/json')return response({message:'Synthetic CSRF validation'},403);
        const ids=new Set(saved());if(method==='PUT')ids.add(favorite[1]);else if(method==='DELETE')ids.delete(favorite[1]);else return response({},405);
        sessionStorage.setItem(storeKey,JSON.stringify([...ids]));return response({is_favorite:method==='PUT'});
      }
      const benefit=u.pathname.match(/^\/api\/student\/benefits\/([^/]+)(\/redemption)?$/);
      if(benefit){
        const item=window.__offers.find(o=>o.id===benefit[1]);if(!item)return response({message:'Synthetic missing'},404);
        if(!benefit[2])return response({benefit:publicItem(item)});
        if(publicItem(item).status!=='active')return response({message:'Synthetic inactive'},409);
        const field={code:{code:'SYNTHETIC-KOC-15'},link:{target_url:'https://example.com/student-benefit'},qr:{qr_content:'https://example.com/öğrenci?kod=TEST'},membership:{membership_label:'TEST ÜYELİK BİLGİSİ'}}[item.method];
        return response({method:item.method,...field});
      }
      return response({message:'Unexpected synthetic API request'},500);
    };
    // Deliberately force the manual-copy path on HTTP LAN.
    Object.defineProperty(navigator,'clipboard',{configurable:true,value:undefined});
    document.execCommand=()=>false;
    """
    email_only=os.environ.get('KOC_SCHOOL_EMAIL_ONLY')=='1'
    savings_only=os.environ.get('KOC_SAVINGS_UI_ONLY')=='1'
    if email_only:script+="sessionStorage.setItem('synthetic-anon','1');"
    if savings_only:
        script+='window.__savings='+json.dumps({'month':str(today)[:7],'currency':'TRY','total_savings':'42.07','purchase_count':7,'businesses':[{'business_id':str(n),'business_name':'Test işletmesi '+str(n),'purchase_count':1,'savings_amount':'6.01'} for n in range(7)]},ensure_ascii=False)+';'
    OUT.mkdir(parents=True,exist_ok=True)
    profile=tempfile.TemporaryDirectory(prefix='koc-student-ui-',ignore_cleanup_errors=True)
    chrome=os.environ.get('KOC_CHROME','C:/Program Files/Google/Chrome/Application/chrome.exe')
    process=subprocess.Popen([chrome,'--headless=new','--disable-gpu','--no-first-run','--remote-debugging-port=0','--user-data-dir='+profile.name,'about:blank'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,creationflags=getattr(subprocess,'CREATE_NO_WINDOW',0))
    dev=None;checks=[]
    def wait(check,label):
        until=time.monotonic()+20
        while time.monotonic()<until:
            try:
                if check():return
            except (OSError,ValueError):pass
            time.sleep(.05)
        raise AssertionError(label)
    try:
        portfile=Path(profile.name)/'DevToolsActivePort'
        wait(lambda:portfile.exists() and bool(portfile.read_text()),'Chrome startup')
        port=portfile.read_text().splitlines()[0]
        with urllib.request.urlopen(f'http://127.0.0.1:{port}/json/list') as r:targets=json.load(r)
        dev=DevTools(next(t['webSocketDebuggerUrl'] for t in targets if t['type']=='page'))
        for method in ('Runtime.enable','Page.enable','Network.enable'):dev.call(method)
        dev.call('Emulation.setFocusEmulationEnabled',{'enabled':True})
        dev.call('Page.addScriptToEvaluateOnNewDocument',{'source':script})
        def present(selector):wait(lambda:dev.evaluate(f'!!document.querySelector({json.dumps(selector)})'),'missing '+selector)
        def text(value):wait(lambda:dev.evaluate(f'document.body.innerText.includes({json.dumps(value)})'),'missing text '+value)
        def button(label):
            assert dev.evaluate(f"(()=>{{const e=Array.from(document.querySelectorAll('button')).find(e=>e.textContent.trim()==={json.dumps(label)});if(!e||e.disabled)return false;e.click();return true}})()"),'button '+label
        def route(value,selector):
            dev.evaluate('location.hash='+json.dumps(value));present(selector);time.sleep(.1)
        def reload(selector):
            before=dev.evaluate('performance.timeOrigin')
            dev.call('Page.reload')
            wait(lambda:dev.evaluate('performance.timeOrigin')!=before,'reload navigation')
            present(selector)
        def fill(selector,value):
            present(selector)
            dev.evaluate(f"(()=>{{const e=document.querySelector({json.dumps(selector)});Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set.call(e,{json.dumps(value)});e.dispatchEvent(new Event('input',{{bubbles:true}}));}})()")
        def size(w,h):
            dev.call('Emulation.setDeviceMetricsOverride',{'width':w,'height':h,'deviceScaleFactor':1,'mobile':w<=1100})
            dev.call('Emulation.setTouchEmulationEnabled',{'enabled':w<=1100})
            time.sleep(.15)
        def shot(name):
            img=dev.call('Page.captureScreenshot',{'format':'png','captureBeyondViewport':False})
            (OUT/(name+'.png')).write_bytes(base64.b64decode(img['data']))
        def layout(name,w,h):
            size(w,h)
            result=dev.evaluate("({viewport:innerWidth,page:document.documentElement.scrollWidth,broken:Array.from(document.images).filter(i=>i.complete&&!i.naturalWidth&&i.getClientRects().length).map(i=>i.src)})")
            assert result['page']<=w+1 and result['viewport']<=w+1,f'overflow {name} {w}: {result}'
            assert not result['broken'],f'broken image {name}'
            checks.append({'page':name,'width':w,'height':h})
            if w in(320,390,1440):dev.evaluate('window.scrollTo(0,0)');shot(f'{name}-{w}')
        def matrix(name):
            for w,h in((320,740),(390,844),(768,1024),(1024,768),(1440,1000),(844,390)):layout(name,w,h)
        if email_only:
            size(1440,1000);dev.call('Page.navigate',{'url':base+'/#register'});present('.school-email-field')
            for w,h in((320,740),(390,844),(1440,1000)):layout('email-register',w,h)
            assert dev.evaluate("document.querySelector('#email-domain').textContent")=='@ogrenci.example'
            fill('#name','Test Öğrencisi');fill('#email','deniz.test');fill('#password','Synthetic password 2026!')
            dev.evaluate("document.querySelector('form.auth-form').requestSubmit()")
            wait(lambda:dev.evaluate("window.__submittedEmails.length===1"),'registration request')
            assert dev.evaluate('window.__submittedEmails[0].email')=='deniz.test@ogrenci.example'
            text('Test kayıt talebi alındı')
            fill('#email','deniz.test@ogrenci.example')
            assert dev.evaluate("document.querySelector('#email').value")=='deniz.test','full email paste not normalized'
            fill('#email','deniz@outside.example');fill('#password','Synthetic password 2026!')
            dev.evaluate("document.querySelector('form.auth-form').requestSubmit()")
            text('yalnızca @ işaretinden önceki kısmını yaz')
            assert dev.evaluate('window.__submittedEmails.length')==1,'foreign domain was submitted'
            route('login/admin','#username')
            assert not dev.evaluate("!!document.querySelector('.school-email-field')"),'admin username altered'
            route('login/student','#email')
            for w,h in((320,740),(390,844),(1440,1000)):layout('email-login',w,h)
            fill('#email','browser');fill('#password','Synthetic password 2026!')
            dev.evaluate("document.querySelector('form.auth-form').requestSubmit()")
            present('.student-panel')
            assert dev.evaluate('window.__submittedEmails.at(-1).email')=='browser@ogrenci.example'
            assert not dev.errors,'JavaScript runtime errors'
            (OUT/'school-email-report.json').write_text(json.dumps({'scope':'Synthetic API UI check; no real account created','checks':checks,'runtimeErrors':dev.errors},ensure_ascii=False,indent=2),encoding='utf-8')
            print('School email UI passed: 6 layouts; prefix-only register/login, paste normalization, foreign-domain rejection, admin username preserved. No servers started.')
            return
        size(1440,1000);dev.call('Page.navigate',{'url':base+'/#overview'});present('.benefit-card')
        if savings_only:
            present('.student-savings-breakdown tbody')
            assert dev.evaluate("document.querySelector('.student-savings-total').textContent")=='₺42,07'
            assert dev.evaluate("document.querySelectorAll('.student-savings-breakdown tbody tr').length")==3
            text('Diğer 4 işletme')
            matrix('savings-approved')
            route('profile','.student-profile');dev.evaluate('window.__failSavings=true')
            route('overview','.student-savings');text('Tasarruf bilgileri alınamadı')
            assert dev.evaluate("document.querySelector('.student-savings-total').textContent")=='—','API error shown as zero or stale savings'
            layout('savings-error',320,740)
            dev.evaluate('window.__failSavings=false');button('Yeniden dene')
            text('₺42,07')
            route('profile','.student-profile')
            dev.evaluate("window.__savings.total_savings='9007199254740993.01';window.__savings.businesses[0].business_name='UZUNİŞLETMEADI'.repeat(25);window.__savings.businesses[0].savings_amount='9007199254740956.95'")
            route('overview','.student-savings');text('₺9.007.199.254.740.993,01')
            layout('savings-long',320,740)
            assert not dev.errors,dev.errors
            (OUT/'savings-report.json').write_text(json.dumps({'scope':'Real frontend, synthetic API, existing Vite only','checks':checks,'runtimeErrors':dev.errors},ensure_ascii=False,indent=2),encoding='utf-8')
            print(f'Savings UI passed: {len(checks)} layouts; exact amount, max3 breakdown/all-business total, error versus zero, retry, long name/amount. No servers started.')
            return
        for section,selector in [('overview','.benefit-card'),('discover','.benefit-card'),('favorites','.empty-state'),('history','.empty-state'),('profile','.student-profile'),('help','.faq-list')]:
            route(section,selector);matrix(section)
        route('profile','.student-profile')
        dev.evaluate("document.querySelector('.student-profile a[href=\"#forgot-password\"]').click()")
        present('form.auth-form #email');text('Parolanı yenileyelim')
        route('discover','.benefit-card')
        dev.evaluate("document.querySelector('.benefit-card .favorite-toggle').click()")
        wait(lambda:dev.evaluate("document.querySelector('.benefit-card .favorite-toggle').getAttribute('aria-pressed')==='true'"),'favorite add')
        route('favorites','.benefit-card');matrix('favorites-saved')
        reload('.benefit-card')
        dev.evaluate("document.querySelector('.benefit-card .favorite-toggle').click()")
        text('Henüz kaydettiğin bir avantaj yok')
        route('discover','.benefit-card');fill('#student-catalog-search','eşleşmeyen-arama')
        dev.evaluate("document.querySelector('.catalog-search').requestSubmit()")
        text('Aramana uygun bir avantaj bulunamadı');layout('no-results',320,740)
        route('discover?category=food','.benefit-card')
        wait(lambda:dev.evaluate("document.querySelectorAll('.benefit-card').length===3"),'category filter')
        dev.evaluate('window.__failCatalog=true')
        route('discover?q='+urllib.parse.quote('Öğrenci'),'.empty-state')
        text('Avantajlara ulaşılamadı')
        layout('catalog-error',320,740)
        dev.evaluate('window.__failCatalog=false');button('Yeniden dene');present('.benefit-card')
        for n,method in enumerate(('code','link','qr','membership','expired','scheduled'),1):
            route('benefit/'+items[n-1]['id'],'.student-redemption');matrix('detail-'+method)
            if n<=4:
                button({'code':'Kodu görüntüle','link':'Avantaj bağlantısını getir','qr':'QR kodunu göster','membership':'Üyelik bilgisini göster'}[method])
                present('.student-payload')
                if method=='code':
                    assert dev.evaluate("document.querySelector('#student-code').value")== 'SYNTHETIC-KOC-15'
                    button('Kodu kopyala');text('Otomatik kopyalanamadı')
                if method=='link':
                    assert dev.evaluate("document.querySelector('.student-payload a').href")== 'https://example.com/student-benefit'
                if method=='qr':
                    present('.student-qr svg path')
                    assert dev.evaluate("document.querySelector('.student-qr svg path').getAttribute('d').length>100")
                if method=='membership':text('TEST ÜYELİK BİLGİSİ')
                layout('payload-'+method,390,844)
            else:assert not dev.evaluate("!!document.querySelector('.student-payload')")
        route('benefit/'+items[0]['id'],'.student-redemption');button('Kodu görüntüle');present('#student-code')
        dev.evaluate("window.dispatchEvent(new Event('blur'))")
        wait(lambda:not dev.evaluate("!!document.querySelector('#student-code')"),'payload survives blur')
        button('Avantajı paylaş · bağlantıyı kopyala');text('Otomatik kopyalanamadı')
        assert dev.evaluate("Array.from(document.querySelectorAll('.student-redemption input')).some(e=>e.value===location.origin+location.pathname+'#benefit/'+window.__offers[0].id)"),'manual share URL missing'
        dev.evaluate('window.__blocked=true');button('Kodu görüntüle')
        text('Bu avantaj şu anda kullanıma açık değil')
        assert not dev.evaluate("!!document.querySelector('#student-code')")
        route('benefit/00000000-0000-4000-8000-999999999999','.empty-state');text('Avantaja ulaşılamadı');layout('missing',320,740)
        route('discover','.benefit-card');dev.evaluate("sessionStorage.setItem('synthetic-long','1')");reload('.benefit-card')
        layout('long-catalog',320,740)
        route('benefit/'+items[0]['id'],'.student-redemption');layout('long-detail',320,740)
        # Direct benefit link must survive authentication, while unsafe next values cannot redirect.
        dev.evaluate("sessionStorage.setItem('synthetic-anon','1')")
        reload('#email')
        fill('#email','browser@ogrenci.example');fill('#password','Synthetic password 2026!')
        dev.evaluate("document.querySelector('form.auth-form').requestSubmit()")
        present('.student-redemption')
        assert dev.evaluate('location.hash')=='#benefit/'+items[0]['id'],'login did not return to benefit'
        route('profile','.student-profile');button('Çıkış yap');present('#email')
        assert not dev.evaluate("!!document.querySelector('.student-panel')"),'student panel remains after logout'
        assert not dev.errors,'JavaScript runtime errors'
        (OUT/'report.json').write_text(json.dumps({'scope':'Chrome viewport/touch emulation; real frontend and synthetic API; no real DB or physical-device claim','checks':checks,'runtimeErrors':dev.errors},ensure_ascii=False,indent=2),encoding='utf-8')
        print(f'Student UI passed: {len(checks)} layouts; favorites, filters, 4 methods, copy fallback, lifecycle, login return, logout. No servers started.')
    except Exception:
        if dev:
            (OUT/'failure.txt').write_text(dev.evaluate('document.body.innerText'),encoding='utf-8');shot('failure')
        raise
    finally:
        if dev:dev.socket.close()
        process.terminate();process.wait(timeout=15);profile.cleanup()


if __name__=='__main__':main()
