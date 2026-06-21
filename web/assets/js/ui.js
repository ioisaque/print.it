import { t } from "./i18n.js";

const PREFS_KEY = "printit.prefs";
const PROFILE_KEY = "printit.printer";

const defaultPrefs = {
  cut_after_page: "none",
  cut_after_document: "partial",
  trim_blank: "never",
  paper_width_mm: 80,
  printable_width_mm: 0,
  text_align: "left",
};

function migratePrefs(raw) {
  const merged = { ...defaultPrefs, ...raw };
  if (raw.cut_default !== undefined && raw.cut_after_document === undefined) {
    merged.cut_after_document = raw.cut_default ? "partial" : "none";
  }
  if (raw.trim_trailing_blank !== undefined && raw.trim_blank === undefined) {
    merged.trim_blank = raw.trim_trailing_blank ? "document" : "never";
  }
  return merged;
}

export function loadPrefs() {
  try {
    return migratePrefs(JSON.parse(localStorage.getItem(PREFS_KEY) || "{}"));
  } catch {
    return { ...defaultPrefs };
  }
}

export function savePrefs(prefs) {
  localStorage.setItem(PREFS_KEY, JSON.stringify({ ...loadPrefs(), ...prefs }));
}

export function loadPrinterProfile() {
  try {
    return JSON.parse(localStorage.getItem(PROFILE_KEY) || "null");
  } catch {
    return null;
  }
}

export function savePrinterProfile(profile) {
  if (profile) {
    localStorage.setItem(PROFILE_KEY, JSON.stringify(profile));
  } else {
    localStorage.removeItem(PROFILE_KEY);
  }
}

export function applyPrefsToUI() {
  const prefs = loadPrefs();
  const prefPaper = document.getElementById("prefPaper");
  const prefCutPage = document.getElementById("prefCutPage");
  const prefCutDoc = document.getElementById("prefCutDoc");
  const prefTrimBlank = document.getElementById("prefTrimBlank");
  const cfgPaper = document.getElementById("cfgPaper");
  const textAlign = document.getElementById("textAlign");

  if (prefPaper) prefPaper.value = String(prefs.paper_width_mm || 80);
  if (prefCutPage) prefCutPage.value = prefs.cut_after_page || "none";
  if (prefCutDoc) prefCutDoc.value = prefs.cut_after_document || "partial";
  if (prefTrimBlank) prefTrimBlank.value = prefs.trim_blank || "never";
  if (cfgPaper) cfgPaper.value = String(prefs.paper_width_mm || 80);
  if (textAlign && prefs.text_align) {
    textAlign.value = prefs.text_align;
    document.querySelectorAll(".format-btn[data-align]").forEach((btn) => {
      btn.classList.toggle("active", btn.dataset.align === prefs.text_align);
    });
  }

  resyncThermalPreviews();
}

export function readPrintOptions() {
  const prefs = loadPrefs();
  return {
    cut_after_page: prefs.cut_after_page || "none",
    cut_after_document: prefs.cut_after_document || "partial",
    trim_blank: prefs.trim_blank || "never",
  };
}

export function readPrefsFromPopover() {
  return {
    cut_after_page: document.getElementById("prefCutPage")?.value || "none",
    cut_after_document: document.getElementById("prefCutDoc")?.value || "partial",
    trim_blank: document.getElementById("prefTrimBlank")?.value || "never",
    paper_width_mm: parseInt(document.getElementById("prefPaper")?.value, 10) || 80,
  };
}

export function toast(message, ok = true) {
  const el = document.createElement("div");
  el.className = "toast " + (ok ? "ok" : "err");
  el.textContent = message;
  document.body.appendChild(el);
  setTimeout(() => el.remove(), 4000);
}

export function showPanel(tab) {
  document.querySelectorAll(".panel").forEach((p) => p.classList.remove("active"));
  document.getElementById("panel-" + tab)?.classList.add("active");
}

export function initNavigation() {
  document.getElementById("btnHome")?.addEventListener("click", () => showPanel("print"));
}

const printButtonIds = {
  text: "btnPrintText",
  file: "btnPrintFile",
  barcode: "btnPrintBc",
  qrcode: "btnPrintQr",
};

function showPrintButton(type) {
  Object.entries(printButtonIds).forEach(([key, id]) => {
    const btn = document.getElementById(id);
    if (btn) btn.hidden = key !== type;
  });
}

