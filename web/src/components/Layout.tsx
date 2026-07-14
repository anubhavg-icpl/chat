import { NavLink, Outlet, useLocation } from "react-router-dom";

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
          <div className="badge online">Live</div>
        </header>
        <main className="content">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
