const state = {
  apiBase: localStorage.getItem("api_base") || defaultApiBase(),
  token: localStorage.getItem("token") || "",
  refreshToken: localStorage.getItem("refresh_token") || "",
  b30CalcMode: (localStorage.getItem("b30_calc_mode") || "official") === "const" ? "const" : "official",
  statsDifficulty: localStorage.getItem("stats_difficulty") || "master",
  statsMode: localStorage.getItem("stats_mode") || "by_difficulty",
  statsMinLevel: Number(localStorage.getItem("stats_min_level") || 26),
  statsMaxLevel: Number(localStorage.getItem("stats_max_level") || 32),
  page: 1,
  size: 20,
  total: 0,
  keyword: "",
  filters: [],
  sort: "newest",
  musics: [],
  statuses: {},
  achievementMap: {},
  currentSelect: null,
  currentMusicDetail: null,
  profile: null,
  trendPoints: [],
  statistics: null,
  characterOptions: [],
  crop: {
    imageURL: "",
    imageLoaded: false,
    fileSelected: false,
    naturalW: 0,
    naturalH: 0,
    minScale: 1,
    scale: 1,
    x: 0,
    y: 0,
    dragging: false,
    dragStartX: 0,
    dragStartY: 0,
    dragOriginX: 0,
    dragOriginY: 0,
  },
};

const difficultyOrder = { easy: 1, normal: 2, hard: 3, expert: 4, master: 5, append: 6 };
let refreshingPromise = null;

function defaultApiBase() {
  if (location.protocol === "http:" || location.protocol === "https:") return location.origin;
  return "http://localhost:8080";
}

function $(id) {
  return document.getElementById(id);
}

function on(id, event, handler) {
  const el = $(id);
  if (el) el.addEventListener(event, handler);
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}

function notify(message, type = "success") {
  let host = $("toastHost");
  if (!host) {
    host = document.createElement("div");
    host.id = "toastHost";
    host.className = "toast-host";
    document.body.appendChild(host);
  }
  const el = document.createElement("div");
  el.className = `toast ${type}`;
  el.textContent = message;
  host.appendChild(el);
  requestAnimationFrame(() => el.classList.add("show"));
  setTimeout(() => {
    el.classList.remove("show");
    setTimeout(() => el.remove(), 220);
  }, 1800);
}

function withBase(path) {
  return `${state.apiBase.replace(/\/+$/, "")}${path}`;
}

function toAbsoluteAsset(urlPath) {
  if (!urlPath) return "";
  if (/^https?:\/\//i.test(urlPath)) return urlPath;
  return withBase(urlPath.startsWith("/") ? urlPath : `/${urlPath}`);
}

function setAuth(accessToken, refreshToken) {
  state.token = accessToken || "";
  state.refreshToken = refreshToken || "";
  if (state.token) localStorage.setItem("token", state.token);
  else localStorage.removeItem("token");
  if (state.refreshToken) localStorage.setItem("refresh_token", state.refreshToken);
  else localStorage.removeItem("refresh_token");
}

function clearAuth() {
  setAuth("", "");
}

function renderAuthPanel(username = "") {
  const loggedIn = !!state.token;
  const guest = $("authGuest");
  const user = $("authUser");
  const name = $("authUserName");
  if (!guest || !user || !name) return;
  if (loggedIn) {
    guest.classList.add("hidden-ui");
    user.classList.remove("hidden-ui");
    name.textContent = username || state.profile?.username || "已登录用户";
  } else {
    user.classList.add("hidden-ui");
    guest.classList.remove("hidden-ui");
    name.textContent = "-";
  }
}

function apiErrorMessage(payload, res) {
  if (payload && typeof payload.message === "string" && payload.message.trim()) return payload.message.trim();
  return `HTTP ${res.status}`;
}

async function refreshAccessToken() {
  if (refreshingPromise) return refreshingPromise;
  if (!state.refreshToken) throw new Error("missing refresh token");

  refreshingPromise = (async () => {
    const res = await fetch(withBase("/refresh"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: state.refreshToken }),
    });
    const payload = await res.json().catch(() => ({}));
    if (!res.ok || (typeof payload.code === "number" && payload.code >= 400)) {
      throw new Error(apiErrorMessage(payload, res));
    }
    const data = payload.data || {};
    const newAccessToken = data.access_token || data.token || "";
    const newRefreshToken = data.refresh_token || state.refreshToken;
    if (!newAccessToken) {
      throw new Error("refresh missing access token");
    }
    setAuth(newAccessToken, newRefreshToken);
  })();

  try {
    await refreshingPromise;
  } finally {
    refreshingPromise = null;
  }
}

async function requestJSON(path, { method = "GET", body, auth = false, form = false, retried = false } = {}) {
  const headers = {};
  if (!form && body !== undefined) headers["Content-Type"] = "application/json";
  if (auth && state.token) headers.Authorization = `Bearer ${state.token}`;

  const res = await fetch(withBase(path), {
    method,
    headers,
    body: body === undefined ? undefined : form ? body : JSON.stringify(body),
  });

  const payload = await res.json().catch(() => ({}));

  if (auth && res.status === 401 && !retried && state.refreshToken) {
    try {
      await refreshAccessToken();
      return requestJSON(path, { method, body, auth, form, retried: true });
    } catch {
      clearAuth();
      renderAuthPanel("");
      throw new Error("登录已过期，请重新登录");
    }
  }

  if (!res.ok || (typeof payload.code === "number" && payload.code >= 400)) {
    throw new Error(apiErrorMessage(payload, res));
  }
  return payload.data;
}

async function fetchBinary(path, { auth = false, retried = false } = {}) {
  const headers = {};
  if (auth && state.token) headers.Authorization = `Bearer ${state.token}`;
  const res = await fetch(withBase(path), { method: "GET", headers });

  if (auth && res.status === 401 && !retried && state.refreshToken) {
    try {
      await refreshAccessToken();
      return fetchBinary(path, { auth, retried: true });
    } catch {
      clearAuth();
      renderAuthPanel("");
      throw new Error("登录已过期，请重新登录");
    }
  }

  if (!res.ok) {
    throw new Error(`HTTP ${res.status}`);
  }

  return res.blob();
}

async function api(path, { method = "GET", body, auth = false } = {}) {
  return requestJSON(path, { method, body, auth, form: false });
}

async function apiMultipart(path, formData, { auth = false } = {}) {
  return requestJSON(path, { method: "POST", body: formData, auth, form: true });
}

function initTabs() {
  document.querySelectorAll(".tab").forEach((btn) => {
    btn.addEventListener("click", () => {
      document.querySelectorAll(".tab").forEach((n) => n.classList.remove("active"));
      document.querySelectorAll(".tab-panel").forEach((n) => n.classList.remove("active"));
      btn.classList.add("active");
      $(`${btn.dataset.tab}Tab`).classList.add("active");
    });
  });
}

function normalizeAchievement(raw) {
  const s = String(raw || "").toLowerCase();
  if (s.includes("all_perfect") || s.includes("all perfect") || s === "ap") return "all_perfect";
  if (s.includes("full_combo") || s.includes("full combo") || s === "fc") return "full_combo";
  if (s.includes("clear")) return "clear";
  return "not_played";
}

function markerClassByStatus(status) {
  if (status === "clear") return "clear";
  if (status === "full_combo") return "fc";
  if (status === "all_perfect") return "ap";
  return "";
}

function normalizeDifficultyKey(raw) {
  const key = String(raw || "").trim().toLowerCase();
  if (key === "matser") return "master";
  return difficultyOrder[key] ? key : "unknown";
}

function prettyDifficulty(raw) {
  const key = normalizeDifficultyKey(raw);
  if (key === "unknown") return String(raw || "-").trim() || "-";
  return key.toUpperCase();
}

function readDifficultyID(diff) {
  const id = Number(diff?.id ?? diff?.music_difficulty_id ?? diff?.musicDifficultyID ?? 0);
  return Number.isFinite(id) && id > 0 ? id : 0;
}

