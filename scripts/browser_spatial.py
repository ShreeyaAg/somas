"""Exercise the real grid dashboard against a running test server."""
import os,pathlib,json
from playwright.sync_api import sync_playwright,expect
url=os.environ.get('SOMAS_TEST_URL','http://127.0.0.1:8080')
token=os.environ['SOMAS_TEST_TOKEN']
pathlib.Path('work').mkdir(exist_ok=True)
with sync_playwright() as p:
 b=p.chromium.launch(executable_path=os.environ.get('CHROMIUM','/usr/bin/chromium'),headless=True,args=['--no-sandbox'])
 page=b.new_page(viewport={'width':1440,'height':1100},device_scale_factor=1)
 errors=[];page.on('pageerror',lambda e:errors.append(str(e)));page.on('console',lambda m:errors.append(m.text) if m.type=='error' else None)
 page.goto(url);page.locator('#token').fill(token);page.get_by_role('button',name='Enter laboratory').click();expect(page.locator('#workspace')).to_be_visible()
 page.locator('.setup > details > summary').click();page.locator('#agents').fill('12');page.locator('#agents-per-patch').fill('4');page.locator('#rounds').fill('8');page.locator('#harvest-access').select_option('open');page.locator('#mix').select_option('random,cooperative,roaming')
 expect(page.locator('#world-preview')).to_contain_text('3 independent patches')
 with page.expect_response(lambda r:r.url.endswith('/api/runs') and r.request.method=='POST') as created:page.get_by_role('button',name='Create experiment',exact=True).click()
 run=created.value.json();runid=run['id'];expect(page.locator('#experiments')).to_have_value(runid);expect(page.locator('.map-patch')).to_have_count(3);expect(page.locator('.map-agent')).to_have_count(12)
 assert run['config']['institutions']==3 and run['config']['auto_patches']
 initial={x['id']:x['position'] for x in run['state']['players']}
 page.locator('.setup > details > summary').click()
 page.locator('.map-agent').first.click();expect(page.locator('#inspector')).to_contain_text('Position / moves')
 page.locator('.map-patch').nth(1).click();expect(page.locator('#patch-inspector')).to_contain_text('Patch 2')
 page.locator('#zoom-in').click();expect(page.locator('#zoom-level')).to_have_text('150%');page.locator('#zoom-fit').click()
 for _ in range(2):page.locator('[data-action=step]').click()
 expect(page.locator('#phase')).to_have_text('operational')
 state=page.request.get(url+'/api/runs/'+runid,headers={'Authorization':'Bearer '+token}).json()['state']
 assert any(x['position']!=initial[x['id']] for x in state['players']),'No movement occurred'
 for _ in range(3):page.locator('[data-action=step]').click()
 expect(page.locator('#phase')).to_have_text('operational')
 page.screenshot(path='work/grid-desktop.png',full_page=True)
 page.locator('#speed').select_option('50');page.locator('[data-action=play]').click();expect(page.locator('#phase')).to_have_text('complete',timeout=20000)
 page.locator('#replay').fill('0');page.locator('#replay').dispatch_event('input');expect(page.locator('#running')).to_have_text('Replay')
 page.locator('#live').click();page.locator('[data-action=reset]').click();expect(page.locator('#phase')).to_have_text('formation')
 page.set_viewport_size({'width':390,'height':844});assert page.evaluate('document.documentElement.scrollWidth<=innerWidth');page.screenshot(path='work/grid-mobile.png',full_page=True)
 page.set_viewport_size({'width':1440,'height':1100});page.locator('.setup > details > summary').click();page.locator('#agents').fill('100');page.locator('#agents-per-patch').fill('5');page.locator('#rounds').fill('2')
 with page.expect_response(lambda r:r.url.endswith('/api/runs') and r.request.method=='POST') as created:page.get_by_role('button',name='Create experiment',exact=True).click()
 large=created.value.json();expect(page.locator('#experiments')).to_have_value(large['id']);expect(page.locator('.map-patch')).to_have_count(20);expect(page.locator('.map-agent')).to_have_count(100)
 assert large['state']['world']['width']>=12 and large['state']['world']['height']>=9
 page.locator('.setup > details > summary').click();page.screenshot(path='work/grid-100-agents.png',full_page=True)
 assert not errors,errors
 assert page.locator('#error').is_hidden(),page.locator('#error').inner_text()
 print('PASS: automatic 12→3 and 100→20 patch scaling, real movement, patch/agent selection, zoom, play, replay, reset, mobile; no console errors')
 b.close()
