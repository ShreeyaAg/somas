"""Regression checks for readable agents and authoritative sunlight/energy UI.

Run against a test server (the script creates experiments):
  SOMAS_TEST_URL=http://127.0.0.1:8080 SOMAS_TEST_TOKEN=... python3 scripts/browser_clarity.py
Requires Python Playwright and Chromium (CHROMIUM may override its path).
"""
import json
import os
import re
from pathlib import Path
from playwright.sync_api import sync_playwright, expect

URL = os.environ.get("SOMAS_TEST_URL", "http://127.0.0.1:8080").rstrip("/")
TOKEN = os.environ["SOMAS_TEST_TOKEN"]
Path("work").mkdir(exist_ok=True)


def non_overlapping_agents(page, expected_count):
    """Use actual painted group extents, including labels/bars, not model positions."""
    expect(page.locator("#world-map .map-agent")).to_have_count(expected_count)
    boxes = page.evaluate("""() => [...document.querySelectorAll("#world-map .map-agent")].map(n => {
        const r=n.getBoundingClientRect();
        return {id:n.dataset.agent,x:r.x,y:r.y,w:r.width,h:r.height};
    })""")
    assert len(boxes) == expected_count, (len(boxes), expected_count)
    for a in boxes:
        assert a["w"] > 0 and a["h"] > 0, f"Invisible agent: {a}"
    for i, a in enumerate(boxes):
        for b in boxes[i + 1 :]:
            overlap_w = min(a["x"] + a["w"], b["x"] + b["w"]) - max(a["x"], b["x"])
            overlap_h = min(a["y"] + a["h"], b["y"] + b["h"]) - max(a["y"], b["y"])
            assert overlap_w <= 0.25 or overlap_h <= 0.25, f"Agent displays overlap: {a} / {b}"


