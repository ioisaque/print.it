import { t } from "./i18n.js";

const ORIGIN = window.location.origin;
const PRINTIT = `${ORIGIN}/printit`;
const PRINT_TIMEOUT_MS = 120000;

async function request(url, options = {}, timeoutMs = 0) {
  const controller = timeoutMs > 0 ? new AbortController() : null;
  const timer = controller ? setTimeout(() => controller.abort(), timeoutMs) : null;

  let response;
  try {
    response = await fetch(url, {
      ...options,
      ...(controller ? { signal: controller.signal } : {}),
    });
  } catch (err) {
    if (err.name === "AbortError") {
      throw new Error(t("toast.printTimeout"));
    }
    throw err;
  } finally {
    if (timer) clearTimeout(timer);
  }

  let data = null;
  const contentType = response.headers.get("content-type") || "";
  if (contentType.includes("application/json")) {
    data = await response.json();
  }
  if (!response.ok) {
    const message = data?.error || data?.message || response.statusText;
    throw new Error(message);
  }
  return data;
}

export const api = {
  status() {
    return request(`${PRINTIT}/status`);
  },

  config: {
    put(body) {
      return request(`${PRINTIT}/config`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
    },
  },

  discover(deep = false) {
    const query = deep ? "?deep=true" : "";
    return request(`${PRINTIT}/discover${query}`);
  },

  test(body = {}) {
    return request(`${PRINTIT}/test`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  },

  reset(body = {}) {
    return request(`${PRINTIT}/reset`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  },

  preview(formData) {
    return fetch(`${PRINTIT}/preview`, {
      method: "POST",
      body: formData,
    });
  },

  text: {
    send(body) {
      return request(`${PRINTIT}/text`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }, PRINT_TIMEOUT_MS);
    },
  },

  pdf: {
    send(formData) {
      return request(`${PRINTIT}/pdf`, {
        method: "POST",
        body: formData,
      }, PRINT_TIMEOUT_MS);
    },
  },

  image: {
    send(formData) {
      return request(`${PRINTIT}/image`, {
        method: "POST",
        body: formData,
      }, PRINT_TIMEOUT_MS);
    },
  },

  barcode: {
    send(body) {
      return request(`${PRINTIT}/barcode`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }, PRINT_TIMEOUT_MS);
    },
  },

  qrcode: {
    send(body) {
      return request(`${PRINTIT}/qrcode`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }, PRINT_TIMEOUT_MS);
    },
  },
};
