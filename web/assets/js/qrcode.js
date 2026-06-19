import { api } from "./api.js";
import { bindFormSubmit, checkbox, toast, value } from "./ui.js";

export function initQrcode() {
  bindFormSubmit("btnPrintQr", async () => {
    await api.qrcode.send({
      data: value("qrData").trim(),
      label: value("qrLabel").trim(),
      align: value("qrAlign"),
      cut: checkbox("qrCut"),
    });
    toast("QR Code enviado");
  });
}
