import { useCallback, useEffect, useState, type FormEvent } from "react";
import {
  api,
  type DirectoryCategory,
  type DirectoryKeyword,
} from "../api/client";

export function Directory() {
  const [categories, setCategories] = useState<DirectoryCategory[]>([]);
  const [selected, setSelected] = useState<number | null>(null);
  const [keywords, setKeywords] = useState<DirectoryKeyword[]>([]);
  const [catName, setCatName] = useState("");
  const [kwName, setKwName] = useState("");
  const [msg, setMsg] = useState<{ type: "ok" | "err"; text: string } | null>(
    null,
  );

  const loadCats = useCallback(async () => {
    try {
      const data = await api.getDirectoryCategories();
      setCategories(data || []);
      setMsg(null);
    } catch (e) {
      setMsg({
        type: "err",
        text: e instanceof Error ? e.message : "load categories failed",
      });
    }
  }, []);

  const loadKeywords = useCallback(async (id: number) => {
    try {
      const data = await api.getDirectoryKeywords(id);
      setKeywords(data || []);
    } catch (e) {
      setKeywords([]);
      setMsg({
        type: "err",
        text: e instanceof Error ? e.message : "load keywords failed",
      });
    }
  }, []);

  useEffect(() => {
    void loadCats();
  }, [loadCats]);

  useEffect(() => {
    if (selected != null) void loadKeywords(selected);
  }, [selected, loadKeywords]);

  async function onCreateCat(e: FormEvent) {
    e.preventDefault();
    try {
      await api.createDirectoryCategory(catName.trim());
      setCatName("");
      setMsg({ type: "ok", text: "category created" });
      await loadCats();
    } catch (err) {
      setMsg({
        type: "err",
        text: err instanceof Error ? err.message : "create category failed",
      });
    }
  }

  async function onDeleteCat(id: number) {
    if (!confirm(`delete category #${id}?`)) return;
    try {
      await api.deleteDirectoryCategory(id);
      if (selected === id) {
        setSelected(null);
        setKeywords([]);
      }
      setMsg({ type: "ok", text: `deleted category #${id}` });
      await loadCats();
    } catch (err) {
      setMsg({
        type: "err",
        text: err instanceof Error ? err.message : "delete failed",
      });
    }
  }

  async function onCreateKw(e: FormEvent) {
    e.preventDefault();
    if (selected == null) return;
    try {
      await api.createDirectoryKeyword(kwName.trim(), selected);
      setKwName("");
      setMsg({ type: "ok", text: "keyword created" });
      await loadKeywords(selected);
    } catch (err) {
      setMsg({
        type: "err",
        text: err instanceof Error ? err.message : "create keyword failed",
      });
    }
  }

  async function onDeleteKw(id: number) {
    if (selected == null) return;
    try {
      await api.deleteDirectoryKeyword(id);
      setMsg({ type: "ok", text: `deleted keyword #${id}` });
      await loadKeywords(selected);
    } catch (err) {
      setMsg({
        type: "err",
        text: err instanceof Error ? err.message : "delete keyword failed",
      });
    }
  }

  return (
    <>
      {msg && <div className={`msg ${msg.type}`}>{msg.text}</div>}
      <div className="grid split">
        <div className="card">
          <div className="card-head">
            <span>GET/POST /directory/category</span>
            <button className="btn ghost" type="button" onClick={() => void loadCats()}>
              refresh
            </button>
          </div>
          <div className="card-body">
            <form className="row" onSubmit={onCreateCat}>
              <div className="field">
                <label>category name</label>
                <input
                  value={catName}
                  onChange={(e) => setCatName(e.target.value)}
                  required
                />
              </div>
              <button className="btn" type="submit">
                create
              </button>
            </form>
          </div>
          <div className="table-wrap">
            {categories.length === 0 ? (
              <div className="empty">no categories</div>
            ) : (
              <table>
                <thead>
                  <tr>
                    <th>id</th>
                    <th>name</th>
                    <th>actions</th>
                  </tr>
                </thead>
                <tbody>
                  {categories.map((c) => (
                    <tr
                      key={c.id}
                      style={{
                        cursor: "pointer",
                        background:
                          selected === c.id
                            ? "rgba(24,163,210,0.08)"
                            : undefined,
                      }}
                      onClick={() => setSelected(c.id)}
                    >
                      <td>{c.id}</td>
                      <td>{c.name}</td>
                      <td>
                        <button
                          className="btn danger"
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation();
                            void onDeleteCat(c.id);
                          }}
                        >
                          delete
                        </button>
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
              {selected != null
                ? `keywords · category #${selected}`
                : "select a category"}
            </span>
          </div>
          <div className="card-body">
            {selected == null ? (
              <div className="empty">
                select category to manage GET/POST /directory/keyword
              </div>
            ) : (
              <>
                <form className="row" onSubmit={onCreateKw}>
                  <div className="field">
                    <label>keyword</label>
                    <input
                      value={kwName}
                      onChange={(e) => setKwName(e.target.value)}
                      required
                    />
                  </div>
                  <button className="btn" type="submit">
                    add keyword
                  </button>
                </form>
                <div className="table-wrap" style={{ marginTop: 12 }}>
                  {keywords.length === 0 ? (
                    <div className="empty">no keywords</div>
                  ) : (
                    <table>
                      <thead>
                        <tr>
                          <th>id</th>
                          <th>name</th>
                          <th>actions</th>
                        </tr>
                      </thead>
                      <tbody>
                        {keywords.map((k) => (
                          <tr key={k.id}>
                            <td>{k.id}</td>
                            <td>{k.name}</td>
                            <td>
                              <button
                                className="btn danger"
                                type="button"
                                onClick={() => void onDeleteKw(k.id)}
                              >
                                delete
                              </button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  )}
                </div>
              </>
            )}
          </div>
        </div>
      </div>
    </>
  );
}
