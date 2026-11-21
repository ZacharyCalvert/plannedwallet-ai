import struct
from pathlib import Path
from playwright.sync_api import sync_playwright

BRANDING_DIR = Path(__file__).resolve().parent
SVG_PATH = BRANDING_DIR / "planned-wallet-icon.svg"


def pack_ico(png_entries: list[tuple[int, bytes]]) -> bytes:
    """Pack multiple PNG images (size, png_bytes) into a standard Windows/Browser ICO."""
    count = len(png_entries)
    header = struct.pack("<HHH", 0, 1, count)
    offset = 6 + 16 * count
    directory = bytearray()
    payload = bytearray()

    for size, data in png_entries:
        dim = size if size < 256 else 0
        directory.extend(
            struct.pack("<BBBBHHII", dim, dim, 0, 0, 1, 32, len(data), offset)
        )
        payload.extend(data)
        offset += len(data)

    return header + bytes(directory) + bytes(payload)


def render_svg_png(page, svg_text: str, size: int) -> bytes:
    html = f"""<!DOCTYPE html>
<html>
<head><meta charset="utf-8" />
<style>
  * {{ margin: 0; padding: 0; box-sizing: border-box; }}
  html, body {{ width: {size}px; height: {size}px; background: transparent; overflow: hidden; }}
  svg {{ width: {size}px; height: {size}px; display: block; }}
</style>
</head>
<body>{svg_text}</body>
</html>"""
    page.set_viewport_size({"width": size, "height": size})
    page.set_content(html, wait_until="load")
    return page.screenshot(type="png", omit_background=True)


def build_linkedin_html(svg_text: str) -> str:
    return f"""<!DOCTYPE html>
<html>
<head><meta charset="utf-8" />
<style>
  * {{ margin: 0; padding: 0; box-sizing: border-box; }}
  body {{
    width: 1584px;
    height: 396px;
    overflow: hidden;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
    background: linear-gradient(125deg, #0b1120 0%, #0f172a 48%, #1e293b 100%);
    color: #ffffff;
    position: relative;
    display: flex;
    align-items: center;
    justify-content: flex-end;
    padding: 0 88px 0 380px;
  }}
  .grid-bg {{
    position: absolute;
    inset: 0;
    background-image:
      linear-gradient(to right, rgba(148, 163, 184, 0.06) 1px, transparent 1px),
      linear-gradient(to bottom, rgba(148, 163, 184, 0.06) 1px, transparent 1px);
    background-size: 44px 44px;
    mask-image: linear-gradient(to left, rgba(0,0,0,0.9), rgba(0,0,0,0.15));
  }}
  .glow-blue {{
    position: absolute;
    right: 260px;
    top: -120px;
    width: 480px;
    height: 480px;
    border-radius: 50%;
    background: radial-gradient(circle, rgba(37, 99, 235, 0.28) 0%, rgba(37, 99, 235, 0) 70%);
  }}
  .glow-emerald {{
    position: absolute;
    right: -60px;
    bottom: -140px;
    width: 460px;
    height: 460px;
    border-radius: 50%;
    background: radial-gradient(circle, rgba(16, 185, 129, 0.22) 0%, rgba(16, 185, 129, 0) 70%);
  }}
  .trend-svg {{
    position: absolute;
    inset: 0;
    width: 1584px;
    height: 396px;
    pointer-events: none;
  }}
  .content {{
    position: relative;
    z-index: 2;
    display: flex;
    align-items: center;
    gap: 34px;
    background: rgba(15, 23, 42, 0.58);
    border: 1px solid rgba(148, 163, 184, 0.18);
    border-radius: 20px;
    padding: 28px 40px;
    box-shadow: 0 20px 50px rgba(0, 0, 0, 0.35);
  }}
  .logo-wrap svg {{
    width: 112px;
    height: 112px;
    display: block;
    filter: drop-shadow(0 10px 22px rgba(0, 0, 0, 0.4));
  }}
  .text-col {{
    display: flex;
    flex-direction: column;
    gap: 8px;
  }}
  .brand-row {{
    display: flex;
    align-items: center;
    gap: 14px;
  }}
  h1 {{
    font-size: 46px;
    font-weight: 800;
    letter-spacing: -0.025em;
    color: #ffffff;
    line-height: 1.08;
  }}
  .pill {{
    background: rgba(37, 99, 235, 0.22);
    border: 1px solid rgba(96, 165, 250, 0.45);
    color: #93c5fd;
    font-size: 13px;
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    padding: 5px 12px;
    border-radius: 999px;
  }}
  .tagline {{
    font-size: 22px;
    font-weight: 500;
    color: #e2e8f0;
    letter-spacing: -0.01em;
  }}
  .features {{
    display: flex;
    align-items: center;
    gap: 12px;
    margin-top: 4px;
    flex-wrap: wrap;
  }}
  .chip {{
    font-size: 13.5px;
    font-weight: 600;
    color: #cbd5e1;
    background: rgba(30, 41, 59, 0.85);
    border: 1px solid rgba(148, 163, 184, 0.22);
    padding: 5px 12px;
    border-radius: 8px;
  }}
  .chip strong {{
    color: #34d399;
  }}
</style>
</head>
<body>
  <div class="grid-bg"></div>
  <div class="glow-blue"></div>
  <div class="glow-emerald"></div>
  <svg class="trend-svg" viewBox="0 0 1584 396" fill="none">
    <path d="M 0 340 Q 260 310, 470 250 T 940 165 T 1584 55" stroke="rgba(16, 185, 129, 0.16)" stroke-width="3" fill="none" />
    <path d="M 0 365 Q 320 345, 580 285 T 1120 205 T 1584 115" stroke="rgba(37, 99, 235, 0.18)" stroke-width="2" fill="none" />
  </svg>
  <div class="content">
    <div class="logo-wrap">{svg_text}</div>
    <div class="text-col">
      <div class="brand-row">
        <h1>Planned Wallet</h1>
        <span class="pill">Privacy-First Finance</span>
      </div>
      <div class="tagline">Every Account. Zero Bank Logins.</div>
      <div class="features">
        <span class="chip"><strong>&#10003;</strong> Manual &amp; CSV Control</span>
        <span class="chip"><strong>&#10003;</strong> Checking, Savings, Cards &amp; Funds</span>
        <span class="chip"><strong>&#10003;</strong> Zero AI / Data Brokers</span>
      </div>
    </div>
  </div>
</body>
</html>"""


