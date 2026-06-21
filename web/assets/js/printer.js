import { api } from "./api.js";
import { onLangChange, t } from "./i18n.js";
import {
  applyPrefsToUI,
  bindFormSubmit,
  loadPrefs,
  loadPrinterProfile,
  readPrefsFromPopover,
  refreshStatus,
  saveDiscoveredPrinters,
  savePrefs,
  savePrinterProfile,
  showPanel,
  toast,
  value,
} from "./ui.js";

const svgTestPrint = `<svg viewBox="0 0 512 512" fill="currentColor" aria-hidden="true"><path d="M0 0h512v512H0z" fill="none" /><path fill="currentColor" d="M234.667 362.667v-128h42.666v128zM256 213.333c11.782 0 21.333-9.551 21.333-21.333s-9.551-21.333-21.333-21.333s-21.333 9.551-21.333 21.333s9.551 21.333 21.333 21.333" /><path fill="currentColor" fill-rule="evenodd" d="M307.503 42.667H85.333v426.666h341.334V161.83zm-17.69 42.666L384 179.52v247.147H128V85.333z" clip-rule="evenodd" /></svg>`;

const svgCheck = `<svg viewBox="0 0 32 27" fill="currentColor" aria-hidden="true"><path d="M26.99 0L10.13 17.17l-5.44-5.54L0 16.41L10.4 27l4.65-4.73l.04.04L32 5.1z"/></svg>`;

const svgSwap = `<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M21.66 10.37a.6.6 0 0 0 .07-.19l.75-4a1 1 0 0 0-2-.36l-.37 2a9.22 9.22 0 0 0-16.58.84a1 1 0 0 0 .55 1.3a1 1 0 0 0 1.31-.55A7.08 7.08 0 0 1 12.07 5a7.17 7.17 0 0 1 6.24 3.58l-1.65-.27a1 1 0 1 0-.32 2l4.25.71h.16a.9.9 0 0 0 .34-.06a.3.3 0 0 0 .1-.06a.8.8 0 0 0 .2-.11l.08-.1a1 1 0 0 0 .14-.16a.6.6 0 0 0 .05-.16m-1.78 3.7a1 1 0 0 0-1.31.56A7.08 7.08 0 0 1 11.93 19a7.17 7.17 0 0 1-6.24-3.58l1.65.27h.16a1 1 0 0 0 .16-2L3.41 13a.9.9 0 0 0-.33 0H3a1.2 1.2 0 0 0-.32.14a1 1 0 0 0-.18.18l-.09.1a1 1 0 0 0-.07.19a.4.4 0 0 0-.07.17l-.75 4a1 1 0 0 0 .8 1.22h.18a1 1 0 0 0 1-.82l.37-2a9.22 9.22 0 0 0 16.58-.83a1 1 0 0 0-.57-1.28"/></svg>`;

const svgClearQueue = `<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 2C6.47 2 2 6.47 2 12s4.47 10 10 10 10-4.47 10-10S17.53 2 12 2zm4.3 12.3a1 1 0 0 1-1.41 1.41L12 13.41l-2.89 2.9a1 1 0 0 1-1.41-1.41L10.59 12 7.7 9.11a1 1 0 0 1 1.41-1.41L12 10.59l2.89-2.89a1 1 0 0 1 1.41 1.41L13.41 12l2.89 2.3z"/></svg>`;

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
  const title = escapeHtml(p.label || t("printer.printerN", { n: index + 1 }));
  const details = [
    isUsefulField(p.manufacturer) ? t("printer.brand", { name: p.manufacturer }) : null,
    isUsefulField(p.model) ? t("printer.model", { name: p.model }) : null,
    t("printer.address", { addr: p.host }),
  ].filter(Boolean);

  return `
    <div class="printer-item${p.configured ? " selected" : ""}">
      <div class="printer-info">
        <strong>${title}</strong>
        ${details.map((line) => `<span class="printer-meta">${escapeHtml(line)}</span>`).join("")}
      </div>
      <div class="printer-item-actions">
        <button type="button" class="btn-icon-action" data-action="test" data-host="${p.host}" data-port="${p.port}" title="${escapeHtml(t("printer.testPrinter"))}" aria-label="${escapeHtml(t("printer.testPrinter"))}">
          ${svgTestPrint}
        </button>
        <button type="button" class="btn-icon-action" data-action="select" data-host="${p.host}" data-port="${p.port}" title="${escapeHtml(t("printer.select"))}" aria-label="${escapeHtml(t("printer.select"))}">
          ${svgCheck}
        </button>
      </div>
    </div>`;
}

