import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
} from "react";
import { useNavigate } from "react-router-dom";
import { CornerDownLeft, Search } from "lucide-react";

type Command = {
  to: string;
  label: string;
  keywords: string;
};

const commands: Command[] = [
  { to: "/", label: "Dashboard", keywords: "system overview home stats" },
  { to: "/users", label: "Users", keywords: "accounts screen names aim icq" },
  { to: "/sessions", label: "Sessions", keywords: "active online connections kick" },
  { to: "/chat", label: "Chat rooms", keywords: "public private rooms" },
  { to: "/directory", label: "Directory", keywords: "keyword categories" },
  { to: "/webapi", label: "Web API keys", keywords: "api tokens webaim" },
  { to: "/im", label: "Send IM", keywords: "instant message relay" },
];

function matches(query: string, haystack: string): boolean {
  if (!query.trim()) return true;
  const needle = query.toLowerCase();
  let ni = 0;
  for (let i = 0; i < haystack.length && ni < needle.length; i++) {
    if (haystack[i] === needle[ni]) ni++;
  }
  return ni === needle.length;
}

type CommandPaletteProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function CommandPalette({ open, onOpenChange }: CommandPaletteProps) {
  const navigate = useNavigate();
  const [query, setQuery] = useState("");
  const [active, setActive] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  const filtered = useMemo(() => {
    if (!query.trim()) return commands;
    return commands.filter((c) => matches(query, `${c.label} ${c.keywords}`));
  }, [query]);

  useEffect(() => {
    if (!open) return;
    setQuery("");
    setActive(0);
    const id = window.setTimeout(() => inputRef.current?.focus(), 0);
    return () => window.clearTimeout(id);
  }, [open]);

  useEffect(() => {
    if (active > filtered.length - 1) setActive(0);
  }, [filtered, active]);

  useEffect(() => {
    if (!open) return;
    const el = listRef.current?.querySelector<HTMLElement>(
      `[data-idx="${active}"]`,
    );
    el?.scrollIntoView({ block: "nearest" });
  }, [active, open]);

  const close = useCallback(() => onOpenChange(false), [onOpenChange]);

  const run = useCallback(
    (cmd: Command) => {
      navigate(cmd.to);
      close();
    },
    [navigate, close],
  );

  const onKeyDown = (e: ReactKeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Escape") {
      e.preventDefault();
      close();
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      setActive((a) => (a + 1) % Math.max(1, filtered.length));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActive(
        (a) => (a - 1 + Math.max(1, filtered.length)) % Math.max(1, filtered.length),
      );
    } else if (e.key === "Enter") {
      e.preventDefault();
      const cmd = filtered[active];
      if (cmd) run(cmd);
    }
  };

  if (!open) return null;

  return (
    <div className="cmdk-backdrop" onClick={close} role="presentation">
      <div
        className="cmdk"
        role="dialog"
        aria-modal="true"
        aria-label="Command palette"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="cmdk-input-wrap">
          <Search className="cmdk-search-icon" />
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={onKeyDown}
            placeholder="Search pages…"
            className="cmdk-input"
            autoComplete="off"
            spellCheck={false}
          />
          <kbd className="cmdk-esc">esc</kbd>
        </div>

        <div className="cmdk-list" ref={listRef}>
          {filtered.length === 0 ? (
            <div className="cmdk-empty">No matches</div>
          ) : (
            filtered.map((cmd, i) => (
              <button
                key={cmd.to}
                type="button"
                data-idx={i}
                className={`cmdk-item${i === active ? " active" : ""}`}
                onMouseMove={() => setActive(i)}
                onClick={() => run(cmd)}
              >
                <span className="cmdk-label">{cmd.label}</span>
                <span className="cmdk-path">{cmd.to}</span>
                {i === active ? (
                  <CornerDownLeft className="cmdk-enter" />
                ) : null}
              </button>
            ))
          )}
        </div>

        <div className="cmdk-foot">
          <span>
            <kbd>↑</kbd>
            <kbd>↓</kbd> navigate
          </span>
          <span>
            <kbd>↵</kbd> open
          </span>
          <span>
            <kbd>esc</kbd> dismiss
          </span>
        </div>
      </div>
    </div>
  );
}
