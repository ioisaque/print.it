import { api } from "./api.js";
import { bindFormSubmit, checkbox, toast, value } from "./ui.js";

export function initText() {
  bindFormSubmit("btnPrintText", async () => {
    await api.text.send({
      text: value("textContent"),
      align: value("textAlign"),
      bold: checkbox("textBold"),
      cut: checkbox("textCut"),
      trim_trailing_blank: checkbox("textTrim"),
    });
    toast("Texto enviado");
  });
}
