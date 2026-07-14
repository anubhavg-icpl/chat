import { useCallback, useEffect, useState, type FormEvent } from "react";
import { api, type ChatRoom } from "../api/client";
import { Button } from "@/components/ui/button";

export function ChatRooms() {
  const [publicRooms, setPublicRooms] = useState<ChatRoom[]>([]);
  const [privateRooms, setPrivateRooms] = useState<ChatRoom[]>([]);
  const [name, setName] = useState("");
  const [msg, setMsg] = useState<{ type: "ok" | "err"; text: string } | null>(
    null,
  );

  const load = useCallback(async () => {
    try {
      const [pub, priv] = await Promise.all([
        api.getPublicRooms(),
        api.getPrivateRooms(),
      ]);
      setPublicRooms(pub || []);
      setPrivateRooms(priv || []);
      setMsg(null);
    } catch (e) {
      setMsg({
        type: "err",
        text: e instanceof Error ? e.message : "failed to load rooms",
      });
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function onCreate(e: FormEvent) {
    e.preventDefault();
    try {
      await api.createPublicRoom(name.trim());
      setName("");
      setMsg({ type: "ok", text: `created room ${name.trim()}` });
      await load();
    } catch (err) {
      setMsg({
        type: "err",
        text: err instanceof Error ? err.message : "create failed",
      });
    }
  }

  async function onDelete(roomName: string) {
    if (!confirm(`delete room ${roomName}?`)) return;
    try {
      await api.deletePublicRooms([roomName]);
      setMsg({ type: "ok", text: `deleted ${roomName}` });
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
          <span>create public room</span>
          <Button variant="outline" type="button" onClick={() => void load()}>
            refresh
          </Button>
        </div>
        <div className="card-body">
          <form className="row" onSubmit={onCreate}>
            <div className="field">
              <label>room name</label>
              <input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Office Hijinks"
                required
              />
            </div>
            <Button type="submit">create</Button>
          </form>
          {msg && <div className={`msg ${msg.type}`} style={{ marginTop: 12 }}>{msg.text}</div>}
        </div>
      </div>

      <div className="grid split">
        <RoomTable
          title={`public · ${publicRooms.length}`}
          rooms={publicRooms}
          onDelete={onDelete}
          canDelete
        />
        <RoomTable
          title={`private · ${privateRooms.length}`}
          rooms={privateRooms}
        />
      </div>
    </>
  );
}

function RoomTable({
  title,
  rooms,
  onDelete,
  canDelete,
}: {
  title: string;
  rooms: ChatRoom[];
  onDelete?: (name: string) => void;
  canDelete?: boolean;
}) {
  return (
    <div className="card">
      <div className="card-head">
        <span>{title}</span>
      </div>
      <div className="table-wrap">
        {rooms.length === 0 ? (
          <div className="empty">no rooms</div>
        ) : (
          <table>
            <thead>
              <tr>
                <th>name</th>
                <th>participants</th>
                <th>created</th>
                {canDelete && <th>actions</th>}
              </tr>
            </thead>
            <tbody>
              {rooms.map((r) => (
                <tr key={r.name}>
                  <td>{r.name}</td>
                  <td>{r.participants?.length || 0}</td>
                  <td>
                    {r.create_time
                      ? new Date(r.create_time).toLocaleString()
                      : "—"}
                  </td>
                  {canDelete && (
                    <td>
                      <Button
                        variant="destructive"
                        type="button"
                        onClick={() => onDelete?.(r.name)}
                      >
                        delete
                      </Button>
                    </td>
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
