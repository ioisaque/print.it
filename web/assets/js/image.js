import { api } from "./api.js";
import { bindFormSubmit, readPrintOptions, toast } from "./ui.js";

export function initImage() {
  bindFormSubmit("btnPrintImg", async () => {
    const file = document.getElementById("imgFile").files[0];
    if (!file) throw new Error("Selecione uma imagem");

    const opts = readPrintOptions();
    const fd = new FormData();
    fd.append("file", file);
    fd.append("cut", opts.cut);
    fd.append("trim_trailing_blank", opts.trim);

    await api.image.send(fd);
    toast("Imagem enviada");
  });
}