def build_youtube_html(svg_text: str) -> str:
    return f"""<!DOCTYPE html>
<html>
<head><meta charset="utf-8" />
<style>
  * {{ margin: 0; padding: 0; box-sizing: border-box; }}
  body {{
    width: 2560px;
    height: 1440px;
    overflow: hidden;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
    background: linear-gradient(135deg, #090e1a 0%, #0f172a 45%, #1e293b 100%);
    color: #ffffff;
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
  }}
  .grid-bg {{
    position: absolute;
    inset: 0;
    background-image:
      linear-gradient(to right, rgba(148, 163, 184, 0.055) 1px, transparent 1px),
      linear-gradient(to bottom, rgba(148, 163, 184, 0.055) 1px, transparent 1px);
    background-size: 64px 64px;
  }}
  .glow-left {{
    position: absolute;
    left: 260px;
    top: 260px;
    width: 760px;
    height: 760px;
    border-radius: 50%;
    background: radial-gradient(circle, rgba(37, 99, 235, 0.25) 0%, rgba(37, 99, 235, 0) 70%);
  }}
  .glow-right {{
    position: absolute;
    right: 220px;
    bottom: 180px;
    width: 820px;
    height: 820px;
    border-radius: 50%;
    background: radial-gradient(circle, rgba(16, 185, 129, 0.2) 0%, rgba(16, 185, 129, 0) 70%);
  }}
  .trend-svg {{
    position: absolute;
    inset: 0;
    width: 2560px;
    height: 1440px;
    pointer-events: none;
  }}
  /* YouTube Safe Area: 1546 x 423 centered */
  .safe-area {{
    position: relative;
    z-index: 2;
    width: 1460px;
    height: 380px;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 54px;
    background: rgba(15, 23, 42, 0.65);
    border: 2px solid rgba(148, 163, 184, 0.2);
    border-radius: 32px;
    padding: 40px 64px;
    box-shadow: 0 32px 80px rgba(0, 0, 0, 0.45);
  }}
  .logo-wrap svg {{
    width: 196px;
    height: 196px;
    display: block;
    filter: drop-shadow(0 16px 32px rgba(0, 0, 0, 0.45));
  }}
  .text-col {{
    display: flex;
    flex-direction: column;
    gap: 14px;
  }}
  .eyebrow {{
    font-size: 20px;
    font-weight: 700;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: #34d399;
  }}
  h1 {{
    font-size: 84px;
    font-weight: 800;
    letter-spacing: -0.03em;
    color: #ffffff;
    line-height: 1.02;
  }}
  .tagline {{
    font-size: 34px;
    font-weight: 500;
    color: #cbd5e1;
    letter-spacing: -0.01em;
  }}
  .sub-row {{
    display: flex;
    align-items: center;
    gap: 18px;
    margin-top: 6px;
  }}
  .badge {{
    font-size: 20px;
    font-weight: 600;
    color: #e2e8f0;
    background: rgba(30, 41, 59, 0.9);
    border: 1px solid rgba(96, 165, 250, 0.35);
    padding: 8px 18px;
    border-radius: 10px;
  }}
</style>
</head>
<body>
  <div class="grid-bg"></div>
  <div class="glow-left"></div>
  <div class="glow-right"></div>
  <svg class="trend-svg" viewBox="0 0 2560 1440" fill="none">
    <path d="M 0 1160 Q 480 1050, 920 880 T 1780 580 T 2560 220" stroke="rgba(16, 185, 129, 0.18)" stroke-width="6" fill="none" />
    <path d="M 0 1240 Q 560 1150, 1080 960 T 1940 680 T 2560 360" stroke="rgba(37, 99, 235, 0.2)" stroke-width="4" fill="none" />
  </svg>
  <div class="safe-area">
    <div class="logo-wrap">{svg_text}</div>
    <div class="text-col">
      <div class="eyebrow">Feature Walkthroughs &bull; Architecture &bull; Privacy</div>
      <h1>Planned Wallet</h1>
      <div class="tagline">Every Account. Zero Bank Logins.</div>
      <div class="sub-row">
        <span class="badge">Banks &amp; Credit Cards</span>
        <span class="badge">Retirement &amp; Funds</span>
        <span class="badge">Assets &amp; Net Worth</span>
      </div>
    </div>
  </div>
</body>
</html>"""


