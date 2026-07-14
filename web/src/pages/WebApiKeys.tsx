import { useCallback, useEffect, useState, type FormEvent } from "react";
import { api, type WebAPIKey } from "../api/client";
import { useConfirm } from "@/components/ConfirmProvider";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";

export function WebApiKeys() {
  const [keys, setKeys] = useState<WebAPIKey[]>([]);
  const [appName, setAppName] = useState("console-app");
  const [origins, setOrigins] = useState("http://localhost:3000");
  const [rateLimit, setRateLimit] = useState(120);
  const [createdSecret, setCreatedSecret] = useState("");
  const confirm = useConfirm();

  const load = useCallback(async () => {
    try {
      const data = await api.getWebApiKeys();
      setKeys(data || []);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "load keys failed");
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
      toast.success(`created key ${created.dev_id} — secret shown below`);
      await load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "create failed");
    }
  }

  async function onToggle(key: WebAPIKey) {
    try {
      await api.updateWebApiKey(key.dev_id, { is_active: !key.is_active });
      toast.success(`is_active → ${!key.is_active}`);
      await load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "update failed");
    }
  }

  async function onDelete(id: string) {
    if (
      !(await confirm({
        title: "Delete API key",
        description: `Permanently delete key ${id}?`,
        action: "Delete",
        destructive: true,
      }))
    )
      return;
    try {
      await api.deleteWebApiKey(id);
      toast.success(`deleted ${id}`);
      await load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "delete failed");
    }
  }

  return (
    <>
      <div className="card">
        <div className="card-head">
          <span>POST /admin/webapi/keys</span>
          <Button variant="outline" type="button" onClick={() => void load()}>
            refresh
          </Button>
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
            <Button type="submit">create key</Button>
          </form>
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
                        <Button
                          variant="outline"
                          type="button"
                          onClick={() => void onToggle(k)}
                        >
                          toggle
                        </Button>
                        <Button
                          variant="destructive"
                          type="button"
                          onClick={() => void onDelete(k.dev_id)}
                        >
                          delete
                        </Button>
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
