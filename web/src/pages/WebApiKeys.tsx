import { useCallback, useEffect, useState, type FormEvent } from "react";
import { api, type WebAPIKey } from "../api/client";

export function WebApiKeys() {
  const [keys, setKeys] = useState<WebAPIKey[]>([]);
  const [appName, setAppName] = useState("console-app");
  const [origins, setOrigins] = useState("http://localhost:3000");
  const [rateLimit, setRateLimit] = useState(120);
  const [createdSecret, setCreatedSecret] = useState("");
  const [msg, setMsg] = useState<{ type: "ok" | "err"; text: string } | null>(
    null,
  );

  const load = useCallback(async () => {
    try {
      const data = await api.getWebApiKeys();
      setKeys(data || []);
      setMsg(null);
    } catch (e) {
      setMsg({
        type: "err",
        text: e instanceof Error ? e.message : "load keys failed",
      });
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function onCreate(e: FormEvent) {
    e.preventDefault();
    try {
      const created = await api.createWebApiKey({
        app_name: appName.trim(),
        allowed_origins: origins
          .split(",")
          .map((s) => s.trim())
          .filter(Boolean),
        rate_limit: rateLimit,
      });
      setCreatedSecret(created.dev_key || "");
      setMsg({
        type: "ok",
        text: `created key ${created.dev_id} — secret shown once below`,
      });
      await load();
    } catch (err) {
      setMsg({
        type: "err",
        text: err instanceof Error ? err.message : "create failed",
      });
    }
  }

  async function onToggle(key: WebAPIKey) {
    try {
      await api.updateWebApiKey(key.dev_id, { is_active: !key.is_active });
      setMsg({ type: "ok", text: `is_active → ${!key.is_active}` });
      await load();
    } catch (err) {
      setMsg({
        type: "err",
        text: err instanceof Error ? err.message : "update failed",
      });
    }
  }

  async function onDelete(id: string) {
    if (!confirm(`delete webapi key ${id}?`)) return;
    try {
      await api.deleteWebApiKey(id);
      setMsg({ type: "ok", text: `deleted ${id}` });
      await load();
    } catch (err) {
      setMsg({
        type: "err",
        text: err instanceof Error ? err.message : "delete failed",
      });
    }
  }

  return (
    <>
      <div className="card">
        <div className="card-head">
          <span>POST /admin/webapi/keys</span>
          <button className="btn ghost" type="button" onClick={() => void load()}>
            refresh
          </button>
        </div>
        <div className="card-body">
          <form className="row" onSubmit={onCreate}>
            <div className="field">
              <label>app name</label>
              <input
                value={appName}
                onChange={(e) => setAppName(e.target.value)}
                required
              />
            </div>
            <div className="field">
              <label>allowed origins (csv)</label>
              <input
                value={origins}
                onChange={(e) => setOrigins(e.target.value)}
              />
            </div>
            <div className="field">
              <label>rate limit / min</label>
              <input
                type="number"
                value={rateLimit}
                onChange={(e) => setRateLimit(Number(e.target.value))}
              />
            </div>
            <button className="btn" type="submit">
              create key
            </button>
          </form>
          {msg && (
            <div className={`msg ${msg.type}`} style={{ marginTop: 12 }}>
              {msg.text}
            </div>
          )}
          {createdSecret && (
            <div className="msg ok" style={{ marginTop: 12 }}>
              dev_key (copy now): {createdSecret}
            </div>
          )}
        </div>
      </div>

      <div className="card">
        <div className="card-head">
          <span>GET /admin/webapi/keys · {keys.length}</span>
        </div>
        <div className="table-wrap">
          {keys.length === 0 ? (
            <div className="empty">no keys</div>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>dev_id</th>
                  <th>app</th>
                  <th>active</th>
                  <th>rate</th>
                  <th>origins</th>
                  <th>actions</th>
                </tr>
              </thead>
              <tbody>
                {keys.map((k) => (
                  <tr key={k.dev_id}>
                    <td>{k.dev_id}</td>
                    <td>{k.app_name}</td>
                    <td>
                      <span className={`badge ${k.is_active ? "online" : "away"}`}>
                        {k.is_active ? "active" : "disabled"}
                      </span>
                    </td>
                    <td>{k.rate_limit ?? "—"}</td>
                    <td style={{ maxWidth: 220, whiteSpace: "normal" }}>
                      {(k.allowed_origins || []).join(", ") || "*"}
                    </td>
                    <td>
                      <div className="actions">
                        <button
                          className="btn ghost"
                          type="button"
                          onClick={() => void onToggle(k)}
                        >
                          toggle
                        </button>
                        <button
                          className="btn danger"
                          type="button"
                          onClick={() => void onDelete(k.dev_id)}
                        >
                          delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </>
  );
}