def main():
    with sync_playwright() as p:
        browser = p.chromium.launch(
            executable_path=os.environ.get("CHROMIUM", "/usr/bin/chromium"),
            headless=True,
            args=["--no-sandbox"],
        )
        page = browser.new_page(viewport={"width": 1440, "height": 1100}, device_scale_factor=1)
        errors = []
        page.on("pageerror", lambda e: errors.append(str(e)))
        page.on("console", lambda m: errors.append(m.text) if m.type == "error" else None)
        page.goto(URL)
        page.locator("#token").fill(TOKEN)
        page.get_by_role("button", name="Enter laboratory").click()
        expect(page.locator("#workspace")).to_be_visible()

        def setup_open():
            details = page.locator(".setup > details")
            if details.get_attribute("open") is None:
                details.locator(":scope > summary").click()

        def create(agents, spatial=False, automatic=False, institutions=1, rounds=4):
            setup_open()
            page.locator("#agents").fill(str(agents))
            page.locator("#spatial").set_checked(spatial)
            page.locator("#auto-patches").set_checked(automatic)
            if not automatic:
                page.locator("#institutions").fill(str(institutions))
            else:
                page.locator("#agents-per-patch").fill("5")
            page.locator("#rounds").fill(str(rounds))
            page.locator("#mix").select_option("random,cooperative,roaming")
            page.locator("#harvest-access").select_option("open")
            with page.expect_response(
                lambda r: r.url.endswith("/api/runs") and r.request.method == "POST"
            ) as created:
                page.get_by_role("button", name="Create experiment", exact=True).click()
            assert created.value.status == 201, created.value.text()
            run = created.value.json()
            expect(page.locator("#experiments")).to_have_value(run["id"])
            expect(page.locator("#world-map .map-agent")).to_have_count(agents)
            page.locator(".setup > details > summary").click()
            return run

        def state(run):
            response = page.request.get(
                URL + "/api/runs/" + run["id"], headers={"Authorization": "Bearer " + TOKEN}
            )
            assert response.ok, response.text()
            return response.json()

        def step(run):
            with page.expect_response(
                lambda r: r.url.endswith("/api/runs/" + run["id"] + "/control")
                and r.request.method == "POST"
            ) as advanced:
                page.locator('[data-action="step"]').click()
            assert advanced.value.ok, advanced.value.text()
            return advanced.value.json()

        # Reproduce the user's screenshot: ten agents in classic mode on one patch.
        classic = create(10)
        non_overlapping_agents(page, 10)
        expect(page.locator("#agent-cards .agent-card")).to_have_count(10)
        card = page.locator('#agent-cards .agent-card[data-agent="agent-001"]')
        expect(card).to_contain_text("Brain")
        expect(card).to_contain_text("Body")
        expect(card).to_contain_text("Stock")
        first = step(classic)
        activities = [a["activity"] for a in first["state"]["players"] if a.get("activity")]
        assert len(activities) == 10
        actual_gain = sum(a["sunlight_gain"] for a in activities)
        expect(page.locator("#sun-total")).to_contain_text(f"{actual_gain:.2f}")
        assert actual_gain > 0, "Fixture did not receive sunlight"
        non_overlapping_agents(page, 10)
        step(classic)
        round_end = step(classic)
        expect(page.locator("#phase")).to_have_text("round end")
        expect(card).to_contain_text("Harvest")
        expect(card).to_contain_text("Eat")
        non_overlapping_agents(page, 10)
        page.locator('#world-map .map-agent[data-agent="agent-002"]').click()
        expect(page.locator('#agent-cards .agent-card[data-agent="agent-002"]')).to_have_class(re.compile(r"\bselected\b"))
        expect(page.locator("#inspector")).to_contain_text("agent-002")
        page.locator('#world-map .map-agent[data-agent="agent-001"]').click()
        expect(page.locator("#inspector")).to_contain_text("agent-001")
        # Replay must use that phase's recorded telemetry rather than live totals.
        expect(page.locator("#replay")).to_have_attribute("max", "3")
        page.locator("#replay").fill("0")
        page.locator("#replay").dispatch_event("input")
        expect(page.locator("#running")).to_have_text("Replay")
        expect(page.locator("#sun-total")).to_have_text("Not recorded")
        page.locator("#live").click()
        expect(page.locator("#sun-total")).to_contain_text(f"{actual_gain:.2f}")
        page.screenshot(path="work/clarity-desktop.png", full_page=True)

        # Historical data must not claim weather or resource effects that were not recorded.
        old_run = json.loads(json.dumps(round_end))
        for agent in old_run["state"]["players"]:
            agent.pop("activity", None)
        endpoint = "**/api/runs/" + classic["id"]
        page.route(endpoint, lambda route: route.fulfill(json=old_run))
        page.locator("#experiments").select_option(classic["id"])
        expect(page.locator(".sun-panel")).to_contain_text(
            re.compile(r"not recorded|unavailable", re.I)
        )
        page.unroute(endpoint)
        page.locator("#experiments").select_option(classic["id"])

        # Mobile rendering should keep overflow within the map/cards, not the page.
        page.set_viewport_size({"width": 390, "height": 844})
        non_overlapping_agents(page, 10)
        assert page.evaluate("document.documentElement.scrollWidth <= innerWidth + 1")
        page.screenshot(path="work/clarity-mobile.png", full_page=True)
        page.set_viewport_size({"width": 1440, "height": 1100})

        # Automatic scaling and severe crowding must preserve distinct clickable displays.
        auto = create(100, spatial=True, automatic=True, rounds=2)
        assert auto["config"]["institutions"] == 20
        non_overlapping_agents(page, 100)
        step(auto)
        step(auto)
        non_overlapping_agents(page, 100)
        page.screenshot(path="work/clarity-100-agents.png", full_page=True)
        crowded = create(200, spatial=True, automatic=False, institutions=1, rounds=1)
        non_overlapping_agents(page, 200)
        expect(page.locator("#agent-cards .agent-card")).to_have_count(200)
        page.screenshot(path="work/clarity-200-one-patch.png", full_page=True)
        assert page.evaluate("document.documentElement.scrollWidth <= innerWidth + 1")
        page.set_viewport_size({"width": 390, "height": 844})
        non_overlapping_agents(page, 200)
        assert page.evaluate("document.documentElement.scrollWidth <= innerWidth + 1")
        # Hold a real polling response, switch experiments, then release the old
        # response. Its stale state must not replace the newly selected world,
        # resource cards, sunlight totals or history.
        delayed_run = state(crowded)
        held = []
        delayed_endpoint = "**/api/runs/" + crowded["id"]
        page.route(delayed_endpoint, lambda route: held.append(route))
        with page.expect_request(lambda request: request.url.endswith("/api/runs/" + crowded["id"])):
            page.wait_for_timeout(1250)  # allow the scheduled one-second poll
        assert held, "The polling regression did not intercept a request"
        page.locator("#experiments").select_option(classic["id"])
        expect(page.locator("#world-map .map-agent")).to_have_count(10)
        expect(page.locator("#sun-total")).to_contain_text(f"{actual_gain:.2f}")
        with page.expect_response(lambda response: response.url.endswith("/api/runs/" + crowded["id"])):
            held.pop().fulfill(json=delayed_run)
        page.wait_for_timeout(200)  # let the released fetch/JSON continuation run
        expect(page.locator("#experiments")).to_have_value(classic["id"])
        expect(page.locator("#world-map .map-agent")).to_have_count(10)
        expect(page.locator("#agent-cards .agent-card")).to_have_count(10)
        expect(page.locator("#sun-total")).to_contain_text(f"{actual_gain:.2f}")
        expect(page.locator("#replay")).to_have_attribute("max", "3")
        page.unroute(delayed_endpoint)
        assert not errors, errors
        assert page.locator("#error").is_hidden(), page.locator("#error").inner_text()
        print(
            "PASS: classic 10-agent/one-patch crowding, 100-agent automatic map, "
            "200-agent/one-patch crowding; non-overlapping agent displays, resource cards, "
            "backend sunlight totals, archived telemetry fallback, inspection, mobile layout "
            "and rejection of stale polling responses after switching experiments"
        )
        browser.close()


if __name__ == "__main__":
    main()
