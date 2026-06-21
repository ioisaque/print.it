import { api } from "./api.js";
import { bindFormSubmit, checkbox, readPrintOptions, toast } from "./ui.js";

export function initPdf() {
  bindFormSubmit("btnPrintPdf", async () => {
    const file = document.getElementById("pdfFile").files[0];
    if (!file) throw new Error("Selecione um PDF");

    const opts = readPrintOptions();
    const fd = new FormData();
    fd.append("file", file);
    fd.append("cut", opts.cut);
    fd.append("cut_between_pages", checkbox("pdfCutBetween"));
    fd.append("trim_trailing_blank", opts.trim);

    await api.pdf.send(fd);
    toast("PDF enviado");
  });
}
