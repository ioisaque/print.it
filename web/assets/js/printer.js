import { api } from "./api.js";
import {
    applyPrefsToUI,
    bindFormSubmit,
    loadPrefs,
    loadPrinterProfile,
    readPrefsFromPopover,
    refreshStatus,
    savePrefs,
    savePrinterProfile,
    showPanel,
    toast,
    value,
} from "./ui.js";

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
    isUsefulField(p.manufacturer) ? `Marca: ${p.manufacturer}` : null,
    isUsefulField(p.model) ? `Modelo: ${p.model}` : null,
    `Endereço: ${p.host}`,
  ].filter(Boolean);

  return `
    <div class="printer-item${p.configured ? " selected" : ""}">
      <div class="printer-info">
        <strong>${title}</strong>
        ${details.map((line) => `<span class="printer-meta">${escapeHtml(line)}</span>`).join("")}
      </div>
      <button type="button" class="btn secondary" data-host="${p.host}" data-port="${p.port}">Usar</button>
    </div>`;
}

function renderProfileDetails(profile) {
  const lines = [
    profile.manufacturer ? `Marca: ${profile.manufacturer}` : null,
    profile.model ? `Modelo: ${profile.model}` : null,
    profile.serial ? `Série: ${profile.serial}` : null,
    profile.mac ? `MAC: ${profile.mac}` : null,
    profile.mac_vendor ? `Chip de rede: ${profile.mac_vendor}` : null,
  ].filter(Boolean);

  return lines.map((line) => `<span class="printer-meta">${escapeHtml(line)}</span>`).join("");
}

function showPopoverView(connected) {
  document.getElementById("popoverConnectView").hidden = connected;
  document.getElementById("popoverProfileView").hidden = !connected;
}

function updateProfilePopover() {
  const profile = loadPrinterProfile();
  if (!profile?.host) {
    showPopoverView(false);
    return;
  }

  showPopoverView(true);
  const address = profile.host ? `${profile.host}:${profile.port || 9100}` : "";
  document.getElementById("popoverPrinterAddress").textContent = address;
  document.getElementById("popoverPrinterDetails").innerHTML = renderProfileDetails(profile);
  applyPrefsToUI();
}

function togglePopover(open) {
  const popover = document.getElementById("printerPopover");
  const btn = document.getElementById("printerProfileBtn");
  const shouldOpen = open ?? popover.hidden;

  if (shouldOpen) {
    updateProfilePopover();
    popover.hidden = false;
    btn.setAttribute("aria-expanded", "true");
  } else {
    popover.hidden = true;
    btn.setAttribute("aria-expanded", "false");
  }
}

async function persistPrefs(prefs) {
  savePrefs(prefs);
  applyPrefsToUI();

  await api.config.put({
    paper_width_mm: prefs.paper_width_mm,
    trim_trailing_blank: prefs.trim_blank !== "never",
  });
}

async function selectPrinter(host, port, printerData = {}) {
  await api.config.put({
    printer_host: host,
    printer_port: port,
  });

  const profile = {
    host,
    port,
    label: printerData.label || printerData.name || "Impressora conectada",
    manufacturer: printerData.manufacturer || "",
    model: printerData.model || "",
    serial: printerData.serial || "",
    mac: printerData.mac || "",
    mac_vendor: printerData.mac_vendor || "",
  };

  savePrinterProfile(profile);
  await loadConfigForm();
  refreshStatus(api);
  updateProfilePopover();
  showPopoverView(true);
  toast("Impressora conectada");
}

