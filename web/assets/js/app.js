import { api } from "./api.js";
import { initBarcode } from "./barcode.js";
import { initFile } from "./file.js";
import { initI18n, onLangChange } from "./i18n.js";
import { initPrinter, loadConfigForm } from "./printer.js";
import { initQrcode } from "./qrcode.js";
import { initText } from "./text.js";
import { applyPrefsToUI, initNavigation, initPrintTypes, refreshStatus, syncReceiptPaperWidth } from "./ui.js";

await initI18n();

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
onLangChange(() => refreshStatus(api));
refreshStatus(api);
setInterval(() => refreshStatus(api), 30000);
