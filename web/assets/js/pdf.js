import { api } from "./api.js";
import { bindFormSubmit, checkbox, toast } from "./ui.js";

export function initPdf() {
  bindFormSubmit("btnPrintPdf", async () => {
    const file = document.getElementById("pdfFile").files[0];
    if (!file) throw new Error("Selecione um PDF");

    const fd = new FormData();
    fd.append("file", file);
    fd.append("cut", checkbox("pdfCut"));
    fd.append("cut_between_pages", checkbox("pdfCutBetween"));
    fd.append("trim_trailing_blank", checkbox("pdfTrim"));

    await api.pdf.send(fd);
    toast("PDF enviado");
  });
}
