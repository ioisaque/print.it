const LANG_KEY = "printit.lang";
const SUPPORTED = ["pt-br", "en"];

const FLAG_SRC = {
  "pt-br": "assets/imgs/br-flag.svg",
  en: "assets/imgs/us-flag.svg",
};

let strings = {};
let currentLang = "pt-br";
let defaultLang = "pt-br";
const langChangeListeners = [];

function normalizeLang(lang) {
  const value = String(lang || "").toLowerCase();
  if (value === "en" || value.startsWith("en-")) return "en";
  if (value === "pt" || value === "pt-br" || value.startsWith("pt-")) return "pt-br";
  return SUPPORTED.includes(value) ? value : "";
}

export function t(key, vars = {}) {
  let text = strings[key] ?? key;
  for (const [name, value] of Object.entries(vars)) {
    text = text.replaceAll(`{${name}}`, String(value));
  }
  return text;
}

export function getLang() {
  return currentLang;
}

export function onLangChange(fn) {
  langChangeListeners.push(fn);
}

function notifyLangChange() {
  for (const fn of langChangeListeners) {
    fn(currentLang);
  }
}

async function fetchDefaultLang() {
  try {
    const res = await fetch("assets/build.json", { cache: "no-store" });
    if (res.ok) {
      const data = await res.json();
      const lang = normalizeLang(data.language);
      if (lang) return lang;
    }
  } catch {
    /* ignore */
  }

  try {
    const res = await fetch(`${window.location.origin}/printit/health`, { cache: "no-store" });
    if (res.ok) {
      const data = await res.json();
      const lang = normalizeLang(data.default_language);
      if (lang) return lang;
    }
  } catch {
    /* ignore */
  }

  return "pt-br";
}

async function loadStrings(lang) {
  const res = await fetch(`assets/lang/${lang}/strings.json`, { cache: "no-store" });
  if (!res.ok) throw new Error(`lang:${lang}`);
  strings = await res.json();
  currentLang = lang;
}

function htmlLang(lang) {
  return lang === "en" ? "en" : "pt-BR";
}

function applyMeta() {
  const meta = document.querySelector('meta[name="description"]');
  if (meta) meta.setAttribute("content", t("meta.description"));
  document.documentElement.lang = htmlLang(currentLang);
}

function applyI18nAttributes() {
  document.querySelectorAll("[data-i18n]").forEach((el) => {
    const key = el.getAttribute("data-i18n");
    if (key) el.textContent = t(key);
  });

  document.querySelectorAll("[data-i18n-placeholder]").forEach((el) => {
    const key = el.getAttribute("data-i18n-placeholder");
    if (key) el.setAttribute("placeholder", t(key));
  });

  document.querySelectorAll("[data-i18n-title]").forEach((el) => {
    const key = el.getAttribute("data-i18n-title");
    if (key) el.setAttribute("title", t(key));
  });

  document.querySelectorAll("[data-i18n-aria-label]").forEach((el) => {
    const key = el.getAttribute("data-i18n-aria-label");
    if (key) el.setAttribute("aria-label", t(key));
  });

  document.querySelectorAll("select option[data-i18n]").forEach((el) => {
    const key = el.getAttribute("data-i18n");
    if (key) el.textContent = t(key);
  });

  const textPreview = document.getElementById("textPreview");
  if (textPreview) {
    textPreview.setAttribute("data-placeholder", t("text.placeholder"));
  }
}

export function applyI18n() {
  applyMeta();
  applyI18nAttributes();
}

function updateLangSwitcher() {
  const icon = document.getElementById("langFlagIcon");
  if (icon) {
    icon.src = FLAG_SRC[currentLang] || FLAG_SRC["pt-br"];
    icon.alt = currentLang === "en" ? t("lang.en") : t("lang.ptBr");
  }

  document.querySelectorAll(".lang-option").forEach((btn) => {
    const lang = btn.getAttribute("data-lang");
    btn.classList.toggle("active", lang === currentLang);
    btn.setAttribute("aria-pressed", lang === currentLang ? "true" : "false");
  });
}

function toggleLangPopover(open) {
  const popover = document.getElementById("langPopover");
  const btn = document.getElementById("langSwitchBtn");
  if (!popover || !btn) return;

  const shouldOpen = open ?? popover.hidden;
  popover.hidden = !shouldOpen;
  btn.setAttribute("aria-expanded", shouldOpen ? "true" : "false");
}

async function resolveLang() {
  try {
    const saved = normalizeLang(localStorage.getItem(LANG_KEY));
    if (saved) return saved;
  } catch {
    /* ignore */
  }

  defaultLang = await fetchDefaultLang();
  return defaultLang;
}

export async function setLang(lang) {
  const next = normalizeLang(lang);
  if (!next || !SUPPORTED.includes(next)) return;

  await loadStrings(next);
  localStorage.setItem(LANG_KEY, next);
  applyI18n();
  updateLangSwitcher();
  notifyLangChange();
}

function initLangSwitcher() {
  const btn = document.getElementById("langSwitchBtn");
  const popover = document.getElementById("langPopover");
  if (!btn || !popover) return;

  btn.addEventListener("click", (e) => {
    e.stopPropagation();
    toggleLangPopover(popover.hidden);
  });

  document.addEventListener("click", (e) => {
    const wrap = document.querySelector(".lang-switch-wrap");
    if (!wrap?.contains(e.target)) {
      toggleLangPopover(false);
    }
  });

  popover.addEventListener("click", (e) => e.stopPropagation());

  popover.querySelectorAll(".lang-option").forEach((option) => {
    option.addEventListener("click", async () => {
      const lang = option.getAttribute("data-lang");
      if (!lang || lang === currentLang) {
        toggleLangPopover(false);
        return;
      }
      try {
        await setLang(lang);
      } catch {
        /* ignore */
      }
      toggleLangPopover(false);
    });
  });
}

export async function initI18n() {
  const lang = await resolveLang();
  await loadStrings(lang);
  applyI18n();
  updateLangSwitcher();
  initLangSwitcher();
}
