export function toast(message, ok = true) {
  const el = document.createElement("div");
  el.className = "toast " + (ok ? "ok" : "err");
  el.textContent = message;
  document.body.appendChild(el);
  setTimeout(() => el.remove(), 4000);
}

export function initTabs() {
  const tabs = document.querySelectorAll("#tabs button");
  tabs.forEach((btn) => {
    btn.addEventListener("click", () => {
      tabs.forEach((b) => b.classList.remove("active"));
      document.querySelectorAll(".panel").forEach((p) => p.classList.remove("active"));
      btn.classList.add("active");
      document.getElementById("panel-" + btn.dataset.tab).classList.add("active");
    });
  });
}

export async function refreshStatus(api) {
  const dot = document.getElementById("statusDot");
  const text = document.getElementById("statusText");
  try {
    const data = await api.status();
    dot.className = "dot ok";
    text.textContent = data.printer || "Conectado";
  } catch {
    dot.className = "dot err";
    text.textContent = "Serviço offline";
  }
}

export function bindFormSubmit(buttonId, handler) {
  document.getElementById(buttonId).addEventListener("click", async () => {
    const btn = document.getElementById(buttonId);
    btn.disabled = true;
    try {
      await handler();
    } catch (err) {
      toast(err.message || "Erro", false);
    } finally {
      btn.disabled = false;
    }
  });
}

export function checkbox(id) {
  return document.getElementById(id).checked;
}

export function value(id) {
  return document.getElementById(id).value;
}
