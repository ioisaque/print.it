import { api } from "./api.js";
import { applyBarcodePreviewSize, bindFormSubmit, readPrintOptions, toast, value } from "./ui.js";

const BC_TYPE_MAP = {
  CODE128: "code128",
  CODE39: "code39",
  EAN13: "ean13",
  EAN8: "ean8",
  ITF: "itf",
};

let previewTimer = null;
let previewUrl = "";

function barcodePreviewUrl(data, type) {
  const params = new URLSearchParams();
  params.set("txt", data.trim() || " ");
  const mapped = BC_TYPE_MAP[type] || "code128";
  params.set("type", mapped);
  return `https://api.isaque.it/barcodes?${params.toString()}`;
}

function updatePreviewLabel() {
  const label = value("bcLabel").trim();
  const el = document.getElementById("bcPreviewLabel");
  if (!label) {
    el.hidden = true;
    el.textContent = "";
    return;
  }
  el.textContent = label;
  el.hidden = false;
}

function clearPreview() {
  if (previewUrl) {
    URL.revokeObjectURL(previewUrl);
    previewUrl = "";
  }
  const img = document.getElementById("bcPreviewImg");
  img.hidden = true;
  img.removeAttribute("src");
  img.style.width = "";
  img.style.height = "";
}

async function updatePreview() {
  const data = value("bcData");
  const type = value("bcType");
  const img = document.getElementById("bcPreviewImg");

  updatePreviewLabel();

  if (!data.trim()) {
    clearPreview();
    return;
  }

  try {
    const res = await fetch(barcodePreviewUrl(data, type));
    if (!res.ok) throw new Error();
    const blob = await res.blob();
    if (previewUrl) URL.revokeObjectURL(previewUrl);
    previewUrl = URL.createObjectURL(blob);
    img.onload = () => applyBarcodePreviewSize(img);
    img.src = previewUrl;
    img.hidden = false;
  } catch {
    clearPreview();
  }
}

function schedulePreview() {
  clearTimeout(previewTimer);
  previewTimer = setTimeout(updatePreview, 350);
}

export function initBarcode() {
  document.getElementById("bcData")?.addEventListener("input", schedulePreview);
  document.getElementById("bcType")?.addEventListener("change", schedulePreview);
  document.getElementById("bcLabel")?.addEventListener("input", updatePreviewLabel);

  updatePreview();

  bindFormSubmit("btnPrintBc", async () => {
    const opts = readPrintOptions();
    await api.barcode.send({
      type: value("bcType"),
      data: value("bcData").trim(),
      label: value("bcLabel").trim(),
      align: value("textAlign"),
      cut_after_document: opts.cut_after_document,
    });
    toast("Código de barras enviado");
  });
}
