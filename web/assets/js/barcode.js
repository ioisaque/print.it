import { api } from "./api.js";
import { bindFormSubmit, checkbox, toast, value } from "./ui.js";

export function initBarcode() {
  bindFormSubmit("btnPrintBc", async () => {
    await api.barcode.send({
      type: value("bcType"),
      data: value("bcData").trim(),
      label: value("bcLabel").trim(),
      align: value("bcAlign"),
      cut: checkbox("bcCut"),
    });
    toast("Código de barras enviado");
  });
}
