import { useEffect, useState } from "react";
import { NavLink, Outlet, useLocation } from "react-router-dom";
import { Moon, Search, Sun } from "lucide-react";
import { Button } from "@/components/ui/button";
import { CommandPalette } from "@/components/CommandPalette";
import { useTheme } from "@/hooks/useTheme";

const links = [
  { to: "/", label: "Dashboard", end: true },
  { to: "/users", label: "Users" },
  { to: "/sessions", label: "Sessions" },
  { to: "/chat", label: "Chat rooms" },
  { to: "/directory", label: "Directory" },
  { to: "/webapi", label: "Web API keys" },
  { to: "/im", label: "Send IM" },
];

const titles: Record<string, string> = {
  "/": "System overview",
  "/users": "User management",
  "/sessions": "Active sessions",
  "/chat": "Chat rooms",
  "/directory": "Keyword directory",
  "/webapi": "Web API keys",
  "/im": "Instant message relay",
};

export function Layout() {
  const { pathname } = useLocation();
  const title = titles[pathname] || "Console";
  const { theme, toggle } = useTheme();
  const [paletteOpen, setPaletteOpen] = useState(false);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setPaletteOpen((open) => !open);
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="brand">
          <strong>Open OSCAR Console</strong>
          <span>Management API · :8080</span>
        </div>
        <nav className="nav">
          {links.map((link) => (
            <NavLink
              key={link.to}
              to={link.to}
              end={link.end}
              className={({ isActive }) => (isActive ? "active" : undefined)}
            >
              <span className="prefix">›</span>
              <span>{link.label}</span>
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-footer">
          <div>
            Proxy: <span className="ok">/api → open-oscar-server:8080</span>
          </div>
          <div>Ports: 5190 · 5193 · 8080 · 9898 · 1088</div>
        </div>
      </aside>
      <div className="main">
        <header className="topbar">
          <h1>
            {title}
            <span>Operator console</span>
          </h1>
          <div className="topbar-actions">
            <Button
              variant="outline"
              size="sm"
              type="button"
              onClick={() => setPaletteOpen(true)}
            >
              <Search />
              <span>Search</span>
              <kbd className="cmdk-kbd">⌘K</kbd>
            </Button>
            <Button
              variant="outline"
              size="icon"
              type="button"
              aria-label="Toggle theme"
              onClick={toggle}
            >
              {theme === "dark" ? <Sun /> : <Moon />}
            </Button>
            <div className="badge online">Live</div>
          </div>
        </header>
        <main className="content">
          <Outlet />
        </main>
      </div>
      <CommandPalette open={paletteOpen} onOpenChange={setPaletteOpen} />
    </div>
  );
}