def main():
    svg_text = SVG_PATH.read_text(encoding="utf-8")

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()

        # 1. Multi-size favicon.ico (16, 32, 48) + PNG favicons & Apple Touch Icon for Safari
        png_16 = render_svg_png(page, svg_text, 16)
        png_32 = render_svg_png(page, svg_text, 32)
        png_48 = render_svg_png(page, svg_text, 48)
        (BRANDING_DIR / "favicon.ico").write_bytes(pack_ico([(16, png_16), (32, png_32), (48, png_48)]))
        (BRANDING_DIR / "icon-16x16.png").write_bytes(png_16)
        (BRANDING_DIR / "icon-32x32.png").write_bytes(png_32)
        (BRANDING_DIR / "apple-touch-icon.png").write_bytes(render_svg_png(page, svg_text, 180))

        # 2. High-res YouTube Avatar / Icon Preview (800x800)
        avatar_png = render_svg_png(page, svg_text, 800)
        (BRANDING_DIR / "youtube-avatar.png").write_bytes(avatar_png)

        # 3. LinkedIn Personal Banner (1584x396)
        page.set_viewport_size({"width": 1584, "height": 396})
        page.set_content(build_linkedin_html(svg_text), wait_until="load")
        page.screenshot(path=str(BRANDING_DIR / "linkedin-banner.png"), type="png")

        # 4. YouTube Channel Banner (2560x1440)
        page.set_viewport_size({"width": 2560, "height": 1440})
        page.set_content(build_youtube_html(svg_text), wait_until="load")
        page.screenshot(path=str(BRANDING_DIR / "youtube-banner.png"), type="png")

        browser.close()

    print("Generated branding assets in branding/:")
    for name in ("planned-wallet-icon.svg", "favicon.ico", "icon-16x16.png", "icon-32x32.png", "apple-touch-icon.png", "youtube-avatar.png", "linkedin-banner.png", "youtube-banner.png"):
        path = BRANDING_DIR / name
        print(f"  {path.name} ({path.stat().st_size} bytes)")


if __name__ == "__main__":
    main()