function readDifficultyType(diff) {
  return normalizeDifficultyKey(diff?.musicDifficulty ?? diff?.music_difficulty ?? "");
}

function readDifficultyLevel(diff) {
  const lv = Number(diff?.playLevel ?? diff?.play_level ?? 0);
  return Number.isFinite(lv) && lv > 0 ? lv : 0;
}

function achievementLabel(status) {
  if (status === "all_perfect") return "ALL PERFECT";
  if (status === "full_combo") return "FULL COMBO";
  if (status === "clear") return "CLEAR";
  return "NOT PLAYED";
}

function calcModeLabel(raw) {
  const mode = String(raw || "").trim().toLowerCase();
  if (mode === "const") return "定数模式";
  if (mode === "official") return "官方等级模式";
  return mode || "-";
}

function achievementShortLabel(status) {
  if (status === "all_perfect") return "AP";
  if (status === "full_combo") return "FC";
  if (status === "clear") return "CLEAR";
  return "NP";
}

function achievementClass(status) {
  if (status === "all_perfect") return "ap";
  if (status === "full_combo") return "fc";
  if (status === "clear") return "clear";
  return "not-played";
}

function formatTrendModeLabel(mode) {
  if (mode === "const") return "当前：定数计算";
  return "当前：官方等级计算";
}

function formatTrendTime(raw, compact = false) {
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) return "-";
  const month = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  const hour = String(d.getHours()).padStart(2, "0");
  if (compact) return `${month}-${day} ${hour}:00`;
  const minute = String(d.getMinutes()).padStart(2, "0");
  return `${month}-${day} ${hour}:${minute}`;
}

function trendPalette(mode) {
  if (mode === "const") {
    return {
      line: "#ad76ff",
      glow: "rgba(173,118,255,0.35)",
      areaTop: "rgba(173,118,255,0.28)",
      areaBottom: "rgba(173,118,255,0.03)",
      point: "#c8a0ff",
    };
  }
  return {
    line: "#58a8ff",
    glow: "rgba(88,168,255,0.32)",
    areaTop: "rgba(88,168,255,0.24)",
    areaBottom: "rgba(88,168,255,0.03)",
    point: "#7fc2ff",
  };
}

function pickTickIndices(total, maxTicks = 6) {
  if (total <= 0) return [];
  if (total <= maxTicks) return Array.from({ length: total }, (_, i) => i);
  const result = [0];
  const step = (total - 1) / (maxTicks - 1);
  for (let i = 1; i < maxTicks - 1; i += 1) {
    result.push(Math.round(i * step));
  }
  result.push(total - 1);
  return [...new Set(result)].sort((a, b) => a - b);
}

function buildSmoothPath(ctx, points) {
  if (!points.length) return;
  ctx.beginPath();
  ctx.moveTo(points[0].x, points[0].y);
  if (points.length === 1) {
    ctx.lineTo(points[0].x, points[0].y);
    return;
  }

  for (let i = 0; i < points.length - 1; i += 1) {
    const p0 = points[i - 1] || points[i];
    const p1 = points[i];
    const p2 = points[i + 1];
    const p3 = points[i + 2] || p2;
    const cp1x = p1.x + (p2.x - p0.x) / 6;
    const cp1y = p1.y + (p2.y - p0.y) / 6;
    const cp2x = p2.x - (p3.x - p1.x) / 6;
    const cp2y = p2.y - (p3.y - p1.y) / 6;
    ctx.bezierCurveTo(cp1x, cp1y, cp2x, cp2y, p2.x, p2.y);
  }
}

function canvasRoundRectPath(ctx, x, y, w, h, r) {
  const radius = Math.max(0, Math.min(r, w / 2, h / 2));
  ctx.beginPath();
  ctx.moveTo(x + radius, y);
  ctx.arcTo(x + w, y, x + w, y + h, radius);
  ctx.arcTo(x + w, y + h, x, y + h, radius);
  ctx.arcTo(x, y + h, x, y, radius);
  ctx.arcTo(x, y, x + w, y, radius);
  ctx.closePath();
}

function renderB30Trend(points, mode) {
  const canvas = $("b30TrendCanvas");
  const hint = $("b30TrendHint");
  const modeLabel = $("trendModeLabel");
  if (!canvas || !hint || !modeLabel) return;

  modeLabel.textContent = formatTrendModeLabel(mode);
  const items = Array.isArray(points) ? points : [];
  state.trendPoints = items;

  const cssWidth = canvas.clientWidth || 760;
  const cssHeight = canvas.clientHeight || 260;
  const dpr = window.devicePixelRatio || 1;
  canvas.width = Math.max(1, Math.floor(cssWidth * dpr));
  canvas.height = Math.max(1, Math.floor(cssHeight * dpr));

  const ctx = canvas.getContext("2d");
  ctx.setTransform(1, 0, 0, 1, 0, 0);
  ctx.scale(dpr, dpr);
  ctx.clearRect(0, 0, cssWidth, cssHeight);

  if (!items.length) {
    const emptyBg = ctx.createLinearGradient(0, 0, 0, cssHeight);
    emptyBg.addColorStop(0, "rgba(239,247,255,0.95)");
    emptyBg.addColorStop(1, "rgba(231,243,255,0.95)");
    ctx.fillStyle = emptyBg;
    ctx.fillRect(0, 0, cssWidth, cssHeight);
    ctx.fillStyle = "#6f84a8";
    ctx.font = '700 15px "Baloo 2", "Microsoft YaHei", sans-serif';
    ctx.fillText("暂无趋势数据", 18, 38);
    ctx.font = '600 12px "Baloo 2", "Microsoft YaHei", sans-serif';
    ctx.fillStyle = "#8ca2c3";
    ctx.fillText("有成绩变动后，每3小时会合并更新一个趋势点", 18, 62);
    hint.textContent = "暂无趋势数据";
    return;
  }

  const values = items.map((it) => Number(it.avg_b30 || 0));
  let minY = Math.min(...values);
  let maxY = Math.max(...values);
  if (Math.abs(maxY-minY) < 0.01) {
    minY -= 0.2;
    maxY += 0.2;
  } else {
    minY -= 0.1;
    maxY += 0.1;
  }

  const padL = 60;
  const padR = 22;
  const padT = 24;
  const padB = 48;
  const plotW = Math.max(1, cssWidth-padL-padR);
  const plotH = Math.max(1, cssHeight-padT-padB);

  const bg = ctx.createLinearGradient(0, 0, 0, cssHeight);
  bg.addColorStop(0, "rgba(246,251,255,0.98)");
  bg.addColorStop(1, "rgba(236,246,255,0.98)");
  ctx.fillStyle = bg;
  ctx.fillRect(0, 0, cssWidth, cssHeight);

  ctx.strokeStyle = "rgba(172,197,230,0.5)";
  ctx.lineWidth = 1;
  ctx.setLineDash([5, 5]);
  for (let i = 0; i <= 5; i += 1) {
    const y = padT + (plotH*i)/4;
    ctx.beginPath();
    ctx.moveTo(padL, y);
    ctx.lineTo(padL + plotW, y);
    ctx.stroke();

    const value = maxY - ((maxY-minY)*i)/5;
    ctx.fillStyle = "#7d94b8";
    ctx.font = '700 11px "Baloo 2", "Microsoft YaHei", sans-serif';
    ctx.fillText(value.toFixed(2), 8, y + 4);
  }
  ctx.setLineDash([]);

  const toX = (idx) => {
    if (items.length === 1) return padL + plotW/2;
    return padL + (plotW*idx)/(items.length - 1);
  };
  const toY = (v) => padT + ((maxY-v)/(maxY-minY))*plotH;

  const palette = trendPalette(mode);
  const graphPoints = items.map((it, idx) => ({ x: toX(idx), y: toY(Number(it.avg_b30 || 0)), raw: it }));

  const area = ctx.createLinearGradient(0, padT, 0, padT + plotH);
  area.addColorStop(0, palette.areaTop);
  area.addColorStop(1, palette.areaBottom);
  ctx.fillStyle = area;
  buildSmoothPath(ctx, graphPoints);
  ctx.lineTo(graphPoints[graphPoints.length - 1].x, padT + plotH);
  ctx.lineTo(graphPoints[0].x, padT + plotH);
  ctx.closePath();
  ctx.fill();

  buildSmoothPath(ctx, graphPoints);
  ctx.strokeStyle = palette.line;
  ctx.shadowColor = palette.glow;
  ctx.shadowBlur = 10;
  ctx.lineWidth = 3;
  ctx.stroke();
  ctx.shadowBlur = 0;

  graphPoints.forEach((p) => {
    ctx.beginPath();
    ctx.fillStyle = palette.point;
    ctx.arc(p.x, p.y, 4, 0, Math.PI * 2);
    ctx.fill();
    ctx.beginPath();
    ctx.strokeStyle = "#ffffff";
    ctx.lineWidth = 2;
    ctx.arc(p.x, p.y, 4, 0, Math.PI * 2);
    ctx.stroke();
  });

  const last = items[items.length - 1];
  const lastP = graphPoints[graphPoints.length - 1];
  const tagText = Number(last.avg_b30 || 0).toFixed(4);
  const textW = ctx.measureText(tagText).width + 16;
  const tagX = Math.min(padL + plotW - textW, Math.max(padL, lastP.x - textW / 2));
  const tagY = Math.max(padT + 2, lastP.y - 34);
  ctx.fillStyle = "rgba(21,34,56,0.82)";
  canvasRoundRectPath(ctx, tagX, tagY, textW, 22, 8);
  ctx.fill();
  ctx.fillStyle = "#f3f8ff";
  ctx.font = '700 12px "Baloo 2", "Microsoft YaHei", sans-serif';
  ctx.fillText(tagText, tagX + 8, tagY + 15);

  const tickIdx = pickTickIndices(items.length, Math.min(6, items.length));
  ctx.fillStyle = "#6983aa";
  ctx.font = '700 11px "Baloo 2", "Microsoft YaHei", sans-serif';
  tickIdx.forEach((idx) => {
    const label = formatTrendTime(items[idx].created_at, true);
    const x = toX(idx);
    const textWidth = ctx.measureText(label).width;
    const drawX = clamp(x - textWidth / 2, padL, padL + plotW - textWidth);
    ctx.fillText(label, drawX, cssHeight - 14);
  });

  hint.textContent = `记录点 ${items.length} 个，最新 ${Number(last.avg_b30 || 0).toFixed(4)}（${formatTrendTime(last.created_at)}）`;
}

