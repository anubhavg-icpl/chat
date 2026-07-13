import { useCallback, useEffect, useState } from "react";
import { api, type Session } from "../api/client";

function formatDuration(seconds?: number) {
  const s = Math.max(0, Math.floor(seconds || 0));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const r = s % 60;
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${r}s`;
  return `${r}s`;
}

export function Sessions() {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.getSessions();
      setSessions(data?.sessions || []);
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "failed to load sessions");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
    const id = window.setInterval(() => void load(), 5000);
    return () => window.clearInterval(id);
  }, [load]);

  async function kick(name: string) {
    if (!confirm(`kick session ${name}?`)) return;
    try {
      await api.kickSession(name);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "kick failed");
    }
  }

  return (
    <div className="card holo-card">
      <div className="card-head">
        <span>sessions · auto-refresh 5s</span>
        <button className="btn ghost" type="button" onClick={() => void load()}>
          refresh
        </button>
      </div>
      {error && <div className="msg err" style={{ margin: 16 }}>{error}</div>}
      <div className="table-wrap">
        {loading && sessions.length === 0 ? (
          <div className="empty">scanning…</div>
        ) : sessions.length === 0 ? (
          <div className="empty">no active sessions</div>
        ) : (
          <table>
            <thead>
              <tr>
                <th>screen name</th>
                <th>type</th>
                <th>online</th>
                <th>idle</th>
                <th>state</th>
                <th>instances</th>
                <th>remote</th>
                <th>actions</th>
              </tr>
            </thead>
            <tbody>
              {sessions.map((s) => {
                const remote = s.instances?.[0];
                return (
                  <tr key={`${s.id}-${s.screen_name}`}>
                    <td>{s.screen_name}</td>
                    <td>
                      <span className={`badge ${s.is_icq ? "icq" : "aim"}`}>
                        {s.is_icq ? "icq" : "aim"}
                      </span>
                    </td>
                    <td>{formatDuration(s.online_seconds)}</td>
                    <td>{formatDuration(s.idle_seconds)}</td>
                    <td>
                      <span
                        className={`badge ${s.is_away ? "away" : "online"}`}
                      >
                        {s.is_invisible
                          ? "invisible"
                          : s.is_away
                            ? "away"
                            : "online"}
                      </span>
                    </td>
                    <td>{s.instance_count || s.instances?.length || 1}</td>
                    <td>
                      {remote?.remote_addr
                        ? `${remote.remote_addr}:${remote.remote_port || ""}`
                        : "—"}
                    </td>
                    <td>
                      <button
                        className="btn danger"
                        type="button"
                        onClick={() => void kick(s.screen_name)}
                      >
                        kick
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
