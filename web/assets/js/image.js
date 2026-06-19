import { api } from "./api.js";
import { bindFormSubmit, checkbox, toast } from "./ui.js";

export function initImage() {
  bindFormSubmit("btnPrintImg", async () => {
    const file = document.getElementById("imgFile").files[0];
    if (!file) throw new Error("Selecione uma imagem");

    const fd = new FormData();
    fd.append("file", file);
    fd.append("cut", checkbox("imgCut"));
    fd.append("trim_trailing_blank", checkbox("imgTrim"));

    await api.image.send(fd);
    toast("Imagem enviada");
  });
}
