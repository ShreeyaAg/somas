from playwright.sync_api import sync_playwright, expect
import json, os, pathlib
base_url=os.environ.get("SOMAS_TEST_URL","http://127.0.0.1:8080")
token=os.environ["SOMAS_TEST_TOKEN"]
pathlib.Path("work").mkdir(exist_ok=True)
with sync_playwright() as p:
 b=p.chromium.launch(executable_path='/usr/bin/chromium',headless=True,args=['--no-sandbox'])
 page=b.new_page(viewport={'width':1440,'height':1100},device_scale_factor=1)
 errors=[];page.on('pageerror',lambda e: errors.append(str(e)));page.on('console',lambda m: errors.append(m.text) if m.type=='error' else None)
 page.goto(base_url)
 page.locator('#token').fill(token)
 page.get_by_role('button',name='Enter laboratory').click()
 page.locator('.setup > details > summary').click()
 page.locator('#rounds').fill('3')
 page.locator('#mix').select_option('random,cooperative')
 with page.expect_response(lambda response: response.url.endswith('/api/runs') and response.request.method=='POST') as created:
  page.get_by_role('button',name='Create experiment',exact=True).click()
 run_id=created.value.json()['id']
 expect(page.locator('#experiments')).to_have_value(run_id)
 page.wait_for_selector('.map-patch')
 assert page.locator('.map-patch').count()==2
 page.locator('[data-action=step]').click()
 expect(page.locator("#phase")).to_have_text("political")
 page.locator('.agent-row').first.click()
 assert 'Brain' in page.locator('#inspector').inner_text()
 page.locator('#speed').select_option('50')
 page.locator('[data-action=play]').click()
 expect(page.locator("#phase")).to_have_text("complete",timeout=20000)
 assert 'Day 03' in page.locator('#day').inner_text()
 page.locator('#replay').fill('2');page.locator('#replay').dispatch_event('input')
 assert 'Replay' in page.locator('#running').inner_text()
 page.locator('#live').click()
 page.screenshot(path='work/dashboard-desktop.png',full_page=True)
 page.locator('[data-action=reset]').click()
 expect(page.locator("#phase")).to_have_text("formation")
 page.set_viewport_size({'width':390,'height':844})
 page.screenshot(path='work/dashboard-mobile.png',full_page=True)
 assert page.evaluate('document.documentElement.scrollWidth<=innerWidth')
 assert not errors,errors
 assert page.locator('#error').is_hidden(),page.locator('#error').inner_text()
 print('PASS: login, create, two spatial patches, politics, inspector, speed/play, complete, replay, reset, responsive layout; no JS errors')
 b.close()