function difficultyBadgeClass(raw) {
  const key = normalizeDifficultyKey(raw);
  if (key === "unknown") return "master";
  return key;
}

function toFilterToken(item) {
  return item.diff ? `${item.diff}${item.level}` : `${item.level}`;
}

function buildDifficultyLevelsParam() {
  return state.filters.map(toFilterToken).join(",");
}

function renderFilterChips() {
  const wrap = $("selectedFilters");
  wrap.innerHTML = "";
  state.filters.forEach((f, idx) => {
    const chip = document.createElement("button");
    chip.className = "filter-chip";
    chip.type = "button";
    chip.textContent = f.diff ? `${f.diff} ${f.level}` : `任意 ${f.level}`;
    chip.title = "点击移除";
    chip.addEventListener("click", () => {
      state.filters.splice(idx, 1);
      renderFilterChips();
    });
    wrap.appendChild(chip);
  });
}

function initDiffPicker() {
  const levelSelect = $("pickerLevel");
  for (let i = 1; i <= 40; i += 1) {
    const option = document.createElement("option");
    option.value = String(i);
    option.textContent = String(i);
    levelSelect.appendChild(option);
  }
  renderFilterChips();
}

function openDiffPicker() {
  $("diffPickerModal").classList.remove("hidden");
}

function closeDiffPicker() {
  $("diffPickerModal").classList.add("hidden");
}

function addDiffFilter() {
  const diff = $("pickerDifficulty").value.trim();
  const level = Number($("pickerLevel").value || 0);
  if (!level) return;
  const token = diff ? `${diff}:${level}` : `*:${level}`;
  const exists = state.filters.some((x) => (x.diff ? `${x.diff}:${x.level}` : `*:${x.level}`) === token);
  if (exists) return;
  state.filters.push({ diff: diff || "", level });
  renderFilterChips();
}

function clearDiffFilter() {
  state.filters = [];
  renderFilterChips();
}

function avatarFallback() {
  return "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='128' height='128'%3E%3Crect width='100%25' height='100%25' fill='%23d8e3f7'/%3E%3C/svg%3E";
}

function getCharacterOptionByKey(key) {
  const current = String(key || "").trim();
  if (!current) return null;
  return state.characterOptions.find((it) => String(it.key || "").trim() === current) || null;
}

function getCharacterDisplayName(profile) {
  if (!profile) return "-";
  if (profile.character_name) return profile.character_name;
  const option = getCharacterOptionByKey(profile.character);
  if (option?.name) return option.name;
  const fallback = String(profile.character || "").trim();
  return fallback || "-";
}

function renderCharacterOptions() {
  const select = $("characterSelect");
  if (!select) return;
  const current = state.profile?.character || "";
  const options = state.characterOptions || [];
  select.innerHTML = `<option value="">不设置角色</option>`;
  options.forEach((item) => {
    const option = document.createElement("option");
    option.value = item.key || "";
    option.textContent = item.name || item.key || "-";
    if (option.value === current) option.selected = true;
    select.appendChild(option);
  });
}

function setCropPreviewFromProfile() {
  const cropImage = $("cropImage");
  if (!cropImage) return;
  const src = state.profile?.avatar_url ? toAbsoluteAsset(state.profile.avatar_url) : avatarFallback();
  initCropSource(src, false);
}

function initCropSource(src, fileSelected) {
  const cropImage = $("cropImage");
  const viewport = $("cropViewport");
  const zoom = $("cropZoom");
  if (!cropImage || !viewport) return;

  state.crop.fileSelected = !!fileSelected;
  state.crop.imageLoaded = false;
  state.crop.scale = 1;
  state.crop.x = 0;
  state.crop.y = 0;

  cropImage.onload = () => {
    state.crop.imageLoaded = true;
    state.crop.naturalW = cropImage.naturalWidth;
    state.crop.naturalH = cropImage.naturalHeight;

    const v = viewport.clientWidth || 260;
    state.crop.minScale = Math.max(v / state.crop.naturalW, v / state.crop.naturalH);

    const zoomPercent = Number(zoom?.value || 100);
    const safeZoom = Number.isFinite(zoomPercent) && zoomPercent >= 100 ? zoomPercent : 100;
    if (zoom) zoom.value = String(safeZoom);
    state.crop.scale = state.crop.minScale * (safeZoom / 100);

    state.crop.x = (v - state.crop.naturalW * state.crop.scale) / 2;
    state.crop.y = (v - state.crop.naturalH * state.crop.scale) / 2;
    clampCropPosition();
    renderCropTransform();
  };

  cropImage.src = src;
}

function renderProfile(profile) {
  state.profile = profile || null;
  renderAuthPanel(profile?.username || "");
  $("profileName").textContent = profile?.username || "-";
  $("profileCharacter").textContent = getCharacterDisplayName(profile);
  $("profileBio").textContent = profile?.profile || "-";
  $("profileB30").textContent = Number(profile?.b30_avg || 0).toFixed(2);
  const src = profile?.avatar_url ? toAbsoluteAsset(profile.avatar_url) : avatarFallback();
  $("profileAvatar").src = src;

  if ($("detailUserName")) $("detailUserName").textContent = profile?.username || "-";
  if ($("detailProfileInput")) $("detailProfileInput").value = profile?.profile || "";
  renderCharacterOptions();

  if (!state.crop.fileSelected) {
    setCropPreviewFromProfile();
  }
}

