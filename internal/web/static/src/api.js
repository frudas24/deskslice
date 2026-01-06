export async function login(password) {
  const res = await fetch("/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ password }),
  });
  if (!res.ok) {
    throw new Error("login failed");
  }
  return res.json().catch(() => ({}));
}

export async function logout() {
  const res = await fetch("/logout", { method: "POST" });
  if (!res.ok) {
    throw new Error("logout failed");
  }
  return res.json().catch(() => ({}));
}

export async function getState() {
  const res = await fetch("/api/state");
  if (!res.ok) {
    throw new Error("state fetch failed");
  }
  return res.json();
}

export async function getMonitors() {
  const res = await fetch("/api/monitors");
  if (!res.ok) {
    throw new Error("monitor fetch failed");
  }
  return res.json();
}
