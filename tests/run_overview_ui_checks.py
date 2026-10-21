"""Check overview UI on a user-started Vite server; all /api calls are synthetic.

KOC_OVERVIEW_URL=http://192.168.1.94:5173 python tests/run_overview_ui_checks.py
No backend/frontend server is launched, no real account or database is accessed.
API integration is tested separately in Go/CI. Chrome emulates touch and sizes.
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
OUT = ROOT / 'tests/artifacts/overview'


def main():
    base = os.environ.get('KOC_OVERVIEW_URL', '').rstrip('/')
    parsed = urllib.parse.urlsplit(base)
    if parsed.scheme != 'http' or not parsed.hostname or not ipaddress.ip_address(parsed.hostname).is_private or parsed.path or parsed.query or parsed.fragment or parsed.username:
        raise SystemExit('Set KOC_OVERVIEW_URL to the existing local/LAN Vite origin.')
    today = datetime.datetime.now(datetime.timezone(datetime.timedelta(hours=3)))
    now = today.isoformat()
    data = {
        'generated_at': now,
        'businesses': {'total': 18, 'active': 12, 'in_progress': 5, 'ended': 1},
        'benefits': {'total': 26, 'published': 18, 'active': 12, 'scheduled': 2, 'expired': 4, 'draft': 5, 'suspended': 3, 'ending_soon': 7},
        'students': {'total': 168, 'active': 143, 'pending_review': 19, 'pending_email': 12, 'review_ready': 11},
        'expiring_benefits': [
            {'id': f'00000000-0000-4000-8000-{i:012d}', 'title': title, 'business_name': name, 'valid_until': (today+datetime.timedelta(days=i)).date().isoformat()}
            for i, (title, name) in enumerate([('Kahvelerde %15 indirim', 'Örnek Kafe'), ('Seçili ürünlerde %20 indirim', 'Örnek Mağaza'), ('İlk derse katılım hediyesi', 'Örnek Spor'), ('Etkinlik biletinde öğrenci avantajı', 'Örnek Sahne'), ('Kitap alışverişine özel fırsat', 'Örnek Kitabevi')])
        ],
        'recent_activity': [
            {'id': str(i), 'action': action, 'actor_name': 'Test yöneticisi', 'entity_type': 'benefit' if action.startswith('benefit') else 'business' if action.startswith('business') else 'account', 'entity_id': f'00000000-0000-4000-8000-{i:012d}', 'entity_label': label, 'created_at': now}
            for i, (action, label) in enumerate([('benefit.publish', 'Kahvelerde %15 indirim'), ('account.review.approved', 'Örnek öğrenci'), ('business.save', 'Örnek Kafe'), ('benefit.save', 'Etkinlik biletinde öğrenci avantajı'), ('benefit.suspend', 'Örnek avantaj'), ('account.role.admin', 'Örnek yönetici')], 1)
        ],
    }
    session = {'user': {'id': '00000000-0000-4000-8000-000000000001', 'name': 'Test yöneticisi', 'username': 'SyntheticOperator', 'email': '', 'role': 'admin', 'status': 'active', 'eligibility_status': 'approved', 'email_verified': False}, 'csrf_token': 'synthetic', 'school_domain': 'ogrenci.example', 'registration_open': True, 'local_email': True}
    script = 'window.__overviewFixture='+json.dumps(data, ensure_ascii=False)+';window.__sessionFixture='+json.dumps(session, ensure_ascii=False)+r""";
    window.__overviewMode=sessionStorage.getItem('overview-test-mode')||'full';
    window.__overviewCalls=[];
    const originalFetch=window.fetch.bind(window);
    window.fetch=async(input,init)=>{
      const url=new URL(input instanceof Request?input.url:String(input),location.href);
      if(!url.pathname.startsWith('/api/'))return originalFetch(input,init);
      window.__overviewCalls.push(url.pathname+url.search);
      const response=(body,status=200)=>new Response(JSON.stringify(body),{status,headers:{'Content-Type':'application/json'}});
      if(url.pathname==='/api/auth/session')return response(window.__sessionFixture);
      if(url.pathname==='/api/admin/overview'){
        await new Promise(resolve=>setTimeout(resolve,400));
        if(window.__overviewMode==='error')return response({code:'unavailable',message:'Synthetic failure'},503);
        if(window.__overviewMode==='forbidden')return response({code:'forbidden',message:'Synthetic permission loss'},403);
        const result=structuredClone(window.__overviewFixture);
        if(window.__overviewMode==='empty'){
          for(const group of ['businesses','benefits','students'])for(const key in result[group])result[group][key]=0;
          result.expiring_benefits=[];result.recent_activity=[];
        }
        if(window.__overviewMode==='long'){
          result.expiring_benefits[0].title='UzunAvantajBaşlığı'.repeat(25);
          result.recent_activity[0].entity_label='UzunİşletmeAdı'.repeat(25);
        }
        return response(result);
      }
      if(url.pathname==='/api/admin/businesses')return response({businesses:[],truncated:false});
      if(url.pathname==='/api/admin/benefits')return response({benefits:[],truncated:false});
      if(url.pathname==='/api/admin/accounts')return response({accounts:[],truncated:false});
      if(url.pathname==='/api/admin/settings')return response({settings:{platform_name:'KoçBenefits',support_contact:'',show_featured:true,revision:1},school_domain:'ogrenci.example'});
      // Fail closed: never pass account/API operations to the development backend.
      return response({code:'unexpected_test_request'},500);
    };
    """
    OUT.mkdir(parents=True, exist_ok=True)
    chrome = os.environ.get('KOC_CHROME', 'C:/Program Files/Google/Chrome/Application/chrome.exe')
    profile = tempfile.TemporaryDirectory(prefix='koc-overview-ui-', ignore_cleanup_errors=True)
    process = subprocess.Popen([chrome, '--headless=new', '--disable-gpu', '--no-first-run', '--remote-debugging-port=0', '--user-data-dir='+profile.name, 'about:blank'], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
    dev = None
    checks = []
    def wait(test, label):
        deadline = time.monotonic()+20
        while time.monotonic()<deadline:
            try:
                if test(): return
            except (OSError, ValueError): pass
            time.sleep(.05)
        raise AssertionError(label)
    try:
        portfile = Path(profile.name)/'DevToolsActivePort'
        wait(lambda: portfile.exists() and bool(portfile.read_text()), 'Chrome startup')
        port = portfile.read_text().splitlines()[0]
        with urllib.request.urlopen(f'http://127.0.0.1:{port}/json/list') as r: targets=json.load(r)
        dev=DevTools(next(t['webSocketDebuggerUrl'] for t in targets if t['type']=='page'))
        for method in ('Page.enable', 'Runtime.enable', 'Network.enable'): dev.call(method)
        dev.call('Emulation.setFocusEmulationEnabled', {'enabled':True})
        dev.call('Page.addScriptToEvaluateOnNewDocument', {'source':script})
        def present(selector): wait(lambda: dev.evaluate(f'!!document.querySelector({json.dumps(selector)})'), 'missing '+selector)
        def text(value): wait(lambda: dev.evaluate(f'document.body.innerText.includes({json.dumps(value)})'), 'missing text '+value)
        def click(selector):
            present(selector)
            assert dev.evaluate(f"(()=>{{const e=document.querySelector({json.dumps(selector)});if(e.disabled)return false;e.click();return true}})()"), 'disabled '+selector
        def size(w,h):
            dev.call('Emulation.setDeviceMetricsOverride', {'width':w,'height':h,'deviceScaleFactor':1,'mobile':w<=1100})
            dev.call('Emulation.setTouchEmulationEnabled', {'enabled':w<=1100})
            time.sleep(.25)
        def shot(name):
            result=dev.call('Page.captureScreenshot', {'format':'png','captureBeyondViewport':False})
            (OUT/(name+'.png')).write_bytes(base64.b64decode(result['data']))
        def layout(name,w,h):
            size(w,h)
            widths=dev.evaluate('({viewport:innerWidth,page:document.documentElement.scrollWidth})')
            assert widths['page']<=widths['viewport'], f'overflow {name} {w}: {widths}'
            checks.append({'state':name,'width':w,'height':h})
            dev.evaluate('window.scrollTo(0,0)');shot(f'{name}-{w}')
        def refresh(mode):
            dev.evaluate('window.__overviewMode='+json.dumps(mode))
            click('.overview-intro button')
            wait(lambda:dev.evaluate("!document.querySelector('.overview-intro button')?.disabled"),'refresh finished')
        def reload(mode):
            dev.evaluate('sessionStorage.setItem("overview-test-mode",'+json.dumps(mode)+')')
            dev.call('Page.reload');present('.overview')
        size(1440,1000)
        dev.call('Page.navigate', {'url':base+'/#admin/overview'})
        present('.overview-loading');shot('loading')
        present('.overview-metrics')
        assert dev.evaluate("Array.from(document.querySelectorAll('.overview-metric > strong')).map(x=>x.textContent)")==['18','12','143','19']
        text('İşletme kaydedildi');text('ilk 5 kayıt')
        assert dev.evaluate("document.querySelectorAll('.overview-activity li').length")==6
        for w,h in [(320,740),(390,844),(768,1024),(1024,768),(1440,1000),(1920,1080),(844,390)]: layout('full',w,h)
        size(390,844);click('a.overview-action[href="#admin/overview"]')
        assert dev.evaluate("document.activeElement.id==='overview-expiring'")
        shot('expiry-phone')
        click('.overview-metric[href="#admin/benefits?status=active"]')
        present('.admin-toolbar')
        wait(lambda:dev.evaluate("window.__overviewCalls.some(x=>x.startsWith('/api/admin/benefits?')&&new URL('http://test'+x).searchParams.get('status')==='active')"),'active filter request')
        dev.evaluate("location.hash='admin/overview'");present('.overview-metrics')
        click('a.overview-action[href="#admin/benefits?status=draft"]')
        present('.admin-toolbar')
        wait(lambda:dev.evaluate("window.__overviewCalls.some(x=>x.startsWith('/api/admin/benefits?')&&new URL('http://test'+x).searchParams.get('status')==='draft')"),'draft filter request')
        dev.evaluate("location.hash='admin/overview'");present('.overview-metrics')
        refresh('long')
        for w,h in [(320,740),(1440,1000)]: layout('long',w,h)
        refresh('error');text('Son alınan bilgiler gösteriliyor')
        assert dev.evaluate("document.querySelectorAll('.overview-metric').length")==4
        for w,h in [(320,740),(1440,1000)]: layout('stale-error',w,h)
        refresh('full');wait(lambda:not dev.evaluate("!!document.querySelector('.overview-warning')"),'warning cleared')
        refresh('empty');text('Henüz avantaj kaydı yok')
        assert dev.evaluate("Array.from(document.querySelectorAll('.overview-metric > strong')).every(x=>x.textContent==='0')")
        for w,h in [(320,740),(1440,1000)]: layout('empty',w,h)
        reload('error');text('Genel bakış alınamadı')
        assert not dev.evaluate("!!document.querySelector('.overview-metrics')")
        for w,h in [(320,740),(1440,1000)]: layout('initial-error',w,h)
        dev.evaluate("window.__overviewMode='full'");click('.overview-warning button');present('.overview-metrics')
        refresh('forbidden')
        wait(lambda:not dev.evaluate("!!document.querySelector('.overview-metrics')"),'privileged data retained on permission loss')
        for w,h in [(320,740),(1440,1000)]: layout('permission-loss',w,h)
        assert not dev.errors, 'JavaScript runtime errors'
        (OUT/'report.json').write_text(json.dumps({'scope':'Real Vite UI with synthetic API responses; no live backend/data or physical-device verification','checks':checks,'runtimeErrors':dev.errors},ensure_ascii=False,indent=2),encoding='utf-8')
        print(f'Overview UI passed: {len(checks)} layouts, loading/empty/error/retry/stale data, permission loss, filtered links; no servers started.')
    except Exception:
        if dev:
            (OUT/'failure.txt').write_text(dev.evaluate('document.body.innerText'),encoding='utf-8')
            shot('failure')
        raise
    finally:
        if dev: dev.socket.close()
        process.terminate();process.wait(timeout=15);profile.cleanup()


if __name__=='__main__': main()
