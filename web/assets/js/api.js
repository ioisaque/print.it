const ORIGIN = window.location.origin;
const PRINTIT = `${ORIGIN}/printit`;

async function request(url, options = {}) {
  const response = await fetch(url, options);
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

  discover() {
    return request(`${PRINTIT}/discover`);
  },

  test() {
    return request(`${PRINTIT}/test`, { method: "POST" });
  },

  text: {
    send(body) {
      return request(`${PRINTIT}/text`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
    },
  },

  pdf: {
    send(formData) {
      return request(`${PRINTIT}/pdf`, {
        method: "POST",
        body: formData,
      });
    },
  },

  image: {
    send(formData) {
      return request(`${PRINTIT}/image`, {
        method: "POST",
        body: formData,
      });
    },
  },

  barcode: {
    send(body) {
      return request(`${PRINTIT}/barcode`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
    },
  },

  qrcode: {
    send(body) {
      return request(`${PRINTIT}/qrcode`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
    },
  },
};
