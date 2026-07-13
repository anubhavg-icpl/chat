import { useState, type FormEvent } from "react";
import { api } from "../api/client";
import { TerminalFrame } from "../components/TerminalFrame";

export function InstantMessage() {
  const [from, setFrom] = useState("console");
  const [to, setTo] = useState("");
  const [text, setText] = useState("");
  const [log, setLog] = useState<string[]>([]);
  const [error, setError] = useState("");

  async function onSend(e: FormEvent) {
    e.preventDefault();
    setError("");
    try {
      await api.sendIM(from.trim(), to.trim(), text);
      setLog((prev) => [
        `[ok] ${from.trim()} -> ${to.trim()}: ${text}`,
        ...prev,
      ].slice(0, 20));
      setText("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "send failed");
    }
  }

  return (
    <div className="grid split">
      <div className="card holo-card">
        <div className="card-head">
          <span>compose</span>
        </div>
        <div className="card-body">
          <form
            onSubmit={onSend}
            style={{ display: "flex", flexDirection: "column", gap: 12 }}
          >
            <div className="field">
              <label>from</label>
              <input
                value={from}
                onChange={(e) => setFrom(e.target.value)}
                required
              />
            </div>
            <div className="field">
              <label>to</label>
              <input
                value={to}
                onChange={(e) => setTo(e.target.value)}
                placeholder="screen name"
                required
              />
            </div>
            <div className="field">
              <label>message</label>
              <textarea
                rows={5}
                value={text}
                onChange={(e) => setText(e.target.value)}
                placeholder="hello from the console"
                required
              />
            </div>
            <button className="btn" type="submit">
              send im
            </button>
          </form>
          {error && <div className="msg err" style={{ marginTop: 12 }}>{error}</div>}
        </div>
      </div>

      <TerminalFrame title="im-relay — outbound log">
        {log.length === 0 ? (
          <p className="logline">
            <span className="muted">awaiting traffic…</span>
          </p>
        ) : (
          log.map((line) => (
            <p className="logline" key={line}>
              <span className="ok">{line}</span>
            </p>
          ))
        )}
        <p className="logline">
          <span className="cmd">$ </span>
          <span className="cursor" />
        </p>
      </TerminalFrame>
    </div>
  );
}
