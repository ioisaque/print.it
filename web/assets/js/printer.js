import { api } from "./api.js";
import { bindFormSubmit, checkbox, refreshStatus, toast, value } from "./ui.js";

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

  document.getElementById("btnDiscover").addEventListener("click", async () => {
    const btn = document.getElementById("btnDiscover");
    const list = document.getElementById("printerList");
    btn.disabled = true;
    btn.textContent = "Buscando...";
    list.innerHTML = '<p class="empty">Varredura em andamento (pode levar ~10s)...</p>';

    try {
      const data = await api.discover();
      if (!data.printers?.length) {
        list.innerHTML = '<p class="empty">Nenhuma impressora encontrada na porta 9100.</p>';
        return;
      }

      list.innerHTML = data.printers
        .map((p) => {
          const title = [p.manufacturer, p.model].filter(Boolean).join(" ") || `${p.host}:${p.port}`;
          const meta = [
            p.model || p.manufacturer ? `${p.host}:${p.port}` : null,
            p.hostname,
            p.mac,
            p.serial ? `S/N ${p.serial}` : null,
          ]
            .filter(Boolean)
            .join(" · ");

          return `
        <div class="printer-item${p.configured ? " selected" : ""}">
          <div class="printer-info">
            <strong>${title}</strong>
            ${meta ? `<span class="printer-meta">${meta}</span>` : ""}
          </div>
          <button class="btn secondary" data-host="${p.host}" data-port="${p.port}">Usar</button>
        </div>`;
        })
        .join("");

      list.querySelectorAll("button").forEach((b) => {
        b.addEventListener("click", async () => {
          try {
            await api.config.put({
              printer_host: b.dataset.host,
              printer_port: parseInt(b.dataset.port, 10),
            });
            toast("Impressora selecionada: " + b.dataset.host);
            loadConfigForm();
            refreshStatus(api);
          } catch (err) {
            toast(err.message, false);
          }
        });
      });

      toast(`${data.count} impressora(s) em ${data.duration}`);
    } catch (err) {
      toast(err.message, false);
      list.innerHTML = '<p class="empty">Falha na busca.</p>';
    } finally {
      btn.disabled = false;
      btn.textContent = "Buscar na rede";
    }
  });

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