function renderB30(list) {
  const tbody = $("b30Table").querySelector("tbody");
  tbody.innerHTML = "";
  (list || []).forEach((it) => {
    const coverURL = toAbsoluteAsset(`/static/assets/${it.assetbundleName || ""}.png`);
    const tr = document.createElement("tr");
    const diffKey = normalizeDifficultyKey(it.music_difficulty);
    const diffText = prettyDifficulty(it.music_difficulty);
    const diffClass = diffKey === "unknown" ? "master" : diffKey;
    const statusKey = normalizeAchievement(it.music_achievement);
    const statusClass = achievementClass(statusKey);
    const statusText = achievementLabel(statusKey);
    const playLevel = Number(it.play_level || 0);
    const constValue = Number(it.const_value || 0);
    const useConst = state.b30CalcMode === "const" && constValue > 0;
    const levelDisplay = useConst ? constValue.toFixed(1) : (playLevel > 0 ? String(playLevel) : "-");
    const badgeLevelDisplay = useConst ? `定数 ${constValue.toFixed(1)}` : `Lv ${playLevel > 0 ? playLevel : "-"}`;
    tr.innerHTML = `
      <td>${it.rank}</td>
      <td><img class="b30-cover" src="${coverURL}" alt="${it.title || "-"}" /></td>
      <td>${it.title || "-"}</td>
      <td class="b30-diff-cell">
        <span class="b30-diff-pill ${diffClass}">
          ${diffText}
          <span class="b30-diff-lv">${badgeLevelDisplay}</span>
        </span>
      </td>
      <td>${levelDisplay}</td>
      <td><span class="b30-status-pill ${statusClass}">${statusText}</span></td>
      <td>${Number(it.score_value || 0).toFixed(2)}</td>
    `;
    tbody.appendChild(tr);
  });
}

function renderSongs() {
  const box = $("songList");
  box.innerHTML = "";

  state.musics.forEach((song) => {
    const row = document.createElement("div");
    row.className = "song-row";
    const coverURL = toAbsoluteAsset(song.cover_url || `/static/assets/${song.assetbundleName || ""}.png`);
    const diffs = [...(song.difficulties || [])].sort(
      (a, b) => {
        const diffOrderA = difficultyOrder[readDifficultyType(a)] || 99;
        const diffOrderB = difficultyOrder[readDifficultyType(b)] || 99;
        if (diffOrderA !== diffOrderB) return diffOrderA - diffOrderB;

        const levelA = readDifficultyLevel(a);
        const levelB = readDifficultyLevel(b);
        if (levelA !== levelB) return levelA - levelB;

        return readDifficultyID(a) - readDifficultyID(b);
      },
    );

    row.innerHTML = `
      <img class="song-cover song-detail-trigger" src="${coverURL}" alt="${song.title || ""}" />
      <div class="song-meta">
        <div class="song-title song-detail-trigger">${song.title || "-"}</div>
        <div class="song-sub">ID: ${song.id} | ${song.composer || "-"}</div>
      </div>
      <div class="diff-buttons"></div>
    `;

    const openDetail = () => {
      openMusicDetail(song).catch((err) => alert(err.message || "加载歌曲详情失败"));
    };
    row.querySelector(".song-cover")?.addEventListener("click", openDetail);
    row.querySelector(".song-title")?.addEventListener("click", openDetail);

    const buttonsWrap = row.querySelector(".diff-buttons");
    diffs.forEach((d) => {
      const diffKey = readDifficultyType(d);
      const status = state.statuses[readDifficultyID(d)] || "not_played";
      const marker = markerClassByStatus(status);
      const btn = document.createElement("button");
      btn.className = `diff-btn ${diffKey}`;
      btn.innerHTML = `${diffKey} ${readDifficultyLevel(d)}<span class="status-line ${marker}"></span>`;
      btn.addEventListener("click", () => {
        if (!state.token) return alert("请先登录");
        openUploadModal(song, d);
      });
      buttonsWrap.appendChild(btn);
    });
    box.appendChild(row);
  });

  const totalPage = Math.max(1, Math.ceil(state.total / state.size));
  $("pageInfo").textContent = `第 ${state.page} / ${totalPage} 页`;
}

function closeMusicDetail() {
  state.currentMusicDetail = null;
  const aliasInput = $("musicAliasInput");
  if (aliasInput) aliasInput.value = "";
  $("musicDetailModal").classList.add("hidden");
}

function safeRate(rate) {
  const n = Number(rate);
  if (!Number.isFinite(n)) return 0;
  return clamp(n, 0, 100);
}

function formatRate(rate) {
  return `${safeRate(rate).toFixed(2)}%`;
}

function sanitizeStatsDifficulty(raw) {
  const key = normalizeDifficultyKey(raw);
  return key === "unknown" ? "master" : key;
}

function sanitizeStatsMode(raw) {
  const mode = String(raw || "").trim().toLowerCase();
  if (mode === "by_level") return "by_level";
  if (mode === "by_global_level") return "by_global_level";
  return "by_difficulty";
}

function formatStatsModeLabel(mode) {
  if (mode === "by_global_level") return "按等级分类（不分难度）";
  if (mode === "by_level") return "按难度分类（当前难度分等级）";
  return "按等级分类（当前难度总览）";
}

function prettyStatsDifficulty(raw) {
  return sanitizeStatsDifficulty(raw).toUpperCase();
}

function sanitizeStatsLevel(raw, fallback = 1) {
  const n = Number(raw);
  if (!Number.isFinite(n)) return clamp(fallback, 1, 40);
  return clamp(Math.round(n), 1, 40);
}

function normalizeStatsLevelRange(minRaw, maxRaw) {
  let min = sanitizeStatsLevel(minRaw, 1);
  let max = sanitizeStatsLevel(maxRaw, 40);
  if (min > max) [min, max] = [max, min];
  return { min, max };
}

function initStatsLevelOptions() {
  const minSel = $("statsMinLevelSelect");
  const maxSel = $("statsMaxLevelSelect");
  if (!minSel || !maxSel) return;
  if (!minSel.options.length) {
    for (let lv = 1; lv <= 40; lv += 1) {
      const op = document.createElement("option");
      op.value = String(lv);
      op.textContent = String(lv);
      minSel.appendChild(op);
    }
  }
  if (!maxSel.options.length) {
    for (let lv = 1; lv <= 40; lv += 1) {
      const op = document.createElement("option");
      op.value = String(lv);
      op.textContent = String(lv);
      maxSel.appendChild(op);
    }
  }
}

function syncStatsControlVisibility() {
  const mode = sanitizeStatsMode(state.statsMode);
  const rangeWrap = $("statsRangeWrap");
  const diffWrap = $("statsDifficultyWrap");
  if (rangeWrap) rangeWrap.classList.toggle("hidden-ui", mode !== "by_global_level");
  if (diffWrap) diffWrap.classList.toggle("hidden-ui", mode === "by_global_level");
}

function buildStatsRowHTML(bucket, mode, difficulty) {
  const total = Number(bucket?.total_charts || 0);
  const playLevel = Number(bucket?.play_level || 0);
  let label = `${prettyStatsDifficulty(difficulty)} 全等级`;
  if (mode === "by_level") {
    label = `${prettyStatsDifficulty(difficulty)} ${playLevel > 0 ? `Lv ${playLevel}` : "-"}`;
  } else if (mode === "by_global_level") {
    label = playLevel > 0 ? `Lv ${playLevel}` : (String(bucket?.label || "Lv -").trim() || "Lv -");
  }

  const clearRate = safeRate(bucket?.clear_rate);
  const fcRate = safeRate(bucket?.fc_rate);
  const apRate = safeRate(bucket?.ap_rate);
  const npRate = safeRate(bucket?.not_played_rate);

  const clearCount = Number(bucket?.clear_count || 0);
  const fcCount = Number(bucket?.fc_count || 0);
  const apCount = Number(bucket?.ap_count || 0);
  const npCount = Number(bucket?.not_played_count || 0);

  return `
    <div class="stats-row">
      <div class="stats-row-head">
        <span class="stats-row-label">${label}</span>
        <span class="stats-row-total">总谱面 ${total}</span>
      </div>
      <div class="stats-bar">
        <span class="stats-bar-segment clear" style="width:${clearRate}%"></span>
        <span class="stats-bar-segment fc" style="width:${fcRate}%"></span>
        <span class="stats-bar-segment ap" style="width:${apRate}%"></span>
        <span class="stats-bar-segment np" style="width:${npRate}%"></span>
      </div>
      <div class="stats-row-meta">
        <span>CLEAR ${formatRate(clearRate)} (${clearCount})</span>
        <span>FC ${formatRate(fcRate)} (${fcCount})</span>
        <span>AP ${formatRate(apRate)} (${apCount})</span>
        <span>NOT PLAYED ${formatRate(npRate)} (${npCount})</span>
      </div>
    </div>
  `;
}