function renderPrinterListLoading(deep) {
  return `<div class="printer-list-loading"><span class="spinner" aria-hidden="true"></span><p class="empty">${escapeHtml(deep ? t("printer.deepScanning") : t("printer.discoveringList"))}</p></div>`;
}

function renderProfileDetails(profile) {
  const lines = [
    profile.manufacturer ? t("printer.brand", { name: profile.manufacturer }) : null,
    profile.model ? t("printer.model", { name: profile.model }) : null,
    profile.serial ? t("printer.serial", { value: profile.serial }) : null,
    profile.mac ? t("printer.mac", { value: profile.mac }) : null,
    profile.mac_vendor ? t("printer.networkChip", { name: profile.mac_vendor }) : null,
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
    print_contrast: prefs.print_contrast || 100,
    trim_trailing_blank: prefs.trim_blank !== "never",
  });
}

function printerProfileFromDiscovery(p) {
  return {
    host: p.host,
    port: p.port || 9100,
    label: p.label || p.name || "",
    name: p.name || "",
    manufacturer: p.manufacturer || "",
    model: p.model || "",
    serial: p.serial || "",
    hostname: p.hostname || "",
    mac: p.mac || "",
    mac_vendor: p.mac_vendor || "",
    description: p.description || "",
  };
}

function testPrintPayload(host, port, profile) {
  const p = profile || {};
  return {
    printer_host: host || p.host || "",
    printer_port: port || p.port || 9100,
    label: p.label || "",
    name: p.name || "",
    manufacturer: p.manufacturer || "",
    model: p.model || "",
    serial: p.serial || "",
    hostname: p.hostname || "",
    mac: p.mac || "",
    description: p.description || "",
  };
}

async function selectPrinter(host, port, printerData = {}) {
  const saved = await api.config.put({
    printer_host: host,
    printer_port: port,
    printer_mac: printerData.mac || "",
  });

  const profile = printerProfileFromDiscovery({
    ...printerData,
    host: saved.printer_host || host,
    port: saved.printer_port || port,
    mac: saved.printer_mac || printerData.mac || "",
    label: printerData.label || printerData.name || t("printer.connected"),
  });

  savePrinterProfile(profile);
  await loadConfigForm();
  refreshStatus(api);
  updateProfilePopover();
  showPopoverView(true);
  toast(t("toast.printerConnected"));
}

async function runResetPrinter(host, port) {
  const body = {};
  if (host) body.printer_host = host;
  if (port) body.printer_port = port;
  const data = await api.reset(body);
  toast(data.message || t("toast.resetSent"));
}

async function runTestPrint(host, port, profile) {
  const body = testPrintPayload(host, port, profile || loadPrinterProfile());
  const data = await api.test(body);
  toast(data.message || t("toast.testSent"));
}

function bindPrinterListActions(list, printers) {
  list.querySelectorAll("button[data-action]").forEach((b) => {
    b.addEventListener("click", async () => {
      const host = b.dataset.host;
      const port = parseInt(b.dataset.port, 10);
      try {
        if (b.dataset.action === "select") {
          const printer = printers.find((p) => p.host === host) || {};
          await selectPrinter(host, port, printer);
          return;
        }
        const printer = printers.find((p) => p.host === host) || {};
        await runTestPrint(host, port, printerProfileFromDiscovery(printer));
      } catch (err) {
        toast(err.message, false);
      }
    });
  });
}

