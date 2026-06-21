import { api } from "./api.js";
import { t } from "./i18n.js";
import { applyQrcodePreviewSize, bindFormSubmit, readPrintOptions, toast, value } from "./ui.js";

let previewTimer = null;
let previewUrl = "";

function qrcodePreviewUrl(data) {
  const params = new URLSearchParams();
  params.set("txt", data.trim() || " ");
  return `${window.location.origin}/printit/barcodes/preview?${params.toString()}`;
}

function updatePreviewLabel() {
  const label = value("qrLabel").trim();
  const el = document.getElementById("qrPreviewLabel");
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
  const img = document.getElementById("qrPreviewImg");
  img.hidden = true;
  img.removeAttribute("src");
  img.style.width = "";
  img.style.height = "";
  delete img.dataset.qrData;
}

async function updatePreview() {
  const data = value("qrData");
  const img = document.getElementById("qrPreviewImg");

  updatePreviewLabel();

  if (!data.trim()) {
    clearPreview();
    return;
  }

  try {
    const res = await fetch(qrcodePreviewUrl(data));
    if (!res.ok) throw new Error();
    const blob = await res.blob();
    if (previewUrl) URL.revokeObjectURL(previewUrl);
    previewUrl = URL.createObjectURL(blob);
    img.dataset.qrData = data.trim();
    img.onload = () => applyQrcodePreviewSize(img, data.trim());
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

export function initQrcode() {
  document.getElementById("qrData")?.addEventListener("input", schedulePreview);
  document.getElementById("qrLabel")?.addEventListener("input", updatePreviewLabel);

  updatePreview();

  bindFormSubmit("btnPrintQr", async () => {
    const opts = readPrintOptions();
    await api.qrcode.send({
      data: value("qrData").trim(),
      label: value("qrLabel").trim(),
      align: value("textAlign"),
      cut_after_document: opts.cut_after_document,
    });
    toast(t("toast.qrcodeSent"));
  });
}
