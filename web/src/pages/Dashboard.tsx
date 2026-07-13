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
        setError(e instanceof Error ? e.message : "failed to load dashboard");
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
        <div className="card holo-card stat">
          <div className="label">accounts</div>
          <div className="value">{loading ? "…" : users.length}</div>
          <div className="hint">
            {aimUsers} aim · {icqUsers} icq
          </div>
        </div>
        <div className="card holo-card stat">
          <div className="label">online</div>
          <div className="value">{loading ? "…" : online}</div>
          <div className="hint">active sessions</div>
        </div>
        <div className="card holo-card stat">
          <div className="label">chat rooms</div>
          <div className="value">{loading ? "…" : rooms}</div>
          <div className="hint">public + private</div>
        </div>
        <div className="card holo-card stat">
          <div className="label">webapi keys</div>
          <div className="value">{loading ? "…" : keys.length}</div>
          <div className="hint">{categories} dir categories</div>
        </div>
      </div>

      <div className="grid split">
        <TerminalFrame title="oscar-console — boot + version">
          <p className="logline">
            <span className="cmd">$ curl /api/version</span>
          </p>
          <p className="logline">
            <span className="ok">
              version={version?.version || "dev"} commit=
              {version?.commit || "none"} date={version?.date || "unknown"}
            </span>
          </p>
          <p className="logline">
            <span className="muted">
              ports: 5190 oscar · 5193 ssl · 8080 mgmt · 9898 toc · 1088 kerberos
              · 3000 console
            </span>
          </p>
          {error ? (
            <p className="logline">
              <span className="err">✗ {error}</span>
            </p>
          ) : (
            <p className="logline">
              <span className="cmd">$ </span>
              <span className="cursor" />
            </p>
          )}
        </TerminalFrame>

        <div className="card holo-card">
          <div className="card-head">
            <span>
              api probe · {okCount}/{probes.length} ok
            </span>
          </div>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>method</th>
                  <th>path</th>
                  <th>status</th>
                  <th>result</th>
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
                        {p.ok ? "ok" : p.detail || "fail"}
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
