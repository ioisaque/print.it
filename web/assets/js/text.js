import { api } from "./api.js";
import { t } from "./i18n.js";
import { bindFormSubmit, readPrintOptions, savePrefs, toast, value } from "./ui.js";

function textPreviewContent() {
  return document.getElementById("textPreview")?.innerText ?? "";
}

function setTextAlign(align) {
  document.getElementById("textAlign").value = align;
  document.querySelectorAll(".format-btn[data-align]").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.align === align);
  });
  savePrefs({ text_align: align });
  applyTextPreviewStyles();
  applyGlobalPreviewAlign(align);
}

function applyGlobalPreviewAlign(align) {
  ["bcPreviewBox", "qrPreviewBox"].forEach((id) => {
    document.getElementById(id)?.style.setProperty("text-align", align);
  });
}

function isTextBold() {
  return document.getElementById("textBold")?.getAttribute("aria-pressed") === "true";
}

function applyTextPreviewStyles() {
  const preview = document.getElementById("textPreview");
  const align = value("textAlign");
  const bold = isTextBold();
  const hasContent = !!textPreviewContent().trim();

  preview.classList.toggle("is-bold", bold);
  preview.style.textAlign = hasContent ? align : "center";
}

export function initText() {
  const preview = document.getElementById("textPreview");

  document.querySelectorAll(".format-btn[data-align]").forEach((btn) => {
    btn.addEventListener("mousedown", (e) => e.preventDefault());
    btn.addEventListener("click", () => setTextAlign(btn.dataset.align));
  });

  const boldBtn = document.getElementById("textBold");
  boldBtn?.addEventListener("mousedown", (e) => e.preventDefault());
  boldBtn?.addEventListener("click", () => {
    const pressed = boldBtn.getAttribute("aria-pressed") === "true";
    boldBtn.setAttribute("aria-pressed", pressed ? "false" : "true");
    boldBtn.classList.toggle("active", !pressed);
    applyTextPreviewStyles();
  });

  preview?.addEventListener("input", applyTextPreviewStyles);
  preview?.addEventListener("focus", applyTextPreviewStyles);
  preview?.addEventListener("blur", () => {
    if (!textPreviewContent().trim()) {
      preview.innerHTML = "";
    }
    applyTextPreviewStyles();
  });

  applyTextPreviewStyles();
  applyGlobalPreviewAlign(value("textAlign"));

  bindFormSubmit("btnPrintText", async () => {
    const opts = readPrintOptions();
    await api.text.send({
      text: textPreviewContent(),
      align: value("textAlign"),
      bold: isTextBold(),
      cut_after_document: opts.cut_after_document,
      trim_blank: opts.trim_blank,
    });
    toast(t("toast.textSent"));
  });
}