async function runDiscover({ deep = false } = {}) {
  const btn = document.getElementById("btnDiscoverDeep");
  const list = document.getElementById("printerList");
  const defaultLabel = t("printer.deepScan");

  btn.disabled = true;
  list.innerHTML = renderPrinterListLoading(deep);

  try {
    const data = await api.discover(deep);
    if (!data.printers?.length) {
      list.innerHTML = deep
        ? `<p class="empty">${escapeHtml(t("printer.noneFound"))}</p>`
        : `<p class="empty">${escapeHtml(t("printer.emptyDeep"))}</p>`;
      return;
    }

    list.innerHTML = data.printers.map((p, index) => renderPrinterItem(p, index)).join("");
    saveDiscoveredPrinters(data.printers);
    bindPrinterListActions(list, data.printers);

    toast(t("printer.foundCount", { count: data.count }));
  } catch (err) {
    toast(err.message, false);
    list.innerHTML = `<p class="empty">${escapeHtml(t("printer.searchFailed"))}</p>`;
  } finally {
    btn.disabled = false;
    btn.textContent = defaultLabel;
  }
}

export function initPrinter() {
  document.getElementById("btnTestPrint").insertAdjacentHTML("afterbegin", svgTestPrint);
  document.getElementById("btnResetPrinter").insertAdjacentHTML("afterbegin", svgClearQueue);
  document.getElementById("btnChangePrinter").innerHTML = svgSwap;

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
    runDiscover({ deep: false });
  });

  document.getElementById("btnTestPrint").addEventListener("click", async () => {
    try {
      await runTestPrint();
    } catch (err) {
      toast(err.message, false);
    } finally {
      setTimeout(() => refreshStatus(api), 2000);
    }
  });

  document.getElementById("btnResetPrinter").addEventListener("click", async () => {
    const btn = document.getElementById("btnResetPrinter");
    btn.disabled = true;
    try {
      await runResetPrinter();
    } catch (err) {
      toast(err.message, false);
    } finally {
      btn.disabled = false;
      setTimeout(() => refreshStatus(api), 2000);
    }
  });

  document.getElementById("btnOpenAdvanced").addEventListener("click", () => {
    togglePopover(false);
    showPanel("advanced");
  });

  document.getElementById("btnDiscoverDeep").addEventListener("click", () => runDiscover({ deep: true }));

  ["prefPaper", "prefCutPage", "prefCutDoc", "prefTrimBlank", "prefContrast"].forEach((id) => {
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
    const profile = loadPrinterProfile();
    prefs.paper_width_mm = parseInt(value("cfgPaper"), 10);
    prefs.printable_width_mm = parseInt(value("cfgPrintable"), 10) || 0;
    savePrefs(prefs);

    await api.config.put({
      printer_host: value("cfgHost").trim(),
      printer_port: parseInt(value("cfgPort"), 10) || 9100,
      printer_mac: profile?.mac || "",
      paper_width_mm: prefs.paper_width_mm,
      printable_width_mm: prefs.printable_width_mm,
      print_contrast: prefs.print_contrast || 100,
      trim_trailing_blank: prefs.trim_blank !== "never",
    });

    if (profile) {
      savePrinterProfile({
        ...profile,
        host: value("cfgHost").trim(),
        port: parseInt(value("cfgPort"), 10) || 9100,
      });
    }

    applyPrefsToUI();
    toast(t("toast.configSaved"));
    refreshStatus(api);
  });

  onLangChange(() => {
    const popover = document.getElementById("printerPopover");
    if (!popover.hidden) {
      updateProfilePopover();
    }
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

  if (c.print_contrast > 0) {
    prefs.print_contrast = c.print_contrast;
    savePrefs(prefs);
  }
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
