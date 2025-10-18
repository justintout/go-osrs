from playwright.sync_api import sync_playwright, expect

def run(playwright):
    browser = playwright.chromium.launch()
    page = browser.new_page()
    page.goto("http://localhost:8080")
    page.get_by_role("button", name="Calculate Profits").click()
    expect(page.locator("table")).to_be_visible(timeout=20000)
    page.screenshot(path="cmd/saplings/screenshot.png")
    browser.close()

with sync_playwright() as playwright:
    run(playwright)