function renderStatistics(data) {
  const board = $("statsBoard");
  const title = $("statsTitle");
  const hint = $("statsHint");
  if (!board || !title || !hint) return;
  syncStatsControlVisibility();

  if (!state.token) {
    title.textContent = "成绩统计";
    hint.textContent = "请先登录";
    board.innerHTML = `<div class="song-sub">请先登录查看统计</div>`;
    return;
  }

  const difficulty = sanitizeStatsDifficulty(data?.difficulty || state.statsDifficulty);
  const mode = sanitizeStatsMode(data?.mode || state.statsMode);
  const minLevel = sanitizeStatsLevel(data?.min_level ?? state.statsMinLevel, 1);
  const maxLevel = sanitizeStatsLevel(data?.max_level ?? state.statsMaxLevel, 40);
  const totalCharts = Number(data?.total_charts || 0);
  const buckets = Array.isArray(data?.buckets) ? data.buckets : [];

  if (mode === "by_global_level") {
    title.textContent = `全难度等级统计`;
    hint.textContent = `${formatStatsModeLabel(mode)} · 等级 ${minLevel}-${maxLevel} · 总谱面 ${totalCharts}`;
  } else {
    title.textContent = `${prettyStatsDifficulty(difficulty)} 统计`;
    hint.textContent = `${formatStatsModeLabel(mode)} · 总谱面 ${totalCharts}`;
  }

  if (!buckets.length) {
    board.innerHTML = `<div class="song-sub">暂无统计数据</div>`;
    return;
  }

  board.innerHTML = buckets.map((it) => buildStatsRowHTML(it, mode, difficulty)).join("");
}

function renderCombinedRateRing(fcRate, apRate) {
  const fc = safeRate(fcRate);
  const ap = safeRate(apRate);
  return `
    <span class="rate-ring-combined" style="--fc:${fc}; --ap:${ap};">
      <span class="rate-ring-combined-core">
        <span class="rate-ring-mini fc">${Math.round(fc)}%</span>
        <span class="rate-ring-mini ap">${Math.round(ap)}%</span>
      </span>
    </span>
  `;
}

function sortDifficultyStats(list) {
  const items = [...(list || [])];
  items.sort((a, b) => {
    const diffA = difficultyOrder[normalizeDifficultyKey(a.music_difficulty)] || 99;
    const diffB = difficultyOrder[normalizeDifficultyKey(b.music_difficulty)] || 99;
    if (diffA !== diffB) return diffA - diffB;
    const lvA = Number(a.play_level || 0);
    const lvB = Number(b.play_level || 0);
    if (lvA !== lvB) return lvA - lvB;
    return Number(a.music_difficulty_id || 0) - Number(b.music_difficulty_id || 0);
  });
  return items;
}

function renderMusicDetail(data, fallbackSong) {
  const music = data?.music || fallbackSong || {};
  state.currentMusicDetail = {
    id: Number(music.id || fallbackSong?.id || 0),
    title: music.title || fallbackSong?.title || "-",
  };
  const stats = sortDifficultyStats(data?.difficulty_stats || []);
  const totalCount = Number(data?.total_count || 0);
  const fcTotalCount = Number(data?.fc_total_count || 0);
  const apTotalCount = Number(data?.ap_total_count || 0);

  $("musicDetailTitle").textContent = `${music.title || fallbackSong?.title || "-"} · 歌曲详情`;
  const coverURL = toAbsoluteAsset(music.cover_url || fallbackSong?.cover_url || `/static/assets/${music.assetbundleName || fallbackSong?.assetbundleName || ""}.png`);
  $("musicDetailMeta").innerHTML = `
    <img class="music-detail-cover" src="${coverURL}" alt="${music.title || "-"}" />
    <div class="music-detail-meta-text">
      <div><span class="label">作曲：</span>${music.composer || "-"}</div>
      <div><span class="label">别名：</span>${music.alias || "-"}</div>
      <div><span class="label">总游玩：</span>${totalCount}</div>
      <div><span class="label">总FC / 总AP：</span>${fcTotalCount} / ${apTotalCount}</div>
    </div>
  `;
  const aliasInput = $("musicAliasInput");
  if (aliasInput) {
    aliasInput.value = "";
    aliasInput.placeholder = `新增别名（当前：${music.alias || "-"})`;
    aliasInput.disabled = !state.token;
  }
  const addAliasBtn = $("addMusicAliasBtn");
  if (addAliasBtn) addAliasBtn.disabled = !state.token;

  const tbody = $("musicDetailTable").querySelector("tbody");
  tbody.innerHTML = "";
  if (!stats.length) {
    const tr = document.createElement("tr");
    tr.innerHTML = `<td colspan="6">暂无该歌曲统计数据</td>`;
    tbody.appendChild(tr);
    return;
  }

  stats.forEach((it) => {
    const diffClass = difficultyBadgeClass(it.music_difficulty);
    const diffText = prettyDifficulty(it.music_difficulty);
    const constValue = Number(it.const_value || 0);
    const constText = constValue > 0 ? ` / 定数 ${constValue.toFixed(1)}` : "";
    const tr = document.createElement("tr");
    tr.innerHTML = `
      <td><span class="b30-diff-pill ${diffClass}">${diffText}</span></td>
      <td>Lv ${Number(it.play_level || 0)}${constText}</td>
      <td>${Number(it.played_count || 0)}</td>
      <td>${Number(it.fc_count || 0)}</td>
      <td>${Number(it.ap_count || 0)}</td>
      <td>
        <div class="rate-cell combined">
          ${renderCombinedRateRing(it.fc_rate, it.ap_rate)}
          <span class="rate-text">
            FC ${formatRate(it.fc_rate)} / AP ${formatRate(it.ap_rate)}
          </span>
        </div>
      </td>
    `;
    tbody.appendChild(tr);
  });
}

async function openMusicDetail(song) {
  state.currentMusicDetail = { id: Number(song?.id || 0), title: song?.title || "-" };
  $("musicDetailModal").classList.remove("hidden");
  $("musicDetailTitle").textContent = `${song.title || "-"} · 歌曲详情`;
  $("musicDetailMeta").innerHTML = `<div class="music-detail-loading">加载中...</div>`;
  if ($("musicAliasInput")) $("musicAliasInput").value = "";
  const tbody = $("musicDetailTable").querySelector("tbody");
  tbody.innerHTML = `<tr><td colspan="6">加载中...</td></tr>`;
  const data = await api(`/musics/${song.id}`);
  renderMusicDetail(data, song);
}

async function addMusicAlias() {
  if (!state.token) return alert("请先登录");
  const songID = Number(state.currentMusicDetail?.id || 0);
  if (!songID) return alert("歌曲信息不存在，请重新打开详情");

  const input = $("musicAliasInput");
  const alias = String(input?.value || "").trim();
  if (!alias) return alert("请输入要新增的别名");

  const title = state.currentMusicDetail?.title || "-";
  if (!window.confirm(`确认要为「${title}」新增别名「${alias}」吗？`)) return;
  if (!window.confirm("请再次确认，提交后会立即写入数据库。")) return;

  await api(`/musics/${songID}/alias`, {
    method: "POST",
    auth: true,
    body: { alias },
  });

  notify("别名新增成功");
  if (input) input.value = "";
  const detail = await api(`/musics/${songID}`);
  renderMusicDetail(detail, {
    id: songID,
    title,
    assetbundleName: detail?.music?.assetbundleName || "",
    cover_url: detail?.music?.cover_url || "",
  });
  await loadSongs();
}

