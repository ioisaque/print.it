import { api } from "./api.js";
import { t } from "./i18n.js";
import { applyPreviewAlign, bindFormSubmit, readPrintOptions, toast, value } from "./ui.js";

let filePreviewUrl = "";

const acceptedTypes = new Set([
  "application/pdf",
  "image/png",
  "image/jpeg",
  "image/jpg",
]);

function isPdfFile(file) {
  return file.type === "application/pdf" || file.name.toLowerCase().endsWith(".pdf");
}

function isAcceptedFile(file) {
  if (!file) return false;
  const lower = file.name.toLowerCase();
  return (
    acceptedTypes.has(file.type) ||
    lower.endsWith(".pdf") ||
    lower.endsWith(".png") ||
    lower.endsWith(".jpg") ||
    lower.endsWith(".jpeg")
  );
}

function setFileInput(file) {
  const input = document.getElementById("fileInput");
  const dt = new DataTransfer();
  dt.items.add(file);
  input.files = dt.files;
}

function showDropzone() {
  document.getElementById("fileDropzone").hidden = false;
  document.getElementById("filePreviewView").hidden = true;
}

function showPreviewView() {
  document.getElementById("fileDropzone").hidden = true;
  document.getElementById("filePreviewView").hidden = false;
}

function resetPreviewMedia() {
  const img = document.getElementById("filePreviewImg");
  img.hidden = true;
  img.removeAttribute("src");
}

function clearFilePreview() {
  if (filePreviewUrl) {
    URL.revokeObjectURL(filePreviewUrl);
    filePreviewUrl = "";
  }

  resetPreviewMedia();
  document.getElementById("fileInput").value = "";
  showDropzone();
}

async function showFilePreview(file) {
  if (!file?.size) {
    toast(t("toast.fileEmpty"), false);
    return;
  }

  if (filePreviewUrl) {
    URL.revokeObjectURL(filePreviewUrl);
  }

  resetPreviewMedia();

  const fd = new FormData();
  fd.append("file", file);

  try {
    const res = await api.preview(fd);
    if (!res.ok) {
      const data = await res.json().catch(() => null);
      throw new Error(data?.error || res.statusText);
    }
    const blob = await res.blob();
    filePreviewUrl = URL.createObjectURL(blob);
    const img = document.getElementById("filePreviewImg");
    img.src = filePreviewUrl;
    img.hidden = false;
    applyPreviewAlign(value("textAlign"));
    showPreviewView();
  } catch (err) {
    toast(err.message || t("toast.previewFailed"), false);
    clearFilePreview();
  }
}

function handleSelectedFile(file) {
  if (!isAcceptedFile(file)) {
    toast(t("toast.fileTypes"), false);
    return;
  }
  setFileInput(file);
  showFilePreview(file);
}

export function initFile() {
  const dropzone = document.getElementById("fileDropzone");
  const input = document.getElementById("fileInput");

  dropzone.addEventListener("click", () => input.click());

  input.addEventListener("change", (e) => {
    const file = e.target.files?.[0];
    if (!file) {
      clearFilePreview();
      return;
    }
    handleSelectedFile(file);
  });

  dropzone.addEventListener("dragover", (e) => {
    e.preventDefault();
    dropzone.classList.add("dragover");
  });

  dropzone.addEventListener("dragleave", () => {
    dropzone.classList.remove("dragover");
  });

  dropzone.addEventListener("drop", (e) => {
    e.preventDefault();
    dropzone.classList.remove("dragover");
    const file = e.dataTransfer?.files?.[0];
    if (file) handleSelectedFile(file);
  });

  document.getElementById("btnChangeFile").addEventListener("click", () => {
    clearFilePreview();
  });

  bindFormSubmit("btnPrintFile", async () => {
    const file = document.getElementById("fileInput").files[0];
    if (!file) throw new Error(t("toast.selectFile"));

    const opts = readPrintOptions();
    const fd = new FormData();
    fd.append("file", file);
    fd.append("cut_after_page", opts.cut_after_page);
    fd.append("cut_after_document", opts.cut_after_document);
    fd.append("trim_blank", opts.trim_blank);

    if (isPdfFile(file)) {
      await api.pdf.send(fd);
      toast(t("toast.pdfSent"));
      return;
    }

    await api.image.send(fd);
    toast(t("toast.imageSent"));
  });
}
