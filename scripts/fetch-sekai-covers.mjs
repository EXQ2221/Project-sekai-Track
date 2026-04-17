import fs from "node:fs/promises";
import path from "node:path";
import { chromium } from "playwright";

const TARGET_URL = "https://sekai.best/music";
const OUT_DIR = path.resolve("static/assets");

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function extFromUrl(rawUrl) {
  try {
    const pathname = new URL(rawUrl).pathname.toLowerCase();
    if (pathname.endsWith(".png")) return ".png";
    if (pathname.endsWith(".jpg") || pathname.endsWith(".jpeg")) return ".jpg";
    if (pathname.endsWith(".webp")) return ".webp";
    return ".jpg";
  } catch {
    return ".jpg";
  }
}

async function download(url, outFile) {
  const res = await fetch(url, {
    headers: { "User-Agent": "Mozilla/5.0 pjsk-cover-fetcher" },
  });
  if (!res.ok) {
    throw new Error(`download failed ${res.status}`);
  }
  const buf = Buffer.from(await res.arrayBuffer());
  await fs.writeFile(outFile, buf);
}

async function main() {
  await fs.mkdir(OUT_DIR, { recursive: true });

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await page.goto(TARGET_URL, { waitUntil: "domcontentloaded", timeout: 120000 });

  for (let i = 0; i < 60; i += 1) {
    await page.mouse.wheel(0, 5000);
    await sleep(200);
  }

  const items = await page.evaluate(() => {
    const list = [];
    const cards = Array.from(document.querySelectorAll("a[href*='/music/']"));
    for (const a of cards) {
      const href = a.getAttribute("href") || "";
      const m = href.match(/\/music\/(\d+)/);
      if (!m) continue;
      const id = m[1];
      const img = a.querySelector("img");
      const src = img?.getAttribute("src") || img?.getAttribute("data-src") || "";
      if (!src) continue;
      const full = src.startsWith("http") ? src : new URL(src, location.origin).href;
      list.push({ id, url: full });
    }
    const dedup = new Map();
    for (const it of list) {
      if (!dedup.has(it.id)) dedup.set(it.id, it.url);
    }
    return Array.from(dedup, ([id, url]) => ({ id, url }));
  });

  console.log(`found ${items.length} covers`);

  let ok = 0;
  for (const it of items) {
    const ext = extFromUrl(it.url);
    const outFile = path.join(OUT_DIR, `${it.id}${ext}`);
    try {
      await download(it.url, outFile);
      ok += 1;
      console.log(`saved ${it.id}${ext}`);
      await sleep(120);
    } catch (err) {
      console.warn(`skip ${it.id}: ${err.message}`);
    }
  }

  await fs.writeFile(path.join(OUT_DIR, "_cover_map.json"), JSON.stringify(items, null, 2), "utf-8");
  await browser.close();
  console.log(`done, saved ${ok}/${items.length}`);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});