async function runDiscover({ deep = false } = {}) {
  const btn = document.getElementById(deep ? "btnDiscoverDeep" : "btnDiscover");
  const list = document.getElementById("printerList");
  const defaultLabel = deep ? "Varredura completa" : "Buscar na rede";

  btn.disabled = true;
  btn.textContent = deep ? "..." : "Buscando...";
  list.innerHTML = `<p class="empty">${deep ? "Varredura completa..." : "Buscando impressoras..."}</p>`;

  try {
    const data = await api.discover(deep);
    if (!data.printers?.length) {
      list.innerHTML = deep
        ? '<p class="empty">Nenhuma impressora encontrada.</p>'
        : '<p class="empty">Nenhuma identificável. Use varredura completa no canto.</p>';
      return;
    }

    list.innerHTML = data.printers.map((p, index) => renderPrinterItem(p, index)).join("");

    list.querySelectorAll("button[data-host]").forEach((b) => {
      b.addEventListener("click", async () => {
        try {
          const printer = data.printers.find((p) => p.host === b.dataset.host) || {};
          await selectPrinter(b.dataset.host, parseInt(b.dataset.port, 10), printer);
        } catch (err) {
          toast(err.message, false);
        }
      });
    });

    toast(`${data.count} impressora(s) encontrada(s)`);
  } catch (err) {
    toast(err.message, false);
    list.innerHTML = '<p class="empty">Falha na busca.</p>';
  } finally {
    btn.disabled = false;
    btn.textContent = defaultLabel;
  }
}

export function initPrinter() {
  document.getElementById("printerProfileBtn").addEventListener("click", (e) => {
    e.stopPropagation();
    togglePopover();
  });

  document.addEventListener("click", (e) => {
    const wrap = document.querySelector(".printer-profile-wrap");
    if (!wrap?.contains(e.target)) {
      togglePopover(false);
    }
  });

  document.getElementById("printerPopover").addEventListener("click", (e) => e.stopPropagation());

  document.getElementById("btnChangePrinter").addEventListener("click", () => {
    showPopoverView(false);
  });

  document.getElementById("btnOpenAdvanced").addEventListener("click", () => {
    togglePopover(false);
    showPanel("advanced");
  });

  document.getElementById("btnDiscover").addEventListener("click", () => runDiscover({ deep: false }));
  document.getElementById("btnDiscoverDeep").addEventListener("click", () => runDiscover({ deep: true }));

  ["prefPaper", "prefCutPage", "prefCutDoc", "prefTrimBlank"].forEach((id) => {
    document.getElementById(id)?.addEventListener("change", async () => {
      try {
        await persistPrefs(readPrefsFromPopover());
      } catch (err) {
        toast(err.message, false);
      }
    });
  });

  bindFormSubmit("btnSaveConfig", async () => {
    const prefs = loadPrefs();
    prefs.paper_width_mm = parseInt(value("cfgPaper"), 10);
    prefs.printable_width_mm = parseInt(value("cfgPrintable"), 10) || 0;
    savePrefs(prefs);

    await api.config.put({
      printer_host: value("cfgHost").trim(),
      printer_port: parseInt(value("cfgPort"), 10) || 9100,
      paper_width_mm: prefs.paper_width_mm,
      printable_width_mm: prefs.printable_width_mm,
      trim_trailing_blank: prefs.trim_blank !== "never",
    });

    const profile = loadPrinterProfile();
    if (profile) {
      savePrinterProfile({
        ...profile,
        host: value("cfgHost").trim(),
        port: parseInt(value("cfgPort"), 10) || 9100,
      });
    }

    applyPrefsToUI();
    toast("Configuração salva");
    refreshStatus(api);
  });

  bindFormSubmit("btnTest", async () => {
    const data = await api.test();
    toast(data.message || "Teste enviado");
  });
}

export async function loadConfigForm() {
  const data = await api.status();
  const c = data.config;
  const prefs = loadPrefs();

  document.getElementById("cfgHost").value = c.printer_host || "";
  document.getElementById("cfgPort").value = c.printer_port || 9100;
  document.getElementById("cfgPaper").value = String(prefs.paper_width_mm || c.paper_width_mm || 80);
  document.getElementById("cfgPrintable").value = prefs.printable_width_mm || c.printable_width_mm || "";

  applyPrefsToUI();

  const profile = loadPrinterProfile();
  if (c.printer_host) {
    if (!profile || profile.host !== c.printer_host) {
      savePrinterProfile({
        host: c.printer_host,
        port: c.printer_port || 9100,
        label: profile?.label || "",
        manufacturer: profile?.manufacturer || "",
        model: profile?.model || "",
        serial: profile?.serial || "",
        mac: profile?.mac || "",
        mac_vendor: profile?.mac_vendor || "",
      });
    }
  }
}