function renderRandomRecommendation(data) {
  const body = $("randomRecommendBody");
  if (!body) return;

  const diffRaw = data?.music_difficulty || "";
  const diffClass = difficultyBadgeClass(diffRaw);
  const diffText = prettyDifficulty(diffRaw);
  const playLevel = Number(data?.play_level || 0);
  const constValue = Number(data?.const || 0);
  const targetKey = normalizeAchievement(data?.type);
  const targetText = achievementShortLabel(targetKey);
  const targetClassName = achievementClass(targetKey);
  const achievementKey = normalizeAchievement(data?.user_achievement);
  const achievementText = achievementShortLabel(achievementKey);
  const achievementClassName = achievementClass(achievementKey);
  const actualMode = String(data?.calc_mode || "").trim().toLowerCase();
  const requestedMode = String(state.b30CalcMode || "official").trim().toLowerCase();
  const modeText = calcModeLabel(actualMode || requestedMode);
  const modeHint = requestedMode === "const" && actualMode === "official" ? "（定数候选不足，已降级）" : "";
  const coverURL = toAbsoluteAsset(`/static/assets/${data?.assetbundleName || ""}.png`);
  const constText = constValue > 0 ? ` | 定数 ${constValue.toFixed(1)}` : "";

  body.innerHTML = `
    <img class="random-cover" src="${coverURL}" alt="${data?.title || "-"}" />
    <div class="random-meta">
      <div class="random-title">${data?.title || "-"}</div>
      <div class="random-sub">歌曲ID: ${Number(data?.song_id || 0)} | 难度ID: ${Number(data?.music_difficulty_id || 0)}</div>
      <div class="random-row">
        <span class="b30-diff-pill ${diffClass}">${diffText}<span class="b30-diff-lv">Lv ${playLevel || "-"}</span></span>
        <span class="b30-status-pill ${achievementClassName} random-status-pill">${achievementText}</span>
      </div>
      <div class="random-row">
        <span class="random-kv"><span class="label">推荐目标：</span><span class="b30-status-pill ${targetClassName} random-status-pill">${targetText}</span></span>
        <span class="random-kv"><span class="label">计算模式：</span><span class="random-value">${modeText}${modeHint}</span></span>
      </div>
      <div class="random-row">
        <span class="random-kv"><span class="label">当前成绩：</span><span class="b30-status-pill ${achievementClassName} random-status-pill">${achievementText}</span></span>
        <span class="random-kv"><span class="label">等级信息：</span><span class="random-value">Lv ${playLevel || "-"}${constText}</span></span>
      </div>
    </div>
  `;
}

function closeRandomRecommendation() {
  $("randomRecommendModal").classList.add("hidden");
}

async function loadRandomRecommendation() {
  if (!state.token) return alert("请先登录");
  $("randomRecommendModal").classList.remove("hidden");
  $("randomRecommendBody").innerHTML = `<div class="song-sub">随机推荐加载中...</div>`;
  const data = await api(`/random/music?calc_mode=${encodeURIComponent(state.b30CalcMode)}`, { auth: true });
  renderRandomRecommendation(data);
}

function openUploadModal(song, diff) {
  state.currentSelect = { song, diff };
  const diffID = readDifficultyID(diff);
  const diffType = readDifficultyType(diff);
  const diffLevel = readDifficultyLevel(diff);
  const current = state.statuses[diffID] || "not_played";
  $("modalTarget").textContent = `${song.title || "-"} | ${diffType.toUpperCase()} ${diffLevel || "-"}`;
  document.querySelectorAll("input[name=scoreStatus]").forEach((radio) => {
    radio.checked = radio.value === current;
  });
  $("uploadModal").classList.remove("hidden");
}

function closeUploadModal() {
  state.currentSelect = null;
  $("uploadModal").classList.add("hidden");
}

function openProfileDetail() {
  if (!state.token) {
    alert("请先登录");
    return;
  }
  if (state.profile) {
    $("detailUserName").textContent = state.profile.username || "-";
    $("detailProfileInput").value = state.profile.profile || "";
  }
  renderCharacterOptions();
  $("profileDetailModal").classList.remove("hidden");
}

function closeProfileDetail() {
  $("profileDetailModal").classList.add("hidden");
}

function renderCropTransform() {
  const cropImage = $("cropImage");
  if (!cropImage) return;
  cropImage.style.transform = `translate(${state.crop.x}px, ${state.crop.y}px) scale(${state.crop.scale})`;
}

function clampCropPosition() {
  const viewport = $("cropViewport");
  if (!viewport || !state.crop.imageLoaded) return;
  const v = viewport.clientWidth || 260;
  const w = state.crop.naturalW * state.crop.scale;
  const h = state.crop.naturalH * state.crop.scale;
  const minX = Math.min(0, v - w);
  const minY = Math.min(0, v - h);
  state.crop.x = clamp(state.crop.x, minX, 0);
  state.crop.y = clamp(state.crop.y, minY, 0);
}

function loadCropFile(file) {
  if (!file) return;
  if (!/^image\/(png|jpeg|webp)$/i.test(file.type)) {
    alert("头像仅支持 png/jpg/webp");
    return;
  }

  if (state.crop.imageURL) URL.revokeObjectURL(state.crop.imageURL);
  state.crop.imageURL = URL.createObjectURL(file);
  const zoom = $("cropZoom");
  if (zoom) zoom.value = "100";
  initCropSource(state.crop.imageURL, true);
}

function initCropInteractions() {
  const viewport = $("cropViewport");
  const zoom = $("cropZoom");

  viewport.addEventListener("mousedown", (e) => {
    if (!state.crop.imageLoaded) return;
    state.crop.dragging = true;
    state.crop.dragStartX = e.clientX;
    state.crop.dragStartY = e.clientY;
    state.crop.dragOriginX = state.crop.x;
    state.crop.dragOriginY = state.crop.y;
  });

  window.addEventListener("mousemove", (e) => {
    if (!state.crop.dragging) return;
    state.crop.x = state.crop.dragOriginX + (e.clientX - state.crop.dragStartX);
    state.crop.y = state.crop.dragOriginY + (e.clientY - state.crop.dragStartY);
    clampCropPosition();
    renderCropTransform();
  });

  window.addEventListener("mouseup", () => {
    state.crop.dragging = false;
  });

  zoom.addEventListener("input", () => {
    if (!state.crop.imageLoaded) return;
    const v = viewport.clientWidth || 260;
    const oldScale = state.crop.scale;
    const centerX = (v / 2 - state.crop.x) / oldScale;
    const centerY = (v / 2 - state.crop.y) / oldScale;
    state.crop.scale = state.crop.minScale * (Number(zoom.value) / 100);
    state.crop.x = v / 2 - centerX * state.crop.scale;
    state.crop.y = v / 2 - centerY * state.crop.scale;
    clampCropPosition();
    renderCropTransform();
  });
}

