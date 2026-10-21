import sys,json,time,tempfile,subprocess,urllib.request,base64
from pathlib import Path
ROOT = Path(__file__).resolve().parent.parent
ARTIFACTS = ROOT / 'tests' / 'artifacts'
ARTIFACTS.mkdir(exist_ok=True)
sys.path.insert(0,str(ROOT/'tests'))
from run_browser_checks import DevTools
for endpoint in ('health','ready'):
 with urllib.request.urlopen('http://127.0.0.1:8080/api/'+endpoint) as response:
  assert response.status==200 and json.load(response)['status']=='ok'
profile=tempfile.TemporaryDirectory(prefix='koc-foundation-',ignore_cleanup_errors=True)
process=subprocess.Popen(['C:/Program Files/Google/Chrome/Application/chrome.exe','--headless=new','--disable-gpu','--no-first-run','--remote-debugging-port=0','--user-data-dir='+profile.name,'about:blank'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,creationflags=subprocess.CREATE_NO_WINDOW)
try:
 portfile=Path(profile.name)/'DevToolsActivePort'
 for _ in range(100):
  if portfile.exists(): break
  time.sleep(.1)
 port=portfile.read_text().splitlines()[0]
 targets=json.load(urllib.request.urlopen(f'http://127.0.0.1:{port}/json/list'))
 dev=DevTools(next(t['webSocketDebuggerUrl'] for t in targets if t['type']=='page'))
 dev.call('Runtime.enable');dev.call('Page.enable');dev.call('Network.enable')
 dev.call('Page.navigate',{'url':'http://127.0.0.1:5173/#system'})
 for _ in range(150):
  if dev.evaluate("document.querySelector('[role=status]')?.textContent.includes('şeması hazır')"): break
  time.sleep(.1)
 assert dev.evaluate("document.querySelector('[role=status]')?.textContent.includes('şeması hazır')"),'readiness never appeared'
 for width in (1440,390,320):
  dev.call('Emulation.setDeviceMetricsOverride',{'width':width,'height':900,'deviceScaleFactor':1,'mobile':width<500})
  assert dev.evaluate('document.documentElement.scrollWidth <= innerWidth'),f'overflow {width}'
  shot=dev.call('Page.captureScreenshot',{'format':'png'})
  (ARTIFACTS / f'foundation-{width}.png').write_bytes(base64.b64decode(shot['data']))
 dev.call('Network.setBlockedURLs',{'urls':['*/api/ready']})
 dev.evaluate("document.querySelector('button').click()")
 for _ in range(100):
  if dev.evaluate("document.querySelector('[role=status]').textContent.includes('kurulamadı')"): break
  time.sleep(.1)
 assert dev.evaluate("document.querySelector('[role=status]').textContent.includes('kurulamadı')"),'missing error state'
 dev.call('Network.setBlockedURLs',{'urls':[]})
 dev.evaluate("document.querySelector('button').click()")
 for _ in range(100):
  if dev.evaluate("document.querySelector('[role=status]')?.textContent.includes('şeması hazır')"): break
  time.sleep(.1)
 assert dev.evaluate("document.querySelector('[role=status]')?.textContent.includes('şeması hazır')"),'retry failed'
 assert not dev.errors,dev.errors
 print('Foundation browser checks passed: live API, database readiness, 1440/390/320 widths, network error and retry; no runtime errors.')
finally:
 process.terminate();process.wait(timeout=15);profile.cleanup()
