import { useEffect, useState } from "react";
import {
  api,
  type ApiProbe,
  type SessionList,
  type User,
  type VersionInfo,
  type WebAPIKey,
} from "../api/client";
import { TerminalFrame } from "../components/TerminalFrame";

export function Dashboard() {
  const [users, setUsers] = useState<User[]>([]);
  const [sessions, setSessions] = useState<SessionList>({
    count: 0,
    sessions: [],
  });
  const [rooms, setRooms] = useState(0);
  const [keys, setKeys] = useState<WebAPIKey[]>([]);
  const [categories, setCategories] = useState(0);
  const [version, setVersion] = useState<VersionInfo | null>(null);
  const [probes, setProbes] = useState<ApiProbe[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const [u, s, publicRooms, privateRooms, v, k, cats, p] =
          await Promise.all([
            api.getUsers(),
            api.getSessions(),
            api.getPublicRooms().catch(() => []),
            api.getPrivateRooms().catch(() => []),
            api.getVersion().catch(() => null),
            api.getWebApiKeys().catch(() => []),
            api.getDirectoryCategories().catch(() => []),
            api.probeApis(),
          ]);
        if (!alive) return;
        setUsers(u || []);
        setSessions(s || { count: 0, sessions: [] });
        setRooms((publicRooms?.length || 0) + (privateRooms?.length || 0));
        setVersion(v);
        setKeys(k || []);
        setCategories(cats?.length || 0);
        setProbes(p);
        setError("");
      } catch (e) {
        if (!alive) return;
        setError(e instanceof Error ? e.message : "Failed to load dashboard");
      } finally {
        if (alive) setLoading(false);
      }
    })();
    return () => {
      alive = false;
    };
  }, []);

  const online = sessions.count || sessions.sessions?.length || 0;
  const aimUsers = users.filter((u) => !u.is_icq).length;
  const icqUsers = users.filter((u) => u.is_icq).length;
  const okCount = probes.filter((p) => p.ok).length;

  return (
    <>
      <div className="grid stats">
        <div className="card stat">
          <div className="label">Accounts</div>
          <div className="value">{loading ? "…" : users.length}</div>
          <div className="hint">
            {aimUsers} AIM · {icqUsers} ICQ
          </div>
        </div>
        <div className="card stat">
          <div className="label">Online</div>
          <div className="value">{loading ? "…" : online}</div>
          <div className="hint">Active sessions</div>
        </div>
        <div className="card stat">
          <div className="label">Chat rooms</div>
          <div className="value">{loading ? "…" : rooms}</div>
          <div className="hint">Public + private</div>
        </div>
        <div className="card stat">
          <div className="label">Web API keys</div>
          <div className="value">{loading ? "…" : keys.length}</div>
          <div className="hint">{categories} directory categories</div>
        </div>
      </div>

      <div className="grid split">
        <TerminalFrame title="server status">
          <p className="logline">
            <span className="cmd">GET /api/version</span>
          </p>
          <p className="logline">
            <span className="ok">
              version={version?.version || "dev"} commit=
              {version?.commit || "none"} date={version?.date || "unknown"}
            </span>
          </p>
          <p className="logline">
            <span className="muted">
              listeners: 5190 OSCAR · 5193 SSL · 8080 Mgmt · 9898 TOC · 1088
              Kerberos · 3000 Console
            </span>
          </p>
          {error ? (
            <p className="logline">
              <span className="err">error: {error}</span>
            </p>
          ) : (
            <p className="logline">
              <span className="cmd">status: ready </span>
              <span className="cursor" />
            </p>
          )}
        </TerminalFrame>

        <div className="card">
          <div className="card-head">
            <span>
              API health · {okCount}/{probes.length} ok
            </span>
          </div>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Method</th>
                  <th>Path</th>
                  <th>Status</th>
                  <th>Result</th>
                </tr>
              </thead>
              <tbody>
                {probes.map((p) => (
                  <tr key={`${p.method}-${p.path}`}>
                    <td>{p.method}</td>
                    <td>{p.path}</td>
                    <td>{p.status || "—"}</td>
                    <td>
                      <span className={`badge ${p.ok ? "online" : "away"}`}>
                        {p.ok ? "OK" : p.detail || "fail"}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </>
  );
}
