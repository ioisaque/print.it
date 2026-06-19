import { api } from "./api.js";
import { bindFormSubmit, checkbox, refreshStatus, toast, value } from "./ui.js";

function escapeHtml(text) {
  return String(text)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function isUsefulField(value) {
  if (!value) return false;
  const lower = String(value).trim().toLowerCase();
  return lower !== "<nil>" && lower !== "nil" && lower !== "n/a" && !lower.startsWith("bsa/");
}

function renderPrinterItem(p, index) {
  const title = escapeHtml(p.label || `Impressora ${index + 1}`);
  const details = [
    p.device_type || "Impressora térmica",
    isUsefulField(p.manufacturer) ? `Marca: ${p.manufacturer}` : null,
    isUsefulField(p.model) ? `Modelo: ${p.model}` : null,
    isUsefulField(p.mac_vendor) && !isUsefulField(p.manufacturer) ? `Chip de rede: ${p.mac_vendor}` : null,
    `Endereço: ${p.host}`,
    isUsefulField(p.hostname) && p.hostname !== p.name ? `Nome na rede: ${p.hostname}` : null,
    isUsefulField(p.serial) ? `Série: ${p.serial}` : null,
  ].filter(Boolean);

  return `
    <div class="printer-item${p.configured ? " selected" : ""}">
      <div class="printer-info">
        <div class="printer-title-row">
          <strong>${title}</strong>
          ${p.configured ? '<span class="printer-badge">Em uso</span>' : ""}
        </div>
        ${details.map((line) => `<span class="printer-meta">${escapeHtml(line)}</span>`).join("")}
      </div>
      <button class="btn secondary" data-host="${p.host}" data-port="${p.port}">Usar</button>
    </div>`;
}

async function runDiscover({ deep = false } = {}) {
  const btn = document.getElementById(deep ? "btnDiscoverDeep" : "btnDiscover");
  const otherBtn = document.getElementById(deep ? "btnDiscover" : "btnDiscoverDeep");
  const list = document.getElementById("printerList");
  const defaultLabel = deep ? "Varredura completa" : "Buscar na rede";

  btn.disabled = true;
  otherBtn.disabled = true;
  btn.textContent = "Buscando...";
  list.innerHTML = `<p class="empty">${
    deep
      ? "Varredura completa em andamento (todos os métodos, pode levar ~20s)..."
      : "Buscando impressoras identificáveis na rede..."
  }</p>`;

  try {
    const data = await api.discover(deep);
    if (!data.printers?.length) {
      list.innerHTML = deep
        ? '<p class="empty">Nenhuma impressora encontrada na porta 9100.</p>'
        : '<p class="empty">Nenhuma impressora identificável encontrada. Tente &quot;Varredura completa&quot; para ver todos os dispositivos na porta 9100.</p>';
      return;
    }

    const hint = deep
      ? '<p class="hint">Varredura completa: inclui dispositivos com pouca informação. Use &quot;Imprimir teste&quot; para identificar qual é qual.</p>'
      : '<p class="hint">Não encontrou a sua? Use &quot;Varredura completa&quot; ou imprima um teste em cada uma para identificar.</p>';

    list.innerHTML = data.printers.map((p, index) => renderPrinterItem(p, index)).join("") + hint;

    list.querySelectorAll("button[data-host]").forEach((b) => {
      b.addEventListener("click", async () => {
        try {
          await api.config.put({
            printer_host: b.dataset.host,
            printer_port: parseInt(b.dataset.port, 10),
          });
          toast("Impressora selecionada");
          loadConfigForm();
          refreshStatus(api);
        } catch (err) {
          toast(err.message, false);
        }
      });
    });

    const modeLabel = deep ? "completa" : "rápida";
    toast(`${data.count} impressora(s) na varredura ${modeLabel} (${data.duration})`);
  } catch (err) {
    toast(err.message, false);
    list.innerHTML = '<p class="empty">Falha na busca.</p>';
  } finally {
    btn.disabled = false;
    otherBtn.disabled = false;
    btn.textContent = defaultLabel;
  }
}

export function initPrinter() {
  bindFormSubmit("btnSaveConfig", async () => {
    await api.config.put({
      printer_host: value("cfgHost").trim(),
      printer_port: parseInt(value("cfgPort"), 10) || 9100,
      paper_width_mm: parseInt(value("cfgPaper"), 10),
      printable_width_mm: parseInt(value("cfgPrintable"), 10) || 0,
      trim_trailing_blank: checkbox("cfgTrim"),
    });
    toast("Configuração salva");
    refreshStatus(api);
  });

  document.getElementById("btnDiscover").addEventListener("click", () => runDiscover({ deep: false }));
  document.getElementById("btnDiscoverDeep").addEventListener("click", () => runDiscover({ deep: true }));

  bindFormSubmit("btnTest", async () => {
    const data = await api.test();
    toast(data.message || "Teste enviado");
  });
}

export async function loadConfigForm() {
  const data = await api.status();
  const c = data.config;
  document.getElementById("cfgHost").value = c.printer_host || "";
  document.getElementById("cfgPort").value = c.printer_port || 9100;
  document.getElementById("cfgPaper").value = String(c.paper_width_mm || 80);
  document.getElementById("cfgPrintable").value = c.printable_width_mm || "";
  document.getElementById("cfgTrim").checked = !!c.trim_trailing_blank;
  document.getElementById("pdfTrim").checked = !!c.trim_trailing_blank;
  document.getElementById("textTrim").checked = !!c.trim_trailing_blank;
  document.getElementById("imgTrim").checked = !!c.trim_trailing_blank;
}
