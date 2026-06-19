import { api } from "./api.js";
import { initBarcode } from "./barcode.js";
import { initImage } from "./image.js";
import { initPdf } from "./pdf.js";
import { initPrinter, loadConfigForm } from "./printer.js";
import { initQrcode } from "./qrcode.js";
import { initText } from "./text.js";
import { initTabs, refreshStatus } from "./ui.js";

initTabs();
initPrinter();
initText();
initPdf();
initBarcode();
initQrcode();
initImage();

loadConfigForm().catch(() => {});
refreshStatus(api);
setInterval(() => refreshStatus(api), 15000);