export function initPrintTypes() {
  const types = document.querySelectorAll("#printTypes button");
  types.forEach((btn) => {
    btn.addEventListener("click", () => {
      types.forEach((b) => b.classList.remove("active"));
      document.querySelectorAll(".print-form").forEach((f) => f.classList.remove("active"));
      btn.classList.add("active");
      document.getElementById("print-form-" + btn.dataset.printType).classList.add("active");
      showPrintButton(btn.dataset.printType);
    });
  });

  const active = document.querySelector("#printTypes button.active");
  if (active) showPrintButton(active.dataset.printType);
}

export async function refreshStatus(api) {
  const icon = document.getElementById("printerIcon");
  const label = document.getElementById("printerProfileLabel");
  const profile = loadPrinterProfile();

  try {
    const data = await api.status();
    const host = data.config?.printer_host;

    icon?.classList.remove("idle", "ok", "err");

    if (!host) {
      icon?.classList.add("idle");
      label.textContent = t("printer.connect");
      return;
    }

    icon?.classList.add("ok");

    if (profile?.host === host && profile?.label) {
      label.textContent = profile.label;
    } else {
      label.textContent = t("printer.connected");
    }
  } catch {
    icon?.classList.remove("idle", "ok", "err");
    icon?.classList.add(profile?.host ? "err" : "idle");
    label.textContent = profile?.label || t("printer.connect");
  }
}

export function bindFormSubmit(buttonId, handler) {
  document.getElementById(buttonId).addEventListener("click", async () => {
    const btn = document.getElementById(buttonId);
    btn.disabled = true;
    try {
      await handler();
    } catch (err) {
      toast(err.message || t("toast.error"), false);
    } finally {
      btn.disabled = false;
    }
  });
}

export function checkbox(id) {
  return document.getElementById(id).checked;
}

export function value(id) {
  return document.getElementById(id).value;
}

export const THERMAL_DPI = 203;
export const BARCODE_HEIGHT_DOTS = 80;
export const QR_MODULE_DOTS = 6;

const QR_BYTE_CAPACITIES_EC_M = [
  16, 28, 44, 64, 86, 108, 124, 154, 182, 216, 254, 290, 330, 372, 412, 450, 504, 560, 624,
  666, 711, 779, 857, 911, 997, 1059, 1125, 1190, 1264, 1370, 1452, 1538, 1628, 1722, 1809,
  1911, 1989, 2099, 2213, 2331,
];

export function getPrintableWidthMm() {
  const prefs = loadPrefs();
  if (prefs.printable_width_mm > 0) return prefs.printable_width_mm;
  const paper = parseInt(prefs.paper_width_mm, 10) || 80;
  if (paper >= 80) return 72;
  if (paper > 0 && paper <= 58) return 48;
  if (paper > 0) return paper;
  return 72;
}

export function dotsToMm(dots) {
  return (dots / THERMAL_DPI) * 25.4;
}

export function qrPrintDots(data) {
  const bytes = new TextEncoder().encode(data);
  let version = QR_BYTE_CAPACITIES_EC_M.length;
  for (let i = 0; i < QR_BYTE_CAPACITIES_EC_M.length; i++) {
    if (bytes.length <= QR_BYTE_CAPACITIES_EC_M[i]) {
      version = i + 1;
      break;
    }
  }
  const modules = 21 + (version - 1) * 4;
  return (modules + 8) * QR_MODULE_DOTS;
}

export function applyBarcodePreviewSize(img) {
  if (!img?.naturalWidth) return;
  img.style.width = `${dotsToMm(img.naturalWidth)}mm`;
  img.style.height = `${dotsToMm(img.naturalHeight)}mm`;
}

export function applyQrcodePreviewSize(img, data) {
  const mm = dotsToMm(qrPrintDots(data));
  img.style.width = `${mm}mm`;
  img.style.height = `${mm}mm`;
}

export function syncReceiptPaperWidth() {
  const mm = getPrintableWidthMm();
  document.querySelectorAll(".receipt-wrap").forEach((el) => {
    el.style.setProperty("--printable-mm", String(mm));
  });
}

export function resyncThermalPreviews() {
  syncReceiptPaperWidth();
  const bcImg = document.getElementById("bcPreviewImg");
  if (bcImg && !bcImg.hidden && bcImg.naturalWidth) {
    applyBarcodePreviewSize(bcImg);
  }
  const qrImg = document.getElementById("qrPreviewImg");
  if (qrImg && !qrImg.hidden && qrImg.dataset.qrData) {
    applyQrcodePreviewSize(qrImg, qrImg.dataset.qrData);
  }
}
