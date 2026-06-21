import { api } from "./api.js";
import { initBarcode } from "./barcode.js";
import { initFile } from "./file.js";
import { initPrinter, loadConfigForm } from "./printer.js";
import { initQrcode } from "./qrcode.js";
import { initText } from "./text.js";
import { applyPrefsToUI, initNavigation, initPrintTypes, refreshStatus, syncReceiptPaperWidth } from "./ui.js";

initNavigation();
initPrintTypes();
initPrinter();
initText();
initFile();
  initBarcode();
  initQrcode();

  applyPrefsToUI();
  syncReceiptPaperWidth();
loadConfigForm().catch(() => {});
refreshStatus(api);
setInterval(() => refreshStatus(api), 15000);
