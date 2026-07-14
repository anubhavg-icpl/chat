import { useCallback, useEffect, useState, type FormEvent } from "react";
import {
  api,
  type FeedbagGroup,
  type User,
  type UserAccount,
} from "../api/client";
import { Skeleton } from "@/components/Skeleton";
import { useConfirm } from "@/components/ConfirmProvider";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";

export function Users() {
  const [users, setUsers] = useState<User[]>([]);
  const [screenName, setScreenName] = useState("");
  const [password, setPassword] = useState("");
  const [selected, setSelected] = useState<string | null>(null);
  const [account, setAccount] = useState<UserAccount | null>(null);
  const [linked, setLinked] = useState<string[]>([]);
  const [feedbag, setFeedbag] = useState<FeedbagGroup[]>([]);
  const [linkName, setLinkName] = useState("");
  const [groupName, setGroupName] = useState("Buddies");
  const [buddyName, setBuddyName] = useState("");
  const [buddyGroupId, setBuddyGroupId] = useState<number | "">("");
  const confirm = useConfirm();
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.getUsers();
      setUsers(data || []);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "failed to load users");
    } finally {
      setLoading(false);
    }
  }, []);

  const loadDetail = useCallback(async (name: string) => {
    try {
      const [acc, links, fb] = await Promise.all([
        api.getAccount(name),
        api.getLinkedAccounts(name).catch(() => ({ linked_accounts: [] })),
        api.getFeedbag(name).catch(() => [] as FeedbagGroup[]),
      ]);
      setAccount(acc);
      setLinked(links.linked_accounts || []);
      setFeedbag(fb || []);
      if (fb?.[0]) setBuddyGroupId(fb[0].group_id);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "failed to load account");
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (selected) void loadDetail(selected);
  }, [selected, loadDetail]);

  async function onCreate(e: FormEvent) {
    e.preventDefault();
    try {
      const sn = screenName.trim();
      await api.createUser(sn, password);
      toast.success(`created ${sn}`);
      setScreenName("");
      setPassword("");
      await load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "create failed");
    }
  }

  async function onDelete(name: string) {
    if (
      !(await confirm({
        title: "Delete user",
        description: `Permanently delete ${name}?`,
        action: "Delete",
        destructive: true,
      }))
    )
      return;
    try {
      await api.deleteUser(name);
      if (selected === name) {
        setSelected(null);
        setAccount(null);
      }
      toast.success(`deleted ${name}`);
      await load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "delete failed");
    }
  }

  async function onReset(name: string) {
    const next = prompt(`new password for ${name}`);
    if (!next) return;
    try {
      await api.setPassword(name, next);
      toast.success(`password updated for ${name}`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "password update failed");
    }
  }

  async function onSuspend(status: string | null) {
    if (!selected) return;
    try {
      await api.patchAccount(selected, { suspended_status: status });
      toast.success(`suspend status → ${status || "cleared"}`);
      await loadDetail(selected);
      await load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "patch failed");
    }
  }

  async function onToggleBot() {
    if (!selected || !account) return;
    try {
      await api.patchAccount(selected, { is_bot: !account.is_bot });
      toast.success(`is_bot → ${!account.is_bot}`);
      await loadDetail(selected);
      await load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "patch failed");
    }
  }

  async function onAddLink(e: FormEvent) {
    e.preventDefault();
    if (!selected || !linkName.trim()) return;
    try {
      await api.addLinkedAccount(selected, linkName.trim());
      setLinkName("");
      toast.success("linked account added");
      await loadDetail(selected);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "link failed");
    }
  }

  async function onRemoveLink(name: string) {
    if (!selected) return;
    try {
      await api.removeLinkedAccount(selected, name);
      toast.success(`unlinked ${name}`);
      await loadDetail(selected);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "unlink failed");
    }
  }

  async function onCreateGroup(e: FormEvent) {
    e.preventDefault();
    if (!selected || !groupName.trim()) return;
    try {
      const g = await api.createFeedbagGroup(selected, groupName.trim());
      setBuddyGroupId(g.group_id);
      toast.success(`group ${g.group_name} (#${g.group_id})`);
      await loadDetail(selected);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "group create failed");
    }
  }

  async function onAddBuddy(e: FormEvent) {
    e.preventDefault();
    if (!selected || buddyGroupId === "" || !buddyName.trim()) return;
    try {
      const buddy = buddyName.trim();
      await api.addBuddy(selected, Number(buddyGroupId), buddy);
      setBuddyName("");
      toast.success(`buddy ${buddy} added`);
      await loadDetail(selected);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "add buddy failed");
    }
  }

  async function onRemoveBuddy(groupId: number, buddy: string) {
    if (!selected) return;
    try {
      await api.removeBuddy(selected, groupId, buddy);
      toast.success(`removed ${buddy}`);
      await loadDetail(selected);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "remove buddy failed");
    }
  }

  return (
    <>
      <div className="card">
        <div className="card-head">
          <span>create user · POST /user</span>
          <Button variant="outline" type="button" onClick={() => void load()}>
            refresh
          </Button>
        </div>
        <div className="card-body">
          <form className="row" onSubmit={onCreate}>
            <div className="field">
              <label>screen name / uin</label>
              <input
                value={screenName}
                onChange={(e) => setScreenName(e.target.value)}
                placeholder="alice or 100003"
                required
              />
            </div>
            <div className="field">
              <label>password</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>
            <Button type="submit">create</Button>
          </form>
        </div>
      </div>

      <div className="grid split">
        <div className="card">
          <div className="card-head">
            <span>GET /user · {users.length}</span>
          </div>
          <div className="table-wrap">
            {loading ? (
              <table>
                <thead>
                  <tr>
                    <th>screen name</th>
                    <th>type</th>
                    <th>status</th>
                    <th>actions</th>
                  </tr>
                </thead>
                <tbody>
                  {Array.from({ length: 6 }).map((_, i) => (
                    <tr key={i}>
                      <td>
                        <Skeleton className="h-4 w-28" />
                      </td>
                      <td>
                        <Skeleton className="h-5 w-12" />
                      </td>
                      <td>
                        <Skeleton className="h-4 w-16" />
                      </td>
                      <td>
                        <Skeleton className="h-7 w-28" />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            ) : users.length === 0 ? (
              <div className="empty">no users</div>
            ) : (
              <table>
                <thead>
                  <tr>
                    <th>screen name</th>
                    <th>type</th>
                    <th>status</th>
                    <th>actions</th>
                  </tr>
                </thead>
                <tbody>
                  {users.map((u) => (
                    <tr
                      key={u.id || u.screen_name}
                      style={{
                        cursor: "pointer",
                        background:
                          selected === u.screen_name
                            ? "rgba(24,163,210,0.08)"
                            : undefined,
                      }}
                      onClick={() => setSelected(u.screen_name)}
                    >
                      <td>{u.screen_name}</td>
                      <td>
                        <span className={`badge ${u.is_icq ? "icq" : "aim"}`}>
                          {u.is_icq ? "icq" : "aim"}
                        </span>
                      </td>
                      <td>{u.suspended_status || "active"}</td>
                      <td>
                        <div className="actions">
                          <Button
                            variant="outline"
                            type="button"
                            onClick={(e) => {
                              e.stopPropagation();
                              void onReset(u.screen_name);
                            }}
                          >
                            passwd
                          </Button>
                          <Button
                            variant="destructive"
                            type="button"
                            onClick={(e) => {
                              e.stopPropagation();
                              void onDelete(u.screen_name);
                            }}
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

        <div className="card">
          <div className="card-head">
            <span>
              {selected
                ? `detail · ${selected}`
                : "select a user for account / feedbag / links"}
            </span>
          </div>
          <div className="card-body">
            {!selected || !account ? (
              <div className="empty">
                click a row to load GET /user/{"{sn}"}/account + feedbag +
                linked-account
              </div>
            ) : (
              <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
                <div className="row">
                  <img
                    src={api.getUserIconUrl(selected)}
                    alt=""
                    width={48}
                    height={48}
                    style={{
                      border: "1px solid var(--border)",
                      borderRadius: 4,
                      background: "#0a0a0a",
                      objectFit: "cover",
                    }}
                    onError={(e) => {
                      (e.target as HTMLImageElement).style.display = "none";
                    }}
                  />
                  <div>
                    <div style={{ color: "var(--ok)" }}>{account.screen_name}</div>
                    <div style={{ color: "var(--muted)", fontSize: 11 }}>
                      confirmed={String(account.confirmed)} · bot=
                      {String(!!account.is_bot)} · email=
                      {account.email_address || "—"}
                    </div>
                  </div>
                </div>

                <div className="actions">
                  <Button variant="outline" type="button" onClick={onToggleBot}>
                    toggle bot
                  </Button>
                  <Button
                    variant="outline"
                    type="button"
                    onClick={() => void onSuspend("suspended")}
                  >
                    suspend
                  </Button>
                  <Button
                    variant="outline"
                    type="button"
                    onClick={() => void onSuspend(null)}
                  >
                    clear suspend
                  </Button>
                </div>

                {account.profile && (
                  <div className="msg">
                    profile: {account.profile.slice(0, 200)}
                  </div>
                )}

                <div>
                  <div className="card-head" style={{ margin: "0 -16px 12px" }}>
                    <span>linked accounts</span>
                  </div>
                  {linked.length === 0 ? (
                    <div className="empty">none</div>
                  ) : (
                    <ul style={{ margin: 0, paddingLeft: 18, color: "var(--dim)" }}>
                      {linked.map((l) => (
                        <li key={l}>
                          {l}{" "}
                          <Button
                            variant="destructive"
                            size="xs"
                            type="button"
                            onClick={() => void onRemoveLink(l)}
                          >
                            unlink
                          </Button>
                        </li>
                      ))}
                    </ul>
                  )}
                  <form className="row" onSubmit={onAddLink} style={{ marginTop: 10 }}>
                    <div className="field">
                      <label>link screen name</label>
                      <input
                        value={linkName}
                        onChange={(e) => setLinkName(e.target.value)}
                      />
                    </div>
                    <Button type="submit">link</Button>
                  </form>
                </div>

                <div>
                  <div className="card-head" style={{ margin: "0 -16px 12px" }}>
                    <span>feedbag / buddy list</span>
                  </div>
                  {feedbag.length === 0 ? (
                    <div className="empty">no feedbag yet — create a group</div>
                  ) : (
                    feedbag.map((g) => (
                      <div key={g.group_id} style={{ marginBottom: 10 }}>
                        <div style={{ color: "var(--ok)" }}>
                          #{g.group_id} {g.group_name}
                        </div>
                        {(g.buddies || []).length === 0 ? (
                          <div className="empty">no buddies</div>
                        ) : (
                          <ul
                            style={{
                              margin: "4px 0 0",
                              paddingLeft: 18,
                              color: "var(--dim)",
                            }}
                          >
                            {g.buddies.map((b) => (
                              <li key={`${g.group_id}-${b.name}`}>
                                {b.name}{" "}
                                <Button
                                  variant="destructive"
                                  size="xs"
                                  type="button"
                                  onClick={() =>
                                    void onRemoveBuddy(g.group_id, b.name)
                                  }
                                >
                                  rm
                                </Button>
                              </li>
                            ))}
                          </ul>
                        )}
                      </div>
                    ))
                  )}
                  <form className="row" onSubmit={onCreateGroup}>
                    <div className="field">
                      <label>new group</label>
                      <input
                        value={groupName}
                        onChange={(e) => setGroupName(e.target.value)}
                      />
                    </div>
                    <Button type="submit">add group</Button>
                  </form>
                  <form className="row" onSubmit={onAddBuddy} style={{ marginTop: 8 }}>
                    <div className="field">
                      <label>group id</label>
                      <input
                        value={buddyGroupId}
                        onChange={(e) =>
                          setBuddyGroupId(
                            e.target.value === "" ? "" : Number(e.target.value),
                          )
                        }
                      />
                    </div>
                    <div className="field">
                      <label>buddy</label>
                      <input
                        value={buddyName}
                        onChange={(e) => setBuddyName(e.target.value)}
                      />
                    </div>
                    <Button type="submit">add buddy</Button>
                  </form>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </>
  );
}