async function uploadCroppedAvatar() {
  if (!state.token) return alert("请先登录");
  if (!state.crop.fileSelected || !state.crop.imageLoaded) return alert("请先选择要裁切的头像");

  const cropImage = $("cropImage");
  const viewport = $("cropViewport");
  const v = viewport.clientWidth || 260;

  let sx = (0 - state.crop.x) / state.crop.scale;
  let sy = (0 - state.crop.y) / state.crop.scale;
  let sSize = v / state.crop.scale;

  sSize = Math.min(sSize, state.crop.naturalW, state.crop.naturalH);
  sx = clamp(sx, 0, Math.max(0, state.crop.naturalW - sSize));
  sy = clamp(sy, 0, Math.max(0, state.crop.naturalH - sSize));

  const canvas = document.createElement("canvas");
  canvas.width = 512;
  canvas.height = 512;
  const ctx = canvas.getContext("2d");
  ctx.drawImage(cropImage, sx, sy, sSize, sSize, 0, 0, 512, 512);

  const blob = await new Promise((resolve) => canvas.toBlob(resolve, "image/png", 0.95));
  if (!blob) throw new Error("头像裁切失败");

  const form = new FormData();
  form.append("avatar", blob, "avatar.png");
  const data = await apiMultipart("/me/avatar", form, { auth: true });

  if (!state.profile) {
    state.profile = {
      username: $("profileName").textContent,
      profile: $("profileBio").textContent,
      b30_avg: Number($("profileB30").textContent || 0),
    };
  }
  state.profile.avatar_url = data.avatar_url;
  renderProfile(state.profile);

  const fileInput = $("detailAvatarFileInput");
  if (fileInput) fileInput.value = "";
  if (state.crop.imageURL) {
    URL.revokeObjectURL(state.crop.imageURL);
    state.crop.imageURL = "";
  }
  state.crop.fileSelected = false;
  notify("头像上传成功");
}

async function loadMyData() {
  if (!state.token) {
    state.profile = null;
    state.statuses = {};
    state.achievementMap = {};
    state.trendPoints = [];
    renderAuthPanel("");
    renderProfile(null);
    renderB30([]);
    renderB30Trend([], state.b30CalcMode);
    renderCharacterOptions();
    return;
  }

  const [profile, b30, statuses, am, trend] = await Promise.all([
    api("/me", { auth: true }),
    api(`/records/b30?calc_mode=${encodeURIComponent(state.b30CalcMode)}`, { auth: true }),
    api("/records/statuses", { auth: true }),
    api("/records/achievement-map", { auth: true }),
    api(`/records/b30/trend?calc_mode=${encodeURIComponent(state.b30CalcMode)}`, { auth: true }),
  ]);

  renderProfile(profile);
  renderB30(b30.list || []);

  state.statuses = {};
  (statuses.list || []).forEach((it) => {
    state.statuses[it.music_difficulty_id] = normalizeAchievement(it.music_achievement);
  });
  state.achievementMap = am.map || {};
  renderB30Trend(trend.list || [], state.b30CalcMode);
}

async function loadCharacters() {
  const data = await api("/characters");
  state.characterOptions = data.list || [];
  renderCharacterOptions();
}

async function loadSongs() {
  const params = new URLSearchParams({ page: String(state.page), size: String(state.size), sort: state.sort });
  if (state.keyword) params.set("keyword", state.keyword);
  const diffParam = buildDifficultyLevelsParam();
  if (diffParam) params.set("difficulty_levels", diffParam);

  const data = await api(`/musics?${params.toString()}`);
  state.musics = data.list || [];
  state.total = data.total || 0;
  renderSongs();
}

async function loadStatistics() {
  if (!state.token) {
    state.statistics = null;
    renderStatistics(null);
    return;
  }

  state.statsDifficulty = sanitizeStatsDifficulty(state.statsDifficulty);
  state.statsMode = sanitizeStatsMode(state.statsMode);
  const range = normalizeStatsLevelRange(state.statsMinLevel, state.statsMaxLevel);
  state.statsMinLevel = range.min;
  state.statsMaxLevel = range.max;
  const params = new URLSearchParams({ mode: state.statsMode });
  if (state.statsMode === "by_global_level") {
    params.set("min_level", String(state.statsMinLevel));
    params.set("max_level", String(state.statsMaxLevel));
  } else {
    params.set("difficulty", state.statsDifficulty);
  }
  const data = await api(`/records/statistics?${params.toString()}`, { auth: true });
  state.statistics = data || null;

  if (state.statistics) {
    state.statsMode = sanitizeStatsMode(state.statistics.mode || state.statsMode);
    if (state.statsMode !== "by_global_level") {
      state.statsDifficulty = sanitizeStatsDifficulty(state.statistics.difficulty || state.statsDifficulty);
    }
    state.statsMinLevel = sanitizeStatsLevel(state.statistics.min_level ?? state.statsMinLevel, state.statsMinLevel);
    state.statsMaxLevel = sanitizeStatsLevel(state.statistics.max_level ?? state.statsMaxLevel, state.statsMaxLevel);
  }
  localStorage.setItem("stats_mode", state.statsMode);
  localStorage.setItem("stats_difficulty", state.statsDifficulty);
  localStorage.setItem("stats_min_level", String(state.statsMinLevel));
  localStorage.setItem("stats_max_level", String(state.statsMaxLevel));

  if ($("statsModeSelect")) $("statsModeSelect").value = state.statsMode;
  if ($("statsDifficultySelect")) $("statsDifficultySelect").value = state.statsDifficulty;
  if ($("statsMinLevelSelect")) $("statsMinLevelSelect").value = String(state.statsMinLevel);
  if ($("statsMaxLevelSelect")) $("statsMaxLevelSelect").value = String(state.statsMaxLevel);
  syncStatsControlVisibility();
  renderStatistics(state.statistics);
}

async function register() {
  const username = $("usernameInput").value.trim();
  const email = $("emailInput").value.trim();
  const password = $("passwordInput").value.trim();
  if (!username || !email || !password) {
    return alert("注册需要用户名、邮箱、密码都填写");
  }
  await api("/register", { method: "POST", body: { username, email, password } });
  notify("注册成功，请登录");
}

async function login() {
  const username = $("usernameInput").value.trim();
  const password = $("passwordInput").value.trim();
  if (!username || !password) return alert("请输入用户名和密码");

  const data = await api("/login", { method: "POST", body: { username, password } });
  setAuth(data.token || data.access_token || "", data.refresh_token || "");
  renderAuthPanel(data.username || username);
  notify("登录成功");
  await refreshAll();
}

async function logout() {
  if (state.token) {
    try {
      await api("/logout", { method: "POST", auth: true });
    } catch {
      // ignore server logout errors; still clear local session
    }
  }

  clearAuth();
  state.statuses = {};
  state.achievementMap = {};
  state.profile = null;
  state.statistics = null;
  renderProfile(null);
  renderB30([]);
  renderStatistics(null);
  renderAuthPanel("");
  closeProfileDetail();
  setCropPreviewFromProfile();
  notify("已退出登录");
  await Promise.all([loadSongs(), loadStatistics()]);
}

async function saveProfile() {
  if (!state.token) return alert("请先登录");
  const profileText = $("detailProfileInput").value.trim();
  await api("/me/profile", { method: "POST", auth: true, body: { profile: profileText } });
  if (state.profile) state.profile.profile = profileText;
  $("profileBio").textContent = profileText || "-";
  notify("简介保存成功");
}

async function saveCharacter() {
  if (!state.token) return alert("请先登录");
  const key = $("characterSelect").value.trim();
  await api("/me/character", { method: "POST", auth: true, body: { character: key } });

  if (!state.profile) state.profile = {};
  state.profile.character = key;
  const option = getCharacterOptionByKey(key);
  state.profile.character_name = option?.name || "";
  state.profile.character_image_url = option?.image_url || "";
  $("profileCharacter").textContent = getCharacterDisplayName(state.profile);
  notify("角色保存成功");
}

async function exportB30Image() {
  if (!state.token) return alert("请先登录");
  const blob = await fetchBinary(`/records/b30/image?calc_mode=${encodeURIComponent(state.b30CalcMode)}`, { auth: true });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = "b30.png";
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
  notify("B30 图片已导出");
}

async function saveRecord() {
  if (!state.currentSelect) return;
  const selected = document.querySelector('input[name="scoreStatus"]:checked');
  if (!selected) return alert("请选择状态");

  const status = selected.value;
  const diffID = readDifficultyID(state.currentSelect.diff);
  const diffType = readDifficultyType(state.currentSelect.diff);
  if (!diffID || diffType === "unknown") return alert("当前难度数据不完整，请刷新后重试");

  if (status === "not_played") {
    await api("/records", { method: "DELETE", auth: true, body: { music_difficulty_id: diffID } });
    notify("成绩已删除");
  } else {
    const diffMap = state.achievementMap[diffType] || {};
    const id = diffMap[status];
    if (!id) return alert(`后端没有找到 ${diffType}/${status} 对应成就ID`);
    await api("/records", { method: "POST", auth: true, body: { music_difficulty_id: diffID, music_achievement_id: id } });
    notify("保存成功");
  }

  closeUploadModal();
  await refreshAll();
}

