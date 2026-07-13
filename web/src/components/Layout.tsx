import { NavLink, Outlet, useLocation } from "react-router-dom";

const links = [
  { to: "/", label: "dashboard", end: true },
  { to: "/users", label: "users" },
  { to: "/sessions", label: "sessions" },
  { to: "/chat", label: "chat rooms" },
  { to: "/directory", label: "directory" },
  { to: "/webapi", label: "webapi keys" },
  { to: "/im", label: "send im" },
];

const titles: Record<string, string> = {
  "/": "system overview + api probe",
  "/users": "users · account · feedbag · links",
  "/sessions": "active sessions",
  "/chat": "chat rooms",
  "/directory": "keyword directory",
  "/webapi": "web api keys",
  "/im": "instant message relay",
};

export function Layout() {
  const { pathname } = useLocation();
  const title = titles[pathname] || "console";

  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="brand">
          <strong>open-oscar // console</strong>
          <span>full mgmt API · terminal UI</span>
        </div>
        <nav className="nav">
          {links.map((link) => (
            <NavLink
              key={link.to}
              to={link.to}
              end={link.end}
              className={({ isActive }) => (isActive ? "active" : undefined)}
            >
              <span className="prefix">$</span>
              <span>{link.label}</span>
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-footer">
          <div>
            target: <span className="ok">/api → :8080</span>
          </div>
          <div>apis: users sessions chat dir webapi feedbag im</div>
        </div>
      </aside>
      <div className="main">
        <header className="topbar">
          <h1>
            root@oscar
            <span>/ {title}</span>
          </h1>
          <div className="badge online">● live</div>
        </header>
        <main className="content">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