async function refreshAll() {
  try {
    await Promise.all([loadCharacters(), loadMyData(), loadSongs(), loadStatistics()]);
  } catch (err) {
    console.error(err);
    alert(err.message || "请求失败");
  }
}

function saveApiBase() {
  const next = $("apiBaseInput").value.trim();
  if (!/^https?:\/\//i.test(next)) return alert("API 地址必须以 http:// 或 https:// 开头");
  state.apiBase = next.replace(/\/+$/, "");
  localStorage.setItem("api_base", state.apiBase);
  refreshAll();
}

function bindEvents() {
  $("apiBaseInput").value = state.apiBase;
  initStatsLevelOptions();
  state.statsMode = sanitizeStatsMode(state.statsMode);
  state.statsDifficulty = sanitizeStatsDifficulty(state.statsDifficulty);
  const range = normalizeStatsLevelRange(state.statsMinLevel, state.statsMaxLevel);
  state.statsMinLevel = range.min;
  state.statsMaxLevel = range.max;
  if ($("b30CalcModeSelect")) $("b30CalcModeSelect").value = state.b30CalcMode;
  if ($("statsDifficultySelect")) $("statsDifficultySelect").value = sanitizeStatsDifficulty(state.statsDifficulty);
  if ($("statsModeSelect")) $("statsModeSelect").value = sanitizeStatsMode(state.statsMode);
  if ($("statsMinLevelSelect")) $("statsMinLevelSelect").value = String(state.statsMinLevel);
  if ($("statsMaxLevelSelect")) $("statsMaxLevelSelect").value = String(state.statsMaxLevel);
  syncStatsControlVisibility();

  on("saveApiBaseBtn", "click", saveApiBase);
  on("registerBtn", "click", () => register().catch((e) => alert(e.message)));
  on("loginBtn", "click", () => login().catch((e) => alert(e.message)));
  on("logoutBtn", "click", () => logout().catch((e) => alert(e.message)));

  on("profileCard", "click", openProfileDetail);
  on("closeProfileDetailBtn", "click", closeProfileDetail);
  on("profileDetailModal", "click", (e) => {
    if (e.target.id === "profileDetailModal") closeProfileDetail();
  });
  on("detailAvatarFileInput", "change", (e) => loadCropFile(e.target.files?.[0]));
  on("applyCropUploadBtn", "click", () => uploadCroppedAvatar().catch((e) => alert(e.message)));
  on("saveProfileBtn", "click", () => saveProfile().catch((e) => alert(e.message)));
  on("saveCharacterBtn", "click", () => saveCharacter().catch((e) => alert(e.message)));
  on("exportB30Btn", "click", () => exportB30Image().catch((e) => alert(e.message)));
  on("randomRecommendBtn", "click", () => loadRandomRecommendation().catch((e) => alert(e.message || "随机推荐失败")));
  on("closeRandomRecommendBtn", "click", closeRandomRecommendation);
  on("randomRecommendModal", "click", (e) => {
    if (e.target.id === "randomRecommendModal") closeRandomRecommendation();
  });

  on("openDiffPickerBtn", "click", openDiffPicker);
  on("closeDiffPickerBtn", "click", closeDiffPicker);
  on("addDiffFilterBtn", "click", addDiffFilter);
  on("clearDiffFilterBtn", "click", clearDiffFilter);
  on("diffPickerModal", "click", (e) => {
    if (e.target.id === "diffPickerModal") closeDiffPicker();
  });

  on("searchBtn", "click", () => {
    state.keyword = $("keywordInput").value.trim();
    state.sort = $("sortSelect").value;
    state.page = 1;
    loadSongs().catch((e) => alert(e.message));
    closeDiffPicker();
  });

  on("prevPageBtn", "click", () => {
    if (state.page <= 1) return;
    state.page -= 1;
    loadSongs().catch((e) => alert(e.message));
  });

  on("nextPageBtn", "click", () => {
    const totalPage = Math.max(1, Math.ceil(state.total / state.size));
    if (state.page >= totalPage) return;
    state.page += 1;
    loadSongs().catch((e) => alert(e.message));
  });

  on("cancelRecordBtn", "click", closeUploadModal);
  on("saveRecordBtn", "click", () => saveRecord().catch((e) => alert(e.message)));
  on("b30CalcModeSelect", "change", (e) => {
    const next = e.target.value === "const" ? "const" : "official";
    state.b30CalcMode = next;
    localStorage.setItem("b30_calc_mode", next);
    loadMyData().catch((err) => alert(err.message || "切换计算方式失败"));
  });
  on("statsDifficultySelect", "change", (e) => {
    const next = sanitizeStatsDifficulty(e.target.value);
    state.statsDifficulty = next;
    localStorage.setItem("stats_difficulty", next);
    loadStatistics().catch((err) => alert(err.message || "加载统计失败"));
  });
  on("statsModeSelect", "change", (e) => {
    const next = sanitizeStatsMode(e.target.value);
    state.statsMode = next;
    localStorage.setItem("stats_mode", next);
    syncStatsControlVisibility();
    loadStatistics().catch((err) => alert(err.message || "加载统计失败"));
  });
  on("statsMinLevelSelect", "change", (e) => {
    const next = sanitizeStatsLevel(e.target.value, state.statsMinLevel);
    state.statsMinLevel = next;
    const fixed = normalizeStatsLevelRange(state.statsMinLevel, state.statsMaxLevel);
    state.statsMinLevel = fixed.min;
    state.statsMaxLevel = fixed.max;
    if ($("statsMinLevelSelect")) $("statsMinLevelSelect").value = String(state.statsMinLevel);
    if ($("statsMaxLevelSelect")) $("statsMaxLevelSelect").value = String(state.statsMaxLevel);
    localStorage.setItem("stats_min_level", String(state.statsMinLevel));
    localStorage.setItem("stats_max_level", String(state.statsMaxLevel));
    loadStatistics().catch((err) => alert(err.message || "加载统计失败"));
  });
  on("statsMaxLevelSelect", "change", (e) => {
    const next = sanitizeStatsLevel(e.target.value, state.statsMaxLevel);
    state.statsMaxLevel = next;
    const fixed = normalizeStatsLevelRange(state.statsMinLevel, state.statsMaxLevel);
    state.statsMinLevel = fixed.min;
    state.statsMaxLevel = fixed.max;
    if ($("statsMinLevelSelect")) $("statsMinLevelSelect").value = String(state.statsMinLevel);
    if ($("statsMaxLevelSelect")) $("statsMaxLevelSelect").value = String(state.statsMaxLevel);
    localStorage.setItem("stats_min_level", String(state.statsMinLevel));
    localStorage.setItem("stats_max_level", String(state.statsMaxLevel));
    loadStatistics().catch((err) => alert(err.message || "加载统计失败"));
  });
  on("uploadModal", "click", (e) => {
    if (e.target.id === "uploadModal") closeUploadModal();
  });
  on("closeMusicDetailBtn", "click", closeMusicDetail);
  on("addMusicAliasBtn", "click", () => addMusicAlias().catch((e) => alert(e.message || "新增别名失败")));
  on("musicDetailModal", "click", (e) => {
    if (e.target.id === "musicDetailModal") closeMusicDetail();
  });
  window.addEventListener("resize", () => {
    renderB30Trend(state.trendPoints || [], state.b30CalcMode);
  });
}

async function bootstrap() {
  renderAuthPanel("");
  initTabs();
  initDiffPicker();
  initCropInteractions();
  bindEvents();
  await refreshAll();
}

bootstrap().catch((e) => {
  console.error(e);
  alert(e.message || "初始化失败");
});


